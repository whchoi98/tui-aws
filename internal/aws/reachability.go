package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ReachabilityResult holds the result of an AWS Reachability Analyzer analysis.
type ReachabilityResult struct {
	PathID       string
	AnalysisID   string
	Reachable    bool
	Explanations []string
}

// RunReachabilityAnalysis creates a network insights path and runs analysis.
// This is a paid AWS API call.
func RunReachabilityAnalysis(ctx context.Context, client *ec2.Client, sourceID, destID, protocol string, port int) (*ReachabilityResult, error) {
	// Map protocol string to EC2 Protocol type
	proto := ec2types.ProtocolTcp
	switch protocol {
	case "udp":
		proto = ec2types.ProtocolUdp
	case "tcp":
		proto = ec2types.ProtocolTcp
	}

	// Step 1: Create Network Insights Path
	pathInput := &ec2.CreateNetworkInsightsPathInput{
		Source:      aws.String(sourceID),
		Destination: aws.String(destID),
		Protocol:    proto,
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeNetworkInsightsPath,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String("tui-aws-check")},
					{Key: aws.String("CreatedBy"), Value: aws.String("tui-aws")},
				},
			},
		},
	}

	if port > 0 {
		pathInput.DestinationPort = aws.Int32(int32(port))
	}

	pathOut, err := client.CreateNetworkInsightsPath(ctx, pathInput)
	if err != nil {
		return nil, fmt.Errorf("create network insights path: %w", err)
	}

	pathID := aws.ToString(pathOut.NetworkInsightsPath.NetworkInsightsPathId)

	// Ensure cleanup
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.DeleteNetworkInsightsPath(cleanupCtx, &ec2.DeleteNetworkInsightsPathInput{
			NetworkInsightsPathId: aws.String(pathID),
		})
	}()

	// Step 2: Start Analysis
	analysisOut, err := client.StartNetworkInsightsAnalysis(ctx, &ec2.StartNetworkInsightsAnalysisInput{
		NetworkInsightsPathId: aws.String(pathID),
	})
	if err != nil {
		return nil, fmt.Errorf("start network insights analysis: %w", err)
	}

	analysisID := aws.ToString(analysisOut.NetworkInsightsAnalysis.NetworkInsightsAnalysisId)

	// Step 3: Poll for completion
	result := &ReachabilityResult{
		PathID:     pathID,
		AnalysisID: analysisID,
	}

	maxAttempts := 15 // 15 * 2s = 30s max
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(2 * time.Second):
		}

		descOut, err := client.DescribeNetworkInsightsAnalyses(ctx, &ec2.DescribeNetworkInsightsAnalysesInput{
			NetworkInsightsAnalysisIds: []string{analysisID},
		})
		if err != nil {
			return result, fmt.Errorf("describe analysis: %w", err)
		}

		if len(descOut.NetworkInsightsAnalyses) == 0 {
			continue
		}

		analysis := descOut.NetworkInsightsAnalyses[0]
		status := analysis.Status

		if status == ec2types.AnalysisStatusSucceeded {
			result.Reachable = aws.ToBool(analysis.NetworkPathFound)

			for _, exp := range analysis.Explanations {
				explanation := ""
				if exp.Direction != nil {
					explanation += string(*exp.Direction) + ": "
				}
				if exp.ExplanationCode != nil {
					explanation += *exp.ExplanationCode
				}
				if exp.Component != nil && exp.Component.Id != nil {
					explanation += fmt.Sprintf(" (%s)", aws.ToString(exp.Component.Id))
				}
				if explanation != "" {
					result.Explanations = append(result.Explanations, explanation)
				}
			}

			return result, nil
		}

		if status == ec2types.AnalysisStatusFailed {
			statusMsg := aws.ToString(analysis.StatusMessage)
			return result, fmt.Errorf("analysis failed: %s", statusMsg)
		}
	}

	return result, fmt.Errorf("analysis timed out after %d seconds", maxAttempts*2)
}
