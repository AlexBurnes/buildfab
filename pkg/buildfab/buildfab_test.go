package buildfab

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPending, "PENDING"},
		{StatusRunning, "RUNNING"},
		{StatusOK, "OK"},
		{StatusWarn, "WARN"},
		{StatusError, "ERROR"},
		{StatusSkipped, "SKIPPED"},
		{Status(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions()

	if opts.ConfigPath != ".project.yml" {
		t.Errorf("ConfigPath = %v, want %v", opts.ConfigPath, ".project.yml")
	}
	if opts.MaxParallel != runtime.NumCPU() {
		t.Errorf("MaxParallel = %v, want %v", opts.MaxParallel, runtime.NumCPU())
	}
	if opts.Verbose != true {
		t.Errorf("Verbose = %v, want %v", opts.Verbose, true)
	}
	if opts.Debug != false {
		t.Errorf("Debug = %v, want %v", opts.Debug, false)
	}
	if opts.Variables == nil {
		t.Error("Variables should not be nil")
	}
	if opts.WorkingDir != "." {
		t.Errorf("WorkingDir = %v, want %v", opts.WorkingDir, ".")
	}
	if opts.Output != os.Stdout {
		t.Errorf("Output = %v, want %v", opts.Output, os.Stdout)
	}
	if opts.ErrorOutput != os.Stderr {
		t.Errorf("ErrorOutput = %v, want %v", opts.ErrorOutput, os.Stderr)
	}
	if opts.Only == nil {
		t.Error("Only should not be nil")
	}
	if opts.WithRequires != false {
		t.Errorf("WithRequires = %v, want %v", opts.WithRequires, false)
	}
}

func TestNewRunner(t *testing.T) {
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
	}

	// Test with nil options
	runner := NewRunner(config, nil)
	if runner.config != config {
		t.Errorf("config = %v, want %v", runner.config, config)
	}
	if runner.opts == nil {
		t.Error("opts should not be nil")
	}

	// Test with custom options
	opts := &RunOptions{
		ConfigPath: "custom.yml",
		Verbose:    true,
	}
	runner2 := NewRunner(config, opts)
	if runner2.config != config {
		t.Errorf("config = %v, want %v", runner2.config, config)
	}
	if runner2.opts != opts {
		t.Errorf("opts = %v, want %v", runner2.opts, opts)
	}
}

func TestConfig_GetAction(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "action1", Run: "echo hello"},
			{Name: "action2", Uses: "git@untracked"},
		},
	}

	// Test existing action
	action, exists := config.GetAction("action1")
	if !exists {
		t.Error("action1 should exist")
	}
	if action.Name != "action1" {
		t.Errorf("action.Name = %v, want %v", action.Name, "action1")
	}

	// Test non-existing action
	_, exists = config.GetAction("nonexistent")
	if exists {
		t.Error("nonexistent action should not exist")
	}
}

func TestConfig_GetStage(t *testing.T) {
	config := &Config{
		Stages: map[string]Stage{
			"stage1": {
				Steps: []Step{
					{Action: "action1"},
				},
			},
		},
	}

	// Test existing stage
	stage, exists := config.GetStage("stage1")
	if !exists {
		t.Error("stage1 should exist")
	}
	if len(stage.Steps) != 1 {
		t.Errorf("stage.Steps length = %v, want %v", len(stage.Steps), 1)
	}

	// Test non-existing stage
	_, exists = config.GetStage("nonexistent")
	if exists {
		t.Error("nonexistent stage should not exist")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {
						Steps: []Step{
							{Action: "action1"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing project name",
			config: &Config{
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
			},
			wantErr: true,
			errMsg:  "project name is required",
		},
		{
			name: "no actions",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
			},
			wantErr: true,
			errMsg:  "at least one action is required",
		},
		{
			name: "action without name",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Run: "echo hello"},
				},
			},
			wantErr: true,
			errMsg:  "action name is required",
		},
		{
			name: "action without run or uses",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1"},
				},
			},
			wantErr: true,
			errMsg:  "action action1 must have either 'run' or 'uses'",
		},
		{
			name: "action with both run and uses",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello", Uses: "git@untracked"},
				},
			},
			wantErr: true,
			errMsg:  "action action1 cannot have both 'run' and 'uses'",
		},
		{
			name: "duplicate action name",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
					{Name: "action1", Run: "echo world"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate action name: action1",
		},
		{
			name: "stage without steps",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {},
				},
			},
			wantErr: true,
			errMsg:  "stage stage1 must have at least one step",
		},
		{
			name: "step without action",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {
						Steps: []Step{
							{},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "step 1 in stage stage1 must have an action",
		},
		{
			name: "step with unknown action",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {
						Steps: []Step{
							{Action: "unknown-action"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "step 1 in stage stage1 references unknown action: unknown-action",
		},
		{
			name: "step with invalid onerror",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {
						Steps: []Step{
							{Action: "action1", OnError: "invalid"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "step 1 in stage stage1 has invalid onerror value: invalid (must be 'stop' or 'warn')",
		},
		{
			name: "step with invalid only value",
			config: &Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{Name: "action1", Run: "echo hello"},
				},
				Stages: map[string]Stage{
					"stage1": {
						Steps: []Step{
							{Action: "action1", Only: []string{"invalid"}},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "step 1 in stage stage1 has invalid only value: invalid (must be 'release', 'prerelease', 'patch', 'minor', or 'major')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestRunner_RunStage(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "action1", Run: "echo action1"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "action1"},
				},
			},
		},
	}
	runner := NewRunner(config, nil)

	// Test existing stage (should now work)
	err := runner.RunStage(context.Background(), "test-stage")
	if err != nil {
		t.Errorf("RunStage() unexpected error: %v", err)
	}

	// Test non-existing stage
	err = runner.RunStage(context.Background(), "nonexistent")
	if err == nil {
		t.Error("RunStage() expected error, got nil")
	}
	if err.Error() != "stage not found: nonexistent" {
		t.Errorf("RunStage() error = %v, want %v", err.Error(), "stage not found: nonexistent")
	}
}

func TestRunner_RunAction(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "test-action", Run: "echo hello"},
		},
	}
	runner := NewRunner(config, nil)

	// Test existing action (should now work)
	err := runner.RunAction(context.Background(), "test-action")
	if err != nil {
		t.Errorf("RunAction() unexpected error: %v", err)
	}

	// Test non-existing action
	err = runner.RunAction(context.Background(), "nonexistent")
	if err == nil {
		t.Error("RunAction() expected error, got nil")
	}
	if err.Error() != "action not found: nonexistent" {
		t.Errorf("RunAction() error = %v, want %v", err.Error(), "action not found: nonexistent")
	}
}

func TestRunner_RunStageStep(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{Name: "step1", Run: "echo step1"},
			{Name: "step2", Run: "echo step2"},
		},
		Stages: map[string]Stage{
			"test-stage": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2"},
				},
			},
		},
	}
	runner := NewRunner(config, nil)

	// Test existing step (should now work)
	err := runner.RunStageStep(context.Background(), "test-stage", "step1")
	if err != nil {
		t.Errorf("RunStageStep() unexpected error: %v", err)
	}

	// Test non-existing stage
	err = runner.RunStageStep(context.Background(), "nonexistent", "step1")
	if err == nil {
		t.Error("RunStageStep() expected error, got nil")
	}
	if err.Error() != "stage not found: nonexistent" {
		t.Errorf("RunStageStep() error = %v, want %v", err.Error(), "stage not found: nonexistent")
	}

	// Test non-existing step
	err = runner.RunStageStep(context.Background(), "test-stage", "nonexistent")
	if err == nil {
		t.Error("RunStageStep() expected error, got nil")
	}
	if err.Error() != "step not found: nonexistent in stage test-stage" {
		t.Errorf("RunStageStep() error = %v, want %v", err.Error(), "step not found: nonexistent in stage test-stage")
	}
}

func TestRunCLI(t *testing.T) {
	// Test CLI function with no arguments (should return error)
	err := RunCLI(context.Background(), []string{})
	if err == nil {
		t.Error("RunCLI() expected error for no arguments, got nil")
	}
	if !strings.Contains(err.Error(), "no arguments provided") {
		t.Errorf("RunCLI() error = %v, want error containing 'no arguments provided'", err.Error())
	}
	
	// Test CLI function with unknown command (should return error)
	err = RunCLI(context.Background(), []string{"unknown"})
	if err == nil {
		t.Error("RunCLI() expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("RunCLI() error = %v, want error containing 'unknown command'", err.Error())
	}
	
	// Test CLI function with run command but no stage (should return error)
	err = RunCLI(context.Background(), []string{"run"})
	if err == nil {
		t.Error("RunCLI() expected error for run without stage, got nil")
	}
	if !strings.Contains(err.Error(), "run command requires a stage name") {
		t.Errorf("RunCLI() error = %v, want error containing 'run command requires a stage name'", err.Error())
	}
}