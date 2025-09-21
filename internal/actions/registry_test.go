package actions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func TestNew(t *testing.T) {
	registry := New()
	if registry.actions == nil {
		t.Error("actions map should not be nil")
	}
	if len(registry.actions) == 0 {
		t.Error("actions map should not be empty")
	}

	// Check that built-in actions are registered
	expectedActions := []string{
		"git@untracked",
		"git@uncommitted", 
		"git@modified",
		"version@check",
		"version@check-greatest",
	}

	for _, actionName := range expectedActions {
		if _, exists := registry.actions[actionName]; !exists {
			t.Errorf("Built-in action %s should be registered", actionName)
		}
	}
}

func TestRegister(t *testing.T) {
	registry := New()
	
	// Create a test action
	testAction := &testActionRunner{}
	
	// Register the test action
	registry.Register("test-action", testAction)
	
	// Verify it was registered
	runner, exists := registry.GetRunner("test-action")
	if !exists {
		t.Error("test-action should be registered")
	}
	if runner != testAction {
		t.Error("registered runner should match the test action")
	}
}

func TestGetRunner(t *testing.T) {
	registry := New()
	
	// Test existing action
	runner, exists := registry.GetRunner("git@untracked")
	if !exists {
		t.Error("git@untracked should exist")
	}
	if runner == nil {
		t.Error("git@untracked runner should not be nil")
	}
	
	// Test non-existing action
	_, exists = registry.GetRunner("nonexistent")
	if exists {
		t.Error("nonexistent action should not exist")
	}
}

func TestListActions(t *testing.T) {
	registry := New()
	actions := registry.ListActions()
	
	if len(actions) == 0 {
		t.Error("ListActions() should return non-empty map")
	}
	
	// Check that all registered actions are listed
	for name, runner := range registry.actions {
		if desc, exists := actions[name]; !exists {
			t.Errorf("Action %s should be in list", name)
		} else if desc != runner.Description() {
			t.Errorf("Description for %s should match runner description", name)
		}
	}
}

func TestGitUntrackedAction_Run(t *testing.T) {
	action := &GitUntrackedAction{}
	
	// This test requires a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping git action test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := action.Run(ctx)
	if err != nil {
		t.Logf("GitUntrackedAction.Run() returned error (expected in some cases): %v", err)
	}
	
	// Result should not be nil
	if result.Status == buildfab.StatusOK || result.Status == buildfab.StatusError {
		// Valid status
	} else {
		t.Errorf("Unexpected status: %v", result.Status)
	}
}

func TestGitUntrackedAction_Description(t *testing.T) {
	action := &GitUntrackedAction{}
	desc := action.Description()
	expected := "Check for untracked files"
	if desc != expected {
		t.Errorf("Description() = %v, want %v", desc, expected)
	}
}

func TestGitUncommittedAction_Run(t *testing.T) {
	action := &GitUncommittedAction{}
	
	// This test requires a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping git action test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := action.Run(ctx)
	if err != nil {
		t.Logf("GitUncommittedAction.Run() returned error (expected in some cases): %v", err)
	}
	
	// Result should not be nil
	if result.Status == buildfab.StatusOK || result.Status == buildfab.StatusError {
		// Valid status
	} else {
		t.Errorf("Unexpected status: %v", result.Status)
	}
}

func TestGitUncommittedAction_Description(t *testing.T) {
	action := &GitUncommittedAction{}
	desc := action.Description()
	expected := "Check for uncommitted changes"
	if desc != expected {
		t.Errorf("Description() = %v, want %v", desc, expected)
	}
}

func TestGitModifiedAction_Run(t *testing.T) {
	action := &GitModifiedAction{}
	
	// This test requires a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping git action test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := action.Run(ctx)
	if err != nil {
		t.Logf("GitModifiedAction.Run() returned error (expected in some cases): %v", err)
	}
	
	// Result should not be nil
	if result.Status == buildfab.StatusOK || result.Status == buildfab.StatusWarn {
		// Valid status
	} else {
		t.Errorf("Unexpected status: %v", result.Status)
	}
}

func TestGitModifiedAction_Description(t *testing.T) {
	action := &GitModifiedAction{}
	desc := action.Description()
	expected := "Check for modified files"
	if desc != expected {
		t.Errorf("Description() = %v, want %v", desc, expected)
	}
}

func TestVersionCheckAction_Run(t *testing.T) {
	action := &VersionCheckAction{}
	
	// Create a temporary directory for test
	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with no VERSION file
	result, err := action.Run(ctx)
	if err == nil {
		t.Error("VersionCheckAction.Run() expected error for missing VERSION file, got nil")
	}
	if result.Status != buildfab.StatusError {
		t.Errorf("Status = %v, want %v", result.Status, buildfab.StatusError)
	}

	// Test with empty VERSION file
	err = os.WriteFile("VERSION", []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to write empty VERSION file: %v", err)
	}

	result, err = action.Run(ctx)
	if err == nil {
		t.Error("VersionCheckAction.Run() expected error for empty VERSION file, got nil")
	}
	if result.Status != buildfab.StatusError {
		t.Errorf("Status = %v, want %v", result.Status, buildfab.StatusError)
	}

	// Test with invalid version format
	err = os.WriteFile("VERSION", []byte("invalid"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid VERSION file: %v", err)
	}

	result, err = action.Run(ctx)
	if err == nil {
		t.Error("VersionCheckAction.Run() expected error for invalid version, got nil")
	}
	if result.Status != buildfab.StatusError {
		t.Errorf("Status = %v, want %v", result.Status, buildfab.StatusError)
	}

	// Test with valid version
	err = os.WriteFile("VERSION", []byte("v1.2.3"), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid VERSION file: %v", err)
	}

	result, err = action.Run(ctx)
	if err != nil {
		t.Errorf("VersionCheckAction.Run() unexpected error for valid version: %v", err)
	}
	if result.Status != buildfab.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, buildfab.StatusOK)
	}
}

func TestVersionCheckAction_Description(t *testing.T) {
	action := &VersionCheckAction{}
	desc := action.Description()
	expected := "Validate version format"
	if desc != expected {
		t.Errorf("Description() = %v, want %v", desc, expected)
	}
}

func TestVersionCheckGreatestAction_Run(t *testing.T) {
	action := &VersionCheckGreatestAction{}
	
	// This test requires a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping version check greatest test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := action.Run(ctx)
	if err != nil {
		t.Logf("VersionCheckGreatestAction.Run() returned error (expected in some cases): %v", err)
	}
	
	// Result should not be nil
	if result.Status == buildfab.StatusOK || result.Status == buildfab.StatusError {
		// Valid status
	} else {
		t.Errorf("Unexpected status: %v", result.Status)
	}
}

func TestVersionCheckGreatestAction_Description(t *testing.T) {
	action := &VersionCheckGreatestAction{}
	desc := action.Description()
	expected := "Check if current version is the greatest"
	if desc != expected {
		t.Errorf("Description() = %v, want %v", desc, expected)
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"v1.2.3", true},
		{"v0.0.1", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0-beta.1", true},
		{"v1.0.0-rc.1", true},
		{"v1.0.0-dev", true},
		{"1.2.3", false}, // Missing v prefix
		{"v1", false},     // Missing dots
		{"v1.2", false},   // Missing patch version
		{"v1.2.", false},  // Empty patch version
		{"v.1.2", false},  // Empty major version
		{"v1..2", false},  // Empty minor version
		{"", false},       // Empty version
		{"v", false},      // Only v prefix
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isValidVersion(tt.version); got != tt.valid {
				t.Errorf("isValidVersion(%q) = %v, want %v", tt.version, got, tt.valid)
			}
		})
	}
}

func TestIsValidVersionPart(t *testing.T) {
	tests := []struct {
		part  string
		valid bool
	}{
		{"1", true},
		{"123", true},
		{"0", true},
		{"alpha", true},
		{"beta", true},
		{"rc", true},
		{"dev", true},
		{"ALPHA", true}, // Case insensitive
		{"BETA.1", true},
		{"rc.1", true},
		{"", false},
		{"1.2", false}, // Contains dot
		{"1-2", false}, // Contains dash
		{"abc", false}, // Not numeric or prerelease
	}

	for _, tt := range tests {
		t.Run(tt.part, func(t *testing.T) {
			if got := isValidVersionPart(tt.part); got != tt.valid {
				t.Errorf("isValidVersionPart(%q) = %v, want %v", tt.part, got, tt.valid)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		s     string
		valid bool
	}{
		{"1", true},
		{"123", true},
		{"0", true},
		{"", false},
		{"1.2", false},
		{"1a", false},
		{"a1", false},
		{"-1", false},
		{"+1", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isNumeric(tt.s); got != tt.valid {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.s, got, tt.valid)
			}
		})
	}
}

// testActionRunner is a test implementation of ActionRunner
type testActionRunner struct{}

func (t *testActionRunner) Run(ctx context.Context) (buildfab.Result, error) {
	return buildfab.Result{
		Status:  buildfab.StatusOK,
		Message: "Test action completed",
	}, nil
}

func (t *testActionRunner) Description() string {
	return "Test action for unit testing"
}