package buildfab

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestSuccessAction always succeeds
type TestSuccessAction struct{}

func (a *TestSuccessAction) Run(ctx context.Context) (Result, error) {
	return Result{
		Status:  StatusOK,
		Message: "Test action succeeded",
	}, nil
}

func (a *TestSuccessAction) Description() string {
	return "Test action that always succeeds"
}

// TestFailureAction always fails
type TestFailureAction struct{}

func (a *TestFailureAction) Run(ctx context.Context) (Result, error) {
	return Result{
		Status:  StatusError,
		Message: "Test action failed",
	}, fmt.Errorf("test action failed")
}

func (a *TestFailureAction) Description() string {
	return "Test action that always fails"
}

// TestWarnAction always returns a warning
type TestWarnAction struct{}

func (a *TestWarnAction) Run(ctx context.Context) (Result, error) {
	return Result{
		Status:  StatusWarn,
		Message: "Test action warning",
	}, nil
}

func (a *TestWarnAction) Description() string {
	return "Test action that always warns"
}

func TestOnErrorPolicyWithTestActions(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		stageName      string
		wantStatus     StepStatus
		wantMessage    string
		wantStageError bool
		wantWarnings   int
		wantErrors     int
	}{
		{
			name: "test@success with onerror warn - should succeed",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-success",
						Uses: "test@success",
					},
				},
				Stages: map[string]Stage{
					"test-stage": {
						Steps: []Step{
							{
								Action:  "test-success",
								OnError: "warn",
							},
						},
					},
				},
			},
			stageName:      "test-stage",
			wantStatus:     StepStatusOK,
			wantMessage:    "executed successfully",
			wantStageError: false,
			wantWarnings:   0,
			wantErrors:     0,
		},
		{
			name: "test@failure with onerror warn - should convert to warning",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-failure",
						Uses: "test@failure",
					},
				},
				Stages: map[string]Stage{
					"test-stage": {
						Steps: []Step{
							{
								Action:  "test-failure",
								OnError: "warn",
							},
						},
					},
				},
			},
			stageName:      "test-stage",
			wantStatus:     StepStatusWarn,
			wantMessage:    "Test action failed",
			wantStageError: false,
			wantWarnings:   1,
			wantErrors:     0,
		},
		{
			name: "test@failure with onerror stop - should fail",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-failure",
						Uses: "test@failure",
					},
				},
				Stages: map[string]Stage{
					"test-stage": {
						Steps: []Step{
							{
								Action:  "test-failure",
								OnError: "stop",
							},
						},
					},
				},
			},
			stageName:      "test-stage",
			wantStatus:     StepStatusError,
			wantMessage:    "Test action failed",
			wantStageError: true,
			wantWarnings:   0,
			wantErrors:     1,
		},
		{
			name: "test@warn with onerror warn - should remain warning",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-warn",
						Uses: "test@warn",
					},
				},
				Stages: map[string]Stage{
					"test-stage": {
						Steps: []Step{
							{
								Action:  "test-warn",
								OnError: "warn",
							},
						},
					},
				},
			},
			stageName:      "test-stage",
			wantStatus:     StepStatusWarn,
			wantMessage:    "Test action warning",
			wantStageError: false,
			wantWarnings:   1,
			wantErrors:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test callback to capture step results
			var stepResults []OnErrorTestResult
			stepCallback := &TestStepCallback{
				results: &stepResults,
			}

			// Create a custom action registry with test actions
			registry := NewDefaultActionRegistry()
			registry.Register("test@success", &TestSuccessAction{})
			registry.Register("test@failure", &TestFailureAction{})
			registry.Register("test@warn", &TestWarnAction{})

			opts := &RunOptions{
				StepCallback: stepCallback,
				Verbose:      true,
			}

			runner := NewRunner(tt.config, opts)
			runner.registry = registry // Set the custom registry
			ctx := context.Background()

			// Run the stage
			err := runner.RunStage(ctx, tt.stageName)

			// Check stage error
			if tt.wantStageError && err == nil {
				t.Errorf("expected stage to fail, but it succeeded")
			}
			if !tt.wantStageError && err != nil {
				t.Errorf("expected stage to succeed, but it failed: %v", err)
			}

			// Check step results
			if len(stepResults) != 1 {
				t.Errorf("expected 1 step result, got %d", len(stepResults))
				return
			}

			result := stepResults[0]
			if result.Status != tt.wantStatus {
				t.Errorf("expected status %v, got %v", tt.wantStatus, result.Status)
			}

			// Check Output field for the expected text
			if !strings.Contains(result.Output, tt.wantMessage) {
				t.Errorf("expected output to contain %q, got %q", tt.wantMessage, result.Output)
			}

			// Count warnings and errors
			warnings := 0
			errors := 0
			for _, r := range stepResults {
				if r.Status == StepStatusWarn {
					warnings++
				} else if r.Status == StepStatusError {
					errors++
				}
			}

			if warnings != tt.wantWarnings {
				t.Errorf("expected %d warnings, got %d", tt.wantWarnings, warnings)
			}
			if errors != tt.wantErrors {
				t.Errorf("expected %d errors, got %d", tt.wantErrors, errors)
			}
		})
	}
}

// TestStepCallback is a test implementation of StepCallback
type TestStepCallback struct {
	results *[]OnErrorTestResult
}

func (c *TestStepCallback) OnStepStart(ctx context.Context, stepName string) {
	// No-op for testing
}

func (c *TestStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	*c.results = append(*c.results, OnErrorTestResult{
		StepName:   stepName,
		ActionName: stepName,
		Status:     status,
		Output:     message,
		Duration:   duration,
	})
}

func (c *TestStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	// No-op for testing
}

func (c *TestStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	// No-op for testing
}

// OnErrorTestResult represents a step execution result for testing
type OnErrorTestResult struct {
	StepName   string
	ActionName string
	Status     StepStatus
	Output     string
	Duration   time.Duration
}