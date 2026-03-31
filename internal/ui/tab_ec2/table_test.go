package tab_ec2

import (
	"testing"

	"tui-aws/internal/aws"
	"tui-aws/internal/store"
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

	if sorted[0].InstanceID != "i-2" {
		t.Errorf("expected favorite i-2 first, got %s", sorted[0].InstanceID)
	}
	if sorted[1].InstanceID != "i-3" {
		t.Errorf("expected recent i-3 second, got %s", sorted[1].InstanceID)
	}
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

	result = FilterByState(instances, map[string]bool{})
	if len(result) != 4 {
		t.Errorf("expected 4 instances with no filter, got %d", len(result))
	}
}
