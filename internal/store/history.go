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
	Type        string    `json:"type"`
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

func HistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tui-aws", "history.json")
}
