package tab_rds

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func RenderDetail(inst internalaws.DBInstance) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (%s %s)\n", inst.ID, inst.Engine, inst.EngineVersion))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  ID:              %s\n", inst.ID))
	b.WriteString(fmt.Sprintf("  ARN:             %s\n", inst.ARN))
	b.WriteString(fmt.Sprintf("  Engine:          %s %s\n", inst.Engine, inst.EngineVersion))
	b.WriteString(fmt.Sprintf("  Class:           %s\n", inst.Class))
	b.WriteString(fmt.Sprintf("  Status:          %s\n", inst.Status))
	b.WriteString(fmt.Sprintf("  Endpoint:        %s\n", inst.Endpoint))
	b.WriteString(fmt.Sprintf("  Port:            %d\n", inst.Port))
	b.WriteString(fmt.Sprintf("  MultiAZ:         %s\n", boolLabel(inst.MultiAZ)))
	b.WriteString(fmt.Sprintf("  Storage:         %s %d GiB\n", inst.StorageType, inst.AllocatedStorage))
	b.WriteString(fmt.Sprintf("  Encrypted:       %s\n", boolLabel(inst.Encrypted)))
	b.WriteString(fmt.Sprintf("  Public:          %s\n", boolLabel(inst.PubliclyAccessible)))
	b.WriteString(fmt.Sprintf("  VPC:             %s\n", inst.VpcID))
	b.WriteString(fmt.Sprintf("  Subnet Group:    %s\n", inst.SubnetGroup))
	b.WriteString(fmt.Sprintf("  AZ:              %s\n", inst.AZ))
	if inst.CreatedTime != "" {
		b.WriteString(fmt.Sprintf("  Created:         %s\n", inst.CreatedTime))
	}

	if len(inst.SecurityGroups) > 0 {
		b.WriteString("\n  Security Groups:\n")
		for _, sg := range inst.SecurityGroups {
			b.WriteString(fmt.Sprintf("    %s\n", sg))
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}

func boolLabel(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
