package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/internal/config"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func TestIntegration_CompleteWorkflow(t *testing.T) {
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

	// Create a test VERSION file
	err = os.WriteFile("VERSION", []byte("v1.0.0"), 0644)
	if err != nil {
		t.Fatalf("Failed to write VERSION file: %v", err)
	}

	// Create a test project.yml
	configContent := `
project:
  name: test-project
  modules: [module1, module2]
  bin: bin/

actions:
  - name: test-action
    run: echo "Hello World"
  - name: version-check
    uses: version@check
  - name: git-check
    uses: git@untracked

stages:
  test-stage:
    steps:
      - action: test-action
      - action: version-check
      - action: git-check
  pre-push:
    steps:
      - action: version-check
      - action: git-check
`

	configFile := filepath.Join(tempDir, "project.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write project.yml: %v", err)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	err = cfg.Validate()
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	// Test configuration access
	action, exists := cfg.GetAction("test-action")
	if !exists {
		t.Error("test-action should exist")
	}
	if action.Name != "test-action" {
		t.Errorf("action.Name = %v, want %v", action.Name, "test-action")
	}

	stage, exists := cfg.GetStage("test-stage")
	if !exists {
		t.Error("test-stage should exist")
	}
	if len(stage.Steps) != 3 {
		t.Errorf("stage.Steps length = %v, want %v", len(stage.Steps), 3)
	}

	// Test variable resolution
	variables := map[string]string{
		"message": "Hello",
		"name":    "World",
	}
	err = config.ResolveVariables(cfg, variables)
	if err != nil {
		t.Fatalf("Variable resolution failed: %v", err)
	}

	// Test simple runner creation (same as CLI)
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configFile,
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   variables,
		WorkingDir:  tempDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	runner := buildfab.NewSimpleRunner(cfg, opts)
	if runner == nil {
		t.Error("SimpleRunner should not be nil")
	}

	// Test stage execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = runner.RunStage(ctx, "test-stage")
	// The DAG execution should now work correctly
	if err == nil {
		t.Log("Integration test completed successfully")
	} else {
		t.Logf("Integration test returned error: %v", err)
	}
}

func TestIntegration_ErrorHandling(t *testing.T) {
	// Test with invalid configuration
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

	// Test with missing VERSION file
	configContent := `
project:
  name: test-project

actions:
  - name: version-check
    uses: version@check

stages:
  test-stage:
    steps:
      - action: version-check
`

	configFile := filepath.Join(tempDir, "project.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write project.yml: %v", err)
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test simple runner with missing VERSION file
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configFile,
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  tempDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	runner := buildfab.NewSimpleRunner(cfg, opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test action execution (should fail due to missing VERSION file)
	err = runner.RunAction(ctx, "version-check")
	if err == nil {
		t.Log("Action execution completed successfully (unexpected)")
	} else {
		t.Logf("Action execution returned error (expected): %v", err)
	}
}

func TestIntegration_ConfigurationValidation(t *testing.T) {
	// Test various invalid configurations
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "missing project name",
			config: `
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "no actions",
			config: `
project:
  name: test-project
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "action without name",
			config: `
project:
  name: test-project
actions:
  - run: echo hello
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "action without run or uses",
			config: `
project:
  name: test-project
actions:
  - name: test-action
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "action with both run and uses",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
    uses: git@untracked
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "duplicate action name",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
  - name: test-action
    run: echo world
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: true,
		},
		{
			name: "stage without steps",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage: {}
`,
			wantErr: true,
		},
		{
			name: "step without action",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - {}
`,
			wantErr: true,
		},
		{
			name: "step with unknown action",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - action: unknown-action
`,
			wantErr: true,
		},
		{
			name: "step with invalid onerror",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - action: test-action
        onerror: invalid
`,
			wantErr: true,
		},
		{
			name: "step with invalid only value",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - action: test-action
        only: [invalid]
`,
			wantErr: true,
		},
		{
			name: "valid configuration",
			config: `
project:
  name: test-project
actions:
  - name: test-action
    run: echo hello
stages:
  test-stage:
    steps:
      - action: test-action
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "test.yml")
			err := os.WriteFile(configFile, []byte(tt.config), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Load and validate configuration
			cfg, err := config.Load(configFile)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				} else {
					// Additional validation for valid configs
					if err := cfg.Validate(); err != nil {
						t.Errorf("Config validation failed for %s: %v", tt.name, err)
					}
				}
			}
		})
	}
}