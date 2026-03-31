package tab_subnet

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the Subnet tab's internal view mode.
type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsActionMenu
	vsDetail
)

// subnetsLoadedMsg is returned when subnets are fetched.
type subnetsLoadedMsg struct {
	subnets []internalaws.Subnet
	err     error
}

// Action represents a menu action for a Subnet.
type Action struct {
	Key   string
	Label string
}

// ActionMenuModel manages the action menu state.
type ActionMenuModel struct {
	Active  bool
	Subnet  internalaws.Subnet
	Actions []Action
	Cursor  int
}

func newActionMenu(sub internalaws.Subnet) ActionMenuModel {
	return ActionMenuModel{
		Active: true,
		Subnet: sub,
		Actions: []Action{
			{Key: "enis", Label: "ENIs in this Subnet"},
			{Key: "goto_vpc", Label: "Go to VPC"},
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
	name := a.Subnet.Name
	if name == "" {
		name = a.Subnet.SubnetID
	}
	b.WriteString(fmt.Sprintf("  %s (%s)\n", name, a.Subnet.SubnetID))
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

// SubnetModel implements the shared.TabModel interface for the Subnet tab.
type SubnetModel struct {
	viewState viewState
	loading   bool
	err       error

	subnets  []internalaws.Subnet
	filtered []internalaws.Subnet
	cursor   int

	search     SearchModel
	actionMenu ActionMenuModel

	// ENI detail overlay
	showDetail    bool
	detailLoading bool
	detailErr     error
	detailENIs    []internalaws.ENI
}

// New creates a new SubnetModel.
func New() *SubnetModel {
	return &SubnetModel{
		viewState: vsTable,
		loading:   true,
	}
}

func (m *SubnetModel) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	return m.loadSubnets(s)
}

func (m *SubnetModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case subnetsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.subnets = msg.subnets
		m.applyFilters()
		return m, nil

	case eniLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.detailErr = msg.err
			return m, nil
		}
		m.detailENIs = msg.enis
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

func (m *SubnetModel) View(s *shared.SharedState) string {
	var sections []string

	// Status bar
	sections = append(sections, renderStatusBar(s.Profile, s.Region, len(m.filtered), s.Width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading Subnets..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No Subnets found in this region."))
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
			overlay = RenderENIDetail(m.filtered[m.cursor], m.detailENIs, m.detailLoading, m.detailErr)
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

func (m *SubnetModel) ShortHelp() string {
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

func (m *SubnetModel) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		return m, m.loadSubnets(s)
	}

	return m, nil
}

func (m *SubnetModel) updateSearch(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

func (m *SubnetModel) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		case "enis":
			m.actionMenu.Active = false
			m.viewState = vsDetail
			m.showDetail = true
			m.detailLoading = true
			m.detailErr = nil
			sub := m.actionMenu.Subnet
			return m, loadENIs(sub.SubnetID, s.Profile, s.Region)
		case "goto_vpc":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabVPC}
			}
		}
	}
	return m, nil
}

func (m *SubnetModel) updateDetail(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

func (m *SubnetModel) applyFilters() {
	result := m.subnets

	if m.search.Query != "" {
		q := strings.ToLower(m.search.Query)
		var filtered []internalaws.Subnet
		for _, sub := range result {
			if strings.Contains(strings.ToLower(sub.Name), q) ||
				strings.Contains(strings.ToLower(sub.SubnetID), q) ||
				strings.Contains(strings.ToLower(sub.VpcID), q) ||
				strings.Contains(strings.ToLower(sub.CIDRBlock), q) ||
				strings.Contains(strings.ToLower(sub.AZ), q) {
				filtered = append(filtered, sub)
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

func (m *SubnetModel) loadSubnets(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return subnetsLoadedMsg{err: err}
		}
		subnets, err := internalaws.FetchSubnets(ctx, clients.EC2)
		if err != nil {
			return subnetsLoadedMsg{err: err}
		}
		return subnetsLoadedMsg{subnets: subnets}
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
