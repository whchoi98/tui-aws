package tab_troubleshoot

import (
	"fmt"
	"strings"
)

// RenderResult renders the connectivity check result as a formatted string.
func RenderResult(result CheckResult, srcName, dstName, protocol, port string) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("  Connectivity: %s -> %s  %s/%s\n",
		srcName, dstName, strings.ToUpper(protocol), port))
	b.WriteString("  " + strings.Repeat("=", 50) + "\n\n")

	// Steps
	for _, step := range result.Steps {
		icon := "x"
		if step.Skipped {
			icon = "-"
		} else if step.Pass {
			icon = "v"
		}

		name := padRight(step.Name, 22)
		detail := step.Detail
		if step.Skipped {
			detail = "(skipped)"
		}

		b.WriteString(fmt.Sprintf("  %s %s %s\n", icon, name, detail))
	}

	b.WriteString("\n  " + strings.Repeat("-", 50) + "\n")

	// Result summary
	if result.Reachable {
		b.WriteString("  Result: v REACHABLE\n")
	} else {
		b.WriteString(fmt.Sprintf("  Result: x BLOCKED at %s\n", result.BlockedAt))
		if result.Suggestion != "" {
			b.WriteString(fmt.Sprintf("  Suggestion: %s\n", result.Suggestion))
		}
	}

	return b.String()
}

// padRight pads a string with spaces to the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
