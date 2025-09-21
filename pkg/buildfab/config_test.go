package buildfab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test with non-existent file
	_, err := LoadConfig("nonexistent.yml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !contains(err.Error(), "configuration file not found") {
		t.Errorf("Expected error about file not found, got %v", err.Error())
	}
	
	// Test with empty path (should try default locations)
	_, err = LoadConfig("")
	if err == nil {
		t.Error("Expected error for empty path with no default files")
	}
	if !contains(err.Error(), "no configuration file found") {
		t.Errorf("Expected error about no configuration file found, got %v", err.Error())
	}
}

func TestLoadConfigFromBytes(t *testing.T) {
	// Test valid YAML
	validYAML := `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test:
    steps:
      - action: test-action
`
	
	config, err := LoadConfigFromBytes([]byte(validYAML))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config.Project.Name != "test-project" {
		t.Errorf("Project name = %v, want %v", config.Project.Name, "test-project")
	}
	if len(config.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(config.Actions))
	}
	if config.Actions[0].Name != "test-action" {
		t.Errorf("Action name = %v, want %v", config.Actions[0].Name, "test-action")
	}
	
	// Test invalid YAML
	invalidYAML := `invalid: yaml: content: [`
	_, err = LoadConfigFromBytes([]byte(invalidYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
	if !contains(err.Error(), "failed to parse configuration") {
		t.Errorf("Expected error about parsing configuration, got %v", err.Error())
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test.yml")
	
	// Create a test configuration file
	configContent := `
project:
  name: test-project
  bin: /custom/bin
actions:
  - name: test-action
    run: echo hello
stages:
  test:
    steps:
      - action: test-action
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Test loading the file
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config.Project.Name != "test-project" {
		t.Errorf("Project name = %v, want %v", config.Project.Name, "test-project")
	}
	if config.Project.BinDir != "/custom/bin" {
		t.Errorf("BinDir = %v, want %v", config.Project.BinDir, "/custom/bin")
	}
}

func TestLoadConfig_DefaultLocations(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	
	// Create a test configuration file with default name
	configContent := `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test:
    steps:
      - action: test-action
`
	
	err = os.WriteFile(".project.yml", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// Test loading with empty path (should find .project.yml)
	config, err := LoadConfig("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if config.Project.Name != "test-project" {
		t.Errorf("Project name = %v, want %v", config.Project.Name, "test-project")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:], substr))))
}