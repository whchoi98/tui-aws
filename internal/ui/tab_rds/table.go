package tab_rds

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "id", Title: "ID", Width: 20},
		{Key: "engine", Title: "Engine", Width: 12},
		{Key: "class", Title: "Class", Width: 15},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "endpoint", Title: "Endpoint", Width: 35},
		{Key: "multiaz", Title: "MAZ", Width: 3},
		{Key: "storage", Title: "Storage", Width: 8},
	}
}

func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "id", Title: "ID", Width: 20},
		{Key: "engine", Title: "Engine", Width: 12},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "endpoint", Title: "Endpoint", Width: 35},
	}
}

func ColumnsForWidth(width int) []shared.Column {
	if width < 110 {
		return CompactColumns()
	}
	return DefaultColumns()
}

func RenderTable(instances []aws.DBInstance, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(instances) && i < offset+maxRows; i++ {
		inst := instances[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, inst)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, inst)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(instances)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func cellValue(key string, inst aws.DBInstance) string {
	switch key {
	case "id":
		return inst.ID
	case "engine":
		return inst.Engine
	case "class":
		return inst.Class
	case "status":
		return inst.Status
	case "endpoint":
		return inst.Endpoint
	case "multiaz":
		if inst.MultiAZ {
			return "Yes"
		}
		return "No"
	case "storage":
		return fmt.Sprintf("%dGiB", inst.AllocatedStorage)
	default:
		return ""
	}
}

func cellStyle(key string, inst aws.DBInstance) lipgloss.Style {
	if key == "status" {
		return rdsStatusStyle(inst.Status)
	}
	return lipgloss.Style{}
}

func rdsStatusStyle(status string) lipgloss.Style {
	switch status {
	case "available":
		return shared.StateRunning
	case "creating", "modifying", "backing-up", "configuring-enhanced-monitoring":
		return shared.StatePending
	case "deleting", "failed", "incompatible-parameters", "storage-full":
		return shared.StateStopped
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Instances]", count)
	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
