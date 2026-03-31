package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/sys/unix"
	internalaws "tui-ssm/internal/aws"
	"tui-ssm/internal/config"
	"tui-ssm/internal/store"
)

// ssmExecCmd wraps exec.Cmd to reset the terminal and flush the input
// buffer after SSM session exits. Without this, the session-manager-plugin
// can leave the terminal in a broken state and residual bytes in stdin
// that cause Bubble Tea's input parser to error, terminating the program.
type ssmExecCmd struct {
	cmd *exec.Cmd
}

func (c *ssmExecCmd) Run() error {
	err := c.cmd.Run()

	// 1. Reset terminal to a sane state before Bubble Tea tries to restore.
	reset := exec.Command("stty", "sane")
	reset.Stdin = os.Stdin
	reset.Run() //nolint:errcheck

	// 2. Flush the stdin input buffer to discard any residual bytes
	//    left by the session-manager-plugin. Without this, stale escape
	//    sequences can cause Bubble Tea's StreamEvents parser to error,
	//    which sends to p.errs and terminates the event loop.
	unix.IoctlSetInt(int(os.Stdin.Fd()), unix.TCFLSH, unix.TCIFLUSH) //nolint:errcheck

	return err
}

func (c *ssmExecCmd) SetStdin(r io.Reader)  { c.cmd.Stdin = r }
func (c *ssmExecCmd) SetStdout(w io.Writer) { c.cmd.Stdout = w }
func (c *ssmExecCmd) SetStderr(w io.Writer) { c.cmd.Stderr = w }

// InterruptFilter prevents SIGINT (delivered as InterruptMsg) from
// terminating the program. In raw mode Ctrl+C is delivered as a
// KeyPressMsg("ctrl+c") which our Update handles directly.
// InterruptMsg only arrives from OS signals — typically from a race
// between exec's RestoreTerminal re-enabling signals and a stale
// SIGINT from the SSM child process group.
func InterruptFilter(_ tea.Model, msg tea.Msg) tea.Msg {
	if _, ok := msg.(tea.InterruptMsg); ok {
		return nil
	}
	return msg
}

// Messages
type instancesLoadedMsg struct {
	instances []internalaws.Instance
	ssmStatus map[string]bool
	err       error
}

type ssmSessionDoneMsg struct {
	err         error
	instanceID  string
	profile     string
	region      string
	alias       string
	sessionType string // "session" or "port_forward"
}

// Sort columns cycle
var sortColumns = []string{"name", "id", "state", "type", "az"}

type Model struct {
	// State
	viewState ViewState
	loading   bool
	err       error

	// Data
	instances []internalaws.Instance
	filtered  []internalaws.Instance
	cursor    int

	// AWS
	profile  string
	region   string
	profiles []string

	// Config & Store
	cfg       config.Config
	favorites *store.Favorites
	history   *store.History

	// UI Components
	search       SearchModel
	filter       FilterModel
	profSelect   SelectorModel
	regionSelect SelectorModel
	portForward  PortForwardModel

	// Layout
	width  int
	height int

	// Sort
	sortBy    string
	sortOrder string
	sortIdx   int
}

type PortForwardModel struct {
	Active     bool
	LocalPort  string
	RemotePort string
	Field      int // 0 = local, 1 = remote
}

func NewModel(cfg config.Config, profiles []string, favs *store.Favorites, hist *store.History) Model {
	return Model{
		viewState: ViewTable,
		loading:   true,
		profile:   cfg.DefaultProfile,
		region:    cfg.DefaultRegion,
		profiles:  profiles,
		cfg:       cfg,
		favorites: favs,
		history:   hist,
		filter:    NewFilterModel(),
		sortBy:    cfg.Table.SortBy,
		sortOrder: cfg.Table.SortOrder,
		sortIdx:   0,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadInstances()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case instancesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.instances = msg.instances
		for i := range m.instances {
			if msg.ssmStatus != nil {
				m.instances[i].SSMConnected = msg.ssmStatus[m.instances[i].InstanceID]
			}
		}
		m.applyFilters()
		return m, nil

	case ssmSessionDoneMsg:
		m.viewState = ViewTable
		if msg.err != nil {
			m.err = fmt.Errorf("SSM session failed for %s: %v", msg.alias, msg.err)
			return m, nil
		}
		// Only record history for successful sessions
		m.history.Add(store.HistoryEntry{
			InstanceID: msg.instanceID,
			Profile:    msg.profile,
			Region:     msg.region,
			Alias:      msg.alias,
			Type:       msg.sessionType,
		})
		m.history.Save(store.HistoryPath())
		m.err = nil
		m.loading = true
		return m, m.loadInstances()
	}

	// Dispatch based on view state
	switch m.viewState {
	case ViewSearch:
		return m.updateSearch(msg)
	case ViewFilter:
		return m.updateFilter(msg)
	case ViewProfileSelect:
		return m.updateProfileSelect(msg)
	case ViewRegionSelect:
		return m.updateRegionSelect(msg)
	case ViewPortForward:
		return m.updatePortForward(msg)
	default:
		return m.updateTable(msg)
	}
}

func (m Model) updateTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "enter":
		if m.cursor < len(m.filtered) {
			return m, m.startSSMSession(m.filtered[m.cursor])
		}

	case "/":
		m.viewState = ViewSearch
		m.search.Active = true
		m.search.Query = ""

	case "f":
		m.viewState = ViewFilter
		m.filter.Active = true

	case "p":
		m.profSelect = NewSelector("Select Profile", m.profiles, m.profile)
		m.profSelect.Active = true
		m.viewState = ViewProfileSelect

	case "r":
		m.regionSelect = NewSelector("Select Region", internalaws.KnownRegions(), m.region)
		m.regionSelect.Active = true
		m.viewState = ViewRegionSelect

	case "s":
		m.sortIdx = (m.sortIdx + 1) % len(sortColumns)
		m.sortBy = sortColumns[m.sortIdx]
		m.applyFilters()

	case "S":
		if m.sortOrder == "asc" {
			m.sortOrder = "desc"
		} else {
			m.sortOrder = "asc"
		}
		m.applyFilters()

	case "F":
		if m.cursor < len(m.filtered) {
			inst := m.filtered[m.cursor]
			if m.favorites.IsFavorite(inst.InstanceID, m.profile, m.region) {
				m.favorites.Remove(inst.InstanceID, m.profile, m.region)
			} else {
				m.favorites.Add(store.Favorite{
					InstanceID: inst.InstanceID,
					Profile:    m.profile,
					Region:     m.region,
					Alias:      inst.DisplayName(),
				})
			}
			m.favorites.Save(store.FavoritesPath())
			m.applyFilters()
		}

	case "P":
		if m.cursor < len(m.filtered) {
			m.viewState = ViewPortForward
			m.portForward = PortForwardModel{Active: true, LocalPort: "8080", RemotePort: "80", Field: 0}
		}

	case "R":
		m.loading = true
		m.err = nil
		return m, m.loadInstances()
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.search.Clear()
		m.viewState = ViewTable
		m.applyFilters()
	case "enter":
		m.viewState = ViewTable
		m.search.Active = false
		if m.cursor < len(m.filtered) {
			return m, m.startSSMSession(m.filtered[m.cursor])
		}
	case "backspace":
		m.search.Backspace()
		m.applyFilters()
	default:
		r := keyMsg.String()
		if len(r) == 1 {
			m.search.Insert(rune(r[0]))
			m.applyFilters()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc", "f":
		m.filter.Active = false
		m.viewState = ViewTable
		m.applyFilters()
	case "up", "k":
		m.filter.MoveUp()
	case "down", "j":
		m.filter.MoveDown()
	case " ", "enter":
		m.filter.Toggle()
		m.applyFilters()
	case "c":
		m.filter.ClearAll()
		m.applyFilters()
	}
	return m, nil
}

func (m Model) updateProfileSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.profSelect.Active = false
		m.viewState = ViewTable
	case "up", "k":
		m.profSelect.MoveUp()
	case "down", "j":
		m.profSelect.MoveDown()
	case "enter":
		m.profile = m.profSelect.Selected()
		m.profSelect.Active = false
		m.viewState = ViewTable
		m.loading = true
		return m, m.loadInstances()
	}
	return m, nil
}

func (m Model) updateRegionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.regionSelect.Active = false
		m.viewState = ViewTable
	case "up", "k":
		m.regionSelect.MoveUp()
	case "down", "j":
		m.regionSelect.MoveDown()
	case "enter":
		m.region = m.regionSelect.Selected()
		m.regionSelect.Active = false
		m.viewState = ViewTable
		m.loading = true
		return m, m.loadInstances()
	}
	return m, nil
}

func (m Model) updatePortForward(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.portForward.Active = false
		m.viewState = ViewTable
	case "tab":
		m.portForward.Field = (m.portForward.Field + 1) % 2
	case "enter":
		if m.cursor < len(m.filtered) {
			m.portForward.Active = false
			return m, m.startPortForward(m.filtered[m.cursor])
		}
	case "backspace":
		if m.portForward.Field == 0 && len(m.portForward.LocalPort) > 0 {
			m.portForward.LocalPort = m.portForward.LocalPort[:len(m.portForward.LocalPort)-1]
		} else if m.portForward.Field == 1 && len(m.portForward.RemotePort) > 0 {
			m.portForward.RemotePort = m.portForward.RemotePort[:len(m.portForward.RemotePort)-1]
		}
	default:
		r := keyMsg.String()
		if len(r) == 1 && r[0] >= '0' && r[0] <= '9' {
			if m.portForward.Field == 0 {
				m.portForward.LocalPort += r
			} else {
				m.portForward.RemotePort += r
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	var sections []string

	// Status bar
	sections = append(sections, RenderStatusBar(m.profile, m.region, m.filter.Label(), len(m.filtered), m.width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(m.width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(2, 2).Render("Loading instances..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(
			ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(2, 2).Render("No instances found in this region."))
	} else {
		columns := ColumnsForWidth(m.width)
		tableHeight := m.height
		if m.search.Active {
			tableHeight--
		}
		sections = append(sections, RenderTable(m.filtered, columns, m.cursor, m.favorites, m.history, m.profile, m.region, m.width, tableHeight))
	}

	// Overlay (filter / profile / region / port forward)
	overlay := ""
	switch {
	case m.filter.Active:
		overlay = m.filter.Render(m.width)
	case m.profSelect.Active:
		overlay = m.profSelect.Render(m.width)
	case m.regionSelect.Active:
		overlay = m.regionSelect.Render(m.width)
	case m.portForward.Active:
		overlay = m.renderPortForward()
	}

	// Help bar
	sections = append(sections, RenderHelpBar(m.viewState, m.width))

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view += "\n" + lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Center, overlay)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (m Model) renderPortForward() string {
	var b strings.Builder
	b.WriteString("  Port Forwarding\n")
	b.WriteString("  ─────────────────\n")
	if m.cursor < len(m.filtered) {
		b.WriteString(fmt.Sprintf("  Target: %s\n\n", m.filtered[m.cursor].DisplayName()))
	}

	localLabel := "  Local Port:  "
	remoteLabel := "  Remote Port: "
	if m.portForward.Field == 0 {
		localLabel = "▸ Local Port:  "
	} else {
		remoteLabel = "▸ Remote Port: "
	}
	b.WriteString(fmt.Sprintf("%s%s\n", localLabel, m.portForward.LocalPort))
	b.WriteString(fmt.Sprintf("%s%s\n", remoteLabel, m.portForward.RemotePort))
	b.WriteString("\n  Tab: switch field  Enter: start  Esc: cancel")

	return OverlayStyle.Render(b.String())
}

func (m *Model) applyFilters() {
	result := m.instances

	// State filter
	result = FilterByState(result, m.filter.ActiveStates)

	// Search filter
	result = FilterBySearch(result, m.search.Query)

	// Sort
	result = SortInstances(result, m.favorites, m.history, m.profile, m.region, m.sortBy, m.sortOrder)

	m.filtered = result

	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) loadInstances() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, m.profile, m.region)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		instances, err := internalaws.FetchInstances(ctx, clients.EC2)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		ssmStatus, _ := internalaws.FetchSSMStatus(ctx, clients.SSM)

		return instancesLoadedMsg{
			instances: instances,
			ssmStatus: ssmStatus,
		}
	}
}

func (m Model) startSSMSession(inst internalaws.Instance) tea.Cmd {
	profile := m.profile
	region := m.region
	alias := inst.DisplayName()
	instanceID := inst.InstanceID

	args := internalaws.BuildSSMSessionArgs(instanceID, profile, region)
	c := exec.Command("aws", args...)
	return tea.Exec(&ssmExecCmd{cmd: c}, func(err error) tea.Msg {
		return ssmSessionDoneMsg{
			err:         err,
			instanceID:  instanceID,
			profile:     profile,
			region:      region,
			alias:       alias,
			sessionType: "session",
		}
	})
}

func (m Model) startPortForward(inst internalaws.Instance) tea.Cmd {
	profile := m.profile
	region := m.region
	alias := inst.DisplayName()
	instanceID := inst.InstanceID
	localPort := m.portForward.LocalPort
	remotePort := m.portForward.RemotePort

	args := internalaws.BuildPortForwardArgs(instanceID, profile, region, localPort, remotePort)
	c := exec.Command("aws", args...)
	return tea.Exec(&ssmExecCmd{cmd: c}, func(err error) tea.Msg {
		return ssmSessionDoneMsg{
			err:         err,
			instanceID:  instanceID,
			profile:     profile,
			region:      region,
			alias:       alias,
			sessionType: "port_forward",
		}
	})
}
