# TUI-SSM Design Spec

AWS EC2 인스턴스를 TUI로 조회하고 Session Manager로 접속하는 Go CLI 도구.

## 기술 스택

- **언어:** Go
- **TUI:** Bubble Tea + Bubbles + Lip Gloss
- **AWS:** aws-sdk-go-v2 (EC2, SSM, STS, Config)
- **SSM 접속:** `os/exec`로 `aws ssm start-session` CLI 실행 (`bubbletea.ExecProcess()`)

## 프로젝트 구조

```
tui-ssm/
├── main.go                  # 엔트리포인트, CLI 플래그 파싱
├── go.mod
├── internal/
│   ├── app/
│   │   └── app.go           # Bubble Tea 프로그램 초기화, 모델 루트
│   ├── ui/
│   │   ├── model.go         # 메인 모델 (상태 머신)
│   │   ├── table.go         # EC2 테이블 렌더링
│   │   ├── statusbar.go     # 상단 바 (프로파일/리전/필터)
│   │   ├── helpbar.go       # 하단 키 바인딩 도움말
│   │   ├── search.go        # 검색 입력 컴포넌트
│   │   ├── filter.go        # 필터 토글 오버레이
│   │   ├── selector.go      # 프로파일/리전 선택 오버레이
│   │   └── styles.go        # Lip Gloss 스타일 정의
│   ├── aws/
│   │   ├── ec2.go           # EC2 인스턴스 조회 (DescribeInstances)
│   │   ├── ssm.go           # SSM 세션 시작 (exec aws ssm)
│   │   ├── profile.go       # AWS 프로파일 목록 파싱
│   │   └── session.go       # AWS SDK 세션/클라이언트 관리
│   ├── store/
│   │   ├── favorites.go     # 즐겨찾기 저장/로드 (~/.tui-ssm/favorites.json)
│   │   └── history.go       # 접속 이력 저장/로드 (~/.tui-ssm/history.json)
│   └── config/
│       └── config.go        # 설정 파일 (~/.tui-ssm/config.json)
└── Makefile
```

## UI 레이아웃

단일 뷰 방식. 하나의 메인 테이블 뷰에 모든 기능을 집약한다.

```
┌─ Profile: production ─── Region: ap-northeast-2 ─── Filter: running ──┐
│ Search: web-server_                                                    │
├────────────────────────────────────────────────────────────────────────┤
│ ★ Name            ID          State    Private IP   Type     AZ       │
│ ★ web-server-1    i-0abc...   running  10.0.1.10    t3.med   2a       │
│   web-server-2    i-0def...   running  10.0.1.11    t3.med   2c       │
│   db-primary      i-0ghi...   running  10.0.2.20    r5.xl    2a       │
│   batch-worker    i-0jkl...   stopped  10.0.3.30    c5.2xl   2b       │
├────────────────────────────────────────────────────────────────────────┤
│ ↑↓ Navigate  Enter: Connect  /: Search  f: Filter  p: Profile        │
│ r: Region  s: Sort  ★: Favorite  P: Port Forward  q: Quit            │
└────────────────────────────────────────────────────────────────────────┘
```

- 상단: 프로파일/리전/필터 상태 바
- 중앙: EC2 테이블 (스크롤 가능)
- 하단: 키 바인딩 도움말

## 데이터 모델

### EC2 Instance

```go
type Instance struct {
    InstanceID       string
    Name             string     // Name 태그
    State            string     // running, stopped, terminated 등
    PrivateIP        string
    PublicIP         string
    InstanceType     string
    AvailabilityZone string
    Platform         string     // Linux/Windows
    LaunchTime       time.Time
    SecurityGroups   []string   // SG 이름 목록
    KeyPair          string
    IAMRole          string     // IAM Instance Profile
    SSMConnected     bool       // SSM Agent 연결 여부
}
```

## 상태 머신

```
                    ┌─────────────┐
                    │   Loading   │ ← 시작, 리전/프로파일 변경 시
                    └──────┬──────┘
                           │ EC2 목록 수신
                           ▼
         ┌─────────────────────────────────────┐
         │            TableView (메인)          │
         │                                     │
         │  /: SearchMode    f: FilterOverlay  │
         │  p: ProfileSelect r: RegionSelect   │
         │  s: SortToggle    ★: ToggleFavorite │
         │  P: PortForward   Enter: Connect    │
         └───┬────┬────┬────┬────┬────┬────┬───┘
             │    │    │    │    │    │    │
             ▼    ▼    ▼    ▼    ▼    ▼    ▼
         Search Filter Profile Region Sort Fav  PortForward
         Mode   Over  Select Select Toggle      Config
             │    │    │      │              │
             └────┴────┴──────┘              │
                  │ Esc: 돌아가기             │
                  ▼                          ▼
             TableView                  ┌──────────┐
                                        │ SSM      │
             Enter ─────────────────────│ Session  │
                                        │ (exec)   │
                                        └────┬─────┘
                                             │ 세션 종료
                                             ▼
                                        TableView
                                        (새로고침 + 이력 업데이트)
```

**상태 목록:**

| 상태 | 설명 |
|------|------|
| `Loading` | EC2 목록 조회 중 (스피너 표시) |
| `TableView` | 메인 테이블 — 기본 상태 |
| `SearchMode` | 테이블 위 검색창 활성화, 실시간 필터링 |
| `FilterOverlay` | 상태별/태그별 필터 토글 오버레이 팝업 |
| `ProfileSelect` | 프로파일 목록 선택 오버레이 |
| `RegionSelect` | 리전 목록 선택 오버레이 |
| `PortForwardConfig` | 포트 포워딩 설정 입력 (로컬 포트, 리모트 포트) |
| `SSMSession` | Bubble Tea 일시 중지 → `os/exec`로 세션 실행 → 종료 시 복귀 |

**SSM 세션 실행:** `bubbletea.ExecProcess()`를 사용하여 TUI를 일시 중지하고 터미널 제어를 `aws ssm start-session`에 넘김. 세션 종료 시 Bubble Tea가 자동 복귀하고, EC2 목록을 새로고침하며 이력을 업데이트.

## AWS 연동

**호출 흐름:**

1. 프로파일/리전 결정 → `aws.NewSession(profile, region)`
2. EC2 목록 조회 (비동기 Bubble Tea Cmd) → `ec2.DescribeInstances()` → `[]Instance`
3. SSM 연결 상태 확인 (필터용) → `ssm.DescribeInstanceInformation()` → SSM Agent 상태 맵
4. SSM 세션 시작 → `os/exec`: `aws ssm start-session --target <instance-id> --profile <profile> --region <region>`
5. 포트 포워딩 → `os/exec`: `aws ssm start-session --target <instance-id> --document-name AWS-StartPortForwardingSession --parameters portNumber=<remote>,localPortNumber=<local>`

**프로파일 파싱:** `~/.aws/credentials`와 `~/.aws/config` 양쪽에서 프로파일 섹션을 파싱. `aws-sdk-go-v2/config` 패키지의 `LoadSharedConfigProfile` 활용.

**리전 목록:** 하드코딩된 주요 리전 목록 + `ec2.DescribeRegions()` API로 동적 조회 (캐싱).

## 로컬 저장소

저장 경로: `~/.tui-ssm/`

### favorites.json

```json
{
  "favorites": [
    {
      "instance_id": "i-0abc1234",
      "profile": "production",
      "region": "ap-northeast-2",
      "alias": "web-server-1",
      "added_at": "2026-03-30T10:00:00Z"
    }
  ]
}
```

- 프로파일+리전+인스턴스ID 조합으로 유니크 식별
- `alias`는 Name 태그 스냅샷 (인스턴스 삭제 후에도 이력에 표시 가능)

### history.json

```json
{
  "sessions": [
    {
      "instance_id": "i-0abc1234",
      "profile": "production",
      "region": "ap-northeast-2",
      "alias": "web-server-1",
      "type": "session",
      "connected_at": "2026-03-30T09:30:00Z"
    }
  ],
  "max_entries": 100
}
```

- `type`: `"session"` 또는 `"port_forward"`
- FIFO로 `max_entries` 초과 시 오래된 항목 제거
- 테이블에서 최근 접속 인스턴스에 시각적 마커 표시 (예: `⏱`)

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

- `refresh_interval_seconds`: 0이면 수동 새로고침만, 양수면 자동 주기 새로고침
- `visible_columns`: 터미널 폭이 좁을 때 사용자가 표시 컬럼을 조정 가능

## 키 바인딩

| 키 | 동작 |
|----|------|
| `↑` / `k` | 위로 이동 |
| `↓` / `j` | 아래로 이동 |
| `Enter` | 선택한 인스턴스에 SSM 접속 |
| `/` | 검색 모드 진입 (실시간 필터링) |
| `Esc` | 검색/오버레이 닫기 |
| `f` | 필터 오버레이 토글 |
| `p` | 프로파일 선택 |
| `r` | 리전 선택 |
| `s` | 정렬 컬럼 순환 (Name → ID → State → Type → AZ) |
| `S` | 정렬 방향 토글 (asc ↔ desc) |
| `F` | 선택한 인스턴스 즐겨찾기 토글 |
| `P` | 포트 포워딩 모드 진입 |
| `R` | 목록 수동 새로고침 |
| `q` / `Ctrl+C` | 프로그램 종료 |

**테이블 정렬 규칙:**

1. 즐겨찾기 항목이 항상 최상단
2. 최근 접속 이력이 있는 항목이 그 다음 (최근순)
3. 나머지는 사용자 선택 정렬 기준 적용

**상태별 시각적 구분:**

| 상태 | 색상 | 아이콘 |
|------|------|--------|
| running | 초록 | `●` |
| stopped | 빨강 | `○` |
| pending | 노랑 | `◐` |
| stopping | 주황 | `◑` |
| terminated | 회색 | `✕` |

**검색 동작:**

- `/` 입력 시 테이블 상단에 검색창 표시
- 타이핑할 때마다 Name, ID, Private IP를 대상으로 실시간 필터링
- `Enter`로 검색 결과 첫 번째 항목에 바로 SSM 접속
- `Esc`로 검색 해제 및 전체 목록 복원

## 에러 핸들링

**AWS 연동 에러:**

| 상황 | 처리 |
|------|------|
| AWS 자격 증명 없음/만료 | 하단에 에러 메시지 표시, 프로파일 변경 유도 |
| API 호출 실패 (네트워크, 권한) | 에러 메시지 + `R`키로 재시도 안내 |
| EC2 인스턴스 0개 | "해당 리전에 인스턴스가 없습니다" 메시지 표시 |
| SSM 세션 시작 실패 | `aws ssm` CLI 미설치 또는 Session Manager Plugin 미설치 감지 → 설치 안내 메시지 |
| SSM Agent 미연결 인스턴스에 접속 시도 | 접속 전 경고 표시, 계속 진행 여부 확인 |

**SSM 전제 조건 검증 (시작 시 1회):**

1. `aws` CLI 존재 여부 → `which aws`
2. Session Manager Plugin 존재 여부 → `which session-manager-plugin`
3. 기본 프로파일로 STS GetCallerIdentity 호출 → 자격 증명 유효성

실패 시 TUI 진입 전에 명확한 에러 메시지와 해결 방법 출력 후 종료.

**터미널 엣지 케이스:**

| 상황 | 처리 |
|------|------|
| 터미널 폭 < 80 | 컬럼 자동 축소 (Name, State, IP만 표시) |
| 터미널 리사이즈 | `tea.WindowSizeMsg`로 감지, 레이아웃 재계산 |
| 즐겨찾기 인스턴스가 삭제됨 | 즐겨찾기 목록에 유지하되 `(not found)` 표시, 접속 불가 처리 |
| 접속 이력의 인스턴스가 삭제됨 | 이력에는 alias로 표시, 테이블 마커는 생략 |

## 빌드 & 배포

**빌드 타겟:**

- `linux/amd64`, `linux/arm64` — EC2, 점프 호스트 등
- `darwin/arm64`, `darwin/amd64` — macOS 로컬

**실행 전제 조건:**

- AWS CLI v2
- Session Manager Plugin
- 유효한 AWS 자격 증명 (`~/.aws/credentials` 또는 환경변수)
- 필요 IAM 권한: `ec2:DescribeInstances`, `ssm:StartSession`, `ssm:DescribeInstanceInformation`, `ec2:DescribeRegions`

**배포:** 단일 바이너리 복사로 완료 — 런타임 의존성 없음.
