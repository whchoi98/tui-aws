package tab_subnet

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the Subnet table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 22},
		{Key: "subnet_id", Title: "Subnet ID", Width: 26},
		{Key: "vpc", Title: "VPC", Width: 15},
		{Key: "cidr", Title: "CIDR", Width: 18},
		{Key: "az", Title: "AZ", Width: 5},
		{Key: "available", Title: "Avail", Width: 5},
		{Key: "public", Title: "Pub", Width: 3},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 22},
		{Key: "subnet_id", Title: "Subnet ID", Width: 26},
		{Key: "cidr", Title: "CIDR", Width: 18},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return CompactColumns()
	}
	return DefaultColumns()
}

// RenderTable renders the Subnet table with header, rows, and scrolling.
func RenderTable(subnets []aws.Subnet, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(subnets) && i < offset+maxRows; i++ {
		sub := subnets[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, sub)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, sub)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(subnets)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, sub aws.Subnet) string {
	switch key {
	case "name":
		if sub.Name != "" {
			return sub.Name
		}
		return "-"
	case "subnet_id":
		return sub.SubnetID
	case "vpc":
		return sub.VpcID
	case "cidr":
		return sub.CIDRBlock
	case "az":
		// Show short AZ (last 2 chars)
		if len(sub.AZ) > 2 {
			return sub.AZ[len(sub.AZ)-2:]
		}
		return sub.AZ
	case "available":
		return fmt.Sprintf("%d", sub.AvailableIPs)
	case "public":
		if sub.MapPublicIP {
			return "Yes"
		}
		return ""
	default:
		return ""
	}
}

func cellStyle(key string, sub aws.Subnet) lipgloss.Style {
	switch key {
	case "public":
		if sub.MapPublicIP {
			return shared.StatePending // yellow for public subnets
		}
	}
	return lipgloss.Style{}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Subnets]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
