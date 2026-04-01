package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
)

type Clients struct {
	EC2     *ec2.Client
	SSM     *ssm.Client
	STS     *sts.Client
	ELBv2   *elbv2.Client // ALB, NLB, GWLB
	ELB     *elb.Client   // Classic LB
	ASG     *autoscaling.Client
	CW      *cloudwatch.Client
	IAM     *iam.Client
	CF      *cloudfront.Client
	WAF     *wafv2.Client
	ACM     *acm.Client
	R53     *route53.Client
	Profile string
	Region  string
}

func NewClients(ctx context.Context, profile, region string) (*Clients, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	// Instance role and "default" use the default credential chain (no explicit profile)
	if profile != "" && profile != "default" && profile != InstanceRoleProfile {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Clients{
		EC2:     ec2.NewFromConfig(cfg),
		SSM:     ssm.NewFromConfig(cfg),
		STS:     sts.NewFromConfig(cfg),
		ELBv2:   elbv2.NewFromConfig(cfg),
		ELB:     elb.NewFromConfig(cfg),
		ASG:     autoscaling.NewFromConfig(cfg),
		CW:      cloudwatch.NewFromConfig(cfg),
		IAM:     iam.NewFromConfig(cfg),
		CF:      cloudfront.NewFromConfig(cfg),
		WAF:     wafv2.NewFromConfig(cfg),
		ACM:     acm.NewFromConfig(cfg),
		R53:     route53.NewFromConfig(cfg),
		Profile: profile,
		Region:  region,
	}, nil
}

func (c *Clients) ValidateCredentials(ctx context.Context) (string, error) {
	out, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Account), nil
}

func KnownRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
		"ap-southeast-1", "ap-southeast-2",
		"ap-south-1",
		"eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "eu-north-1",
		"sa-east-1",
		"ca-central-1",
		"me-south-1",
		"af-south-1",
	}
}
