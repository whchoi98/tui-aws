<div align="center">

# tui-aws

### **[ [English](#english) | [한국어](#한국어) ]**

A terminal UI for managing AWS EC2 instances, exploring VPC networking infrastructure, and troubleshooting connectivity — all from your terminal.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)
![License](https://img.shields.io/badge/License-MIT-blue)

```
┌─ 1 EC2  2 VPC  3 Subnets  4 Routes  5 SG  6 Check ── Profile: prod ── Region: ap-northeast-2 ─┐
│ ★ ● web-server-1         i-0abc1234   running  10.0.1.10   t3.medium  2a                        │
│   ● web-server-2         i-0def5678   running  10.0.1.11   t3.medium  2c                        │
│   ● db-primary           i-0ghi9012   running  10.0.2.20   r5.xlarge  2a                        │
│   ○ batch-worker         i-0jkl3456   stopped  10.0.3.30   c5.2xlarge 2b                        │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓:Navigate  Enter:Actions  /:Search  f:Filter  p:Profile  r:Region  s:Sort  F:Fav  q:Quit     │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

</div>

---

<a id="english"></a>

## English

> **[ [English](#english) | [한국어](#한국어) ]**

### Table of Contents

- [Overview](#overview)
- [Features by Tab](#features-by-tab)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage Guide](#usage-guide)
- [Key Bindings](#key-bindings)
- [Use Cases](#use-cases)
- [IAM Permissions](#iam-permissions)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)
- [Tech Stack](#tech-stack)

---

## Overview

**tui-aws** is a single-binary terminal UI tool that replaces the need to juggle multiple AWS Console tabs or remember complex CLI commands. It provides:

- **6 integrated views** for EC2, VPC, Subnet, Route Table, Security Group, and Connectivity Check
- **SSM Session Manager** integration — connect to instances without SSH keys or open security group rules
- **Network path visualization** — trace the full path from VPC to NACL in one screen
- **Local connectivity checker** — validate SG + Route + NACL rules between any two instances without calling AWS APIs
- **Cross-resource navigation** — drill down from an EC2 instance to its VPC, Subnet, Route Table, or Security Group

---

## Features by Tab

### Tab 1: EC2 Instances

The primary view. Lists all EC2 instances in the selected region with real-time search, sorting, and filtering.

**Table columns:** Favorite (★/⏱), State icon (●/○), Name, Instance ID, State, Private IP, Type, AZ, Platform, Public IP, Launch Time, Security Groups, Key Pair, IAM Role

**Action menu (Enter):**

| Action | Description |
|--------|-------------|
| **SSM Session** | Opens an interactive shell via Session Manager. The TUI pauses and gives full terminal control to the SSM session. On exit, the TUI resumes and refreshes the instance list. |
| **Port Forwarding** | Tunnels a local port to a remote port on the instance. Enter local/remote port numbers, then the tunnel starts. Useful for accessing RDS, internal web servers, or debug ports on private instances. |
| **Network Path** | Shows the complete network path: VPC (name, CIDR) → Subnet (name, CIDR) → Route Table (all routes) → Security Group (inbound/outbound rules) → NACL (inbound/outbound rules). All in one scrollable overlay. |
| **Security Groups** | Lists all security group names attached to the instance. |
| **Instance Details** | Full detail overlay: ID, State, Type, AZ, IPs, VPC (name + ID + CIDR), Subnet (name + ID + CIDR), Platform, Key Pair, IAM Role, Launch Time, SSM connection status. |
| **Go to VPC** | Switches to the VPC tab with the instance's VPC focused. |
| **Go to Subnet** | Switches to the Subnet tab with the instance's subnet focused. |

**Special features:**
- **★ Favorites** (`F` key): Pin frequently accessed instances to the top of the list. Persisted to `~/.tui-aws/favorites.json`.
- **⏱ Recent history**: Instances you've recently connected to via SSM are marked and sorted higher. Persisted to `~/.tui-aws/history.json`.
- **Sort priority**: Favorites first → Recent history → User-selected sort field.

### Tab 2: VPC

Lists all VPCs in the region with CIDR blocks, default VPC indicator, and subnet counts.

**Table columns:** Name, VPC ID, CIDR, State, Default (✓/-)

**Action menu:**
- **VPC Details** — Comprehensive overlay showing all associated resources:
  - Internet Gateways (ID, Name, State)
  - NAT Gateways (ID, Name, Subnet, Private/Public IP, State)
  - VPC Peering Connections (ID, Requester/Accepter VPC, State)
  - Transit Gateway Attachments (ID, TGW ID, State)
  - VPC Endpoints (ID, Service Name, Type, State)
  - Elastic IPs (Public IP, Allocation ID, Associated Instance)
- **Subnets in this VPC** — Jumps to the Subnet tab filtered to this VPC
- **Route Tables** — Jumps to the Route Table tab filtered to this VPC
- **Security Groups** — Jumps to the SG tab filtered to this VPC

### Tab 3: Subnets

Lists all subnets with network details.

**Table columns:** Name, Subnet ID, VPC Name, CIDR, AZ (shortened), Available IPs, Public (✓/-)

**Action menu:**
- **ENIs in this Subnet** — Lists all Elastic Network Interfaces: ID, Description, Private/Public IP, Attached Instance, Security Groups, Status
- **Go to VPC** — Jumps to VPC tab with the subnet's VPC focused

### Tab 4: Route Tables

Lists all route tables with association info.

**Table columns:** Name, Route Table ID, VPC Name, Main (✓/-), Subnets count, Routes count

**Action menu:**
- **Route Entries** — Shows all route entries in a table format:
  ```
  Destination        Target           State
  10.1.0.0/16        local            active
  0.0.0.0/0          nat-0abc123      active
  10.11.0.0/16       pcx-0def456      active
  ```
- **Associated Subnets** — Lists subnet IDs explicitly associated with this route table

### Tab 5: Security Groups / NACLs

Two modes toggled by `f` key:

**Security Group mode (default):**

Table columns: Name, SG ID, VPC Name, Inbound rule count, Outbound rule count, Description

Action menu:
- **Inbound Rules** — Protocol, Port Range, Source (CIDR/SG/Prefix List), Description
- **Outbound Rules** — Same format as inbound

**NACL mode (press `f`):**

Table columns: Name, ACL ID, VPC Name, Default (✓/-), Subnets count

Action menu:
- **Inbound Rules** — Rule number, Protocol, Port Range, CIDR, Action (ALLOW/DENY). Rule number `*` represents the default deny rule.
- **Outbound Rules** — Same format

### Tab 6: Connectivity Check

Interactive troubleshooting tool for verifying network connectivity between two EC2 instances.

**Form fields:**
- Source instance (pick from list)
- Destination instance (pick from list)
- Protocol (tcp / udp / all)
- Port (e.g., 443)

**Local check (5 steps):**

```
  Connectivity: web-server → db-primary  TCP/443
  ══════════════════════════════════════════════

  ✓ Source SG Outbound     sg-0abc: TCP 443 → 0.0.0.0/0 ALLOW
  ✓ Source NACL Outbound   acl-xxx: Rule 100 All ALLOW
  ✓ Source Route           rtb-xxx: 10.2.0.0/16 → tgw-xxx (active)
  ✗ Dest SG Inbound        sg-0def: TCP 443 ← 10.1.0.0/16 NOT FOUND

  Result: ✗ BLOCKED at Destination SG Inbound
  Suggestion: Add inbound rule TCP 443 from 10.1.88.66/32
```

Each step checks: source SG outbound → source NACL outbound → source route → dest NACL inbound → dest SG inbound. Stops at first failure with a fix suggestion.

**AWS Reachability Analyzer (optional):**

Press `R` on the result screen to run AWS's own network path analysis. This calls the `CreateNetworkInsightsPath` and `StartNetworkInsightsAnalysis` APIs (may incur costs). A confirmation prompt is shown before execution.

---

## Quick Start

### One-line Install & Run

```bash
git clone https://github.com/whchoi98/tui-aws.git && cd tui-aws && ./scripts/setup.sh
```

That's it. The setup script handles everything — checks your system, installs missing packages, builds the binary, and launches tui-aws.

### What the setup script does

```
╔══════════════════════════════════════════╗
║         tui-aws Setup & Launcher         ║
╚══════════════════════════════════════════╝

[1/5] Checking AWS CLI...
  ✓ AWS CLI v2 (aws-cli/2.x.x)           ← installs if missing (macOS pkg / Linux zip)

[2/5] Checking Session Manager Plugin...
  ✓ Session Manager Plugin installed       ← installs if missing (macOS zip / Linux deb or rpm)

[3/5] Checking Go...
  ✓ Go 1.23 (/usr/local/go/bin/go)       ← installs to ~/.local/go/ if missing

[4/5] Checking AWS credentials...
  ✓ EC2 Instance Role detected             ← checks Instance Role / env vars / ~/.aws/credentials
  ✓ ~/.aws/credentials (2 profiles)

[5/5] Building tui-aws...
  ✓ Built: ./tui-aws
  ✓ Version: tui-aws 0.1.0

  ? Install tui-aws to /usr/local/bin/ (requires sudo)? [Y/n]
```

Each step prompts before installing. You can decline any step and install manually later.

### Already have prerequisites?

If AWS CLI, Session Manager Plugin, and Go are already installed:

```bash
git clone https://github.com/whchoi98/tui-aws.git
cd tui-aws
make build
./tui-aws
```

---

## Installation

### Supported Platforms

| OS | Architecture | Package Manager |
|----|-------------|-----------------|
| macOS | arm64 (Apple Silicon) | Homebrew / manual |
| macOS | amd64 (Intel) | Homebrew / manual |
| Linux | arm64 | apt (deb) / yum (rpm) |
| Linux | amd64 | apt (deb) / yum (rpm) |

### Prerequisites

| Tool | Required | Purpose | Install |
|------|----------|---------|---------|
| **AWS CLI v2** | Yes | Runs `aws ssm start-session` | [Install guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) |
| **Session Manager Plugin** | Yes | Enables SSM session connections | [Install guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) |
| **Go 1.21+** | Build only | Compiles the binary | [go.dev/dl](https://go.dev/dl/) |
| **AWS Credentials** | Yes | API access | `aws configure`, env vars, or EC2 Instance Role |

### Build from Source

```bash
# Build for current platform
make build

# Cross-compile all platforms
make build-all

# Output binaries
ls dist/
# tui-aws-linux-amd64
# tui-aws-linux-arm64
# tui-aws-darwin-arm64
# tui-aws-darwin-amd64

# Run tests
make test

# Clean build artifacts
make clean
```

### Run

```bash
./tui-aws              # Launch TUI
./tui-aws --version    # Print version
```

---

## Usage Guide

### Switching Profiles and Regions

Press `p` to open the profile selector. The list includes:
- `(instance role)` — uses the EC2 instance's IAM role (no `--profile` flag)
- Named profiles from `~/.aws/credentials` and `~/.aws/config`

Press `r` to open the region selector with all standard AWS regions.

Changing profile or region reloads the active tab's data.

### Searching and Filtering

Press `/` in any tab to activate search mode. Type to filter rows by name, ID, or IP address. Press `Esc` to clear and exit search.

Press `f` to open the filter overlay (EC2 tab: filter by state; SG tab: toggle SG/NACL mode).

### SSM Session

1. Select an instance in the EC2 tab
2. Press `Enter` → select **SSM Session**
3. The TUI suspends and you get a full terminal shell on the instance
4. Type `exit` or `Ctrl+D` to return to the TUI
5. Instance list automatically refreshes

**If the session fails**, the error is displayed in the TUI (e.g., `exit status 255` for permission issues or missing SSM agent).

### Port Forwarding

1. Select an instance → `Enter` → **Port Forwarding**
2. Enter local port (default: 8080) and remote port (default: 80)
3. Press `Enter` to start the tunnel
4. Access the service at `localhost:<local-port>`
5. Press `Ctrl+C` to stop the tunnel and return to TUI

**Example use cases:**
- `localhost:3306` → RDS on EC2 (MySQL)
- `localhost:8080` → Internal web server
- `localhost:9229` → Node.js remote debugger

### Connectivity Check

1. Switch to tab `6` (Check)
2. Navigate to Source → press `Enter` → pick an instance
3. Navigate to Destination → press `Enter` → pick an instance
4. Edit Protocol (default: tcp) and Port (default: 443)
5. Press `Enter` to run the local check
6. Review the 5-step result
7. (Optional) Press `R` for AWS Reachability Analyzer

---

## Key Bindings

### Global Keys (all tabs)

| Key | Action |
|-----|--------|
| `1` `2` `3` `4` `5` `6` | Switch to tab (EC2 / VPC / Subnets / Routes / SG / Check) |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `p` | Select AWS profile |
| `r` | Select AWS region |
| `R` | Refresh current tab data |
| `q` / `Ctrl+C` | Quit the application |

### Table Keys (all tabs)

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Move cursor up / down |
| `Enter` | Open action menu for selected row |
| `/` | Start search (type to filter, `Esc` to cancel) |
| `f` | Open filter / toggle mode (SG tab) |
| `s` | Cycle sort column (Name → ID → State → Type → AZ) |
| `S` | Reverse sort direction (asc ↔ desc) |
| `F` | Toggle favorite (EC2 tab only) |
| `Esc` | Close any overlay, cancel search |

### Connectivity Check Keys (Tab 6)

| Key | Action |
|-----|--------|
| `Tab` / `↑` `↓` | Navigate between form fields |
| `Enter` | Pick instance (on Source/Dest) / Run check (on Protocol/Port) |
| `R` | Run AWS Reachability Analyzer (on result screen) |
| `y` / `n` | Confirm/cancel Reachability Analyzer |
| `Esc` | Back to previous screen |

---

## Use Cases

### 1. Quick SSM Access to Private Instances

No need to remember instance IDs or type long CLI commands:
```
tui-aws → select instance → Enter → SSM Session → you're in
```

### 2. Investigate "Why Can't A Talk to B?"

```
tui-aws → Tab 6 (Check) → pick Source → pick Dest → Enter
→ See exactly which SG/NACL/Route is blocking
→ Get a fix suggestion
```

### 3. Audit VPC Networking

```
tui-aws → Tab 2 (VPC) → Enter → VPC Details
→ See all IGWs, NATs, Peering, TGW, Endpoints, EIPs at a glance
→ Jump to Subnets/Routes/SGs for this VPC
```

### 4. Review Security Group Rules

```
tui-aws → Tab 5 (SG) → Enter → Inbound Rules
→ See all rules in a table: Protocol, Ports, Source, Description
→ Press f to switch to NACLs
```

### 5. Port Forward to a Database

```
tui-aws → select DB instance → Enter → Port Forwarding
→ Local: 3306, Remote: 3306 → Enter
→ mysql -h localhost -P 3306 -u admin -p  (in another terminal)
```

---

## IAM Permissions

### Minimum (EC2 + SSM only)

Sufficient for Tab 1 (EC2) with SSM connections:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeInstances",
      "ec2:DescribeVpcs",
      "ec2:DescribeSubnets",
      "ssm:StartSession",
      "ssm:TerminateSession",
      "ssm:DescribeInstanceInformation",
      "sts:GetCallerIdentity"
    ],
    "Resource": "*"
  }]
}
```

### Full (all tabs)

Required for all 6 tabs:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeInstances",
      "ec2:DescribeVpcs",
      "ec2:DescribeSubnets",
      "ec2:DescribeInternetGateways",
      "ec2:DescribeNatGateways",
      "ec2:DescribeVpcPeeringConnections",
      "ec2:DescribeTransitGatewayAttachments",
      "ec2:DescribeVpcEndpoints",
      "ec2:DescribeAddresses",
      "ec2:DescribeNetworkInterfaces",
      "ec2:DescribeRouteTables",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeNetworkAcls",
      "ssm:StartSession",
      "ssm:TerminateSession",
      "ssm:DescribeInstanceInformation",
      "sts:GetCallerIdentity"
    ],
    "Resource": "*"
  }]
}
```

### Reachability Analyzer (optional, may incur costs)

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:CreateNetworkInsightsPath",
      "ec2:DeleteNetworkInsightsPath",
      "ec2:StartNetworkInsightsAnalysis",
      "ec2:DescribeNetworkInsightsAnalyses"
    ],
    "Resource": "*"
  }]
}
```

> **Note:** If a tab shows "AccessDenied", only that tab is affected. Other tabs continue working.

---

## Configuration

### Config Directory

All config files are stored in `~/.tui-aws/`. On first run, the directory is created automatically. If migrating from the older `tui-ssm`, the setup process auto-renames `~/.tui-ssm/` → `~/.tui-aws/`.

| File | Purpose |
|------|---------|
| `config.json` | Default profile, region, table display settings |
| `favorites.json` | Favorited instances (★ marker), keyed by instance ID + profile + region |
| `history.json` | SSM session history (⏱ marker), FIFO with max 100 entries |

### config.json

```json
{
  "default_profile": "default",
  "default_region": "ap-northeast-2",
  "refresh_interval_seconds": 0,
  "table": {
    "visible_columns": ["name", "id", "state", "private_ip", "type", "az"],
    "sort_by": "name",
    "sort_order": "asc"
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `default_profile` | `"default"` | AWS profile to use on startup |
| `default_region` | `"us-east-1"` | AWS region to use on startup |
| `refresh_interval_seconds` | `0` | Auto-refresh interval (0 = manual only) |
| `table.sort_by` | `"name"` | Default sort column |
| `table.sort_order` | `"asc"` | Default sort direction |

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    RootModel                         │
│  ┌─────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │ Tab Bar  │  │ SharedState  │  │ Global        │  │
│  │ 1-6 keys │  │ profile      │  │ Overlays      │  │
│  │ active   │  │ region       │  │ (profile/     │  │
│  │ highlight│  │ cache        │  │  region       │  │
│  │          │  │ dimensions   │  │  selector)    │  │
│  └─────────┘  └──────────────┘  └───────────────┘  │
│                       │                              │
│  ┌────────┬────────┬──┴────┬────────┬────────┬────┐ │
│  │ EC2    │ VPC    │Subnet │ Route  │ SG/    │Check│ │
│  │ Tab    │ Tab    │ Tab   │ Tab    │ NACL   │ Tab │ │
│  │        │        │       │        │ Tab    │     │ │
│  └────────┴────────┴───────┴────────┴────────┴────┘ │
│              Each tab: TabModel interface            │
│              Init() / Update() / View()              │
└─────────────────────────────────────────────────────┘
         │                            │
         ▼                            ▼
┌─────────────────┐         ┌──────────────────┐
│  internal/aws/   │         │  ~/.tui-aws/      │
│  EC2, VPC, SG,   │         │  config.json      │
│  Route, NACL,    │         │  favorites.json   │
│  Reachability    │         │  history.json     │
│  (aws-sdk-go-v2) │         │                   │
└─────────────────┘         └──────────────────┘
```

### Project Structure

```
tui-aws/
├── main.go                          Entry point, config migration, TUI launch
├── Makefile                         Build targets (build, build-all, test, clean)
├── scripts/
│   └── setup.sh                     Cross-platform setup & install script
├── internal/
│   ├── aws/                         AWS SDK integration (all use ec2.Client)
│   │   ├── ec2.go                   Instance model, FetchInstances, EnrichVpcSubnetInfo
│   │   ├── vpc.go                   VPC, IGW, NAT, Peering, TGW, Endpoint, EIP fetching
│   │   ├── subnet.go               Subnet, ENI fetching
│   │   ├── network.go              Route Table, Route entry fetching
│   │   ├── security.go             Security Group rules, Network ACL rules fetching
│   │   ├── reachability.go         VPC Reachability Analyzer (create/start/poll/cleanup)
│   │   ├── profile.go              AWS profile parsing (~/.aws/credentials + config)
│   │   ├── session.go              SDK client factory (EC2/SSM/STS)
│   │   └── ssm.go                  SSM command building, prerequisite checks
│   ├── config/
│   │   └── config.go               Load/save user config (~/.tui-aws/config.json)
│   ├── store/
│   │   ├── favorites.go            Favorites CRUD + persistence
│   │   └── history.go              Session history FIFO + persistence
│   └── ui/
│       ├── root.go                  RootModel, tab switching, SSM exec, InterruptFilter
│       ├── tab.go                   Re-exports TabModel, SharedState, TabID from shared/
│       ├── placeholder.go           PlaceholderTab for future tabs
│       ├── shared/
│       │   ├── tab.go              TabModel interface, SharedState, CachedData, TabID enum
│       │   ├── styles.go           Lip Gloss styles (Gruvbox theme), tab bar styles
│       │   ├── table.go            Column, RenderRow, ExpandNameColumn
│       │   ├── overlay.go          RenderOverlay, PlaceOverlay (centered on screen)
│       │   └── selector.go         SelectorModel (reusable list picker)
│       ├── tab_ec2/                 EC2: model, table, actions, search, filter
│       ├── tab_vpc/                 VPC: model, table, detail (lazy sub-resources)
│       ├── tab_subnet/             Subnet: model, table, detail (ENIs)
│       ├── tab_routetable/         Routes: model, table, detail (route entries)
│       ├── tab_sg/                 SG/NACL: model, table, detail (dual mode)
│       └── tab_troubleshoot/       Check: model, checker engine, result renderer
├── docs/
│   ├── architecture.md              System architecture document
│   ├── decisions/                   Architecture Decision Records (ADRs)
│   └── runbooks/                    Operational runbooks
└── .claude/                          Claude Code settings, hooks, skills
```

---

## Troubleshooting

### "exit status 255" when connecting via SSM

The `aws ssm start-session` command failed. Common causes:

| Cause | Solution |
|-------|----------|
| Invalid AWS credentials | Check `~/.aws/credentials` — look for syntax errors (stray characters on line 1) |
| Missing SSM Agent on instance | Verify the instance has SSM Agent installed and running |
| Missing IAM role | The instance needs an IAM role with `AmazonSSMManagedInstanceCore` policy |
| VPC endpoint missing | For private subnets without NAT, create SSM VPC endpoints (`ssm`, `ssmmessages`, `ec2messages`) |
| Wrong profile/region | Press `p`/`r` to switch profile/region in tui-aws |

### "AccessDenied" on a tab

The current IAM identity lacks the required EC2 Describe permissions. Only the affected tab shows the error — other tabs continue working. See [IAM Permissions](#iam-permissions) for the full policy.

### Garbled text or broken columns

Ensure your terminal supports:
- **UTF-8** encoding
- **256-color** or **TrueColor** mode
- A **monospace font** with Unicode support (e.g., JetBrains Mono, Fira Code, Menlo)

If using SSH, ensure `TERM` is set correctly: `export TERM=xterm-256color`

### TUI doesn't return after SSM session

tui-aws includes terminal reset (`stty sane` + stdin flush) after SSM sessions. If issues persist:

```bash
# Manual terminal reset
reset
# Or
stty sane
```

### Setup script fails

```bash
# Run with debug output
bash -x ./scripts/setup.sh

# Check Go installation
go version

# Check AWS CLI
aws --version

# Check Session Manager Plugin
session-manager-plugin --version
```

---

## Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | [Go 1.25](https://go.dev/) | Fast compilation, single binary, cross-platform |
| TUI Framework | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) | Elm architecture (Model-View-Update) |
| Styling | [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) | Terminal styling with Gruvbox color theme |
| AWS SDK | [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | EC2, SSM, STS API calls |
| Terminal Width | [charmbracelet/x/ansi](https://github.com/charmbracelet/x) | Unicode-aware text truncation |

### Contributors

- **whchoi98** — [whchoi98@gmail.com](mailto:whchoi98@gmail.com) — [GitHub](https://github.com/whchoi98)

<p align="right"><a href="#tui-aws">⬆ Back to top</a></p>

---

<br>

---

<a id="한국어"></a>

# 한국어

> **[ [English](#english) | [한국어](#한국어) ]**

### 목차

- [개요](#개요)
- [탭별 기능 상세](#탭별-기능-상세)
- [빠른 시작](#빠른-시작)
- [설치 방법](#설치-방법)
- [사용 가이드](#사용-가이드)
- [키 바인딩 전체](#키-바인딩-전체)
- [활용 시나리오](#활용-시나리오)
- [IAM 권한 설정](#iam-권한-설정)
- [설정 파일](#설정-파일)
- [아키텍처](#아키텍처)
- [문제 해결](#문제-해결)
- [기술 스택 상세](#기술-스택-상세)

---

## 개요

**tui-aws**는 여러 AWS 콘솔 탭을 오가거나 복잡한 CLI 명령을 기억할 필요 없이, 터미널 하나에서 AWS 인프라를 관리할 수 있는 도구입니다.

- **6개 통합 뷰** — EC2, VPC, Subnet, Route Table, Security Group, 연결성 검사
- **SSM Session Manager 통합** — SSH 키나 보안 그룹 인바운드 규칙 없이 인스턴스 접속
- **네트워크 경로 시각화** — VPC에서 NACL까지 전체 경로를 한 화면에 표시
- **로컬 연결성 검사기** — AWS API 호출 없이 SG + Route + NACL 규칙을 검증
- **크로스 리소스 탐색** — EC2 인스턴스에서 VPC, Subnet, Route Table, SG로 즉시 이동

---

## 탭별 기능 상세

### 탭 1: EC2 인스턴스

기본 뷰. 선택한 리전의 모든 EC2 인스턴스를 실시간 검색, 정렬, 필터링과 함께 표시합니다.

**테이블 컬럼:** 즐겨찾기(★/⏱), 상태 아이콘(●/○), Name, Instance ID, State, Private IP, Type, AZ, Platform, Public IP, Launch Time, Security Groups, Key Pair, IAM Role

**액션 메뉴 (Enter):**

| 액션 | 설명 |
|------|------|
| **SSM Session** | Session Manager를 통해 대화형 셸을 엽니다. TUI가 일시 중지되고 SSM 세션에 터미널 제어가 넘어갑니다. `exit`으로 종료하면 TUI가 재개되고 인스턴스 목록이 새로고침됩니다. |
| **Port Forwarding** | 로컬 포트를 인스턴스의 리모트 포트로 터널링합니다. 로컬/리모트 포트 번호를 입력하면 터널이 시작됩니다. 프라이빗 인스턴스의 RDS, 내부 웹 서버, 디버그 포트에 접근할 때 유용합니다. |
| **Network Path** | 전체 네트워크 경로를 표시: VPC(이름, CIDR) → Subnet(이름, CIDR) → Route Table(모든 경로) → Security Group(인바운드/아웃바운드 규칙) → NACL(인바운드/아웃바운드 규칙). 하나의 스크롤 가능한 오버레이에 모두 표시됩니다. |
| **Security Groups** | 인스턴스에 연결된 모든 보안 그룹 이름을 표시합니다. |
| **Instance Details** | 전체 상세 정보: ID, State, Type, AZ, IP, VPC(이름+ID+CIDR), Subnet(이름+ID+CIDR), Platform, Key Pair, IAM Role, Launch Time, SSM 연결 상태. |
| **Go to VPC** | 해당 인스턴스의 VPC가 포커스된 상태로 VPC 탭으로 전환합니다. |
| **Go to Subnet** | 해당 인스턴스의 서브넷이 포커스된 상태로 Subnet 탭으로 전환합니다. |

**특수 기능:**
- **★ 즐겨찾기** (`F` 키): 자주 접근하는 인스턴스를 목록 최상단에 고정. `~/.tui-aws/favorites.json`에 저장.
- **⏱ 최근 이력**: SSM으로 최근 접속한 인스턴스에 마커가 표시되고 상위 정렬. `~/.tui-aws/history.json`에 저장.
- **정렬 우선순위**: 즐겨찾기 → 최근 이력 → 사용자 선택 정렬 필드.

### 탭 2: VPC

리전의 모든 VPC를 CIDR 블록, 기본 VPC 표시, 서브넷 수와 함께 나열합니다.

**액션 메뉴:**
- **VPC Details** — 모든 연관 리소스를 보여주는 종합 오버레이:
  - Internet Gateway (ID, 이름, 상태)
  - NAT Gateway (ID, 이름, 서브넷, Private/Public IP, 상태)
  - VPC Peering (ID, 요청자/수락자 VPC, 상태)
  - Transit Gateway Attachment (ID, TGW ID, 상태)
  - VPC Endpoint (ID, 서비스 이름, 유형, 상태)
  - Elastic IP (Public IP, Allocation ID, 연결된 인스턴스)
- **Subnets in this VPC** — 해당 VPC로 필터링된 Subnet 탭으로 이동
- **Route Tables** — 해당 VPC로 필터링된 Route Table 탭으로 이동
- **Security Groups** — 해당 VPC로 필터링된 SG 탭으로 이동

### 탭 3: Subnets

모든 서브넷을 네트워크 상세 정보와 함께 나열합니다.

**테이블 컬럼:** Name, Subnet ID, VPC Name, CIDR, AZ, Available IPs, Public(✓/-)

**액션 메뉴:**
- **ENIs in this Subnet** — 모든 ENI 나열: ID, 설명, Private/Public IP, 연결된 인스턴스, 보안 그룹, 상태
- **Go to VPC** — 해당 서브넷의 VPC로 VPC 탭 이동

### 탭 4: Route Tables

모든 라우트 테이블을 연결 정보와 함께 나열합니다.

**테이블 컬럼:** Name, Route Table ID, VPC Name, Main(✓/-), Subnets 수, Routes 수

**액션 메뉴:**
- **Route Entries** — 모든 경로 엔트리를 테이블 형식으로 표시:
  ```
  Destination        Target           State
  10.1.0.0/16        local            active
  0.0.0.0/0          nat-0abc123      active
  10.11.0.0/16       pcx-0def456      active
  ```
- **Associated Subnets** — 명시적으로 연결된 서브넷 ID 목록

### 탭 5: Security Groups / NACLs

`f` 키로 전환되는 두 가지 모드:

**Security Group 모드 (기본):**
- 테이블: Name, SG ID, VPC Name, Inbound 수, Outbound 수, Description
- 액션: Inbound Rules / Outbound Rules (Protocol, Port, Source, Description)

**NACL 모드 (`f` 누름):**
- 테이블: Name, ACL ID, VPC Name, Default(✓/-), Subnets 수
- 액션: Inbound Rules / Outbound Rules (Rule#, Protocol, Port, CIDR, Action)
- Rule number `*`는 기본 거부 규칙을 나타냅니다.

### 탭 6: 연결성 검사

두 EC2 인스턴스 간 네트워크 연결성을 검증하는 대화형 도구.

**입력 필드:**
- Source 인스턴스 (목록에서 선택)
- Destination 인스턴스 (목록에서 선택)
- Protocol (tcp / udp / all)
- Port (예: 443)

**로컬 검사 (5단계):**

```
  Connectivity: web-server → db-primary  TCP/443
  ══════════════════════════════════════════════

  ✓ Source SG Outbound     sg-0abc: TCP 443 → 0.0.0.0/0 ALLOW
  ✓ Source NACL Outbound   acl-xxx: Rule 100 All ALLOW
  ✓ Source Route           rtb-xxx: 10.2.0.0/16 → tgw-xxx (active)
  ✗ Dest SG Inbound        sg-0def: TCP 443 ← 10.1.0.0/16 NOT FOUND

  Result: ✗ BLOCKED at Destination SG Inbound
  Suggestion: Add inbound rule TCP 443 from 10.1.88.66/32
```

각 단계: Source SG 아웃바운드 → Source NACL 아웃바운드 → Source 라우팅 → Dest NACL 인바운드 → Dest SG 인바운드. 첫 실패 지점에서 중단하고 수정 제안을 표시합니다.

**AWS Reachability Analyzer (선택적):**

결과 화면에서 `R`을 눌러 AWS 자체 네트워크 경로 분석을 실행합니다. `CreateNetworkInsightsPath`와 `StartNetworkInsightsAnalysis` API를 호출합니다 (비용 발생 가능). 실행 전 확인 프롬프트가 표시됩니다.

---

## 빠른 시작

### 한 줄로 설치 & 실행

```bash
git clone https://github.com/whchoi98/tui-aws.git && cd tui-aws && ./scripts/setup.sh
```

이 한 줄이면 끝입니다. 설치 스크립트가 시스템을 점검하고, 부족한 패키지를 설치하고, 바이너리를 빌드한 후 tui-aws를 실행합니다.

### 설치 스크립트 동작 과정

```
╔══════════════════════════════════════════╗
║         tui-aws Setup & Launcher         ║
╚══════════════════════════════════════════╝

[1/5] Checking AWS CLI...
  ✓ AWS CLI v2 (aws-cli/2.x.x)           ← 미설치 시 자동 설치 (macOS pkg / Linux zip)

[2/5] Checking Session Manager Plugin...
  ✓ Session Manager Plugin installed       ← 미설치 시 자동 설치 (macOS zip / Linux deb 또는 rpm)

[3/5] Checking Go...
  ✓ Go 1.23 (/usr/local/go/bin/go)       ← 미설치 시 ~/.local/go/에 설치

[4/5] Checking AWS credentials...
  ✓ EC2 Instance Role detected             ← Instance Role / 환경변수 / ~/.aws/credentials 확인
  ✓ ~/.aws/credentials (2 profiles)

[5/5] Building tui-aws...
  ✓ Built: ./tui-aws
  ✓ Version: tui-aws 0.1.0

  ? Install tui-aws to /usr/local/bin/ (requires sudo)? [Y/n]
```

각 단계에서 설치 전 확인을 요청합니다. 거절하고 나중에 수동 설치할 수 있습니다.

### 이미 필수 패키지가 설치되어 있다면?

AWS CLI, Session Manager Plugin, Go가 이미 있으면:

```bash
git clone https://github.com/whchoi98/tui-aws.git
cd tui-aws
make build
./tui-aws
```

---

## 설치 방법

### 지원 플랫폼

| OS | 아키텍처 | 패키지 관리자 |
|----|---------|-------------|
| macOS | arm64 (Apple Silicon) | Homebrew / 수동 |
| macOS | amd64 (Intel) | Homebrew / 수동 |
| Linux | arm64 | apt (deb) / yum (rpm) |
| Linux | amd64 | apt (deb) / yum (rpm) |

### 필수 조건

| 도구 | 필수 | 용도 | 설치 |
|------|------|------|------|
| **AWS CLI v2** | 예 | `aws ssm start-session` 실행 | [설치 가이드](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) |
| **Session Manager Plugin** | 예 | SSM 세션 연결 | [설치 가이드](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) |
| **Go 1.21+** | 빌드 시 | 바이너리 컴파일 | [go.dev/dl](https://go.dev/dl/) |
| **AWS 자격 증명** | 예 | API 접근 | `aws configure`, 환경변수, 또는 EC2 Instance Role |

### 소스에서 빌드

```bash
make build          # 현재 플랫폼 빌드
make build-all      # 크로스 컴파일 (linux/darwin × amd64/arm64)
make test           # 테스트 실행
make clean          # 빌드 산출물 삭제
```

---

## 사용 가이드

### 프로파일 및 리전 변경

`p`를 눌러 프로파일 선택기를 엽니다:
- `(instance role)` — EC2 인스턴스의 IAM 역할 사용 (`--profile` 플래그 없음)
- `~/.aws/credentials`와 `~/.aws/config`의 명명된 프로파일

`r`을 눌러 모든 표준 AWS 리전이 있는 리전 선택기를 엽니다.

프로파일이나 리전을 변경하면 현재 탭의 데이터가 리로드됩니다.

### 검색 및 필터링

아무 탭에서 `/`를 눌러 검색 모드를 활성화합니다. 이름, ID, IP로 필터링됩니다. `Esc`로 검색을 해제합니다.

`f`를 눌러 필터 오버레이를 엽니다 (EC2 탭: 상태별 필터, SG 탭: SG/NACL 모드 전환).

### SSM 세션

1. EC2 탭에서 인스턴스 선택
2. `Enter` → **SSM Session** 선택
3. TUI가 일시 중지되고 인스턴스에서 전체 터미널 셸을 사용
4. `exit` 또는 `Ctrl+D`로 TUI 복귀
5. 인스턴스 목록 자동 새로고침

**세션 실패 시** TUI에 에러가 표시됩니다 (예: 권한 문제나 SSM 에이전트 미설치 시 `exit status 255`).

### 포트 포워딩

1. 인스턴스 선택 → `Enter` → **Port Forwarding**
2. 로컬 포트(기본: 8080)와 리모트 포트(기본: 80) 입력
3. `Enter`로 터널 시작
4. `localhost:<로컬포트>`로 서비스 접근
5. `Ctrl+C`로 터널 중지 및 TUI 복귀

**활용 예시:**
- `localhost:3306` → EC2의 MySQL (RDS)
- `localhost:8080` → 내부 웹 서버
- `localhost:9229` → Node.js 원격 디버거

---

## 키 바인딩 전체

### 전역 키 (모든 탭)

| 키 | 동작 |
|----|------|
| `1` `2` `3` `4` `5` `6` | 탭 전환 (EC2 / VPC / Subnets / Routes / SG / Check) |
| `Tab` / `Shift+Tab` | 다음 / 이전 탭 |
| `p` | AWS 프로파일 선택 |
| `r` | AWS 리전 선택 |
| `R` | 현재 탭 데이터 새로고침 |
| `q` / `Ctrl+C` | 종료 |

### 테이블 키 (모든 탭)

| 키 | 동작 |
|----|------|
| `↑` `↓` / `j` `k` | 커서 위/아래 이동 |
| `Enter` | 선택한 행의 액션 메뉴 열기 |
| `/` | 검색 시작 (입력으로 필터링, `Esc`로 취소) |
| `f` | 필터 열기 / 모드 전환 (SG 탭) |
| `s` | 정렬 컬럼 순환 (Name → ID → State → Type → AZ) |
| `S` | 정렬 방향 반전 (asc ↔ desc) |
| `F` | 즐겨찾기 토글 (EC2 탭 전용) |
| `Esc` | 모든 오버레이 닫기, 검색 취소 |

### 연결성 검사 키 (탭 6)

| 키 | 동작 |
|----|------|
| `Tab` / `↑` `↓` | 폼 필드 간 이동 |
| `Enter` | 인스턴스 선택 (Source/Dest) / 검사 실행 (Protocol/Port) |
| `R` | AWS Reachability Analyzer 실행 (결과 화면에서) |
| `y` / `n` | Reachability Analyzer 확인/취소 |
| `Esc` | 이전 화면으로 |

---

## 활용 시나리오

### 1. 프라이빗 인스턴스에 빠르게 SSM 접속

인스턴스 ID를 외우거나 긴 CLI 명령을 입력할 필요 없이:
```
tui-aws → 인스턴스 선택 → Enter → SSM Session → 바로 접속
```

### 2. "A에서 B로 왜 통신이 안 되지?" 조사

```
tui-aws → 탭 6 (Check) → Source 선택 → Dest 선택 → Enter
→ 어떤 SG/NACL/Route가 차단하는지 정확히 확인
→ 수정 제안 받기
```

### 3. VPC 네트워킹 감사

```
tui-aws → 탭 2 (VPC) → Enter → VPC Details
→ 모든 IGW, NAT, Peering, TGW, Endpoint, EIP를 한눈에 파악
→ 해당 VPC의 Subnet/Route/SG로 바로 이동
```

### 4. 보안 그룹 규칙 검토

```
tui-aws → 탭 5 (SG) → Enter → Inbound Rules
→ 모든 규칙을 테이블로 확인: Protocol, Port, Source, Description
→ f를 눌러 NACL로 전환
```

### 5. 데이터베이스 포트 포워딩

```
tui-aws → DB 인스턴스 선택 → Enter → Port Forwarding
→ Local: 3306, Remote: 3306 → Enter
→ mysql -h localhost -P 3306 -u admin -p  (다른 터미널에서)
```

---

## IAM 권한 설정

### 최소 (EC2 + SSM만)

탭 1 (EC2)과 SSM 접속에 충분:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeInstances",
      "ec2:DescribeVpcs",
      "ec2:DescribeSubnets",
      "ssm:StartSession",
      "ssm:TerminateSession",
      "ssm:DescribeInstanceInformation",
      "sts:GetCallerIdentity"
    ],
    "Resource": "*"
  }]
}
```

### 전체 (모든 탭)

6개 탭 모두 사용:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeInstances",
      "ec2:DescribeVpcs",
      "ec2:DescribeSubnets",
      "ec2:DescribeInternetGateways",
      "ec2:DescribeNatGateways",
      "ec2:DescribeVpcPeeringConnections",
      "ec2:DescribeTransitGatewayAttachments",
      "ec2:DescribeVpcEndpoints",
      "ec2:DescribeAddresses",
      "ec2:DescribeNetworkInterfaces",
      "ec2:DescribeRouteTables",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeNetworkAcls",
      "ssm:StartSession",
      "ssm:TerminateSession",
      "ssm:DescribeInstanceInformation",
      "sts:GetCallerIdentity"
    ],
    "Resource": "*"
  }]
}
```

### Reachability Analyzer (선택적, 비용 발생 가능)

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:CreateNetworkInsightsPath",
      "ec2:DeleteNetworkInsightsPath",
      "ec2:StartNetworkInsightsAnalysis",
      "ec2:DescribeNetworkInsightsAnalyses"
    ],
    "Resource": "*"
  }]
}
```

> **참고:** 탭에서 "AccessDenied"가 표시되면 해당 탭만 영향을 받습니다. 다른 탭은 정상 동작합니다.

---

## 설정 파일

### 설정 디렉토리

모든 설정 파일은 `~/.tui-aws/`에 저장됩니다. 첫 실행 시 자동 생성됩니다. 이전 `tui-ssm`에서 마이그레이션 시 `~/.tui-ssm/` → `~/.tui-aws/`로 자동 이름 변경됩니다.

| 파일 | 용도 |
|------|------|
| `config.json` | 기본 프로파일, 리전, 테이블 표시 설정 |
| `favorites.json` | 즐겨찾기 인스턴스 (★ 마커), instance ID + profile + region으로 키 |
| `history.json` | SSM 세션 이력 (⏱ 마커), 최대 100개 FIFO |

### config.json

```json
{
  "default_profile": "default",
  "default_region": "ap-northeast-2",
  "refresh_interval_seconds": 0,
  "table": {
    "visible_columns": ["name", "id", "state", "private_ip", "type", "az"],
    "sort_by": "name",
    "sort_order": "asc"
  }
}
```

| 필드 | 기본값 | 설명 |
|------|--------|------|
| `default_profile` | `"default"` | 시작 시 사용할 AWS 프로파일 |
| `default_region` | `"us-east-1"` | 시작 시 사용할 AWS 리전 |
| `refresh_interval_seconds` | `0` | 자동 새로고침 간격 (0 = 수동만) |
| `table.sort_by` | `"name"` | 기본 정렬 컬럼 |
| `table.sort_order` | `"asc"` | 기본 정렬 방향 |

---

## 문제 해결

### SSM 접속 시 "exit status 255"

`aws ssm start-session` 명령이 실패했습니다. 주요 원인:

| 원인 | 해결 방법 |
|------|----------|
| 잘못된 AWS 자격 증명 | `~/.aws/credentials` 확인 — 구문 오류(1번째 줄의 잘못된 문자 등) 점검 |
| 인스턴스에 SSM Agent 없음 | SSM Agent가 설치되고 실행 중인지 확인 |
| IAM 역할 없음 | `AmazonSSMManagedInstanceCore` 정책이 포함된 IAM 역할 필요 |
| VPC 엔드포인트 없음 | NAT 없는 프라이빗 서브넷은 SSM VPC 엔드포인트 필요 (`ssm`, `ssmmessages`, `ec2messages`) |
| 잘못된 프로파일/리전 | tui-aws에서 `p`/`r`로 프로파일/리전 변경 |

### 탭에서 "AccessDenied"

현재 IAM 자격 증명에 필요한 EC2 Describe 권한이 없습니다. 해당 탭만 에러를 표시하고 다른 탭은 정상 동작합니다. [IAM 권한 설정](#iam-권한-설정)에서 전체 정책을 확인하세요.

### 텍스트 깨짐 또는 컬럼 정렬 오류

터미널이 다음을 지원하는지 확인하세요:
- **UTF-8** 인코딩
- **256색** 또는 **TrueColor** 모드
- Unicode를 지원하는 **고정폭 글꼴** (예: JetBrains Mono, Fira Code, Menlo)

SSH 사용 시 `TERM`이 올바르게 설정되었는지 확인: `export TERM=xterm-256color`

### SSM 세션 후 TUI가 복귀하지 않음

tui-aws는 SSM 세션 후 터미널 리셋(`stty sane` + stdin flush)을 포함합니다. 문제가 지속되면:

```bash
# 수동 터미널 리셋
reset
# 또는
stty sane
```

### 설치 스크립트 실패

```bash
# 디버그 출력으로 실행
bash -x ./scripts/setup.sh

# Go 설치 확인
go version

# AWS CLI 확인
aws --version

# Session Manager Plugin 확인
session-manager-plugin --version
```

---

## 기술 스택 상세

| 구성 요소 | 기술 | 용도 |
|----------|------|------|
| 언어 | [Go 1.25](https://go.dev/) | 빠른 컴파일, 단일 바이너리, 크로스 플랫폼 |
| TUI 프레임워크 | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) | Elm 아키텍처 (Model-View-Update) |
| 스타일링 | [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) | Gruvbox 테마 터미널 스타일링 |
| AWS SDK | [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | EC2, SSM, STS API 호출 |
| 터미널 너비 | [charmbracelet/x/ansi](https://github.com/charmbracelet/x) | Unicode 인식 텍스트 truncation |

### Contributors

- **whchoi98** — [whchoi98@gmail.com](mailto:whchoi98@gmail.com) — [GitHub](https://github.com/whchoi98)

<p align="right"><a href="#tui-aws">⬆ 맨 위로</a></p>
