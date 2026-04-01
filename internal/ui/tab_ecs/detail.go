package tab_ecs

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func RenderDetail(c internalaws.ECSCluster, services []internalaws.ECSService, loading bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n", c.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:              %s\n", c.Name))
	b.WriteString(fmt.Sprintf("  ARN:               %s\n", c.ARN))
	b.WriteString(fmt.Sprintf("  Status:            %s\n", c.Status))
	b.WriteString(fmt.Sprintf("  Running Tasks:     %d\n", c.RunningTasks))
	b.WriteString(fmt.Sprintf("  Pending Tasks:     %d\n", c.PendingTasks))
	b.WriteString(fmt.Sprintf("  Active Services:   %d\n", c.Services))
	b.WriteString(fmt.Sprintf("  Instances:         %d\n", c.Instances))

	if len(c.CapacityProviders) > 0 {
		b.WriteString(fmt.Sprintf("  Cap Providers:     %s\n", strings.Join(c.CapacityProviders, ", ")))
	}

	if loading {
		b.WriteString("\n  Loading services...\n")
	} else if len(services) > 0 {
		b.WriteString("\n  Services:\n")
		for _, svc := range services {
			b.WriteString(fmt.Sprintf("    %-25s %s  %d/%d  %s\n",
				svc.Name, svc.Status,
				svc.RunningCount, svc.DesiredCount,
				svc.LaunchType))
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}
