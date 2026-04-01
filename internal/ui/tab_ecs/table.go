package tab_ecs

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// --- Cluster table ---

func clusterColumns(width int) []shared.Column {
	cols := []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "tasks", Title: "Tasks", Width: 8},
		{Key: "services", Title: "Svcs", Width: 5},
		{Key: "instances", Title: "Inst", Width: 5},
		{Key: "capproviders", Title: "Cap Providers", Width: 20},
	}
	if width < 100 {
		cols = []shared.Column{
			{Key: "name", Title: "Name", Width: 25},
			{Key: "status", Title: "Status", Width: 10},
			{Key: "tasks", Title: "Tasks", Width: 8},
			{Key: "services", Title: "Svcs", Width: 5},
		}
	}
	return shared.ExpandNameColumn(cols, width)
}

func renderClusterTable(clusters []aws.ECSCluster, cursor, width, height int) string {
	columns := clusterColumns(width)
	return renderGenericTable(len(clusters), cursor, columns, width, height,
		func(i int, col shared.Column) string { return clusterCellValue(col.Key, clusters[i]) },
		func(i int, col shared.Column) lipgloss.Style { return clusterCellStyle(col.Key, clusters[i]) },
	)
}

func clusterCellValue(key string, c aws.ECSCluster) string {
	switch key {
	case "name":
		return c.Name
	case "status":
		return c.Status
	case "tasks":
		return fmt.Sprintf("%d/%d", c.RunningTasks, c.RunningTasks+c.PendingTasks)
	case "services":
		return fmt.Sprintf("%d", c.Services)
	case "instances":
		return fmt.Sprintf("%d", c.Instances)
	case "capproviders":
		if len(c.CapacityProviders) == 0 {
			return "-"
		}
		return strings.Join(c.CapacityProviders, ",")
	default:
		return ""
	}
}

func clusterCellStyle(key string, c aws.ECSCluster) lipgloss.Style {
	if key == "status" {
		return ecsStatusStyle(c.Status)
	}
	return lipgloss.Style{}
}

// --- Service table ---

func serviceColumns(width int) []shared.Column {
	cols := []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "running", Title: "Running", Width: 8},
		{Key: "desired", Title: "Desired", Width: 8},
		{Key: "pending", Title: "Pending", Width: 8},
		{Key: "launch", Title: "Launch", Width: 10},
		{Key: "taskdef", Title: "Task Def", Width: 25},
	}
	if width < 110 {
		cols = []shared.Column{
			{Key: "name", Title: "Name", Width: 25},
			{Key: "status", Title: "Status", Width: 10},
			{Key: "running", Title: "Running", Width: 8},
			{Key: "desired", Title: "Desired", Width: 8},
			{Key: "launch", Title: "Launch", Width: 10},
		}
	}
	return shared.ExpandNameColumn(cols, width)
}

func renderServiceTable(services []aws.ECSService, cursor, width, height int) string {
	columns := serviceColumns(width)
	return renderGenericTable(len(services), cursor, columns, width, height,
		func(i int, col shared.Column) string { return serviceCellValue(col.Key, services[i]) },
		func(i int, col shared.Column) lipgloss.Style { return serviceCellStyle(col.Key, services[i]) },
	)
}

func serviceCellValue(key string, s aws.ECSService) string {
	switch key {
	case "name":
		return s.Name
	case "status":
		return s.Status
	case "running":
		return fmt.Sprintf("%d", s.RunningCount)
	case "desired":
		return fmt.Sprintf("%d", s.DesiredCount)
	case "pending":
		return fmt.Sprintf("%d", s.PendingCount)
	case "launch":
		if s.LaunchType == "" {
			return "-"
		}
		return s.LaunchType
	case "taskdef":
		// Show just the family:revision
		parts := strings.Split(s.TaskDefinition, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return s.TaskDefinition
	default:
		return ""
	}
}

func serviceCellStyle(key string, s aws.ECSService) lipgloss.Style {
	if key == "status" {
		return ecsStatusStyle(s.Status)
	}
	return lipgloss.Style{}
}

// --- Task table ---

func taskColumns(width int) []shared.Column {
	cols := []shared.Column{
		{Key: "id", Title: "Task ID", Width: 14},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "launch", Title: "Launch", Width: 8},
		{Key: "cpu", Title: "CPU", Width: 5},
		{Key: "mem", Title: "Mem", Width: 5},
		{Key: "started", Title: "Started", Width: 20},
		{Key: "health", Title: "Health", Width: 10},
		{Key: "group", Title: "Group", Width: 20},
	}
	if width < 120 {
		cols = []shared.Column{
			{Key: "id", Title: "Task ID", Width: 14},
			{Key: "status", Title: "Status", Width: 10},
			{Key: "launch", Title: "Launch", Width: 8},
			{Key: "cpu", Title: "CPU", Width: 5},
			{Key: "mem", Title: "Mem", Width: 5},
			{Key: "health", Title: "Health", Width: 10},
		}
	}
	return cols
}

func renderTaskTable(tasks []aws.ECSTask, cursor, width, height int) string {
	columns := taskColumns(width)
	return renderGenericTable(len(tasks), cursor, columns, width, height,
		func(i int, col shared.Column) string { return taskCellValue(col.Key, tasks[i]) },
		func(i int, col shared.Column) lipgloss.Style { return taskCellStyle(col.Key, tasks[i]) },
	)
}

func taskCellValue(key string, t aws.ECSTask) string {
	switch key {
	case "id":
		return t.ShortTaskID()
	case "status":
		return t.LastStatus
	case "launch":
		if t.LaunchType == "" {
			return "-"
		}
		return t.LaunchType
	case "cpu":
		if t.CPU == "" {
			return "-"
		}
		return t.CPU
	case "mem":
		if t.Memory == "" {
			return "-"
		}
		return t.Memory
	case "started":
		if t.StartedAt == "" {
			return "-"
		}
		return t.StartedAt
	case "health":
		if t.HealthStatus == "" || t.HealthStatus == "UNKNOWN" {
			return "-"
		}
		return t.HealthStatus
	case "group":
		return t.Group
	default:
		return ""
	}
}

func taskCellStyle(key string, t aws.ECSTask) lipgloss.Style {
	switch key {
	case "status":
		return taskStatusStyle(t.LastStatus)
	case "health":
		return healthStatusStyle(t.HealthStatus)
	}
	return lipgloss.Style{}
}

// --- Container table ---

func containerColumns(width int) []shared.Column {
	cols := []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "image", Title: "Image", Width: 35},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "health", Title: "Health", Width: 10},
		{Key: "ports", Title: "Ports", Width: 25},
	}
	if width < 110 {
		cols = []shared.Column{
			{Key: "name", Title: "Name", Width: 20},
			{Key: "status", Title: "Status", Width: 10},
			{Key: "health", Title: "Health", Width: 10},
			{Key: "image", Title: "Image", Width: 30},
		}
	}
	return cols
}

func renderContainerTable(containers []aws.ECSContainer, cursor, width, height int) string {
	columns := containerColumns(width)
	return renderGenericTable(len(containers), cursor, columns, width, height,
		func(i int, col shared.Column) string { return containerCellValue(col.Key, containers[i]) },
		func(i int, col shared.Column) lipgloss.Style { return containerCellStyle(col.Key, containers[i]) },
	)
}

func containerCellValue(key string, c aws.ECSContainer) string {
	switch key {
	case "name":
		return c.Name
	case "image":
		return c.Image
	case "status":
		return c.Status
	case "health":
		if c.HealthStatus == "" || c.HealthStatus == "UNKNOWN" {
			return "-"
		}
		return c.HealthStatus
	case "ports":
		if len(c.Ports) == 0 {
			return "-"
		}
		return strings.Join(c.Ports, ", ")
	default:
		return ""
	}
}

func containerCellStyle(key string, c aws.ECSContainer) lipgloss.Style {
	switch key {
	case "status":
		return taskStatusStyle(c.Status)
	case "health":
		return healthStatusStyle(c.HealthStatus)
	}
	return lipgloss.Style{}
}

// --- Generic table renderer ---

func renderGenericTable(
	count, cursor int,
	columns []shared.Column,
	width, height int,
	getText func(rowIdx int, col shared.Column) string,
	getStyle func(rowIdx int, col shared.Column) lipgloss.Style,
) string {
	var b strings.Builder

	header := shared.RenderRow(columns, func(col shared.Column) string {
		return col.Title
	}, nil)
	b.WriteString(shared.TableHeaderStyle.Width(width).Render(header))
	b.WriteString("\n")

	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	offset := 0
	if cursor >= maxRows {
		offset = cursor - maxRows + 1
	}

	for i := offset; i < count && i < offset+maxRows; i++ {
		idx := i
		row := shared.RenderRow(columns,
			func(col shared.Column) string { return getText(idx, col) },
			func(col shared.Column) lipgloss.Style { return getStyle(idx, col) },
		)
		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < count-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// --- Status styles ---

func ecsStatusStyle(status string) lipgloss.Style {
	switch status {
	case "ACTIVE":
		return shared.StateRunning
	case "PROVISIONING":
		return shared.StatePending
	case "DEPROVISIONING", "FAILED", "INACTIVE":
		return shared.StateStopped
	default:
		return lipgloss.Style{}
	}
}

func taskStatusStyle(status string) lipgloss.Style {
	switch status {
	case "RUNNING":
		return shared.StateRunning
	case "PENDING", "PROVISIONING", "ACTIVATING":
		return shared.StatePending
	case "STOPPED", "DEPROVISIONING":
		return shared.StateStopped
	case "STOPPING", "DEACTIVATING":
		return shared.StateStopping
	default:
		return lipgloss.Style{}
	}
}

func healthStatusStyle(status string) lipgloss.Style {
	switch status {
	case "HEALTHY":
		return shared.StateRunning
	case "UNHEALTHY":
		return shared.StateStopped
	default:
		return lipgloss.Style{}
	}
}
