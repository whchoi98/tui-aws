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

// viewState tracks the drill-down level in the ECS hierarchy.
type viewState int

const (
	vsClusterList    viewState = iota // top-level cluster table
	vsClusterSearch                   // search within cluster list
	vsClusterAction                   // action menu on a cluster
	vsServiceList                     // service list for a cluster
	vsServiceAction                   // action menu on a service
	vsServiceDetail                   // service detail overlay
	vsTaskList                        // task list for a cluster or service
	vsTaskAction                      // action menu on a task
	vsTaskDetail                      // task detail overlay
	vsTaskDefDetail                   // task definition detail overlay
	vsContainerList                   // container list for a task
	vsContainerAction                 // action menu on a container
	vsContainerDetail                 // container detail overlay
	vsLogs                            // log viewer overlay
	vsClusterDetail                   // cluster detail overlay
)

// ECSExecRequest is sent to the root model to run `aws ecs execute-command`.
type ECSExecRequest struct {
	ClusterARN    string
	TaskARN       string
	ContainerName string
	Profile       string
	Region        string
	Args          []string
}

// ECSExecDoneMsg is sent after ECS exec completes.
type ECSExecDoneMsg struct {
	Err error
}

// --- async messages ---

type clustersLoadedMsg struct {
	clusters []internalaws.ECSCluster
	err      error
}

type servicesLoadedMsg struct {
	services []internalaws.ECSService
	err      error
}

type tasksLoadedMsg struct {
	tasks []internalaws.ECSTask
	err   error
}

type taskDefLoadedMsg struct {
	defs []internalaws.ECSContainerDef
	err  error
}

type logsLoadedMsg struct {
	logs []internalaws.LogEvent
	err  error
}

// --- Action menu ---

type Action struct {
	Key   string
	Label string
}

type actionMenu struct {
	title   string
	actions []Action
	cursor  int
}

func (a *actionMenu) MoveUp() {
	if a.cursor > 0 {
		a.cursor--
	}
}

func (a *actionMenu) MoveDown() {
	if a.cursor < len(a.actions)-1 {
		a.cursor++
	}
}

func (a *actionMenu) Selected() string {
	if a.cursor < len(a.actions) {
		return a.actions[a.cursor].Key
	}
	return ""
}

func (a *actionMenu) Render() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n", a.title))
	b.WriteString("  ─────────────────────────\n")
	for i, act := range a.actions {
		cursor := "  "
		if i == a.cursor {
			cursor = "▸ "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, act.Label))
	}
	b.WriteString("\n  Enter: select  Esc: cancel")
	return shared.RenderOverlay(b.String())
}

// --- Search ---

type searchModel struct {
	query  string
	active bool
}

func (s *searchModel) Insert(char rune) { s.query += string(char) }
func (s *searchModel) Backspace() {
	if len(s.query) > 0 {
		s.query = s.query[:len(s.query)-1]
	}
}
func (s *searchModel) Clear() { s.query = ""; s.active = false }
func (s *searchModel) Render(width int) string {
	if !s.active {
		return ""
	}
	prompt := shared.SearchPromptStyle.Render(" /")
	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s %s█", prompt, s.query))
}

// --- ECSModel ---

// ECSModel implements the shared.TabModel interface for the ECS tab.
type ECSModel struct {
	viewState viewState
	loading   bool
	err       error

	// Cluster level
	clusters      []internalaws.ECSCluster
	filtered      []internalaws.ECSCluster
	clusterCursor int
	search        searchModel

	// Service level
	services      []internalaws.ECSService
	serviceCursor int

	// Task level
	tasks     []internalaws.ECSTask
	taskCursor int

	// Container level
	containers      []internalaws.ECSContainer
	containerCursor int

	// Task definition (for detail/enrichment)
	taskDefs []internalaws.ECSContainerDef

	// Logs
	logs    []internalaws.LogEvent
	logsErr error

	// Action menu
	menu actionMenu

	// Selected references (breadcrumb)
	selectedCluster   *internalaws.ECSCluster
	selectedService   *internalaws.ECSService
	selectedTask      *internalaws.ECSTask
	selectedContainer *internalaws.ECSContainer

	// Detail loading
	detailLoading bool
}

func New() *ECSModel {
	return &ECSModel{viewState: vsClusterList, loading: true}
}

func (m *ECSModel) Init(s *shared.SharedState) tea.Cmd {
	m.loading = true
	m.err = nil
	m.viewState = vsClusterList
	return m.loadClusters(s)
}

func (m *ECSModel) Update(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case ECSExecDoneMsg:
		// Return to container list after exec finishes
		m.viewState = vsContainerList
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
			m.err = msg.err
			return m, nil
		}
		m.services = msg.services
		m.serviceCursor = 0
		return m, nil

	case tasksLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tasks = msg.tasks
		m.taskCursor = 0
		return m, nil

	case taskDefLoadedMsg:
		m.detailLoading = false
		if msg.err != nil {
			return m, nil
		}
		m.taskDefs = msg.defs
		// Enrich containers with log info from the task definition
		if m.selectedTask != nil {
			m.containers = internalaws.EnrichContainerLogs(
				m.selectedTask.Containers, msg.defs, m.selectedTask.ShortTaskID())
		}
		m.containerCursor = 0
		return m, nil

	case logsLoadedMsg:
		m.detailLoading = false
		m.logsErr = msg.err
		m.logs = msg.logs
		return m, nil
	}

	switch m.viewState {
	case vsClusterList:
		return m.updateClusterList(msg, s)
	case vsClusterSearch:
		return m.updateClusterSearch(msg, s)
	case vsClusterAction:
		return m.updateActionMenu(msg, s)
	case vsClusterDetail:
		return m.updateCloseOverlay(msg, s, vsClusterList)
	case vsServiceList:
		return m.updateServiceList(msg, s)
	case vsServiceAction:
		return m.updateActionMenu(msg, s)
	case vsServiceDetail:
		return m.updateCloseOverlay(msg, s, vsServiceList)
	case vsTaskList:
		return m.updateTaskList(msg, s)
	case vsTaskAction:
		return m.updateActionMenu(msg, s)
	case vsTaskDetail:
		return m.updateCloseOverlay(msg, s, vsTaskList)
	case vsTaskDefDetail:
		return m.updateCloseOverlay(msg, s, vsTaskList)
	case vsContainerList:
		return m.updateContainerList(msg, s)
	case vsContainerAction:
		return m.updateActionMenu(msg, s)
	case vsContainerDetail:
		return m.updateCloseOverlay(msg, s, vsContainerList)
	case vsLogs:
		return m.updateCloseOverlay(msg, s, vsContainerList)
	default:
		return m, nil
	}
}

func (m *ECSModel) View(s *shared.SharedState) string {
	var sections []string

	// Status bar with breadcrumb
	sections = append(sections, m.renderStatusBar(s))

	if m.search.active {
		sections = append(sections, m.search.Render(s.Width))
	}

	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading ECS Clusters..."))
	} else if m.err != nil && m.viewState == vsClusterList {
		sections = append(sections, lipgloss.NewStyle().Width(s.Width).Padding(1, 2).Render(
			shared.ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else {
		tableHeight := s.Height
		if m.search.active {
			tableHeight--
		}
		sections = append(sections, m.renderContent(s, tableHeight))
	}

	// Overlay
	overlay := m.renderOverlay(s)

	view := strings.Join(sections, "\n")
	if overlay != "" {
		view = shared.PlaceOverlay(s.Width, s.Height, overlay)
	}
	return view
}

func (m *ECSModel) ShortHelp() string {
	switch m.viewState {
	case vsClusterSearch:
		return helpLine("Esc", "Cancel")
	case vsClusterAction, vsServiceAction, vsTaskAction, vsContainerAction:
		return helpLine("↑↓", "Navigate", "Enter", "Select", "Esc", "Cancel")
	case vsClusterDetail, vsServiceDetail, vsTaskDetail, vsTaskDefDetail, vsContainerDetail:
		return helpLine("Esc", "Close")
	case vsLogs:
		return helpLine("Esc", "Close")
	case vsServiceList:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "Esc", "Back")
	case vsTaskList:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "Esc", "Back")
	case vsContainerList:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "Esc", "Back")
	default:
		return helpLine("↑↓", "Navigate", "Enter", "Actions", "/", "Search", "R", "Refresh")
	}
}

// --- Update handlers ---

func (m *ECSModel) updateClusterList(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "up", "k":
		if m.clusterCursor > 0 {
			m.clusterCursor--
		}
	case "down", "j":
		if m.clusterCursor < len(m.filtered)-1 {
			m.clusterCursor++
		}
	case "enter":
		if m.clusterCursor < len(m.filtered) {
			c := m.filtered[m.clusterCursor]
			m.selectedCluster = &c
			m.menu = actionMenu{
				title: fmt.Sprintf("%s (%s)", c.Name, c.Status),
				actions: []Action{
					{Key: "services", Label: "Services"},
					{Key: "tasks", Label: "Tasks (all in cluster)"},
					{Key: "detail", Label: "Cluster Details"},
				},
			}
			m.viewState = vsClusterAction
		}
	case "/":
		m.viewState = vsClusterSearch
		m.search.active = true
		m.search.query = ""
	case "R":
		m.loading = true
		m.err = nil
		return m, m.loadClusters(s)
	}
	return m, nil
}

func (m *ECSModel) updateClusterSearch(msg tea.Msg, _ *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.search.Clear()
		m.viewState = vsClusterList
		m.applyFilters()
	case "enter":
		m.viewState = vsClusterList
		m.search.active = false
	case "backspace":
		m.search.Backspace()
		m.applyFilters()
	default:
		r := keyMsg.String()
		if len(r) == 1 {
			m.search.Insert(rune(r[0]))
			m.applyFilters()
			m.clusterCursor = 0
		}
	}
	return m, nil
}

func (m *ECSModel) updateActionMenu(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	parentState := m.parentOfAction()

	switch keyMsg.String() {
	case "esc":
		m.viewState = parentState
		return m, nil
	case "up", "k":
		m.menu.MoveUp()
	case "down", "j":
		m.menu.MoveDown()
	case "enter":
		return m.executeAction(s)
	}
	return m, nil
}

func (m *ECSModel) updateServiceList(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.viewState = vsClusterList
		m.services = nil
		m.selectedService = nil
	case "up", "k":
		if m.serviceCursor > 0 {
			m.serviceCursor--
		}
	case "down", "j":
		if m.serviceCursor < len(m.services)-1 {
			m.serviceCursor++
		}
	case "enter":
		if m.serviceCursor < len(m.services) {
			svc := m.services[m.serviceCursor]
			m.selectedService = &svc
			m.menu = actionMenu{
				title: fmt.Sprintf("%s (%s)", svc.Name, svc.Status),
				actions: []Action{
					{Key: "svc_tasks", Label: "Tasks (in service)"},
					{Key: "svc_detail", Label: "Service Details"},
				},
			}
			m.viewState = vsServiceAction
		}
	}
	return m, nil
}

func (m *ECSModel) updateTaskList(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		// Go back: if we came from a service, go to service list; otherwise cluster list
		if m.selectedService != nil {
			m.viewState = vsServiceList
		} else {
			m.viewState = vsClusterList
		}
		m.tasks = nil
		m.selectedTask = nil
	case "up", "k":
		if m.taskCursor > 0 {
			m.taskCursor--
		}
	case "down", "j":
		if m.taskCursor < len(m.tasks)-1 {
			m.taskCursor++
		}
	case "enter":
		if m.taskCursor < len(m.tasks) {
			t := m.tasks[m.taskCursor]
			m.selectedTask = &t
			m.menu = actionMenu{
				title: fmt.Sprintf("Task %s (%s)", t.ShortTaskID(), t.LastStatus),
				actions: []Action{
					{Key: "containers", Label: "Containers"},
					{Key: "task_detail", Label: "Task Details"},
					{Key: "task_def", Label: "Task Definition"},
				},
			}
			m.viewState = vsTaskAction
		}
	}
	return m, nil
}

func (m *ECSModel) updateContainerList(msg tea.Msg, s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "esc":
		m.viewState = vsTaskList
		m.containers = nil
		m.selectedContainer = nil
	case "up", "k":
		if m.containerCursor > 0 {
			m.containerCursor--
		}
	case "down", "j":
		if m.containerCursor < len(m.containers)-1 {
			m.containerCursor++
		}
	case "enter":
		if m.containerCursor < len(m.containers) {
			c := m.containers[m.containerCursor]
			m.selectedContainer = &c
			actions := []Action{
				{Key: "logs", Label: "View Logs (last 50 lines)"},
				{Key: "exec", Label: "ECS Exec (interactive shell)"},
				{Key: "container_detail", Label: "Container Details"},
			}
			m.menu = actionMenu{
				title:   fmt.Sprintf("%s (%s)", c.Name, c.Status),
				actions: actions,
			}
			m.viewState = vsContainerAction
		}
	}
	return m, nil
}

func (m *ECSModel) updateCloseOverlay(msg tea.Msg, _ *shared.SharedState, backState viewState) (shared.TabModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "esc" {
			m.viewState = backState
		}
	}
	return m, nil
}

// parentOfAction returns the view state to return to when Esc is pressed on a menu.
func (m *ECSModel) parentOfAction() viewState {
	switch m.viewState {
	case vsClusterAction:
		return vsClusterList
	case vsServiceAction:
		return vsServiceList
	case vsTaskAction:
		return vsTaskList
	case vsContainerAction:
		return vsContainerList
	default:
		return vsClusterList
	}
}

// executeAction performs the selected action from the current menu.
func (m *ECSModel) executeAction(s *shared.SharedState) (shared.TabModel, tea.Cmd) {
	key := m.menu.Selected()

	switch key {
	// --- Cluster actions ---
	case "services":
		m.viewState = vsServiceList
		m.detailLoading = true
		m.services = nil
		return m, m.loadServices(s, m.selectedCluster.ARN)
	case "tasks":
		m.viewState = vsTaskList
		m.detailLoading = true
		m.tasks = nil
		m.selectedService = nil
		return m, m.loadTasks(s, m.selectedCluster.ARN, "")
	case "detail":
		m.viewState = vsClusterDetail
		return m, nil

	// --- Service actions ---
	case "svc_tasks":
		m.viewState = vsTaskList
		m.detailLoading = true
		m.tasks = nil
		return m, m.loadTasks(s, m.selectedCluster.ARN, m.selectedService.Name)
	case "svc_detail":
		m.viewState = vsServiceDetail
		return m, nil

	// --- Task actions ---
	case "containers":
		m.viewState = vsContainerList
		m.detailLoading = true
		m.containers = nil
		m.taskDefs = nil
		return m, m.loadTaskDef(s, m.selectedTask.TaskDefinitionARN)
	case "task_detail":
		m.viewState = vsTaskDetail
		return m, nil
	case "task_def":
		m.viewState = vsTaskDefDetail
		m.detailLoading = true
		m.taskDefs = nil
		return m, m.loadTaskDef(s, m.selectedTask.TaskDefinitionARN)

	// --- Container actions ---
	case "logs":
		m.viewState = vsLogs
		m.detailLoading = true
		m.logs = nil
		m.logsErr = nil
		c := m.selectedContainer
		return m, m.loadLogs(s, c.LogGroup, c.LogStream)
	case "exec":
		if m.selectedContainer != nil && m.selectedTask != nil && m.selectedCluster != nil {
			args := internalaws.BuildECSExecArgs(
				m.selectedCluster.ARN,
				m.selectedTask.TaskARN,
				m.selectedContainer.Name,
				s.Profile,
				s.Region,
			)
			return m, func() tea.Msg {
				return ECSExecRequest{
					ClusterARN:    m.selectedCluster.ARN,
					TaskARN:       m.selectedTask.TaskARN,
					ContainerName: m.selectedContainer.Name,
					Profile:       s.Profile,
					Region:        s.Region,
					Args:          args,
				}
			}
		}
		m.viewState = vsContainerList
		return m, nil
	case "container_detail":
		m.viewState = vsContainerDetail
		return m, nil
	}

	return m, nil
}

// --- Rendering ---

func (m *ECSModel) renderStatusBar(s *shared.SharedState) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + s.Profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + s.Region

	breadcrumb := m.breadcrumb()
	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, breadcrumb)
	return shared.StatusBarStyle.Width(s.Width).Render(content)
}

func (m *ECSModel) breadcrumb() string {
	parts := []string{fmt.Sprintf("[%d Clusters]", len(m.filtered))}

	if m.selectedCluster != nil && m.viewState > vsClusterList && m.viewState != vsClusterSearch {
		parts = append(parts, m.selectedCluster.Name)
	}
	if m.selectedService != nil && m.viewState >= vsTaskList {
		parts = append(parts, m.selectedService.Name)
	}
	if m.selectedTask != nil && m.viewState >= vsContainerList {
		parts = append(parts, "task:"+m.selectedTask.ShortTaskID())
	}

	return strings.Join(parts, " > ")
}

func (m *ECSModel) renderContent(s *shared.SharedState, tableHeight int) string {
	switch m.viewState {
	case vsClusterList, vsClusterSearch, vsClusterAction, vsClusterDetail:
		if len(m.filtered) == 0 {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No ECS clusters found in this region.")
		}
		return renderClusterTable(m.filtered, m.clusterCursor, s.Width, tableHeight)

	case vsServiceList, vsServiceAction, vsServiceDetail:
		if m.detailLoading {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading services...")
		}
		if len(m.services) == 0 {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No services found in this cluster. Press Esc to go back.")
		}
		return renderServiceTable(m.services, m.serviceCursor, s.Width, tableHeight)

	case vsTaskList, vsTaskAction, vsTaskDetail, vsTaskDefDetail:
		if m.detailLoading {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading tasks...")
		}
		if len(m.tasks) == 0 {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No tasks found. Press Esc to go back.")
		}
		return renderTaskTable(m.tasks, m.taskCursor, s.Width, tableHeight)

	case vsContainerList, vsContainerAction, vsContainerDetail, vsLogs:
		if m.detailLoading && len(m.containers) == 0 {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("Loading containers...")
		}
		if len(m.containers) == 0 {
			return lipgloss.NewStyle().Width(s.Width).Padding(2, 2).Render("No containers found. Press Esc to go back.")
		}
		return renderContainerTable(m.containers, m.containerCursor, s.Width, tableHeight)

	default:
		return ""
	}
}

func (m *ECSModel) renderOverlay(s *shared.SharedState) string {
	switch m.viewState {
	case vsClusterAction, vsServiceAction, vsTaskAction, vsContainerAction:
		return m.menu.Render()
	case vsClusterDetail:
		if m.selectedCluster != nil {
			return renderClusterDetail(*m.selectedCluster)
		}
	case vsServiceDetail:
		if m.selectedService != nil {
			return renderServiceDetail(*m.selectedService)
		}
	case vsTaskDetail:
		if m.selectedTask != nil {
			return renderTaskDetail(*m.selectedTask)
		}
	case vsTaskDefDetail:
		if m.detailLoading {
			return shared.RenderOverlay("  Loading task definition...")
		}
		if m.taskDefs != nil {
			return renderTaskDefDetail(m.taskDefs)
		}
	case vsContainerDetail:
		if m.selectedContainer != nil {
			return renderContainerDetail(*m.selectedContainer)
		}
	case vsLogs:
		if m.detailLoading {
			return shared.RenderOverlay("  Loading logs...")
		}
		return renderLogsOverlay(m.logs, m.logsErr, m.selectedContainer)
	}
	return ""
}

// --- Filters ---

func (m *ECSModel) applyFilters() {
	result := m.clusters
	if m.search.query != "" {
		q := strings.ToLower(m.search.query)
		var filtered []internalaws.ECSCluster
		for _, c := range result {
			if strings.Contains(internalaws.ECSSearchFields(c), q) {
				filtered = append(filtered, c)
			}
		}
		result = filtered
	}
	m.filtered = result
	if m.clusterCursor >= len(m.filtered) {
		m.clusterCursor = len(m.filtered) - 1
	}
	if m.clusterCursor < 0 {
		m.clusterCursor = 0
	}
}

// --- Loaders ---

func (m *ECSModel) loadClusters(s *shared.SharedState) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return clustersLoadedMsg{err: err}
		}
		clusters, err := internalaws.FetchECSClusters(ctx, clients.ECS)
		return clustersLoadedMsg{clusters: clusters, err: err}
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
		return servicesLoadedMsg{services: services, err: err}
	}
}

func (m *ECSModel) loadTasks(s *shared.SharedState, clusterARN, serviceName string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return tasksLoadedMsg{err: err}
		}
		tasks, err := internalaws.FetchECSTasks(ctx, clients.ECS, clusterARN, serviceName)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m *ECSModel) loadTaskDef(s *shared.SharedState, taskDefARN string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return taskDefLoadedMsg{err: err}
		}
		defs, err := internalaws.FetchTaskDefinition(ctx, clients.ECS, taskDefARN)
		return taskDefLoadedMsg{defs: defs, err: err}
	}
}

func (m *ECSModel) loadLogs(s *shared.SharedState, logGroup, logStream string) tea.Cmd {
	profile := s.Profile
	region := s.Region
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return logsLoadedMsg{err: err}
		}
		logs, err := internalaws.FetchContainerLogs(ctx, clients.CWL, logGroup, logStream, 50)
		return logsLoadedMsg{logs: logs, err: err}
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
