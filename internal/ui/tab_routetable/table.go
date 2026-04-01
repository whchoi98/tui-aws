package tab_routetable

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the Route Table table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "rt_id", Title: "RT ID", Width: 24},
		{Key: "vpc", Title: "VPC", Width: 15},
		{Key: "main", Title: "Main", Width: 3},
		{Key: "subnets", Title: "Subnets", Width: 5},
		{Key: "routes", Title: "Routes", Width: 5},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "rt_id", Title: "RT ID", Width: 24},
		{Key: "vpc", Title: "VPC", Width: 15},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 80 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

// RenderTable renders the route table table with header, rows, and scrolling.
func RenderTable(rts []aws.RouteTable, columns []shared.Column, cursor, width, height int) string {
	var b strings.Builder

	// Header
	header := shared.RenderRow(columns, func(col shared.Column) string {
		return col.Title
	}, nil)
	b.WriteString(shared.TableHeaderStyle.Width(width).Render(header))
	b.WriteString("\n")

	// Available rows: total height minus statusbar(1) + helpbar(1) + header(1) + tabbar(1)
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	// Calculate scroll offset
	offset := 0
	if cursor >= maxRows {
		offset = cursor - maxRows + 1
	}

	for i := offset; i < len(rts) && i < offset+maxRows; i++ {
		rt := rts[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, rt)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, rt)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(rts)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, rt aws.RouteTable) string {
	switch key {
	case "name":
		if rt.Name != "" {
			return rt.Name
		}
		return "-"
	case "rt_id":
		return rt.ID
	case "vpc":
		return rt.VpcID
	case "main":
		if rt.IsMain {
			return "✓"
		}
		return "-"
	case "subnets":
		return fmt.Sprintf("%d", len(rt.Subnets))
	case "routes":
		return fmt.Sprintf("%d", len(rt.Routes))
	default:
		return ""
	}
}

func cellStyle(key string, _ aws.RouteTable) lipgloss.Style {
	_ = key
	return lipgloss.Style{}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Route Tables]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
