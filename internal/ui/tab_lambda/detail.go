package tab_lambda

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func RenderDetail(fn internalaws.LambdaFunction) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n", fn.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:             %s\n", fn.Name))
	b.WriteString(fmt.Sprintf("  ARN:              %s\n", fn.ARN))
	b.WriteString(fmt.Sprintf("  Runtime:          %s\n", fn.Runtime))
	b.WriteString(fmt.Sprintf("  Handler:          %s\n", fn.Handler))
	b.WriteString(fmt.Sprintf("  Memory:           %d MB\n", fn.MemorySize))
	b.WriteString(fmt.Sprintf("  Timeout:          %d seconds\n", fn.Timeout))
	b.WriteString(fmt.Sprintf("  Code Size:        %s\n", formatBytes(fn.CodeSize)))
	state := fn.State
	if state == "" {
		state = "Active"
	}
	b.WriteString(fmt.Sprintf("  State:            %s\n", state))
	b.WriteString(fmt.Sprintf("  Last Modified:    %s\n", fn.LastModified))

	if fn.Description != "" {
		b.WriteString(fmt.Sprintf("  Description:      %s\n", fn.Description))
	}

	if fn.VpcID != "" {
		b.WriteString(fmt.Sprintf("\n  VPC:              %s\n", fn.VpcID))
		if len(fn.SubnetIDs) > 0 {
			b.WriteString("  Subnets:\n")
			for _, sub := range fn.SubnetIDs {
				b.WriteString(fmt.Sprintf("    %s\n", sub))
			}
		}
		if len(fn.SecurityGroupIDs) > 0 {
			b.WriteString("  Security Groups:\n")
			for _, sg := range fn.SecurityGroupIDs {
				b.WriteString(fmt.Sprintf("    %s\n", sg))
			}
		}
	}

	if len(fn.Layers) > 0 {
		b.WriteString("\n  Layers:\n")
		for _, layer := range fn.Layers {
			b.WriteString(fmt.Sprintf("    %s\n", layer))
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
