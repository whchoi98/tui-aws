package tab_cloudfront

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// RenderDistributionDetail renders the CloudFront distribution detail overlay.
func RenderDistributionDetail(d internalaws.Distribution) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  %s\n", d.ID, d.DomainName))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  ID:            %s\n", d.ID))
	b.WriteString(fmt.Sprintf("  ARN:           %s\n", d.ARN))
	b.WriteString(fmt.Sprintf("  Domain:        %s\n", d.DomainName))
	b.WriteString(fmt.Sprintf("  Status:        %s\n", d.Status))
	b.WriteString(fmt.Sprintf("  Enabled:       %s\n", boolLabel(d.Enabled)))
	b.WriteString(fmt.Sprintf("  Price Class:   %s\n", displayStr(d.PriceClass)))
	b.WriteString(fmt.Sprintf("  Comment:       %s\n", displayStr(d.Comment)))
	if d.LastModified != "" {
		b.WriteString(fmt.Sprintf("  Last Modified: %s\n", d.LastModified))
	}

	if d.WebACLID != "" {
		b.WriteString(fmt.Sprintf("\n  WAF Web ACL:   %s\n", d.WebACLID))
	}

	if d.CertificateARN != "" {
		b.WriteString(fmt.Sprintf("  Certificate:   %s\n", d.CertificateARN))
	}

	if len(d.Origins) > 0 {
		b.WriteString("\n  Origins:\n")
		for _, o := range d.Origins {
			b.WriteString(fmt.Sprintf("    %s\n", o))
		}
	}

	if len(d.Aliases) > 0 {
		b.WriteString("\n  Aliases (CNAMEs):\n")
		for _, a := range d.Aliases {
			b.WriteString(fmt.Sprintf("    %s\n", a))
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

func displayStr(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
