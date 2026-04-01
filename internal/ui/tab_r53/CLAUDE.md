# R53 Tab
## Role
Route 53 Hosted Zones — list zones, view DNS records on demand.
## Key Files
- `model.go` — R53Model with zone list + record detail (lazy loaded)
- `table.go` — Columns: Name, ID, Private, Records, Comment
- `detail.go` — Zone info + scrollable record list (Name, Type, TTL, Value/Alias)
## Rules
- Uses `route53.Client` (ListHostedZones, ListResourceRecordSets)
- Records loaded on demand when user opens zone detail
