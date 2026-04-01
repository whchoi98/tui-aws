# AWS Module

## Role
AWS SDK integration: profile parsing, client factory, and all AWS service API calls across 25 source files.

## Key Files

### Core
- `profile.go` — Parse AWS profiles from `~/.aws/credentials` and `~/.aws/config`
- `session.go` — NewClients factory (18 service clients), ValidateCredentials, KnownRegions
- `ec2.go` — Instance struct, DisplayName/StateIcon/ShortAZ helpers, FetchInstances (paginated), EnrichVpcSubnetInfo
- `ssm.go` — CheckPrerequisites, BuildSSMSessionArgs, BuildPortForwardArgs, FetchSSMStatus

### Networking (Phase 1-3)
- `vpc.go` — VPC, IGW, NAT, Peering, TGW attachment, Endpoint, EIP
- `subnet.go` — Subnet, ENI
- `network.go` — RouteTable, Route entries
- `security.go` — SecurityGroup rules, NetworkACL rules
- `reachability.go` — VPC Reachability Analyzer

### Compute / Load Balancing
- `elb.go` — ALB/NLB/CLB, listeners, target groups, targets
- `asg.go` — Auto Scaling Groups, scaling policies, instances
- `ebs.go` — EBS volumes, attachments
- `tgw.go` — Transit Gateways, attachments, route tables, routes

### Monitoring / Security
- `cloudwatch.go` — CloudWatch alarms
- `iam.go` — IAM users, groups, policies

### CDN / DNS
- `cloudfront.go` — CloudFront distributions
- `waf.go` — WAFv2 Web ACLs, rules
- `acm.go` — ACM certificates
- `r53.go` — Route 53 hosted zones, records

### Data
- `rds.go` — RDS DB instances
- `s3.go` — S3 buckets, metadata

### Containers / Serverless
- `ecs.go` — ECS clusters, services, tasks, containers, exec
- `eks.go` — EKS clusters, node groups
- `lambda.go` — Lambda functions

### K8s
- `k8s.go` — K8s REST API client: namespaces, pods, deployments, services, nodes, pod logs (no kubectl dependency)

### Tests
- `*_test.go` — Unit tests for profile parsing, command building, instance helpers (ec2_test.go, profile_test.go, ssm_test.go)

## Clients Struct (18 service clients)

```go
type Clients struct {
    EC2     *ec2.Client
    SSM     *ssm.Client
    STS     *sts.Client
    ELBv2   *elbv2.Client    // ALB, NLB, GWLB
    ELB     *elb.Client      // Classic LB
    ASG     *autoscaling.Client
    CW      *cloudwatch.Client
    CWL     *cwl.Client      // CloudWatch Logs
    IAM     *iam.Client
    CF      *cloudfront.Client
    WAF     *wafv2.Client
    ACM     *acm.Client
    R53     *route53.Client
    RDS     *rds.Client
    S3      *s3.Client
    ECS     *ecs.Client
    EKS     *eks.Client
    Lambda  *lambda.Client
    Profile string
    Region  string
}
```

## Rules
- Profile parsing handles both credentials `[name]` and config `[profile name]` formats
- "default" profile and InstanceRoleProfile omit `--profile` flag in SSM commands
- FetchInstances/FetchSSMStatus use AWS SDK paginators
- Instance.SSMConnected is populated separately via FetchSSMStatus
- K8s token obtained via `aws eks get-token` (os/exec), direct HTTPS calls to EKS API server
