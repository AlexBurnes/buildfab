package buildfab

import (
	"testing"
)

func TestInterpolateVariables(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		variables map[string]string
		expected  string
		wantErr   bool
	}{
		{
			name:      "simple variable",
			text:      "Hello ${{ name }}",
			variables: map[string]string{"name": "world"},
			expected:  "Hello world",
		},
		{
			name:      "multiple variables",
			text:      "Platform: ${{ platform }}, Arch: ${{ arch }}",
			variables: map[string]string{"platform": "linux", "arch": "amd64"},
			expected:  "Platform: linux, Arch: amd64",
		},
		{
			name:      "no variables",
			text:      "Hello world",
			variables: map[string]string{},
			expected:  "Hello world",
		},
		{
			name:      "undefined variable",
			text:      "Hello ${{ undefined }}",
			variables: map[string]string{"name": "world"},
			expected:  "",
			wantErr:   true,
		},
		{
			name:      "whitespace in variable",
			text:      "Hello ${{ name }}",
			variables: map[string]string{"name": "world"},
			expected:  "Hello world",
		},
		{
			name:      "platform variables",
			text:      "Building for ${{ platform }}-${{ arch }} on ${{ os }} ${{ os_version }} with ${{ cpu }} CPUs",
			variables: map[string]string{
				"platform":   "linux",
				"arch":       "amd64",
				"os":         "ubuntu",
				"os_version": "22.04",
				"cpu":        "8",
			},
			expected: "Building for linux-amd64 on ubuntu 22.04 with 8 CPUs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateVariables(tt.text, tt.variables)
			if tt.wantErr {
				if err == nil {
					t.Errorf("InterpolateVariables() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("InterpolateVariables() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("InterpolateVariables() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInterpolateAction(t *testing.T) {
	action := Action{
		Name: "test-action",
		Run:  "echo 'Platform: ${{ platform }}, Arch: ${{ arch }}'",
	}
	
	variables := map[string]string{
		"platform": "linux",
		"arch":     "amd64",
	}
	
	interpolated, err := InterpolateAction(action, variables)
	if err != nil {
		t.Errorf("InterpolateAction() error = %v", err)
		return
	}
	
	expected := "echo 'Platform: linux, Arch: amd64'"
	if interpolated.Run != expected {
		t.Errorf("InterpolateAction() = %v, want %v", interpolated.Run, expected)
	}
}

func TestInterpolateStep(t *testing.T) {
	step := Step{
		Action:  "test-${{ platform }}",
		Require: []string{"dep-${{ arch }}"},
		If:      "${{ platform }} == linux",
	}
	
	variables := map[string]string{
		"platform": "linux",
		"arch":     "amd64",
	}
	
	interpolated, err := InterpolateStep(step, variables)
	if err != nil {
		t.Errorf("InterpolateStep() error = %v", err)
		return
	}
	
	if interpolated.Action != "test-linux" {
		t.Errorf("InterpolateStep() Action = %v, want test-linux", interpolated.Action)
	}
	
	if len(interpolated.Require) != 1 || interpolated.Require[0] != "dep-amd64" {
		t.Errorf("InterpolateStep() Require = %v, want [dep-amd64]", interpolated.Require)
	}
	
	if interpolated.If != "linux == linux" {
		t.Errorf("InterpolateStep() If = %v, want linux == linux", interpolated.If)
	}
}

func TestMergeStepVariables(t *testing.T) {
	tests := []struct {
		name           string
		globalVars     map[string]string
		stepVars       map[string]string
		expectedVars   map[string]string
		checkKey       string
		expectedValue  string
	}{
		{
			name:       "no step variables",
			globalVars: map[string]string{"key1": "global1", "key2": "global2"},
			stepVars:   nil,
			checkKey:   "key1",
			expectedValue: "global1",
		},
		{
			name:       "step variables override global",
			globalVars: map[string]string{"key1": "global1", "key2": "global2"},
			stepVars:   map[string]string{"key1": "step1"},
			checkKey:   "key1",
			expectedValue: "step1",
		},
		{
			name:       "step variables add new keys",
			globalVars: map[string]string{"key1": "global1"},
			stepVars:   map[string]string{"key2": "step2"},
			checkKey:   "key2",
			expectedValue: "step2",
		},
		{
			name:       "step variables with matrix.image",
			globalVars: map[string]string{"platform": "linux"},
			stepVars:   map[string]string{"matrix.image": "registry.svc/burnes/development-env:9"},
			checkKey:   "matrix.image",
			expectedValue: "registry.svc/burnes/development-env:9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Project: Project{Name: "test"},
			}
			opts := &RunOptions{
				Variables: tt.globalVars,
			}
			runner := NewRunner(config, opts)
			
			step := Step{
				Action:    "test-action",
				Variables: tt.stepVars,
			}
			
			merged := runner.mergeStepVariables(step)
			
			if val, exists := merged[tt.checkKey]; !exists {
				t.Errorf("Expected key %s to exist in merged variables", tt.checkKey)
			} else if val != tt.expectedValue {
				t.Errorf("mergeStepVariables() key %s = %v, want %v", tt.checkKey, val, tt.expectedValue)
			}
			
			// Ensure original global variables are not modified
			if len(tt.stepVars) > 0 {
				if val, exists := runner.opts.Variables[tt.checkKey]; exists && tt.globalVars[tt.checkKey] != val {
					t.Errorf("Global variables were modified: %s = %v", tt.checkKey, val)
				}
			}
		})
	}
}

func TestWithStepVariables(t *testing.T) {
	tests := []struct {
		name           string
		globalVars     map[string]string
		stepVars       map[string]string
		checkInside    func(*Runner) error
		checkAfter     func(*Runner) error
	}{
		{
			name:       "variables restored after execution",
			globalVars: map[string]string{"key1": "global1"},
			stepVars:   map[string]string{"key1": "step1"},
			checkInside: func(r *Runner) error {
				if r.opts.Variables["key1"] != "step1" {
					t.Errorf("Inside function: key1 = %v, want step1", r.opts.Variables["key1"])
				}
				return nil
			},
			checkAfter: func(r *Runner) error {
				if r.opts.Variables["key1"] != "global1" {
					t.Errorf("After function: key1 = %v, want global1", r.opts.Variables["key1"])
				}
				return nil
			},
		},
		{
			name:       "no step variables - no changes",
			globalVars: map[string]string{"key1": "global1"},
			stepVars:   nil,
			checkInside: func(r *Runner) error {
				if r.opts.Variables["key1"] != "global1" {
					t.Errorf("Inside function: key1 = %v, want global1", r.opts.Variables["key1"])
				}
				return nil
			},
			checkAfter: func(r *Runner) error {
				if r.opts.Variables["key1"] != "global1" {
					t.Errorf("After function: key1 = %v, want global1", r.opts.Variables["key1"])
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Project: Project{Name: "test"},
			}
			opts := &RunOptions{
				Variables: tt.globalVars,
			}
			runner := NewRunner(config, opts)
			
			step := Step{
				Action:    "test-action",
				Variables: tt.stepVars,
			}
			
			err := runner.withStepVariables(step, func() error {
				return tt.checkInside(runner)
			})
			
			if err != nil {
				t.Errorf("withStepVariables() error = %v", err)
			}
			
			if err := tt.checkAfter(runner); err != nil {
				t.Errorf("checkAfter() error = %v", err)
			}
		})
	}
}
