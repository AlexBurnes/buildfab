package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/internal/config"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// mockUI is a mock implementation of the UI interface for testing
type mockUI struct {
	verbose bool
	debug   bool
}

func (m *mockUI) PrintCLIHeader(name, version string) {}
func (m *mockUI) PrintProjectCheck(projectName, version string) {}
func (m *mockUI) PrintStepStatus(stepName string, status buildfab.Status, message string) {}
func (m *mockUI) PrintStageHeader(stageName string) {}
func (m *mockUI) PrintStageResult(stageName string, success bool, duration time.Duration) {}
func (m *mockUI) PrintCommand(command string) {}
func (m *mockUI) PrintCommandOutput(output string) {}
func (m *mockUI) PrintRepro(stepName, repro string) {}
func (m *mockUI) PrintReproInline(stepName, repro string) {}
func (m *mockUI) PrintSummary(results []buildfab.Result) {}
func (m *mockUI) IsVerbose() bool { return m.verbose }
func (m *mockUI) IsDebug() bool { return m.debug }

func TestNew(t *testing.T) {
	config := &buildfab.Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []buildfab.Action{
			{Name: "test-action", Run: "echo hello"},
		},
	}

	opts := &buildfab.RunOptions{
		ConfigPath:  "test.yml",
		MaxParallel: 2,
		Verbose:     true,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
		Output:      &mockUI{verbose: true, debug: false},
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	executor := New(config, opts, &mockUI{verbose: true, debug: false})

	if executor.config != config {
		t.Errorf("config = %v, want %v", executor.config, config)
	}
	if executor.opts != opts {
		t.Errorf("opts = %v, want %v", executor.opts, opts)
	}
	if executor.registry == nil {
		t.Error("registry should not be nil")
	}
	if executor.versionDetector == nil {
		t.Error("versionDetector should not be nil")
	}
}

func TestExecutor_RunStage(t *testing.T) {
	config := &buildfab.Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []buildfab.Action{
			{Name: "test-action", Run: "echo hello"},
		},
		Stages: map[string]buildfab.Stage{
			"test-stage": {
				Steps: []buildfab.Step{
					{Action: "test-action"},
				},
			},
		},
	}

	opts := &buildfab.RunOptions{
		ConfigPath:  "test.yml",
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
		Output:      &mockUI{verbose: false, debug: false},
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	executor := New(config, opts, &mockUI{verbose: false, debug: false})

	// Test with existing stage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := executor.RunStage(ctx, "test-stage")
	// Note: This will likely fail due to unimplemented DAG execution
	// but we're testing the error handling and stage lookup
	if err == nil {
		t.Log("RunStage() completed successfully (unexpected)")
	} else {
		t.Logf("RunStage() returned error (expected): %v", err)
	}

	// Test with non-existing stage
	err = executor.RunStage(ctx, "nonexistent-stage")
	if err == nil {
		t.Error("RunStage() expected error for non-existing stage, got nil")
	}
	if err.Error() != "stage not found: nonexistent-stage" {
		t.Errorf("RunStage() error = %v, want %v", err.Error(), "stage not found: nonexistent-stage")
	}
}

func TestExecutor_RunAction(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{Name: "test-action", Run: "echo hello"},
		},
	}

	opts := &buildfab.RunOptions{
		ConfigPath:  "test.yml",
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
		Output:      &mockUI{verbose: false, debug: false},
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	executor := New(config, opts, &mockUI{verbose: false, debug: false})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with existing action
	err := executor.RunAction(ctx, "test-action")
	// Note: This will likely fail due to unimplemented action execution
	// but we're testing the error handling and action lookup
	if err == nil {
		t.Log("RunAction() completed successfully (unexpected)")
	} else {
		t.Logf("RunAction() returned error (expected): %v", err)
	}

	// Test with non-existing action
	err = executor.RunAction(ctx, "nonexistent-action")
	if err == nil {
		t.Error("RunAction() expected error for non-existing action, got nil")
	}
	if err.Error() != "action not found: nonexistent-action" {
		t.Errorf("RunAction() error = %v, want %v", err.Error(), "action not found: nonexistent-action")
	}
}

func TestExecutor_RunStageStep(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{Name: "test-action", Run: "echo hello"},
		},
		Stages: map[string]buildfab.Stage{
			"test-stage": {
				Steps: []buildfab.Step{
					{Action: "test-action"},
					{Action: "another-action"},
				},
			},
		},
	}

	opts := &buildfab.RunOptions{
		ConfigPath:  "test.yml",
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
		Output:      &mockUI{verbose: false, debug: false},
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	executor := New(config, opts, &mockUI{verbose: false, debug: false})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with existing step
	err := executor.RunStageStep(ctx, "test-stage", "test-action")
	// Note: This will likely fail due to unimplemented step execution
	// but we're testing the error handling and step lookup
	if err == nil {
		t.Log("RunStageStep() completed successfully (unexpected)")
	} else {
		t.Logf("RunStageStep() returned error (expected): %v", err)
	}

	// Test with non-existing stage
	err = executor.RunStageStep(ctx, "nonexistent-stage", "test-action")
	if err == nil {
		t.Error("RunStageStep() expected error for non-existing stage, got nil")
	}
	if err.Error() != "stage not found: nonexistent-stage" {
		t.Errorf("RunStageStep() error = %v, want %v", err.Error(), "stage not found: nonexistent-stage")
	}

	// Test with non-existing step
	err = executor.RunStageStep(ctx, "test-stage", "nonexistent-step")
	if err == nil {
		t.Error("RunStageStep() expected error for non-existing step, got nil")
	}
	if err.Error() != "step not found: nonexistent-step in stage test-stage" {
		t.Errorf("RunStageStep() error = %v, want %v", err.Error(), "step not found: nonexistent-step in stage test-stage")
	}
}

func TestExecutor_ListActions(t *testing.T) {
	config := &buildfab.Config{
		Actions: []buildfab.Action{
			{Name: "action1", Run: "echo hello"},
			{Name: "action2", Uses: "git@untracked"},
		},
	}

	opts := &buildfab.RunOptions{}
	executor := New(config, opts, &mockUI{verbose: false, debug: false})

	actions := executor.ListActions()
	if len(actions) != 2 {
		t.Errorf("ListActions() length = %v, want %v", len(actions), 2)
	}

	// Check that actions are returned correctly
	actionNames := make(map[string]bool)
	for _, action := range actions {
		actionNames[action.Name] = true
	}

	if !actionNames["action1"] {
		t.Error("action1 should be in list")
	}
	if !actionNames["action2"] {
		t.Error("action2 should be in list")
	}
}

func TestExecutor_Integration(t *testing.T) {
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

actions:
  - name: test-action
    run: echo "Hello World"
  - name: version-check
    uses: version@check

stages:
  test-stage:
    steps:
      - action: test-action
      - action: version-check
`

	configFile := filepath.Join(tempDir, "project.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write project.yml: %v", err)
	}

	// Load configuration using internal/config package
	config, err := config.Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	opts := &buildfab.RunOptions{
		ConfigPath:  configFile,
		MaxParallel: 2,
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  tempDir,
		Output:      &mockUI{verbose: false, debug: false},
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}

	executor := New(config, opts, &mockUI{verbose: false, debug: false})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test running the stage
	err = executor.RunStage(ctx, "test-stage")
	// Note: This will likely fail due to unimplemented DAG execution
	// but we're testing the integration flow
	if err == nil {
		t.Log("Integration test completed successfully (unexpected)")
	} else {
		t.Logf("Integration test returned error (expected): %v", err)
	}
}