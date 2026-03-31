package tab_subnet

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// eniLoadedMsg is returned when ENIs for a subnet are fetched.
type eniLoadedMsg struct {
	enis []internalaws.ENI
	err  error
}

// loadENIs fetches ENIs filtered to the given subnet.
func loadENIs(subnetID, profile, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return eniLoadedMsg{err: err}
		}
		allENIs, err := internalaws.FetchENIs(ctx, clients.EC2)
		if err != nil {
			return eniLoadedMsg{err: err}
		}

		var filtered []internalaws.ENI
		for _, e := range allENIs {
			if e.SubnetID == subnetID {
				filtered = append(filtered, e)
			}
		}
		return eniLoadedMsg{enis: filtered}
	}
}

// RenderENIDetail renders the ENI list overlay for a subnet.
func RenderENIDetail(sub internalaws.Subnet, enis []internalaws.ENI, loading bool, detailErr error) string {
	var b strings.Builder

	name := sub.Name
	if name == "" {
		name = sub.SubnetID
	}
	b.WriteString(fmt.Sprintf("  ENIs in %s\n", name))
	b.WriteString("  ─────────────────────────────────\n")

	if loading {
		b.WriteString("  Loading ENIs...\n")
	} else if detailErr != nil {
		b.WriteString(fmt.Sprintf("  Error: %v\n", detailErr))
	} else if len(enis) == 0 {
		b.WriteString("  (no ENIs found)\n")
	} else {
		for _, eni := range enis {
			b.WriteString(fmt.Sprintf("  - %s  [%s]\n", eni.ID, eni.Status))
			if eni.Description != "" {
				b.WriteString(fmt.Sprintf("      Desc: %s\n", eni.Description))
			}
			b.WriteString(fmt.Sprintf("      IP:   %s", eni.PrivateIP))
			if eni.PublicIP != "" {
				b.WriteString(fmt.Sprintf(" / %s", eni.PublicIP))
			}
			b.WriteString("\n")
			if eni.AttachedInstanceID != "" {
				b.WriteString(fmt.Sprintf("      Instance: %s\n", eni.AttachedInstanceID))
			}
			if len(eni.SecurityGroups) > 0 {
				b.WriteString(fmt.Sprintf("      SG:   %s\n", strings.Join(eni.SecurityGroups, ", ")))
			}
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}
