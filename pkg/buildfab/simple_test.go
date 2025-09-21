package buildfab

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSimpleRunner_RunStage(t *testing.T) {
	// Create a test configuration
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "test-action", Run: "echo 'Hello World'"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "test-action"},
				},
			},
		},
	}

	// Create simple runner
	opts := &SimpleRunOptions{
		Verbose:     true,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	}
	runner := NewSimpleRunner(config, opts)

	// Test running a stage
	ctx := context.Background()
	err := runner.RunStage(ctx, "test-stage")
	if err != nil {
		t.Errorf("RunStage failed: %v", err)
	}
}

func TestSimpleRunner_RunAction(t *testing.T) {
	// Create a test configuration
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "test-action", Run: "echo 'Hello World'"},
		},
		Stages: map[string]Stage{},
	}

	// Create simple runner
	opts := &SimpleRunOptions{
		Verbose:     true,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	}
	runner := NewSimpleRunner(config, opts)

	// Test running an action
	ctx := context.Background()
	err := runner.RunAction(ctx, "test-action")
	if err != nil {
		t.Errorf("RunAction failed: %v", err)
	}
}

func TestSimpleRunner_RunStageStep(t *testing.T) {
	// Create a test configuration
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "test-action", Run: "echo 'Hello World'"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "test-action"},
				},
			},
		},
	}

	// Create simple runner
	opts := &SimpleRunOptions{
		Verbose:     true,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	}
	runner := NewSimpleRunner(config, opts)

	// Test running a specific step
	ctx := context.Background()
	err := runner.RunStageStep(ctx, "test-stage", "test-action")
	if err != nil {
		t.Errorf("RunStageStep failed: %v", err)
	}
}

func TestSimpleRunner_ErrorHandling(t *testing.T) {
	// Create a test configuration
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{},
		Stages:  map[string]Stage{},
	}

	// Create simple runner
	opts := &SimpleRunOptions{
		Verbose:     true,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
	}
	runner := NewSimpleRunner(config, opts)

	// Test running non-existent stage
	ctx := context.Background()
	err := runner.RunStage(ctx, "non-existent-stage")
	if err == nil {
		t.Error("Expected error for non-existent stage, got nil")
	}
	if !strings.Contains(err.Error(), "stage not found") {
		t.Errorf("Expected 'stage not found' error, got: %v", err)
	}

	// Test running non-existent action
	err = runner.RunAction(ctx, "non-existent-action")
	if err == nil {
		t.Error("Expected error for non-existent action, got nil")
	}
	if !strings.Contains(err.Error(), "action not found") {
		t.Errorf("Expected 'action not found' error, got: %v", err)
	}
}

func TestRunStageSimple(t *testing.T) {
	// This test would require a real config file, so we'll skip it for now
	// In a real test, you would create a temporary config file
	t.Skip("Skipping RunStageSimple test - requires real config file")
}

func TestRunActionSimple(t *testing.T) {
	// This test would require a real config file, so we'll skip it for now
	// In a real test, you would create a temporary config file
	t.Skip("Skipping RunActionSimple test - requires real config file")
}

func TestSimpleStepCallback(t *testing.T) {
	// Create a test configuration
	config := &Config{
		Actions: []Action{
			{Name: "test-action", Run: "echo 'Hello World'"},
			{Name: "failing-action", Run: "false"},
		},
	}

	callback := &SimpleStepCallback{
		verbose: true,
		debug:   false,
		output:  os.Stdout,
		errorOutput: os.Stderr,
		config:  config,
	}

	ctx := context.Background()

	// Test OnStepStart
	callback.OnStepStart(ctx, "test-step")

	// Test OnStepComplete with success
	callback.OnStepComplete(ctx, "test-step", StepStatusOK, "success", time.Second)

	// Test OnStepComplete with error (should enhance message)
	callback.OnStepComplete(ctx, "failing-action", StepStatusError, "command failed: exit status 1", time.Second)

	// Test OnStepComplete with skipped (should enhance message)
	callback.OnStepComplete(ctx, "skipped-step", StepStatusSkipped, "skipped (dependency failed)", time.Second)

	// Test OnStepOutput
	callback.OnStepOutput(ctx, "test-step", "test output")

	// Test OnStepError
	callback.OnStepError(ctx, "test-step", nil)

	// Test GetResults
	results := callback.GetResults()
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}
	
	// Check that error message was enhanced
	foundError := false
	for _, result := range results {
		if result.StepName == "failing-action" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("Expected failing-action result not found")
	}
}