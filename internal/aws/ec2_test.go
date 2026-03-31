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
