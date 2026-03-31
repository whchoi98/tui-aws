package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type Subnet struct {
	SubnetID     string
	Name         string
	VpcID        string
	CIDRBlock    string
	AZ           string
	AvailableIPs int
	MapPublicIP  bool
	RouteTableID string // populated by EnrichSubnetRouteTables
}

type ENI struct {
	ID, Description, SubnetID, PrivateIP, PublicIP, Status string
	AttachedInstanceID                                      string
	SecurityGroups                                          []string
}

// FetchSubnets returns all subnets with Name tags.
func FetchSubnets(ctx context.Context, client *ec2.Client) ([]Subnet, error) {
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})
	var subnets []Subnet
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, s := range page.Subnets {
			sub := Subnet{
				SubnetID:     aws.ToString(s.SubnetId),
				VpcID:        aws.ToString(s.VpcId),
				CIDRBlock:    aws.ToString(s.CidrBlock),
				AZ:           aws.ToString(s.AvailabilityZone),
				AvailableIPs: int(aws.ToInt32(s.AvailableIpAddressCount)),
				MapPublicIP:  aws.ToBool(s.MapPublicIpOnLaunch),
			}
			for _, tag := range s.Tags {
				if aws.ToString(tag.Key) == "Name" {
					sub.Name = aws.ToString(tag.Value)
					break
				}
			}
			subnets = append(subnets, sub)
		}
	}
	return subnets, nil
}

// FetchENIs returns all ENIs.
func FetchENIs(ctx context.Context, client *ec2.Client) ([]ENI, error) {
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(client, &ec2.DescribeNetworkInterfacesInput{})
	var enis []ENI
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ni := range page.NetworkInterfaces {
			e := ENI{
				ID:          aws.ToString(ni.NetworkInterfaceId),
				Description: aws.ToString(ni.Description),
				SubnetID:    aws.ToString(ni.SubnetId),
				PrivateIP:   aws.ToString(ni.PrivateIpAddress),
				Status:      string(ni.Status),
			}
			if ni.Association != nil {
				e.PublicIP = aws.ToString(ni.Association.PublicIp)
			}
			if ni.Attachment != nil {
				e.AttachedInstanceID = aws.ToString(ni.Attachment.InstanceId)
			}
			for _, sg := range ni.Groups {
				e.SecurityGroups = append(e.SecurityGroups, aws.ToString(sg.GroupName))
			}
			enis = append(enis, e)
		}
	}
	return enis, nil
}
