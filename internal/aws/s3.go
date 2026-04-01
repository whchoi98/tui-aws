package aws

import (
	"context"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Bucket represents an S3 bucket.
type Bucket struct {
	Name         string
	Region       string
	CreationDate string
	// Populated on demand:
	Versioning  string // Enabled, Suspended, or empty
	Encryption  string // AES256, aws:kms, or none
	PublicAccess string // Blocked or Open
}

// FetchBuckets retrieves all S3 buckets (global service).
func FetchBuckets(ctx context.Context, client *s3.Client) ([]Bucket, error) {
	out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var buckets []Bucket
	for _, b := range out.Buckets {
		bucket := Bucket{
			Name: awssdk.ToString(b.Name),
		}
		if b.CreationDate != nil {
			bucket.CreationDate = b.CreationDate.Format("2006-01-02 15:04:05")
		}
		// Try to get bucket region
		locOut, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: b.Name,
		})
		if err == nil {
			loc := string(locOut.LocationConstraint)
			if loc == "" {
				loc = "us-east-1" // default when LocationConstraint is empty
			}
			bucket.Region = loc
		}

		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

// FetchBucketDetails populates versioning, encryption, and public access for a bucket.
func FetchBucketDetails(ctx context.Context, client *s3.Client, bucketName string) (Bucket, error) {
	bucket := Bucket{Name: bucketName}

	// Versioning
	vOut, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: &bucketName,
	})
	if err == nil {
		switch vOut.Status {
		case s3types.BucketVersioningStatusEnabled:
			bucket.Versioning = "Enabled"
		case s3types.BucketVersioningStatusSuspended:
			bucket.Versioning = "Suspended"
		default:
			bucket.Versioning = "-"
		}
	}

	// Encryption
	eOut, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: &bucketName,
	})
	if err == nil && eOut.ServerSideEncryptionConfiguration != nil {
		for _, rule := range eOut.ServerSideEncryptionConfiguration.Rules {
			if rule.ApplyServerSideEncryptionByDefault != nil {
				switch rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm {
				case s3types.ServerSideEncryptionAes256:
					bucket.Encryption = "AES256"
				case s3types.ServerSideEncryptionAwsKms:
					bucket.Encryption = "aws:kms"
				default:
					bucket.Encryption = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				}
				break
			}
		}
	}
	if bucket.Encryption == "" {
		bucket.Encryption = "none"
	}

	// Public access
	paOut, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: &bucketName,
	})
	if err == nil && paOut.PublicAccessBlockConfiguration != nil {
		pa := paOut.PublicAccessBlockConfiguration
		if awssdk.ToBool(pa.BlockPublicAcls) &&
			awssdk.ToBool(pa.BlockPublicPolicy) &&
			awssdk.ToBool(pa.IgnorePublicAcls) &&
			awssdk.ToBool(pa.RestrictPublicBuckets) {
			bucket.PublicAccess = "Blocked"
		} else {
			bucket.PublicAccess = "Open"
		}
	} else {
		bucket.PublicAccess = "Open"
	}

	return bucket, nil
}

// S3SearchFields returns a lowercase concatenation of searchable fields.
func S3SearchFields(b Bucket) string {
	return strings.ToLower(b.Name + " " + b.Region + " " + b.Versioning + " " + b.Encryption)
}
