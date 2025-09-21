package buildfab

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Config represents the buildfab configuration loaded from YAML
type Config struct {
	Project struct {
		Name    string   `yaml:"name"`
		Modules []string `yaml:"modules"`
		BinDir  string   `yaml:"bin,omitempty"`
	} `yaml:"project"`
	
	Actions []Action `yaml:"actions"`
	Stages  map[string]Stage `yaml:"stages"`
}

// Action represents a single action that can be executed
type Action struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run,omitempty"`
	Uses string `yaml:"uses,omitempty"`
}

// Stage represents a collection of steps to execute
type Stage struct {
	Steps []Step `yaml:"steps"`
}

// Step represents a single step in a stage
type Step struct {
	Action  string   `yaml:"action"`
	Require []string `yaml:"require,omitempty"`
	OnError string   `yaml:"onerror,omitempty"`
	If      string   `yaml:"if,omitempty"`
	Only    []string `yaml:"only,omitempty"`
}

// Result represents the result of executing a step
type Result struct {
	Name    string
	Status  Status
	Message string
	Error   error
	Duration time.Duration
}

// Status represents the execution status of a step
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusOK
	StatusWarn
	StatusError
	StatusSkipped
)

// String returns the string representation of the status
func (s Status) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusRunning:
		return "RUNNING"
	case StatusOK:
		return "OK"
	case StatusWarn:
		return "WARN"
	case StatusError:
		return "ERROR"
	case StatusSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

// RunOptions configures stage execution
type RunOptions struct {
	ConfigPath  string            // Path to project.yml (default: ".project.yml")
	MaxParallel int               // Maximum parallel execution (default: CPU count)
	Verbose     bool              // Enable verbose output
	Debug       bool              // Enable debug output
	Variables   map[string]string // Additional variables for interpolation
	WorkingDir  string            // Working directory for execution
	Output      interface{}       // Output writer (default: os.Stdout)
	ErrorOutput interface{}       // Error output writer (default: os.Stderr)
	Only        []string          // Only run steps matching these labels
	WithRequires bool             // Include required dependencies when running single step
}

// DefaultRunOptions returns default run options
func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		ConfigPath:  ".project.yml",
		MaxParallel: runtime.NumCPU(),
		Verbose:     false,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        []string{},
		WithRequires: false,
	}
}

// Runner provides the main execution interface
type Runner struct {
	config *Config
	opts   *RunOptions
}

// NewRunner creates a new buildfab runner
func NewRunner(config *Config, opts *RunOptions) *Runner {
	if opts == nil {
		opts = DefaultRunOptions()
	}
	return &Runner{
		config: config,
		opts:   opts,
	}
}

// RunStage executes a specific stage
func (r *Runner) RunStage(ctx context.Context, stageName string) error {
	_, exists := r.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// TODO: Implement stage execution with DAG
	return fmt.Errorf("stage execution not yet implemented")
}

// RunAction executes a specific action
func (r *Runner) RunAction(ctx context.Context, actionName string) error {
	_, exists := r.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	// TODO: Implement action execution
	return fmt.Errorf("action execution not yet implemented")
}

// RunStageStep executes a specific step within a stage
func (r *Runner) RunStageStep(ctx context.Context, stageName, stepName string) error {
	stage, exists := r.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// Find the step
	var targetStep *Step
	for i, step := range stage.Steps {
		if step.Action == stepName {
			targetStep = &stage.Steps[i]
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step not found: %s in stage %s", stepName, stageName)
	}

	// TODO: Implement step execution
	return fmt.Errorf("step execution not yet implemented")
}

// RunCLI executes the buildfab CLI with the given arguments
func RunCLI(ctx context.Context, args []string) error {
	// TODO: Implement CLI parsing and execution
	// This is a placeholder implementation
	fmt.Fprintf(os.Stderr, "buildfab CLI not yet implemented\n")
	fmt.Fprintf(os.Stderr, "Arguments: %v\n", args)
	return fmt.Errorf("not implemented")
}

// GetAction returns the action with the specified name
func (c *Config) GetAction(name string) (Action, bool) {
	for _, action := range c.Actions {
		if action.Name == name {
			return action, true
		}
	}
	return Action{}, false
}

// GetStage returns the stage with the specified name
func (c *Config) GetStage(name string) (Stage, bool) {
	stage, exists := c.Stages[name]
	return stage, exists
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Project.Name == "" {
		return fmt.Errorf("project name is required")
	}
	
	if len(c.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}
	
	// Validate actions
	actionNames := make(map[string]bool)
	for _, action := range c.Actions {
		if action.Name == "" {
			return fmt.Errorf("action name is required")
		}
		
		if action.Run == "" && action.Uses == "" {
			return fmt.Errorf("action %s must have either 'run' or 'uses'", action.Name)
		}
		
		if action.Run != "" && action.Uses != "" {
			return fmt.Errorf("action %s cannot have both 'run' and 'uses'", action.Name)
		}
		
		if actionNames[action.Name] {
			return fmt.Errorf("duplicate action name: %s", action.Name)
		}
		actionNames[action.Name] = true
	}
	
	// Validate stages
	for stageName, stage := range c.Stages {
		if len(stage.Steps) == 0 {
			return fmt.Errorf("stage %s must have at least one step", stageName)
		}
		
		for i, step := range stage.Steps {
			if step.Action == "" {
				return fmt.Errorf("step %d in stage %s must have an action", i+1, stageName)
			}
			
			if !actionNames[step.Action] {
				return fmt.Errorf("step %d in stage %s references unknown action: %s", i+1, stageName, step.Action)
			}
			
			if step.OnError != "" && step.OnError != "stop" && step.OnError != "warn" {
				return fmt.Errorf("step %d in stage %s has invalid onerror value: %s (must be 'stop' or 'warn')", i+1, stageName, step.OnError)
			}
			
			// Validate only field contains valid values
			for _, onlyValue := range step.Only {
				if onlyValue != "release" && onlyValue != "prerelease" && onlyValue != "patch" && onlyValue != "minor" && onlyValue != "major" {
					return fmt.Errorf("step %d in stage %s has invalid only value: %s (must be 'release', 'prerelease', 'patch', 'minor', or 'major')", i+1, stageName, onlyValue)
				}
			}
		}
	}
	
	return nil
}