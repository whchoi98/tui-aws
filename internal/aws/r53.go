package aws

import (
	"context"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// HostedZone represents a Route 53 hosted zone.
type HostedZone struct {
	ID          string
	Name        string
	IsPrivate   bool
	RecordCount int64
	Comment     string
	Records     []DNSRecord // loaded on demand
}

// DNSRecord represents a DNS record in a hosted zone.
type DNSRecord struct {
	Name        string
	Type        string
	Value       string
	TTL         int64
	AliasTarget string // for alias records
}

// FetchHostedZones returns all Route 53 hosted zones.
func FetchHostedZones(ctx context.Context, r53Client *route53.Client) ([]HostedZone, error) {
	var zones []HostedZone
	var marker *string

	for {
		out, err := r53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for _, z := range out.HostedZones {
			id := awssdk.ToString(z.Id)
			// Strip the /hostedzone/ prefix
			id = strings.TrimPrefix(id, "/hostedzone/")

			zone := HostedZone{
				ID:   id,
				Name: awssdk.ToString(z.Name),
			}

			if z.Config != nil {
				zone.IsPrivate = z.Config.PrivateZone
				zone.Comment = awssdk.ToString(z.Config.Comment)
			}

			if z.ResourceRecordSetCount != nil {
				zone.RecordCount = *z.ResourceRecordSetCount
			}

			zones = append(zones, zone)
		}

		if out.IsTruncated {
			marker = out.NextMarker
		} else {
			break
		}
	}

	return zones, nil
}

// FetchRecords returns DNS records for a hosted zone.
func FetchRecords(ctx context.Context, r53Client *route53.Client, zoneID string) ([]DNSRecord, error) {
	var records []DNSRecord
	var nextName *string
	var nextType types.RRType
	hasNextType := false
	isFirst := true

	for isFirst || nextName != nil {
		isFirst = false
		input := &route53.ListResourceRecordSetsInput{
			HostedZoneId: awssdk.String(zoneID),
		}
		if nextName != nil {
			input.StartRecordName = nextName
		}
		if hasNextType {
			input.StartRecordType = nextType
		}

		out, err := r53Client.ListResourceRecordSets(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, rrs := range out.ResourceRecordSets {
			rec := DNSRecord{
				Name: awssdk.ToString(rrs.Name),
				Type: string(rrs.Type),
			}

			if rrs.TTL != nil {
				rec.TTL = *rrs.TTL
			}

			if rrs.AliasTarget != nil {
				rec.AliasTarget = awssdk.ToString(rrs.AliasTarget.DNSName)
				rec.Value = "ALIAS -> " + rec.AliasTarget
			} else if len(rrs.ResourceRecords) > 0 {
				var values []string
				for _, r := range rrs.ResourceRecords {
					values = append(values, awssdk.ToString(r.Value))
				}
				rec.Value = strings.Join(values, ", ")
			}

			records = append(records, rec)
		}

		if out.IsTruncated {
			nextName = out.NextRecordName
			nextType = out.NextRecordType
			hasNextType = true
		} else {
			nextName = nil
		}
	}

	return records, nil
}
