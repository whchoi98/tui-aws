# ASG Tab
## Role
Auto Scaling Groups — list groups with min/max/desired, instances, scaling policies, target groups.
## Key Files
- `model.go` — ASGModel implementing TabModel, table + action menu + detail
- `table.go` — Columns: Name, Min, Max, Desired, Instances, Health, AZs
- `detail.go` — Full ASG detail overlay
## Rules
- Uses `autoscaling.Client` (separate from EC2)
- Instance count shown as running/total
