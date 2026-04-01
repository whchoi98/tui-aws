# Config Module

## Role
Load, save, and manage user configuration from `~/.tui-aws/config.json`. Provides default values for new installations.

## Key Files
- `config.go` — Config struct, Load/Save/DefaultConfig, Dir/Path helpers
- `config_test.go` — Tests for load, save, default, and missing file scenarios

## Rules
- Config file is JSON with `json:"..."` struct tags
- Missing config file returns DefaultConfig (no error)
- Dir() returns `~/.tui-aws/`, Path() returns `~/.tui-aws/config.json`
- Save creates parent directories automatically
