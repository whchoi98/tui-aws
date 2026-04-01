# ECS Tab
## Role
ECS deep dive — Clusters → Services → Tasks → Containers → Logs → ECS Exec.
## Key Files
- `model.go` — 15-state viewState for drill-down hierarchy, ECSExecRequest/DoneMsg
- `table.go` — Per-level table columns (cluster, service, task, container)
- `detail.go` — Cluster/service/task/container/task-def detail + log viewer
## Rules
- ECS Exec sends ECSExecRequest to RootModel (similar to SSM flow)
- Container logs from CloudWatch Logs via CWL client
- Task definition loaded on demand (FetchTaskDefinition)
- Breadcrumb status bar tracks drill-down path
