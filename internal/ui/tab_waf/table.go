package tab_waf

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the WAF table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "id", Title: "ID", Width: 23},
		{Key: "scope", Title: "Scope", Width: 10},
		{Key: "rules", Title: "Rules", Width: 5},
		{Key: "default", Title: "Action", Width: 6},
		{Key: "resources", Title: "Assoc", Width: 5},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 25},
		{Key: "scope", Title: "Scope", Width: 10},
		{Key: "rules", Title: "Rules", Width: 5},
		{Key: "default", Title: "Action", Width: 6},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

// RenderTable renders the WAF table with header, rows, and scrolling.
func RenderTable(acls []aws.WebACL, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(acls) && i < offset+maxRows; i++ {
		acl := acls[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, acl)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, acl)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(acls)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, acl aws.WebACL) string {
	switch key {
	case "name":
		return acl.Name
	case "id":
		return acl.ID
	case "scope":
		return acl.Scope
	case "rules":
		return fmt.Sprintf("%d", acl.Rules)
	case "default":
		return acl.DefaultAction
	case "resources":
		return fmt.Sprintf("%d", len(acl.AssociatedResources))
	default:
		return ""
	}
}

func cellStyle(key string, acl aws.WebACL) lipgloss.Style {
	switch key {
	case "default":
		if acl.DefaultAction == "Allow" {
			return shared.StateRunning
		}
		if acl.DefaultAction == "Block" {
			return shared.StateStopped
		}
		return lipgloss.Style{}
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Web ACLs]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
