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
	StepCallback StepCallback     // Optional callback for step execution events
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

// ActionRegistry defines the interface for built-in action execution
type ActionRegistry interface {
	GetRunner(name string) (ActionRunner, bool)
	ListActions() map[string]string
}

// ActionRunner defines the interface for action runners
type ActionRunner interface {
	Run(ctx context.Context) (Result, error)
	Description() string
}

// Runner provides the main execution interface
type Runner struct {
	config   *Config
	opts     *RunOptions
	registry ActionRegistry
}

// NewRunner creates a new buildfab runner with default built-in actions
func NewRunner(config *Config, opts *RunOptions) *Runner {
	return NewRunnerWithRegistry(config, opts, NewDefaultActionRegistry())
}

// NewRunnerWithRegistry creates a new buildfab runner with a custom action registry
func NewRunnerWithRegistry(config *Config, opts *RunOptions, registry ActionRegistry) *Runner {
	if opts == nil {
		opts = DefaultRunOptions()
	}
	return &Runner{
		config:   config,
		opts:     opts,
		registry: registry,
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
	// Check if it's a built-in action first
	if runner, exists := r.registry.GetRunner(actionName); exists {
		// Call step start callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepStart(ctx, actionName)
		}

		start := time.Now()
		_, err := runner.Run(ctx)
		duration := time.Since(start)

		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			status := StepStatusOK
			message := "executed successfully"
			
			if err != nil {
				status = StepStatusError
				message = err.Error()
				r.opts.StepCallback.OnStepError(ctx, actionName, err)
			}
			
			r.opts.StepCallback.OnStepComplete(ctx, actionName, status, message, duration)
		}

		return err
	}

	// Check if it's a custom action
	action, exists := r.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	// Call step start callback if provided
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, actionName)
	}

	start := time.Now()
	err := r.runActionInternal(ctx, action)
	duration := time.Since(start)

	// Call step complete callback if provided
	if r.opts.StepCallback != nil {
		status := StepStatusOK
		message := "executed successfully"
		
		if err != nil {
			status = StepStatusError
			message = err.Error()
			r.opts.StepCallback.OnStepError(ctx, actionName, err)
		}
		
		r.opts.StepCallback.OnStepComplete(ctx, actionName, status, message, duration)
	}

	return err
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

	// Call step start callback if provided
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, stepName)
	}

	start := time.Now()
	err := r.runActionInternal(ctx, action)
	duration := time.Since(start)

	// Call step complete callback if provided
	if r.opts.StepCallback != nil {
		status := StepStatusOK
		message := "executed successfully"
		
		if err != nil {
			status = StepStatusError
			message = err.Error()
			r.opts.StepCallback.OnStepError(ctx, stepName, err)
		}
		
		r.opts.StepCallback.OnStepComplete(ctx, stepName, status, message, duration)
	}

	return err
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
	
	// Check command validity before loading config
	switch command {
	case "run":
		if len(args) < 2 {
			return fmt.Errorf("run command requires a stage name")
		}
	case "action":
		if len(args) < 2 {
			return fmt.Errorf("action command requires an action name")
		}
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	
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
		stageName := args[1]
		return runner.RunStage(ctx, stageName)
		
	case "action":
		actionName := args[1]
		return runner.RunAction(ctx, actionName)
		
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// loadConfig is a helper function to load configuration
func loadConfig(path string) (*Config, error) {
	return LoadConfig(path)
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

// ListBuiltInActions returns all available built-in actions
func (r *Runner) ListBuiltInActions() map[string]string {
	if r.registry == nil {
		return make(map[string]string)
	}
	return r.registry.ListActions()
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
		
		// Call step start callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepStart(ctx, step.Action)
		}
		
		start := time.Now()
		err := r.runActionInternal(ctx, action)
		duration := time.Since(start)
		
		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			status := StepStatusOK
			message := "executed successfully"
			
			if err != nil {
				status = StepStatusError
				// Use the original error message for the callback, not the wrapped one
				message = err.Error()
				r.opts.StepCallback.OnStepError(ctx, step.Action, err)
			}
			
			r.opts.StepCallback.OnStepComplete(ctx, step.Action, status, message, duration)
		}
		
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
	if r.registry == nil {
		return fmt.Errorf("built-in action %s not supported: no action registry provided", action.Uses)
	}
	
	runner, exists := r.registry.GetRunner(action.Uses)
	if !exists {
		return fmt.Errorf("unknown built-in action: %s", action.Uses)
	}
	
	result, err := runner.Run(ctx)
	if err != nil {
		return err
	}
	
	// Call step output callback if provided and verbose mode is enabled
	if r.opts.StepCallback != nil && r.opts.Verbose && result.Message != "" {
		r.opts.StepCallback.OnStepOutput(ctx, action.Name, result.Message)
	}
	
	// Print result if verbose mode is enabled
	if r.opts.Verbose {
		if result.Status == StatusOK {
			fmt.Fprintf(r.opts.Output, "✓ %s: %s\n", action.Name, result.Message)
		} else if result.Status == StatusWarn {
			fmt.Fprintf(r.opts.Output, "! %s: %s\n", action.Name, result.Message)
		} else if result.Status == StatusError {
			fmt.Fprintf(r.opts.ErrorOutput, "✗ %s: %s\n", action.Name, result.Message)
		}
	}
	
	// Return error if action failed
	if result.Status == StatusError {
		return fmt.Errorf("built-in action failed: %s", result.Message)
	}
	
	return nil
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
	
	// Call step output callback if provided and verbose mode is enabled
	if r.opts.StepCallback != nil && r.opts.Verbose {
		if stdout.Len() > 0 {
			r.opts.StepCallback.OnStepOutput(ctx, action.Name, stdout.String())
		}
		if stderr.Len() > 0 {
			r.opts.StepCallback.OnStepOutput(ctx, action.Name, stderr.String())
		}
	}
	
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
		// Provide better error message with reproduction instructions
		return fmt.Errorf("failed, to check run:\n  %s", action.Run)
	}
	
	return nil
}