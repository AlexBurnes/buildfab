package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func TestNew(t *testing.T) {
	loader := New("test.yml")
	if loader.configPath != "test.yml" {
		t.Errorf("configPath = %v, want %v", loader.configPath, "test.yml")
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	
	// Test with non-existent file
	_, err := Load("nonexistent.yml")
	if err == nil {
		t.Error("Load() expected error for non-existent file, got nil")
	}

	// Create a valid test configuration file
	configContent := `
project:
  name: test-project
  modules: [module1, module2]
  bin: bin/

actions:
  - name: test-action
    run: echo "Hello World"

stages:
  test-stage:
    steps:
      - action: test-action
`

	configFile := filepath.Join(tempDir, "test.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading valid configuration
	config, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if config.Project.Name != "test-project" {
		t.Errorf("Project.Name = %v, want %v", config.Project.Name, "test-project")
	}
	if len(config.Project.Modules) != 2 {
		t.Errorf("Project.Modules length = %v, want %v", len(config.Project.Modules), 2)
	}
	if config.Project.BinDir != "bin/" {
		t.Errorf("Project.BinDir = %v, want %v", config.Project.BinDir, "bin/")
	}
	if len(config.Actions) != 1 {
		t.Errorf("Actions length = %v, want %v", len(config.Actions), 1)
	}
	if config.Actions[0].Name != "test-action" {
		t.Errorf("Actions[0].Name = %v, want %v", config.Actions[0].Name, "test-action")
	}
	if config.Actions[0].Run != `echo "Hello World"` {
		t.Errorf("Actions[0].Run = %v, want %v", config.Actions[0].Run, `echo "Hello World"`)
	}
	if len(config.Stages) != 1 {
		t.Errorf("Stages length = %v, want %v", len(config.Stages), 1)
	}
	if _, exists := config.Stages["test-stage"]; !exists {
		t.Error("test-stage should exist in stages")
	}
}

func TestLoadFromDir(t *testing.T) {
	tempDir := t.TempDir()

	// Test with empty directory
	_, err := LoadFromDir(tempDir)
	if err == nil {
		t.Error("LoadFromDir() expected error for empty directory, got nil")
	}

	// Create test configuration files
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello World"

stages:
  test-stage:
    steps:
      - action: test-action
`

	// Test with .project.yml
	projectFile := filepath.Join(tempDir, ".project.yml")
	err = os.WriteFile(projectFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .project.yml: %v", err)
	}

	config, err := LoadFromDir(tempDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error: %v", err)
	}
	if config.Project.Name != "test-project" {
		t.Errorf("Project.Name = %v, want %v", config.Project.Name, "test-project")
	}

	// Test with project.yml (should still find .project.yml first)
	projectFile2 := filepath.Join(tempDir, "project.yml")
	err = os.WriteFile(projectFile2, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write project.yml: %v", err)
	}

	config2, err := LoadFromDir(tempDir)
	if err != nil {
		t.Fatalf("LoadFromDir() unexpected error: %v", err)
	}
	if config2.Project.Name != "test-project" {
		t.Errorf("Project.Name = %v, want %v", config2.Project.Name, "test-project")
	}
}

func TestResolveVariables(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{
				Name: "test-action",
				Run:  "echo ${{message}} and ${{name}}",
			},
		},
	}

	variables := map[string]string{
		"message": "Hello",
		"name":    "World",
	}

	err := ResolveVariables(config, variables)
	if err != nil {
		t.Fatalf("ResolveVariables() unexpected error: %v", err)
	}

	expected := "echo Hello and World"
	if config.Actions[0].Run != expected {
		t.Errorf("Actions[0].Run = %v, want %v", config.Actions[0].Run, expected)
	}
}

func TestResolveVariables_UndefinedVariable(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{
				Name: "test-action",
				Run:  "echo ${{undefined}}",
			},
		},
	}

	variables := map[string]string{}

	err := ResolveVariables(config, variables)
	if err == nil {
		t.Error("ResolveVariables() expected error for undefined variable, got nil")
	}
	if err.Error() != "failed to resolve variables in action test-action: undefined variable: undefined" {
		t.Errorf("ResolveVariables() error = %v, want %v", err.Error(), "failed to resolve variables in action test-action: undefined variable: undefined")
	}
}

func TestResolveVariables_UnclosedVariable(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{
				Name: "test-action",
				Run:  "echo ${{unclosed",
			},
		},
	}

	variables := map[string]string{}

	err := ResolveVariables(config, variables)
	if err == nil {
		t.Error("ResolveVariables() expected error for unclosed variable, got nil")
	}
	if err.Error() != "failed to resolve variables in action test-action: unclosed variable reference: ${{unclosed" {
		t.Errorf("ResolveVariables() error = %v, want %v", err.Error(), "failed to resolve variables in action test-action: unclosed variable reference: ${{unclosed")
	}
}

func TestResolveString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		variables map[string]string
		expected  string
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "simple variable",
			input: "Hello ${{name}}",
			variables: map[string]string{
				"name": "World",
			},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:  "multiple variables",
			input: "${{greeting}} ${{name}}!",
			variables: map[string]string{
				"greeting": "Hello",
				"name":     "World",
			},
			expected: "Hello World!",
			wantErr:  false,
		},
		{
			name:  "no variables",
			input: "Hello World",
			variables: map[string]string{},
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:  "undefined variable",
			input: "Hello ${{undefined}}",
			variables: map[string]string{},
			expected: "",
			wantErr:  true,
			errMsg:   "undefined variable: undefined",
		},
		{
			name:  "unclosed variable",
			input: "Hello ${{unclosed",
			variables: map[string]string{},
			expected: "",
			wantErr:  true,
			errMsg:   "unclosed variable reference: ${{unclosed",
		},
		{
			name:  "variable with spaces",
			input: "Hello ${{ name }}",
			variables: map[string]string{
				"name": "World",
			},
			expected: "Hello World",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveString(tt.input, tt.variables)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveString() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("resolveString() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveString() unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("resolveString() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestGetDefaultVariables(t *testing.T) {
	variables := GetDefaultVariables()
	
	expectedKeys := []string{"tag", "branch"}
	for _, key := range expectedKeys {
		if _, exists := variables[key]; !exists {
			t.Errorf("Default variables should contain key: %s", key)
		}
	}
	
	// Default values should be empty
	if variables["tag"] != "" {
		t.Errorf("Default tag value = %v, want empty string", variables["tag"])
	}
	if variables["branch"] != "" {
		t.Errorf("Default branch value = %v, want empty string", variables["branch"])
	}
}

func TestDetectGitVariables(t *testing.T) {
	// This test requires a git repository
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping git variable detection test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	variables, err := DetectGitVariables(ctx)
	if err != nil {
		t.Fatalf("DetectGitVariables() unexpected error: %v", err)
	}

	// Should have at least tag and branch keys
	if _, exists := variables["tag"]; !exists {
		t.Error("Detected variables should contain 'tag' key")
	}
	if _, exists := variables["branch"]; !exists {
		t.Error("Detected variables should contain 'branch' key")
	}
}

func TestDetectGitVariables_NoGit(t *testing.T) {
	// Create a temporary directory without git
	tempDir := t.TempDir()
	
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	variables, err := DetectGitVariables(ctx)
	if err != nil {
		t.Fatalf("DetectGitVariables() unexpected error: %v", err)
	}

	// Should still return empty variables
	if len(variables) != 0 {
		t.Errorf("DetectGitVariables() should return empty variables in non-git directory, got %v", variables)
	}
}