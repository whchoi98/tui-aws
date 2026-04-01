# Project Context

## Overview
tui-aws — AWS 인프라 전체를 터미널에서 탐색, 관리, 트러블슈팅하는 Go TUI 도구. 22개 탭, 104개 소스 파일, ~23,000 lines of code.

## Tech Stack
- **Language:** Go 1.25
- **TUI:** Bubble Tea v2 (Elm architecture), Lip Gloss v2 (Gruvbox theme)
- **AWS:** aws-sdk-go-v2 — EC2, SSM, STS, ELBv2, ELB (Classic), AutoScaling, CloudWatch, CloudWatch Logs, IAM, CloudFront, WAFv2, ACM, Route53, RDS, S3, ECS, EKS, Lambda
- **K8s:** REST API via `net/http` — token from `aws eks get-token`, direct HTTP calls to EKS API server (no kubectl dependency)
- **SSM/ECS Exec:** `os/exec` → `aws ssm start-session` / `aws ecs execute-command` via custom exec wrappers + `tea.Exec()`
- **Build:** Makefile, cross-compile (linux/darwin x amd64/arm64)

## Project Structure
```
main.go                          Entry point, config migration, TUI launch
scripts/setup.sh                 Cross-platform setup & install script
internal/
  config/                        Config load/save (~/.tui-aws/config.json)
  store/                         Favorites & history CRUD (~/.tui-aws/)
  aws/
    ec2.go                       Instance model, FetchInstances, EnrichVpcSubnetInfo
    vpc.go                       VPC, IGW, NAT, Peering, TGW, Endpoint, EIP
    subnet.go                    Subnet, ENI
    network.go                   RouteTable, Route entries
    security.go                  SecurityGroup rules, NetworkACL rules
    reachability.go              VPC Reachability Analyzer
    profile.go                   AWS profile parsing (~/.aws/credentials + config)
    session.go                   SDK client factory (all 18 service clients)
    ssm.go                       SSM command building, prerequisite checks
    elb.go                       ALB/NLB/CLB, listeners, target groups, targets
    asg.go                       Auto Scaling Groups, scaling policies, instances
    ebs.go                       EBS volumes, attachments
    tgw.go                       Transit Gateways, attachments, route tables, routes
    cloudwatch.go                CloudWatch alarms
    iam.go                       IAM users, groups, policies
    cloudfront.go                CloudFront distributions
    waf.go                       WAFv2 Web ACLs, rules
    acm.go                       ACM certificates
    r53.go                       Route 53 hosted zones, records
    rds.go                       RDS DB instances
    s3.go                        S3 buckets, metadata
    ecs.go                       ECS clusters, services, tasks, containers, exec
    eks.go                       EKS clusters, node groups
    k8s.go                       K8s REST API (namespaces, pods, deployments, services, nodes, logs)
    lambda.go                    Lambda functions
  ui/
    root.go                      RootModel (tea.Model), tab switching, global overlays
    tab.go                       Re-exports: TabModel, TabID, SharedState, NavigateToTab
    placeholder.go               PlaceholderTab for future tabs
    shared/
      tab.go                     TabModel interface, SharedState, CachedData, TabID enum (22 tabs)
      styles.go                  All Lip Gloss styles (Gruvbox), tab bar styles
      table.go                   Column, RenderRow, ExpandNameColumn
      overlay.go                 RenderOverlay, PlaceOverlay (centered)
      selector.go                SelectorModel (generic list picker)
    tab_ec2/                     EC2: SSM, port forward, Network Path, favorites
    tab_asg/                     ASG: groups, scaling policies, instances
    tab_ebs/                     EBS: volumes, state, type, IOPS, encryption, attachments
    tab_vpc/                     VPC: list + details (IGW/NAT/Peering/TGW/Endpoint/EIP)
    tab_subnet/                  Subnet: list + ENI viewer
    tab_routetable/              Route Table: list + route entries
    tab_sg/                      SG/NACL: rules viewer (f toggles mode)
    tab_vpce/                    VPCE: VPC Endpoints, service name, type, state
    tab_tgw/                     TGW: transit gateways, attachments, route tables, routes
    tab_elb/                     ELB: ALB/NLB/CLB, listeners, target groups, targets
    tab_cloudfront/              CF: distributions, origins, aliases, WAF, certificates
    tab_waf/                     WAF: Web ACLs, rules, associated resources
    tab_acm/                     ACM: certificates, domain, status, expiry, SANs
    tab_r53/                     R53: hosted zones, records loaded on demand
    tab_rds/                     RDS: DB instances, engine, class, endpoint, multi-AZ
    tab_s3/                      S3: buckets, region, versioning, encryption, public access
    tab_ecs/                     ECS: Clusters > Services > Tasks > Containers > Logs > ECS Exec
    tab_eks/                     EKS: Clusters > Namespaces > Pods/Deployments/Services, Nodes, Pod Logs
    tab_lambda/                  Lambda: functions, runtime, memory, timeout, VPC config, layers
    tab_cloudwatch/              CW: alarms, state, metric, threshold, dimensions
    tab_iam/                     IAM: users, groups, policies, last used
    tab_troubleshoot/            Check: connectivity checker + Reachability Analyzer
docs/                            Architecture docs, ADRs, runbooks, specs
.claude/                         Claude settings, hooks, skills
```

## Conventions
- **Tab architecture:** RootModel owns SharedState, each tab implements TabModel interface
- **SharedState** in `shared/` package to avoid circular imports; `ui/tab.go` re-exports
- **Tab navigation:** `[` / `]` keys move between tabs; `Tab` / `Shift+Tab` also works (except on Check tab when editing)
- **EC2Model** sends `SSMExecRequest` messages; RootModel intercepts and runs `tea.Exec`
- **ECSModel** sends `ECSExecRequest` messages; RootModel intercepts similarly (ECS Exec flow)
- **EKS K8s integration:** token via `aws eks get-token`, direct HTTP calls to K8s API (no kubectl dependency)
- **Lazy loading:** tabs fetch data on first switch, 30s cache TTL
- **ssmExecCmd:** wraps exec.Cmd with `stty sane` + stdin TCIFLUSH after SSM/ECS Exec session
- **InterruptFilter:** blocks OS SIGINT (raw mode delivers Ctrl+C as KeyPressMsg)
- **View()** always sets `v.AltScreen = true` (Bubble Tea v2 API)
- **Cell-width aware:** `lipgloss.Width()` + `ansi.Truncate()` for Unicode/emoji columns
- **ExpandNameColumn:** Name column fills remaining terminal width (min 20, max 60)
- **Deep-dive tabs:** ECS and EKS use hierarchical drill-down (cluster > service > task > container)
- Test files colocated: `*_test.go` alongside implementation
- JSON config/store files under `~/.tui-aws/`

## Key Commands
```bash
make build          # Build binary (tui-aws)
make build-all      # Cross-compile for all platforms
make test           # Run all tests (go test ./... -v)
make clean          # Remove build artifacts
go vet ./...        # Static analysis
go test ./internal/ui/tab_troubleshoot/ -v  # Connectivity checker tests
./scripts/setup.sh  # Install prerequisites + build
```

---

## Auto-Sync Rules

Rules below are applied automatically after Plan mode exit and on major code changes.

### Post-Plan Mode Actions
After exiting Plan mode (`/plan`), before starting implementation:

1. **Architecture decision made** -> Update `docs/architecture.md`
2. **Technical choice/trade-off made** -> Create `docs/decisions/ADR-NNN-title.md`
3. **New tab added** -> Create `tab_<name>/` package with model.go, table.go, detail.go
4. **New module added** -> Create `CLAUDE.md` in that module directory
5. **Operational procedure defined** -> Create runbook in `docs/runbooks/`
6. **Changes needed in this file** -> Update relevant sections above

### Code Change Sync Rules
- New directory under `internal/` -> Must create `CLAUDE.md` alongside
- New AWS API usage -> Update `internal/aws/CLAUDE.md`
- New tab added -> Register in `root.go`, update `shared/tab.go` TabID enum
- UI shared component changed -> Update `internal/ui/shared/` CLAUDE.md or inline docs
- Config/store schema changed -> Update respective module `CLAUDE.md`
- Infrastructure changed -> Update `docs/architecture.md` Infrastructure section

### ADR Numbering
Find the highest number in `docs/decisions/ADR-*.md` and increment by 1.
Format: `ADR-NNN-concise-title.md`
