# CloudWatch Tab
## Role
CloudWatch Alarms — list alarms with state, metric, namespace, threshold.
## Key Files
- `model.go` — CWModel implementing TabModel
- `table.go` — State column: OK=green, ALARM=red, INSUFFICIENT_DATA=yellow
- `detail.go` — Full alarm detail + dimensions + actions
## Rules
- Uses `cloudwatch.Client` (DescribeAlarms)
- State colors match AWS Console convention
