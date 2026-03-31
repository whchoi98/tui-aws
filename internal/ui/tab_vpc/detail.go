package tab_vpc

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	internalaws "tui-aws/internal/aws"
	"tui-aws/internal/ui/shared"
)

// vpcDetailData holds all sub-resources for a VPC detail view.
type vpcDetailData struct {
	igws      []internalaws.InternetGateway
	natgws    []internalaws.NATGateway
	peerings  []internalaws.VPCPeering
	tgwAtts   []internalaws.TGWAttachment
	endpoints []internalaws.VPCEndpoint
	eips      []internalaws.ElasticIP
}

// vpcDetailLoadedMsg is returned when VPC detail sub-resources are fetched.
type vpcDetailLoadedMsg struct {
	data vpcDetailData
	err  error
}

// loadVPCDetail fetches all sub-resources for the given VPC.
func loadVPCDetail(vpcID, profile, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, profile, region)
		if err != nil {
			return vpcDetailLoadedMsg{err: err}
		}

		var data vpcDetailData

		data.igws, _ = internalaws.FetchIGWs(ctx, clients.EC2)
		data.natgws, _ = internalaws.FetchNATGWs(ctx, clients.EC2)
		data.peerings, _ = internalaws.FetchVPCPeerings(ctx, clients.EC2)
		data.tgwAtts, _ = internalaws.FetchTGWAttachments(ctx, clients.EC2)
		data.endpoints, _ = internalaws.FetchVPCEndpoints(ctx, clients.EC2)
		data.eips, _ = internalaws.FetchEIPs(ctx, clients.EC2)

		// Filter to the selected VPC
		data.igws = filterIGWs(data.igws, vpcID)
		data.natgws = filterNATGWs(data.natgws, vpcID)
		data.peerings = filterPeerings(data.peerings, vpcID)
		data.tgwAtts = filterTGWAtts(data.tgwAtts, vpcID)
		data.endpoints = filterEndpoints(data.endpoints, vpcID)

		return vpcDetailLoadedMsg{data: data}
	}
}

func filterIGWs(igws []internalaws.InternetGateway, vpcID string) []internalaws.InternetGateway {
	var result []internalaws.InternetGateway
	for _, g := range igws {
		if g.VpcID == vpcID {
			result = append(result, g)
		}
	}
	return result
}

func filterNATGWs(natgws []internalaws.NATGateway, vpcID string) []internalaws.NATGateway {
	var result []internalaws.NATGateway
	for _, g := range natgws {
		if g.VpcID == vpcID {
			result = append(result, g)
		}
	}
	return result
}

func filterPeerings(peerings []internalaws.VPCPeering, vpcID string) []internalaws.VPCPeering {
	var result []internalaws.VPCPeering
	for _, p := range peerings {
		if p.RequesterVpcID == vpcID || p.AccepterVpcID == vpcID {
			result = append(result, p)
		}
	}
	return result
}

func filterTGWAtts(atts []internalaws.TGWAttachment, vpcID string) []internalaws.TGWAttachment {
	var result []internalaws.TGWAttachment
	for _, a := range atts {
		if a.VpcID == vpcID {
			result = append(result, a)
		}
	}
	return result
}

func filterEndpoints(eps []internalaws.VPCEndpoint, vpcID string) []internalaws.VPCEndpoint {
	var result []internalaws.VPCEndpoint
	for _, e := range eps {
		if e.VpcID == vpcID {
			result = append(result, e)
		}
	}
	return result
}

// RenderVPCDetail renders the VPC detail overlay.
func RenderVPCDetail(vpc internalaws.VPC, detail vpcDetailData, detailLoading bool, detailErr error) string {
	var b strings.Builder

	name := vpc.Name
	if name == "" {
		name = vpc.VpcID
	}
	b.WriteString(fmt.Sprintf("  VPC Details - %s\n", name))
	b.WriteString("  ─────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  ID:      %s\n", vpc.VpcID))
	b.WriteString(fmt.Sprintf("  CIDR:    %s\n", vpc.CIDRBlock))
	b.WriteString(fmt.Sprintf("  State:   %s\n", vpc.State))
	if vpc.IsDefault {
		b.WriteString("  Default: Yes\n")
	}

	if detailLoading {
		b.WriteString("\n  Loading sub-resources...\n")
	} else if detailErr != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v\n", detailErr))
	} else {
		// Internet Gateways
		b.WriteString("\n  Internet Gateways:\n")
		if len(detail.igws) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, igw := range detail.igws {
			label := igw.ID
			if igw.Name != "" {
				label = fmt.Sprintf("%s (%s)", igw.Name, igw.ID)
			}
			b.WriteString(fmt.Sprintf("    - %s  [%s]\n", label, igw.State))
		}

		// NAT Gateways
		b.WriteString("\n  NAT Gateways:\n")
		if len(detail.natgws) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, ngw := range detail.natgws {
			label := ngw.ID
			if ngw.Name != "" {
				label = fmt.Sprintf("%s (%s)", ngw.Name, ngw.ID)
			}
			ips := ngw.PrivateIP
			if ngw.PublicIP != "" {
				ips += " / " + ngw.PublicIP
			}
			b.WriteString(fmt.Sprintf("    - %s  [%s]  %s\n", label, ngw.State, ips))
		}

		// Peering connections
		b.WriteString("\n  VPC Peering:\n")
		if len(detail.peerings) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, p := range detail.peerings {
			label := p.ID
			if p.Name != "" {
				label = fmt.Sprintf("%s (%s)", p.Name, p.ID)
			}
			b.WriteString(fmt.Sprintf("    - %s  [%s]  %s <-> %s\n", label, p.State, p.RequesterVpcID, p.AccepterVpcID))
		}

		// Transit Gateway Attachments
		b.WriteString("\n  TGW Attachments:\n")
		if len(detail.tgwAtts) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, a := range detail.tgwAtts {
			label := a.ID
			if a.Name != "" {
				label = fmt.Sprintf("%s (%s)", a.Name, a.ID)
			}
			b.WriteString(fmt.Sprintf("    - %s  TGW: %s  [%s]\n", label, a.TGWID, a.State))
		}

		// VPC Endpoints
		b.WriteString("\n  VPC Endpoints:\n")
		if len(detail.endpoints) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, e := range detail.endpoints {
			label := e.ID
			if e.Name != "" {
				label = fmt.Sprintf("%s (%s)", e.Name, e.ID)
			}
			b.WriteString(fmt.Sprintf("    - %s  %s  [%s]  %s\n", label, e.ServiceName, e.State, e.Type))
		}

		// Elastic IPs
		b.WriteString("\n  Elastic IPs:\n")
		if len(detail.eips) == 0 {
			b.WriteString("    (none)\n")
		}
		for _, eip := range detail.eips {
			label := eip.AllocationID
			if eip.Name != "" {
				label = fmt.Sprintf("%s (%s)", eip.Name, eip.AllocationID)
			}
			extra := eip.PublicIP
			if eip.InstanceID != "" {
				extra += " -> " + eip.InstanceID
			}
			b.WriteString(fmt.Sprintf("    - %s  %s\n", label, extra))
		}
	}

	b.WriteString("\n  Press any key to close")
	return shared.RenderOverlay(b.String())
}
