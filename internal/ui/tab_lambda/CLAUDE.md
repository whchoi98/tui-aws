# Lambda Tab
## Role
Lambda Functions — list functions with runtime, memory, timeout, VPC config, layers.
## Key Files
- `model.go` — LambdaModel implementing TabModel
- `table.go` — State: Active=green, Pending=yellow, Inactive=gray, Failed=red
- `detail.go` — Full function detail + layers + VPC config (subnets/SGs)
## Rules
- Uses `lambda.Client` (ListFunctions)
