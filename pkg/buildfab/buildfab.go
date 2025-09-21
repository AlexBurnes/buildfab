package buildfab

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
	Output      io.Writer         // Output writer (default: os.Stdout)
	ErrorOutput io.Writer         // Error output writer (default: os.Stderr)
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

	// Use the internal executor for actual execution
	// We need to import the internal packages, but since this is a public API,
	// we'll create a simple implementation that works with the existing structure
	return r.runStageInternal(ctx, stageName)
}

// RunAction executes a specific action
func (r *Runner) RunAction(ctx context.Context, actionName string) error {
	action, exists := r.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	// Execute the action directly
	return r.runActionInternal(ctx, action)
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

	// Get the action and execute it
	action, exists := r.config.GetAction(targetStep.Action)
	if !exists {
		return fmt.Errorf("action not found: %s", targetStep.Action)
	}

	return r.runActionInternal(ctx, action)
}

// RunCLI executes the buildfab CLI with the given arguments
func RunCLI(ctx context.Context, args []string) error {
	// This function provides a programmatic way to run the buildfab CLI
	// It's primarily used for testing and embedding scenarios
	
	// For now, we'll provide a simple implementation that loads config and runs stages
	// In a full implementation, this would parse CLI arguments and delegate to appropriate functions
	
	if len(args) == 0 {
		return fmt.Errorf("no arguments provided")
	}
	
	// Simple argument parsing for common cases
	command := args[0]
	
	// Load default configuration
	configPath := ".project.yml"
	for i, arg := range args {
		if arg == "-c" || arg == "--config" {
			if i+1 < len(args) {
				configPath = args[i+1]
			}
			break
		}
	}
	
	// Load configuration
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create default options
	opts := DefaultRunOptions()
	opts.ConfigPath = configPath
	
	// Create runner
	runner := NewRunner(cfg, opts)
	
	// Handle different commands
	switch command {
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("run command requires a stage name")
		}
		stageName := args[1]
		return runner.RunStage(ctx, stageName)
		
	case "action":
		if len(args) < 2 {
			return fmt.Errorf("action command requires an action name")
		}
		actionName := args[1]
		return runner.RunAction(ctx, actionName)
		
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// loadConfig is a helper function to load configuration
func loadConfig(path string) (*Config, error) {
	// This is a simplified version - in practice, this would use the internal/config package
	// For now, we'll return a basic config
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "default-project",
		},
		Actions: []Action{},
		Stages:  map[string]Stage{},
	}
	
	// In a real implementation, this would load from the YAML file
	// For now, we'll return the empty config
	return config, nil
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

// runStageInternal executes a stage using a simplified DAG approach
func (r *Runner) runStageInternal(ctx context.Context, stageName string) error {
	stage, _ := r.config.GetStage(stageName)
	
	// Simple sequential execution for now
	// In a full implementation, this would use the DAG executor
	for _, step := range stage.Steps {
		action, exists := r.config.GetAction(step.Action)
		if !exists {
			return fmt.Errorf("action not found: %s", step.Action)
		}
		
		err := r.runActionInternal(ctx, action)
		if err != nil {
			// Check error policy
			if step.OnError == "warn" {
				// Log warning but continue
				if r.opts.Verbose {
					fmt.Fprintf(r.opts.ErrorOutput, "Warning: step %s failed: %v\n", step.Action, err)
				}
				continue
			}
			// Default is "stop" - return error
			return fmt.Errorf("step %s failed: %w", step.Action, err)
		}
	}
	
	return nil
}

// runActionInternal executes a single action
func (r *Runner) runActionInternal(ctx context.Context, action Action) error {
	if action.Uses != "" {
		return r.runBuiltInAction(ctx, action)
	}
	
	return r.runCustomAction(ctx, action)
}

// runBuiltInAction executes a built-in action
func (r *Runner) runBuiltInAction(ctx context.Context, action Action) error {
	// For now, return an error for built-in actions since we can't import internal packages
	// In a full implementation, this would use the action registry
	return fmt.Errorf("built-in action %s not supported in public API", action.Uses)
}

// runCustomAction executes a custom action with run command
func (r *Runner) runCustomAction(ctx context.Context, action Action) error {
	if action.Run == "" {
		return fmt.Errorf("action %s has no run command", action.Name)
	}
	
	// Create command
	cmd := exec.CommandContext(ctx, "sh", "-c", action.Run)
	cmd.Dir = r.opts.WorkingDir
	
	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range r.opts.Variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Execute command
	err := cmd.Run()
	
	// Print output if verbose
	if r.opts.Verbose {
		if stdout.Len() > 0 {
			fmt.Fprintf(r.opts.Output, "Output: %s\n", stdout.String())
		}
		if stderr.Len() > 0 {
			fmt.Fprintf(r.opts.ErrorOutput, "Error: %s\n", stderr.String())
		}
	}
	
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	
	return nil
}