package buildfab

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockStepCallback implements StepCallback for testing
type MockStepCallback struct {
	mu                  sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnStepStartCalls = append(m.OnStepStartCalls, stepName)
}

func (m *MockStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnStepCompleteCalls = append(m.OnStepCompleteCalls, StepCompleteCall{
		StepName: stepName,
		Status:   status,
		Message:  message,
		Duration: duration,
	})
}

func (m *MockStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnStepOutputCalls = append(m.OnStepOutputCalls, StepOutputCall{
		StepName: stepName,
		Output:   output,
	})
}

func (m *MockStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnStepErrorCalls = append(m.OnStepErrorCalls, StepErrorCall{
		StepName: stepName,
		Error:    err,
	})
}

func (m *MockStepCallback) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	onStepErrorCalls := len(mockCallback.OnStepErrorCalls)
	
	// Copy the slices to avoid holding the lock while accessing them
	stepStartCalls := make([]string, len(mockCallback.OnStepStartCalls))
	copy(stepStartCalls, mockCallback.OnStepStartCalls)
	stepCompleteCalls := make([]StepCompleteCall, len(mockCallback.OnStepCompleteCalls))
	copy(stepCompleteCalls, mockCallback.OnStepCompleteCalls)
	stepErrorCalls := make([]StepErrorCall, len(mockCallback.OnStepErrorCalls))
	copy(stepErrorCalls, mockCallback.OnStepErrorCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 2 {
		t.Errorf("Expected 2 OnStepStart calls, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 2 {
		t.Errorf("Expected 2 OnStepComplete calls, got %d", onStepCompleteCalls)
	}

	// Verify steps were called (order may vary due to parallel execution)
	// Find the test-action step
	var testActionComplete *StepCompleteCall
	var failingActionComplete *StepCompleteCall
	
	for _, call := range stepCompleteCalls {
		if call.StepName == "test-action" {
			testActionComplete = &call
		} else if call.StepName == "failing-action" {
			failingActionComplete = &call
		}
	}
	
	if testActionComplete == nil {
		t.Errorf("Expected 'test-action' step to complete, but it wasn't found")
	} else if testActionComplete.Status != StepStatusOK {
		t.Errorf("Expected 'test-action' step status to be OK, got %v", testActionComplete.Status)
	}
	
	if failingActionComplete == nil {
		t.Errorf("Expected 'failing-action' step to complete, but it wasn't found")
	} else if failingActionComplete.Status != StepStatusWarn {
		t.Errorf("Expected 'failing-action' step status to be Warn (due to onerror: warn), got %v", failingActionComplete.Status)
	}

	// Verify error callback was NOT called (because onerror: warn converts error to warning)
	if onStepErrorCalls != 0 {
		t.Errorf("Expected 0 OnStepError calls (due to onerror: warn), got %d", onStepErrorCalls)
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	stepStartCalls := make([]string, len(mockCallback.OnStepStartCalls))
	copy(stepStartCalls, mockCallback.OnStepStartCalls)
	stepCompleteCalls := make([]StepCompleteCall, len(mockCallback.OnStepCompleteCalls))
	copy(stepCompleteCalls, mockCallback.OnStepCompleteCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", onStepCompleteCalls)
	}

	if stepStartCalls[0] != "test-action" {
		t.Errorf("Expected step to be 'test-action', got '%s'", stepStartCalls[0])
	}

	complete := stepCompleteCalls[0]
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	stepStartCalls := make([]string, len(mockCallback.OnStepStartCalls))
	copy(stepStartCalls, mockCallback.OnStepStartCalls)
	stepCompleteCalls := make([]StepCompleteCall, len(mockCallback.OnStepCompleteCalls))
	copy(stepCompleteCalls, mockCallback.OnStepCompleteCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", onStepCompleteCalls)
	}

	if stepStartCalls[0] != "test-action" {
		t.Errorf("Expected step to be 'test-action', got '%s'", stepStartCalls[0])
	}

	complete := stepCompleteCalls[0]
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	onStepOutputCalls := len(mockCallback.OnStepOutputCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", onStepCompleteCalls)
	}

	// OnStepOutput should not be called when not verbose
	if onStepOutputCalls != 0 {
		t.Errorf("Expected 0 OnStepOutput calls when not verbose, got %d", onStepOutputCalls)
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	stepStartCalls := make([]string, len(mockCallback.OnStepStartCalls))
	copy(stepStartCalls, mockCallback.OnStepStartCalls)
	stepCompleteCalls := make([]StepCompleteCall, len(mockCallback.OnStepCompleteCalls))
	copy(stepCompleteCalls, mockCallback.OnStepCompleteCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", onStepCompleteCalls)
	}

	if stepStartCalls[0] != "version-check" {
		t.Errorf("Expected step to be 'version-check', got '%s'", stepStartCalls[0])
	}

	complete := stepCompleteCalls[0]
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
	mockCallback.mu.Lock()
	onStepStartCalls := len(mockCallback.OnStepStartCalls)
	onStepCompleteCalls := len(mockCallback.OnStepCompleteCalls)
	onStepOutputCalls := len(mockCallback.OnStepOutputCalls)
	stepOutputCalls := make([]StepOutputCall, len(mockCallback.OnStepOutputCalls))
	copy(stepOutputCalls, mockCallback.OnStepOutputCalls)
	mockCallback.mu.Unlock()

	if onStepStartCalls != 1 {
		t.Errorf("Expected 1 OnStepStart call, got %d", onStepStartCalls)
	}

	if onStepCompleteCalls != 1 {
		t.Errorf("Expected 1 OnStepComplete call, got %d", onStepCompleteCalls)
	}

	// Should have output callbacks in verbose mode
	if onStepOutputCalls == 0 {
		t.Errorf("Expected OnStepOutput calls in verbose mode, got %d", onStepOutputCalls)
	}

	// Verify output contains expected content
	outputFound := false
	for _, outputCall := range stepOutputCalls {
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