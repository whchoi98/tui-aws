package tab_ecs

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "tasks", Title: "Tasks", Width: 8},
		{Key: "services", Title: "Svcs", Width: 5},
		{Key: "instances", Title: "Inst", Width: 5},
		{Key: "capproviders", Title: "Cap Providers", Width: 20},
	}
}

func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "tasks", Title: "Tasks", Width: 8},
		{Key: "services", Title: "Svcs", Width: 5},
	}
}

func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

func RenderTable(clusters []aws.ECSCluster, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(clusters) && i < offset+maxRows; i++ {
		c := clusters[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, c)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, c)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(clusters)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func cellValue(key string, c aws.ECSCluster) string {
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

func cellStyle(key string, c aws.ECSCluster) lipgloss.Style {
	if key == "status" {
		return ecsStatusStyle(c.Status)
	}
	return lipgloss.Style{}
}

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

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Clusters]", count)
	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
