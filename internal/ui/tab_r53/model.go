package tab_r53

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the R53 tab's internal view mode.
type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsActionMenu
	vsDetail
)

// zonesLoadedMsg is returned when hosted zones are fetched.
type zonesLoadedMsg struct {
	zones []internalaws.HostedZone
	err   error
}

// recordsLoadedMsg is returned when zone records are fetched on demand.
type recordsLoadedMsg struct {
	zoneID  string
	records []internalaws.DNSRecord
	err     error
}

// Action represents a menu action for a hosted zone.
type Action struct {
	Key   string
	Label string
}

// ActionMenuModel manages the action menu state.
type ActionMenuModel struct {
	Active  bool
	Zone    internalaws.HostedZone
	Actions []Action
	Cursor  int
}

func newActionMenu(zone internalaws.HostedZone) ActionMenuModel {
	return ActionMenuModel{
		Active: true,
		Zone:   zone,
		Actions: []Action{
			{Key: "detail", Label: "Zone Details & Records"},
		},
		Cursor: 0,
	}
}

func (a *ActionMenuModel) MoveUp() {
	if a.Cursor > 0 {
		a.Cursor--
	}
}

func (a *ActionMenuModel) MoveDown() {
	if a.Cursor < len(a.Actions)-1 {
		a.Cursor++
	}
}

func (a *ActionMenuModel) Selected() string {
	if a.Cursor < len(a.Actions) {
		return a.Actions[a.Cursor].Key
	}
	return ""
}

func (a *ActionMenuModel) Render(width int) string {
	if !a.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s (%s)\n", a.Zone.Name, a.Zone.ID))
	b.WriteString("  ─────────────────────────\n")

	for i, action := range a.Actions {
		cursor := "  "
		if i == a.Cursor {
			cursor = "▸ "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, action.Label))
	}
	b.WriteString("\n  Enter: select  Esc: cancel")

	return shared.RenderOverlay(b.String())
}

// SearchModel manages the search input state.
type SearchModel struct {
	Query  string
	Active bool
}

func (s *SearchModel) Insert(char rune) {
	s.Query += string(char)
}

func (s *SearchModel) Backspace() {
	if len(s.Query) > 0 {
		s.Query = s.Query[:len(s.Query)-1]
	}
}

func (s *SearchModel) Clear() {
	s.Query = ""
	s.Active = false
}

func (s *SearchModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	prompt := shared.SearchPromptStyle.Render(" /")
	return lipgloss.NewStyle().Width(width).Render(
		fmt.Sprintf("%s %s█", prompt, s.Query),
	)
}

// R53Model implements the shared.TabModel interface for the Route 53 tab.
type R53Model struct {
	viewState viewState
	loading   bool
	err       error

	zones    []internalaws.HostedZone
	filtered []internalaws.HostedZone
	cursor   int

	search     SearchModel
	actionMenu ActionMenuModel

	// Detail overlay with on-demand record loading
	showDetail    bool
	detailZone    *internalaws.HostedZone
	detailLoading bool
	detailScroll  int // scroll offset within detail records
}

// New creates a new R53Model.
func New() *R53Model {
	return &R53Model{
		viewState: vsTable,
		loading:   true,
	}
}

func (m *R53Model) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	return m.loadZones(s)
}

func (m *R53Model) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case zonesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.zones = msg.zones
		m.applyFilters()
		return m, nil

	case recordsLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			return m, nil
		}
		if m.detailZone != nil && m.detailZone.ID == msg.zoneID {
			m.detailZone.Records = msg.records
		}
		return m, nil
	}

	switch m.viewState {
	case vsSearch:
		return m.updateSearch(msg, s)
	case vsActionMenu:
		return m.updateActionMenu(msg, s)
	case vsDetail:
		return m.updateDetail(msg, s)
	default:
		return m.updateTable(msg, s)
	}
}

func (m *R53Model) View(s *shared.SharedState) string {
	var sections []string

	sections = append(sections, renderStatusBar(s.Profile, s.Region, len(m.filtered), s.Width))

	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}

	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading Route 53 Hosted Zones..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No Route 53 Hosted Zones found."))
	} else {
		columns := ColumnsForWidth(s.Width)
		tableHeight := s.Height
		if m.search.Active {
			tableHeight--
		}
		sections = append(sections, RenderTable(m.filtered, columns, m.cursor, s.Width, tableHeight))
	}

	overlay := ""
	switch {
	case m.showDetail && m.detailZone != nil:
		overlay = RenderZoneDetail(*m.detailZone, m.detailLoading, m.detailScroll, s.Height)
	case m.actionMenu.Active:
		overlay = m.actionMenu.Render(s.Width)
	}

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view = shared.PlaceOverlay(s.Width, s.Height, overlay)
	}

	return view
}

func (m *R53Model) ShortHelp() string {
	switch m.viewState {
	case vsSearch:
		return helpLine("Esc", "Cancel")
	case vsActionMenu:
		return helpLine("↑↓", "Navigate", "Enter", "Select", "Esc", "Cancel")
	case vsDetail:
		return helpLine("↑↓", "Scroll", "Esc", "Close")
	default:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "/", "Search", "R", "Refresh")
	}
}

// --- Internal update handlers ---

func (m *R53Model) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
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
			m.actionMenu = newActionMenu(m.filtered[m.cursor])
			m.viewState = vsActionMenu
		}
	case "/":
		m.viewState = vsSearch
		m.search.Active = true
		m.search.Query = ""
	case "R":
		m.loading = true
		m.err = nil
		return m, m.loadZones(s)
	}

	return m, nil
}

func (m *R53Model) updateSearch(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.search.Clear()
		m.viewState = vsTable
		m.applyFilters()
	case "enter":
		m.viewState = vsTable
		m.search.Active = false
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

func (m *R53Model) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.actionMenu.Active = false
		m.viewState = vsTable
	case "up", "k":
		m.actionMenu.MoveUp()
	case "down", "j":
		m.actionMenu.MoveDown()
	case "enter":
		action := m.actionMenu.Selected()
		switch action {
		case "detail":
			m.actionMenu.Active = false
			m.viewState = vsDetail
			m.showDetail = true
			m.detailScroll = 0

			if m.cursor < len(m.filtered) {
				zone := m.filtered[m.cursor]
				m.detailZone = &zone
				// Load records on demand
				m.detailLoading = true
				return m, m.loadRecords(s, zone.ID)
			}
		}
	}
	return m, nil
}

func (m *R53Model) updateDetail(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.showDetail = false
		m.detailZone = nil
		m.detailScroll = 0
		m.viewState = vsTable
	case "up", "k":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "down", "j":
		if m.detailZone != nil && m.detailScroll < len(m.detailZone.Records)-1 {
			m.detailScroll++
		}
	}
	return m, nil
}

// --- Helpers ---

func (m *R53Model) applyFilters() {
	result := m.zones

	if m.search.Query != "" {
		q := strings.ToLower(m.search.Query)
		var filtered []internalaws.HostedZone
		for _, zone := range result {
			if strings.Contains(strings.ToLower(zone.Name), q) ||
				strings.Contains(strings.ToLower(zone.ID), q) ||
				strings.Contains(strings.ToLower(zone.Comment), q) {
				filtered = append(filtered, zone)
			}
		}
		result = filtered
	}

	m.filtered = result

	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *R53Model) loadZones(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return zonesLoadedMsg{err: err}
		}
		zones, err := internalaws.FetchHostedZones(ctx, clients.R53)
		if err != nil {
			return zonesLoadedMsg{err: err}
		}
		return zonesLoadedMsg{zones: zones}
	}
}

func (m *R53Model) loadRecords(s *shared.SharedState, zoneID string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return recordsLoadedMsg{zoneID: zoneID, err: err}
		}
		records, err := internalaws.FetchRecords(ctx, clients.R53, zoneID)
		return recordsLoadedMsg{zoneID: zoneID, records: records, err: err}
	}
}

func helpLine(keyvals ...string) string {
	var s string
	for i := 0; i < len(keyvals)-1; i += 2 {
		if s != "" {
			s += "  "
		}
		s += fmt.Sprintf("%s: %s", shared.HelpKeyStyle.Render(keyvals[i]), keyvals[i+1])
	}
	return " " + s
}
