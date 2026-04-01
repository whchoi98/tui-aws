# EKS Tab
## Role
EKS K8s integration — Clusters → Namespaces → Pods/Deployments/Services, Nodes, Pod Logs.
## Key Files
- `model.go` — 17-state viewState for K8s drill-down, breadcrumb status bar
- `table.go` — Per-level columns (cluster, namespace, pod, deploy, svc, node)
- `detail.go` — Pod/deploy/service/node detail + pod log viewer
## Rules
- K8s REST API via net/http (NO client-go dependency)
- Token via `aws eks get-token` with 14-min caching (internal/aws/k8s.go)
- TLS verified with cluster CA cert (base64 decoded from EKS API)
- Pod logs: /api/v1/namespaces/{ns}/pods/{name}/log?tailLines=50
