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
