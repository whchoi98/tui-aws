package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
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

// ECSTask represents a running or stopped ECS task.
type ECSTask struct {
	TaskARN           string
	TaskDefinitionARN string
	ClusterARN        string
	LastStatus        string // RUNNING, STOPPED, etc.
	DesiredStatus     string
	LaunchType        string // FARGATE, EC2
	CPU               string // "256", "512"
	Memory            string
	StartedAt         string
	StoppedAt         string
	StoppedReason     string
	Group             string // service:<name>
	Containers        []ECSContainer
	ConnectivityStatus string
	HealthStatus      string
}

// ShortTaskID returns the last segment of the task ARN (the task ID).
func (t ECSTask) ShortTaskID() string {
	parts := strings.Split(t.TaskARN, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return t.TaskARN
}

// ECSContainer represents a container within a task.
type ECSContainer struct {
	Name         string
	Image        string
	Status       string
	ContainerARN string
	RuntimeID    string // docker container ID
	ExitCode     *int32
	Reason       string
	CPU          int // hard limit from task definition
	Memory       int
	Ports        []string // "0.0.0.0:80->80/tcp"
	HealthStatus string
	LogGroup     string
	LogStream    string
}

// ECSContainerDef represents a container definition from a task definition.
type ECSContainerDef struct {
	Name            string
	Image           string
	CPU             int
	Memory          int
	Essential       bool
	PortMappings    []string
	Environment     map[string]string
	LogGroup        string
	LogStreamPrefix string
}

// LogEvent represents a single CloudWatch log event.
type LogEvent struct {
	Timestamp string
	Message   string
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

// FetchECSTasks returns all tasks in a cluster, optionally filtered by service name.
func FetchECSTasks(ctx context.Context, client *ecs.Client, clusterARN string, serviceName string) ([]ECSTask, error) {
	input := &ecs.ListTasksInput{
		Cluster: &clusterARN,
	}
	if serviceName != "" {
		input.ServiceName = &serviceName
	}

	var taskARNs []string
	paginator := ecs.NewListTasksPaginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		taskARNs = append(taskARNs, page.TaskArns...)
	}

	if len(taskARNs) == 0 {
		return nil, nil
	}

	var tasks []ECSTask
	// DescribeTasks accepts up to 100 at a time
	for i := 0; i < len(taskARNs); i += 100 {
		end := i + 100
		if end > len(taskARNs) {
			end = len(taskARNs)
		}
		batch := taskARNs[i:end]

		out, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: &clusterARN,
			Tasks:   batch,
		})
		if err != nil {
			return nil, err
		}

		for _, t := range out.Tasks {
			task := ECSTask{
				TaskARN:           awssdk.ToString(t.TaskArn),
				TaskDefinitionARN: awssdk.ToString(t.TaskDefinitionArn),
				ClusterARN:        awssdk.ToString(t.ClusterArn),
				LastStatus:        awssdk.ToString(t.LastStatus),
				DesiredStatus:     awssdk.ToString(t.DesiredStatus),
				LaunchType:        string(t.LaunchType),
				CPU:               awssdk.ToString(t.Cpu),
				Memory:            awssdk.ToString(t.Memory),
				StoppedReason:     awssdk.ToString(t.StoppedReason),
				Group:             awssdk.ToString(t.Group),
				HealthStatus:      string(t.HealthStatus),
			}
			if t.Connectivity != "" {
				task.ConnectivityStatus = string(t.Connectivity)
			}
			if t.StartedAt != nil {
				task.StartedAt = t.StartedAt.Format(time.RFC3339)
			}
			if t.StoppedAt != nil {
				task.StoppedAt = t.StoppedAt.Format(time.RFC3339)
			}

			for _, c := range t.Containers {
				container := ECSContainer{
					Name:         awssdk.ToString(c.Name),
					Image:        awssdk.ToString(c.Image),
					Status:       awssdk.ToString(c.LastStatus),
					ContainerARN: awssdk.ToString(c.ContainerArn),
					RuntimeID:    awssdk.ToString(c.RuntimeId),
					HealthStatus: string(c.HealthStatus),
					ExitCode:     c.ExitCode,
					Reason:       awssdk.ToString(c.Reason),
				}
				for _, nb := range c.NetworkBindings {
					host := awssdk.ToString(nb.BindIP)
					if host == "" {
						host = "0.0.0.0"
					}
					container.Ports = append(container.Ports,
						fmt.Sprintf("%s:%d->%d/%s", host, awssdk.ToInt32(nb.HostPort), nb.ContainerPort, string(nb.Protocol)))
				}
				task.Containers = append(task.Containers, container)
			}

			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

// FetchTaskDefinition returns container definitions for a task definition ARN.
func FetchTaskDefinition(ctx context.Context, client *ecs.Client, taskDefARN string) ([]ECSContainerDef, error) {
	out, err := client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefARN,
	})
	if err != nil {
		return nil, err
	}

	var defs []ECSContainerDef
	for _, cd := range out.TaskDefinition.ContainerDefinitions {
		def := ECSContainerDef{
			Name:      awssdk.ToString(cd.Name),
			Image:     awssdk.ToString(cd.Image),
			CPU:       int(cd.Cpu),
			Memory:    int(awssdk.ToInt32(cd.Memory)),
			Essential: awssdk.ToBool(cd.Essential),
			Environment: make(map[string]string),
		}
		for _, pm := range cd.PortMappings {
			proto := string(pm.Protocol)
			if proto == "" {
				proto = "tcp"
			}
			def.PortMappings = append(def.PortMappings,
				fmt.Sprintf("%d->%d/%s", awssdk.ToInt32(pm.HostPort), pm.ContainerPort, proto))
		}
		for _, env := range cd.Environment {
			def.Environment[awssdk.ToString(env.Name)] = awssdk.ToString(env.Value)
		}
		if cd.LogConfiguration != nil && cd.LogConfiguration.LogDriver == ecstypes.LogDriverAwslogs {
			opts := cd.LogConfiguration.Options
			def.LogGroup = opts["awslogs-group"]
			def.LogStreamPrefix = opts["awslogs-stream-prefix"]
		}
		defs = append(defs, def)
	}
	return defs, nil
}

// EnrichContainerLogs populates LogGroup and LogStream on containers using the task definition.
// The awslogs stream format is: {prefix}/{container-name}/{task-id}
func EnrichContainerLogs(containers []ECSContainer, defs []ECSContainerDef, taskID string) []ECSContainer {
	defMap := make(map[string]ECSContainerDef, len(defs))
	for _, d := range defs {
		defMap[d.Name] = d
	}
	result := make([]ECSContainer, len(containers))
	copy(result, containers)
	for i, c := range result {
		if d, ok := defMap[c.Name]; ok {
			result[i].LogGroup = d.LogGroup
			if d.LogStreamPrefix != "" {
				result[i].LogStream = fmt.Sprintf("%s/%s/%s", d.LogStreamPrefix, c.Name, taskID)
			}
			if d.CPU > 0 {
				result[i].CPU = d.CPU
			}
			if d.Memory > 0 {
				result[i].Memory = d.Memory
			}
		}
	}
	return result
}

// FetchContainerLogs returns recent log events from CloudWatch Logs.
func FetchContainerLogs(ctx context.Context, cwlClient *cwl.Client, logGroup, logStream string, limit int) ([]LogEvent, error) {
	if logGroup == "" || logStream == "" {
		return nil, nil
	}
	lim := int32(limit)
	out, err := cwlClient.GetLogEvents(ctx, &cwl.GetLogEventsInput{
		LogGroupName:  &logGroup,
		LogStreamName: &logStream,
		Limit:         &lim,
		StartFromHead: awssdk.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	var events []LogEvent
	for _, ev := range out.Events {
		ts := ""
		if ev.Timestamp != nil {
			ts = time.UnixMilli(*ev.Timestamp).Format("2006-01-02 15:04:05")
		}
		events = append(events, LogEvent{
			Timestamp: ts,
			Message:   awssdk.ToString(ev.Message),
		})
	}
	return events, nil
}

// BuildECSExecArgs constructs arguments for `aws ecs execute-command`.
func BuildECSExecArgs(clusterARN, taskARN, containerName, profile, region string) []string {
	args := []string{
		"ecs", "execute-command",
		"--cluster", clusterARN,
		"--task", taskARN,
		"--container", containerName,
		"--interactive",
		"--command", "/bin/sh",
	}
	if profile != "" && profile != "default" && profile != InstanceRoleProfile {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--region", region)
	return args
}

// ECSSearchFields returns a lowercase concatenation of searchable fields.
func ECSSearchFields(c ECSCluster) string {
	return strings.ToLower(c.Name + " " + c.ARN + " " + c.Status + " " + strings.Join(c.CapacityProviders, " "))
}
