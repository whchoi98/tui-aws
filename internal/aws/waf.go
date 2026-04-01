package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	"github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// WebACL represents a WAFv2 Web ACL.
type WebACL struct {
	Name                string
	ID                  string
	ARN                 string
	Scope               string // REGIONAL or CLOUDFRONT
	Rules               int    // rule count
	DefaultAction       string // Allow or Block
	Description         string
	AssociatedResources []string // ARNs of ALBs, API GWs, etc.
}

// FetchWebACLs returns all WAFv2 Web ACLs for the given scope (REGIONAL).
// Scope "CLOUDFRONT" only works in us-east-1.
func FetchWebACLs(ctx context.Context, wafClient *wafv2.Client, scope string) ([]WebACL, error) {
	var wafScope types.Scope
	switch scope {
	case "CLOUDFRONT":
		wafScope = types.ScopeCloudfront
	default:
		wafScope = types.ScopeRegional
	}

	var acls []WebACL
	var nextMarker *string

	for {
		out, err := wafClient.ListWebACLs(ctx, &wafv2.ListWebACLsInput{
			Scope:      wafScope,
			NextMarker: nextMarker,
		})
		if err != nil {
			return nil, err
		}

		for _, summary := range out.WebACLs {
			acl := WebACL{
				Name:        awssdk.ToString(summary.Name),
				ID:          awssdk.ToString(summary.Id),
				ARN:         awssdk.ToString(summary.ARN),
				Scope:       scope,
				Description: awssdk.ToString(summary.Description),
			}

			// Get full ACL details for rule count and default action
			getOut, getErr := wafClient.GetWebACL(ctx, &wafv2.GetWebACLInput{
				Name:  summary.Name,
				Scope: wafScope,
				Id:    summary.Id,
			})
			if getErr == nil && getOut.WebACL != nil {
				acl.Rules = len(getOut.WebACL.Rules)
				if getOut.WebACL.DefaultAction != nil {
					if getOut.WebACL.DefaultAction.Allow != nil {
						acl.DefaultAction = "Allow"
					} else if getOut.WebACL.DefaultAction.Block != nil {
						acl.DefaultAction = "Block"
					}
				}
			}

			// Get associated resources (only for REGIONAL scope)
			if wafScope == types.ScopeRegional {
				resOut, resErr := wafClient.ListResourcesForWebACL(ctx, &wafv2.ListResourcesForWebACLInput{
					WebACLArn: summary.ARN,
				})
				if resErr == nil {
					acl.AssociatedResources = resOut.ResourceArns
				}
			}

			acls = append(acls, acl)
		}

		if out.NextMarker != nil {
			nextMarker = out.NextMarker
		} else {
			break
		}
	}

	return acls, nil
}
