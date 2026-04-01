package tab_ecs

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// renderClusterDetail renders the cluster info overlay.
func renderClusterDetail(c internalaws.ECSCluster) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Cluster: %s\n", c.Name))
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
	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}

// renderServiceDetail renders the service info overlay.
func renderServiceDetail(s internalaws.ECSService) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Service: %s\n", s.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:            %s\n", s.Name))
	b.WriteString(fmt.Sprintf("  ARN:             %s\n", s.ARN))
	b.WriteString(fmt.Sprintf("  Status:          %s\n", s.Status))
	b.WriteString(fmt.Sprintf("  Cluster ARN:     %s\n", s.ClusterARN))
	b.WriteString(fmt.Sprintf("  Desired Count:   %d\n", s.DesiredCount))
	b.WriteString(fmt.Sprintf("  Running Count:   %d\n", s.RunningCount))
	b.WriteString(fmt.Sprintf("  Pending Count:   %d\n", s.PendingCount))
	launch := s.LaunchType
	if launch == "" {
		launch = "-"
	}
	b.WriteString(fmt.Sprintf("  Launch Type:     %s\n", launch))
	b.WriteString(fmt.Sprintf("  Task Definition: %s\n", s.TaskDefinition))
	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}

// renderTaskDetail renders the task info overlay.
func renderTaskDetail(t internalaws.ECSTask) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Task: %s\n", t.ShortTaskID()))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Task ARN:        %s\n", t.TaskARN))
	b.WriteString(fmt.Sprintf("  Task Def:        %s\n", t.TaskDefinitionARN))
	b.WriteString(fmt.Sprintf("  Last Status:     %s\n", t.LastStatus))
	b.WriteString(fmt.Sprintf("  Desired Status:  %s\n", t.DesiredStatus))
	launch := t.LaunchType
	if launch == "" {
		launch = "-"
	}
	b.WriteString(fmt.Sprintf("  Launch Type:     %s\n", launch))
	cpu := t.CPU
	if cpu == "" {
		cpu = "-"
	}
	mem := t.Memory
	if mem == "" {
		mem = "-"
	}
	b.WriteString(fmt.Sprintf("  CPU:             %s\n", cpu))
	b.WriteString(fmt.Sprintf("  Memory:          %s\n", mem))
	b.WriteString(fmt.Sprintf("  Group:           %s\n", t.Group))
	if t.HealthStatus != "" && t.HealthStatus != "UNKNOWN" {
		b.WriteString(fmt.Sprintf("  Health:          %s\n", t.HealthStatus))
	}
	if t.ConnectivityStatus != "" {
		b.WriteString(fmt.Sprintf("  Connectivity:    %s\n", t.ConnectivityStatus))
	}
	if t.StartedAt != "" {
		b.WriteString(fmt.Sprintf("  Started At:      %s\n", t.StartedAt))
	}
	if t.StoppedAt != "" {
		b.WriteString(fmt.Sprintf("  Stopped At:      %s\n", t.StoppedAt))
	}
	if t.StoppedReason != "" {
		b.WriteString(fmt.Sprintf("  Stopped Reason:  %s\n", t.StoppedReason))
	}

	if len(t.Containers) > 0 {
		b.WriteString("\n  Containers:\n")
		for _, c := range t.Containers {
			exit := ""
			if c.ExitCode != nil {
				exit = fmt.Sprintf("  exit:%d", *c.ExitCode)
			}
			b.WriteString(fmt.Sprintf("    %-20s %-10s%s\n", c.Name, c.Status, exit))
		}
	}

	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}

// renderTaskDefDetail renders task definition container definitions.
func renderTaskDefDetail(defs []internalaws.ECSContainerDef) string {
	var b strings.Builder
	b.WriteString("  Task Definition\n")
	b.WriteString("  ──────────────────────────────────────────────────\n")

	for i, d := range defs {
		if i > 0 {
			b.WriteString("\n")
		}
		essential := ""
		if d.Essential {
			essential = " (essential)"
		}
		b.WriteString(fmt.Sprintf("  Container: %s%s\n", d.Name, essential))
		b.WriteString(fmt.Sprintf("    Image:   %s\n", d.Image))
		if d.CPU > 0 {
			b.WriteString(fmt.Sprintf("    CPU:     %d\n", d.CPU))
		}
		if d.Memory > 0 {
			b.WriteString(fmt.Sprintf("    Memory:  %d MiB\n", d.Memory))
		}
		if len(d.PortMappings) > 0 {
			b.WriteString(fmt.Sprintf("    Ports:   %s\n", strings.Join(d.PortMappings, ", ")))
		}
		if d.LogGroup != "" {
			b.WriteString(fmt.Sprintf("    Log Group:  %s\n", d.LogGroup))
			if d.LogStreamPrefix != "" {
				b.WriteString(fmt.Sprintf("    Log Prefix: %s\n", d.LogStreamPrefix))
			}
		}
		if len(d.Environment) > 0 {
			b.WriteString("    Environment:\n")
			for k, v := range d.Environment {
				b.WriteString(fmt.Sprintf("      %s = %s\n", k, v))
			}
		}
	}

	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}

// renderContainerDetail renders a single container's details.
func renderContainerDetail(c internalaws.ECSContainer) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Container: %s\n", c.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:          %s\n", c.Name))
	b.WriteString(fmt.Sprintf("  Image:         %s\n", c.Image))
	b.WriteString(fmt.Sprintf("  Status:        %s\n", c.Status))
	if c.HealthStatus != "" && c.HealthStatus != "UNKNOWN" {
		b.WriteString(fmt.Sprintf("  Health:        %s\n", c.HealthStatus))
	}
	if c.RuntimeID != "" {
		b.WriteString(fmt.Sprintf("  Runtime ID:    %s\n", c.RuntimeID))
	}
	if c.ContainerARN != "" {
		b.WriteString(fmt.Sprintf("  ARN:           %s\n", c.ContainerARN))
	}
	if c.ExitCode != nil {
		b.WriteString(fmt.Sprintf("  Exit Code:     %d\n", *c.ExitCode))
	}
	if c.Reason != "" {
		b.WriteString(fmt.Sprintf("  Reason:        %s\n", c.Reason))
	}
	if c.CPU > 0 {
		b.WriteString(fmt.Sprintf("  CPU:           %d\n", c.CPU))
	}
	if c.Memory > 0 {
		b.WriteString(fmt.Sprintf("  Memory:        %d MiB\n", c.Memory))
	}
	if len(c.Ports) > 0 {
		b.WriteString(fmt.Sprintf("  Ports:         %s\n", strings.Join(c.Ports, ", ")))
	}
	if c.LogGroup != "" {
		b.WriteString(fmt.Sprintf("  Log Group:     %s\n", c.LogGroup))
	}
	if c.LogStream != "" {
		b.WriteString(fmt.Sprintf("  Log Stream:    %s\n", c.LogStream))
	}
	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}

// renderLogsOverlay renders the log viewer.
func renderLogsOverlay(logs []internalaws.LogEvent, logsErr error, container *internalaws.ECSContainer) string {
	var b strings.Builder

	title := "Container Logs"
	if container != nil {
		title = fmt.Sprintf("Logs: %s", container.Name)
	}
	b.WriteString(fmt.Sprintf("  %s\n", title))
	b.WriteString("  ──────────────────────────────────────────────────\n")

	if logsErr != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v\n", logsErr))
	} else if len(logs) == 0 {
		if container != nil && (container.LogGroup == "" || container.LogStream == "") {
			b.WriteString("\n  No log configuration found for this container.\n")
			b.WriteString("  (awslogs driver not configured in task definition)\n")
		} else {
			b.WriteString("\n  No log events found.\n")
		}
	} else {
		for _, ev := range logs {
			msg := strings.TrimRight(ev.Message, "\n")
			if ev.Timestamp != "" {
				b.WriteString(fmt.Sprintf("  %s  %s\n", ev.Timestamp, msg))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", msg))
			}
		}
	}

	b.WriteString("\n  Esc: close")
	return shared.RenderOverlay(b.String())
}
