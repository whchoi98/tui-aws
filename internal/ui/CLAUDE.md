# UI Module

## Role
Bubble Tea v2 TUI implementation: tab-based architecture with root model, shared components, and 22 per-tab packages.

## Architecture
```
internal/ui/
├── root.go              — RootModel (tea.Model), tab switching, global overlays (profile/region), SSMExecCmd, ECSExecCmd, InterruptFilter
├── tab.go               — Re-exports from shared: TabModel, TabID, SharedState, NavigateToTab
├── placeholder.go       — PlaceholderTab for any unregistered tabs
├── flush_linux.go       — Linux-specific stdin flush (TCIFLUSH via syscall)
├── flush_other.go       — Non-Linux stub flush
├── shared/
│   ├── tab.go           — TabModel interface, TabID enum (22 tabs), SharedState, CachedData, NavigateToTab
│   ├── styles.go        — All Lip Gloss Gruvbox styles + StateStyle helper
│   ├── table.go         — Column type + RenderRow (generic table renderer), ExpandNameColumn
│   ├── overlay.go       — RenderOverlay, PlaceOverlay helpers
│   └── selector.go      — SelectorModel (generic list picker)
├── tab_ec2/             — EC2: SSM, port forward, Network Path, favorites, search, filter
├── tab_asg/             — ASG: groups, scaling policies, instances
├── tab_ebs/             — EBS: volumes, state, type, IOPS, encryption, attachments
├── tab_vpc/             — VPC: list + details (IGW/NAT/Peering/TGW/Endpoint/EIP)
├── tab_subnet/          — Subnet: list + ENI viewer
├── tab_routetable/      — Route Table: list + route entries
├── tab_sg/              — SG/NACL: rules viewer (f toggles mode)
├── tab_vpce/            — VPCE: VPC Endpoints, service name, type, state
├── tab_tgw/             — TGW: transit gateways, attachments, route tables, routes
├── tab_elb/             — ELB: ALB/NLB/CLB, listeners, target groups, targets
├── tab_cloudfront/      — CF: distributions, origins, aliases, WAF, certificates
├── tab_waf/             — WAF: Web ACLs, rules, associated resources
├── tab_acm/             — ACM: certificates, domain, status, expiry, SANs
├── tab_r53/             — R53: hosted zones, records loaded on demand
├── tab_rds/             — RDS: DB instances, engine, class, endpoint, multi-AZ
├── tab_s3/              — S3: buckets, region, versioning, encryption, public access
├── tab_ecs/             — ECS: Clusters > Services > Tasks > Containers > Logs > ECS Exec
├── tab_eks/             — EKS: Clusters > Namespaces > Pods/Deployments/Services, Nodes, Pod Logs
├── tab_lambda/          — Lambda: functions, runtime, memory, timeout, VPC config, layers
├── tab_cloudwatch/      — CW: alarms, state, metric, threshold, dimensions
├── tab_iam/             — IAM: users, groups, policies, last used
└── tab_troubleshoot/    — Check: connectivity checker + Reachability Analyzer
```

## Key Design
- **RootModel** owns SharedState, handles global keys (q, p, r, [/], Tab/Shift+Tab), delegates to active tab
- **TabModel interface** defined in shared/: Init, Update, View, ShortHelp
- **Tab navigation:** `[` / `]` keys move between tabs; `Tab` / `Shift+Tab` also works (except on Check tab when editing)
- **EC2Model** sends SSMExecRequest messages; RootModel intercepts and runs tea.Exec
- **ECSModel** sends ECSExecRequest messages; RootModel intercepts and runs tea.Exec (same ssmExecCmd wrapper)
- **SharedState** lives in shared/ to avoid circular imports between ui and tab packages
- **ui/tab.go** re-exports shared types so external callers (main.go) use the ui package
- All 22 tabs are fully implemented — no placeholder tabs remain

## Rules
- EC2Model uses pointer receiver (state mutations); RootModel uses value receiver (Bubble Tea requirement)
- ssmExecCmd wraps exec.Cmd with `stty sane` + stdin flush (TCIFLUSH on Linux, stub on other) after SSM/ECS Exec session
- InterruptFilter blocks OS SIGINT (raw mode delivers Ctrl+C as KeyPressMsg)
- View() always sets AltScreen = true (Bubble Tea v2 API)
- Profile/region selectors are global overlays in RootModel, not in individual tabs
- SSM session history is recorded in RootModel on SSMSessionDoneMsg, then forwarded to EC2 tab
- ECS Exec completion (ECSExecDoneMsg) is handled in RootModel, forwarded to ECS tab
- Check tab's IsEditing() guard prevents global keys (p, r, q, digits) from being consumed during text input
