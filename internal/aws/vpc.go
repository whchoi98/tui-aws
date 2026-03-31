package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type VPC struct {
	VpcID     string
	Name      string
	CIDRBlock string
	State     string
	IsDefault bool
}

type InternetGateway struct {
	ID, Name, State string
	VpcID           string
}

type NATGateway struct {
	ID, Name, SubnetID, PrivateIP, PublicIP, State string
	VpcID                                          string
}

type VPCPeering struct {
	ID, Name, RequesterVpcID, AccepterVpcID, State string
}

type TGWAttachment struct {
	ID, TGWID, Name, State string
	VpcID                  string
}

type VPCEndpoint struct {
	ID, Name, ServiceName, Type, State string
	VpcID                              string
}

type ElasticIP struct {
	AllocationID, PublicIP, AssociationID, InstanceID, Name string
}

// FetchVPCs returns all VPCs with Name tags.
func FetchVPCs(ctx context.Context, client *ec2.Client) ([]VPC, error) {
	out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}
	vpcs := make([]VPC, 0, len(out.Vpcs))
	for _, v := range out.Vpcs {
		vpc := VPC{
			VpcID:     aws.ToString(v.VpcId),
			CIDRBlock: aws.ToString(v.CidrBlock),
			State:     string(v.State),
			IsDefault: aws.ToBool(v.IsDefault),
		}
		for _, tag := range v.Tags {
			if aws.ToString(tag.Key) == "Name" {
				vpc.Name = aws.ToString(tag.Value)
				break
			}
		}
		vpcs = append(vpcs, vpc)
	}
	return vpcs, nil
}

// FetchIGWs returns all internet gateways.
func FetchIGWs(ctx context.Context, client *ec2.Client) ([]InternetGateway, error) {
	out, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
	if err != nil {
		return nil, err
	}
	igws := make([]InternetGateway, 0, len(out.InternetGateways))
	for _, igw := range out.InternetGateways {
		g := InternetGateway{
			ID: aws.ToString(igw.InternetGatewayId),
		}
		for _, tag := range igw.Tags {
			if aws.ToString(tag.Key) == "Name" {
				g.Name = aws.ToString(tag.Value)
				break
			}
		}
		// An IGW is attached to at most one VPC.
		for _, att := range igw.Attachments {
			g.VpcID = aws.ToString(att.VpcId)
			g.State = string(att.State)
			break
		}
		igws = append(igws, g)
	}
	return igws, nil
}

// FetchNATGWs returns all NAT gateways.
func FetchNATGWs(ctx context.Context, client *ec2.Client) ([]NATGateway, error) {
	paginator := ec2.NewDescribeNatGatewaysPaginator(client, &ec2.DescribeNatGatewaysInput{})
	var natgws []NATGateway
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ngw := range page.NatGateways {
			g := NATGateway{
				ID:       aws.ToString(ngw.NatGatewayId),
				SubnetID: aws.ToString(ngw.SubnetId),
				VpcID:    aws.ToString(ngw.VpcId),
				State:    string(ngw.State),
			}
			for _, tag := range ngw.Tags {
				if aws.ToString(tag.Key) == "Name" {
					g.Name = aws.ToString(tag.Value)
					break
				}
			}
			// Pick the first address entry for private/public IPs.
			for _, addr := range ngw.NatGatewayAddresses {
				g.PrivateIP = aws.ToString(addr.PrivateIp)
				g.PublicIP = aws.ToString(addr.PublicIp)
				break
			}
			natgws = append(natgws, g)
		}
	}
	return natgws, nil
}

// FetchVPCPeerings returns all VPC peering connections.
func FetchVPCPeerings(ctx context.Context, client *ec2.Client) ([]VPCPeering, error) {
	out, err := client.DescribeVpcPeeringConnections(ctx, &ec2.DescribeVpcPeeringConnectionsInput{})
	if err != nil {
		return nil, err
	}
	peerings := make([]VPCPeering, 0, len(out.VpcPeeringConnections))
	for _, p := range out.VpcPeeringConnections {
		peer := VPCPeering{
			ID: aws.ToString(p.VpcPeeringConnectionId),
		}
		if p.RequesterVpcInfo != nil {
			peer.RequesterVpcID = aws.ToString(p.RequesterVpcInfo.VpcId)
		}
		if p.AccepterVpcInfo != nil {
			peer.AccepterVpcID = aws.ToString(p.AccepterVpcInfo.VpcId)
		}
		if p.Status != nil {
			peer.State = string(p.Status.Code)
		}
		for _, tag := range p.Tags {
			if aws.ToString(tag.Key) == "Name" {
				peer.Name = aws.ToString(tag.Value)
				break
			}
		}
		peerings = append(peerings, peer)
	}
	return peerings, nil
}

// FetchTGWAttachments returns all transit gateway attachments.
// Returns an empty slice on error (TGW may not be configured in all accounts).
func FetchTGWAttachments(ctx context.Context, client *ec2.Client) ([]TGWAttachment, error) {
	paginator := ec2.NewDescribeTransitGatewayAttachmentsPaginator(client, &ec2.DescribeTransitGatewayAttachmentsInput{})
	var attachments []TGWAttachment
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			// TGW may not be available; return empty rather than an error.
			return []TGWAttachment{}, nil
		}
		for _, att := range page.TransitGatewayAttachments {
			a := TGWAttachment{
				ID:    aws.ToString(att.TransitGatewayAttachmentId),
				TGWID: aws.ToString(att.TransitGatewayId),
				VpcID: aws.ToString(att.ResourceId),
				State: string(att.State),
			}
			for _, tag := range att.Tags {
				if aws.ToString(tag.Key) == "Name" {
					a.Name = aws.ToString(tag.Value)
					break
				}
			}
			attachments = append(attachments, a)
		}
	}
	return attachments, nil
}

// FetchVPCEndpoints returns all VPC endpoints.
func FetchVPCEndpoints(ctx context.Context, client *ec2.Client) ([]VPCEndpoint, error) {
	paginator := ec2.NewDescribeVpcEndpointsPaginator(client, &ec2.DescribeVpcEndpointsInput{})
	var endpoints []VPCEndpoint
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ep := range page.VpcEndpoints {
			e := VPCEndpoint{
				ID:          aws.ToString(ep.VpcEndpointId),
				ServiceName: aws.ToString(ep.ServiceName),
				Type:        string(ep.VpcEndpointType),
				State:       string(ep.State),
				VpcID:       aws.ToString(ep.VpcId),
			}
			for _, tag := range ep.Tags {
				if aws.ToString(tag.Key) == "Name" {
					e.Name = aws.ToString(tag.Value)
					break
				}
			}
			endpoints = append(endpoints, e)
		}
	}
	return endpoints, nil
}

// FetchEIPs returns all elastic IPs.
func FetchEIPs(ctx context.Context, client *ec2.Client) ([]ElasticIP, error) {
	out, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}
	eips := make([]ElasticIP, 0, len(out.Addresses))
	for _, addr := range out.Addresses {
		eip := ElasticIP{
			AllocationID:  aws.ToString(addr.AllocationId),
			PublicIP:      aws.ToString(addr.PublicIp),
			AssociationID: aws.ToString(addr.AssociationId),
			InstanceID:    aws.ToString(addr.InstanceId),
		}
		for _, tag := range addr.Tags {
			if aws.ToString(tag.Key) == "Name" {
				eip.Name = aws.ToString(tag.Value)
				break
			}
		}
		eips = append(eips, eip)
	}
	return eips, nil
}
