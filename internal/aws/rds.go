package aws

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
)

// DBInstance represents an RDS DB instance.
type DBInstance struct {
	ID                 string
	ARN                string
	Engine             string
	EngineVersion      string
	Class              string
	Status             string // available, creating, deleting, etc.
	Endpoint           string // hostname:port
	Port               int
	MultiAZ            bool
	StorageType        string
	AllocatedStorage   int // GiB
	VpcID              string
	SubnetGroup        string
	SecurityGroups     []string
	AZ                 string
	PubliclyAccessible bool
	Encrypted          bool
	CreatedTime        string
}

// FetchDBInstances retrieves all RDS DB instances using paginated DescribeDBInstances.
func FetchDBInstances(ctx context.Context, client *rds.Client) ([]DBInstance, error) {
	var instances []DBInstance
	paginator := rds.NewDescribeDBInstancesPaginator(client, &rds.DescribeDBInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, db := range page.DBInstances {
			inst := DBInstance{
				ID:                 awssdk.ToString(db.DBInstanceIdentifier),
				ARN:                awssdk.ToString(db.DBInstanceArn),
				Engine:             awssdk.ToString(db.Engine),
				EngineVersion:      awssdk.ToString(db.EngineVersion),
				Class:              awssdk.ToString(db.DBInstanceClass),
				Status:             awssdk.ToString(db.DBInstanceStatus),
				MultiAZ:            awssdk.ToBool(db.MultiAZ),
				StorageType:        awssdk.ToString(db.StorageType),
				AllocatedStorage:   int(awssdk.ToInt32(db.AllocatedStorage)),
				PubliclyAccessible: awssdk.ToBool(db.PubliclyAccessible),
				Encrypted:          awssdk.ToBool(db.StorageEncrypted),
			}

			if db.Endpoint != nil {
				host := awssdk.ToString(db.Endpoint.Address)
				port := int(awssdk.ToInt32(db.Endpoint.Port))
				inst.Endpoint = fmt.Sprintf("%s:%d", host, port)
				inst.Port = port
			}

			if db.AvailabilityZone != nil {
				inst.AZ = awssdk.ToString(db.AvailabilityZone)
			}

			if db.DBSubnetGroup != nil {
				inst.SubnetGroup = awssdk.ToString(db.DBSubnetGroup.DBSubnetGroupName)
				inst.VpcID = awssdk.ToString(db.DBSubnetGroup.VpcId)
			}

			for _, sg := range db.VpcSecurityGroups {
				inst.SecurityGroups = append(inst.SecurityGroups, awssdk.ToString(sg.VpcSecurityGroupId))
			}

			if db.InstanceCreateTime != nil {
				inst.CreatedTime = db.InstanceCreateTime.Format("2006-01-02 15:04:05")
			}

			instances = append(instances, inst)
		}
	}
	return instances, nil
}

// RDSSearchFields returns a lowercase concatenation of searchable fields.
func RDSSearchFields(inst DBInstance) string {
	return strings.ToLower(inst.ID + " " + inst.Engine + " " + inst.Class + " " + inst.Status + " " + inst.VpcID)
}
