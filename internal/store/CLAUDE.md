# Store Module

## Role
Persist favorites and session history as JSON files under `~/.tui-aws/`.

## Key Files
- `favorites.go` — Favorites CRUD (Add/Remove/IsFavorite), FavoritesPath()
- `history.go` — History FIFO with MaxEntries cap, HistoryPath()
- `*_test.go` — Tests for add/remove, dedup, max entries, persistence

## Rules
- Favorites keyed by (InstanceID, Profile, Region) tuple
- History uses FIFO eviction when exceeding MaxEntries (default 100)
- Missing files return empty structs (no error)
- Pointer receivers on Favorites/History (shared state across model)
