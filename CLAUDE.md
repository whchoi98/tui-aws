# Project Context

## Overview
tui-ssm — AWS EC2 인스턴스를 TUI로 조회하고 Session Manager로 접속하는 Go CLI 도구.

## Tech Stack
- **Language:** Go 1.25
- **TUI:** Bubble Tea v2 (Elm architecture), Lip Gloss v2, Bubbles
- **AWS:** aws-sdk-go-v2 (EC2, SSM, STS, Config)
- **SSM Session:** `os/exec` → `aws ssm start-session` via `tea.Exec()`
- **Build:** Makefile, cross-compile (linux/darwin × amd64/arm64)

## Project Structure
```
main.go              - Entry point, prerequisite checks, TUI launch
internal/
  config/            - Config load/save (~/.tui-ssm/config.json)
  store/             - Favorites & history CRUD (~/.tui-ssm/)
  aws/               - Profile parsing, SDK clients, EC2/SSM operations
  ui/                - Bubble Tea model, views, components (Gruvbox theme)
docs/                - Architecture docs, ADRs, runbooks
.claude/             - Claude settings, hooks, skills
tools/               - Scripts, prompts
```

## Conventions
- Go standard project layout: `internal/` for private packages
- Bubble Tea Elm architecture: Model → Update → View
- State machine pattern for UI (ViewTable, ViewSearch, ViewFilter, etc.)
- Value receivers on Model (Bubble Tea requirement), pointer receivers on Store types
- Test files colocated: `*_test.go` alongside implementation
- JSON config/store files under `~/.tui-ssm/`
- Error handling: return errors up, display in TUI via `m.err`

## Key Commands
```bash
make build          # Build binary
make build-all      # Cross-compile for all platforms
make test           # Run all tests (go test ./... -v)
make clean          # Remove build artifacts
go vet ./...        # Static analysis
go test ./internal/aws/ -run TestInstance -v  # Run specific tests
```

---

## Auto-Sync Rules

Rules below are applied automatically after Plan mode exit and on major code changes.

### Post-Plan Mode Actions
After exiting Plan mode (`/plan`), before starting implementation:

1. **Architecture decision made** -> Update `docs/architecture.md`
2. **Technical choice/trade-off made** -> Create `docs/decisions/ADR-NNN-title.md`
3. **New module added** -> Create `CLAUDE.md` in that module directory
4. **Operational procedure defined** -> Create runbook in `docs/runbooks/`
5. **Changes needed in this file** -> Update relevant sections above

### Code Change Sync Rules
- New directory under `internal/` -> Must create `CLAUDE.md` alongside
- AWS API usage added/changed -> Update `internal/aws/CLAUDE.md`
- UI component added/changed -> Update `internal/ui/CLAUDE.md`
- Config/store schema changed -> Update respective module `CLAUDE.md`
- Infrastructure changed -> Update `docs/architecture.md` Infrastructure section

### ADR Numbering
Find the highest number in `docs/decisions/ADR-*.md` and increment by 1.
Format: `ADR-NNN-concise-title.md`
