package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type SecurityGroup struct {
	ID, Name, Description, VpcID string
	InboundRules                  []SGRule
	OutboundRules                 []SGRule
}

type SGRule struct {
	Protocol    string // tcp, udp, icmp, -1 (all)
	PortRange   string // 80, 443, 1024-65535, All
	Source      string // CIDR, sg-xxx, pl-xxx (for inbound) or destination (for outbound)
	Description string
}

type NetworkACL struct {
	ID, Name, VpcID string
	IsDefault       bool
	Subnets         []string // associated subnet IDs
	InboundRules    []NACLRule
	OutboundRules   []NACLRule
}

type NACLRule struct {
	RuleNumber int    // 100, 200, 32767 (for default deny shown as *)
	Protocol   string // tcp, udp, icmp, -1 (all)
	PortRange  string // 80, 443, 1024-65535, All
	CIDRBlock  string
	Action     string // allow, deny
}

// FetchSecurityGroups returns all security groups with their rules.
func FetchSecurityGroups(ctx context.Context, client *ec2.Client) ([]SecurityGroup, error) {
	out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	sgs := make([]SecurityGroup, 0, len(out.SecurityGroups))
	for _, sg := range out.SecurityGroups {
		g := SecurityGroup{
			ID:          aws.ToString(sg.GroupId),
			Name:        aws.ToString(sg.GroupName),
			Description: aws.ToString(sg.Description),
			VpcID:       aws.ToString(sg.VpcId),
		}

		for _, perm := range sg.IpPermissions {
			g.InboundRules = append(g.InboundRules, parseIpPermission(perm)...)
		}

		for _, perm := range sg.IpPermissionsEgress {
			g.OutboundRules = append(g.OutboundRules, parseIpPermission(perm)...)
		}

		sgs = append(sgs, g)
	}
	return sgs, nil
}

// parseIpPermission converts one ec2types.IpPermission into SGRule values,
// producing one rule per source CIDR / security-group / prefix-list entry.
func parseIpPermission(perm ec2types.IpPermission) []SGRule {
	protocol := aws.ToString(perm.IpProtocol)
	if protocol == "-1" {
		protocol = "All"
	}
	portRange := portRangeStr(perm.FromPort, perm.ToPort)

	var rules []SGRule

	for _, r := range perm.IpRanges {
		rules = append(rules, SGRule{
			Protocol:    protocol,
			PortRange:   portRange,
			Source:      aws.ToString(r.CidrIp),
			Description: aws.ToString(r.Description),
		})
	}

	for _, r := range perm.Ipv6Ranges {
		rules = append(rules, SGRule{
			Protocol:    protocol,
			PortRange:   portRange,
			Source:      aws.ToString(r.CidrIpv6),
			Description: aws.ToString(r.Description),
		})
	}

	for _, r := range perm.UserIdGroupPairs {
		rules = append(rules, SGRule{
			Protocol:    protocol,
			PortRange:   portRange,
			Source:      aws.ToString(r.GroupId),
			Description: aws.ToString(r.Description),
		})
	}

	for _, r := range perm.PrefixListIds {
		rules = append(rules, SGRule{
			Protocol:    protocol,
			PortRange:   portRange,
			Source:      aws.ToString(r.PrefixListId),
			Description: aws.ToString(r.Description),
		})
	}

	// If no specific source was listed, emit a single rule with an empty source.
	if len(rules) == 0 {
		rules = append(rules, SGRule{
			Protocol:  protocol,
			PortRange: portRange,
		})
	}

	return rules
}

// portRangeStr converts from/to port pointers into a human-readable string.
func portRangeStr(from, to *int32) string {
	if from == nil || to == nil {
		return "All"
	}
	f := aws.ToInt32(from)
	t := aws.ToInt32(to)
	if f == -1 && t == -1 {
		return "All"
	}
	if f == t {
		return fmt.Sprintf("%d", f)
	}
	return fmt.Sprintf("%d-%d", f, t)
}

// FetchNetworkACLs returns all network ACLs with their rules.
func FetchNetworkACLs(ctx context.Context, client *ec2.Client) ([]NetworkACL, error) {
	out, err := client.DescribeNetworkAcls(ctx, &ec2.DescribeNetworkAclsInput{})
	if err != nil {
		return nil, err
	}

	acls := make([]NetworkACL, 0, len(out.NetworkAcls))
	for _, nacl := range out.NetworkAcls {
		a := NetworkACL{
			ID:        aws.ToString(nacl.NetworkAclId),
			VpcID:     aws.ToString(nacl.VpcId),
			IsDefault: aws.ToBool(nacl.IsDefault),
		}

		for _, tag := range nacl.Tags {
			if aws.ToString(tag.Key) == "Name" {
				a.Name = aws.ToString(tag.Value)
				break
			}
		}

		for _, assoc := range nacl.Associations {
			if assoc.SubnetId != nil {
				a.Subnets = append(a.Subnets, aws.ToString(assoc.SubnetId))
			}
		}

		for _, entry := range nacl.Entries {
			rule := NACLRule{
				RuleNumber: int(aws.ToInt32(entry.RuleNumber)),
				CIDRBlock:  aws.ToString(entry.CidrBlock),
				Action:     string(entry.RuleAction),
			}

			proto := aws.ToString(entry.Protocol)
			switch proto {
			case "-1":
				rule.Protocol = "All"
			case "6":
				rule.Protocol = "tcp"
			case "17":
				rule.Protocol = "udp"
			case "1":
				rule.Protocol = "icmp"
			default:
				rule.Protocol = proto
			}

			if entry.PortRange != nil {
				rule.PortRange = portRangeStr(entry.PortRange.From, entry.PortRange.To)
			} else {
				rule.PortRange = "All"
			}

			if aws.ToBool(entry.Egress) {
				a.OutboundRules = append(a.OutboundRules, rule)
			} else {
				a.InboundRules = append(a.InboundRules, rule)
			}
		}

		acls = append(acls, a)
	}
	return acls, nil
}
