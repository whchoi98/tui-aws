package tab_sg

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// --- SG Columns ---

// SGDefaultColumns returns the SG table columns.
func SGDefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "sg_id", Title: "SG ID", Width: 23},
		{Key: "vpc", Title: "VPC", Width: 15},
		{Key: "inbound", Title: "In", Width: 4},
		{Key: "outbound", Title: "Out", Width: 4},
		{Key: "description", Title: "Description", Width: 30},
	}
}

// SGCompactColumns returns a minimal column set for narrow terminals.
func SGCompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "sg_id", Title: "SG ID", Width: 23},
		{Key: "vpc", Title: "VPC", Width: 15},
	}
}

// SGColumnsForWidth returns the appropriate SG column set for the given terminal width.
func SGColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(SGCompactColumns(), width)
	}
	return shared.ExpandNameColumn(SGDefaultColumns(), width)
}

// RenderSGTable renders the security group table.
func RenderSGTable(sgs []aws.SecurityGroup, columns []shared.Column, cursor, width, height int) string {
	var b strings.Builder

	// Header
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

	for i := offset; i < len(sgs) && i < offset+maxRows; i++ {
		sg := sgs[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return sgCellValue(col.Key, sg)
		}, func(col shared.Column) lipgloss.Style {
			_ = col
			return lipgloss.Style{}
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(sgs)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func sgCellValue(key string, sg aws.SecurityGroup) string {
	switch key {
	case "name":
		if sg.Name != "" {
			return sg.Name
		}
		return "-"
	case "sg_id":
		return sg.ID
	case "vpc":
		return sg.VpcID
	case "inbound":
		return fmt.Sprintf("%d", len(sg.InboundRules))
	case "outbound":
		return fmt.Sprintf("%d", len(sg.OutboundRules))
	case "description":
		return sg.Description
	default:
		return ""
	}
}

// --- NACL Columns ---

// NACLDefaultColumns returns the NACL table columns.
func NACLDefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "acl_id", Title: "ACL ID", Width: 23},
		{Key: "vpc", Title: "VPC", Width: 15},
		{Key: "default", Title: "Def", Width: 3},
		{Key: "subnets", Title: "Subnets", Width: 5},
	}
}

// NACLCompactColumns returns a minimal column set for narrow terminals.
func NACLCompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 20},
		{Key: "acl_id", Title: "ACL ID", Width: 23},
		{Key: "vpc", Title: "VPC", Width: 15},
	}
}

// NACLColumnsForWidth returns the appropriate NACL column set for the given terminal width.
func NACLColumnsForWidth(width int) []shared.Column {
	if width < 80 {
		return shared.ExpandNameColumn(NACLCompactColumns(), width)
	}
	return shared.ExpandNameColumn(NACLDefaultColumns(), width)
}

// RenderNACLTable renders the network ACL table.
func RenderNACLTable(nacls []aws.NetworkACL, columns []shared.Column, cursor, width, height int) string {
	var b strings.Builder

	// Header
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

	for i := offset; i < len(nacls) && i < offset+maxRows; i++ {
		nacl := nacls[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return naclCellValue(col.Key, nacl)
		}, func(col shared.Column) lipgloss.Style {
			_ = col
			return lipgloss.Style{}
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(nacls)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func naclCellValue(key string, nacl aws.NetworkACL) string {
	switch key {
	case "name":
		if nacl.Name != "" {
			return nacl.Name
		}
		return "-"
	case "acl_id":
		return nacl.ID
	case "vpc":
		return nacl.VpcID
	case "default":
		if nacl.IsDefault {
			return "✓"
		}
		return "-"
	case "subnets":
		return fmt.Sprintf("%d", len(nacl.Subnets))
	default:
		return ""
	}
}

func renderStatusBar(profile, region, mode string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region

	label := "Security Groups"
	if mode == "nacl" {
		label = "Network ACLs"
	}
	countPart := fmt.Sprintf("[%d %s]", count, label)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
