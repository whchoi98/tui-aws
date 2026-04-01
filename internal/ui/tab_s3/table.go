package tab_s3

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 35},
		{Key: "region", Title: "Region", Width: 15},
		{Key: "created", Title: "Created", Width: 18},
		{Key: "versioning", Title: "Versioning", Width: 10},
		{Key: "encryption", Title: "Encryption", Width: 10},
		{Key: "public", Title: "Public", Width: 7},
	}
}

func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "name", Title: "Name", Width: 35},
		{Key: "region", Title: "Region", Width: 15},
		{Key: "created", Title: "Created", Width: 18},
	}
}

func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return shared.ExpandNameColumn(CompactColumns(), width)
	}
	return shared.ExpandNameColumn(DefaultColumns(), width)
}

func RenderTable(buckets []aws.Bucket, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(buckets) && i < offset+maxRows; i++ {
		bkt := buckets[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, bkt)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, bkt)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(buckets)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func cellValue(key string, bkt aws.Bucket) string {
	switch key {
	case "name":
		return bkt.Name
	case "region":
		return bkt.Region
	case "created":
		return bkt.CreationDate
	case "versioning":
		if bkt.Versioning == "" {
			return "-"
		}
		return bkt.Versioning
	case "encryption":
		if bkt.Encryption == "" {
			return "-"
		}
		return bkt.Encryption
	case "public":
		if bkt.PublicAccess == "" {
			return "-"
		}
		return bkt.PublicAccess
	default:
		return ""
	}
}

func cellStyle(key string, bkt aws.Bucket) lipgloss.Style {
	if key == "public" && bkt.PublicAccess == "Open" {
		return shared.StateStopped // red for open
	}
	return lipgloss.Style{}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Buckets]", count)
	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
