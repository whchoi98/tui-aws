package aws

import (
	"context"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// LambdaFunction represents a Lambda function.
type LambdaFunction struct {
	Name             string
	ARN              string
	Runtime          string
	Handler          string
	MemorySize       int    // MB
	Timeout          int    // seconds
	CodeSize         int64  // bytes
	LastModified     string
	State            string // Active, Pending, Inactive, Failed
	VpcID            string
	SubnetIDs        []string
	SecurityGroupIDs []string
	Layers           []string
	Description      string
}

// FetchLambdaFunctions retrieves all Lambda functions using paginated ListFunctions.
func FetchLambdaFunctions(ctx context.Context, client *lambda.Client) ([]LambdaFunction, error) {
	var functions []LambdaFunction
	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fn := range page.Functions {
			f := LambdaFunction{
				Name:         awssdk.ToString(fn.FunctionName),
				ARN:          awssdk.ToString(fn.FunctionArn),
				Runtime:      string(fn.Runtime),
				Handler:      awssdk.ToString(fn.Handler),
				MemorySize:   int(awssdk.ToInt32(fn.MemorySize)),
				Timeout:      int(awssdk.ToInt32(fn.Timeout)),
				CodeSize:     fn.CodeSize,
				LastModified: awssdk.ToString(fn.LastModified),
				State:        string(fn.State),
				Description:  awssdk.ToString(fn.Description),
			}

			if fn.VpcConfig != nil {
				f.VpcID = awssdk.ToString(fn.VpcConfig.VpcId)
				f.SubnetIDs = fn.VpcConfig.SubnetIds
				f.SecurityGroupIDs = fn.VpcConfig.SecurityGroupIds
			}

			for _, layer := range fn.Layers {
				f.Layers = append(f.Layers, awssdk.ToString(layer.Arn))
			}

			functions = append(functions, f)
		}
	}
	return functions, nil
}

// LambdaSearchFields returns a lowercase concatenation of searchable fields.
func LambdaSearchFields(f LambdaFunction) string {
	return strings.ToLower(f.Name + " " + f.Runtime + " " + f.State + " " + f.VpcID + " " + f.Description)
}
