package tab_r53

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// RenderZoneDetail renders the Route 53 hosted zone detail overlay with records.
func RenderZoneDetail(zone internalaws.HostedZone, loading bool, scrollOffset, screenHeight int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (%s)\n", zone.Name, zone.ID))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:          %s\n", zone.Name))
	b.WriteString(fmt.Sprintf("  ID:            %s\n", zone.ID))
	b.WriteString(fmt.Sprintf("  Private:       %s\n", boolLabel(zone.IsPrivate)))
	b.WriteString(fmt.Sprintf("  Record Count:  %d\n", zone.RecordCount))
	if zone.Comment != "" {
		b.WriteString(fmt.Sprintf("  Comment:       %s\n", zone.Comment))
	}

	if loading {
		b.WriteString("\n  Loading records...\n")
		b.WriteString("\n  Esc: close")
		return shared.RenderOverlay(b.String())
	}

	if len(zone.Records) > 0 {
		b.WriteString(fmt.Sprintf("\n  Records (%d):\n", len(zone.Records)))
		b.WriteString(fmt.Sprintf("  %-30s %-6s %-6s %s\n", "Name", "Type", "TTL", "Value/Alias"))
		b.WriteString("  " + strings.Repeat("─", 75) + "\n")

		// Show a scrollable window of records
		maxVisible := screenHeight - 16 // account for header + footer
		if maxVisible < 5 {
			maxVisible = 5
		}

		endIdx := scrollOffset + maxVisible
		if endIdx > len(zone.Records) {
			endIdx = len(zone.Records)
		}

		for i := scrollOffset; i < endIdx; i++ {
			rec := zone.Records[i]
			ttlStr := "-"
			if rec.TTL > 0 {
				ttlStr = fmt.Sprintf("%d", rec.TTL)
			}
			value := rec.Value
			if len(value) > 50 {
				value = value[:47] + "..."
			}
			b.WriteString(fmt.Sprintf("  %-30s %-6s %-6s %s\n",
				truncate(rec.Name, 30), rec.Type, ttlStr, value))
		}

		if endIdx < len(zone.Records) {
			b.WriteString(fmt.Sprintf("  ... (%d more, scroll with j/k)\n", len(zone.Records)-endIdx))
		}
	} else {
		b.WriteString("\n  No records found.\n")
	}

	b.WriteString("\n  ↑↓: Scroll  Esc: Close")
	return shared.RenderOverlay(b.String())
}

func boolLabel(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
