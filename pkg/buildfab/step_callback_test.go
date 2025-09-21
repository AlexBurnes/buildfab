package buildfab

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockStepCallback implements StepCallback for testing
type MockStepCallback struct {
	OnStepStartCalls    []string
	OnStepCompleteCalls []StepCompleteCall
	OnStepOutputCalls   []StepOutputCall
	OnStepErrorCalls    []StepErrorCall
}

type StepCompleteCall struct {
	StepName string
	Status   StepStatus
	Message  string
	Duration time.Duration
}

type StepOutputCall struct {
	StepName string
	Output   string
}

type StepErrorCall struct {
	StepName string
	Error    error
}

func (m *MockStepCallback) OnStepStart(ctx context.Context, stepName string) {
	m.OnStepStartCalls = append(m.OnStepStartCalls, stepName)
}

func (m *MockStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	m.OnStepCompleteCalls = append(m.OnStepCompleteCalls, StepCompleteCall{
		StepName: stepName,
		Status:   status,
		Message:  message,
		Duration: duration,
	})
}

func (m *MockStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	m.OnStepOutputCalls = append(m.OnStepOutputCalls, StepOutputCall{
		StepName: stepName,
		Output:   output,
	})
}

func (m *MockStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	m.OnStepErrorCalls = append(m.OnStepErrorCalls, StepErrorCall{
		StepName: stepName,
		Error:    err,
	})
}

func (m *MockStepCallback) Reset() {
	m.OnStepStartCalls = nil
	m.OnStepCompleteCalls = nil
	m.OnStepOutputCalls = nil
	m.OnStepErrorCalls = nil
}

func TestStepCallbackIntegration(t *testing.T) {
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
			{
				Name: "test-action",
				Run:  "echo 'test output'",
			},
			{
				Name: "failing-action",
				Run:  "exit 1",
			},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{
						Action: "test-action",
					},
					{
						Action:  "failing-action",
						OnError: "warn",
					},
				},
			},
		},
	}

	// Create mock callback
	mockCallback := &MockStepCallback{}

	// Create run options with callback
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = true

	// Create runner
	runner := NewRunner(config, opts)

	// Test RunStage with callbacks
	ctx := context.Background()
	err := runner.RunStage(ctx, "test-stage")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 2 {
		t.Errorf("Expected 2 OnStepStart calls, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 2 {
		t.Errorf("Expected 2 OnStepComplete calls, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	// Verify first step (success)
	if mockCallback.OnStepStartCalls[0] != "test-action" {
		t.Errorf("Expected first step to be 'test-action', got '%s'", mockCallback.OnStepStartCalls[0])
	}

	firstComplete := mockCallback.OnStepCompleteCalls[0]
	if firstComplete.StepName != "test-action" {
		t.Errorf("Expected first complete step to be 'test-action', got '%s'", firstComplete.StepName)
	}
	if firstComplete.Status != StepStatusOK {
		t.Errorf("Expected first step status to be OK, got %v", firstComplete.Status)
	}

	// Verify second step (error with warn policy)
	secondComplete := mockCallback.OnStepCompleteCalls[1]
	if secondComplete.StepName != "failing-action" {
		t.Errorf("Expected second complete step to be 'failing-action', got '%s'", secondComplete.StepName)
	}
	if secondComplete.Status != StepStatusError {
		t.Errorf("Expected second step status to be Error, got %v", secondComplete.Status)
	}

	// Verify error callback was called
	if len(mockCallback.OnStepErrorCalls) != 1 {
		t.Errorf("Expected 1 OnStepError call, got %d", len(mockCallback.OnStepErrorCalls))
	}

	if mockCallback.OnStepErrorCalls[0].StepName != "failing-action" {
		t.Errorf("Expected error step to be 'failing-action', got '%s'", mockCallback.OnStepErrorCalls[0].StepName)
	}

	// Stage should complete successfully due to warn policy
	if err != nil {
		t.Errorf("Expected stage to complete successfully, got error: %v", err)
	}
}

func TestRunActionWithCallbacks(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo 'test output'",
			},
		},
	}

	mockCallback := &MockStepCallback{}
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = true

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunAction(ctx, "test-action")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	if mockCallback.OnStepStartCalls[0] != "test-action" {
		t.Errorf("Expected step to be 'test-action', got '%s'", mockCallback.OnStepStartCalls[0])
	}

	complete := mockCallback.OnStepCompleteCalls[0]
	if complete.StepName != "test-action" {
		t.Errorf("Expected complete step to be 'test-action', got '%s'", complete.StepName)
	}
	if complete.Status != StepStatusOK {
		t.Errorf("Expected step status to be OK, got %v", complete.Status)
	}

	if err != nil {
		t.Errorf("Expected action to complete successfully, got error: %v", err)
	}
}

func TestRunStageStepWithCallbacks(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo 'test output'",
			},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{
						Action: "test-action",
					},
				},
			},
		},
	}

	mockCallback := &MockStepCallback{}
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = true

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunStageStep(ctx, "test-stage", "test-action")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	if mockCallback.OnStepStartCalls[0] != "test-action" {
		t.Errorf("Expected step to be 'test-action', got '%s'", mockCallback.OnStepStartCalls[0])
	}

	complete := mockCallback.OnStepCompleteCalls[0]
	if complete.StepName != "test-action" {
		t.Errorf("Expected complete step to be 'test-action', got '%s'", complete.StepName)
	}
	if complete.Status != StepStatusOK {
		t.Errorf("Expected step status to be OK, got %v", complete.Status)
	}

	if err != nil {
		t.Errorf("Expected step to complete successfully, got error: %v", err)
	}
}

func TestStepCallbackWithoutVerbose(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo 'test output'",
			},
		},
	}

	mockCallback := &MockStepCallback{}
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = false // Not verbose

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunAction(ctx, "test-action")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	// OnStepOutput should not be called when not verbose
	if len(mockCallback.OnStepOutputCalls) != 0 {
		t.Errorf("Expected 0 OnStepOutput calls when not verbose, got %d", len(mockCallback.OnStepOutputCalls))
	}

	if err != nil {
		t.Errorf("Expected action to complete successfully, got error: %v", err)
	}
}

func TestStepCallbackNil(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo 'test output'",
			},
		},
	}

	// No callback provided
	opts := DefaultRunOptions()
	opts.StepCallback = nil

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunAction(ctx, "test-action")

	// Should work without callbacks
	if err != nil {
		t.Errorf("Expected action to complete successfully without callbacks, got error: %v", err)
	}
}

func TestStepStatusString(t *testing.T) {
	testCases := []struct {
		status StepStatus
		expected string
	}{
		{StepStatusPending, "pending"},
		{StepStatusRunning, "running"},
		{StepStatusOK, "ok"},
		{StepStatusWarn, "warn"},
		{StepStatusError, "error"},
		{StepStatusSkipped, "skipped"},
		{StepStatus(999), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.status.String()
		if result != tc.expected {
			t.Errorf("Expected StepStatus(%d).String() to return '%s', got '%s'", tc.status, tc.expected, result)
		}
	}
}

// Test callback with built-in actions
func TestStepCallbackWithBuiltInActions(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "version-check",
				Uses: "version@check",
			},
		},
	}

	mockCallback := &MockStepCallback{}
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = true

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunAction(ctx, "version-check")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	if mockCallback.OnStepStartCalls[0] != "version-check" {
		t.Errorf("Expected step to be 'version-check', got '%s'", mockCallback.OnStepStartCalls[0])
	}

	complete := mockCallback.OnStepCompleteCalls[0]
	if complete.StepName != "version-check" {
		t.Errorf("Expected complete step to be 'version-check', got '%s'", complete.StepName)
	}

	// Built-in actions may succeed or fail depending on environment
	// Just verify the callback was called with appropriate status
	if complete.Status != StepStatusOK && complete.Status != StepStatusError {
		t.Errorf("Expected step status to be OK or Error, got %v", complete.Status)
	}

	// Note: err may be non-nil for built-in actions depending on environment
	// This is expected behavior, so we don't check err here
	_ = err
}

// Test callback output with verbose mode
func TestStepCallbackOutput(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "output-action",
				Run:  "echo 'test output line 1' && echo 'test output line 2'",
			},
		},
	}

	mockCallback := &MockStepCallback{}
	opts := DefaultRunOptions()
	opts.StepCallback = mockCallback
	opts.Verbose = true

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunAction(ctx, "output-action")

	// Verify callbacks were called
	if len(mockCallback.OnStepStartCalls) != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", len(mockCallback.OnStepStartCalls))
	}

	if len(mockCallback.OnStepCompleteCalls) != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", len(mockCallback.OnStepCompleteCalls))
	}

	// Should have output callbacks in verbose mode
	if len(mockCallback.OnStepOutputCalls) == 0 {
		t.Errorf("Expected OnStepOutput calls in verbose mode, got %d", len(mockCallback.OnStepOutputCalls))
	}

	// Verify output contains expected content
	outputFound := false
	for _, outputCall := range mockCallback.OnStepOutputCalls {
		if strings.Contains(outputCall.Output, "test output line 1") {
			outputFound = true
			break
		}
	}

	if !outputFound {
		t.Errorf("Expected output to contain 'test output line 1'")
	}

	if err != nil {
		t.Errorf("Expected action to complete successfully, got error: %v", err)
	}
}