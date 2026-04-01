package tab_elb

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the ELB tab's internal view mode.
type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsActionMenu
	vsDetail
	vsTGDetail // target group detail sub-view
)

// elbLoadedMsg is returned when load balancers are fetched.
type elbLoadedMsg struct {
	lbs []internalaws.LoadBalancer
	err error
}

// detailLoadedMsg is returned when detail (listeners + target groups) are fetched.
type detailLoadedMsg struct {
	listeners    []internalaws.Listener
	targetGroups []internalaws.TargetGroup
	err          error
}

// tgTargetsLoadedMsg is returned when target group targets are fetched.
type tgTargetsLoadedMsg struct {
	targets []internalaws.Target
	err     error
}

// Action represents a menu action for a load balancer.
type Action struct {
	Key   string
	Label string
}

// ActionMenuModel manages the action menu state.
type ActionMenuModel struct {
	Active bool
	LB     internalaws.LoadBalancer
	Actions []Action
	Cursor  int
}

func newActionMenu(lb internalaws.LoadBalancer) ActionMenuModel {
	return ActionMenuModel{
		Active: true,
		LB:     lb,
		Actions: []Action{
			{Key: "detail", Label: "ELB Details"},
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
	b.WriteString(fmt.Sprintf("  %s  (%s)\n", a.LB.Name, a.LB.TypeLabel()))
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

// ELBModel implements the shared.TabModel interface for the ELB tab.
type ELBModel struct {
	viewState viewState
	loading   bool
	err       error

	lbs      []internalaws.LoadBalancer
	filtered []internalaws.LoadBalancer
	cursor   int

	search     SearchModel
	actionMenu ActionMenuModel

	// Detail overlay — interactive: cursor selects target groups
	showDetail    bool
	detailLoading bool
	detailLB      *internalaws.LoadBalancer
	detailCursor  int // cursor within detail view (0=info, 1..N=target groups)
	detailSection string // "" = main detail, "tg" = target group detail

	// Target group detail sub-view
	selectedTG      *internalaws.TargetGroup
	tgTargets       []internalaws.Target
	tgTargetsLoading bool
}

// New creates a new ELBModel.
func New() *ELBModel {
	return &ELBModel{
		viewState: vsTable,
		loading:   true,
	}
}

func (m *ELBModel) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	return m.loadELBs(s)
}

func (m *ELBModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case elbLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.lbs = msg.lbs
		m.applyFilters()
		return m, nil

	case detailLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			return m, nil
		}
		if m.detailLB != nil {
			m.detailLB.Listeners = msg.listeners
			m.detailLB.TargetGroups = msg.targetGroups
		}
		return m, nil

	case tgTargetsLoadedMsg:
		m.tgTargetsLoading = false
		if msg.err != nil {
			return m, nil
		}
		m.tgTargets = msg.targets
		if m.selectedTG != nil {
			m.selectedTG.Targets = msg.targets
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
	case vsTGDetail:
		return m.updateTGDetail(msg, s)
	default:
		return m.updateTable(msg, s)
	}
}

func (m *ELBModel) View(s *shared.SharedState) string {
	var sections []string

	// Status bar
	sections = append(sections, renderStatusBar(s.Profile, s.Region, len(m.filtered), s.Width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading Load Balancers..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No Load Balancers found in this region."))
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
	case m.viewState == vsTGDetail && m.selectedTG != nil:
		overlay = RenderTGDetail(*m.selectedTG, m.tgTargets, m.tgTargetsLoading)
	case m.showDetail && m.detailLB != nil:
		overlay = RenderELBDetailInteractive(*m.detailLB, m.detailLoading, m.detailCursor)
	case m.actionMenu.Active:
		overlay = m.actionMenu.Render(s.Width)
	}

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view = shared.PlaceOverlay(s.Width, s.Height, overlay)
	}

	return view
}

func (m *ELBModel) ShortHelp() string {
	switch m.viewState {
	case vsSearch:
		return helpLine("Esc", "Cancel")
	case vsActionMenu:
		return helpLine("↑↓", "Navigate", "Enter", "Select", "Esc", "Cancel")
	case vsDetail:
		return helpLine("↑↓", "Select TG", "Enter", "TG Detail", "Esc", "Close")
	case vsTGDetail:
		return helpLine("Esc", "Back to ELB")
	default:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "/", "Search", "R", "Refresh")
	}
}

// --- Internal update handlers ---

func (m *ELBModel) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
		return m, m.loadELBs(s)
	}

	return m, nil
}

func (m *ELBModel) updateSearch(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

func (m *ELBModel) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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

			// Copy the selected LB for detail view
			if m.cursor < len(m.filtered) {
				lb := m.filtered[m.cursor]
				m.detailLB = &lb

				// For non-classic LBs, fetch listeners and target groups on demand
				if lb.Type != "classic" && lb.ARN != "" {
					m.detailLoading = true
					return m, m.loadDetail(s, lb.ARN)
				}
			}
		}
	}
	return m, nil
}

func (m *ELBModel) updateDetail(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if m.detailLoading {
		if keyMsg.String() == "esc" {
			m.showDetail = false
			m.detailLB = nil
			m.viewState = vsTable
		}
		return m, nil
	}

	tgCount := 0
	if m.detailLB != nil {
		tgCount = len(m.detailLB.TargetGroups)
	}

	switch keyMsg.String() {
	case "esc":
		m.showDetail = false
		m.detailLB = nil
		m.detailCursor = 0
		m.viewState = vsTable
	case "up", "k":
		if m.detailCursor > 0 {
			m.detailCursor--
		}
	case "down", "j":
		if m.detailCursor < tgCount-1 {
			m.detailCursor++
		}
	case "enter":
		if tgCount > 0 && m.detailCursor < tgCount {
			tg := m.detailLB.TargetGroups[m.detailCursor]
			m.selectedTG = &tg
			m.tgTargets = nil
			m.tgTargetsLoading = true
			m.viewState = vsTGDetail
			return m, m.loadTGTargets(s, tg.ARN)
		}
	}
	return m, nil
}

func (m *ELBModel) updateTGDetail(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.selectedTG = nil
		m.tgTargets = nil
		m.viewState = vsDetail
	}
	return m, nil
}

func (m *ELBModel) loadTGTargets(s *shared.SharedState, tgARN string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return tgTargetsLoadedMsg{err: err}
		}
		targets, err := internalaws.FetchTargets(ctx, clients.ELBv2, tgARN)
		return tgTargetsLoadedMsg{targets: targets, err: err}
	}
}

// --- Helpers ---

func (m *ELBModel) applyFilters() {
	result := m.lbs

	if m.search.Query != "" {
		q := strings.ToLower(m.search.Query)
		var filtered []internalaws.LoadBalancer
		for _, lb := range result {
			if strings.Contains(strings.ToLower(lb.Name), q) ||
				strings.Contains(strings.ToLower(lb.Type), q) ||
				strings.Contains(strings.ToLower(lb.TypeLabel()), q) ||
				strings.Contains(strings.ToLower(lb.DNSName), q) ||
				strings.Contains(strings.ToLower(lb.VpcID), q) ||
				strings.Contains(strings.ToLower(lb.Scheme), q) {
				filtered = append(filtered, lb)
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

func (m *ELBModel) loadELBs(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return elbLoadedMsg{err: err}
		}

		// Fetch elbv2 (ALB, NLB, GWLB) and classic (CLB) in parallel
		var v2Lbs []internalaws.LoadBalancer
		var v2Err error
		var clbLbs []internalaws.LoadBalancer
		var clbErr error

		done := make(chan struct{}, 2)

		go func() {
			v2Lbs, v2Err = internalaws.FetchLoadBalancers(ctx, clients.ELBv2)
			done <- struct{}{}
		}()

		go func() {
			clbLbs, clbErr = internalaws.FetchClassicLoadBalancers(ctx, clients.ELB)
			done <- struct{}{}
		}()

		<-done
		<-done

		// Return first error encountered
		if v2Err != nil {
			return elbLoadedMsg{err: v2Err}
		}
		if clbErr != nil {
			return elbLoadedMsg{err: clbErr}
		}

		all := append(v2Lbs, clbLbs...)
		return elbLoadedMsg{lbs: all}
	}
}

func (m *ELBModel) loadDetail(s *shared.SharedState, lbARN string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return detailLoadedMsg{err: err}
		}

		var listeners []internalaws.Listener
		var targetGroups []internalaws.TargetGroup
		var lErr, tErr error

		done := make(chan struct{}, 2)

		go func() {
			listeners, lErr = internalaws.FetchListeners(ctx, clients.ELBv2, lbARN)
			done <- struct{}{}
		}()

		go func() {
			targetGroups, tErr = internalaws.FetchTargetGroups(ctx, clients.ELBv2, lbARN)
			done <- struct{}{}
		}()

		<-done
		<-done

		if lErr != nil {
			return detailLoadedMsg{err: lErr}
		}
		if tErr != nil {
			return detailLoadedMsg{err: tErr}
		}

		return detailLoadedMsg{
			listeners:    listeners,
			targetGroups: targetGroups,
		}
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
