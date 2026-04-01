package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sys/unix"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/config"
	"tui-aws/internal/store"
	"tui-aws/internal/ui/shared"
	"tui-aws/internal/ui/tab_acm"
	"tui-aws/internal/ui/tab_asg"
	"tui-aws/internal/ui/tab_cloudfront"
	"tui-aws/internal/ui/tab_cloudwatch"
	"tui-aws/internal/ui/tab_ebs"
	"tui-aws/internal/ui/tab_ec2"
	"tui-aws/internal/ui/tab_ecs"
	"tui-aws/internal/ui/tab_eks"
	"tui-aws/internal/ui/tab_elb"
	"tui-aws/internal/ui/tab_iam"
	"tui-aws/internal/ui/tab_lambda"
	"tui-aws/internal/ui/tab_r53"
	"tui-aws/internal/ui/tab_rds"
	"tui-aws/internal/ui/tab_routetable"
	"tui-aws/internal/ui/tab_s3"
	"tui-aws/internal/ui/tab_sg"
	"tui-aws/internal/ui/tab_subnet"
	"tui-aws/internal/ui/tab_tgw"
	"tui-aws/internal/ui/tab_troubleshoot"
	"tui-aws/internal/ui/tab_vpc"
	"tui-aws/internal/ui/tab_vpce"
	"tui-aws/internal/ui/tab_waf"
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

// overlayState tracks which global overlay is active.
type overlayState int

const (
	overlayNone overlayState = iota
	overlayProfileSelect
	overlayRegionSelect
)

// RootModel is the top-level Bubble Tea model that manages tabs and global state.
type RootModel struct {
	shared   shared.SharedState
	tabs     []shared.TabModel
	tabIDs   []shared.TabID
	activeTab int

	// Global overlays
	overlay      overlayState
	profSelect   shared.SelectorModel
	regionSelect shared.SelectorModel
}

// NewRootModel creates the root model with all tabs.
func NewRootModel(cfg config.Config, profiles []string, favs *store.Favorites, hist *store.History) RootModel {
	s := shared.SharedState{
		Profile:   cfg.DefaultProfile,
		Region:    cfg.DefaultRegion,
		Profiles:  profiles,
		Cfg:       cfg,
		Favorites: favs,
		History:   hist,
		Cache:     make(map[string]shared.CachedData),
	}

	tabIDs := shared.AllTabs()
	tabs := make([]shared.TabModel, len(tabIDs))
	for i, id := range tabIDs {
		switch id {
		case shared.TabEC2:
			tabs[i] = tab_ec2.New(cfg.Table.SortBy, cfg.Table.SortOrder)
		case shared.TabASG:
			tabs[i] = tab_asg.New()
		case shared.TabEBS:
			tabs[i] = tab_ebs.New()
		case shared.TabVPC:
			tabs[i] = tab_vpc.New()
		case shared.TabSubnet:
			tabs[i] = tab_subnet.New()
		case shared.TabRoutes:
			tabs[i] = tab_routetable.New()
		case shared.TabSG:
			tabs[i] = tab_sg.New()
		case shared.TabVPCEndpoint:
			tabs[i] = tab_vpce.New()
		case shared.TabTGW:
			tabs[i] = tab_tgw.New()
		case shared.TabELB:
			tabs[i] = tab_elb.New()
		case shared.TabCloudWatch:
			tabs[i] = tab_cloudwatch.New()
		case shared.TabCloudFront:
			tabs[i] = tab_cloudfront.New()
		case shared.TabWAF:
			tabs[i] = tab_waf.New()
		case shared.TabACM:
			tabs[i] = tab_acm.New()
		case shared.TabR53:
			tabs[i] = tab_r53.New()
		case shared.TabRDS:
			tabs[i] = tab_rds.New()
		case shared.TabS3:
			tabs[i] = tab_s3.New()
		case shared.TabECS:
			tabs[i] = tab_ecs.New()
		case shared.TabEKS:
			tabs[i] = tab_eks.New()
		case shared.TabLambda:
			tabs[i] = tab_lambda.New()
		case shared.TabIAM:
			tabs[i] = tab_iam.New()
		case shared.TabCheck:
			tabs[i] = tab_troubleshoot.New()
		default:
			tabs[i] = NewPlaceholderTab(id.Label())
		}
	}

	return RootModel{
		shared:    s,
		tabs:      tabs,
		tabIDs:    tabIDs,
		activeTab: 0,
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.tabs[m.activeTab].Init(&m.shared)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.shared.Width = msg.Width
		m.shared.Height = msg.Height
		return m, nil

	// Handle SSM exec requests from the EC2 tab
	case tab_ec2.SSMExecRequest:
		c := exec.Command("aws", msg.Args...)
		return m, tea.Exec(&ssmExecCmd{cmd: c}, func(err error) tea.Msg {
			return tab_ec2.SSMSessionDoneMsg{
				Err:         err,
				InstanceID:  msg.InstanceID,
				Profile:     msg.Profile,
				Region:      msg.Region,
				Alias:       msg.Alias,
				SessionType: msg.Type,
			}
		})

	// Handle ECS Exec requests from the ECS tab
	case tab_ecs.ECSExecRequest:
		c := exec.Command("aws", msg.Args...)
		return m, tea.Exec(&ssmExecCmd{cmd: c}, func(err error) tea.Msg {
			return tab_ecs.ECSExecDoneMsg{Err: err}
		})

	// Handle ECS Exec completion — forward to the ECS tab
	case tab_ecs.ECSExecDoneMsg:
		tab, cmd := m.tabs[m.activeTab].Update(msg, &m.shared)
		m.tabs[m.activeTab] = tab
		return m, cmd

	// Handle SSM session completion — record history
	case tab_ec2.SSMSessionDoneMsg:
		if msg.Err == nil {
			m.shared.History.Add(store.HistoryEntry{
				InstanceID: msg.InstanceID,
				Profile:    msg.Profile,
				Region:     msg.Region,
				Alias:      msg.Alias,
				Type:       msg.SessionType,
			})
			m.shared.History.Save(store.HistoryPath())
		}
		// Forward to the EC2 tab so it can update its state
		tab, cmd := m.tabs[m.activeTab].Update(msg, &m.shared)
		m.tabs[m.activeTab] = tab
		return m, cmd

	// Handle tab navigation messages
	case shared.NavigateToTab:
		for i, id := range m.tabIDs {
			if id == msg.Tab {
				return m.switchTab(i)
			}
		}
		return m, nil
	}

	// Handle global overlays
	if m.overlay != overlayNone {
		return m.updateOverlay(msg)
	}

	// Handle global keys
	// When the Check tab is editing text fields (protocol/port), skip global
	// key handling so characters like p, r, q, c, and digits reach the tab.
	checkEditing := false
	if m.tabIDs[m.activeTab] == shared.TabCheck {
		if tm, ok := m.tabs[m.activeTab].(*tab_troubleshoot.TroubleshootModel); ok {
			checkEditing = tm.IsEditing()
		}
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && !checkEditing {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "p":
			m.profSelect = shared.NewSelector("Select Profile", m.shared.Profiles, m.shared.Profile)
			m.profSelect.Active = true
			m.overlay = overlayProfileSelect
			return m, nil

		case "r":
			m.regionSelect = shared.NewSelector("Select Region", internalaws.KnownRegions(), m.shared.Region)
			m.regionSelect.Active = true
			m.overlay = overlayRegionSelect
			return m, nil

		case "]":
			next := (m.activeTab + 1) % len(m.tabs)
			return m.switchTab(next)

		case "[":
			prev := (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m.switchTab(prev)

		case "tab":
			// Check tab uses Tab for field switching — delegate to tab
			if m.tabIDs[m.activeTab] == shared.TabCheck {
				break
			}
			next := (m.activeTab + 1) % len(m.tabs)
			return m.switchTab(next)

		case "shift+tab":
			if m.tabIDs[m.activeTab] == shared.TabCheck {
				break
			}
			prev := (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			return m.switchTab(prev)
		}
	}

	// Delegate to active tab
	tab, cmd := m.tabs[m.activeTab].Update(msg, &m.shared)
	m.tabs[m.activeTab] = tab
	return m, cmd
}

func (m RootModel) View() tea.View {
	if m.shared.Width == 0 {
		return tea.NewView("Loading...")
	}

	var sections []string

	// Tab bar
	sections = append(sections, m.renderTabBar())

	// Active tab content
	sections = append(sections, m.tabs[m.activeTab].View(&m.shared))

	// Help bar
	tabHelp := m.tabs[m.activeTab].ShortHelp()
	globalHelp := globalHelpLine()
	help := tabHelp
	if help != "" && globalHelp != "" {
		help += "  "
	}
	help += globalHelp
	sections = append(sections, shared.HelpBarStyle.Width(m.shared.Width).Render(help))

	view := strings.Join(sections, "\n")

	// Global overlay
	overlay := ""
	switch m.overlay {
	case overlayProfileSelect:
		overlay = m.profSelect.Render(m.shared.Width)
	case overlayRegionSelect:
		overlay = m.regionSelect.Render(m.shared.Width)
	}
	if overlay != "" {
		view = shared.PlaceOverlay(m.shared.Width, m.shared.Height, overlay)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

// --- Internal helpers ---

func (m RootModel) switchTab(idx int) (tea.Model, tea.Cmd) {
	if idx == m.activeTab {
		return m, nil
	}
	m.activeTab = idx
	cmd := m.tabs[m.activeTab].Init(&m.shared)
	return m, cmd
}

func (m RootModel) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.overlay {
	case overlayProfileSelect:
		switch keyMsg.String() {
		case "esc":
			m.profSelect.Active = false
			m.overlay = overlayNone
		case "up", "k":
			m.profSelect.MoveUp()
		case "down", "j":
			m.profSelect.MoveDown()
		case "enter":
			m.shared.Profile = m.profSelect.Selected()
			m.profSelect.Active = false
			m.overlay = overlayNone
			m.shared.ClearCache()
			cmd := m.tabs[m.activeTab].Init(&m.shared)
			return m, cmd
		}

	case overlayRegionSelect:
		switch keyMsg.String() {
		case "esc":
			m.regionSelect.Active = false
			m.overlay = overlayNone
		case "up", "k":
			m.regionSelect.MoveUp()
		case "down", "j":
			m.regionSelect.MoveDown()
		case "enter":
			m.shared.Region = m.regionSelect.Selected()
			m.regionSelect.Active = false
			m.overlay = overlayNone
			m.shared.ClearCache()
			cmd := m.tabs[m.activeTab].Init(&m.shared)
			return m, cmd
		}
	}

	return m, nil
}

func (m RootModel) renderTabBar() string {
	var parts []string
	for i, id := range m.tabIDs {
		name := id.Label()
		if i == m.activeTab {
			parts = append(parts, shared.TabActiveStyle.Render(name))
		} else {
			parts = append(parts, shared.TabInactiveStyle.Render(name))
		}
	}

	bar := strings.Join(parts, "")

	// Profile/Region on the right (if space allows)
	profile := shared.StatusKeyStyle.Render(" Profile:") + " " + m.shared.Profile
	region := shared.StatusKeyStyle.Render(" Region:") + " " + m.shared.Region
	right := "  " + profile + "  " + region

	return shared.TabBarStyle.Width(m.shared.Width).Render(bar + right)
}

func globalHelpLine() string {
	var s string
	pairs := [][2]string{
		{"p", "Profile"},
		{"r", "Region"},
		{"[/]", "Prev/Next Tab"},
		{"q", "Quit"},
	}
	for _, p := range pairs {
		if s != "" {
			s += "  "
		}
		s += fmt.Sprintf("%s: %s", shared.HelpKeyStyle.Render(p[0]), p[1])
	}
	return " " + s
}
