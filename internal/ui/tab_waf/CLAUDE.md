# WAF Tab
## Role
WAFv2 Web ACLs — list ACLs with rule count, default action, associated resources.
## Key Files
- `model.go` — WAFModel implementing TabModel
- `table.go` — Default action: Allow=green, Block=red
- `detail.go` — Full ACL detail + associated resource ARNs
## Rules
- Uses `wafv2.Client` (REGIONAL scope only — CLOUDFRONT scope requires us-east-1)
- Fetches rules count via GetWebACL, resources via ListResourcesForWebACL
