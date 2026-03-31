# Architecture

## System Overview
tui-ssm is a single-binary Go CLI that provides a terminal UI for browsing AWS EC2 instances and connecting via SSM Session Manager. It follows the Elm architecture (Model-View-Update) via Bubble Tea v2.

## Components

| Component | Path | Role |
|-----------|------|------|
| **Entry Point** | `main.go` | CLI flags, prereq checks, program launch |
| **Config** | `internal/config/` | Load/save user preferences (`~/.tui-ssm/config.json`) |
| **Store** | `internal/store/` | Favorites & session history persistence |
| **AWS** | `internal/aws/` | Profile parsing, SDK clients, EC2/SSM API calls |
| **UI** | `internal/ui/` | Bubble Tea model, views, state machine, styles |

## Data Flow

```
┌─────────┐     ┌──────────┐     ┌─────────────┐
│  User    │────▶│  Bubble  │────▶│  AWS SDK     │
│  Input   │     │  Tea     │     │  (EC2/SSM)   │
│  (keys)  │     │  Update  │     └──────┬───────┘
└─────────┘     └────┬─────┘            │
                     │                   │
                     ▼                   ▼
                ┌─────────┐     ┌─────────────┐
                │  View   │     │  Instance    │
                │  Render │◀────│  List / SSM  │
                └─────────┘     │  Status      │
                                └─────────────┘
```

1. User input → Bubble Tea KeyPressMsg → Update dispatches by ViewState
2. Profile/region change → loadInstances Cmd → AWS EC2 DescribeInstances
3. Enter key → tea.Exec → `aws ssm start-session` (TUI suspended)
4. Session exit → ssmSessionDoneMsg → reload instances

## Infrastructure
- **Runtime:** Single binary, no runtime dependencies beyond AWS CLI + Session Manager Plugin
- **Storage:** Local JSON files under `~/.tui-ssm/` (config, favorites, history)
- **Build:** Cross-compiled via Makefile for linux/darwin × amd64/arm64
