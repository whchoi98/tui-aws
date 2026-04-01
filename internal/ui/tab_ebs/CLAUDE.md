# EBS Tab
## Role
EBS Volumes — list volumes with state, type, size, IOPS, encryption status, attachments.
## Key Files
- `model.go` — EBSModel implementing TabModel
- `table.go` — Columns include Encrypted (✓ green / ✗ red color-coded)
- `detail.go` — Full volume detail + attachment info
## Rules
- Uses existing `ec2.Client` (DescribeVolumes)
- Encryption column styled green/red for visual compliance check
