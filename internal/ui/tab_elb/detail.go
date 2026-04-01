package tab_elb

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// RenderELBDetailInteractive renders the ELB detail with selectable target groups.
func RenderELBDetailInteractive(lb internalaws.LoadBalancer, loading bool, tgCursor int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (%s)\n", lb.Name, lb.TypeLabel()))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:          %s\n", lb.Name))
	if lb.ARN != "" {
		b.WriteString(fmt.Sprintf("  ARN:           %s\n", lb.ARN))
	}
	b.WriteString(fmt.Sprintf("  DNS:           %s\n", lb.DNSName))
	b.WriteString(fmt.Sprintf("  Type:          %s (%s)\n", lb.Type, lb.TypeLabel()))
	b.WriteString(fmt.Sprintf("  Scheme:        %s\n", lb.Scheme))
	b.WriteString(fmt.Sprintf("  State:         %s\n", lb.State))
	b.WriteString(fmt.Sprintf("  VPC:           %s\n", lb.VpcID))
	if lb.CreatedTime != "" {
		b.WriteString(fmt.Sprintf("  Created:       %s\n", lb.CreatedTime))
	}

	// Availability Zones
	if len(lb.AZs) > 0 {
		b.WriteString("\n  Availability Zones:\n")
		for _, az := range lb.AZs {
			b.WriteString(fmt.Sprintf("    %s\n", az))
		}
	}

	// Security Groups — show each SG ID
	if len(lb.SecurityGroups) > 0 {
		b.WriteString("\n  Security Groups:\n")
		for _, sg := range lb.SecurityGroups {
			b.WriteString(fmt.Sprintf("    %s\n", sg))
		}
	}

	if loading {
		b.WriteString("\n  Loading listeners and target groups...\n")
		b.WriteString("\n  Esc: close")
		return shared.RenderOverlay(b.String())
	}

	// Listeners — show protocol, port, and rule count
	if len(lb.Listeners) > 0 {
		b.WriteString("\n  Listeners:\n")
		for _, l := range lb.Listeners {
			line := fmt.Sprintf("    %-8s :%d", l.Protocol, l.Port)
			if l.Rules > 0 {
				line += fmt.Sprintf("  (%d rules)", l.Rules)
			}
			if l.ARN != "" {
				// Show shortened ARN suffix
				parts := strings.Split(l.ARN, "/")
				if len(parts) > 0 {
					line += fmt.Sprintf("  [%s]", parts[len(parts)-1])
				}
			}
			b.WriteString(line + "\n")
		}
	} else if lb.Type != "classic" {
		b.WriteString("\n  Listeners:     (none)\n")
	}

	// Target Groups — selectable with cursor
	if len(lb.TargetGroups) > 0 {
		b.WriteString("\n  Target Groups:  (↑↓ to select, Enter for detail)\n")
		for i, tg := range lb.TargetGroups {
			cursor := "  "
			if i == tgCursor {
				cursor = "▸ "
			}
			health := tg.HealthCheck
			if health == "" {
				health = "-"
			}
			line := fmt.Sprintf("  %s  %-20s  %s:%d  %-10s  %s",
				cursor, truncate(tg.Name, 20), tg.Protocol, tg.Port, tg.TargetType, health)
			b.WriteString(line + "\n")
		}
	} else if lb.Type != "classic" {
		b.WriteString("\n  Target Groups: (none)\n")
	}

	b.WriteString("\n  ↑↓: Select TG  Enter: TG Detail  Esc: Close")
	return shared.RenderOverlay(b.String())
}

// RenderTGDetail renders the target group detail with registered targets.
func RenderTGDetail(tg internalaws.TargetGroup, targets []internalaws.Target, loading bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  Target Group: %s\n", tg.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:         %s\n", tg.Name))
	b.WriteString(fmt.Sprintf("  ARN:          %s\n", tg.ARN))
	b.WriteString(fmt.Sprintf("  Protocol:     %s\n", tg.Protocol))
	b.WriteString(fmt.Sprintf("  Port:         %d\n", tg.Port))
	b.WriteString(fmt.Sprintf("  Target Type:  %s\n", tg.TargetType))
	b.WriteString(fmt.Sprintf("  VPC:          %s\n", tg.VpcID))
	if tg.HealthCheck != "" {
		b.WriteString(fmt.Sprintf("  Health:       %s\n", tg.HealthCheck))
	}

	if loading {
		b.WriteString("\n  Loading targets...\n")
		b.WriteString("\n  Esc: back to ELB")
		return shared.RenderOverlay(b.String())
	}

	if len(targets) > 0 {
		b.WriteString("\n  Registered Targets:\n")
		b.WriteString(fmt.Sprintf("  %-22s %-6s %-8s %-12s %s\n", "ID", "Port", "AZ", "Health", "Reason"))
		b.WriteString("  " + strings.Repeat("─", 65) + "\n")
		for _, t := range targets {
			id := truncate(t.ID, 22)
			az := t.AZ
			if az == "" {
				az = "-"
			}
			reason := t.Reason
			if reason == "" {
				reason = "-"
			}
			healthStyle := t.Health
			switch t.Health {
			case "healthy":
				healthStyle = shared.StateRunning.Render("healthy")
			case "unhealthy":
				healthStyle = shared.StateStopped.Render("unhealthy")
			case "draining":
				healthStyle = shared.StatePending.Render("draining")
			case "initial":
				healthStyle = shared.StatePending.Render("initial")
			case "unused":
				healthStyle = shared.StateTerminated.Render("unused")
			}
			b.WriteString(fmt.Sprintf("  %-22s %-6d %-8s %-12s %s\n",
				id, t.Port, az, healthStyle, truncate(reason, 30)))
		}
	} else {
		b.WriteString("\n  No registered targets.\n")
	}

	b.WriteString("\n  Esc: back to ELB")
	return shared.RenderOverlay(b.String())
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
