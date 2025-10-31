package buildfab

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TrackStepStatusCallback is a test helper that tracks step statuses
type TrackStepStatusCallback struct {
	*MockStepCallback
	stepStatuses map[string]Status
	mu           sync.Mutex
}

func NewTrackStepStatusCallback() *TrackStepStatusCallback {
	return &TrackStepStatusCallback{
		MockStepCallback: &MockStepCallback{},
		stepStatuses:     make(map[string]Status),
	}
}

func (c *TrackStepStatusCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
	c.MockStepCallback.OnStepComplete(ctx, stepName, status, message, duration, bufferedOutput)
	c.mu.Lock()
	defer c.mu.Unlock()
	if status == StepStatusOK {
		c.stepStatuses[stepName] = StatusOK
	} else if status == StepStatusError {
		c.stepStatuses[stepName] = StatusError
	} else if status == StepStatusSkipped {
		c.stepStatuses[stepName] = StatusSkipped
	}
}

func (c *TrackStepStatusCallback) GetResults() []StepResult {
	return c.MockStepCallback.GetResults()
}

// TestDAGSkipPropagation_DirectFailure tests that steps depending on failed steps are skipped
func TestDAGSkipPropagation_DirectFailure(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "false"}, // This will fail
			{Name: "step2", Run: "echo step2"},
			{Name: "step3", Run: "echo step3"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2", Require: []string{"step1"}}, // Depends on step1
					{Action: "step3", Require: []string{"step2"}}, // Depends on step2
				},
			},
		},
	}

	callback := NewTrackStepStatusCallback()

	opts := &RunOptions{
		StepCallback: callback,
		VerboseLevel: 1,
	}

	runner := NewRunner(config, opts)
	ctx := context.Background()

	err := runner.RunStage(ctx, "test-stage")
	// Stage should fail because step1 failed
	if err == nil {
		t.Error("RunStage should return error when step1 fails")
	}

	// Verify step1 failed
	if callback.stepStatuses["step1"] != StatusError {
		t.Errorf("step1 should have StatusError, got %v", callback.stepStatuses["step1"])
	}

	// Verify step2 was skipped (depends on failed step1)
	if callback.stepStatuses["step2"] != StatusSkipped {
		t.Errorf("step2 should have StatusSkipped (depends on failed step1), got %v", callback.stepStatuses["step2"])
	}

	// Verify step3 was skipped (depends on skipped step2)
	if callback.stepStatuses["step3"] != StatusSkipped {
		t.Errorf("step3 should have StatusSkipped (depends on skipped step2), got %v", callback.stepStatuses["step3"])
	}
}

// TestDAGSkipPropagation_TransitiveSkip tests transitive skip propagation
func TestDAGSkipPropagation_TransitiveSkip(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "false"}, // This will fail
			{Name: "step2", Run: "echo step2"},
			{Name: "step3", Run: "echo step3"},
			{Name: "step4", Run: "echo step4"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2", Require: []string{"step1"}}, // Depends on step1 (will be skipped)
					{Action: "step3", Require: []string{"step2"}}, // Depends on step2 (will be skipped)
					{Action: "step4", Require: []string{"step3"}}, // Depends on step3 (will be skipped)
				},
			},
		},
	}

	callback := NewTrackStepStatusCallback()

	opts := &RunOptions{
		StepCallback: callback,
		VerboseLevel: 1,
	}

	runner := NewRunner(config, opts)
	ctx := context.Background()

	err := runner.RunStage(ctx, "test-stage")
	// Stage should fail
	if err == nil {
		t.Error("RunStage should return error when step1 fails")
	}

	// Verify all dependent steps are skipped
	if callback.stepStatuses["step1"] != StatusError {
		t.Errorf("step1 should have StatusError, got %v", callback.stepStatuses["step1"])
	}
	if callback.stepStatuses["step2"] != StatusSkipped {
		t.Errorf("step2 should have StatusSkipped, got %v", callback.stepStatuses["step2"])
	}
	if callback.stepStatuses["step3"] != StatusSkipped {
		t.Errorf("step3 should have StatusSkipped (transitive skip), got %v", callback.stepStatuses["step3"])
	}
	if callback.stepStatuses["step4"] != StatusSkipped {
		t.Errorf("step4 should have StatusSkipped (transitive skip), got %v", callback.stepStatuses["step4"])
	}
}

// TestDAGSkipPropagation_MultipleDependencies tests skip when multiple dependencies fail
func TestDAGSkipPropagation_MultipleDependencies(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "false"}, // This will fail
			{Name: "step2", Run: "false"}, // This will also fail
			{Name: "step3", Run: "echo step3"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2"},
					{Action: "step3", Require: []string{"step1", "step2"}}, // Depends on both
				},
			},
		},
	}

	callback := NewTrackStepStatusCallback()

	opts := &RunOptions{
		StepCallback: callback,
		VerboseLevel: 1,
	}

	runner := NewRunner(config, opts)
	ctx := context.Background()

	err := runner.RunStage(ctx, "test-stage")
	// Stage should fail
	if err == nil {
		t.Error("RunStage should return error when dependencies fail")
	}

	// Verify step3 was skipped when both dependencies failed
	if callback.stepStatuses["step1"] != StatusError {
		t.Errorf("step1 should have StatusError, got %v", callback.stepStatuses["step1"])
	}
	if callback.stepStatuses["step2"] != StatusError {
		t.Errorf("step2 should have StatusError, got %v", callback.stepStatuses["step2"])
	}
	if callback.stepStatuses["step3"] != StatusSkipped {
		t.Errorf("step3 should have StatusSkipped (both dependencies failed), got %v", callback.stepStatuses["step3"])
	}
}

// TestDAGSkipPropagation_PartialFailure tests skip when one dependency fails but others succeed
func TestDAGSkipPropagation_PartialFailure(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "echo step1"},
			{Name: "step2", Run: "false"}, // This will fail
			{Name: "step3", Run: "echo step3"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2"},
					{Action: "step3", Require: []string{"step1", "step2"}}, // Depends on both
				},
			},
		},
	}

	callback := NewTrackStepStatusCallback()

	opts := &RunOptions{
		StepCallback: callback,
		VerboseLevel: 1,
	}

	runner := NewRunner(config, opts)
	ctx := context.Background()

	err := runner.RunStage(ctx, "test-stage")
	// Stage should fail
	if err == nil {
		t.Error("RunStage should return error when step2 fails")
	}

	// Verify step1 succeeded
	if callback.stepStatuses["step1"] != StatusOK {
		t.Errorf("step1 should have StatusOK, got %v", callback.stepStatuses["step1"])
	}

	// Verify step2 failed
	if callback.stepStatuses["step2"] != StatusError {
		t.Errorf("step2 should have StatusError, got %v", callback.stepStatuses["step2"])
	}

	// Verify step3 was skipped (one dependency failed)
	if callback.stepStatuses["step3"] != StatusSkipped {
		t.Errorf("step3 should have StatusSkipped (step2 failed), got %v", callback.stepStatuses["step3"])
	}
}

// TestDAGSkipPropagation_ComplexChain tests a complex dependency chain with mixed failures
func TestDAGSkipPropagation_ComplexChain(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "echo step1"},
			{Name: "step2", Run: "false"}, // This will fail
			{Name: "step3", Run: "echo step3"},
			{Name: "step4", Run: "echo step4"},
			{Name: "step5", Run: "echo step5"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2", Require: []string{"step1"}}, // Depends on step1, will fail
					{Action: "step3", Require: []string{"step2"}}, // Depends on step2, will be skipped
					{Action: "step4", Require: []string{"step3"}}, // Depends on step3, will be skipped
					{Action: "step5", Require: []string{"step1"}}, // Depends on step1 (succeeds), should run
				},
			},
		},
	}

	callback := NewTrackStepStatusCallback()

	opts := &RunOptions{
		StepCallback: callback,
		VerboseLevel: 1,
	}

	runner := NewRunner(config, opts)
	ctx := context.Background()

	err := runner.RunStage(ctx, "test-stage")
	// Stage should fail
	if err == nil {
		t.Error("RunStage should return error when step2 fails")
	}

	// Verify step1 succeeded
	if callback.stepStatuses["step1"] != StatusOK {
		t.Errorf("step1 should have StatusOK, got %v", callback.stepStatuses["step1"])
	}

	// Verify step2 failed
	if callback.stepStatuses["step2"] != StatusError {
		t.Errorf("step2 should have StatusError, got %v", callback.stepStatuses["step2"])
	}

	// Verify step3 was skipped (depends on failed step2)
	if callback.stepStatuses["step3"] != StatusSkipped {
		t.Errorf("step3 should have StatusSkipped, got %v", callback.stepStatuses["step3"])
	}

	// Verify step4 was skipped (depends on skipped step3)
	if callback.stepStatuses["step4"] != StatusSkipped {
		t.Errorf("step4 should have StatusSkipped (transitive skip), got %v", callback.stepStatuses["step4"])
	}

	// Verify step5 succeeded (depends on successful step1)
	if callback.stepStatuses["step5"] != StatusOK {
		t.Errorf("step5 should have StatusOK (depends on successful step1), got %v", callback.stepStatuses["step5"])
	}
}

// TestHasFailedOrSkippedDependency tests the helper function directly
func TestHasFailedOrSkippedDependency(t *testing.T) {
	runner := &Runner{}

	node := &DAGNode{
		Dependencies: []string{"dep1", "dep2", "dep3"},
	}

	tests := []struct {
		name     string
		status   map[string]Status
		expected bool
	}{
		{
			name: "no failed or skipped dependencies",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusOK,
				"dep3": StatusOK,
			},
			expected: false,
		},
		{
			name: "one failed dependency",
			status: map[string]Status{
				"dep1": StatusError,
				"dep2": StatusOK,
				"dep3": StatusOK,
			},
			expected: true,
		},
		{
			name: "one skipped dependency",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusSkipped,
				"dep3": StatusOK,
			},
			expected: true,
		},
		{
			name: "multiple failed/skipped dependencies",
			status: map[string]Status{
				"dep1": StatusError,
				"dep2": StatusSkipped,
				"dep3": StatusOK,
			},
			expected: true,
		},
		{
			name: "pending dependency",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusPending,
				"dep3": StatusOK,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.hasFailedOrSkippedDependency(node, tt.status)
			if result != tt.expected {
				t.Errorf("hasFailedOrSkippedDependency() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAllDependenciesCompleted tests the helper function directly
func TestAllDependenciesCompleted(t *testing.T) {
	runner := &Runner{}

	node := &DAGNode{
		Dependencies: []string{"dep1", "dep2", "dep3"},
	}

	tests := []struct {
		name     string
		status   map[string]Status
		expected bool
	}{
		{
			name: "all dependencies completed successfully",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusOK,
				"dep3": StatusOK,
			},
			expected: true,
		},
		{
			name: "one dependency failed",
			status: map[string]Status{
				"dep1": StatusError,
				"dep2": StatusOK,
				"dep3": StatusOK,
			},
			expected: false,
		},
		{
			name: "one dependency skipped",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusSkipped,
				"dep3": StatusOK,
			},
			expected: false,
		},
		{
			name: "pending dependency",
			status: map[string]Status{
				"dep1": StatusOK,
				"dep2": StatusOK,
				"dep3": StatusPending,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.allDependenciesCompleted(node, tt.status)
			if result != tt.expected {
				t.Errorf("allDependenciesCompleted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

