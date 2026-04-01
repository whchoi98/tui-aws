# ACM Tab
## Role
ACM Certificates — list certs with domain, status, expiry, SANs, in-use resources.
## Key Files
- `model.go` — ACMModel implementing TabModel
- `table.go` — Status: ISSUED=green, PENDING=yellow, EXPIRED/REVOKED/FAILED=red
- `detail.go` — Full cert detail + SANs list + InUseBy list
## Rules
- Uses `acm.Client` (ListCertificates + DescribeCertificate per cert)
- Expiry date highlighted when approaching
