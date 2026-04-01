package tab_ec2

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/store"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the EC2 tab's internal view mode.
type viewState int

const (
	vsTable viewState = iota
	vsSearch
	vsFilter
	vsPortForward
	vsActionMenu
)

// networkPathData holds all network-path info for a single instance.
type networkPathData struct {
	VPC        internalaws.VPC
	Subnet     internalaws.Subnet
	RouteTable internalaws.RouteTable
	SGs        []internalaws.SecurityGroup
	NACL       internalaws.NetworkACL
}

// networkPathLoadedMsg is sent when the network-path data has been fetched.
type networkPathLoadedMsg struct {
	data networkPathData
	err  error
}

// Messages internal to the EC2 tab.
type instancesLoadedMsg struct {
	instances []internalaws.Instance
	ssmStatus map[string]bool
	err       error
}

// SSMSessionDoneMsg is sent when an SSM session or port-forward completes.
// It is exported because root.go needs to pattern-match on it to record history.
type SSMSessionDoneMsg struct {
	Err         error
	InstanceID  string
	Profile     string
	Region      string
	Alias       string
	SessionType string // "session" or "port_forward"
}

// SSMExecRequest is sent by the EC2 tab to ask the root model to execute
// an SSM command via tea.Exec (which only the root model can do).
type SSMExecRequest struct {
	InstanceID string
	Profile    string
	Region     string
	Alias      string
	Args       []string
	Type       string // "session" or "port_forward"
}

// Sort columns cycle
var sortColumns = []string{"name", "id", "state", "type", "az"}

// EC2Model implements the shared.TabModel interface for the EC2 instances tab.
type EC2Model struct {
	// Internal view state
	viewState viewState
	loading   bool
	err       error

	// Data
	instances []internalaws.Instance
	filtered  []internalaws.Instance
	cursor    int

	// UI Components
	search      SearchModel
	filter      FilterModel
	portForward PortForwardModel
	actionMenu  ActionMenuModel
	showDetail     string // "sg", "detail", or "netpath" for info overlays
	netPathData    *networkPathData
	netPathLoading bool

	// Sort
	sortBy    string
	sortOrder string
	sortIdx   int
}

// New creates a new EC2Model.
func New(sortBy, sortOrder string) *EC2Model {
	return &EC2Model{
		viewState: vsTable,
		loading:   true,
		filter:    NewFilterModel(),
		sortBy:    sortBy,
		sortOrder: sortOrder,
		sortIdx:   0,
	}
}

func (m *EC2Model) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	return m.loadInstances(s)
}

func (m *EC2Model) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Handled by root, but we accept it here gracefully.
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
		m.applyFilters(s)
		return m, nil

	case networkPathLoadedMsg:
		m.netPathLoading = false
		if msg.err != nil {
			m.err = msg.err
			m.showDetail = ""
			return m, nil
		}
		m.netPathData = &msg.data
		return m, nil

	case SSMSessionDoneMsg:
		m.viewState = vsTable
		if msg.Err != nil {
			m.err = fmt.Errorf("SSM session failed for %s: %v", msg.Alias, msg.Err)
			return m, nil
		}
		m.err = nil
		m.loading = true
		return m, m.loadInstances(s)
	}

	// Dispatch based on internal view state
	switch m.viewState {
	case vsSearch:
		return m.updateSearch(msg, s)
	case vsFilter:
		return m.updateFilter(msg, s)
	case vsPortForward:
		return m.updatePortForward(msg, s)
	case vsActionMenu:
		return m.updateActionMenu(msg, s)
	default:
		return m.updateTable(msg, s)
	}
}

func (m *EC2Model) View(s *shared.SharedState) string {
	var sections []string

	// Status bar
	sections = append(sections, renderStatusBar(s.Profile, s.Region, m.filter.Label(), len(m.filtered), s.Width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(s.Width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading instances..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No instances found in this region."))
	} else {
		columns := ColumnsForWidth(s.Width)
		tableHeight := s.Height
		if m.search.Active {
			tableHeight--
		}
		sections = append(sections, RenderTable(m.filtered, columns, m.cursor, s.Favorites, s.History, s.Profile, s.Region, s.Width, tableHeight))
	}

	// Overlay
	overlay := ""
	switch {
	case m.showDetail == "netpath" && m.actionMenu.Active:
		if m.netPathLoading {
			overlay = shared.RenderOverlay("  Loading network path...")
		} else if m.netPathData != nil {
			overlay = RenderNetworkPath(m.actionMenu.Instance, *m.netPathData)
		}
	case m.showDetail == "sg" && m.actionMenu.Active:
		overlay = RenderSecurityGroups(m.actionMenu.Instance)
	case m.showDetail == "detail" && m.actionMenu.Active:
		overlay = RenderInstanceDetail(m.actionMenu.Instance)
	case m.actionMenu.Active:
		overlay = m.actionMenu.Render(s.Width)
	case m.filter.Active:
		overlay = m.filter.Render(s.Width)
	case m.portForward.Active:
		overlay = m.renderPortForward()
	}

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view += "\n" + shared.PlaceOverlay(s.Width, overlay)
	}

	return view
}

func (m *EC2Model) ShortHelp() string {
	switch m.viewState {
	case vsSearch:
		return helpLine(
			"Enter", "Connect",
			"Esc", "Cancel",
		)
	case vsFilter:
		return helpLine(
			"↑↓", "Navigate",
			"Enter", "Select",
			"Esc", "Cancel",
		)
	case vsPortForward:
		return helpLine(
			"Enter", "Start",
			"Esc", "Cancel",
		)
	case vsActionMenu:
		return helpLine(
			"↑↓", "Navigate",
			"Enter", "Select",
			"Esc", "Cancel",
		)
	default:
		return helpLine(
			"↑↓", "Navigate",
			"Enter", "Connect",
			"/", "Search",
			"f", "Filter",
			"s", "Sort",
			"F", "Fav",
			"P", "Port Fwd",
			"R", "Refresh",
		)
	}
}

// --- Internal update handlers ---

func (m *EC2Model) updateTable(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
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
			m.actionMenu = NewActionMenu(m.filtered[m.cursor])
			m.viewState = vsActionMenu
		}

	case "/":
		m.viewState = vsSearch
		m.search.Active = true
		m.search.Query = ""

	case "f":
		m.viewState = vsFilter
		m.filter.Active = true

	case "s":
		m.sortIdx = (m.sortIdx + 1) % len(sortColumns)
		m.sortBy = sortColumns[m.sortIdx]
		m.applyFilters(s)

	case "S":
		if m.sortOrder == "asc" {
			m.sortOrder = "desc"
		} else {
			m.sortOrder = "asc"
		}
		m.applyFilters(s)

	case "F":
		if m.cursor < len(m.filtered) {
			inst := m.filtered[m.cursor]
			if s.Favorites.IsFavorite(inst.InstanceID, s.Profile, s.Region) {
				s.Favorites.Remove(inst.InstanceID, s.Profile, s.Region)
			} else {
				s.Favorites.Add(store.Favorite{
					InstanceID: inst.InstanceID,
					Profile:    s.Profile,
					Region:     s.Region,
					Alias:      inst.DisplayName(),
				})
			}
			s.Favorites.Save(store.FavoritesPath())
			m.applyFilters(s)
		}

	case "P":
		if m.cursor < len(m.filtered) {
			m.viewState = vsPortForward
			m.portForward = PortForwardModel{Active: true, LocalPort: "8080", RemotePort: "80", Field: 0}
		}

	case "R":
		m.loading = true
		m.err = nil
		return m, m.loadInstances(s)
	}

	return m, nil
}

func (m *EC2Model) updateSearch(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.search.Clear()
		m.viewState = vsTable
		m.applyFilters(s)
	case "enter":
		m.viewState = vsTable
		m.search.Active = false
		if m.cursor < len(m.filtered) {
			return m, m.requestSSMSession(m.filtered[m.cursor], s)
		}
	case "backspace":
		m.search.Backspace()
		m.applyFilters(s)
	default:
		r := keyMsg.String()
		if len(r) == 1 {
			m.search.Insert(rune(r[0]))
			m.applyFilters(s)
			m.cursor = 0
		}
	}
	return m, nil
}

func (m *EC2Model) updateFilter(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc", "f":
		m.filter.Active = false
		m.viewState = vsTable
		m.applyFilters(s)
	case "up", "k":
		m.filter.MoveUp()
	case "down", "j":
		m.filter.MoveDown()
	case " ", "enter":
		m.filter.Toggle()
		m.applyFilters(s)
	case "c":
		m.filter.ClearAll()
		m.applyFilters(s)
	}
	return m, nil
}

func (m *EC2Model) updatePortForward(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.portForward.Active = false
		m.viewState = vsTable
	case "tab":
		m.portForward.Field = (m.portForward.Field + 1) % 2
	case "enter":
		if m.cursor < len(m.filtered) {
			m.portForward.Active = false
			return m, m.requestPortForward(m.filtered[m.cursor], s)
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

func (m *EC2Model) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	// If showing a detail overlay (sg/detail/netpath), any key closes it
	if m.showDetail != "" {
		// While netpath is still loading, only esc can cancel
		if m.showDetail == "netpath" && m.netPathLoading {
			if keyMsg.String() == "esc" {
				m.showDetail = ""
				m.netPathLoading = false
				m.netPathData = nil
				m.viewState = vsTable
				m.actionMenu.Active = false
			}
			return m, nil
		}
		m.showDetail = ""
		m.netPathData = nil
		m.netPathLoading = false
		m.viewState = vsTable
		m.actionMenu.Active = false
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
		inst := m.actionMenu.Instance
		switch action {
		case "ssm":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, m.requestSSMSession(inst, s)
		case "portfwd":
			m.actionMenu.Active = false
			m.viewState = vsPortForward
			m.portForward = PortForwardModel{Active: true, LocalPort: "8080", RemotePort: "80", Field: 0}
		case "netpath":
			m.showDetail = "netpath"
			m.netPathData = nil
			m.netPathLoading = true
			return m, loadNetworkPath(s.Profile, s.Region, inst)
		case "sg":
			m.showDetail = "sg"
		case "detail":
			m.showDetail = "detail"
		case "goto_vpc":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabVPC}
			}
		case "goto_subnet":
			m.actionMenu.Active = false
			m.viewState = vsTable
			return m, func() tea.Msg {
				return shared.NavigateToTab{Tab: shared.TabSubnet}
			}
		}
	}
	return m, nil
}

// --- Helpers ---

func (m *EC2Model) applyFilters(s *shared.SharedState) {
	result := m.instances

	// State filter
	result = FilterByState(result, m.filter.ActiveStates)

	// Search filter
	result = FilterBySearch(result, m.search.Query)

	// Sort
	result = SortInstances(result, s.Favorites, s.History, s.Profile, s.Region, m.sortBy, m.sortOrder)

	m.filtered = result

	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *EC2Model) loadInstances(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		instances, err := internalaws.FetchInstances(ctx, clients.EC2)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		internalaws.EnrichVpcSubnetInfo(ctx, clients.EC2, instances)
		ssmStatus, _ := internalaws.FetchSSMStatus(ctx, clients.SSM)

		return instancesLoadedMsg{
			instances: instances,
			ssmStatus: ssmStatus,
		}
	}
}

func (m *EC2Model) requestSSMSession(inst internalaws.Instance, s *shared.SharedState) tea.Cmd {
	args := internalaws.BuildSSMSessionArgs(inst.InstanceID, s.Profile, s.Region)
	return func() tea.Msg {
		return SSMExecRequest{
			InstanceID: inst.InstanceID,
			Profile:    s.Profile,
			Region:     s.Region,
			Alias:      inst.DisplayName(),
			Args:       args,
			Type:       "session",
		}
	}
}

func (m *EC2Model) requestPortForward(inst internalaws.Instance, s *shared.SharedState) tea.Cmd {
	args := internalaws.BuildPortForwardArgs(inst.InstanceID, s.Profile, s.Region, m.portForward.LocalPort, m.portForward.RemotePort)
	return func() tea.Msg {
		return SSMExecRequest{
			InstanceID: inst.InstanceID,
			Profile:    s.Profile,
			Region:     s.Region,
			Alias:      inst.DisplayName(),
			Args:       args,
			Type:       "port_forward",
		}
	}
}

func loadNetworkPath(profile, region string, inst internalaws.Instance) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return networkPathLoadedMsg{err: err}
		}

		// Fetch route tables — find the one for this subnet, fallback to main RT for VPC
		rts, _ := internalaws.FetchRouteTables(ctx, clients.EC2)
		var rt internalaws.RouteTable
		for _, r := range rts {
			for _, sid := range r.Subnets {
				if sid == inst.SubnetID {
					rt = r
					break
				}
			}
			if rt.ID != "" {
				break
			}
		}
		if rt.ID == "" {
			for _, r := range rts {
				if r.VpcID == inst.VpcID && r.IsMain {
					rt = r
					break
				}
			}
		}

		// Fetch security groups — match by VPC and name
		allSGs, _ := internalaws.FetchSecurityGroups(ctx, clients.EC2)
		var sgs []internalaws.SecurityGroup
		for _, sg := range allSGs {
			if sg.VpcID == inst.VpcID {
				for _, instSG := range inst.SecurityGroups {
					if sg.Name == instSG {
						sgs = append(sgs, sg)
					}
				}
			}
		}

		// Fetch NACLs — find the one for this subnet
		nacls, _ := internalaws.FetchNetworkACLs(ctx, clients.EC2)
		var nacl internalaws.NetworkACL
		for _, n := range nacls {
			for _, sid := range n.Subnets {
				if sid == inst.SubnetID {
					nacl = n
					break
				}
			}
			if nacl.ID != "" {
				break
			}
		}

		return networkPathLoadedMsg{
			data: networkPathData{
				VPC: internalaws.VPC{
					VpcID:     inst.VpcID,
					Name:      inst.VpcName,
					CIDRBlock: inst.VpcCIDR,
				},
				Subnet: internalaws.Subnet{
					SubnetID:  inst.SubnetID,
					Name:      inst.SubnetName,
					CIDRBlock: inst.SubnetCIDR,
				},
				RouteTable: rt,
				SGs:        sgs,
				NACL:       nacl,
			},
		}
	}
}

func (m *EC2Model) renderPortForward() string {
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

	return shared.RenderOverlay(b.String())
}

func renderStatusBar(profile, region, filter string, count int, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	filterPart := shared.StatusKeyStyle.Render("Filter: ") + filter
	countPart := fmt.Sprintf("[%d instances]", count)

	content := fmt.Sprintf(" %s  ┊  %s  ┊  %s  ┊  %s", profilePart, regionPart, filterPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
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
