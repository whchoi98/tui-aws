# UI Module

## Role
Bubble Tea v2 TUI implementation: tab-based architecture with root model, shared components, and per-tab packages.

## Architecture
```
internal/ui/
├── root.go          — RootModel (tea.Model), tab switching, global overlays (profile/region), ssmExecCmd, InterruptFilter
├── tab.go           — Re-exports from shared: TabModel, TabID, SharedState, NavigateToTab
├── placeholder.go   — PlaceholderTab for coming-soon tabs (VPC, Subnet, Routes, SG, Check)
├── shared/
│   ├── tab.go       — TabModel interface, TabID enum, SharedState, CachedData, NavigateToTab
│   ├── styles.go    — All Lip Gloss Gruvbox styles + StateStyle helper
│   ├── table.go     — Column type + RenderRow (generic table renderer)
│   ├── overlay.go   — RenderOverlay, PlaceOverlay helpers
│   └── selector.go  — SelectorModel (generic list picker)
└── tab_ec2/
    ├── model.go     — EC2Model implementing TabModel, all update handlers
    ├── table.go     — EC2 columns, cellValue, cellStyle, SortInstances, FilterBySearch/State, RenderTable
    ├── actions.go   — ActionMenuModel, PortForwardModel, RenderSecurityGroups, RenderInstanceDetail
    ├── search.go    — SearchModel with Insert/Backspace/Clear
    ├── filter.go    — FilterModel with state toggle checkboxes
    └── table_test.go — TestSortInstances, TestFilterInstancesBySearch, TestFilterInstancesByState
```

## Key Design
- **RootModel** owns SharedState, handles global keys (q, p, r, 1-6, Tab/Shift+Tab), delegates to active tab
- **TabModel interface** defined in shared/: Init, Update, View, ShortHelp
- **EC2Model** sends SSMExecRequest messages; RootModel intercepts and runs tea.Exec
- **SharedState** lives in shared/ to avoid circular imports between ui and tab_ec2
- **ui/tab.go** re-exports shared types so external callers (main.go) use the ui package

## Rules
- EC2Model uses pointer receiver (state mutations); RootModel uses value receiver (Bubble Tea requirement)
- ssmExecCmd wraps exec.Cmd with `stty sane` + stdin flush after SSM session
- InterruptFilter blocks OS SIGINT (raw mode delivers Ctrl+C as KeyPressMsg)
- View() always sets AltScreen = true (Bubble Tea v2 API)
- Profile/region selectors are global overlays in RootModel, not in EC2 tab
- SSM session history is recorded in RootModel on SSMSessionDoneMsg, then forwarded to EC2 tab
