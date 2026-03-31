package tab_vpc

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the VPC table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "vpc_id", Title: "VPC ID", Width: 23},
		{Key: "cidr", Title: "CIDR", Width: 18},
		{Key: "state", Title: "State", Width: 10},
		{Key: "default", Title: "Def", Width: 3},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "vpc_id", Title: "VPC ID", Width: 23},
		{Key: "cidr", Title: "CIDR", Width: 18},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 80 {
		return CompactColumns()
	}
	return DefaultColumns()
}

// RenderTable renders the VPC table with header, rows, and scrolling.
func RenderTable(vpcs []aws.VPC, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(vpcs) && i < offset+maxRows; i++ {
		vpc := vpcs[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, vpc)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, vpc)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(vpcs)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, vpc aws.VPC) string {
	switch key {
	case "name":
		if vpc.Name != "" {
			return vpc.Name
		}
		return "-"
	case "vpc_id":
		return vpc.VpcID
	case "cidr":
		return vpc.CIDRBlock
	case "state":
		return vpc.State
	case "default":
		if vpc.IsDefault {
			return "Yes"
		}
		return ""
	default:
		return ""
	}
}

func cellStyle(key string, vpc aws.VPC) lipgloss.Style {
	switch key {
	case "state":
		return vpcStateStyle(vpc.State)
	default:
		return lipgloss.Style{}
	}
}

func vpcStateStyle(state string) lipgloss.Style {
	switch state {
	case "available":
		return shared.StateRunning
	case "pending":
		return shared.StatePending
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d VPCs]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
