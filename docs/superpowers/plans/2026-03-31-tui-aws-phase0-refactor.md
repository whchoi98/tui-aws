# tui-aws Phase 0: Refactor to Tab Architecture

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename tui-ssm → tui-aws, extract the current monolithic model.go into a RootModel + EC2 TabModel architecture, and create shared UI components — all while keeping existing functionality identical.

**Architecture:** RootModel owns tab switching, profile/region state, and StatusBar. Each tab implements TabModel interface (Init/Update/View/ShortHelp). SharedState holds profile, region, clients, dimensions, favorites, history, and cache. EC2 tab is the only tab in Phase 0 — placeholder tabs for VPC/Subnet/Routes/SG/Troubleshoot show "Coming soon".

**Tech Stack:** Go 1.25, Bubble Tea v2, Lip Gloss v2, aws-sdk-go-v2

---

## File Structure

```
tui-aws/
├── main.go                         # MODIFY: module rename, config dir migration
├── Makefile                        # MODIFY: binary name tui-aws
├── go.mod                          # MODIFY: module tui-aws
├── .gitignore                      # MODIFY: tui-aws binary
├── internal/
│   ├── config/config.go            # MODIFY: ~/.tui-aws/ paths
│   ├── store/favorites.go          # MODIFY: ~/.tui-aws/ paths
│   ├── store/history.go            # MODIFY: ~/.tui-aws/ paths
│   ├── aws/                        # NO CHANGE
│   └── ui/
│       ├── root.go                 # CREATE: RootModel, tab switching, shared state
│       ├── tab.go                  # CREATE: TabModel interface, SharedState, CachedData
│       ├── shared/
│       │   ├── styles.go           # CREATE: move from ui/styles.go
│       │   ├── table.go            # CREATE: move renderRow, cellValue, Column from ui/table.go
│       │   ├── overlay.go          # CREATE: move OverlayStyle helpers
│       │   └── selector.go         # CREATE: move SelectorModel from ui/selector.go
│       └── tab_ec2/
│           ├── model.go            # CREATE: EC2Model extracted from ui/model.go
│           ├── actions.go          # CREATE: move ActionMenuModel from ui/actionmenu.go
│           ├── table.go            # CREATE: EC2-specific columns, cellValue, cellStyle
│           ├── search.go           # CREATE: move SearchModel from ui/search.go
│           └── filter.go           # CREATE: move FilterModel from ui/filter.go
│
│   (DELETE after extraction:)
│   └── ui/
│       ├── model.go                # DELETE (replaced by root.go + tab_ec2/model.go)
│       ├── styles.go               # DELETE (moved to shared/styles.go)
│       ├── table.go                # DELETE (split into shared/table.go + tab_ec2/table.go)
│       ├── statusbar.go            # DELETE (merged into root.go)
│       ├── helpbar.go              # DELETE (merged into root.go)
│       ├── search.go               # DELETE (moved to tab_ec2/search.go)
│       ├── filter.go               # DELETE (moved to tab_ec2/filter.go)
│       ├── selector.go             # DELETE (moved to shared/selector.go)
│       └── actionmenu.go           # DELETE (moved to tab_ec2/actions.go)
```

---

### Task 1: Rename Module and Binary

**Files:**
- Modify: `go.mod`
- Modify: `Makefile`
- Modify: `.gitignore`

- [ ] **Step 1: Update go.mod module name**

Change `module tui-ssm` to `module tui-aws` in go.mod, and update all import paths in every .go file.

```bash
cd /home/ec2-user/my-project/tui-ssm
sed -i 's|module tui-ssm|module tui-aws|' go.mod
find . -name '*.go' -exec sed -i 's|"tui-ssm/|"tui-aws/|g' {} +
```

- [ ] **Step 2: Update Makefile**

```makefile
BINARY := tui-aws
```

- [ ] **Step 3: Update .gitignore**

Replace `tui-ssm` with `tui-aws`.

- [ ] **Step 4: Verify build**

```bash
go build -o tui-aws ./main.go && ./tui-aws --version
```

Expected: `tui-aws dev`

- [ ] **Step 5: Run all tests**

```bash
go test ./... -v
```

Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor: rename module tui-ssm to tui-aws"
```

---

### Task 2: Update Config and Store Paths

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/store/favorites.go`
- Modify: `internal/store/history.go`
- Modify: `main.go`

- [ ] **Step 1: Update config.go Dir() and Path()**

Change `".tui-ssm"` to `".tui-aws"` in `Dir()` function.

- [ ] **Step 2: Update store paths**

Change `".tui-ssm"` to `".tui-aws"` in `FavoritesPath()` and `HistoryPath()`.

- [ ] **Step 3: Add config directory migration to main.go**

Before loading config, check if `~/.tui-ssm/` exists and `~/.tui-aws/` does not. If so, copy files over.

```go
// Migrate config from old directory if needed
oldDir := filepath.Join(home, ".tui-ssm")
newDir := config.Dir()
if _, err := os.Stat(oldDir); err == nil {
    if _, err := os.Stat(newDir); os.IsNotExist(err) {
        os.Rename(oldDir, newDir)
    }
}
```

- [ ] **Step 4: Update main.go version print**

Change `"tui-ssm %s\n"` to `"tui-aws %s\n"`.

- [ ] **Step 5: Run tests**

```bash
go test ./... -v
```

Expected: All PASS.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor: update config/store paths to ~/.tui-aws/"
```

---

### Task 3: Create TabModel Interface and SharedState

**Files:**
- Create: `internal/ui/tab.go`

- [ ] **Step 1: Create tab.go with interface and shared types**

```go
// internal/ui/tab.go
package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"tui-aws/internal/aws"
	"tui-aws/internal/config"
	"tui-aws/internal/store"
)

type TabModel interface {
	Init(shared *SharedState) tea.Cmd
	Update(msg tea.Msg, shared *SharedState) (TabModel, tea.Cmd)
	View(shared *SharedState) string
	ShortHelp() string
}

type TabID int

const (
	TabEC2 TabID = iota
	TabVPC
	TabSubnet
	TabRoutes
	TabSG
	TabTroubleshoot
	TabCount // sentinel for counting
)

func (t TabID) Label() string {
	switch t {
	case TabEC2:
		return "EC2"
	case TabVPC:
		return "VPC"
	case TabSubnet:
		return "Subnets"
	case TabRoutes:
		return "Routes"
	case TabSG:
		return "SG"
	case TabTroubleshoot:
		return "Check"
	default:
		return "?"
	}
}

type SharedState struct {
	Profile   string
	Region    string
	Profiles  []string
	Cfg       config.Config
	Favorites *store.Favorites
	History   *store.History
	Width     int
	Height    int
	Cache     map[string]CachedData
}

type CachedData struct {
	Data      any
	FetchedAt time.Time
}

const CacheTTL = 30 * time.Second

func (s *SharedState) GetCache(key string) (any, bool) {
	cd, ok := s.Cache[key]
	if !ok {
		return nil, false
	}
	if time.Since(cd.FetchedAt) > CacheTTL {
		return cd.Data, false // stale but return data for background refresh
	}
	return cd.Data, true
}

func (s *SharedState) SetCache(key string, data any) {
	s.Cache[key] = CachedData{Data: data, FetchedAt: time.Now()}
}

func (s *SharedState) ClearCache(keys ...string) {
	if len(keys) == 0 {
		s.Cache = map[string]CachedData{}
		return
	}
	for _, k := range keys {
		delete(s.Cache, k)
	}
}

// NavigateToTab is a message sent to RootModel to switch tabs with optional filter context.
type NavigateToTab struct {
	Tab      TabID
	FilterID string // e.g., VPC ID to auto-filter in target tab
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: Success (tab.go compiles alongside existing code).

- [ ] **Step 3: Commit**

```bash
git add internal/ui/tab.go && git commit -m "feat: add TabModel interface and SharedState"
```

---

### Task 4: Create Shared UI Components

**Files:**
- Create: `internal/ui/shared/styles.go`
- Create: `internal/ui/shared/table.go`
- Create: `internal/ui/shared/overlay.go`
- Create: `internal/ui/shared/selector.go`

- [ ] **Step 1: Create shared/styles.go**

Move all style variables and `StateStyle()` from `internal/ui/styles.go` to `internal/ui/shared/styles.go`. Change `package ui` to `package shared`.

- [ ] **Step 2: Create shared/table.go**

Move `Column`, `renderRow` function from `internal/ui/table.go` to `internal/ui/shared/table.go`. This is the generic table renderer that all tabs will reuse. Change package to `shared`. Update imports (`lipgloss`, `ansi`).

```go
package shared

// Column, renderRow (generic — no EC2-specific logic)
```

- [ ] **Step 3: Create shared/overlay.go**

Extract `OverlayStyle` and a helper function for rendering overlay boxes:

```go
package shared

func RenderOverlay(content string) string {
    return OverlayStyle.Render(content)
}
```

- [ ] **Step 4: Create shared/selector.go**

Move `SelectorModel` (NewSelector, MoveUp, MoveDown, Selected, Render) from `internal/ui/selector.go` to `internal/ui/shared/selector.go`. Change package to `shared`. Update style references to use `shared.OverlayStyle`.

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: Success (shared/ compiles, existing ui/ still works with its own copies temporarily).

- [ ] **Step 6: Commit**

```bash
git add internal/ui/shared/ && git commit -m "feat: create shared UI components (styles, table, overlay, selector)"
```

---

### Task 5: Create EC2 Tab SubModel

This is the largest task — extract EC2-specific logic from `internal/ui/model.go` into `internal/ui/tab_ec2/`.

**Files:**
- Create: `internal/ui/tab_ec2/model.go`
- Create: `internal/ui/tab_ec2/table.go`
- Create: `internal/ui/tab_ec2/actions.go`
- Create: `internal/ui/tab_ec2/search.go`
- Create: `internal/ui/tab_ec2/filter.go`

- [ ] **Step 1: Create tab_ec2/search.go**

Move `SearchModel` from `internal/ui/search.go`. Change package to `tab_ec2`. Update style references to `shared.SearchPromptStyle`.

- [ ] **Step 2: Create tab_ec2/filter.go**

Move `FilterModel`, `NewFilterModel` from `internal/ui/filter.go`. Change package to `tab_ec2`. Update style references.

- [ ] **Step 3: Create tab_ec2/table.go**

Move EC2-specific column definitions (`DefaultColumns`, `CompactColumns`, `ColumnsForWidth`), `cellValue`, `cellStyle`, `SortInstances`, `FilterBySearch`, `FilterByState`, and `RenderTable` from `internal/ui/table.go`. Change package to `tab_ec2`. Use `shared.Column` and `shared.RenderRow` for the generic parts.

- [ ] **Step 4: Create tab_ec2/actions.go**

Move `ActionMenuModel`, `NewActionMenu`, `RenderSecurityGroups`, `RenderInstanceDetail` from `internal/ui/actionmenu.go`. Change package to `tab_ec2`. Add `NavigateToTab` returns for "Go to VPC" and "Go to Subnet" actions (these will be no-ops until Phase 1, but the message type exists from Task 3).

- [ ] **Step 5: Create tab_ec2/model.go**

This is the core extraction. Create `EC2Model` that implements `TabModel`:

```go
package tab_ec2

import (
    tea "charm.land/bubbletea/v2"
    "tui-aws/internal/ui"
    // ...
)

type EC2Model struct {
    // Data
    instances []aws.Instance
    filtered  []aws.Instance
    cursor    int
    loading   bool
    err       error

    // UI Components
    search      SearchModel
    filter      FilterModel
    actionMenu  ActionMenuModel
    showDetail  string
    portForward PortForwardModel

    // Sort
    sortBy    string
    sortOrder string
    sortIdx   int

    // Internal view state (search/filter/action overlays)
    viewState viewState
}

// Implement TabModel interface
func (m EC2Model) Init(shared *ui.SharedState) tea.Cmd { ... }
func (m EC2Model) Update(msg tea.Msg, shared *ui.SharedState) (ui.TabModel, tea.Cmd) { ... }
func (m EC2Model) View(shared *ui.SharedState) string { ... }
func (m EC2Model) ShortHelp() string { ... }
```

Move all update handlers (updateTable, updateSearch, updateFilter, updateProfileSelect, updateRegionSelect, updatePortForward, updateActionMenu), loadInstances, startSSMSession, startPortForward, applyFilters from `model.go`.

Key changes:
- `m.profile` / `m.region` → `shared.Profile` / `shared.Region`
- `m.width` / `m.height` → `shared.Width` / `shared.Height`
- `m.favorites` / `m.history` → `shared.Favorites` / `shared.History`
- `m.profiles` → `shared.Profiles`
- Profile/region selection is handled by RootModel (not EC2 tab) — remove from EC2 tab
- `ssmExecCmd` and `InterruptFilter` stay in `internal/ui/` (root level, used by main.go)

- [ ] **Step 6: Verify build**

```bash
go build ./...
```

May have compile errors from import cycles or missing references. Fix iteratively.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/tab_ec2/ && git commit -m "feat: extract EC2 tab submodel from model.go"
```

---

### Task 6: Create RootModel

**Files:**
- Create: `internal/ui/root.go`

- [ ] **Step 1: Create root.go**

```go
package ui

import (
    "fmt"
    "strings"

    tea "charm.land/bubbletea/v2"
    "charm.land/lipgloss/v2"
    "tui-aws/internal/ui/shared"
    "tui-aws/internal/ui/tab_ec2"
    // placeholder tabs import later
)

type RootModel struct {
    shared    SharedState
    activeTab TabID
    tabs      [int(TabCount)]TabModel

    // Global overlays (profile/region selector)
    profSelect   shared.SelectorModel
    regionSelect shared.SelectorModel
    globalView   globalViewState // none, profileSelect, regionSelect
}

type globalViewState int
const (
    globalNone globalViewState = iota
    globalProfileSelect
    globalRegionSelect
)

func NewRootModel(cfg config.Config, profiles []string, favs *store.Favorites, hist *store.History) RootModel {
    s := SharedState{
        Profile:   cfg.DefaultProfile,
        Region:    cfg.DefaultRegion,
        Profiles:  profiles,
        Cfg:       cfg,
        Favorites: favs,
        History:   hist,
        Cache:     map[string]CachedData{},
    }
    var tabs [int(TabCount)]TabModel
    tabs[TabEC2] = tab_ec2.New(cfg)
    // Placeholder tabs for Phase 1-3
    tabs[TabVPC] = NewPlaceholderTab("VPC")
    tabs[TabSubnet] = NewPlaceholderTab("Subnets")
    tabs[TabRoutes] = NewPlaceholderTab("Routes")
    tabs[TabSG] = NewPlaceholderTab("Security Groups")
    tabs[TabTroubleshoot] = NewPlaceholderTab("Troubleshoot")

    return RootModel{shared: s, tabs: tabs}
}

func (m RootModel) Init() tea.Cmd {
    return m.tabs[m.activeTab].Init(&m.shared)
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle global keys first (tab switching, profile/region, quit)
    // Then delegate to active tab
    ...
}

func (m RootModel) View() tea.View {
    var sections []string
    sections = append(sections, m.renderTabBar())
    sections = append(sections, m.tabs[m.activeTab].View(&m.shared))
    sections = append(sections, m.renderHelpBar())
    // Global overlays (profile/region selector)
    ...
    v := tea.NewView(strings.Join(sections, "\n"))
    v.AltScreen = true
    return v
}
```

- [ ] **Step 2: Create PlaceholderTab**

```go
// internal/ui/placeholder.go
package ui

type PlaceholderTab struct {
    name string
}

func NewPlaceholderTab(name string) PlaceholderTab { return PlaceholderTab{name: name} }
func (p PlaceholderTab) Init(shared *SharedState) tea.Cmd { return nil }
func (p PlaceholderTab) Update(msg tea.Msg, shared *SharedState) (TabModel, tea.Cmd) { return p, nil }
func (p PlaceholderTab) View(shared *SharedState) string {
    return fmt.Sprintf("\n  %s — Coming soon\n", p.name)
}
func (p PlaceholderTab) ShortHelp() string { return "" }
```

- [ ] **Step 3: Implement tab bar rendering**

```go
func (m RootModel) renderTabBar() string {
    var parts []string
    for i := 0; i < int(TabCount); i++ {
        tid := TabID(i)
        label := fmt.Sprintf(" %d:%s ", i+1, tid.Label())
        if tid == m.activeTab {
            label = shared.StatusKeyStyle.Render(label)
        }
        parts = append(parts, label)
    }
    tabBar := strings.Join(parts, "")
    profile := shared.StatusKeyStyle.Render("Profile: ") + m.shared.Profile
    region := shared.StatusKeyStyle.Render("Region: ") + m.shared.Region
    right := fmt.Sprintf("  %s  ┊  %s", profile, region)
    return shared.StatusBarStyle.Width(m.shared.Width).Render(tabBar + right)
}
```

- [ ] **Step 4: Implement global key handling in Update**

Global keys handled before tab delegation:
- `1`-`6`: tab switch (only when no overlay active)
- `Tab`/`shift+tab`: next/prev tab
- `p`: profile selector (global overlay)
- `r`: region selector (global overlay)
- `q`/`ctrl+c`: quit
- `R`: delegate to active tab (tab handles refresh)
- `NavigateToTab` message: switch tab with filter context

All other messages: delegate to `m.tabs[m.activeTab].Update(msg, &m.shared)`.

- [ ] **Step 5: Update main.go**

Change `ui.NewModel(...)` to `ui.NewRootModel(...)` and `ui.InterruptFilter` stays the same.

- [ ] **Step 6: Verify build**

```bash
go build -o tui-aws ./main.go
```

- [ ] **Step 7: Commit**

```bash
git add internal/ui/root.go internal/ui/placeholder.go main.go && git commit -m "feat: create RootModel with tab switching and placeholder tabs"
```

---

### Task 7: Delete Old Files and Wire Everything

**Files:**
- Delete: `internal/ui/model.go`
- Delete: `internal/ui/styles.go`
- Delete: `internal/ui/table.go`
- Delete: `internal/ui/statusbar.go`
- Delete: `internal/ui/helpbar.go`
- Delete: `internal/ui/search.go`
- Delete: `internal/ui/filter.go`
- Delete: `internal/ui/selector.go`
- Delete: `internal/ui/actionmenu.go`

- [ ] **Step 1: Remove old ui/ files**

```bash
rm internal/ui/model.go internal/ui/styles.go internal/ui/table.go \
   internal/ui/statusbar.go internal/ui/helpbar.go internal/ui/search.go \
   internal/ui/filter.go internal/ui/selector.go internal/ui/actionmenu.go
```

Keep: `internal/ui/root.go`, `internal/ui/tab.go`, `internal/ui/placeholder.go`
Keep: `internal/ui/shared/`, `internal/ui/tab_ec2/`

- [ ] **Step 2: Fix all import errors**

Build and fix iteratively:

```bash
go build ./... 2>&1 | head -20
```

Common fixes:
- `ui.InterruptFilter` → keep in `internal/ui/root.go` (exported)
- `ui.NewModel` → `ui.NewRootModel`
- Test files referencing old packages → update imports

- [ ] **Step 3: Update table_test.go**

Move `internal/ui/table_test.go` to `internal/ui/tab_ec2/table_test.go`. Update package name and imports.

- [ ] **Step 4: Run all tests**

```bash
go test ./... -v
```

Expected: All PASS.

- [ ] **Step 5: Build and smoke test**

```bash
go build -o tui-aws ./main.go && ./tui-aws --version
```

Expected: `tui-aws dev`

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor: remove old ui/ files, complete tab architecture migration"
```

---

### Task 8: Verify Full Functionality

- [ ] **Step 1: Run go vet**

```bash
go vet ./...
```

Expected: No issues.

- [ ] **Step 2: Run all tests**

```bash
go test ./... -v -count=1
```

Expected: All PASS.

- [ ] **Step 3: Manual smoke test**

Launch `./tui-aws` and verify:
- EC2 tab shows instances (tab 1)
- Tab switching works (keys 1-6, Tab/Shift+Tab)
- Placeholder tabs show "Coming soon"
- Profile/region selection works (p/r keys)
- Search, filter, sort, favorites work
- Enter → action menu → SSM session works
- Enter → action menu → Instance Details shows VPC/Subnet info
- Port forwarding overlay works
- ESC closes all overlays
- q quits

- [ ] **Step 4: Build cross-platform**

```bash
make build-all && ls -la dist/
```

Expected: 4 binaries named `tui-aws-*`.

- [ ] **Step 5: Final commit**

```bash
git add -A && git commit -m "chore: Phase 0 complete — tui-aws tab architecture verified"
```

---

## Summary

| Task | Description | Key Files |
|------|-------------|-----------|
| 1 | Rename module tui-ssm → tui-aws | go.mod, Makefile, all .go imports |
| 2 | Update config/store paths to ~/.tui-aws/ | config.go, favorites.go, history.go, main.go |
| 3 | Create TabModel interface + SharedState | ui/tab.go |
| 4 | Create shared UI components | ui/shared/ (styles, table, overlay, selector) |
| 5 | Extract EC2 tab submodel | ui/tab_ec2/ (model, table, actions, search, filter) |
| 6 | Create RootModel with tab switching | ui/root.go, ui/placeholder.go, main.go |
| 7 | Delete old files, wire imports | Remove 9 old files, fix imports |
| 8 | Verify full functionality | Tests, vet, manual smoke test, cross-build |

## Next Steps

After Phase 0 is complete and verified:
- Create `docs/superpowers/plans/2026-04-XX-tui-aws-phase1-vpc-subnet.md` for Phase 1
- Phase 1 adds: `aws/vpc.go`, `aws/subnet.go`, `tab_vpc/`, `tab_subnet/`, EC2 drilldown actions
