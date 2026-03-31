// internal/config/config.go
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type TableConfig struct {
	VisibleColumns []string `json:"visible_columns"`
	SortBy         string   `json:"sort_by"`
	SortOrder      string   `json:"sort_order"`
}

type Config struct {
	DefaultProfile         string      `json:"default_profile"`
	DefaultRegion          string      `json:"default_region"`
	RefreshIntervalSeconds int         `json:"refresh_interval_seconds"`
	Table                  TableConfig `json:"table"`
}

func DefaultConfig() Config {
	return Config{
		DefaultProfile:         "default",
		DefaultRegion:          "us-east-1",
		RefreshIntervalSeconds: 0,
		Table: TableConfig{
			VisibleColumns: []string{"name", "id", "state", "private_ip", "type", "az"},
			SortBy:         "name",
			SortOrder:      "asc",
		},
	}
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tui-ssm")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(cfg Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
