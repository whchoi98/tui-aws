package tab_cloudfront

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// DefaultColumns returns the CloudFront table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "id", Title: "ID", Width: 15},
		{Key: "domain", Title: "Domain", Width: 40},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "enabled", Title: "On", Width: 3},
		{Key: "origins", Title: "Origins", Width: 30},
		{Key: "aliases", Title: "Aliases", Width: 20},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "id", Title: "ID", Width: 15},
		{Key: "domain", Title: "Domain", Width: 40},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "enabled", Title: "On", Width: 3},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 120 {
		return CompactColumns()
	}
	return DefaultColumns()
}

// RenderTable renders the CloudFront table with header, rows, and scrolling.
func RenderTable(distributions []aws.Distribution, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(distributions) && i < offset+maxRows; i++ {
		d := distributions[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, d)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, d)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(distributions)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, d aws.Distribution) string {
	switch key {
	case "id":
		return d.ID
	case "domain":
		return d.DomainName
	case "status":
		return d.Status
	case "enabled":
		if d.Enabled {
			return "Yes"
		}
		return "No"
	case "origins":
		return strings.Join(d.Origins, ", ")
	case "aliases":
		return strings.Join(d.Aliases, ", ")
	default:
		return ""
	}
}

func cellStyle(key string, d aws.Distribution) lipgloss.Style {
	switch key {
	case "status":
		return statusStyle(d.Status)
	case "enabled":
		if d.Enabled {
			return shared.StateRunning
		}
		return shared.StateStopped
	default:
		return lipgloss.Style{}
	}
}

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "Deployed":
		return shared.StateRunning
	case "InProgress":
		return shared.StatePending
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Distributions]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
