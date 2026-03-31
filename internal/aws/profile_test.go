package aws

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProfiles(t *testing.T) {
	dir := t.TempDir()

	credPath := filepath.Join(dir, "credentials")
	os.WriteFile(credPath, []byte("[default]\naws_access_key_id = AKIA_DEFAULT\n\n[production]\naws_access_key_id = AKIA_PROD\n"), 0o644)

	configPath := filepath.Join(dir, "config")
	os.WriteFile(configPath, []byte("[default]\nregion = us-east-1\n\n[profile staging]\nregion = eu-west-1\n"), 0o644)

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
	// Should contain only the instance role sentinel
	if len(profiles) != 1 || profiles[0] != InstanceRoleProfile {
		t.Errorf("expected only [%q] for missing files, got %v", InstanceRoleProfile, profiles)
	}
}

func TestParseProfilesIncludesInstanceRole(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials")
	os.WriteFile(credPath, []byte("[myprofile]\naws_access_key_id = AKIA\n"), 0o644)

	profiles := ParseProfiles(credPath, "/nonexistent/config")
	if profiles[0] != InstanceRoleProfile {
		t.Errorf("expected first profile to be %q, got %q", InstanceRoleProfile, profiles[0])
	}
}
