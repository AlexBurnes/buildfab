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
			expected:  "Hello ${{ undefined }}",
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
