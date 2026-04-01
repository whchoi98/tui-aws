package tab_acm

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// RenderCertDetail renders the ACM certificate detail overlay.
func RenderCertDetail(cert internalaws.Certificate) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s  (%s)\n", cert.DomainName, cert.Status))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Domain:         %s\n", cert.DomainName))
	b.WriteString(fmt.Sprintf("  ARN:            %s\n", cert.ARN))
	b.WriteString(fmt.Sprintf("  Status:         %s\n", cert.Status))
	b.WriteString(fmt.Sprintf("  Type:           %s\n", cert.Type))
	b.WriteString(fmt.Sprintf("  Key Algorithm:  %s\n", displayStr(cert.KeyAlgorithm)))
	b.WriteString(fmt.Sprintf("  Issuer:         %s\n", displayStr(cert.Issuer)))
	if cert.NotBefore != "" {
		b.WriteString(fmt.Sprintf("  Not Before:     %s\n", cert.NotBefore))
	}
	if cert.NotAfter != "" {
		b.WriteString(fmt.Sprintf("  Not After:      %s\n", cert.NotAfter))
	}
	if cert.RenewalStatus != "" {
		b.WriteString(fmt.Sprintf("  Renewal:        %s\n", cert.RenewalStatus))
	}

	if len(cert.SubjectAlternativeNames) > 0 {
		b.WriteString(fmt.Sprintf("\n  Subject Alternative Names (%d):\n", len(cert.SubjectAlternativeNames)))
		for _, san := range cert.SubjectAlternativeNames {
			b.WriteString(fmt.Sprintf("    %s\n", san))
		}
	}

	if len(cert.InUseBy) > 0 {
		b.WriteString(fmt.Sprintf("\n  In Use By (%d):\n", len(cert.InUseBy)))
		for _, arn := range cert.InUseBy {
			b.WriteString(fmt.Sprintf("    %s\n", arn))
		}
	} else {
		b.WriteString("\n  In Use By: (none)\n")
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}

func displayStr(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
