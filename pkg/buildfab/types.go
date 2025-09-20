package buildfab

import "time"

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
}

// StepStatus represents the execution status of a step
type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusRunning
	StepStatusOK
	StepStatusWarn
	StepStatusError
	StepStatusSkipped
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
	default:
		return "unknown"
	}
}