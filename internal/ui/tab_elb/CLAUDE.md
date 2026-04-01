# ELB Tab
## Role
Load Balancers (ALB/NLB/CLB) — list LBs with interactive target group detail.
## Key Files
- `model.go` — ELBModel with vsDetail (interactive TG cursor) and vsTGDetail states
- `table.go` — Type column: ALB=blue, NLB=green, GWLB=yellow, CLB=gray
- `detail.go` — Interactive detail: SGs, listeners, selectable TG list → target health
## Rules
- Uses `elbv2.Client` (ALB/NLB) and `elb.Client` (CLB) in parallel
- Target groups selectable with ↑↓, Enter shows registered targets with health
- Target health: healthy=green, unhealthy=red, draining/initial=yellow
