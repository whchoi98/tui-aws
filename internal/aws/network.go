package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type RouteTable struct {
	ID      string
	Name    string
	VpcID   string
	Subnets []string // associated subnet IDs
	IsMain  bool
	Routes  []Route
}

type Route struct {
	Destination string // 0.0.0.0/0, 10.0.0.0/16, pl-xxx etc.
	Target      string // igw-xxx, nat-xxx, tgw-xxx, pcx-xxx, local, etc.
	State       string // active, blackhole
}

// FetchRouteTables returns all route tables with their routes and subnet associations.
func FetchRouteTables(ctx context.Context, client *ec2.Client) ([]RouteTable, error) {
	out, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{})
	if err != nil {
		return nil, err
	}

	rts := make([]RouteTable, 0, len(out.RouteTables))
	for _, rt := range out.RouteTables {
		r := RouteTable{
			ID:    aws.ToString(rt.RouteTableId),
			VpcID: aws.ToString(rt.VpcId),
		}

		// Parse Name from Tags.
		for _, tag := range rt.Tags {
			if aws.ToString(tag.Key) == "Name" {
				r.Name = aws.ToString(tag.Value)
				break
			}
		}

		// Parse associations: detect main and collect subnet IDs.
		for _, assoc := range rt.Associations {
			if aws.ToBool(assoc.Main) {
				r.IsMain = true
			} else if assoc.SubnetId != nil {
				r.Subnets = append(r.Subnets, aws.ToString(assoc.SubnetId))
			}
		}

		// Parse routes.
		for _, route := range rt.Routes {
			dest := ""
			switch {
			case route.DestinationCidrBlock != nil:
				dest = aws.ToString(route.DestinationCidrBlock)
			case route.DestinationIpv6CidrBlock != nil:
				dest = aws.ToString(route.DestinationIpv6CidrBlock)
			case route.DestinationPrefixListId != nil:
				dest = aws.ToString(route.DestinationPrefixListId)
			}

			target := ""
			switch {
			case route.GatewayId != nil:
				target = aws.ToString(route.GatewayId)
			case route.NatGatewayId != nil:
				target = aws.ToString(route.NatGatewayId)
			case route.TransitGatewayId != nil:
				target = aws.ToString(route.TransitGatewayId)
			case route.VpcPeeringConnectionId != nil:
				target = aws.ToString(route.VpcPeeringConnectionId)
			case route.NetworkInterfaceId != nil:
				target = aws.ToString(route.NetworkInterfaceId)
			case route.InstanceId != nil:
				target = aws.ToString(route.InstanceId)
			default:
				target = "local"
			}

			r.Routes = append(r.Routes, Route{
				Destination: dest,
				Target:      target,
				State:       string(route.State),
			})
		}

		rts = append(rts, r)
	}
	return rts, nil
}
