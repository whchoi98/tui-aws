package tab_vpc

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the VPC tab's internal view mode.
type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsActionMenu
	vsDetail
)

// vpcsLoadedMsg is returned when VPCs are fetched.
type vpcsLoadedMsg struct {
	vpcs []internalaws.VPC
	err  error
}

// Action represents a menu action for a VPC.
type Action struct {
	Key   string
	Label string
}

// ActionMenuModel manages the action menu state.
type ActionMenuModel struct {
	Active  bool
	VPC     internalaws.VPC
	Actions []Action
	Cursor  int
}

func newActionMenu(vpc internalaws.VPC) ActionMenuModel {
	return ActionMenuModel{
		Active: true,
		VPC:    vpc,
		Actions: []Action{
			{Key: "detail", Label: "VPC Details"},
			{Key: "subnets", Label: "Subnets in this VPC"},
			{Key: "routes", Label: "Route Tables"},
			{Key: "sg", Label: "Security Groups"},
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
	name := a.VPC.Name
	if name == "" {
		name = a.VPC.VpcID
	}
	b.WriteString(fmt.Sprintf("  %s (%s)\n", name, a.VPC.VpcID))
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

// VPCModel implements the shared.TabModel interface for the VPC tab.
type VPCModel struct {
	viewState viewState
	loading   bool
	err       error

	vpcs     []internalaws.VPC
	filtered []internalaws.VPC
	cursor   int

	search     SearchModel
	actionMenu ActionMenuModel

	// Detail overlay
	showDetail    bool
	detailLoading bool
	detailErr     error
	detailData    vpcDetailData
}

// New creates a new VPCModel.
func New() *VPCModel {
	return &VPCModel{
		viewState: vsTable,
		loading:   true,
	}
}

func (m *VPCModel) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	return m.loadVPCs(s)
}

func (m *VPCModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case vpcsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.vpcs = msg.vpcs
		m.applyFilters()
		return m, nil

	case vpcDetailLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.detailErr = msg.err
			return m, nil
		}
		m.detailData = msg.data
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

func (m *VPCModel) View(s *shared.SharedState) string {
	var sections []string

	// Status bar
	sections = append(sections, renderStatusBar(s.Profile, s.Region, len(m.filtered), s.Width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading VPCs..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No VPCs found in this region."))
	} else {
		columns := ColumnsForWidth(s.Width)
		tableHeight := s.Height
		if m.search.Active {
			tableHeight--
		}
		sections = append(sections, RenderTable(m.filtered, columns, m.cursor, s.Width, tableHeight))
	}

	// Overlay
	overlay := ""
	switch {
	case m.showDetail:
		if m.cursor < len(m.filtered) {
			overlay = RenderVPCDetail(m.filtered[m.cursor], m.detailData, m.detailLoading, m.detailErr)
		}
	case m.actionMenu.Active:
		overlay = m.actionMenu.Render(s.Width)
	}

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view += "\n" + shared.PlaceOverlay(s.Width, overlay)
	}

	return view
}

func (m *VPCModel) ShortHelp() string {
	switch m.viewState {
	case vsSearch:
		return helpLine("Esc", "Cancel")
	case vsActionMenu:
		return helpLine("↑↓", "Navigate", "Enter", "Select", "Esc", "Cancel")
	case vsDetail:
		return helpLine("any key", "Close")
	default:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "/", "Search", "R", "Refresh")
	}
}

// --- Internal update handlers ---

func (m *VPCModel) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		return m, m.loadVPCs(s)
	}

	return m, nil
}

func (m *VPCModel) updateSearch(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

func (m *VPCModel) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
			m.detailLoading = true
			m.detailErr = nil
			vpc := m.actionMenu.VPC
			return m, loadVPCDetail(vpc.VpcID, s.Profile, s.Region)
		case "subnets":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabSubnet}
			}
		case "routes":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabRoutes}
			}
		case "sg":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabSG}
			}
		}
	}
	return m, nil
}

func (m *VPCModel) updateDetail(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	// If still loading, only handle the loaded message (already handled in Update)
	if m.detailLoading {
		return m, nil
	}

	// Any key closes the detail overlay
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.showDetail = false
		m.viewState = vsTable
	}
	return m, nil
}

// --- Helpers ---

func (m *VPCModel) applyFilters() {
	result := m.vpcs

	if m.search.Query != "" {
		q := strings.ToLower(m.search.Query)
		var filtered []internalaws.VPC
		for _, vpc := range result {
			if strings.Contains(strings.ToLower(vpc.Name), q) ||
				strings.Contains(strings.ToLower(vpc.VpcID), q) ||
				strings.Contains(strings.ToLower(vpc.CIDRBlock), q) {
				filtered = append(filtered, vpc)
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

func (m *VPCModel) loadVPCs(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return vpcsLoadedMsg{err: err}
		}
		vpcs, err := internalaws.FetchVPCs(ctx, clients.EC2)
		if err != nil {
			return vpcsLoadedMsg{err: err}
		}
		return vpcsLoadedMsg{vpcs: vpcs}
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
