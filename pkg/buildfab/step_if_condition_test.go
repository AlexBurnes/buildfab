package buildfab

import (
	"context"
	"testing"
)

func TestStepIfCondition(t *testing.T) {
	// Test configuration with step if conditions
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
				Run:  "echo 'Hello World'",
			},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{
						Action: "test-action",
						If:     "os == 'linux'",
					},
				},
			},
		},
	}

	// Create runner with variables
	runner := &Runner{
		config: config,
		opts: &RunOptions{
			Variables: map[string]string{
				"os": "linux",
			},
		},
	}

	// Test step with condition that should pass
	step := Step{
		Action: "test-action",
		If:     "os == 'linux'",
	}

	shouldExecute, err := runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if !shouldExecute {
		t.Error("shouldExecuteStepByCondition() = false, want true")
	}

	// Test step with condition that should fail
	step.If = "os == 'windows'"
	shouldExecute, err = runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if shouldExecute {
		t.Error("shouldExecuteStepByCondition() = true, want false")
	}

	// Test step without condition
	step.If = ""
	shouldExecute, err = runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if !shouldExecute {
		t.Error("shouldExecuteStepByCondition() = false, want true")
	}
}

func TestStepIfConditionComplex(t *testing.T) {
	// Test configuration with complex step if conditions
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
				Run:  "echo 'Hello World'",
			},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{
						Action: "test-action",
						If:     "os == 'linux' && arch == 'amd64'",
					},
				},
			},
		},
	}

	// Create runner with variables
	runner := &Runner{
		config: config,
		opts: &RunOptions{
			Variables: map[string]string{
				"os":   "linux",
				"arch": "amd64",
			},
		},
	}

	// Test step with complex condition that should pass
	step := Step{
		Action: "test-action",
		If:     "os == 'linux' && arch == 'amd64'",
	}

	shouldExecute, err := runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if !shouldExecute {
		t.Error("shouldExecuteStepByCondition() = false, want true")
	}

	// Test step with complex condition that should fail
	step.If = "os == 'linux' && arch == 'arm64'"
	shouldExecute, err = runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if shouldExecute {
		t.Error("shouldExecuteStepByCondition() = true, want false")
	}
}

func TestStepIfConditionWithFunctions(t *testing.T) {
	// Test configuration with step if conditions using functions
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
				Run:  "echo 'Hello World'",
			},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{
						Action: "test-action",
						If:     "contains(os, 'linux')",
					},
				},
			},
		},
	}

	// Create runner with variables
	runner := &Runner{
		config: config,
		opts: &RunOptions{
			Variables: map[string]string{
				"os": "linux",
			},
		},
	}

	// Test step with function condition that should pass
	step := Step{
		Action: "test-action",
		If:     "contains(os, 'linux')",
	}

	shouldExecute, err := runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if !shouldExecute {
		t.Error("shouldExecuteStepByCondition() = false, want true")
	}

	// Test step with function condition that should fail
	step.If = "contains(os, 'windows')"
	shouldExecute, err = runner.shouldExecuteStepByCondition(context.Background(), step)
	if err != nil {
		t.Fatalf("shouldExecuteStepByCondition() error = %v", err)
	}
	if shouldExecute {
		t.Error("shouldExecuteStepByCondition() = true, want false")
	}
}
