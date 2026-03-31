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
	for _, arg := range args {
		if arg == "--profile" {
			t.Error("default profile should not include --profile flag")
		}
	}
}

func TestBuildSSMCommandInstanceRole(t *testing.T) {
	args := BuildSSMSessionArgs("i-abc123", InstanceRoleProfile, "us-east-1")
	for _, arg := range args {
		if arg == "--profile" {
			t.Error("instance role should not include --profile flag")
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
