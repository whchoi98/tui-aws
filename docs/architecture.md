# Architecture

## System Overview
tui-aws is a single-binary Go CLI providing a 22-tab terminal UI for comprehensive AWS infrastructure management. Built on Bubble Tea v2 (Elm architecture: Model-View-Update) with a tab-based architecture separating concerns into independent submodels. 104 source files, ~23,000 lines of code.

## Tab Architecture

```
RootModel (tea.Model)
├── SharedState (profile, region, clients, cache, dimensions)
├── Tab Bar ([/] switching, active highlight)
├── Global Overlays (profile/region selector)
└── TabModel[] (each implements Init/Update/View/ShortHelp)
    │
    │  ── Compute ──
    ├── [EC2]    EC2Model      — instances, SSM, port forward, Network Path, favorites
    ├── [ASG]    ASGModel      — Auto Scaling Groups, scaling policies, instances
    ├── [EBS]    EBSModel      — EBS volumes, state, type, IOPS, encryption, attachments
    │
    │  ── Networking ──
    ├── [VPC]    VPCModel      — VPCs, IGW, NAT, Peering, TGW, Endpoint, EIP
    ├── [Subnet] SubnetModel   — subnets, ENIs
    ├── [Routes] RouteModel    — route tables, route entries
    ├── [SG]     SGModel       — security groups, NACLs (f toggles)
    ├── [VPCE]   VPCEModel     — VPC Endpoints, service name, type, state
    ├── [TGW]    TGWModel      — Transit Gateways, attachments, route tables, routes
    │
    │  ── Load Balancing & Edge ──
    ├── [ELB]    ELBModel      — ALB/NLB/CLB, listeners, target groups, targets
    ├── [CF]     CFModel       — CloudFront distributions, origins, aliases, WAF, certs
    ├── [WAF]    WAFModel      — WAFv2 Web ACLs, rules, associated resources
    ├── [ACM]    ACMModel      — certificates, domain, status, expiry, SANs, in-use
    │
    │  ── Data & DNS ──
    ├── [R53]    R53Model      — Route 53 hosted zones, records (loaded on demand)
    ├── [RDS]    RDSModel      — DB instances, engine, class, endpoint, multi-AZ, storage
    ├── [S3]     S3Model       — buckets, region, versioning, encryption, public access
    │
    │  ── Containers & Serverless ──
    ├── [ECS]    ECSModel      — Clusters > Services > Tasks > Containers > Logs > ECS Exec
    ├── [EKS]    EKSModel      — Clusters > Namespaces > Pods/Deployments/Services, Nodes, Pod Logs
    ├── [Lambda] LambdaModel   — functions, runtime, memory, timeout, VPC config, layers
    │
    │  ── Monitoring & Security ──
    ├── [CW]     CWModel       — CloudWatch alarms, state, metric, threshold, dimensions
    ├── [IAM]    IAMModel      — users, groups, policies, last used
    │
    │  ── Troubleshooting ──
    └── [Check]  CheckModel    — connectivity checker, Reachability Analyzer
```

## Components

| Package | Path | Role |
|---------|------|------|
| **aws** | `internal/aws/` | All AWS SDK calls via 17 service clients (EC2, SSM, STS, ELBv2, ELB, AutoScaling, CloudWatch, CloudWatch Logs, IAM, CloudFront, WAFv2, ACM, Route53, RDS, S3, ECS, EKS, Lambda) + K8s REST API |
| **config** | `internal/config/` | User preferences (`~/.tui-aws/config.json`) |
| **store** | `internal/store/` | Favorites & session history persistence |
| **ui/root** | `internal/ui/root.go` | RootModel: tab switching, global keys, SSM exec, ECS exec, InterruptFilter |
| **ui/shared** | `internal/ui/shared/` | TabModel interface, SharedState, styles, table renderer, selector, overlay |
| **ui/tab_ec2** | `internal/ui/tab_ec2/` | EC2 tab: list, actions, search, filter, SSM, port forward, Network Path |
| **ui/tab_asg** | `internal/ui/tab_asg/` | ASG tab: groups, scaling policies, instances |
| **ui/tab_ebs** | `internal/ui/tab_ebs/` | EBS tab: volumes, state, type, IOPS, encryption, attachments |
| **ui/tab_vpc** | `internal/ui/tab_vpc/` | VPC tab: list, details (lazy-loads sub-resources) |
| **ui/tab_subnet** | `internal/ui/tab_subnet/` | Subnet tab: list, ENI viewer |
| **ui/tab_routetable** | `internal/ui/tab_routetable/` | Route Table tab: list, route entries |
| **ui/tab_sg** | `internal/ui/tab_sg/` | SG/NACL tab: rules viewer, mode toggle |
| **ui/tab_vpce** | `internal/ui/tab_vpce/` | VPCE tab: VPC Endpoints, service name, type, state |
| **ui/tab_tgw** | `internal/ui/tab_tgw/` | TGW tab: transit gateways, attachments, route tables, routes |
| **ui/tab_elb** | `internal/ui/tab_elb/` | ELB tab: ALB/NLB/CLB, listeners, target groups, target health |
| **ui/tab_cloudfront** | `internal/ui/tab_cloudfront/` | CF tab: distributions, origins, aliases, WAF, certificates |
| **ui/tab_waf** | `internal/ui/tab_waf/` | WAF tab: Web ACLs, rules, associated resources |
| **ui/tab_acm** | `internal/ui/tab_acm/` | ACM tab: certificates, domain, status, expiry, SANs |
| **ui/tab_r53** | `internal/ui/tab_r53/` | R53 tab: hosted zones, records loaded on demand |
| **ui/tab_rds** | `internal/ui/tab_rds/` | RDS tab: DB instances, engine, class, endpoint, multi-AZ |
| **ui/tab_s3** | `internal/ui/tab_s3/` | S3 tab: buckets, region, versioning, encryption, public access |
| **ui/tab_ecs** | `internal/ui/tab_ecs/` | ECS tab: deep-dive Clusters > Services > Tasks > Containers > Logs > ECS Exec |
| **ui/tab_eks** | `internal/ui/tab_eks/` | EKS tab: K8s integration Clusters > Namespaces > Pods/Deployments/Services, Nodes, Pod Logs |
| **ui/tab_lambda** | `internal/ui/tab_lambda/` | Lambda tab: functions, runtime, memory, timeout, VPC config, layers |
| **ui/tab_cloudwatch** | `internal/ui/tab_cloudwatch/` | CW tab: alarms, state, metric, threshold, dimensions |
| **ui/tab_iam** | `internal/ui/tab_iam/` | IAM tab: users, groups, policies, last used |
| **ui/tab_troubleshoot** | `internal/ui/tab_troubleshoot/` | Connectivity checker engine + Reachability Analyzer |

## Data Flow

```
┌──────────┐      ┌──────────────┐      ┌────────────────┐
│  User    │─────>│  RootModel   │─────>│  Active Tab    │
│  Input   │      │  (global     │      │  .Update()     │
│  (keys)  │      │   keys)      │      └───────┬────────┘
└──────────┘      └──────┬───────┘              │
                         │                       │ tea.Cmd
                         │                       v
                    ┌────┴─────┐         ┌──────────────┐
                    │  View()  │<────────│  AWS SDK /    │
                    │  render  │         │  K8s REST API │
                    └──────────┘         │  (async Cmd)  │
                                         └──────────────┘
```

1. User input -> RootModel checks global keys (tab switch, profile, region, quit)
2. Non-global keys -> delegated to active tab's Update()
3. Tab returns tea.Cmd for async AWS API calls
4. Response messages flow back through Update -> re-render
5. SSM sessions: EC2 tab sends SSMExecRequest -> RootModel runs tea.Exec (TUI suspended)
6. ECS Exec: ECS tab sends ECSExecRequest -> RootModel runs tea.Exec (similar to SSM flow)
7. Tab switching: NavigateToTab message -> RootModel switches active tab

## Caching Strategy

- **Lazy loading:** tabs fetch data on first activation, not at startup
- **30-second TTL:** fresh within TTL -> use cache; stale -> background reload showing cached data
- **Cache invalidation:** profile/region change -> clear all; `R` key -> clear active tab; SSM return -> clear EC2 tab
- **Memory only:** no disk cache, SharedState.Cache map

## SSM Session Flow

```
EC2Tab -> SSMExecRequest msg -> RootModel intercepts
  -> ssmExecCmd.Run() (aws ssm start-session)
  -> stty sane + TCIFLUSH (terminal reset)
  -> SSMSessionDoneMsg -> RootModel records history
  -> Forward to EC2Tab -> reload instances
```

## ECS Exec Flow

```
ECSTab -> ECSExecRequest msg -> RootModel intercepts
  -> ecsExecCmd.Run() (aws ecs execute-command)
  -> stty sane + TCIFLUSH (terminal reset)
  -> ECSExecDoneMsg -> RootModel forwards to ECSTab
  -> Resume at container list view
```

Requires `enableExecuteCommand` on the ECS service and the task role to have SSM permissions.

## K8s REST API Integration (EKS Tab)

```
EKS Cluster -> aws eks get-token (via os/exec)
  -> Bearer token for K8s API server
  -> Direct HTTP calls to https://<cluster-endpoint>/api/v1/...
  -> TLS verified via cluster CA certificate (base64-decoded from EKS API)
  -> No kubectl dependency
```

Supports: Namespaces, Pods, Deployments, Services, Nodes, and Pod log streaming.

## Connectivity Checker (Check Tab)

Local 5-step validation:
1. Source SG Outbound
2. Source NACL Outbound
3. Source Route Table path
4. Destination NACL Inbound
5. Destination SG Inbound

Each step: pass / blocked (stops, shows suggestion). Optional AWS Reachability Analyzer API for confirmation.

## Infrastructure
- **Runtime:** Single binary, requires AWS CLI + Session Manager Plugin
- **Storage:** `~/.tui-aws/` (config.json, favorites.json, history.json)
- **Build:** Cross-compiled via Makefile for linux/darwin x amd64/arm64
- **Setup:** `scripts/setup.sh` auto-installs prerequisites on macOS/Linux
