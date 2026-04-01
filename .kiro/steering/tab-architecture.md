# Tab Architecture

- Every tab implements `TabModel` interface from `internal/ui/shared/tab.go`
- Tab packages live under `internal/ui/tab_<name>/`
- Each tab package contains: model.go (state + Update), table.go (columns + rendering), detail.go (action overlays)
- SharedState is the single source of truth for profile, region, clients, cache, dimensions
- Tab navigation: `[`/`]` keys, `Tab`/`Shift+Tab` (except Check tab when editing)
- Lazy loading: fetch on first activation, 30s cache TTL
- Cache invalidation: profile/region change clears all; `R` key clears active tab
- New tab checklist:
  1. Add TabID constant in `shared/tab.go`
  2. Create `tab_<name>/` package
  3. Register in `root.go` tab initialization
  4. Add tab label in tab bar rendering
