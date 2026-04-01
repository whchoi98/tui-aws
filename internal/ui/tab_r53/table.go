package tab_r53

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the R53 table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 30},
		{Key: "id", Title: "ID", Width: 20},
		{Key: "private", Title: "Pvt", Width: 3},
		{Key: "records", Title: "Recs", Width: 6},
		{Key: "comment", Title: "Comment", Width: 30},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 30},
		{Key: "id", Title: "ID", Width: 20},
		{Key: "private", Title: "Pvt", Width: 3},
		{Key: "records", Title: "Recs", Width: 6},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

// RenderTable renders the R53 table with header, rows, and scrolling.
func RenderTable(zones []aws.HostedZone, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(zones) && i < offset+maxRows; i++ {
		zone := zones[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, zone)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, zone)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(zones)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, zone aws.HostedZone) string {
	switch key {
	case "name":
		return zone.Name
	case "id":
		return zone.ID
	case "private":
		if zone.IsPrivate {
			return "Yes"
		}
		return "No"
	case "records":
		return fmt.Sprintf("%d", zone.RecordCount)
	case "comment":
		if zone.Comment != "" {
			return zone.Comment
		}
		return "-"
	default:
		return ""
	}
}

func cellStyle(key string, zone aws.HostedZone) lipgloss.Style {
	switch key {
	case "private":
		if zone.IsPrivate {
			return shared.StatePending
		}
		return lipgloss.Style{}
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Hosted Zones]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
