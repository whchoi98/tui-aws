package tab_eks

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func RenderDetail(c internalaws.EKSCluster, nodeGroups []internalaws.EKSNodeGroup, loading bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (v%s)\n", c.Name, c.Version))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:             %s\n", c.Name))
	b.WriteString(fmt.Sprintf("  ARN:              %s\n", c.ARN))
	b.WriteString(fmt.Sprintf("  Version:          %s\n", c.Version))
	b.WriteString(fmt.Sprintf("  Platform:         %s\n", c.PlatformVersion))
	b.WriteString(fmt.Sprintf("  Status:           %s\n", c.Status))
	b.WriteString(fmt.Sprintf("  Endpoint:         %s\n", c.Endpoint))
	b.WriteString(fmt.Sprintf("  VPC:              %s\n", c.VpcID))
	if c.CreatedTime != "" {
		b.WriteString(fmt.Sprintf("  Created:          %s\n", c.CreatedTime))
	}

	if len(c.SubnetIDs) > 0 {
		b.WriteString("\n  Subnets:\n")
		for _, sub := range c.SubnetIDs {
			b.WriteString(fmt.Sprintf("    %s\n", sub))
		}
	}

	if len(c.SecurityGroupIDs) > 0 {
		b.WriteString("\n  Security Groups:\n")
		for _, sg := range c.SecurityGroupIDs {
			b.WriteString(fmt.Sprintf("    %s\n", sg))
		}
	}

	if loading {
		b.WriteString("\n  Loading node groups...\n")
	} else if len(nodeGroups) > 0 {
		b.WriteString("\n  Node Groups:\n")
		for _, ng := range nodeGroups {
			b.WriteString(fmt.Sprintf("    %-20s %s  %s  %d/%d/%d  %s\n",
				ng.Name, ng.Status, ng.InstanceTypes,
				ng.MinSize, ng.DesiredSize, ng.MaxSize,
				ng.AmiType))
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}
