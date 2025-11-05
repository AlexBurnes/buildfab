package buildfab

import (
	"context"
	"time"
)

// StageResult contains execution results for a stage
type StageResult struct {
	StageName string
	Success   bool
	Steps     []StepResult
	Duration  time.Duration
	Error     error
}

// StepResult contains execution results for a step
type StepResult struct {
	StepName   string
	ActionName string
	Status     StepStatus
	Duration   time.Duration
	Output     string
	Error      error
	Message    string // Custom message from action Result (for built-in actions)
}

// StepStatus represents the execution status of a step
type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusRunning
	StepStatusOK
	StepStatusWarn
	StepStatusError
	StepStatusSkipped            // Skipped due to dependency failure
	StepStatusSkippedCondition   // Skipped due to if condition not met (doesn't block dependents)
)

// String returns the string representation of StepStatus
func (s StepStatus) String() string {
	switch s {
	case StepStatusPending:
		return "pending"
	case StepStatusRunning:
		return "running"
	case StepStatusOK:
		return "ok"
	case StepStatusWarn:
		return "warn"
	case StepStatusError:
		return "error"
	case StepStatusSkipped:
		return "skipped"
	case StepStatusSkippedCondition:
		return "skipped"  // Display as "skipped" to users (internal distinction only)
	default:
		return "unknown"
	}
}

// StepCallback defines the interface for step execution callbacks
type StepCallback interface {
	// OnStepStart is called when a step starts execution
	OnStepStart(ctx context.Context, stepName string)
	
	// OnStepComplete is called when a step completes (success, warning, or error)
	OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string)
	
	// OnStepOutput is called for step output (when verbose mode is enabled)
	OnStepOutput(ctx context.Context, stepName string, output string)
	
	// OnStepError is called for step errors
	OnStepError(ctx context.Context, stepName string, err error)
	
	// GetResults returns the collected step results
	GetResults() []StepResult
}