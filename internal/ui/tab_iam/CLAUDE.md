# IAM Tab
## Role
IAM Users — list users with groups, policies, creation date, last password used.
## Key Files
- `model.go` — IAMModel implementing TabModel, status bar shows account ID
- `table.go` — Columns: UserName, UserID, ARN, Created, LastUsed
- `detail.go` — Full user detail + groups + attached policies
## Rules
- Uses `iam.Client` (ListUsers, ListGroupsForUser, ListAttachedUserPolicies)
- Account ID fetched via STS GetCallerIdentity
