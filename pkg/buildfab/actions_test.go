package buildfab

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultActionRegistry(t *testing.T) {
	registry := NewDefaultActionRegistry()
	
	// Test that all expected actions are registered
	expectedActions := []string{
		"git@untracked",
		"git@uncommitted", 
		"git@modified",
		"version@check",
		"version@check-greatest",
	}
	
	for _, action := range expectedActions {
		runner, exists := registry.GetRunner(action)
		if !exists {
			t.Errorf("Expected action %s to be registered", action)
		}
		if runner == nil {
			t.Errorf("Expected runner for %s to not be nil", action)
		}
	}
	
	// Test non-existent action
	_, exists := registry.GetRunner("nonexistent")
	if exists {
		t.Error("Expected nonexistent action to not exist")
	}
}

func TestDefaultActionRegistry_ListActions(t *testing.T) {
	registry := NewDefaultActionRegistry()
	actions := registry.ListActions()
	
	expectedCount := 5
	if len(actions) != expectedCount {
		t.Errorf("Expected %d actions, got %d", expectedCount, len(actions))
	}
	
	// Check that all actions have descriptions
	for name, description := range actions {
		if description == "" {
			t.Errorf("Action %s should have a description", name)
		}
	}
}

func TestGitUntrackedAction(t *testing.T) {
	action := &GitUntrackedAction{}
	
	// Test description
	desc := action.Description()
	if desc != "Check for untracked files" {
		t.Errorf("Description = %v, want %v", desc, "Check for untracked files")
	}
	
	// Note: We can't easily test the actual git functionality in unit tests
	// without setting up a git repository, so we'll just test the interface
}

func TestGitUncommittedAction(t *testing.T) {
	action := &GitUncommittedAction{}
	
	// Test description
	desc := action.Description()
	if desc != "Check for uncommitted changes" {
		t.Errorf("Description = %v, want %v", desc, "Check for uncommitted changes")
	}
}

func TestGitModifiedAction(t *testing.T) {
	action := &GitModifiedAction{}
	
	// Test description
	desc := action.Description()
	if desc != "Check for modified files" {
		t.Errorf("Description = %v, want %v", desc, "Check for modified files")
	}
}

func TestVersionCheckAction(t *testing.T) {
	action := &VersionCheckAction{}
	
	// Test description
	desc := action.Description()
	if desc != "Validate version format" {
		t.Errorf("Description = %v, want %v", desc, "Validate version format")
	}
	
	// Test with missing VERSION file
	result, err := action.Run(context.Background())
	if err == nil {
		t.Error("Expected error for missing VERSION file")
	}
	if result.Status != StatusError {
		t.Errorf("Expected StatusError, got %v", result.Status)
	}
	if !strings.Contains(result.Message, "VERSION file not found") {
		t.Errorf("Expected error message about VERSION file, got %v", result.Message)
	}
}

func TestVersionCheckGreatestAction(t *testing.T) {
	action := &VersionCheckGreatestAction{}
	
	// Test description
	desc := action.Description()
	if desc != "Check if current version is the greatest" {
		t.Errorf("Description = %v, want %v", desc, "Check if current version is the greatest")
	}
	
	// Test with missing VERSION file
	result, err := action.Run(context.Background())
	if err == nil {
		t.Error("Expected error for missing VERSION file")
	}
	if result.Status != StatusError {
		t.Errorf("Expected StatusError, got %v", result.Status)
	}
	if !strings.Contains(result.Message, "VERSION file not found") {
		t.Errorf("Expected error message about VERSION file, got %v", result.Message)
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"v1.0.0", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0-beta", true},
		{"v1.0.0-rc1", true},
		{"v1.0.0-dev", true},
		{"1.0.0", false}, // missing v prefix
		{"v1.0", false},  // not enough parts
		{"v1", false},    // not enough parts
		{"v", false},     // empty after v
		{"v1.0.0.", false}, // empty part
		{"v1..0", false}, // empty part
		{"", false},      // empty
	}
	
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if got := isValidVersion(tt.version); got != tt.want {
				t.Errorf("isValidVersion(%v) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestIsValidVersionPart(t *testing.T) {
	tests := []struct {
		part string
		want bool
	}{
		{"0", true},
		{"1", true},
		{"123", true},
		{"alpha", true},
		{"beta", true},
		{"rc1", true},
		{"dev", true},
		{"ALPHA", true}, // case insensitive
		{"BETA", true},  // case insensitive
		{"", false},     // empty
		{"1a", false},   // mixed
		{"a1", false},   // mixed
	}
	
	for _, tt := range tests {
		t.Run(tt.part, func(t *testing.T) {
			if got := isValidVersionPart(tt.part); got != tt.want {
				t.Errorf("isValidVersionPart(%v) = %v, want %v", tt.part, got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"0", true},
		{"1", true},
		{"123", true},
		{"", false},
		{"a", false},
		{"1a", false},
		{"a1", false},
		{"1.0", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isNumeric(tt.s); got != tt.want {
				t.Errorf("isNumeric(%v) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestRunner_ListBuiltInActions(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
	}
	
	// Test with default registry
	runner := NewRunner(config, nil)
	actions := runner.ListBuiltInActions()
	
	expectedCount := 5
	if len(actions) != expectedCount {
		t.Errorf("Expected %d built-in actions, got %d", expectedCount, len(actions))
	}
	
	// Test with nil registry
	runner2 := NewRunnerWithRegistry(config, nil, nil)
	actions2 := runner2.ListBuiltInActions()
	
	if len(actions2) != 0 {
		t.Errorf("Expected 0 built-in actions with nil registry, got %d", len(actions2))
	}
}

func TestRunner_RunAction_WithBuiltInAction(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "test-version-check", Uses: "version@check"},
		},
	}
	
	runner := NewRunner(config, nil)
	
	// Test built-in action (should fail because no VERSION file)
	err := runner.RunAction(context.Background(), "test-version-check")
	if err == nil {
		t.Error("Expected error for missing VERSION file")
	}
	if !strings.Contains(err.Error(), "VERSION file not found") {
		t.Errorf("Expected error about VERSION file, got %v", err.Error())
	}
}

func TestRunner_RunAction_WithUnknownBuiltInAction(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "test-unknown", Uses: "unknown@action"},
		},
	}
	
	runner := NewRunner(config, nil)
	
	// Test unknown built-in action
	err := runner.RunAction(context.Background(), "test-unknown")
	if err == nil {
		t.Error("Expected error for unknown built-in action")
	}
	if !strings.Contains(err.Error(), "unknown built-in action") {
		t.Errorf("Expected error about unknown built-in action, got %v", err.Error())
	}
}

func TestRunner_RunAction_WithCustomAction(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "test-custom", Run: "echo hello"},
		},
	}
	
	runner := NewRunner(config, nil)
	
	// Test custom action (should work)
	err := runner.RunAction(context.Background(), "test-custom")
	if err != nil {
		t.Errorf("Unexpected error for custom action: %v", err)
	}
}