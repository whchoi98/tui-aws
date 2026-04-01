# S3 Tab
## Role
S3 Buckets — list all buckets (global), view versioning/encryption/public access on demand.
## Key Files
- `model.go` — S3Model implementing TabModel, detail loads on demand
- `table.go` — Columns: Name, Region, Created, Versioning, Encryption, Public
- `detail.go` — Bucket detail with versioning/encryption/public access
## Rules
- Uses `s3.Client` (ListBuckets is global — returns all regions)
- Detail fetches GetBucketVersioning, GetBucketEncryption, GetPublicAccessBlock
