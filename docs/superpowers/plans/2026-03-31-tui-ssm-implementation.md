# TUI-SSM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go TUI tool that lists EC2 instances and connects via AWS Session Manager with keyboard navigation, search, filtering, favorites, and port forwarding.

**Architecture:** Single-view Bubble Tea v2 app with Elm architecture (Model-View-Update). AWS SDK Go v2 for EC2/SSM queries, `os/exec` for SSM sessions via `tea.ExecProcess()`. Local JSON files for config, favorites, and history.

**Tech Stack:** Go, Bubble Tea v2 (`charm.land/bubbletea/v2`), Lip Gloss v2 (`charm.land/lipgloss/v2`), Bubbles (`charm.land/x/bubbles`), aws-sdk-go-v2

---

## File Structure

```
tui-ssm/
├── main.go                       # CLI entry point, prerequisite checks, program launch
├── go.mod
├── Makefile
├── internal/
│   ├── config/
│   │   └── config.go             # Load/save/default config (~/.tui-ssm/config.json)
│   ├── store/
│   │   ├── favorites.go          # Favorites CRUD (~/.tui-ssm/favorites.json)
│   │   └── history.go            # Session history CRUD (~/.tui-ssm/history.json)
│   ├── aws/
│   │   ├── profile.go            # Parse AWS profiles from ~/.aws/credentials + config
│   │   ├── session.go            # AWS SDK client factory (profile + region)
│   │   ├── ec2.go                # DescribeInstances → []Instance
│   │   └── ssm.go                # SSM prereq check, DescribeInstanceInformation
│   └── ui/
│       ├── styles.go             # Lip Gloss style definitions
│       ├── model.go              # Root model, state machine, Update dispatch
│       ├── table.go              # EC2 table rendering + sorting
│       ├── statusbar.go          # Top bar (profile/region/filter indicator)
│       ├── helpbar.go            # Bottom bar (key binding hints)
│       ├── search.go             # Search input component
│       ├── filter.go             # Filter overlay (state/tag toggles)
│       └── selector.go           # List selector overlay (profile/region picker)
```

---

### Task 1: Project Scaffolding & Configuration

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Initialize Go module and install dependencies**

```bash
cd /home/ec2-user/my-project/tui-ssm
go mod init tui-ssm
go get charm.land/bubbletea/v2@latest
go get charm.land/lipgloss/v2@latest
go get charm.land/x/bubbles@latest
go get github.com/aws/aws-sdk-go-v2@latest
go get github.com/aws/aws-sdk-go-v2/config@latest
go get github.com/aws/aws-sdk-go-v2/service/ec2@latest
go get github.com/aws/aws-sdk-go-v2/service/ssm@latest
go get github.com/aws/aws-sdk-go-v2/service/sts@latest
```

- [ ] **Step 2: Create Makefile**

```makefile
.PHONY: build build-all install clean test

BINARY := tui-ssm
VERSION := 0.1.0

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./main.go

build-all:
	GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-linux-amd64 ./main.go
	GOOS=linux   GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-linux-arm64 ./main.go
	GOOS=darwin  GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-arm64 ./main.go
	GOOS=darwin  GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY)-darwin-amd64 ./main.go

install:
	go build -ldflags "-X main.version=$(VERSION)" -o $(GOPATH)/bin/$(BINARY) ./main.go

test:
	go test ./... -v

clean:
	rm -f $(BINARY)
	rm -rf dist/
```

- [ ] **Step 3: Write config test**

```go
// internal/config/config_test.go
package config

import (
	"os"
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
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/config/ -v
```

Expected: FAIL — package not found.

- [ ] **Step 5: Implement config.go**

```go
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
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/config/ -v
```

Expected: PASS (3 tests).

- [ ] **Step 7: Create minimal main.go**

```go
// main.go
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("tui-ssm %s\n", version)
		os.Exit(0)
	}
	fmt.Println("tui-ssm: starting...")
}
```

- [ ] **Step 8: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build -o tui-ssm ./main.go && ./tui-ssm --version
```

Expected: `tui-ssm dev`

- [ ] **Step 9: Commit**

```bash
git init
git add go.mod go.sum Makefile main.go internal/config/
git commit -m "feat: project scaffolding with config layer"
```

---

### Task 2: Local Store — Favorites & History

**Files:**
- Create: `internal/store/favorites.go`
- Create: `internal/store/favorites_test.go`
- Create: `internal/store/history.go`
- Create: `internal/store/history_test.go`

- [ ] **Step 1: Write favorites test**

```go
// internal/store/favorites_test.go
package store

import (
	"path/filepath"
	"testing"
)

func TestFavoritesAddRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	f, err := LoadFavorites(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	fav := Favorite{
		InstanceID: "i-abc123",
		Profile:    "prod",
		Region:     "ap-northeast-2",
		Alias:      "web-server-1",
	}

	f.Add(fav)
	if len(f.Items) != 1 {
		t.Fatalf("expected 1 favorite, got %d", len(f.Items))
	}

	if !f.IsFavorite("i-abc123", "prod", "ap-northeast-2") {
		t.Error("expected i-abc123 to be a favorite")
	}

	f.Remove("i-abc123", "prod", "ap-northeast-2")
	if len(f.Items) != 0 {
		t.Fatalf("expected 0 favorites after remove, got %d", len(f.Items))
	}

	if err := f.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadFavorites(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if len(loaded.Items) != 0 {
		t.Errorf("expected 0 favorites after reload, got %d", len(loaded.Items))
	}
}

func TestFavoritesNoDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	f, _ := LoadFavorites(path)

	fav := Favorite{InstanceID: "i-abc123", Profile: "prod", Region: "us-east-1", Alias: "web"}
	f.Add(fav)
	f.Add(fav)
	if len(f.Items) != 1 {
		t.Errorf("expected 1 favorite (no dup), got %d", len(f.Items))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/store/ -run TestFavorites -v
```

Expected: FAIL — package not found.

- [ ] **Step 3: Implement favorites.go**

```go
// internal/store/favorites.go
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Favorite struct {
	InstanceID string    `json:"instance_id"`
	Profile    string    `json:"profile"`
	Region     string    `json:"region"`
	Alias      string    `json:"alias"`
	AddedAt    time.Time `json:"added_at"`
}

type Favorites struct {
	Items []Favorite `json:"favorites"`
}

func LoadFavorites(path string) (*Favorites, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Favorites{}, nil
		}
		return nil, err
	}
	var f Favorites
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *Favorites) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (f *Favorites) Add(fav Favorite) {
	if f.IsFavorite(fav.InstanceID, fav.Profile, fav.Region) {
		return
	}
	fav.AddedAt = time.Now()
	f.Items = append(f.Items, fav)
}

func (f *Favorites) Remove(instanceID, profile, region string) {
	items := make([]Favorite, 0, len(f.Items))
	for _, item := range f.Items {
		if item.InstanceID == instanceID && item.Profile == profile && item.Region == region {
			continue
		}
		items = append(items, item)
	}
	f.Items = items
}

func (f *Favorites) IsFavorite(instanceID, profile, region string) bool {
	for _, item := range f.Items {
		if item.InstanceID == instanceID && item.Profile == profile && item.Region == region {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run favorites tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/store/ -run TestFavorites -v
```

Expected: PASS (2 tests).

- [ ] **Step 5: Write history test**

```go
// internal/store/history_test.go
package store

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestHistoryAddAndRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	h, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	h.Add(HistoryEntry{
		InstanceID: "i-abc123",
		Profile:    "prod",
		Region:     "us-east-1",
		Alias:      "web-1",
		Type:       "session",
	})
	h.Add(HistoryEntry{
		InstanceID: "i-def456",
		Profile:    "prod",
		Region:     "us-east-1",
		Alias:      "web-2",
		Type:       "session",
	})

	if len(h.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(h.Sessions))
	}

	// Most recent should be last added
	if h.Sessions[len(h.Sessions)-1].InstanceID != "i-def456" {
		t.Error("expected most recent to be i-def456")
	}

	if err := h.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadHistory(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if len(loaded.Sessions) != 2 {
		t.Errorf("expected 2 sessions after reload, got %d", len(loaded.Sessions))
	}
}

func TestHistoryMaxEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	h, _ := LoadHistory(path)
	h.MaxEntries = 3

	for i := 0; i < 5; i++ {
		h.Add(HistoryEntry{
			InstanceID: fmt.Sprintf("i-%d", i),
			Profile:    "prod",
			Region:     "us-east-1",
			Alias:      fmt.Sprintf("srv-%d", i),
			Type:       "session",
		})
	}

	if len(h.Sessions) != 3 {
		t.Errorf("expected 3 sessions (max), got %d", len(h.Sessions))
	}
	// Oldest should have been evicted
	if h.Sessions[0].InstanceID != "i-2" {
		t.Errorf("expected oldest to be i-2, got %s", h.Sessions[0].InstanceID)
	}
}

func TestHistoryIsRecent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	h, _ := LoadHistory(path)
	h.Add(HistoryEntry{
		InstanceID: "i-abc123",
		Profile:    "prod",
		Region:     "us-east-1",
		Alias:      "web-1",
		Type:       "session",
	})

	if !h.IsRecent("i-abc123", "prod", "us-east-1") {
		t.Error("expected i-abc123 to be in recent history")
	}
	if h.IsRecent("i-notexist", "prod", "us-east-1") {
		t.Error("expected i-notexist to not be in recent history")
	}
}
```

- [ ] **Step 6: Run history test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/store/ -run TestHistory -v
```

Expected: FAIL.

- [ ] **Step 7: Implement history.go**

```go
// internal/store/history.go
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type HistoryEntry struct {
	InstanceID  string    `json:"instance_id"`
	Profile     string    `json:"profile"`
	Region      string    `json:"region"`
	Alias       string    `json:"alias"`
	Type        string    `json:"type"` // "session" or "port_forward"
	ConnectedAt time.Time `json:"connected_at"`
}

type History struct {
	Sessions   []HistoryEntry `json:"sessions"`
	MaxEntries int            `json:"max_entries"`
}

func LoadHistory(path string) (*History, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &History{MaxEntries: 100}, nil
		}
		return nil, err
	}
	h := History{MaxEntries: 100}
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	if h.MaxEntries == 0 {
		h.MaxEntries = 100
	}
	return &h, nil
}

func (h *History) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (h *History) Add(entry HistoryEntry) {
	entry.ConnectedAt = time.Now()
	h.Sessions = append(h.Sessions, entry)
	if len(h.Sessions) > h.MaxEntries {
		h.Sessions = h.Sessions[len(h.Sessions)-h.MaxEntries:]
	}
}

func (h *History) IsRecent(instanceID, profile, region string) bool {
	for _, s := range h.Sessions {
		if s.InstanceID == instanceID && s.Profile == profile && s.Region == region {
			return true
		}
	}
	return false
}
```

- [ ] **Step 8: Run all store tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/store/ -v
```

Expected: PASS (5 tests).

- [ ] **Step 9: Commit**

```bash
git add internal/store/
git commit -m "feat: add favorites and history store"
```

---

### Task 3: AWS Profile Parsing & Session Management

**Files:**
- Create: `internal/aws/profile.go`
- Create: `internal/aws/profile_test.go`
- Create: `internal/aws/session.go`

- [ ] **Step 1: Write profile parsing test**

```go
// internal/aws/profile_test.go
package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProfiles(t *testing.T) {
	dir := t.TempDir()

	credPath := filepath.Join(dir, "credentials")
	os.WriteFile(credPath, []byte(`[default]
aws_access_key_id = AKIA_DEFAULT

[production]
aws_access_key_id = AKIA_PROD
`), 0o644)

	configPath := filepath.Join(dir, "config")
	os.WriteFile(configPath, []byte(`[default]
region = us-east-1

[profile staging]
region = eu-west-1
`), 0o644)

	profiles := ParseProfiles(credPath, configPath)

	if len(profiles) < 3 {
		t.Fatalf("expected at least 3 profiles, got %d: %v", len(profiles), profiles)
	}

	found := map[string]bool{}
	for _, p := range profiles {
		found[p] = true
	}
	for _, want := range []string{"default", "production", "staging"} {
		if !found[want] {
			t.Errorf("expected profile %q in list %v", want, profiles)
		}
	}
}

func TestParseProfilesMissingFiles(t *testing.T) {
	profiles := ParseProfiles("/nonexistent/creds", "/nonexistent/config")
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles for missing files, got %d", len(profiles))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run TestParseProfile -v
```

Expected: FAIL.

- [ ] **Step 3: Implement profile.go**

```go
// internal/aws/profile.go
package aws

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DefaultCredentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

func ParseProfiles(credentialsPath, configPath string) []string {
	seen := map[string]bool{}

	// Parse credentials file: sections are [profile_name]
	for _, name := range parseSections(credentialsPath, false) {
		seen[name] = true
	}

	// Parse config file: sections are [profile profile_name] or [default]
	for _, name := range parseSections(configPath, true) {
		seen[name] = true
	}

	profiles := make([]string, 0, len(seen))
	for name := range seen {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	return profiles
}

func parseSections(path string, isConfig bool) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		name := line[1 : len(line)-1]
		if isConfig {
			// config file uses [profile xxx] for named profiles, [default] for default
			if strings.HasPrefix(name, "profile ") {
				name = strings.TrimPrefix(name, "profile ")
			}
		}
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
```

- [ ] **Step 4: Run profile tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run TestParseProfile -v
```

Expected: PASS (2 tests).

- [ ] **Step 5: Implement session.go**

```go
// internal/aws/session.go
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type Clients struct {
	EC2     *ec2.Client
	SSM     *ssm.Client
	STS     *sts.Client
	Profile string
	Region  string
}

func NewClients(ctx context.Context, profile, region string) (*Clients, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if profile != "" && profile != "default" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Clients{
		EC2:     ec2.NewFromConfig(cfg),
		SSM:     ssm.NewFromConfig(cfg),
		STS:     sts.NewFromConfig(cfg),
		Profile: profile,
		Region:  region,
	}, nil
}

func (c *Clients) ValidateCredentials(ctx context.Context) (string, error) {
	out, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Account), nil
}

func KnownRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
		"ap-southeast-1", "ap-southeast-2",
		"ap-south-1",
		"eu-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "eu-north-1",
		"sa-east-1",
		"ca-central-1",
		"me-south-1",
		"af-south-1",
	}
}
```

- [ ] **Step 6: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 7: Commit**

```bash
git add internal/aws/profile.go internal/aws/profile_test.go internal/aws/session.go
git commit -m "feat: AWS profile parsing and session management"
```

---

### Task 4: EC2 Instance Fetching

**Files:**
- Create: `internal/aws/ec2.go`
- Create: `internal/aws/ec2_test.go`

- [ ] **Step 1: Write EC2 data model and fetch test**

```go
// internal/aws/ec2_test.go
package aws

import (
	"testing"
	"time"
)

func TestInstanceDisplayName(t *testing.T) {
	inst := Instance{
		InstanceID: "i-abc123",
		Name:       "web-server-1",
	}
	if inst.DisplayName() != "web-server-1" {
		t.Errorf("expected web-server-1, got %s", inst.DisplayName())
	}

	noName := Instance{InstanceID: "i-def456"}
	if noName.DisplayName() != "i-def456" {
		t.Errorf("expected i-def456, got %s", noName.DisplayName())
	}
}

func TestInstanceStateIcon(t *testing.T) {
	tests := []struct {
		state string
		icon  string
	}{
		{"running", "●"},
		{"stopped", "○"},
		{"pending", "◐"},
		{"stopping", "◑"},
		{"terminated", "✕"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		inst := Instance{State: tt.state}
		if got := inst.StateIcon(); got != tt.icon {
			t.Errorf("state %q: expected icon %q, got %q", tt.state, tt.icon, got)
		}
	}
}

func TestInstanceShortAZ(t *testing.T) {
	inst := Instance{AvailabilityZone: "ap-northeast-2a"}
	if got := inst.ShortAZ(); got != "2a" {
		t.Errorf("expected 2a, got %s", got)
	}
}

func TestInstanceLaunchTimeFormatted(t *testing.T) {
	inst := Instance{
		LaunchTime: time.Date(2026, 1, 15, 9, 30, 0, 0, time.UTC),
	}
	if got := inst.LaunchTimeFormatted(); got != "2026-01-15 09:30" {
		t.Errorf("expected '2026-01-15 09:30', got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run TestInstance -v
```

Expected: FAIL.

- [ ] **Step 3: Implement ec2.go**

```go
// internal/aws/ec2.go
package aws

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Instance struct {
	InstanceID       string
	Name             string
	State            string
	PrivateIP        string
	PublicIP         string
	InstanceType     string
	AvailabilityZone string
	Platform         string
	LaunchTime       time.Time
	SecurityGroups   []string
	KeyPair          string
	IAMRole          string
	SSMConnected     bool
}

func (i Instance) DisplayName() string {
	if i.Name != "" {
		return i.Name
	}
	return i.InstanceID
}

func (i Instance) StateIcon() string {
	switch i.State {
	case "running":
		return "●"
	case "stopped":
		return "○"
	case "pending":
		return "◐"
	case "stopping":
		return "◑"
	case "terminated":
		return "✕"
	default:
		return "?"
	}
}

func (i Instance) ShortAZ() string {
	// "ap-northeast-2a" → "2a"
	parts := strings.Split(i.AvailabilityZone, "-")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return i.AvailabilityZone
}

func (i Instance) LaunchTimeFormatted() string {
	if i.LaunchTime.IsZero() {
		return "-"
	}
	return i.LaunchTime.Format("2006-01-02 15:04")
}

func FetchInstances(ctx context.Context, client *ec2.Client) ([]Instance, error) {
	var instances []Instance
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				instances = append(instances, toInstance(inst))
			}
		}
	}
	return instances, nil
}

func toInstance(inst ec2types.Instance) Instance {
	i := Instance{
		InstanceID:       aws.ToString(inst.InstanceId),
		InstanceType:     string(inst.InstanceType),
		AvailabilityZone: aws.ToString(inst.Placement.AvailabilityZone),
		PrivateIP:        aws.ToString(inst.PrivateIpAddress),
		PublicIP:         aws.ToString(inst.PublicIpAddress),
		KeyPair:          aws.ToString(inst.KeyName),
	}

	if inst.State != nil {
		i.State = string(inst.State.Name)
	}

	if inst.LaunchTime != nil {
		i.LaunchTime = *inst.LaunchTime
	}

	if inst.PlatformDetails != nil {
		i.Platform = aws.ToString(inst.PlatformDetails)
	} else {
		i.Platform = "Linux"
	}

	for _, tag := range inst.Tags {
		if aws.ToString(tag.Key) == "Name" {
			i.Name = aws.ToString(tag.Value)
			break
		}
	}

	for _, sg := range inst.SecurityGroups {
		i.SecurityGroups = append(i.SecurityGroups, aws.ToString(sg.GroupName))
	}

	if inst.IamInstanceProfile != nil {
		arn := aws.ToString(inst.IamInstanceProfile.Arn)
		// Extract role name from ARN: arn:aws:iam::123:instance-profile/RoleName
		if parts := strings.Split(arn, "/"); len(parts) > 1 {
			i.IAMRole = parts[len(parts)-1]
		}
	}

	return i
}
```

- [ ] **Step 4: Run tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run TestInstance -v
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/aws/ec2.go internal/aws/ec2_test.go
git commit -m "feat: EC2 instance data model and fetching"
```

---

### Task 5: SSM Integration & Prerequisite Checks

**Files:**
- Create: `internal/aws/ssm.go`
- Create: `internal/aws/ssm_test.go`

- [ ] **Step 1: Write SSM prereq test**

```go
// internal/aws/ssm_test.go
package aws

import (
	"testing"
)

func TestBuildSSMCommand(t *testing.T) {
	args := BuildSSMSessionArgs("i-abc123", "production", "ap-northeast-2")
	expected := []string{
		"ssm", "start-session",
		"--target", "i-abc123",
		"--profile", "production",
		"--region", "ap-northeast-2",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, args[i])
		}
	}
}

func TestBuildSSMCommandDefaultProfile(t *testing.T) {
	args := BuildSSMSessionArgs("i-abc123", "default", "us-east-1")
	// default profile should not include --profile flag
	for _, arg := range args {
		if arg == "--profile" {
			t.Error("default profile should not include --profile flag")
		}
	}
}

func TestBuildPortForwardArgs(t *testing.T) {
	args := BuildPortForwardArgs("i-abc123", "production", "ap-northeast-2", "8080", "80")
	found := map[string]bool{}
	for _, a := range args {
		found[a] = true
	}
	if !found["AWS-StartPortForwardingSession"] {
		t.Error("expected document name in args")
	}
	if !found["--parameters"] {
		t.Error("expected --parameters in args")
	}
}

func TestCheckPrerequisites(t *testing.T) {
	// This tests the structure of prereq results, not actual binary existence
	results := CheckPrerequisites()
	if len(results) != 2 {
		t.Errorf("expected 2 prereq checks, got %d", len(results))
	}
	for _, r := range results {
		if r.Name == "" {
			t.Error("prereq name should not be empty")
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run TestBuild -v
```

Expected: FAIL.

- [ ] **Step 3: Implement ssm.go**

```go
// internal/aws/ssm.go
package aws

import (
	"context"
	"fmt"
	"os/exec"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type PrereqResult struct {
	Name    string
	OK      bool
	Message string
}

func CheckPrerequisites() []PrereqResult {
	var results []PrereqResult

	// Check AWS CLI
	if _, err := exec.LookPath("aws"); err != nil {
		results = append(results, PrereqResult{
			Name:    "AWS CLI",
			OK:      false,
			Message: "aws CLI not found. Install: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
		})
	} else {
		results = append(results, PrereqResult{Name: "AWS CLI", OK: true, Message: "OK"})
	}

	// Check Session Manager Plugin
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		results = append(results, PrereqResult{
			Name:    "Session Manager Plugin",
			OK:      false,
			Message: "session-manager-plugin not found. Install: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
		})
	} else {
		results = append(results, PrereqResult{Name: "Session Manager Plugin", OK: true, Message: "OK"})
	}

	return results
}

func BuildSSMSessionArgs(instanceID, profile, region string) []string {
	args := []string{"ssm", "start-session", "--target", instanceID}
	if profile != "" && profile != "default" {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--region", region)
	return args
}

func BuildPortForwardArgs(instanceID, profile, region, localPort, remotePort string) []string {
	args := []string{
		"ssm", "start-session",
		"--target", instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", fmt.Sprintf("portNumber=%s,localPortNumber=%s", remotePort, localPort),
	}
	if profile != "" && profile != "default" {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--region", region)
	return args
}

func FetchSSMStatus(ctx context.Context, client *ssm.Client) (map[string]bool, error) {
	status := make(map[string]bool)
	paginator := ssm.NewDescribeInstanceInformationPaginator(client, &ssm.DescribeInstanceInformationInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, info := range page.InstanceInformationList {
			id := awssdk.ToString(info.InstanceId)
			status[id] = string(info.PingStatus) == "Online"
		}
	}
	return status, nil
}
```

- [ ] **Step 4: Run SSM tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/aws/ -run "TestBuild|TestCheck" -v
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/aws/ssm.go internal/aws/ssm_test.go
git commit -m "feat: SSM command building and prerequisite checks"
```

---

### Task 6: UI Styles

**Files:**
- Create: `internal/ui/styles.go`

- [ ] **Step 1: Create styles.go**

```go
// internal/ui/styles.go
package ui

import "charm.land/lipgloss/v2"

var (
	// Status bar (top)
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3c3836")).
			Foreground(lipgloss.Color("#ebdbb2")).
			Padding(0, 1)

	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fabd2f")).
			Bold(true)

	// Help bar (bottom)
	HelpBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3c3836")).
			Foreground(lipgloss.Color("#a89984")).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#83a598")).
			Bold(true)

	// Table
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#83a598")).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#504945"))

	TableSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#504945")).
				Foreground(lipgloss.Color("#ebdbb2"))

	// State colors
	StateRunning    = lipgloss.NewStyle().Foreground(lipgloss.Color("#b8bb26"))
	StateStopped    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fb4934"))
	StatePending    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f"))
	StateStopping   = lipgloss.NewStyle().Foreground(lipgloss.Color("#fe8019"))
	StateTerminated = lipgloss.NewStyle().Foreground(lipgloss.Color("#928374"))

	// Favorites & history markers
	FavoriteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f"))
	RecentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#83a598"))

	// Overlay
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#83a598")).
			Padding(1, 2)

	// Error
	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fb4934")).
			Bold(true)

	// Search
	SearchPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#fabd2f")).
				Bold(true)
)

func StateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return StateRunning
	case "stopped":
		return StateStopped
	case "pending":
		return StatePending
	case "stopping":
		return StateStopping
	case "terminated":
		return StateTerminated
	default:
		return lipgloss.NewStyle()
	}
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/styles.go
git commit -m "feat: TUI styles with Gruvbox color scheme"
```

---

### Task 7: UI Status Bar & Help Bar

**Files:**
- Create: `internal/ui/statusbar.go`
- Create: `internal/ui/helpbar.go`

- [ ] **Step 1: Create statusbar.go**

```go
// internal/ui/statusbar.go
package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func RenderStatusBar(profile, region, filter string, count int, width int) string {
	profilePart := StatusKeyStyle.Render("Profile: ") + profile
	regionPart := StatusKeyStyle.Render("Region: ") + region
	filterPart := StatusKeyStyle.Render("Filter: ") + filter
	countPart := fmt.Sprintf("[%d instances]", count)

	content := fmt.Sprintf(" %s  ┊  %s  ┊  %s  ┊  %s", profilePart, regionPart, filterPart, countPart)
	return StatusBarStyle.Width(width).Render(content)
}
```

- [ ] **Step 2: Create helpbar.go**

```go
// internal/ui/helpbar.go
package ui

import "fmt"

type ViewState int

const (
	ViewTable ViewState = iota
	ViewSearch
	ViewFilter
	ViewProfileSelect
	ViewRegionSelect
	ViewPortForward
)

func RenderHelpBar(state ViewState, width int) string {
	var keys string
	switch state {
	case ViewSearch:
		keys = helpLine(
			"Enter", "Connect",
			"Esc", "Cancel",
		)
	case ViewFilter, ViewProfileSelect, ViewRegionSelect:
		keys = helpLine(
			"↑↓", "Navigate",
			"Enter", "Select",
			"Esc", "Cancel",
		)
	case ViewPortForward:
		keys = helpLine(
			"Enter", "Start",
			"Esc", "Cancel",
		)
	default:
		keys = helpLine(
			"↑↓", "Navigate",
			"Enter", "Connect",
			"/", "Search",
			"f", "Filter",
			"p", "Profile",
			"r", "Region",
			"s", "Sort",
			"F", "Fav",
			"P", "Port Fwd",
			"R", "Refresh",
			"q", "Quit",
		)
	}
	return HelpBarStyle.Width(width).Render(keys)
}

func helpLine(keyvals ...string) string {
	var s string
	for i := 0; i < len(keyvals)-1; i += 2 {
		if s != "" {
			s += "  "
		}
		s += fmt.Sprintf("%s: %s", HelpKeyStyle.Render(keyvals[i]), keyvals[i+1])
	}
	return " " + s
}
```

- [ ] **Step 3: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/statusbar.go internal/ui/helpbar.go
git commit -m "feat: status bar and help bar components"
```

---

### Task 8: UI Table Rendering & Sorting

**Files:**
- Create: `internal/ui/table.go`
- Create: `internal/ui/table_test.go`

- [ ] **Step 1: Write table sorting test**

```go
// internal/ui/table_test.go
package ui

import (
	"testing"

	"tui-ssm/internal/aws"
	"tui-ssm/internal/store"
)

func TestSortInstances(t *testing.T) {
	instances := []aws.Instance{
		{InstanceID: "i-3", Name: "charlie", State: "running"},
		{InstanceID: "i-1", Name: "alpha", State: "stopped"},
		{InstanceID: "i-2", Name: "bravo", State: "running"},
	}

	favs := &store.Favorites{}
	favs.Add(store.Favorite{InstanceID: "i-2", Profile: "default", Region: "us-east-1", Alias: "bravo"})

	hist := &store.History{MaxEntries: 100}
	hist.Add(store.HistoryEntry{InstanceID: "i-3", Profile: "default", Region: "us-east-1", Alias: "charlie", Type: "session"})

	sorted := SortInstances(instances, favs, hist, "default", "us-east-1", "name", "asc")

	// Favorite (i-2 bravo) should be first
	if sorted[0].InstanceID != "i-2" {
		t.Errorf("expected favorite i-2 first, got %s", sorted[0].InstanceID)
	}
	// Recent (i-3 charlie) should be second
	if sorted[1].InstanceID != "i-3" {
		t.Errorf("expected recent i-3 second, got %s", sorted[1].InstanceID)
	}
	// Remaining (i-1 alpha) should be last
	if sorted[2].InstanceID != "i-1" {
		t.Errorf("expected i-1 last, got %s", sorted[2].InstanceID)
	}
}

func TestFilterInstancesBySearch(t *testing.T) {
	instances := []aws.Instance{
		{InstanceID: "i-abc123", Name: "web-server", PrivateIP: "10.0.1.1"},
		{InstanceID: "i-def456", Name: "db-primary", PrivateIP: "10.0.2.1"},
		{InstanceID: "i-ghi789", Name: "web-worker", PrivateIP: "10.0.1.2"},
	}

	result := FilterBySearch(instances, "web")
	if len(result) != 2 {
		t.Errorf("expected 2 matches for 'web', got %d", len(result))
	}

	result = FilterBySearch(instances, "10.0.2")
	if len(result) != 1 {
		t.Errorf("expected 1 match for '10.0.2', got %d", len(result))
	}

	result = FilterBySearch(instances, "i-def")
	if len(result) != 1 {
		t.Errorf("expected 1 match for 'i-def', got %d", len(result))
	}
}

func TestFilterInstancesByState(t *testing.T) {
	instances := []aws.Instance{
		{InstanceID: "i-1", State: "running"},
		{InstanceID: "i-2", State: "stopped"},
		{InstanceID: "i-3", State: "running"},
		{InstanceID: "i-4", State: "terminated"},
	}

	result := FilterByState(instances, map[string]bool{"running": true})
	if len(result) != 2 {
		t.Errorf("expected 2 running instances, got %d", len(result))
	}

	// Empty filter = show all
	result = FilterByState(instances, map[string]bool{})
	if len(result) != 4 {
		t.Errorf("expected 4 instances with no filter, got %d", len(result))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/ui/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement table.go**

```go
// internal/ui/table.go
package ui

import (
	"fmt"
	"sort"
	"strings"

	"tui-ssm/internal/aws"
	"tui-ssm/internal/store"
)

type Column struct {
	Key   string
	Title string
	Width int
}

func DefaultColumns() []Column {
	return []Column{
		{Key: "fav", Title: " ", Width: 2},
		{Key: "state_icon", Title: " ", Width: 2},
		{Key: "name", Title: "Name", Width: 20},
		{Key: "id", Title: "Instance ID", Width: 21},
		{Key: "state", Title: "State", Width: 10},
		{Key: "private_ip", Title: "Private IP", Width: 15},
		{Key: "type", Title: "Type", Width: 12},
		{Key: "az", Title: "AZ", Width: 5},
		{Key: "platform", Title: "Platform", Width: 10},
		{Key: "public_ip", Title: "Public IP", Width: 15},
		{Key: "launch_time", Title: "Launch Time", Width: 18},
		{Key: "sg", Title: "Security Groups", Width: 20},
		{Key: "key_pair", Title: "Key Pair", Width: 15},
		{Key: "iam_role", Title: "IAM Role", Width: 20},
	}
}

func CompactColumns() []Column {
	return []Column{
		{Key: "fav", Title: " ", Width: 2},
		{Key: "state_icon", Title: " ", Width: 2},
		{Key: "name", Title: "Name", Width: 20},
		{Key: "state", Title: "State", Width: 10},
		{Key: "private_ip", Title: "Private IP", Width: 15},
	}
}

func ColumnsForWidth(width int) []Column {
	if width < 80 {
		return CompactColumns()
	}
	return DefaultColumns()
}

func RenderTable(instances []aws.Instance, columns []Column, cursor int, favs *store.Favorites, hist *store.History, profile, region string, width, height int) string {
	var b strings.Builder

	// Header
	header := renderRow(columns, func(col Column) string {
		return col.Title
	})
	b.WriteString(TableHeaderStyle.Width(width).Render(header))
	b.WriteString("\n")

	// Rows (fit available height: total - statusbar(1) - helpbar(1) - header(1) - search(1 if active))
	maxRows := height - 4
	if maxRows < 1 {
		maxRows = 1
	}

	// Calculate scroll offset
	offset := 0
	if cursor >= maxRows {
		offset = cursor - maxRows + 1
	}

	for i := offset; i < len(instances) && i < offset+maxRows; i++ {
		inst := instances[i]
		row := renderRow(columns, func(col Column) string {
			return cellValue(col.Key, inst, favs, hist, profile, region)
		})

		if i == cursor {
			row = TableSelectedStyle.Width(width).Render(row)
		}
		b.WriteString(row)
		if i < offset+maxRows-1 && i < len(instances)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderRow(columns []Column, getValue func(Column) string) string {
	var parts []string
	for _, col := range columns {
		val := getValue(col)
		if len(val) > col.Width {
			val = val[:col.Width-1] + "…"
		}
		parts = append(parts, fmt.Sprintf("%-*s", col.Width, val))
	}
	return strings.Join(parts, " ")
}

func cellValue(key string, inst aws.Instance, favs *store.Favorites, hist *store.History, profile, region string) string {
	switch key {
	case "fav":
		if favs != nil && favs.IsFavorite(inst.InstanceID, profile, region) {
			return FavoriteStyle.Render("★")
		}
		if hist != nil && hist.IsRecent(inst.InstanceID, profile, region) {
			return RecentStyle.Render("⏱")
		}
		return " "
	case "state_icon":
		return StateStyle(inst.State).Render(inst.StateIcon())
	case "name":
		return inst.DisplayName()
	case "id":
		return inst.InstanceID
	case "state":
		return StateStyle(inst.State).Render(inst.State)
	case "private_ip":
		return inst.PrivateIP
	case "public_ip":
		return inst.PublicIP
	case "type":
		return inst.InstanceType
	case "az":
		return inst.ShortAZ()
	case "platform":
		return inst.Platform
	case "launch_time":
		return inst.LaunchTimeFormatted()
	case "sg":
		return strings.Join(inst.SecurityGroups, ",")
	case "key_pair":
		return inst.KeyPair
	case "iam_role":
		return inst.IAMRole
	default:
		return ""
	}
}

func SortInstances(instances []aws.Instance, favs *store.Favorites, hist *store.History, profile, region, sortBy, sortOrder string) []aws.Instance {
	sorted := make([]aws.Instance, len(instances))
	copy(sorted, instances)

	sort.SliceStable(sorted, func(i, j int) bool {
		// Priority 1: Favorites first
		iFav := favs != nil && favs.IsFavorite(sorted[i].InstanceID, profile, region)
		jFav := favs != nil && favs.IsFavorite(sorted[j].InstanceID, profile, region)
		if iFav != jFav {
			return iFav
		}

		// Priority 2: Recent history
		iRecent := hist != nil && hist.IsRecent(sorted[i].InstanceID, profile, region)
		jRecent := hist != nil && hist.IsRecent(sorted[j].InstanceID, profile, region)
		if iRecent != jRecent {
			return iRecent
		}

		// Priority 3: User-selected sort
		var less bool
		switch sortBy {
		case "id":
			less = sorted[i].InstanceID < sorted[j].InstanceID
		case "state":
			less = sorted[i].State < sorted[j].State
		case "type":
			less = sorted[i].InstanceType < sorted[j].InstanceType
		case "az":
			less = sorted[i].AvailabilityZone < sorted[j].AvailabilityZone
		default: // "name"
			less = sorted[i].DisplayName() < sorted[j].DisplayName()
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	return sorted
}

func FilterBySearch(instances []aws.Instance, query string) []aws.Instance {
	if query == "" {
		return instances
	}
	q := strings.ToLower(query)
	var result []aws.Instance
	for _, inst := range instances {
		if strings.Contains(strings.ToLower(inst.Name), q) ||
			strings.Contains(strings.ToLower(inst.InstanceID), q) ||
			strings.Contains(inst.PrivateIP, q) {
			result = append(result, inst)
		}
	}
	return result
}

func FilterByState(instances []aws.Instance, states map[string]bool) []aws.Instance {
	if len(states) == 0 {
		return instances
	}
	var result []aws.Instance
	for _, inst := range instances {
		if states[inst.State] {
			result = append(result, inst)
		}
	}
	return result
}
```

- [ ] **Step 4: Run table tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./internal/ui/ -v
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/table.go internal/ui/table_test.go
git commit -m "feat: table rendering, sorting, and filtering"
```

---

### Task 9: UI Search & Filter Components

**Files:**
- Create: `internal/ui/search.go`
- Create: `internal/ui/filter.go`

- [ ] **Step 1: Create search.go**

```go
// internal/ui/search.go
package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type SearchModel struct {
	Query  string
	Active bool
}

func (s *SearchModel) Insert(char rune) {
	s.Query += string(char)
}

func (s *SearchModel) Backspace() {
	if len(s.Query) > 0 {
		s.Query = s.Query[:len(s.Query)-1]
	}
}

func (s *SearchModel) Clear() {
	s.Query = ""
	s.Active = false
}

func (s *SearchModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	prompt := SearchPromptStyle.Render(" /")
	return lipgloss.NewStyle().Width(width).Render(
		fmt.Sprintf("%s %s█", prompt, s.Query),
	)
}
```

- [ ] **Step 2: Create filter.go**

```go
// internal/ui/filter.go
package ui

import (
	"fmt"
	"strings"
)

type FilterModel struct {
	States       []string
	ActiveStates map[string]bool
	Cursor       int
	Active       bool
}

func NewFilterModel() FilterModel {
	return FilterModel{
		States:       []string{"running", "stopped", "pending", "stopping", "terminated"},
		ActiveStates: map[string]bool{},
		Cursor:       0,
	}
}

func (f *FilterModel) Toggle() {
	state := f.States[f.Cursor]
	if f.ActiveStates[state] {
		delete(f.ActiveStates, state)
	} else {
		f.ActiveStates[state] = true
	}
}

func (f *FilterModel) ClearAll() {
	f.ActiveStates = map[string]bool{}
}

func (f *FilterModel) MoveUp() {
	if f.Cursor > 0 {
		f.Cursor--
	}
}

func (f *FilterModel) MoveDown() {
	if f.Cursor < len(f.States)-1 {
		f.Cursor++
	}
}

func (f *FilterModel) Label() string {
	if len(f.ActiveStates) == 0 {
		return "all"
	}
	var active []string
	for _, s := range f.States {
		if f.ActiveStates[s] {
			active = append(active, s)
		}
	}
	return strings.Join(active, ",")
}

func (f *FilterModel) Render(width int) string {
	if !f.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString("  Filter by State\n")
	b.WriteString("  ─────────────────\n")

	for i, state := range f.States {
		cursor := "  "
		if i == f.Cursor {
			cursor = "▸ "
		}
		check := "[ ]"
		if f.ActiveStates[state] {
			check = "[✓]"
		}
		icon := StateStyle(state).Render(fmt.Sprintf("%-12s", state))
		b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, check, icon))
	}
	b.WriteString("\n  Space: toggle  c: clear all  Esc: close")

	return OverlayStyle.Render(b.String())
}
```

- [ ] **Step 3: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/search.go internal/ui/filter.go
git commit -m "feat: search and filter components"
```

---

### Task 10: UI Selector (Profile/Region)

**Files:**
- Create: `internal/ui/selector.go`

- [ ] **Step 1: Create selector.go**

```go
// internal/ui/selector.go
package ui

import (
	"fmt"
	"strings"
)

type SelectorModel struct {
	Title   string
	Items   []string
	Cursor  int
	Active  bool
}

func NewSelector(title string, items []string, current string) SelectorModel {
	cursor := 0
	for i, item := range items {
		if item == current {
			cursor = i
			break
		}
	}
	return SelectorModel{
		Title:  title,
		Items:  items,
		Cursor: cursor,
	}
}

func (s *SelectorModel) MoveUp() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

func (s *SelectorModel) MoveDown() {
	if s.Cursor < len(s.Items)-1 {
		s.Cursor++
	}
}

func (s *SelectorModel) Selected() string {
	if s.Cursor < len(s.Items) {
		return s.Items[s.Cursor]
	}
	return ""
}

func (s *SelectorModel) Render(width int) string {
	if !s.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s\n", s.Title))
	b.WriteString("  ─────────────────────────\n")

	// Show a window of items around cursor
	maxVisible := 15
	start := 0
	if s.Cursor >= maxVisible {
		start = s.Cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(s.Items) {
		end = len(s.Items)
	}

	if start > 0 {
		b.WriteString("    ↑ more\n")
	}
	for i := start; i < end; i++ {
		cursor := "  "
		if i == s.Cursor {
			cursor = "▸ "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, s.Items[i]))
	}
	if end < len(s.Items) {
		b.WriteString("    ↓ more\n")
	}
	b.WriteString("\n  Enter: select  Esc: cancel")

	return OverlayStyle.Render(b.String())
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/selector.go
git commit -m "feat: profile/region selector component"
```

---

### Task 11: Root Model & State Machine

**Files:**
- Create: `internal/ui/model.go`

This is the central piece that wires together all UI components. It implements the Bubble Tea `Model` interface and dispatches messages to sub-components based on the current view state.

- [ ] **Step 1: Create model.go**

```go
// internal/ui/model.go
package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	internalaws "tui-ssm/internal/aws"
	"tui-ssm/internal/config"
	"tui-ssm/internal/store"
)

// Messages
type instancesLoadedMsg struct {
	instances []internalaws.Instance
	ssmStatus map[string]bool
	err       error
}

type ssmSessionDoneMsg struct{ err error }

// Sort columns cycle
var sortColumns = []string{"name", "id", "state", "type", "az"}

type Model struct {
	// State
	viewState   ViewState
	loading     bool
	err         error

	// Data
	instances   []internalaws.Instance
	filtered    []internalaws.Instance
	cursor      int

	// AWS
	profile     string
	region      string
	profiles    []string
	clients     *internalaws.Clients

	// Config & Store
	cfg         config.Config
	favorites   *store.Favorites
	history     *store.History

	// UI Components
	search      SearchModel
	filter      FilterModel
	profSelect  SelectorModel
	regionSelect SelectorModel
	portForward PortForwardModel

	// Layout
	width       int
	height      int

	// Sort
	sortBy      string
	sortOrder   string
	sortIdx     int
}

type PortForwardModel struct {
	Active     bool
	LocalPort  string
	RemotePort string
	Field      int // 0 = local, 1 = remote
}

func NewModel(cfg config.Config, profiles []string, favs *store.Favorites, hist *store.History) Model {
	return Model{
		viewState: ViewTable,
		loading:   true,
		profile:   cfg.DefaultProfile,
		region:    cfg.DefaultRegion,
		profiles:  profiles,
		cfg:       cfg,
		favorites: favs,
		history:   hist,
		filter:    NewFilterModel(),
		sortBy:    cfg.Table.SortBy,
		sortOrder: cfg.Table.SortOrder,
		sortIdx:   0,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadInstances()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case instancesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.instances = msg.instances
		// Apply SSM status
		for i := range m.instances {
			if msg.ssmStatus != nil {
				m.instances[i].SSMConnected = msg.ssmStatus[m.instances[i].InstanceID]
			}
		}
		m.applyFilters()
		return m, nil

	case ssmSessionDoneMsg:
		// Returned from SSM session, refresh
		m.viewState = ViewTable
		m.loading = true
		m.history.Save(store.HistoryPath())
		return m, m.loadInstances()
	}

	// Dispatch based on view state
	switch m.viewState {
	case ViewSearch:
		return m.updateSearch(msg)
	case ViewFilter:
		return m.updateFilter(msg)
	case ViewProfileSelect:
		return m.updateProfileSelect(msg)
	case ViewRegionSelect:
		return m.updateRegionSelect(msg)
	case ViewPortForward:
		return m.updatePortForward(msg)
	default:
		return m.updateTable(msg)
	}
}

func (m Model) updateTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "enter":
		if m.cursor < len(m.filtered) {
			return m, m.startSSMSession(m.filtered[m.cursor])
		}

	case "/":
		m.viewState = ViewSearch
		m.search.Active = true
		m.search.Query = ""

	case "f":
		m.viewState = ViewFilter
		m.filter.Active = true

	case "p":
		m.profSelect = NewSelector("Select Profile", m.profiles, m.profile)
		m.profSelect.Active = true
		m.viewState = ViewProfileSelect

	case "r":
		m.regionSelect = NewSelector("Select Region", internalaws.KnownRegions(), m.region)
		m.regionSelect.Active = true
		m.viewState = ViewRegionSelect

	case "s":
		m.sortIdx = (m.sortIdx + 1) % len(sortColumns)
		m.sortBy = sortColumns[m.sortIdx]
		m.applyFilters()

	case "S":
		if m.sortOrder == "asc" {
			m.sortOrder = "desc"
		} else {
			m.sortOrder = "asc"
		}
		m.applyFilters()

	case "F":
		if m.cursor < len(m.filtered) {
			inst := m.filtered[m.cursor]
			if m.favorites.IsFavorite(inst.InstanceID, m.profile, m.region) {
				m.favorites.Remove(inst.InstanceID, m.profile, m.region)
			} else {
				m.favorites.Add(store.Favorite{
					InstanceID: inst.InstanceID,
					Profile:    m.profile,
					Region:     m.region,
					Alias:      inst.DisplayName(),
				})
			}
			m.favorites.Save(store.FavoritesPath())
			m.applyFilters()
		}

	case "P":
		if m.cursor < len(m.filtered) {
			m.viewState = ViewPortForward
			m.portForward = PortForwardModel{Active: true, LocalPort: "8080", RemotePort: "80", Field: 0}
		}

	case "R":
		m.loading = true
		m.err = nil
		return m, m.loadInstances()
	}

	return m, nil
}

func (m Model) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "escape":
		m.search.Clear()
		m.viewState = ViewTable
		m.applyFilters()
	case "enter":
		m.viewState = ViewTable
		m.search.Active = false
		if m.cursor < len(m.filtered) {
			return m, m.startSSMSession(m.filtered[m.cursor])
		}
	case "backspace":
		m.search.Backspace()
		m.applyFilters()
	default:
		r := keyMsg.String()
		if len(r) == 1 {
			m.search.Insert(rune(r[0]))
			m.applyFilters()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "escape", "f":
		m.filter.Active = false
		m.viewState = ViewTable
		m.applyFilters()
	case "up", "k":
		m.filter.MoveUp()
	case "down", "j":
		m.filter.MoveDown()
	case " ", "enter":
		m.filter.Toggle()
		m.applyFilters()
	case "c":
		m.filter.ClearAll()
		m.applyFilters()
	}
	return m, nil
}

func (m Model) updateProfileSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "escape":
		m.profSelect.Active = false
		m.viewState = ViewTable
	case "up", "k":
		m.profSelect.MoveUp()
	case "down", "j":
		m.profSelect.MoveDown()
	case "enter":
		m.profile = m.profSelect.Selected()
		m.profSelect.Active = false
		m.viewState = ViewTable
		m.loading = true
		return m, m.loadInstances()
	}
	return m, nil
}

func (m Model) updateRegionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "escape":
		m.regionSelect.Active = false
		m.viewState = ViewTable
	case "up", "k":
		m.regionSelect.MoveUp()
	case "down", "j":
		m.regionSelect.MoveDown()
	case "enter":
		m.region = m.regionSelect.Selected()
		m.regionSelect.Active = false
		m.viewState = ViewTable
		m.loading = true
		return m, m.loadInstances()
	}
	return m, nil
}

func (m Model) updatePortForward(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "escape":
		m.portForward.Active = false
		m.viewState = ViewTable
	case "tab":
		m.portForward.Field = (m.portForward.Field + 1) % 2
	case "enter":
		if m.cursor < len(m.filtered) {
			m.portForward.Active = false
			return m, m.startPortForward(m.filtered[m.cursor])
		}
	case "backspace":
		if m.portForward.Field == 0 && len(m.portForward.LocalPort) > 0 {
			m.portForward.LocalPort = m.portForward.LocalPort[:len(m.portForward.LocalPort)-1]
		} else if m.portForward.Field == 1 && len(m.portForward.RemotePort) > 0 {
			m.portForward.RemotePort = m.portForward.RemotePort[:len(m.portForward.RemotePort)-1]
		}
	default:
		r := keyMsg.String()
		if len(r) == 1 && r[0] >= '0' && r[0] <= '9' {
			if m.portForward.Field == 0 {
				m.portForward.LocalPort += r
			} else {
				m.portForward.RemotePort += r
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	var sections []string

	// Status bar
	sections = append(sections, RenderStatusBar(m.profile, m.region, m.filter.Label(), len(m.filtered), m.width))

	// Search bar (if active)
	if m.search.Active {
		sections = append(sections, m.search.Render(m.width))
	}

	// Main content
	if m.loading {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(2, 2).Render("Loading instances..."))
	} else if m.err != nil {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(
			ErrorStyle.Render(fmt.Sprintf("Error: %v\n\nPress R to retry, p to change profile, r to change region", m.err)),
		))
	} else if len(m.filtered) == 0 {
		sections = append(sections, lipgloss.NewStyle().Width(m.width).Padding(2, 2).Render("No instances found in this region."))
	} else {
		columns := ColumnsForWidth(m.width)
		tableHeight := m.height
		if m.search.Active {
			tableHeight-- // account for search bar
		}
		sections = append(sections, RenderTable(m.filtered, columns, m.cursor, m.favorites, m.history, m.profile, m.region, m.width, tableHeight))
	}

	// Overlay (filter / profile / region / port forward)
	overlay := ""
	switch {
	case m.filter.Active:
		overlay = m.filter.Render(m.width)
	case m.profSelect.Active:
		overlay = m.profSelect.Render(m.width)
	case m.regionSelect.Active:
		overlay = m.regionSelect.Render(m.width)
	case m.portForward.Active:
		overlay = m.renderPortForward()
	}

	// Help bar
	sections = append(sections, RenderHelpBar(m.viewState, m.width))

	view := strings.Join(sections, "\n")
	if overlay != "" {
		// Place overlay centered on screen
		view += "\n" + lipgloss.Place(m.width, 0, lipgloss.Center, lipgloss.Center, overlay)
	}

	return tea.NewView(view)
}

func (m Model) renderPortForward() string {
	var b strings.Builder
	b.WriteString("  Port Forwarding\n")
	b.WriteString("  ─────────────────\n")
	if m.cursor < len(m.filtered) {
		b.WriteString(fmt.Sprintf("  Target: %s\n\n", m.filtered[m.cursor].DisplayName()))
	}

	localLabel := "  Local Port:  "
	remoteLabel := "  Remote Port: "
	if m.portForward.Field == 0 {
		localLabel = "▸ Local Port:  "
	} else {
		remoteLabel = "▸ Remote Port: "
	}
	b.WriteString(fmt.Sprintf("%s%s\n", localLabel, m.portForward.LocalPort))
	b.WriteString(fmt.Sprintf("%s%s\n", remoteLabel, m.portForward.RemotePort))
	b.WriteString("\n  Tab: switch field  Enter: start  Esc: cancel")

	return OverlayStyle.Render(b.String())
}

func (m *Model) applyFilters() {
	result := m.instances

	// State filter
	result = FilterByState(result, m.filter.ActiveStates)

	// Search filter
	result = FilterBySearch(result, m.search.Query)

	// Sort
	result = SortInstances(result, m.favorites, m.history, m.profile, m.region, m.sortBy, m.sortOrder)

	m.filtered = result

	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) loadInstances() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		clients, err := internalaws.NewClients(ctx, m.profile, m.region)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		instances, err := internalaws.FetchInstances(ctx, clients.EC2)
		if err != nil {
			return instancesLoadedMsg{err: err}
		}

		ssmStatus, _ := internalaws.FetchSSMStatus(ctx, clients.SSM)

		return instancesLoadedMsg{
			instances: instances,
			ssmStatus: ssmStatus,
		}
	}
}

func (m Model) startSSMSession(inst internalaws.Instance) tea.Cmd {
	m.history.Add(store.HistoryEntry{
		InstanceID: inst.InstanceID,
		Profile:    m.profile,
		Region:     m.region,
		Alias:      inst.DisplayName(),
		Type:       "session",
	})

	args := internalaws.BuildSSMSessionArgs(inst.InstanceID, m.profile, m.region)
	c := exec.Command("aws", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ssmSessionDoneMsg{err: err}
	})
}

func (m Model) startPortForward(inst internalaws.Instance) tea.Cmd {
	m.history.Add(store.HistoryEntry{
		InstanceID: inst.InstanceID,
		Profile:    m.profile,
		Region:     m.region,
		Alias:      inst.DisplayName(),
		Type:       "port_forward",
	})

	args := internalaws.BuildPortForwardArgs(inst.InstanceID, m.profile, m.region, m.portForward.LocalPort, m.portForward.RemotePort)
	c := exec.Command("aws", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ssmSessionDoneMsg{err: err}
	})
}
```

- [ ] **Step 2: Verify build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build ./...
```

Expected: Success.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/model.go
git commit -m "feat: root model with state machine and all UI interactions"
```

---

### Task 12: Store Path Helpers & App Wiring

**Files:**
- Modify: `internal/store/favorites.go` — add `FavoritesPath()` helper
- Modify: `internal/store/history.go` — add `HistoryPath()` helper
- Modify: `main.go` — full wiring with prereq checks and Bubble Tea program

- [ ] **Step 1: Add path helpers to favorites.go**

Add at end of `internal/store/favorites.go`:

```go
func FavoritesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tui-ssm", "favorites.json")
}
```

- [ ] **Step 2: Add path helpers to history.go**

Add at end of `internal/store/history.go`:

```go
func HistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tui-ssm", "history.json")
}
```

- [ ] **Step 3: Implement full main.go**

```go
// main.go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	internalaws "tui-ssm/internal/aws"
	"tui-ssm/internal/config"
	"tui-ssm/internal/store"
	"tui-ssm/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("tui-ssm %s\n", version)
		os.Exit(0)
	}

	// Prerequisite checks
	results := internalaws.CheckPrerequisites()
	for _, r := range results {
		if !r.OK {
			fmt.Fprintf(os.Stderr, "ERROR: %s — %s\n", r.Name, r.Message)
			os.Exit(1)
		}
	}

	// Load config
	cfg, err := config.Load(config.Path())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Ensure data directory exists
	os.MkdirAll(config.Dir(), 0o755)

	// Load stores
	favs, err := store.LoadFavorites(store.FavoritesPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load favorites: %v\n", err)
		os.Exit(1)
	}

	hist, err := store.LoadHistory(store.HistoryPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history: %v\n", err)
		os.Exit(1)
	}

	// Parse AWS profiles
	profiles := internalaws.ParseProfiles(
		internalaws.DefaultCredentialsPath(),
		internalaws.DefaultConfigPath(),
	)
	if len(profiles) == 0 {
		profiles = []string{"default"}
	}

	// Create and run TUI
	model := ui.NewModel(cfg, profiles, favs, hist)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run full build**

```bash
cd /home/ec2-user/my-project/tui-ssm && go build -o tui-ssm ./main.go
```

Expected: Success — produces `tui-ssm` binary.

- [ ] **Step 5: Run all tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./... -v
```

Expected: All tests PASS.

- [ ] **Step 6: Commit**

```bash
git add main.go internal/store/favorites.go internal/store/history.go
git commit -m "feat: complete app wiring with prereq checks and TUI launch"
```

---

### Task 13: Final Integration Testing & Polish

**Files:**
- Modify: various — fix any issues found during integration testing

- [ ] **Step 1: Verify binary runs with --version**

```bash
cd /home/ec2-user/my-project/tui-ssm && ./tui-ssm --version
```

Expected: `tui-ssm dev`

- [ ] **Step 2: Run go vet and check for issues**

```bash
cd /home/ec2-user/my-project/tui-ssm && go vet ./...
```

Expected: No issues.

- [ ] **Step 3: Run all tests**

```bash
cd /home/ec2-user/my-project/tui-ssm && go test ./... -v -count=1
```

Expected: All PASS.

- [ ] **Step 4: Test binary launches (requires AWS credentials)**

```bash
cd /home/ec2-user/my-project/tui-ssm && ./tui-ssm
```

Expected: TUI launches with loading indicator, then shows EC2 list (or error if no credentials).

- [ ] **Step 5: Build cross-platform binaries**

```bash
cd /home/ec2-user/my-project/tui-ssm && make build-all
ls -la dist/
```

Expected: 4 binaries in `dist/`.

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "chore: final integration testing and polish"
```

---

## Summary

| Task | Description | Estimated Steps |
|------|-------------|-----------------|
| 1 | Project scaffolding + config | 9 |
| 2 | Favorites & history store | 9 |
| 3 | AWS profile & session | 7 |
| 4 | EC2 data model & fetching | 5 |
| 5 | SSM integration & prereqs | 5 |
| 6 | UI styles | 3 |
| 7 | Status bar & help bar | 4 |
| 8 | Table rendering & sorting | 5 |
| 9 | Search & filter components | 4 |
| 10 | Profile/region selector | 3 |
| 11 | Root model & state machine | 3 |
| 12 | Store helpers & app wiring | 6 |
| 13 | Integration testing & polish | 6 |
| **Total** | | **69 steps** |
