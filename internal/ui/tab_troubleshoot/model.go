package tab_troubleshoot

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// viewState tracks the troubleshoot tab's internal view mode.
type viewState int

const (
	vsForm viewState = iota
	vsPicker
	vsResult
	vsConfirmRA // confirm reachability analyzer
	vsRARunning // reachability analyzer running
	vsRAResult  // reachability analyzer result
)

// form field indices
const (
	fieldSource   = 0
	fieldDest     = 1
	fieldProtocol = 2
	fieldPort     = 3
)

// Message types
type instancesLoadedMsg struct {
	instances []internalaws.Instance
	err       error
}

type checkDataLoadedMsg struct {
	routeTables    []internalaws.RouteTable
	securityGroups []internalaws.SecurityGroup
	nacls          []internalaws.NetworkACL
	subnets        []internalaws.Subnet
	err            error
}

type raResultMsg struct {
	result *internalaws.ReachabilityResult
	err    error
}

// TroubleshootModel implements the shared.TabModel interface for the Troubleshoot tab.
type TroubleshootModel struct {
	viewState viewState
	loading   bool
	err       error

	// Form fields
	srcInst  *internalaws.Instance
	dstInst  *internalaws.Instance
	protocol string
	port     string
	field    int // 0=source, 1=dest, 2=protocol, 3=port

	// Instance picker
	picking         bool
	pickerInstances []internalaws.Instance
	pickerCursor    int
	pickerFor       int // which field the picker is for (fieldSource or fieldDest)

	// Check result
	result *CheckResult

	// Reachability Analyzer
	raResult *internalaws.ReachabilityResult
	raErr    error
}

// New creates a new TroubleshootModel.
func New() *TroubleshootModel {
	return &TroubleshootModel{
		viewState: vsForm,
		protocol:  "tcp",
		port:      "443",
		field:     fieldSource,
	}
}

func (m *TroubleshootModel) Init(s *shared.SharedState) tea.Cmd {
	m.err = nil
	return nil
}

func (m *TroubleshootModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case instancesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.viewState = vsForm
			return m, nil
		}
		m.pickerInstances = msg.instances
		m.pickerCursor = 0
		m.viewState = vsPicker
		return m, nil

	case checkDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		result := CheckConnectivity(
			*m.srcInst, *m.dstInst,
			m.protocol, m.port,
			msg.routeTables, msg.securityGroups, msg.nacls, msg.subnets,
		)
		m.result = &result
		m.viewState = vsResult
		return m, nil

	case raResultMsg:
		m.loading = false
		if msg.err != nil {
			m.raErr = msg.err
			m.viewState = vsRAResult
			return m, nil
		}
		m.raResult = msg.result
		m.viewState = vsRAResult
		return m, nil
	}

	switch m.viewState {
	case vsPicker:
		return m.updatePicker(msg, s)
	case vsResult:
		return m.updateResult(msg, s)
	case vsConfirmRA:
		return m.updateConfirmRA(msg, s)
	case vsRARunning:
		return m, nil // waiting for result
	case vsRAResult:
		return m.updateRAResult(msg, s)
	default:
		return m.updateForm(msg, s)
	}
}

func (m *TroubleshootModel) View(s *shared.SharedState) string {
	var sections []string

	// Status bar
	sections = append(sections, renderStatusBar(s.Profile, s.Region, s.Width))

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress Esc to go back", m.err)),
		))
	} else {
		switch m.viewState {
		case vsForm:
			sections = append(sections, m.renderForm(s))
		case vsPicker:
			sections = append(sections, m.renderPicker(s))
		case vsResult:
			sections = append(sections, m.renderResult(s))
		case vsConfirmRA:
			sections = append(sections, m.renderResult(s))
			sections = append(sections, m.renderRAConfirm())
		case vsRARunning:
			sections = append(sections, m.renderResult(s))
			sections = append(sections, lipgloss.NewStyle().Padding(0, 2).Render("  Running Reachability Analyzer..."))
		case vsRAResult:
			sections = append(sections, m.renderResult(s))
			sections = append(sections, m.renderRAResult())
		}
	}

	return strings.Join(sections, "\n")
}

func (m *TroubleshootModel) ShortHelp() string {
	switch m.viewState {
	case vsPicker:
		return helpLine("up/dn", "Navigate", "Enter", "Select", "Esc", "Cancel")
	case vsResult:
		return helpLine("Esc", "Back to form", "R", "Reachability Analyzer")
	case vsConfirmRA:
		return helpLine("y", "Yes", "n", "No")
	case vsRARunning:
		return helpLine("Esc", "Cancel")
	case vsRAResult:
		return helpLine("Esc", "Back to result")
	default:
		return helpLine("Tab", "Switch field", "Enter", "Pick instance", "c", "Check")
	}
}

// --- Form update ---

func (m *TroubleshootModel) updateForm(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "tab":
		m.field = (m.field + 1) % 4

	case "shift+tab":
		m.field = (m.field - 1 + 4) % 4

	case "enter":
		if m.field == fieldSource || m.field == fieldDest {
			m.loading = true
			m.pickerFor = m.field
			return m, m.loadInstances(s)
		}

	case "c":
		if m.srcInst != nil && m.dstInst != nil {
			m.loading = true
			m.err = nil
			return m, m.loadCheckData(s)
		}

	case "backspace":
		switch m.field {
		case fieldProtocol:
			if len(m.protocol) > 0 {
				m.protocol = m.protocol[:len(m.protocol)-1]
			}
		case fieldPort:
			if len(m.port) > 0 {
				m.port = m.port[:len(m.port)-1]
			}
		}

	default:
		r := keyMsg.String()
		if len(r) == 1 {
			switch m.field {
			case fieldProtocol:
				m.protocol += r
			case fieldPort:
				if r[0] >= '0' && r[0] <= '9' {
					m.port += r
				}
			}
		}
	}

	return m, nil
}

// --- Picker update ---

func (m *TroubleshootModel) updatePicker(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.viewState = vsForm
	case "up", "k":
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
	case "down", "j":
		if m.pickerCursor < len(m.pickerInstances)-1 {
			m.pickerCursor++
		}
	case "enter":
		if m.pickerCursor < len(m.pickerInstances) {
			inst := m.pickerInstances[m.pickerCursor]
			if m.pickerFor == fieldSource {
				m.srcInst = &inst
			} else {
				m.dstInst = &inst
			}
			m.viewState = vsForm
		}
	}

	return m, nil
}

// --- Result update ---

func (m *TroubleshootModel) updateResult(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.viewState = vsForm
		m.result = nil
		m.raResult = nil
		m.raErr = nil
	case "R":
		if m.srcInst != nil && m.dstInst != nil {
			m.viewState = vsConfirmRA
		}
	}

	return m, nil
}

// --- Reachability Analyzer confirm ---

func (m *TroubleshootModel) updateConfirmRA(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y", "Y":
		m.viewState = vsRARunning
		m.loading = true
		m.raErr = nil
		m.raResult = nil
		return m, m.runReachabilityAnalysis(s)
	case "n", "N", "esc":
		m.viewState = vsResult
	}

	return m, nil
}

// --- RA result update ---

func (m *TroubleshootModel) updateRAResult(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.viewState = vsResult
		m.raResult = nil
		m.raErr = nil
	}

	return m, nil
}

// --- Render helpers ---

func (m *TroubleshootModel) renderForm(s *shared.SharedState) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  Connectivity Check\n")
	b.WriteString("  " + strings.Repeat("-", 35) + "\n")

	srcLabel := "(none selected)"
	if m.srcInst != nil {
		srcLabel = fmt.Sprintf("%s (%s)", m.srcInst.DisplayName(), m.srcInst.PrivateIP)
	}

	dstLabel := "(none selected)"
	if m.dstInst != nil {
		dstLabel = fmt.Sprintf("%s (%s)", m.dstInst.DisplayName(), m.dstInst.PrivateIP)
	}

	fields := []struct {
		label string
		value string
	}{
		{"Source:      ", srcLabel},
		{"Destination: ", dstLabel},
		{"Protocol:    ", m.protocol},
		{"Port:        ", m.port},
	}

	for i, f := range fields {
		cursor := "  "
		if i == m.field {
			cursor = "> "
		}
		line := fmt.Sprintf("  %s%s%s", cursor, f.label, f.value)
		if i == m.field && (i == fieldProtocol || i == fieldPort) {
			line += "_"
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")

	ready := m.srcInst != nil && m.dstInst != nil
	if ready {
		b.WriteString("  [Tab: switch field] [Enter: pick instance] [c: Check]\n")
	} else {
		b.WriteString("  [Tab: switch field] [Enter: pick instance]\n")
	}

	return lipgloss.NewStyle().Width(s.Width).Padding(1, 0).Render(b.String())
}

func (m *TroubleshootModel) renderPicker(s *shared.SharedState) string {
	var b strings.Builder

	label := "Select Source Instance"
	if m.pickerFor == fieldDest {
		label = "Select Destination Instance"
	}

	b.WriteString(fmt.Sprintf("\n  %s\n", label))
	b.WriteString("  " + strings.Repeat("-", 40) + "\n")

	if len(m.pickerInstances) == 0 {
		b.WriteString("  No instances found.\n")
		b.WriteString("\n  Esc: back\n")
		return lipgloss.NewStyle().Width(s.Width).Padding(1, 0).Render(b.String())
	}

	maxVisible := s.Height - 8
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if m.pickerCursor >= maxVisible {
		start = m.pickerCursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.pickerInstances) {
		end = len(m.pickerInstances)
	}

	if start > 0 {
		b.WriteString("    ^ more\n")
	}

	for i := start; i < end; i++ {
		inst := m.pickerInstances[i]
		cursor := "  "
		if i == m.pickerCursor {
			cursor = "> "
		}
		name := inst.DisplayName()
		ip := inst.PrivateIP
		if ip == "" {
			ip = "-"
		}
		vpc := inst.VpcName
		if vpc == "" {
			vpc = inst.VpcID
		}
		line := fmt.Sprintf("  %s%-20s %-16s %s", cursor, truncate(name, 20), ip, truncate(vpc, 20))
		b.WriteString(line + "\n")
	}

	if end < len(m.pickerInstances) {
		b.WriteString("    v more\n")
	}

	b.WriteString(fmt.Sprintf("\n  [%d instances] Enter: select  Esc: cancel\n", len(m.pickerInstances)))

	return lipgloss.NewStyle().Width(s.Width).Padding(1, 0).Render(b.String())
}

func (m *TroubleshootModel) renderResult(s *shared.SharedState) string {
	if m.result == nil {
		return ""
	}

	srcName := m.srcInst.DisplayName()
	dstName := m.dstInst.DisplayName()

	rendered := RenderResult(*m.result, srcName, dstName, m.protocol, m.port)

	return lipgloss.NewStyle().Width(s.Width).Padding(1, 0).Render(rendered)
}

func (m *TroubleshootModel) renderRAConfirm() string {
	return lipgloss.NewStyle().Padding(0, 2).Render(
		shared.RenderOverlay(
			"  This will call AWS Reachability Analyzer API\n" +
				"  (may incur costs). Continue?\n\n" +
				"  y: Yes  n: No",
		),
	)
}

func (m *TroubleshootModel) renderRAResult() string {
	var b strings.Builder

	b.WriteString("\n  Reachability Analyzer Result\n")
	b.WriteString("  " + strings.Repeat("-", 40) + "\n")

	if m.raErr != nil {
		b.WriteString(fmt.Sprintf("  Error: %v\n", m.raErr))
	} else if m.raResult != nil {
		if m.raResult.Reachable {
			b.WriteString("  v REACHABLE (confirmed by AWS)\n")
		} else {
			b.WriteString("  x NOT REACHABLE (confirmed by AWS)\n")
		}
		b.WriteString(fmt.Sprintf("  Analysis ID: %s\n", m.raResult.AnalysisID))
		if len(m.raResult.Explanations) > 0 {
			b.WriteString("\n  Explanations:\n")
			for _, exp := range m.raResult.Explanations {
				b.WriteString(fmt.Sprintf("    - %s\n", exp))
			}
		}
	}

	b.WriteString("\n  Esc: back to result\n")

	return lipgloss.NewStyle().Padding(0, 2).Render(b.String())
}

// --- Data loading ---

func (m *TroubleshootModel) loadInstances(s *shared.SharedState) tea.Cmd {
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
		return instancesLoadedMsg{instances: instances}
	}
}

func (m *TroubleshootModel) loadCheckData(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return checkDataLoadedMsg{err: err}
		}

		rts, err := internalaws.FetchRouteTables(ctx, clients.EC2)
		if err != nil {
			return checkDataLoadedMsg{err: err}
		}

		sgs, err := internalaws.FetchSecurityGroups(ctx, clients.EC2)
		if err != nil {
			return checkDataLoadedMsg{err: err}
		}

		nacls, err := internalaws.FetchNetworkACLs(ctx, clients.EC2)
		if err != nil {
			return checkDataLoadedMsg{err: err}
		}

		subnets, err := internalaws.FetchSubnets(ctx, clients.EC2)
		if err != nil {
			return checkDataLoadedMsg{err: err}
		}

		return checkDataLoadedMsg{
			routeTables:    rts,
			securityGroups: sgs,
			nacls:          nacls,
			subnets:        subnets,
		}
	}
}

func (m *TroubleshootModel) runReachabilityAnalysis(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	srcID := m.srcInst.InstanceID
	dstID := m.dstInst.InstanceID
	protocol := m.protocol
	port := m.port
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return raResultMsg{err: err}
		}

		portNum := 443
		if p := port; p != "" {
			fmt.Sscanf(p, "%d", &portNum)
		}

		result, err := internalaws.RunReachabilityAnalysis(ctx, clients.EC2, srcID, dstID, protocol, portNum)
		return raResultMsg{result: result, err: err}
	}
}

func renderStatusBar(profile, region string, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region

	content := fmt.Sprintf(" %s  |  %s  |  Connectivity Check", profilePart, regionPart)
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
