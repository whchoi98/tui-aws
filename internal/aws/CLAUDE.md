# AWS Module

## Role
AWS SDK integration: profile parsing, client factory, EC2 instance fetching, SSM session management, and prerequisite checks.

## Key Files
- `profile.go` — Parse AWS profiles from `~/.aws/credentials` and `~/.aws/config`
- `session.go` — NewClients factory (EC2/SSM/STS), ValidateCredentials, KnownRegions
- `ec2.go` — Instance struct, DisplayName/StateIcon/ShortAZ helpers, FetchInstances (paginated)
- `ssm.go` — CheckPrerequisites, BuildSSMSessionArgs, BuildPortForwardArgs, FetchSSMStatus
- `*_test.go` — Unit tests for profile parsing, command building, instance helpers

## Rules
- Profile parsing handles both credentials `[name]` and config `[profile name]` formats
- "default" profile omits `--profile` flag in SSM commands
- FetchInstances/FetchSSMStatus use AWS SDK paginators
- Instance.SSMConnected is populated separately via FetchSSMStatus
