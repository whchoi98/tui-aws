package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
)

// Distribution represents a CloudFront distribution.
type Distribution struct {
	ID             string
	ARN            string
	DomainName     string
	Status         string
	Comment        string
	Enabled        bool
	Origins        []string // origin domain names
	Aliases        []string // CNAMEs
	PriceClass     string
	WebACLID       string
	CertificateARN string
	LastModified   string
}

// FetchDistributions returns all CloudFront distributions via ListDistributions.
func FetchDistributions(ctx context.Context, cfClient *cloudfront.Client) ([]Distribution, error) {
	var distributions []Distribution
	var marker *string

	for {
		out, err := cfClient.ListDistributions(ctx, &cloudfront.ListDistributionsInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		if out.DistributionList == nil {
			break
		}

		for _, d := range out.DistributionList.Items {
			dist := Distribution{
				ID:         awssdk.ToString(d.Id),
				ARN:        awssdk.ToString(d.ARN),
				DomainName: awssdk.ToString(d.DomainName),
				Status:     awssdk.ToString(d.Status),
				Comment:    awssdk.ToString(d.Comment),
				Enabled:    d.Enabled != nil && *d.Enabled,
			}

			if d.Origins != nil {
				for _, o := range d.Origins.Items {
					dist.Origins = append(dist.Origins, awssdk.ToString(o.DomainName))
				}
			}

			if d.Aliases != nil {
				for _, a := range d.Aliases.Items {
					dist.Aliases = append(dist.Aliases, a)
				}
			}

			if d.PriceClass != "" {
				dist.PriceClass = string(d.PriceClass)
			}

			dist.WebACLID = awssdk.ToString(d.WebACLId)

			if d.ViewerCertificate != nil && d.ViewerCertificate.ACMCertificateArn != nil {
				dist.CertificateARN = awssdk.ToString(d.ViewerCertificate.ACMCertificateArn)
			}

			if d.LastModifiedTime != nil {
				dist.LastModified = d.LastModifiedTime.Format("2006-01-02 15:04")
			}

			distributions = append(distributions, dist)
		}

		if out.DistributionList.IsTruncated != nil && *out.DistributionList.IsTruncated && out.DistributionList.NextMarker != nil {
			marker = out.DistributionList.NextMarker
		} else {
			break
		}
	}

	return distributions, nil
}
