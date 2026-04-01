package tab_waf

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// RenderACLDetail renders the WAF Web ACL detail overlay.
func RenderACLDetail(acl internalaws.WebACL) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (%s)\n", acl.Name, acl.Scope))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:           %s\n", acl.Name))
	b.WriteString(fmt.Sprintf("  ID:             %s\n", acl.ID))
	b.WriteString(fmt.Sprintf("  ARN:            %s\n", acl.ARN))
	b.WriteString(fmt.Sprintf("  Scope:          %s\n", acl.Scope))
	b.WriteString(fmt.Sprintf("  Rules:          %d\n", acl.Rules))
	b.WriteString(fmt.Sprintf("  Default Action: %s\n", acl.DefaultAction))
	if acl.Description != "" {
		b.WriteString(fmt.Sprintf("  Description:    %s\n", acl.Description))
	}

	if len(acl.AssociatedResources) > 0 {
		b.WriteString(fmt.Sprintf("\n  Associated Resources (%d):\n", len(acl.AssociatedResources)))
		for _, arn := range acl.AssociatedResources {
			b.WriteString(fmt.Sprintf("    %s\n", arn))
		}
	} else {
		b.WriteString("\n  Associated Resources: (none)\n")
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}
