package tab_ecs

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsActionMenu
	vsDetail
)

type clustersLoadedMsg struct {
	clusters []internalaws.ECSCluster
	err      error
}

type servicesLoadedMsg struct {
	services []internalaws.ECSService
	err      error
}

type Action struct {
	Key   string
	Label string
}

type ActionMenuModel struct {
	Active  bool
	Cluster internalaws.ECSCluster
	Actions []Action
	Cursor  int
}

func newActionMenu(c internalaws.ECSCluster) ActionMenuModel {
	return ActionMenuModel{
		Active:  true,
		Cluster: c,
		Actions: []Action{
			{Key: "detail", Label: "Cluster Details"},
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
	b.WriteString(fmt.Sprintf("  %s (%s)\n", a.Cluster.Name, a.Cluster.Status))
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

type SearchModel struct {
	Query  string
	Active bool
}

func (s *SearchModel) Insert(char rune) { s.Query += string(char) }
func (s *SearchModel) Backspace() {
	if len(s.Query) > 0 {
		s.Query = s.Query[:len(s.Query)-1]
	}
}
func (s *SearchModel) Clear() { s.Query = ""; s.Active = false }
func (s *SearchModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	prompt := shared.SearchPromptStyle.Render(" /")
	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s %s█", prompt, s.Query))
}

// ECSModel implements the shared.TabModel interface for the ECS tab.
type ECSModel struct {
	viewState     viewState
	loading       bool
	err           error
	clusters      []internalaws.ECSCluster
	filtered      []internalaws.ECSCluster
	cursor        int
	search        SearchModel
	actionMenu    ActionMenuModel
	showDetail    bool
	detailServices []internalaws.ECSService
	detailLoading bool
}

func New() *ECSModel {
	return &ECSModel{viewState: vsTable, loading: true}
}

func (m *ECSModel) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	return m.loadData(s)
}

func (m *ECSModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil
	case clustersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.clusters = msg.clusters
		m.applyFilters()
		return m, nil
	case servicesLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			return m, nil
		}
		m.detailServices = msg.services
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

func (m *ECSModel) View(s *shared.SharedState) string {
	var sections []string
	sections = append(sections, renderStatusBar(s.Profile, s.Region, len(m.filtered), s.Width))
	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading ECS Clusters..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No ECS clusters found in this region."))
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
	case m.showDetail:
		if m.cursor < len(m.filtered) {
			overlay = RenderDetail(m.filtered[m.cursor], m.detailServices, m.detailLoading)
		}
	case m.actionMenu.Active:
		overlay = m.actionMenu.Render(s.Width)
	}

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view = shared.PlaceOverlay(s.Width, s.Height, overlay)
	}
	return view
}

func (m *ECSModel) ShortHelp() string {
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

func (m *ECSModel) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		return m, m.loadData(s)
	}
	return m, nil
}

func (m *ECSModel) updateSearch(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

func (m *ECSModel) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		if m.actionMenu.Selected() == "detail" {
			m.actionMenu.Active = false
			m.viewState = vsDetail
			m.showDetail = true
			m.detailLoading = true
			m.detailServices = nil
			return m, m.loadServices(s, m.actionMenu.Cluster.ARN)
		}
	}
	return m, nil
}

func (m *ECSModel) updateDetail(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	if _, ok := msg.(tea.KeyPressMsg); ok {
		m.showDetail = false
		m.detailServices = nil
		m.viewState = vsTable
	}
	return m, nil
}

func (m *ECSModel) applyFilters() {
	result := m.clusters
	if m.search.Query != "" {
		q := strings.ToLower(m.search.Query)
		var filtered []internalaws.ECSCluster
		for _, c := range result {
			if strings.Contains(internalaws.ECSSearchFields(c), q) {
				filtered = append(filtered, c)
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

func (m *ECSModel) loadData(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return clustersLoadedMsg{err: err}
		}
		clusters, err := internalaws.FetchECSClusters(ctx, clients.ECS)
		if err != nil {
			return clustersLoadedMsg{err: err}
		}
		return clustersLoadedMsg{clusters: clusters}
	}
}

func (m *ECSModel) loadServices(s *shared.SharedState, clusterARN string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return servicesLoadedMsg{err: err}
		}
		services, err := internalaws.FetchECSServices(ctx, clients.ECS, clusterARN)
		if err != nil {
			return servicesLoadedMsg{err: err}
		}
		return servicesLoadedMsg{services: services}
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
