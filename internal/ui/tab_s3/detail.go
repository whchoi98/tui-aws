package tab_s3

import (
	"fmt"
	"strings"

	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

func RenderDetail(bkt internalaws.Bucket) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n", bkt.Name))
	b.WriteString("  ──────────────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Name:           %s\n", bkt.Name))
	b.WriteString(fmt.Sprintf("  Region:         %s\n", bkt.Region))
	b.WriteString(fmt.Sprintf("  Created:        %s\n", bkt.CreationDate))
	b.WriteString(fmt.Sprintf("  Versioning:     %s\n", displayOrDash(bkt.Versioning)))
	b.WriteString(fmt.Sprintf("  Encryption:     %s\n", displayOrDash(bkt.Encryption)))
	b.WriteString(fmt.Sprintf("  Public Access:  %s\n", displayOrDash(bkt.PublicAccess)))

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}

func displayOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
