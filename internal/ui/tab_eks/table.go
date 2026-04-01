package tab_eks

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
		{Key: "version", Title: "Version", Width: 8},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "endpoint", Title: "Endpoint", Width: 40},
		{Key: "vpc", Title: "VPC", Width: 15},
	}
}

func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "version", Title: "Version", Width: 8},
		{Key: "status", Title: "Status", Width: 10},
	}
}

func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

func RenderTable(clusters []aws.EKSCluster, columns []shared.Column, cursor, width, height int) string {
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

func cellValue(key string, c aws.EKSCluster) string {
	switch key {
	case "name":
		return c.Name
	case "version":
		return c.Version
	case "status":
		return c.Status
	case "endpoint":
		return c.Endpoint
	case "vpc":
		return c.VpcID
	default:
		return ""
	}
}

func cellStyle(key string, c aws.EKSCluster) lipgloss.Style {
	if key == "status" {
		return eksStatusStyle(c.Status)
	}
	return lipgloss.Style{}
}

func eksStatusStyle(status string) lipgloss.Style {
	switch status {
	case "ACTIVE":
		return shared.StateRunning
	case "CREATING", "UPDATING":
		return shared.StatePending
	case "DELETING", "FAILED":
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
