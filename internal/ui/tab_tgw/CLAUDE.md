# TGW Tab
## Role
Transit Gateways — list TGWs with attachments, route tables, and routes.
## Key Files
- `model.go` — TGWModel with table + detail (attachments + routes loaded on demand)
- `table.go` — Columns: Name, TGW ID, State, ASN, Attachments
- `detail.go` — Attachments list + route tables with route entries
## Rules
- Uses `ec2.Client` (DescribeTransitGateways, SearchTransitGatewayRoutes)
- Route tables fetched on demand when viewing detail
