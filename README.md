# tui-aws

> **[한국어 README는 여기를 클릭하세요 →](#한국어)**

A terminal UI for browsing AWS EC2 instances, VPC networking, and connecting via SSM Session Manager.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey)
![License](https://img.shields.io/badge/License-MIT-blue)

---

## Features

| Tab | Key | Description |
|-----|-----|-------------|
| **EC2** | `1` | Instance list, SSM connect, port forwarding, favorites, Network Path |
| **VPC** | `2` | VPC list with details (IGW, NAT, Peering, TGW, Endpoints, EIP) |
| **Subnets** | `3` | Subnet list (CIDR, AZ, available IPs, public/private), ENI viewer |
| **Routes** | `4` | Route Table list with route entries (Destination → Target → State) |
| **SG** | `5` | Security Group rules & Network ACL rules (toggle with `f`) |
| **Check** | `6` | Connectivity checker (SG + Route + NACL validation, AWS Reachability Analyzer) |

### Highlights

- **Tab-based navigation** — switch between 6 views with number keys
- **SSM Session Manager** — connect to EC2 without SSH keys or open ports
- **Port forwarding** — tunnel local ports to private EC2 instances
- **Network Path** — view VPC → Subnet → Route Table → SG → NACL in one overlay
- **Connectivity checker** — validate network paths between two instances locally
- **AWS Reachability Analyzer** — optional AWS-powered path analysis
- **Cross-VPC drill-down** — jump from EC2 → VPC → Subnet → Routes seamlessly
- **Instance Role support** — works on EC2 instances with IAM roles (no explicit credentials needed)

---

## Quick Start

```bash
git clone <repository-url> && cd tui-aws
./scripts/setup.sh
```

The setup script checks and installs all prerequisites automatically.

### Manual Installation

#### Prerequisites

| Tool | Required | Install |
|------|----------|---------|
| **AWS CLI v2** | Yes | [Install guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) |
| **Session Manager Plugin** | Yes | [Install guide](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) |
| **Go 1.21+** | For building | [go.dev/dl](https://go.dev/dl/) |
| **AWS Credentials** | Yes | `aws configure` or EC2 Instance Role |

#### Build

```bash
make build          # Build for current platform
make build-all      # Cross-compile (linux/darwin × amd64/arm64)
make test           # Run tests
```

#### Run

```bash
./tui-aws
./tui-aws --version
```

---

## Key Bindings

### Global Keys

| Key | Action |
|-----|--------|
| `1`-`6` | Switch tab |
| `Tab` / `Shift+Tab` | Next / previous tab |
| `p` | Select AWS profile |
| `r` | Select region |
| `R` | Refresh current tab |
| `q` / `Ctrl+C` | Quit |

### Table Navigation (all tabs)

| Key | Action |
|-----|--------|
| `↑` `↓` / `j` `k` | Move cursor |
| `Enter` | Action menu |
| `/` | Search (filter by name, ID, IP) |
| `f` | Filter (SG tab: toggle SG/NACL mode) |
| `s` / `S` | Sort column / reverse direction |
| `F` | Toggle favorite (EC2 tab only) |
| `Esc` | Close overlay / cancel search |

### EC2 Action Menu

| Action | Description |
|--------|-------------|
| SSM Session | Connect via Session Manager |
| Port Forwarding | Tunnel local:remote ports |
| Network Path | VPC → Subnet → Route → SG → NACL summary |
| Security Groups | View attached SG names |
| Instance Details | Full instance info (VPC, Subnet, CIDR, IAM Role, etc.) |
| Go to VPC | Jump to VPC tab |
| Go to Subnet | Jump to Subnet tab |

### Connectivity Check (Tab 6)

| Key | Action |
|-----|--------|
| `Tab` / `↑` `↓` | Switch field (Source, Dest, Protocol, Port) |
| `Enter` | Pick instance / Run check |
| `R` | Run AWS Reachability Analyzer (on result screen) |
| `Esc` | Back |

---

## IAM Permissions

### Minimum (EC2 + SSM)

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:DescribeInstances",
    "ec2:DescribeVpcs",
    "ec2:DescribeSubnets",
    "ssm:StartSession",
    "ssm:DescribeInstanceInformation",
    "sts:GetCallerIdentity"
  ],
  "Resource": "*"
}
```

### Full (all tabs)

```json
{
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
    "ssm:DescribeInstanceInformation",
    "sts:GetCallerIdentity"
  ],
  "Resource": "*"
}
```

### Reachability Analyzer (optional)

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:CreateNetworkInsightsPath",
    "ec2:StartNetworkInsightsAnalysis",
    "ec2:DescribeNetworkInsightsAnalyses",
    "ec2:DeleteNetworkInsightsPath"
  ],
  "Resource": "*"
}
```

---

## Configuration

Config files are stored in `~/.tui-aws/`:

| File | Purpose |
|------|---------|
| `config.json` | Default profile, region, table settings |
| `favorites.json` | Favorited instances (★) |
| `history.json` | SSM session history (⏱) |

### config.json example

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

---

## Project Structure

```
tui-aws/
├── main.go                          Entry point
├── Makefile                         Build targets
├── scripts/
│   └── setup.sh                     Cross-platform setup script
├── internal/
│   ├── aws/                         AWS SDK integration
│   │   ├── ec2.go                   EC2 instances
│   │   ├── vpc.go                   VPC, IGW, NAT, Peering, TGW, Endpoint, EIP
│   │   ├── subnet.go               Subnets, ENIs
│   │   ├── network.go              Route Tables
│   │   ├── security.go             Security Groups, NACLs
│   │   ├── reachability.go         VPC Reachability Analyzer
│   │   ├── profile.go              AWS profile parsing
│   │   ├── session.go              SDK client factory
│   │   └── ssm.go                  SSM session commands
│   ├── config/                      User configuration (~/.tui-aws/)
│   ├── store/                       Favorites & history persistence
│   └── ui/
│       ├── root.go                  Root model, tab switching
│       ├── shared/                  Shared styles, table renderer, selector
│       ├── tab_ec2/                 EC2 tab
│       ├── tab_vpc/                 VPC tab
│       ├── tab_subnet/             Subnet tab
│       ├── tab_routetable/         Route Table tab
│       ├── tab_sg/                 Security Group / NACL tab
│       └── tab_troubleshoot/       Connectivity checker tab
└── docs/                            Design specs, ADRs
```

---

## Tech Stack

- **Language:** Go 1.25
- **TUI:** [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) (Elm architecture)
- **Styling:** [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) (Gruvbox theme)
- **AWS:** [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2)

---

<br><br>

---

# 한국어

> **[Click here for English README →](#tui-aws)**

AWS EC2 인스턴스 조회, VPC 네트워크 탐색, SSM Session Manager 접속을 위한 터미널 UI 도구.

---

## 주요 기능

| 탭 | 키 | 설명 |
|----|-----|------|
| **EC2** | `1` | 인스턴스 목록, SSM 접속, 포트 포워딩, 즐겨찾기, Network Path |
| **VPC** | `2` | VPC 목록 + 상세 (IGW, NAT, Peering, TGW, Endpoint, EIP) |
| **Subnets** | `3` | 서브넷 목록 (CIDR, AZ, 가용 IP, Public/Private) + ENI 조회 |
| **Routes** | `4` | 라우트 테이블 + 경로 엔트리 (Destination → Target → State) |
| **SG** | `5` | 보안 그룹 규칙 & Network ACL 규칙 (`f`로 전환) |
| **Check** | `6` | 연결성 검사 (SG + Route + NACL 검증, AWS Reachability Analyzer) |

### 주요 특징

- **탭 기반 네비게이션** — 숫자키로 6개 뷰 전환
- **SSM Session Manager** — SSH 키 없이 EC2 접속
- **포트 포워딩** — 로컬 포트를 프라이빗 EC2에 터널링
- **Network Path** — VPC → Subnet → Route Table → SG → NACL 한 화면 요약
- **연결성 검사기** — 두 인스턴스 간 네트워크 경로를 로컬에서 검증
- **AWS Reachability Analyzer** — AWS 자체 경로 분석 (선택적)
- **크로스 VPC 드릴다운** — EC2 → VPC → Subnet → Routes 탭 간 이동
- **Instance Role 지원** — IAM 역할이 있는 EC2에서 자격 증명 없이 사용

---

## 빠른 시작

```bash
git clone <repository-url> && cd tui-aws
./scripts/setup.sh
```

설치 스크립트가 필수 패키지를 자동으로 확인하고 설치합니다.

### 수동 설치

#### 필수 조건

| 도구 | 필수 | 설치 |
|------|------|------|
| **AWS CLI v2** | 예 | [설치 가이드](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) |
| **Session Manager Plugin** | 예 | [설치 가이드](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html) |
| **Go 1.21+** | 빌드 시 | [go.dev/dl](https://go.dev/dl/) |
| **AWS 자격 증명** | 예 | `aws configure` 또는 EC2 Instance Role |

#### 빌드

```bash
make build          # 현재 플랫폼 빌드
make build-all      # 크로스 컴파일 (linux/darwin × amd64/arm64)
make test           # 테스트 실행
```

#### 실행

```bash
./tui-aws
./tui-aws --version
```

---

## 키 바인딩

### 전역 키

| 키 | 동작 |
|----|------|
| `1`-`6` | 탭 전환 |
| `Tab` / `Shift+Tab` | 다음 / 이전 탭 |
| `p` | AWS 프로파일 선택 |
| `r` | 리전 선택 |
| `R` | 현재 탭 새로고침 |
| `q` / `Ctrl+C` | 종료 |

### 테이블 내 키 (모든 탭)

| 키 | 동작 |
|----|------|
| `↑` `↓` / `j` `k` | 커서 이동 |
| `Enter` | 액션 메뉴 |
| `/` | 검색 (이름, ID, IP로 필터링) |
| `f` | 필터 (SG 탭: SG/NACL 모드 전환) |
| `s` / `S` | 정렬 컬럼 / 방향 반전 |
| `F` | 즐겨찾기 토글 (EC2 탭 전용) |
| `Esc` | 오버레이 닫기 / 검색 취소 |

### EC2 액션 메뉴

| 액션 | 설명 |
|------|------|
| SSM Session | Session Manager로 접속 |
| Port Forwarding | 로컬:리모트 포트 터널링 |
| Network Path | VPC → Subnet → Route → SG → NACL 요약 |
| Security Groups | 연결된 SG 이름 목록 |
| Instance Details | 전체 인스턴스 정보 (VPC, Subnet, CIDR, IAM Role 등) |
| Go to VPC | VPC 탭으로 이동 |
| Go to Subnet | Subnet 탭으로 이동 |

### 연결성 검사 (탭 6)

| 키 | 동작 |
|----|------|
| `Tab` / `↑` `↓` | 필드 전환 (Source, Dest, Protocol, Port) |
| `Enter` | 인스턴스 선택 / 검사 실행 |
| `R` | AWS Reachability Analyzer 실행 (결과 화면에서) |
| `Esc` | 뒤로 |

---

## IAM 권한

### 최소 (EC2 + SSM)

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:DescribeInstances",
    "ec2:DescribeVpcs",
    "ec2:DescribeSubnets",
    "ssm:StartSession",
    "ssm:DescribeInstanceInformation",
    "sts:GetCallerIdentity"
  ],
  "Resource": "*"
}
```

### 전체 (모든 탭)

```json
{
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
    "ssm:DescribeInstanceInformation",
    "sts:GetCallerIdentity"
  ],
  "Resource": "*"
}
```

### Reachability Analyzer (선택적)

```json
{
  "Effect": "Allow",
  "Action": [
    "ec2:CreateNetworkInsightsPath",
    "ec2:StartNetworkInsightsAnalysis",
    "ec2:DescribeNetworkInsightsAnalyses",
    "ec2:DeleteNetworkInsightsPath"
  ],
  "Resource": "*"
}
```

---

## 설정

설정 파일은 `~/.tui-aws/`에 저장됩니다:

| 파일 | 용도 |
|------|------|
| `config.json` | 기본 프로파일, 리전, 테이블 설정 |
| `favorites.json` | 즐겨찾기 인스턴스 (★) |
| `history.json` | SSM 접속 이력 (⏱) |

### config.json 예시

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

---

## 기술 스택

- **언어:** Go 1.25
- **TUI:** [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) (Elm 아키텍처)
- **스타일링:** [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) (Gruvbox 테마)
- **AWS:** [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2)
