package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Clients struct {
	EC2     *ec2.Client
	SSM     *ssm.Client
	STS     *sts.Client
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
