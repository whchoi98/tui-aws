package tab_acm

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// Status color styles
var (
	statusIssued  = lipgloss.NewStyle().Foreground(lipgloss.Color("#b8bb26")) // green
	statusPending = lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f")) // yellow
	statusFailed  = lipgloss.NewStyle().Foreground(lipgloss.Color("#fb4934")) // red
)

// DefaultColumns returns the ACM table columns.
func DefaultColumns() []shared.Column {
	return []shared.Column{
		{Key: "domain", Title: "Domain", Width: 30},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "type", Title: "Type", Width: 10},
		{Key: "expires", Title: "Expires", Width: 18},
		{Key: "inuse", Title: "InUse", Width: 5},
		{Key: "sans", Title: "SANs", Width: 3},
	}
}

// CompactColumns returns a minimal column set for narrow terminals.
func CompactColumns() []shared.Column {
	return []shared.Column{
		{Key: "domain", Title: "Domain", Width: 30},
		{Key: "status", Title: "Status", Width: 10},
		{Key: "type", Title: "Type", Width: 10},
		{Key: "expires", Title: "Expires", Width: 18},
	}
}

// ColumnsForWidth returns the appropriate column set for the given terminal width.
func ColumnsForWidth(width int) []shared.Column {
	if width < 100 {
		return CompactColumns()
	}
	return DefaultColumns()
}

// RenderTable renders the ACM table with header, rows, and scrolling.
func RenderTable(certs []aws.Certificate, columns []shared.Column, cursor, width, height int) string {
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

	for i := offset; i < len(certs) && i < offset+maxRows; i++ {
		cert := certs[i]
		row := shared.RenderRow(columns, func(col shared.Column) string {
			return cellValue(col.Key, cert)
		}, func(col shared.Column) lipgloss.Style {
			return cellStyle(col.Key, cert)
		})

		if i == cursor {
			row = shared.TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(certs)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func cellValue(key string, cert aws.Certificate) string {
	switch key {
	case "domain":
		return cert.DomainName
	case "status":
		return cert.Status
	case "type":
		return cert.Type
	case "expires":
		if cert.NotAfter != "" {
			return cert.NotAfter
		}
		return "-"
	case "inuse":
		return fmt.Sprintf("%d", len(cert.InUseBy))
	case "sans":
		return fmt.Sprintf("%d", len(cert.SubjectAlternativeNames))
	default:
		return ""
	}
}

func cellStyle(key string, cert aws.Certificate) lipgloss.Style {
	switch key {
	case "status":
		return certStatusStyle(cert.Status)
	default:
		return lipgloss.Style{}
	}
}

func certStatusStyle(status string) lipgloss.Style {
	switch status {
	case "ISSUED":
		return statusIssued
	case "PENDING_VALIDATION":
		return statusPending
	case "EXPIRED", "REVOKED", "FAILED":
		return statusFailed
	default:
		return lipgloss.Style{}
	}
}

func renderStatusBar(profile, region string, count, width int) string {
	profilePart := shared.StatusKeyStyle.Render("Profile: ") + profile
	regionPart := shared.StatusKeyStyle.Render("Region: ") + region
	countPart := fmt.Sprintf("[%d Certificates]", count)

	content := fmt.Sprintf(" %s  |  %s  |  %s", profilePart, regionPart, countPart)
	return shared.StatusBarStyle.Width(width).Render(content)
}
