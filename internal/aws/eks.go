package aws

import (
	"context"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
)

// EKSCluster represents an EKS cluster.
type EKSCluster struct {
	Name             string
	ARN              string
	Version          string
	Status           string
	Endpoint         string
	VpcID            string
	SubnetIDs        []string
	SecurityGroupIDs []string
	PlatformVersion  string
	CreatedTime      string
	NodeGroups       []EKSNodeGroup // loaded on demand
}

// EKSNodeGroup represents an EKS managed node group.
type EKSNodeGroup struct {
	Name          string
	Status        string
	InstanceTypes string
	DesiredSize   int
	MinSize       int
	MaxSize       int
	AmiType       string
}

// FetchEKSClusters retrieves all EKS clusters using ListClusters + DescribeCluster.
func FetchEKSClusters(ctx context.Context, client *eks.Client) ([]EKSCluster, error) {
	var clusterNames []string
	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		clusterNames = append(clusterNames, page.Clusters...)
	}

	var clusters []EKSCluster
	for _, name := range clusterNames {
		out, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
			Name: awssdk.String(name),
		})
		if err != nil {
			continue // skip clusters we can't describe
		}

		c := out.Cluster
		cluster := EKSCluster{
			Name:            awssdk.ToString(c.Name),
			ARN:             awssdk.ToString(c.Arn),
			Version:         awssdk.ToString(c.Version),
			Status:          string(c.Status),
			Endpoint:        awssdk.ToString(c.Endpoint),
			PlatformVersion: awssdk.ToString(c.PlatformVersion),
		}

		if c.ResourcesVpcConfig != nil {
			cluster.VpcID = awssdk.ToString(c.ResourcesVpcConfig.VpcId)
			cluster.SubnetIDs = c.ResourcesVpcConfig.SubnetIds
			cluster.SecurityGroupIDs = c.ResourcesVpcConfig.SecurityGroupIds
		}

		if c.CreatedAt != nil {
			cluster.CreatedTime = c.CreatedAt.Format("2006-01-02 15:04:05")
		}

		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

// FetchEKSNodeGroups retrieves node groups for a given EKS cluster.
func FetchEKSNodeGroups(ctx context.Context, client *eks.Client, clusterName string) ([]EKSNodeGroup, error) {
	var ngNames []string
	paginator := eks.NewListNodegroupsPaginator(client, &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		ngNames = append(ngNames, page.Nodegroups...)
	}

	var nodeGroups []EKSNodeGroup
	for _, ngName := range ngNames {
		out, err := client.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &ngName,
		})
		if err != nil {
			continue
		}

		ng := out.Nodegroup
		nodeGroup := EKSNodeGroup{
			Name:          awssdk.ToString(ng.NodegroupName),
			Status:        string(ng.Status),
			InstanceTypes: strings.Join(ng.InstanceTypes, ","),
			AmiType:       string(ng.AmiType),
		}

		if ng.ScalingConfig != nil {
			nodeGroup.DesiredSize = int(awssdk.ToInt32(ng.ScalingConfig.DesiredSize))
			nodeGroup.MinSize = int(awssdk.ToInt32(ng.ScalingConfig.MinSize))
			nodeGroup.MaxSize = int(awssdk.ToInt32(ng.ScalingConfig.MaxSize))
		}

		nodeGroups = append(nodeGroups, nodeGroup)
	}
	return nodeGroups, nil
}

// EKSSearchFields returns a lowercase concatenation of searchable fields.
func EKSSearchFields(c EKSCluster) string {
	return strings.ToLower(c.Name + " " + c.Version + " " + c.Status + " " + c.VpcID)
}
