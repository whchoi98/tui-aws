// internal/config/config_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultRegion != "us-east-1" {
		t.Errorf("expected default region us-east-1, got %s", cfg.DefaultRegion)
	}
	if cfg.DefaultProfile != "default" {
		t.Errorf("expected default profile 'default', got %s", cfg.DefaultProfile)
	}
	if cfg.Table.SortBy != "name" {
		t.Errorf("expected sort by name, got %s", cfg.Table.SortBy)
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	cfg.DefaultRegion = "ap-northeast-2"

	if err := Save(cfg, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.DefaultRegion != "ap-northeast-2" {
		t.Errorf("expected ap-northeast-2, got %s", loaded.DefaultRegion)
	}
}

func TestLoadNonExistentReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load should not fail for missing file: %v", err)
	}
	if cfg.DefaultProfile != "default" {
		t.Errorf("expected default config, got profile %s", cfg.DefaultProfile)
	}
}
