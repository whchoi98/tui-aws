# CloudFront Tab
## Role
CloudFront Distributions — list distributions with origins, aliases, WAF, certificates.
## Key Files
- `model.go` — CFModel implementing TabModel
- `table.go` — Columns: ID, Domain, Status, Enabled, Origins, Aliases
- `detail.go` — Full distribution detail + origins + aliases + WAF + cert
## Rules
- Uses `cloudfront.Client` (ListDistributions)
- CloudFront is a global service but client uses configured region
