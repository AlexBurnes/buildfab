package buildfab

import (
	"testing"
	"time"
)

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   string
	}{
		{StepStatusPending, "pending"},
		{StepStatusRunning, "running"},
		{StepStatusOK, "ok"},
		{StepStatusWarn, "warn"},
		{StepStatusError, "error"},
		{StepStatusSkipped, "skipped"},
		{StepStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("StepStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStageResult(t *testing.T) {
	steps := []StepResult{
		{
			StepName:   "step1",
			ActionName: "action1",
			Status:     StepStatusOK,
			Duration:   100 * time.Millisecond,
			Output:     "success",
			Error:      nil,
		},
		{
			StepName:   "step2",
			ActionName: "action2",
			Status:     StepStatusError,
			Duration:   200 * time.Millisecond,
			Output:     "error output",
			Error:      &ExecutionError{StepName: "step2", Action: "action2", Message: "test error"},
		},
	}

	result := StageResult{
		StageName: "test-stage",
		Success:   false, // step2 failed
		Steps:     steps,
		Duration:  300 * time.Millisecond,
		Error:     nil,
	}

	// Test basic properties
	if result.StageName != "test-stage" {
		t.Errorf("StageName = %v, want %v", result.StageName, "test-stage")
	}
	if result.Success != false {
		t.Errorf("Success = %v, want %v", result.Success, false)
	}
	if result.Duration != 300*time.Millisecond {
		t.Errorf("Duration = %v, want %v", result.Duration, 300*time.Millisecond)
	}
	if len(result.Steps) != 2 {
		t.Errorf("Steps length = %v, want %v", len(result.Steps), 2)
	}
}

func TestStepResult(t *testing.T) {
	result := StepResult{
		StepName:   "test-step",
		ActionName: "test-action",
		Status:     StepStatusOK,
		Duration:   150 * time.Millisecond,
		Output:     "test output",
		Error:      nil,
	}

	// Test basic properties
	if result.StepName != "test-step" {
		t.Errorf("StepName = %v, want %v", result.StepName, "test-step")
	}
	if result.ActionName != "test-action" {
		t.Errorf("ActionName = %v, want %v", result.ActionName, "test-action")
	}
	if result.Status != StepStatusOK {
		t.Errorf("Status = %v, want %v", result.Status, StepStatusOK)
	}
	if result.Duration != 150*time.Millisecond {
		t.Errorf("Duration = %v, want %v", result.Duration, 150*time.Millisecond)
	}
	if result.Output != "test output" {
		t.Errorf("Output = %v, want %v", result.Output, "test output")
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want %v", result.Error, nil)
	}
}