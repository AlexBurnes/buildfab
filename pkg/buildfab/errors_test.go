package buildfab

import (
	"testing"
)

func TestConfigurationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigurationError
		expected string
	}{
		{
			name: "with path and line info",
			err: &ConfigurationError{
				Message: "invalid YAML syntax",
				Path:    "project.yml",
				Line:    10,
				Column:  5,
			},
			expected: "configuration error in project.yml at line 10, column 5: invalid YAML syntax",
		},
		{
			name: "without path info",
			err: &ConfigurationError{
				Message: "missing required field",
			},
			expected: "configuration error: missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("ConfigurationError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExecutionError_Error(t *testing.T) {
	err := &ExecutionError{
		StepName: "build",
		Action:   "make",
		Message:  "command failed",
		Output:   "make: *** [target] Error 1",
	}

	expected := `execution error in step "build" (action "make"): command failed`
	if got := err.Error(); got != expected {
		t.Errorf("ExecutionError.Error() = %v, want %v", got, expected)
	}
}

func TestDependencyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DependencyError
		expected string
	}{
		{
			name: "with cycle",
			err: &DependencyError{
				Message: "circular dependency detected",
				Cycle:   []string{"step1", "step2", "step3", "step1"},
			},
			expected: "dependency error: circular dependency detected (cycle: [step1 step2 step3 step1])",
		},
		{
			name: "without cycle",
			err: &DependencyError{
				Message: "missing dependency",
			},
			expected: "dependency error: missing dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("DependencyError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVariableError_Error(t *testing.T) {
	err := &VariableError{
		Variable: "version.version",
		Message:  "version not found",
	}

	expected := `variable error for "version.version": version not found`
	if got := err.Error(); got != expected {
		t.Errorf("VariableError.Error() = %v, want %v", got, expected)
	}
}