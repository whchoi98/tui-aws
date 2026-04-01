package aws

import (
	"context"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
)

// Certificate represents an ACM certificate.
type Certificate struct {
	ARN                     string
	DomainName              string
	Status                  string
	Type                    string
	SubjectAlternativeNames []string
	Issuer                  string
	NotBefore               string
	NotAfter                string
	InUseBy                 []string // resource ARNs using this cert
	RenewalStatus           string
	KeyAlgorithm            string
}

// FetchCertificates returns all ACM certificates via ListCertificates + DescribeCertificate.
func FetchCertificates(ctx context.Context, acmClient *acm.Client) ([]Certificate, error) {
	var certs []Certificate

	paginator := acm.NewListCertificatesPaginator(acmClient, &acm.ListCertificatesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, summary := range page.CertificateSummaryList {
			arn := awssdk.ToString(summary.CertificateArn)

			cert := Certificate{
				ARN:        arn,
				DomainName: awssdk.ToString(summary.DomainName),
				Status:     string(summary.Status),
				Type:       string(summary.Type),
			}

			// Get full details
			descOut, descErr := acmClient.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
				CertificateArn: summary.CertificateArn,
			})
			if descErr == nil && descOut.Certificate != nil {
				d := descOut.Certificate
				cert.SubjectAlternativeNames = d.SubjectAlternativeNames
				cert.Issuer = awssdk.ToString(d.Issuer)
				cert.InUseBy = d.InUseBy
				cert.KeyAlgorithm = string(d.KeyAlgorithm)

				if d.NotBefore != nil {
					cert.NotBefore = d.NotBefore.Format("2006-01-02 15:04")
				}
				if d.NotAfter != nil {
					cert.NotAfter = d.NotAfter.Format("2006-01-02 15:04")
				}
				if d.RenewalSummary != nil {
					cert.RenewalStatus = string(d.RenewalSummary.RenewalStatus)
				}
			}

			certs = append(certs, cert)
		}
	}

	return certs, nil
}
