# UI Module

## Role
Bubble Tea v2 TUI implementation: root model, state machine, table rendering, overlays, and styling.

## Key Files
- `model.go` — Root Model, Init/Update/View, state machine dispatch, SSM exec handling
- `table.go` — Table rendering, column definitions, SortInstances, FilterBySearch/State
- `styles.go` — Lip Gloss Gruvbox color definitions, StateStyle helper
- `statusbar.go` — Top status bar (profile/region/filter/count)
- `helpbar.go` — Bottom key binding hints, ViewState enum
- `search.go` — SearchModel with Insert/Backspace/Clear
- `filter.go` — FilterModel with state toggle checkboxes
- `selector.go` — SelectorModel for profile/region list picker

## Rules
- Model uses value receiver (Bubble Tea requirement), stores use pointer fields
- ssmExecCmd wraps exec.Cmd with `stty sane` + stdin flush after SSM session
- InterruptFilter blocks OS SIGINT (raw mode delivers Ctrl+C as KeyPressMsg)
- View() always sets AltScreen = true (Bubble Tea v2 API — no WithAltScreen option)
- SSM session errors are displayed to user, only successful sessions recorded in history
