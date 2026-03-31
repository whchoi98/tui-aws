package aws

import (
	"context"
	"fmt"
	"os/exec"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type PrereqResult struct {
	Name    string
	OK      bool
	Message string
}

func CheckPrerequisites() []PrereqResult {
	var results []PrereqResult

	if _, err := exec.LookPath("aws"); err != nil {
		results = append(results, PrereqResult{
			Name:    "AWS CLI",
			OK:      false,
			Message: "aws CLI not found. Install: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
		})
	} else {
		results = append(results, PrereqResult{Name: "AWS CLI", OK: true, Message: "OK"})
	}

	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		results = append(results, PrereqResult{
			Name:    "Session Manager Plugin",
			OK:      false,
			Message: "session-manager-plugin not found. Install: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
		})
	} else {
		results = append(results, PrereqResult{Name: "Session Manager Plugin", OK: true, Message: "OK"})
	}

	return results
}

func needsProfileFlag(profile string) bool {
	return profile != "" && profile != "default" && profile != InstanceRoleProfile
}

func BuildSSMSessionArgs(instanceID, profile, region string) []string {
	args := []string{"ssm", "start-session", "--target", instanceID}
	if needsProfileFlag(profile) {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--region", region)
	return args
}

func BuildPortForwardArgs(instanceID, profile, region, localPort, remotePort string) []string {
	args := []string{
		"ssm", "start-session",
		"--target", instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf("portNumber=%s,localPortNumber=%s", remotePort, localPort),
	}
	if needsProfileFlag(profile) {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--region", region)
	return args
}

func FetchSSMStatus(ctx context.Context, client *ssm.Client) (map[string]bool, error) {
	status := make(map[string]bool)
	paginator := ssm.NewDescribeInstanceInformationPaginator(client, &ssm.DescribeInstanceInformationInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, info := range page.InstanceInformationList {
			id := awssdk.ToString(info.InstanceId)
			status[id] = string(info.PingStatus) == "Online"
		}
	}
	return status, nil
}
