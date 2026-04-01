package aws

import (
	"context"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// ECSCluster represents an ECS cluster.
type ECSCluster struct {
	Name              string
	ARN               string
	Status            string
	RunningTasks      int
	PendingTasks      int
	Services          int
	Instances         int
	CapacityProviders []string
}

// ECSService represents an ECS service.
type ECSService struct {
	Name           string
	ARN            string
	Status         string
	ClusterARN     string
	DesiredCount   int
	RunningCount   int
	PendingCount   int
	LaunchType     string // FARGATE, EC2
	TaskDefinition string
}

// FetchECSClusters retrieves all ECS clusters using ListClusters + DescribeClusters.
func FetchECSClusters(ctx context.Context, client *ecs.Client) ([]ECSCluster, error) {
	// Collect all cluster ARNs
	var clusterARNs []string
	paginator := ecs.NewListClustersPaginator(client, &ecs.ListClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		clusterARNs = append(clusterARNs, page.ClusterArns...)
	}

	if len(clusterARNs) == 0 {
		return nil, nil
	}

	// DescribeClusters accepts up to 100 ARNs at a time
	var clusters []ECSCluster
	for i := 0; i < len(clusterARNs); i += 100 {
		end := i + 100
		if end > len(clusterARNs) {
			end = len(clusterARNs)
		}
		batch := clusterARNs[i:end]

		out, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: batch,
		})
		if err != nil {
			return nil, err
		}

		for _, c := range out.Clusters {
			cluster := ECSCluster{
				Name:         awssdk.ToString(c.ClusterName),
				ARN:          awssdk.ToString(c.ClusterArn),
				Status:       awssdk.ToString(c.Status),
				RunningTasks: int(c.RunningTasksCount),
				PendingTasks: int(c.PendingTasksCount),
				Services:     int(c.ActiveServicesCount),
				Instances:    int(c.RegisteredContainerInstancesCount),
			}
			for _, cp := range c.CapacityProviders {
				cluster.CapacityProviders = append(cluster.CapacityProviders, cp)
			}
			clusters = append(clusters, cluster)
		}
	}
	return clusters, nil
}

// FetchECSServices retrieves services for a given ECS cluster.
func FetchECSServices(ctx context.Context, client *ecs.Client, clusterARN string) ([]ECSService, error) {
	var serviceARNs []string
	paginator := ecs.NewListServicesPaginator(client, &ecs.ListServicesInput{
		Cluster: &clusterARN,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		serviceARNs = append(serviceARNs, page.ServiceArns...)
	}

	if len(serviceARNs) == 0 {
		return nil, nil
	}

	var services []ECSService
	// DescribeServices accepts up to 10 at a time
	for i := 0; i < len(serviceARNs); i += 10 {
		end := i + 10
		if end > len(serviceARNs) {
			end = len(serviceARNs)
		}
		batch := serviceARNs[i:end]

		out, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  &clusterARN,
			Services: batch,
		})
		if err != nil {
			return nil, err
		}

		for _, s := range out.Services {
			svc := ECSService{
				Name:           awssdk.ToString(s.ServiceName),
				ARN:            awssdk.ToString(s.ServiceArn),
				Status:         awssdk.ToString(s.Status),
				ClusterARN:     awssdk.ToString(s.ClusterArn),
				DesiredCount:   int(s.DesiredCount),
				RunningCount:   int(s.RunningCount),
				PendingCount:   int(s.PendingCount),
				LaunchType:     string(s.LaunchType),
				TaskDefinition: awssdk.ToString(s.TaskDefinition),
			}
			services = append(services, svc)
		}
	}
	return services, nil
}

// ECSSearchFields returns a lowercase concatenation of searchable fields.
func ECSSearchFields(c ECSCluster) string {
	return strings.ToLower(c.Name + " " + c.ARN + " " + c.Status + " " + strings.Join(c.CapacityProviders, " "))
}
