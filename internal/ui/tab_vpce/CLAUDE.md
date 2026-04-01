# VPCE Tab
## Role
VPC Endpoints — list endpoints with service name, type, state, and full details.
## Key Files
- `model.go` — VPCEModel implementing TabModel
- `table.go` — Columns: Name, Endpoint ID, Service Name, Type, VPC, State
- `detail.go` — Full endpoint detail: subnets, route tables, SGs, ENIs, private DNS, creation time
## Rules
- Uses existing `ec2.Client` (DescribeVpcEndpoints)
- Type: Gateway (route table based) vs Interface (ENI based)
