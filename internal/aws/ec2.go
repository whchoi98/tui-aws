package aws

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Instance struct {
	InstanceID       string
	Name             string
	State            string
	PrivateIP        string
	PublicIP         string
	InstanceType     string
	AvailabilityZone string
	Platform         string
	LaunchTime       time.Time
	SecurityGroups   []string
	KeyPair          string
	IAMRole          string
	SSMConnected     bool
}

func (i Instance) DisplayName() string {
	if i.Name != "" {
		return i.Name
	}
	return i.InstanceID
}

func (i Instance) StateIcon() string {
	switch i.State {
	case "running":
		return "●"
	case "stopped":
		return "○"
	case "pending":
		return "◐"
	case "stopping":
		return "◑"
	case "terminated":
		return "✕"
	default:
		return "?"
	}
}

func (i Instance) ShortAZ() string {
	parts := strings.Split(i.AvailabilityZone, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return i.AvailabilityZone
}

func (i Instance) LaunchTimeFormatted() string {
	if i.LaunchTime.IsZero() {
		return "-"
	}
	return i.LaunchTime.Format("2006-01-02 15:04")
}

func FetchInstances(ctx context.Context, client *ec2.Client) ([]Instance, error) {
	var instances []Instance
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				instances = append(instances, toInstance(inst))
			}
		}
	}
	return instances, nil
}

func toInstance(inst ec2types.Instance) Instance {
	i := Instance{
		InstanceID:       aws.ToString(inst.InstanceId),
		InstanceType:     string(inst.InstanceType),
		AvailabilityZone: aws.ToString(inst.Placement.AvailabilityZone),
		PrivateIP:        aws.ToString(inst.PrivateIpAddress),
		PublicIP:         aws.ToString(inst.PublicIpAddress),
		KeyPair:          aws.ToString(inst.KeyName),
	}

	if inst.State != nil {
		i.State = string(inst.State.Name)
	}

	if inst.LaunchTime != nil {
		i.LaunchTime = *inst.LaunchTime
	}

	if inst.PlatformDetails != nil {
		i.Platform = aws.ToString(inst.PlatformDetails)
	} else {
		i.Platform = "Linux"
	}

	for _, tag := range inst.Tags {
		if aws.ToString(tag.Key) == "Name" {
			i.Name = aws.ToString(tag.Value)
			break
		}
	}

	for _, sg := range inst.SecurityGroups {
		i.SecurityGroups = append(i.SecurityGroups, aws.ToString(sg.GroupName))
	}

	if inst.IamInstanceProfile != nil {
		arn := aws.ToString(inst.IamInstanceProfile.Arn)
		if parts := strings.Split(arn, "/"); len(parts) > 1 {
			i.IAMRole = parts[len(parts)-1]
		}
	}

	return i
}
