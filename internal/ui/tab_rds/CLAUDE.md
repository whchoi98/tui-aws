# RDS Tab
## Role
RDS DB Instances — list instances with engine, class, endpoint, multi-AZ, storage.
## Key Files
- `model.go` — RDSModel implementing TabModel
- `table.go` — Status: available=green, creating/modifying=yellow, deleting/failed=red
- `detail.go` — Full DB detail + endpoint + SG list + subnet group
## Rules
- Uses `rds.Client` (DescribeDBInstances)
