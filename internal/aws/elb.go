package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

// LoadBalancer represents an ALB, NLB, GWLB, or CLB.
type LoadBalancer struct {
	Name           string
	ARN            string
	DNSName        string
	Type           string // "application" (ALB), "network" (NLB), "gateway" (GWLB), "classic" (CLB)
	Scheme         string // "internet-facing" or "internal"
	State          string // "active", "provisioning", "failed"
	VpcID          string
	AZs            []string // availability zones
	SecurityGroups []string // ALB only
	CreatedTime    string
	// Listener info (fetched on demand)
	Listeners []Listener
	// Target group info (fetched on demand)
	TargetGroups []TargetGroup
}

// Listener represents a load balancer listener.
type Listener struct {
	ARN      string
	Port     int
	Protocol string // HTTP, HTTPS, TCP, TLS, UDP, TCP_UDP
	Rules    int    // number of rules (ALB)
}

// TargetGroup represents a target group attached to a load balancer.
type TargetGroup struct {
	Name        string
	ARN         string
	Port        int
	Protocol    string
	TargetType  string // instance, ip, lambda, alb
	HealthCheck string // healthy/unhealthy summary
	VpcID       string
	Targets     []Target // populated on demand
}

// Target represents a registered target in a target group.
type Target struct {
	ID     string // instance ID, IP, or Lambda ARN
	Port   int
	AZ     string
	Health string // healthy, unhealthy, draining, initial, unused
	Reason string // health check failure reason
}

// FetchTargets returns the registered targets and their health for a target group.
func FetchTargets(ctx context.Context, elbv2Client *elbv2.Client, tgARN string) ([]Target, error) {
	out, err := elbv2Client.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(tgARN),
	})
	if err != nil {
		return nil, err
	}

	targets := make([]Target, 0, len(out.TargetHealthDescriptions))
	for _, desc := range out.TargetHealthDescriptions {
		t := Target{}
		if desc.Target != nil {
			t.ID = aws.ToString(desc.Target.Id)
			if desc.Target.Port != nil {
				t.Port = int(aws.ToInt32(desc.Target.Port))
			}
			if desc.Target.AvailabilityZone != nil {
				t.AZ = aws.ToString(desc.Target.AvailabilityZone)
			}
		}
		if desc.TargetHealth != nil {
			t.Health = string(desc.TargetHealth.State)
			t.Reason = string(desc.TargetHealth.Reason)
			if desc.TargetHealth.Description != nil {
				reason := aws.ToString(desc.TargetHealth.Description)
				if reason != "" {
					t.Reason = reason
				}
			}
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// TypeLabel returns a short display label for the load balancer type.
func (lb *LoadBalancer) TypeLabel() string {
	switch lb.Type {
	case "application":
		return "ALB"
	case "network":
		return "NLB"
	case "gateway":
		return "GWLB"
	case "classic":
		return "CLB"
	default:
		return strings.ToUpper(lb.Type)
	}
}

// FetchLoadBalancers returns all ALBs, NLBs, and GWLBs via DescribeLoadBalancers (elbv2).
func FetchLoadBalancers(ctx context.Context, elbv2Client *elbv2.Client) ([]LoadBalancer, error) {
	paginator := elbv2.NewDescribeLoadBalancersPaginator(elbv2Client, &elbv2.DescribeLoadBalancersInput{})
	var lbs []LoadBalancer
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, lb := range page.LoadBalancers {
			azNames := make([]string, 0, len(lb.AvailabilityZones))
			for _, az := range lb.AvailabilityZones {
				azNames = append(azNames, aws.ToString(az.ZoneName))
			}

			state := ""
			if lb.State != nil {
				state = string(lb.State.Code)
			}

			createdTime := ""
			if lb.CreatedTime != nil {
				createdTime = lb.CreatedTime.Format("2006-01-02 15:04")
			}

			entry := LoadBalancer{
				Name:           aws.ToString(lb.LoadBalancerName),
				ARN:            aws.ToString(lb.LoadBalancerArn),
				DNSName:        aws.ToString(lb.DNSName),
				Type:           string(lb.Type),
				Scheme:         string(lb.Scheme),
				State:          state,
				VpcID:          aws.ToString(lb.VpcId),
				AZs:            azNames,
				SecurityGroups: lb.SecurityGroups,
				CreatedTime:    createdTime,
			}
			lbs = append(lbs, entry)
		}
	}
	return lbs, nil
}

// FetchClassicLoadBalancers returns all CLBs via DescribeLoadBalancers (elb).
func FetchClassicLoadBalancers(ctx context.Context, elbClient *elb.Client) ([]LoadBalancer, error) {
	paginator := elb.NewDescribeLoadBalancersPaginator(elbClient, &elb.DescribeLoadBalancersInput{})
	var lbs []LoadBalancer
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, lb := range page.LoadBalancerDescriptions {
			azNames := make([]string, len(lb.AvailabilityZones))
			copy(azNames, lb.AvailabilityZones)

			createdTime := ""
			if lb.CreatedTime != nil {
				createdTime = lb.CreatedTime.Format("2006-01-02 15:04")
			}

			scheme := aws.ToString(lb.Scheme)
			if scheme == "" {
				scheme = "internet-facing"
			}

			dnsName := aws.ToString(lb.DNSName)

			entry := LoadBalancer{
				Name:           aws.ToString(lb.LoadBalancerName),
				DNSName:        dnsName,
				Type:           "classic",
				Scheme:         scheme,
				State:          "active", // CLBs don't have a state field; if returned they are active
				VpcID:          aws.ToString(lb.VPCId),
				AZs:            azNames,
				SecurityGroups: lb.SecurityGroups,
				CreatedTime:    createdTime,
			}
			lbs = append(lbs, entry)
		}
	}
	return lbs, nil
}

// FetchListeners returns listeners for a load balancer ARN (elbv2 only).
func FetchListeners(ctx context.Context, elbv2Client *elbv2.Client, lbARN string) ([]Listener, error) {
	out, err := elbv2Client.DescribeListeners(ctx, &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbARN),
	})
	if err != nil {
		return nil, err
	}

	listeners := make([]Listener, 0, len(out.Listeners))
	for _, l := range out.Listeners {
		listener := Listener{
			ARN:      aws.ToString(l.ListenerArn),
			Port:     int(aws.ToInt32(l.Port)),
			Protocol: string(l.Protocol),
		}

		// Count rules for this listener
		rulesOut, err := elbv2Client.DescribeRules(ctx, &elbv2.DescribeRulesInput{
			ListenerArn: l.ListenerArn,
		})
		if err == nil {
			// Subtract 1 for the default rule
			count := len(rulesOut.Rules)
			if count > 0 {
				count--
			}
			listener.Rules = count
		}

		listeners = append(listeners, listener)
	}
	return listeners, nil
}

// FetchTargetGroups returns target groups for a load balancer ARN (elbv2 only).
func FetchTargetGroups(ctx context.Context, elbv2Client *elbv2.Client, lbARN string) ([]TargetGroup, error) {
	out, err := elbv2Client.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: aws.String(lbARN),
	})
	if err != nil {
		return nil, err
	}

	tgs := make([]TargetGroup, 0, len(out.TargetGroups))
	for _, tg := range out.TargetGroups {
		port := 0
		if tg.Port != nil {
			port = int(aws.ToInt32(tg.Port))
		}

		healthSummary := ""
		thOut, thErr := elbv2Client.DescribeTargetHealth(ctx, &elbv2.DescribeTargetHealthInput{
			TargetGroupArn: tg.TargetGroupArn,
		})
		if thErr == nil {
			healthy := 0
			unhealthy := 0
			for _, desc := range thOut.TargetHealthDescriptions {
				if desc.TargetHealth != nil && string(desc.TargetHealth.State) == "healthy" {
					healthy++
				} else {
					unhealthy++
				}
			}
			if unhealthy > 0 {
				healthSummary = fmt.Sprintf("%d healthy, %d unhealthy", healthy, unhealthy)
			} else {
				healthSummary = fmt.Sprintf("%d healthy", healthy)
			}
		}

		entry := TargetGroup{
			Name:        aws.ToString(tg.TargetGroupName),
			ARN:         aws.ToString(tg.TargetGroupArn),
			Port:        port,
			Protocol:    string(tg.Protocol),
			TargetType:  string(tg.TargetType),
			HealthCheck: healthSummary,
			VpcID:       aws.ToString(tg.VpcId),
		}
		tgs = append(tgs, entry)
	}
	return tgs, nil
}
