package buildfab

import "fmt"

// ConfigurationError represents errors in project.yml configuration
type ConfigurationError struct {
	Message string
	Path    string
	Line    int
	Column  int
}

func (e *ConfigurationError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("configuration error in %s at line %d, column %d: %s", e.Path, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("configuration error: %s", e.Message)
}

// ExecutionError represents errors during step execution
type ExecutionError struct {
	StepName string
	Action   string
	Message  string
	Output   string
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("execution error in step %q (action %q): %s", e.StepName, e.Action, e.Message)
}

// DependencyError represents errors in dependency resolution
type DependencyError struct {
	Message string
	Cycle   []string
}

func (e *DependencyError) Error() string {
	if len(e.Cycle) > 0 {
		return fmt.Sprintf("dependency error: %s (cycle: %v)", e.Message, e.Cycle)
	}
	return fmt.Sprintf("dependency error: %s", e.Message)
}

// VariableError represents errors in variable interpolation
type VariableError struct {
	Variable string
	Message  string
}

func (e *VariableError) Error() string {
	return fmt.Sprintf("variable error for %q: %s", e.Variable, e.Message)
}