package buildfab

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
	containerRunner "github.com/AlexBurnes/buildfab/internal/container"
)

// Color constants for output formatting
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Config represents the buildfab configuration loaded from YAML
type Config struct {
	Project struct {
		Name    string   `yaml:"name"`
		Modules []string `yaml:"modules"`
		BinDir  string   `yaml:"bin,omitempty"`
	} `yaml:"project"`
	
	Include []string          `yaml:"include,omitempty"` // File patterns to include
	Actions []Action          `yaml:"actions"`
	Stages  map[string]Stage  `yaml:"stages"`
}

// Action represents a single action that can be executed
type Action struct {
	Name      string                 `yaml:"name"`
	Run       string                 `yaml:"run,omitempty"`
	Uses      string                 `yaml:"uses,omitempty"`
	Shell     string                 `yaml:"shell,omitempty"` // Optional shell specification
	Variants  []ActionVariant        `yaml:"variants,omitempty"` // Optional variants for conditional execution
	Options   map[string]interface{} `yaml:"options,omitempty"` // Optional action options (e.g., delay)
	Container *container.ContainerConfig `yaml:"container,omitempty"` // Optional container configuration
}

// ActionVariant represents a conditional variant of an action
type ActionVariant struct {
	When  string `yaml:"when"`  // Condition expression (e.g., "${{ os == 'linux' }}")
	Run   string `yaml:"run,omitempty"`
	Uses  string `yaml:"uses,omitempty"`
	Shell string `yaml:"shell,omitempty"`
}

// Stage represents a collection of steps to execute
type Stage struct {
	Steps []Step `yaml:"steps"`
}

// Step represents a single step in a stage
type Step struct {
	Action      string   `yaml:"action"`
	Description string   `yaml:"description,omitempty"` // Step description for display
	Require     []string `yaml:"require,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty"` // Alternative to require
	OnError     string   `yaml:"onerror,omitempty"`
	If          string   `yaml:"if,omitempty"`
	Only        []string `yaml:"only,omitempty"`
	Matrix      *MatrixConfig `yaml:"matrix,omitempty"` // Matrix configuration for this step
}

// MatrixConfig represents matrix configuration for parallel execution
type MatrixConfig struct {
	Values   map[string][]interface{} `yaml:"values"`   // Matrix values (e.g., {"os": ["linux", "windows"]})
	Strategy MatrixStrategy           `yaml:"strategy"` // Matrix execution strategy
}

// MatrixStrategy defines how matrix jobs should be executed
type MatrixStrategy struct {
	MaxParallel     int    `yaml:"max_parallel"`     // Maximum concurrent jobs (default: all)
	FailFast        bool   `yaml:"fail_fast"`        // Stop all jobs on first failure (default: false)
	ContinueOnError bool   `yaml:"continue_on_error"` // Stage succeeds even if some jobs fail (default: false)
	Order          string `yaml:"order"`             // Job scheduling order: "fifo" or "random" (default: "fifo")
}

// MatrixJob represents a single matrix job
type MatrixJob struct {
	ID       string                 // Unique job identifier
	Matrix   map[string]interface{} // Matrix values for this job
	Action   *Action                // Action to execute
	Status   Status                 // Job execution status
	Result   *Result                // Job execution result
	StartTime time.Time             // Job start time
	EndTime   time.Time             // Job end time
}

// MatrixJobStatus represents the status of a matrix job
type MatrixJobStatus int

const (
	MatrixJobPending MatrixJobStatus = iota
	MatrixJobRunning
	MatrixJobCompleted
	MatrixJobFailed
	MatrixJobSkipped
)

// String returns the string representation of the matrix job status
func (s MatrixJobStatus) String() string {
	switch s {
	case MatrixJobPending:
		return "PENDING"
	case MatrixJobRunning:
		return "RUNNING"
	case MatrixJobCompleted:
		return "COMPLETED"
	case MatrixJobFailed:
		return "FAILED"
	case MatrixJobSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

// Result represents the result of executing a step
type Result struct {
	Name           string
	Status         Status
	Message        string
	Error          error
	Duration       time.Duration
	BufferedOutput string // Store buffered stdout/stderr for quiet mode
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
	VerboseLevel int              // Verbosity level: 0=quiet, 1=-v, 2=-vv, 3=-vvv
	Debug       bool              // Enable debug output
	DryRun      bool              // Show what would be executed without running commands
	Variables   map[string]string // Additional variables for interpolation
	WorkingDir  string            // Working directory for execution
	Output      io.Writer         // Output writer (default: os.Stdout)
	ErrorOutput io.Writer         // Error output writer (default: os.Stderr)
	Only        []string          // Only run steps matching these labels
	WithRequires bool             // Include required dependencies when running single step
	StepCallback StepCallback     // Optional callback for step execution events
	InterpolatedActions map[string]*Action // Interpolated actions from matrix expansion (internal use)
}

// DefaultRunOptions returns default run options
func DefaultRunOptions() *RunOptions {
	variables := make(map[string]string)
	// Add platform variables by default
	variables = AddPlatformVariables(variables)
	// Add version variables by default
	variables = AddVersionVariables(variables)
	
	return &RunOptions{
		ConfigPath:  ".project.yml",
		MaxParallel: runtime.NumCPU(),
		VerboseLevel: 1,
		Debug:       false,
		Variables:   variables,
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
	interpolatedActions map[string]*Action // Store interpolated actions from matrix expansion
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
		interpolatedActions: make(map[string]*Action),
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
		// Note: Step callbacks are handled at the DAG execution level, not here
		// to avoid duplicate callbacks when running actions inside containers

		// Handle dry-run mode for built-in actions
		if r.opts.DryRun {
			description := runner.Description()
			if r.opts.StepCallback != nil {
				r.opts.StepCallback.OnStepStart(ctx, actionName)
				r.opts.StepCallback.OnStepComplete(ctx, actionName, StepStatusOK, fmt.Sprintf("would execute built-in action: %s", description), 0, "")
			}
			return nil
		}

		// Call step start callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepStart(ctx, actionName)
		}

		start := time.Now()
		result, err := runner.Run(ctx)
		duration := time.Since(start)

		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			status := StepStatusOK
			message := "executed successfully"
			
			// Prioritize result status and message over error when available
			if result.Status == StatusError {
				status = StepStatusError
				message = result.Message
				if err != nil {
					r.opts.StepCallback.OnStepError(ctx, actionName, err)
				}
			} else if result.Status == StatusWarn {
				status = StepStatusWarn
				message = result.Message
			} else if result.Status == StatusSkipped {
				status = StepStatusSkipped
				message = result.Message
			} else if err != nil {
				status = StepStatusError
				message = err.Error()
				r.opts.StepCallback.OnStepError(ctx, actionName, err)
			}
			
			r.opts.StepCallback.OnStepComplete(ctx, actionName, status, message, duration, "")
		}

		return err
	}

	// Check if it's a custom action
	action, exists := r.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	// Note: Step callbacks are handled at the DAG execution level, not here
	// to avoid duplicate callbacks when running actions inside containers

		// Handle dry-run mode for custom actions
		if r.opts.DryRun {
			err := r.runActionInternalDryRun(ctx, action)
			if r.opts.StepCallback != nil {
				r.opts.StepCallback.OnStepStart(ctx, actionName)
				status := StepStatusOK
				message := "would execute action"
				if err != nil {
					status = StepStatusError
					message = err.Error()
				}
				r.opts.StepCallback.OnStepComplete(ctx, actionName, status, message, 0, "")
			}
			return err
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
		
		r.opts.StepCallback.OnStepComplete(ctx, actionName, status, message, duration, "")
	}

	return err
}

// RunStageWithSteps runs a stage with specific steps (for matrix expansion)
func (r *Runner) RunStageWithSteps(ctx context.Context, stageName string, steps []Step) error {
	// Execute the steps directly
	return r.executeStageWithCallback(ctx, steps)
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
		
		r.opts.StepCallback.OnStepComplete(ctx, stepName, status, message, duration, "")
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

// SelectVariant selects the first matching variant for an action based on when conditions
func (a *Action) SelectVariant(variables map[string]string) (*ActionVariant, error) {
	if len(a.Variants) == 0 {
		// No variants, return nil to indicate action should use direct run/uses
		return nil, nil
	}
	
	// Evaluate each variant's when condition
	for _, variant := range a.Variants {
		matches, err := evaluateCondition(variant.When, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate condition '%s' for action %s: %w", variant.When, a.Name, err)
		}
		
		if matches {
			return &variant, nil
		}
	}
	
	// No variant matched - action should be skipped
	return nil, nil
}

// evaluateCondition evaluates a when condition expression using the enhanced expression system
func evaluateCondition(condition string, variables map[string]string) (bool, error) {
	// Create expression context with variables
	ctx := NewExpressionContext(variables)
	
	// Evaluate the expression using the new expression system
	return EvaluateExpression(condition, ctx)
}

// shouldExecuteStepByCondition determines if a step should be executed based on its if condition
func (r *Runner) shouldExecuteStepByCondition(ctx context.Context, step Step) (bool, error) {
	// If no if condition is specified, execute the step
	if step.If == "" {
		return true, nil
	}
	
	// Evaluate the if condition using the expression evaluator
	shouldExecute, err := evaluateCondition(step.If, r.opts.Variables)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate if condition for step %s: %w", step.Action, err)
	}
	
	return shouldExecute, nil
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
		
		// Check if action has variants
		if len(action.Variants) > 0 {
			// Action with variants: validate variants instead of direct run/uses
			for i, variant := range action.Variants {
				if variant.When == "" {
					return fmt.Errorf("action %s variant %d must have 'when' condition", action.Name, i)
				}
				
				if variant.Run == "" && variant.Uses == "" {
					return fmt.Errorf("action %s variant %d must have either 'run' or 'uses'", action.Name, i)
				}
				
				if variant.Run != "" && variant.Uses != "" {
					return fmt.Errorf("action %s variant %d cannot have both 'run' and 'uses'", action.Name, i)
				}
			}
		} else {
			// Action without variants: validate direct run/uses or container
			if action.Run == "" && action.Uses == "" && action.Container == nil {
				return fmt.Errorf("action %s must have either 'run', 'uses', or 'container'", action.Name)
			}
			
			if action.Run != "" && action.Uses != "" {
				return fmt.Errorf("action %s cannot have both 'run' and 'uses'", action.Name)
			}
			
			if action.Container != nil && (action.Run != "" || action.Uses != "") {
				return fmt.Errorf("action %s cannot have both 'container' and 'run'/'uses'", action.Name)
			}
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

// expandMatrixSteps expands matrix steps into individual steps for DAG execution
func (r *Runner) expandMatrixSteps(steps []Step) ([]Step, error) {
	var expandedSteps []Step
	
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandMatrixSteps: processing %d steps\n", len(steps))
	}
	
	for i, step := range steps {
		if r.opts.Debug {
			fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Step %d: Action=%s, Matrix=%v\n", i, step.Action, step.Matrix != nil)
		}
		
		// Check if this step has matrix configuration
		if step.Matrix != nil {
			if r.opts.Debug {
				fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanding matrix step: %s\n", step.Action)
			}
			
			// Get the action for this step
			action, exists := r.config.GetAction(step.Action)
			if !exists {
				return nil, fmt.Errorf("action not found: %s", step.Action)
			}
			
			// Extract matrix variables from CLI variables
			matrixVars := make(map[string]string)
			for key, value := range r.opts.Variables {
				if strings.HasPrefix(key, "matrix.") {
					matrixKey := strings.TrimPrefix(key, "matrix.")
					matrixVars[matrixKey] = value
				}
			}
			
			
			// Create matrix expander with CLI matrix variables
			expander := NewMatrixExpander(r.config, matrixVars)
			
			// Expand matrix into individual steps with interpolated actions
			matrixSteps, interpolatedActions, err := expander.ExpandMatrixToStepsWithActions(&step, &action)
			if err != nil {
				return nil, fmt.Errorf("failed to expand matrix for step %s: %w", step.Action, err)
			}
			
			// Store interpolated actions for later use by OrderedOutputManager
			for stepName, interpolatedAction := range interpolatedActions {
				r.interpolatedActions[stepName] = interpolatedAction
			}
			
			if r.opts.Debug {
				fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanded into %d steps\n", len(matrixSteps))
				for j, matrixStep := range matrixSteps {
					fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix step %d: Action=%s, Description=%s\n", j, matrixStep.Action, matrixStep.Description)
				}
			}
			
			// Add expanded steps
			expandedSteps = append(expandedSteps, matrixSteps...)
		} else {
			// Regular step - add as is
			expandedSteps = append(expandedSteps, step)
		}
	}
	
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandMatrixSteps: returning %d expanded steps\n", len(expandedSteps))
	}
	
	return expandedSteps, nil
}

// expandMatrixStepsWithActions expands matrix steps into individual steps and returns interpolated actions
func (r *Runner) expandMatrixStepsWithActions(steps []Step) ([]Step, map[string]*Action, error) {
	var expandedSteps []Step
	allInterpolatedActions := make(map[string]*Action)
	
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandMatrixStepsWithActions: processing %d steps\n", len(steps))
	}
	
	for i, step := range steps {
		if r.opts.Debug {
			fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Step %d: Action=%s, Matrix=%v\n", i, step.Action, step.Matrix != nil)
		}
		
		// Check if this step has matrix configuration
		if step.Matrix != nil {
			if r.opts.Debug {
				fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanding matrix step: %s\n", step.Action)
			}
			
			// Get the action for this step
			action, exists := r.config.GetAction(step.Action)
			if !exists {
				return nil, nil, fmt.Errorf("action not found: %s", step.Action)
			}
			
			// Extract matrix variables from CLI variables
			matrixVars := make(map[string]string)
			for key, value := range r.opts.Variables {
				if strings.HasPrefix(key, "matrix.") {
					matrixVars[key] = value
				}
			}
			
			// Create matrix expander with CLI matrix variables
			expander := NewMatrixExpander(r.config, matrixVars)
			
			// Expand matrix into individual steps with interpolated actions
			matrixSteps, interpolatedActions, err := expander.ExpandMatrixToStepsWithActions(&step, &action)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to expand matrix for step %s: %w", step.Action, err)
			}
			
			// Store interpolated actions for later use by OrderedOutputManager
			for stepName, interpolatedAction := range interpolatedActions {
				allInterpolatedActions[stepName] = interpolatedAction
			}
			
			if r.opts.Debug {
				fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanded into %d steps\n", len(matrixSteps))
				for j, matrixStep := range matrixSteps {
					fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix step %d: Action=%s, Description=%s\n", j, matrixStep.Action, matrixStep.Description)
				}
			}
			
			// Add expanded steps
			expandedSteps = append(expandedSteps, matrixSteps...)
		} else {
			// Regular step - add as-is
			expandedSteps = append(expandedSteps, step)
		}
	}
	
	return expandedSteps, allInterpolatedActions, nil
}

// runStageInternal executes a stage using parallel execution with ordered streaming output
func (r *Runner) runStageInternal(ctx context.Context, stageName string) error {
	stage, _ := r.config.GetStage(stageName)
	
	// Debug output
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] runStageInternal: stageName=%s, DryRun=%v\n", stageName, r.opts.DryRun)
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Steps: %+v\n", stage.Steps)
	}
	
	// Handle dry-run mode for stages
	if r.opts.DryRun {
		return r.executeStageDryRun(ctx, stageName, stage.Steps)
	}
	
	// Expand matrix steps into individual steps with interpolated actions
	expandedSteps, interpolatedActions, err := r.expandMatrixStepsWithActions(stage.Steps)
	if err != nil {
		return fmt.Errorf("failed to expand matrix steps: %w", err)
	}
	
	// Store interpolated actions in RunOptions for use by StepCallback
	r.opts.InterpolatedActions = interpolatedActions
	
	// If we have a step callback, use it for execution
	if r.opts.StepCallback != nil {
		// Update interpolated actions in the step callback if it supports it
		if orderedCallback, ok := r.opts.StepCallback.(*OrderedStepCallback); ok {
			orderedCallback.UpdateInterpolatedActions(interpolatedActions)
			orderedCallback.manager.SetConfigPath(r.opts.ConfigPath)
		}
		return r.executeStageWithCallback(ctx, expandedSteps)
	}
	
	// Build execution DAG
	dag, err := r.buildDAG(expandedSteps)
	if err != nil {
		return fmt.Errorf("failed to build execution DAG: %w", err)
	}
	
	// Execute DAG with parallel execution but ordered streaming output
	results, err := r.executeDAGWithOrderedStreaming(ctx, dag, expandedSteps, interpolatedActions)
	
	// Check if execution was terminated due to context cancellation
	terminated := ctx.Err() != nil
	
	// Check for errors in results
	for _, result := range results {
		if result.Status == StatusError {
			// Find the step to check error policy
			for _, step := range stage.Steps {
				if step.Action == result.Name {
					if step.OnError == "warn" {
						// Log warning but continue
						if r.opts.VerboseLevel > 0 {
							fmt.Fprintf(r.opts.ErrorOutput, "Warning: step %s failed: %v\n", step.Action, result.Error)
						}
						continue
					}
					// Default is "stop" - return error
					if result.Error != nil {
						return fmt.Errorf("step %s failed: %w", step.Action, result.Error)
					} else {
						return fmt.Errorf("step %s failed: %s", step.Action, result.Message)
					}
				}
			}
		}
	}
	
	// If terminated, return the context error
	if terminated {
		return ctx.Err()
	}
	
	return err
}

// executeStageDryRun simulates stage execution for dry-run mode
func (r *Runner) executeStageDryRun(ctx context.Context, stageName string, steps []Step) error {
	// Print stage header
	fmt.Fprintf(r.opts.Output, "▶️  Dry run stage: %s\n\n", stageName)
	
	// Count total steps
	totalSteps := len(steps)
	skippedSteps := 0
	executedSteps := 0
	
	// Process each step
	for _, step := range steps {
		// Debug output for step configuration
		if r.opts.Debug {
			fmt.Fprintf(r.opts.Output, "[DEBUG] Dry run step: %+v\n", step)
			fmt.Fprintf(r.opts.Output, "[DEBUG] Matrix config: %+v\n", step.Matrix)
		}
		// Check if step should be executed based on conditions
		shouldExecute, err := r.shouldExecuteStepByCondition(ctx, step)
		if err != nil {
			return fmt.Errorf("failed to evaluate step condition: %w", err)
		}
		
		if !shouldExecute {
			skippedSteps++
			if r.opts.VerboseLevel > 0 {
				fmt.Fprintf(r.opts.Output, "→ %s: would skip (condition not met)\n", step.Action)
			}
			continue
		}
		
		// Check if step should be executed based on only filter
		if len(r.opts.Only) > 0 {
			stepMatches := false
			for _, label := range r.opts.Only {
				for _, stepLabel := range step.Only {
					if stepLabel == label {
						stepMatches = true
						break
					}
				}
			}
			if !stepMatches {
				skippedSteps++
				if r.opts.VerboseLevel > 0 {
					fmt.Fprintf(r.opts.Output, "→ %s: would skip (not in only filter)\n", step.Action)
				}
				continue
			}
		}
		
		executedSteps++
		
		// Simulate action execution
		action, exists := r.config.GetAction(step.Action)
		if !exists {
			// Check if it's a built-in action
			if runner, exists := r.registry.GetRunner(step.Action); exists {
				description := runner.Description()
				if r.opts.VerboseLevel > 0 {
					fmt.Fprintf(r.opts.Output, "✓ %s: would execute built-in action: %s\n", step.Action, description)
				}
			} else {
				if r.opts.VerboseLevel > 0 {
					fmt.Fprintf(r.opts.Output, "✗ %s: would fail (action not found)\n", step.Action)
				}
			}
		} else {
			// Simulate custom action execution
			err := r.runActionInternalDryRun(ctx, action)
			if err != nil {
				if r.opts.VerboseLevel > 0 {
					fmt.Fprintf(r.opts.Output, "✗ %s: would fail (%v)\n", step.Action, err)
				}
			}
		}
	}
	
	// Print summary
	fmt.Fprintf(r.opts.Output, "\n")
	fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(r.opts.Output, "🔍 %s%s%s - %s\n", colorCyan, "DRY RUN", colorReset, stageName)
	fmt.Fprintf(r.opts.Output, "\n")
	fmt.Fprintf(r.opts.Output, "📊 Summary:\n")
	fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGreen, "✓", colorReset, colorGreen, "would run", executedSteps, colorReset)
	fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGray, "→", colorReset, colorGray, "skipped", skippedSteps, colorReset)
	fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGray, "?", colorReset, colorGray, "total", totalSteps, colorReset)
	
	return nil
}

// executeStageWithCallback executes a stage using the step callback for output management
func (r *Runner) executeStageWithCallback(ctx context.Context, steps []Step) error {
	// Steps are already expanded (matrix steps have been expanded by the caller)
	expandedSteps := steps
	
	// Build execution DAG
	dag, err := r.buildDAG(expandedSteps)
	if err != nil {
		return fmt.Errorf("failed to build execution DAG: %w", err)
	}
	
	
	// Execute DAG with step callback
	results, err := r.executeDAGWithCallback(ctx, dag, expandedSteps)
	
	// Check if execution was terminated due to context cancellation
	terminated := ctx.Err() != nil
	
	// Check for errors in results
	for _, result := range results {
		if result.Status == StatusError {
			// Find the step to check error policy
			for _, step := range steps {
				if step.Action == result.Name {
					if step.OnError == "warn" {
						// Log warning but continue
						if r.opts.VerboseLevel > 0 {
							fmt.Fprintf(r.opts.ErrorOutput, "Warning: step %s failed: %v\n", step.Action, result.Error)
						}
						continue
					}
					// Default is "stop" - return error
					if result.Error != nil {
						return fmt.Errorf("step %s failed: %w", step.Action, result.Error)
					} else {
						return fmt.Errorf("step %s failed: %s", step.Action, result.Message)
					}
				}
			}
		}
	}
	
	// If terminated, return the context error
	if terminated {
		return ctx.Err()
	}
	
	return err
}

// executeStepWithMatrix handles step execution with matrix support
func (r *Runner) executeStepWithMatrix(ctx context.Context, step Step) error {
	// Debug output
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] executeStepWithMatrix called for step: %s\n", step.Action)
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix config: %+v\n", step.Matrix)
	}
	
	// Check if step has matrix configuration
	if step.Matrix == nil {
		// Regular step execution - delegate to existing logic
		return r.executeStepRegular(ctx, step)
	}
	
	// Get the action for this step
	action, exists := r.config.GetAction(step.Action)
	if !exists {
		return fmt.Errorf("action not found: %s", step.Action)
	}
	
	// Create matrix expander
	expander := NewMatrixExpander(r.config)
	
	// Expand matrix into jobs
	jobs, err := expander.ExpandMatrix(&step, &action)
	if err != nil {
		return fmt.Errorf("failed to expand matrix: %w", err)
	}
	
	if len(jobs) == 0 {
		// No matrix jobs to execute
		return nil
	}
	
	// Set default strategy values if not specified
	strategy := step.Matrix.Strategy
	if strategy.MaxParallel <= 0 {
		strategy.MaxParallel = r.opts.MaxParallel
	}
	if strategy.Order == "" {
		strategy.Order = "fifo"
	}
	
	// Create matrix scheduler
	scheduler := NewMatrixScheduler(jobs, strategy, strategy.MaxParallel, r.opts)
	
	// Set step callback if available
	if r.opts.StepCallback != nil {
		scheduler.SetStepCallback(r.opts.StepCallback, step.Action)
	}
	
	// Execute matrix jobs
	return scheduler.ScheduleJobs(r, r.opts)
}

// executeStepRegular handles regular step execution without matrix
func (r *Runner) executeStepRegular(ctx context.Context, step Step) error {
	// Check if it's a built-in action
	if runner, exists := r.registry.GetRunner(step.Action); exists {
		// Call step start callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepStart(ctx, step.Action)
		}

		// Handle dry-run mode for built-in actions
		if r.opts.DryRun {
			description := runner.Description()
			if r.opts.StepCallback != nil {
				r.opts.StepCallback.OnStepComplete(ctx, step.Action, StepStatusOK, fmt.Sprintf("would execute built-in action: %s", description), 0, "")
			}
			return nil
		}

		start := time.Now()
		result, err := runner.Run(ctx)
		duration := time.Since(start)

		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			status := StepStatusOK
			message := "executed successfully"
			
			// Prioritize result status and message over error when available
			if result.Status == StatusError {
				status = StepStatusError
				message = result.Message
				if err != nil {
					r.opts.StepCallback.OnStepError(ctx, step.Action, err)
				}
			} else if result.Status == StatusWarn {
				status = StepStatusWarn
				message = result.Message
			} else if result.Status == StatusSkipped {
				status = StepStatusSkipped
				message = result.Message
			} else if err != nil {
				status = StepStatusError
				message = err.Error()
				r.opts.StepCallback.OnStepError(ctx, step.Action, err)
			}
			
			r.opts.StepCallback.OnStepComplete(ctx, step.Action, status, message, duration, "")
		}

		return err
	}

	// Check if it's a custom action
	action, exists := r.config.GetAction(step.Action)
	if !exists {
		return fmt.Errorf("action not found: %s", step.Action)
	}

	// Call step start callback if provided
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, step.Action)
	}

	// Handle dry-run mode for custom actions
	if r.opts.DryRun {
		err := r.runActionInternalDryRun(ctx, action)
		if r.opts.StepCallback != nil {
			status := StepStatusOK
			message := "would execute action"
			if err != nil {
				status = StepStatusError
				message = err.Error()
			}
			r.opts.StepCallback.OnStepComplete(ctx, step.Action, status, message, 0, "")
		}
		return err
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
			r.opts.StepCallback.OnStepError(ctx, step.Action, err)
		}
		
		r.opts.StepCallback.OnStepComplete(ctx, step.Action, status, message, duration, "")
	}

	return err
}

// executeDAGWithCallback executes the DAG using step callbacks for output management
func (r *Runner) executeDAGWithCallback(ctx context.Context, dag map[string]*DAGNode, steps []Step) ([]Result, error) {
	var results []Result
	completed := make(map[string]bool)
	failed := make(map[string]bool)
	executing := make(map[string]bool)
	
	// Create a map of results by step name for quick lookup
	resultMap := make(map[string]Result)
	
	// Create channels for communication
	resultChan := make(chan Result, len(dag))
	done := make(chan bool)
	ctxDone := ctx.Done()
	
	// Mutex for thread-safe access to shared state
	var mu sync.Mutex
	
	// Start a goroutine to handle results
	go func() {
		defer close(done)
		for result := range resultChan {
			mu.Lock()
			results = append(results, result)
			resultMap[result.Name] = result
			completed[result.Name] = true
			executing[result.Name] = false
			
			if result.Status == StatusError {
				failed[result.Name] = true
			}
			mu.Unlock()
		}
	}()
	
	// Start execution goroutine that continuously starts new steps
	go func() {
		defer func() {
			// Wait for all executing goroutines to complete before closing the channel
			for {
				mu.Lock()
				anyExecuting := false
				for _, isExecuting := range executing {
					if isExecuting {
						anyExecuting = true
						break
					}
				}
				mu.Unlock()
				
				if !anyExecuting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			close(resultChan)
		}()
		
		for {
			// Check for context cancellation
			select {
			case <-ctxDone:
				return
			default:
			}
			
			// Find ready steps
			var readySteps []string
			mu.Lock()
			for nodeName, node := range dag {
				if !completed[nodeName] && !executing[nodeName] && !failed[nodeName] {
					// Check if all dependencies are completed
					allDepsCompleted := true
					for _, dep := range node.Dependencies {
						if !completed[dep] {
							allDepsCompleted = false
							break
						}
					}
					
					if allDepsCompleted {
						readySteps = append(readySteps, nodeName)
					}
				}
			}
			mu.Unlock()
			
			// If no ready steps, we're done
			if len(readySteps) == 0 {
				// Check if all steps are completed or failed
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if !completed[nodeName] && !failed[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					return
				}
				
				// Wait a bit before checking again
				time.Sleep(10 * time.Millisecond)
				continue
			}
			
			// Start ready steps immediately without waiting
			for _, nodeName := range readySteps {
				// Check for context cancellation before starting each step
				select {
				case <-ctx.Done():
					return
				default:
					// Continue with step execution
				}
				
				node := dag[nodeName]
				
				mu.Lock()
				executing[nodeName] = true
				mu.Unlock()
				
				// Skip if already failed and this node requires it
				if r.hasFailedDependency(node, failed) {
					failedDeps := r.getFailedDependencyNames(node, failed)
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: fmt.Sprintf("skipped (dependency failed: %s)", strings.Join(failedDeps, ", ")),
					}
					resultChan <- result
					continue
				}
				
				// Check if step should be executed based on conditions
				if !r.shouldExecuteStep(ctx, node) {
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: "skipped (condition not met)",
					}
					
					// Call step callback for skipped step
					if r.opts.StepCallback != nil {
						r.opts.StepCallback.OnStepComplete(ctx, nodeName, StepStatusSkipped, "skipped (condition not met)", 0, "")
					}
					
					resultChan <- result
					continue
				}
				
				// Execute the node in parallel
				go func(nodeName string, node *DAGNode) {
					// Find the step configuration
					var stepConfig *Step
					for _, step := range steps {
						if step.Action == nodeName {
							stepConfig = &step
							break
						}
					}
					
					result, _ := r.executeActionForDAGWithCallback(ctx, node.Action, stepConfig)
					result.Name = nodeName
					// Check if context was cancelled during execution
					if ctx.Err() != nil {
						result.Status = StatusError
						result.Message = "cancelled"
						result.Error = ctx.Err()
					}
					
					resultChan <- result
				}(nodeName, node)
			}
		}
	}()
	
	<-done
	
	return results, nil
}

// executeActionForDAGWithCallback executes a single action for DAG execution using step callbacks
func (r *Runner) executeActionForDAGWithCallback(ctx context.Context, action Action, stepConfig *Step) (Result, error) {
	// Matrix steps are now expanded into individual steps, so this is just a regular step
	
	// Call step start callback if provided
	stepName := action.Name
	if stepConfig != nil && stepConfig.Action != "" {
		stepName = stepConfig.Action // Use step action name for matrix steps (e.g., "container-platform-view.alpine:latest")
	}
	
	// Debug: Print step information
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] executeActionForDAGWithCallback: action.Name=%s, stepConfig.Action=%s, stepName=%s\n", action.Name, stepConfig.Action, stepName)
	}
	
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, stepName)
	}

	var result Result
	var err error

	// Measure execution time from when the action actually starts to when it finishes
	start := time.Now()
	
	// Handle variants - select appropriate variant or skip if no match
	variant, variantErr := action.SelectVariant(r.opts.Variables)
	if variantErr != nil {
		result = Result{
			Status:  StatusError,
			Message: variantErr.Error(),
			Error:   variantErr,
		}
		duration := time.Since(start)
		result.Duration = duration
		
		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepComplete(ctx, action.Name, StepStatusError, variantErr.Error(), duration, "")
		}
		
		return result, variantErr
	}
	
	// If variant is nil and action has variants, it means no variant matched - skip
	if variant == nil && len(action.Variants) > 0 {
		result = Result{
			Status:  StatusSkipped,
			Message: "no matching variant",
		}
		duration := time.Since(start)
		result.Duration = duration
		
		// Call step complete callback if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepComplete(ctx, action.Name, StepStatusSkipped, "no matching variant", duration, "")
		}
		
		return result, nil // Not an error, just skipped
	}
	
	// Use variant if available, otherwise use action directly
	effectiveAction := action
	if variant != nil {
		effectiveAction = Action{
			Name:      action.Name,
			Run:       variant.Run,
			Uses:      variant.Uses,
			Shell:     variant.Shell,
			Container: action.Container, // Preserve container configuration
		}
	}
	
	if effectiveAction.Uses != "" {
		result, err = r.runBuiltInActionForDAG(ctx, effectiveAction)
	} else {
		result, err = r.runCustomActionForDAG(ctx, effectiveAction)
	}
	duration := time.Since(start)
	
	// Set the duration in the result
	result.Duration = duration

	// Apply onerror policy if step has one
	if result.Status == StatusError && stepConfig != nil && stepConfig.OnError == "warn" {
		// Convert error to warning
		result.Status = StatusWarn
		err = nil // Clear the error since it's now a warning
	}

	// Call step complete callback if provided
	if r.opts.StepCallback != nil {
		status := StepStatusOK
		message := "executed successfully"
		
		// Prioritize result status and message over error when available
		if result.Status == StatusError {
			status = StepStatusError
			message = result.Message
			if err != nil {
				r.opts.StepCallback.OnStepError(ctx, action.Name, err)
			}
		} else if result.Status == StatusWarn {
			status = StepStatusWarn
			message = result.Message
		} else if result.Status == StatusSkipped {
			status = StepStatusSkipped
			message = result.Message
		} else if err != nil {
			status = StepStatusError
			message = err.Error()
			r.opts.StepCallback.OnStepError(ctx, action.Name, err)
		}
		
		r.opts.StepCallback.OnStepComplete(ctx, action.Name, status, message, duration, result.BufferedOutput)
	}

	return result, err
}

// buildContainerCommandForDisplay builds a human-readable representation of the container command
func (r *Runner) buildContainerCommandForDisplay(config *container.ContainerConfig) string {
	var parts []string

	// Use specified engine or default to podman
	engineName := "podman" // Default to podman
	if config.Engine != "" {
		engineName = config.Engine
	}

	// Add engine (Docker/Podman)
	parts = append(parts, engineName)

	// Handle slim operations differently
	if config.Image.Slim != nil {
		// For slim operations, we need to mount the Docker socket
		if engineName == "docker" {
			parts = append(parts, "-v", "/var/run/docker.sock:/var/run/docker.sock")
		} else if engineName == "podman" {
			parts = append(parts, "-v", "/run/podman/podman.sock:/run/podman/podman.sock")
		}
		
		// For slim operations, we run the dslim/slim container with specific arguments
		parts = append(parts, "dslim/slim:latest")
		parts = append(parts, "slim", "build")
		
		// Add slim-specific flags
		if !config.Image.Slim.HttpProbe {
			parts = append(parts, "--http-probe=false")
			parts = append(parts, "--continue-after=exit")
		}
		
		parts = append(parts, config.Image.Slim.Target)
		
		// Add exec command if specified
		if config.Image.Slim.Exec != "" {
			parts = append(parts, "--exec", config.Image.Slim.Exec)
		}
		
		// Add tags for the slim image
		for _, tag := range config.Image.Slim.Tags {
			parts = append(parts, "--tag", tag)
		}
	} else if config.Image.Build != nil {
		// Build operation
		parts = append(parts, "build")
		
		// Add build arguments
		for key, value := range config.Image.Build.Args {
			parts = append(parts, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}
		
		// Add tags
		for _, tag := range config.Image.Build.Tags {
			parts = append(parts, "--tag", tag)
		}
		
		// Add network
		if config.Image.Build.Network != "" {
			parts = append(parts, "--network", config.Image.Build.Network)
		}
		
		// Add progress
		if config.Image.Build.Progress != "" {
			parts = append(parts, "--progress", config.Image.Build.Progress)
		}
		
		// Add dockerfile
		if config.Image.Build.Dockerfile != "" {
			parts = append(parts, "-f", config.Image.Build.Dockerfile)
		}
		
		// Add context
		context := config.Image.Build.Context
		if context == "" {
			context = "."
		}
		parts = append(parts, context)
	} else {
		// Regular container execution
		parts = append(parts, "run", "--rm")
		
		// Check if there's a bind mount with target /tmp/buildfab-workspace and add -w accordingly
		hasWorkspaceMount := false
		for _, mount := range config.Mounts {
			if mount.Type == "bind" && mount.Target == "/tmp/buildfab-workspace" {
				hasWorkspaceMount = true
				break
			}
		}
		
		// Add working directory option only if workspace mount exists
		if hasWorkspaceMount {
			parts = append(parts, "-w", "/tmp/buildfab-workspace")
		}
		
		// Add mount arguments
		for _, mount := range config.Mounts {
			mountArg := fmt.Sprintf("--mount=type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target)
			if mount.RO {
				mountArg += ",readonly"
			}
			parts = append(parts, mountArg)
		}
		
		// Add network if specified
		if config.Network != "" {
			parts = append(parts, "--network", config.Network)
		}

		// Add image
		parts = append(parts, config.Image.From)

		// Add command to run
		if config.Run != "" {
			runCommand := config.Run
			if config.Workdir != "" {
				runCommand = fmt.Sprintf("cd %s && %s", config.Workdir, config.Run)
			}
			parts = append(parts, "sh", "-c", runCommand)
		} else if config.RunAction != "" {
			runCommand := fmt.Sprintf("buildfab action %s", config.RunAction)
			if config.Workdir != "" {
				runCommand = fmt.Sprintf("cd %s && %s", config.Workdir, runCommand)
			}
			parts = append(parts, "sh", "-c", runCommand)
		} else if config.RunStage != "" {
			runCommand := fmt.Sprintf("buildfab run %s", config.RunStage)
			if config.Workdir != "" {
				runCommand = fmt.Sprintf("cd %s && %s", config.Workdir, runCommand)
			}
			parts = append(parts, "sh", "-c", runCommand)
		}
	}

	return strings.Join(parts, " ")
}

// shouldExecuteStep checks if a step should be executed based on conditions
func (r *Runner) shouldExecuteStep(ctx context.Context, node *DAGNode) bool {
	// Check if step should be executed based on its if condition
	shouldExecute, err := r.shouldExecuteStepByCondition(ctx, node.Step)
	if err != nil {
		// If there's an error evaluating the condition, log it and skip the step
		if r.opts.VerboseLevel > 0 {
			fmt.Fprintf(r.opts.ErrorOutput, "Warning: failed to evaluate if condition for step %s: %v\n", node.Step.Action, err)
		}
		return false
	}
	return shouldExecute
}

// DAGNode represents a node in the execution DAG
type DAGNode struct {
	Step         Step
	Action       Action
	Dependencies []string
	Dependents   []string
}

// StreamingOutputManager manages which step's output should be streamed
type StreamingOutputManager struct {
	steps     []Step
	displayed map[string]bool
	started   map[string]bool
	mu        *sync.Mutex
	interpolatedActions map[string]*Action // Interpolated actions for matrix steps
}

// ShouldStreamOutput checks if the given step should have its output streamed
func (s *StreamingOutputManager) ShouldStreamOutput(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		// Step not found in steps
		return false
	}
	
	// Only allow streaming for the first step in declaration order that hasn't been displayed yet
	// Check if all previous steps in declaration order have been displayed
	for i := 0; i < stepIndex; i++ {
		if !s.displayed[s.steps[i].Action] {
			// Previous step not displayed yet
			return false
		}
	}
	
	// Check if this step itself has been displayed - if so, don't stream
	if s.displayed[stepName] {
		// Step already displayed
		return false
	}
	
	// Step can stream
	return true
}

// ShouldShowStepStart checks if the given step should show its start message
func (s *StreamingOutputManager) ShouldShowStepStart(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if all previous steps in declaration order have been started
	for i := 0; i < stepIndex; i++ {
		if !s.started[s.steps[i].Action] {
			return false
		}
	}
	
	return true
}

// MarkStepStarted marks a step as started
func (s *StreamingOutputManager) MarkStepStarted(stepName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started[stepName] = true
}

// buildDAG builds the execution DAG from stage steps
func (r *Runner) buildDAG(steps []Step) (map[string]*DAGNode, error) {
	dag := make(map[string]*DAGNode)
	
	// Create nodes for each step
	for _, step := range steps {
		action, exists := r.config.GetAction(step.Action)
		if !exists {
			return nil, fmt.Errorf("action not found: %s", step.Action)
		}
		
		// Use the step action as the node name (matrix steps now have unique action names)
		nodeName := step.Action
		
		node := &DAGNode{
			Step:         step,
			Action:       action,
			Dependencies: step.Require,
			Dependents:   []string{},
		}
		
		dag[nodeName] = node
	}
	
	// Build dependency relationships
	for _, node := range dag {
		for _, dep := range node.Dependencies {
			if depNode, exists := dag[dep]; exists {
				depNode.Dependents = append(depNode.Dependents, node.Step.Action)
			} else {
				return nil, fmt.Errorf("dependency not found: %s", dep)
			}
		}
	}
	
	// Check for cycles
	if err := r.detectCycles(dag); err != nil {
		return nil, fmt.Errorf("circular dependency detected: %w", err)
	}
	
	return dag, nil
}

// detectCycles detects cycles in the DAG using DFS
func (r *Runner) detectCycles(dag map[string]*DAGNode) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var dfs func(string) error
	dfs = func(nodeName string) error {
		if recStack[nodeName] {
			return fmt.Errorf("cycle detected involving node: %s", nodeName)
		}
		if visited[nodeName] {
			return nil
		}
		
		visited[nodeName] = true
		recStack[nodeName] = true
		defer func() { recStack[nodeName] = false }()
		
		node := dag[nodeName]
		for _, dep := range node.Dependencies {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		
		return nil
	}
	
	for nodeName := range dag {
		if !visited[nodeName] {
			if err := dfs(nodeName); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// executeDAGWithOrderedStreaming executes the DAG with parallel execution but ordered streaming output
func (r *Runner) executeDAGWithOrderedStreaming(ctx context.Context, dag map[string]*DAGNode, steps []Step, interpolatedActions map[string]*Action) ([]Result, error) {
	var results []Result
	completed := make(map[string]bool)
	failed := make(map[string]bool)
	executing := make(map[string]bool)
	displayed := make(map[string]bool)
	started := make(map[string]bool)
	
	// Create a map of results by step name for quick lookup
	resultMap := make(map[string]Result)
	
	// Create channels for communication
	resultChan := make(chan Result, len(dag))
	done := make(chan bool)
	ctxDone := ctx.Done()
	
	// Mutex for thread-safe access to shared state
	var mu sync.Mutex
	
	// Create a streaming output manager using the expanded steps
	streamingManager := &StreamingOutputManager{
		steps:     steps,
		displayed: displayed,
		started:   make(map[string]bool),
		mu:        &mu,
		interpolatedActions: interpolatedActions,
	}
	
	// Start a goroutine to handle results and display them in order
	go func() {
		defer close(done)
		for result := range resultChan {
			mu.Lock()
			results = append(results, result)
			resultMap[result.Name] = result
			completed[result.Name] = true
			executing[result.Name] = false
			
			if result.Status == StatusError {
				failed[result.Name] = true
			}
			mu.Unlock()
			
			// Display immediately if it's ready in declaration order
			r.displayStepInOrder(ctx, result.Name, steps, resultMap, displayed, completed)
			
			// Check if we can now display the next step
			r.checkAndDisplayNextStep(ctx, steps, resultMap, displayed, completed, started)
		}
	}()
	
	// Start execution goroutine that continuously starts new steps
	go func() {
		defer func() {
			// Wait for all executing goroutines to complete before closing the channel
			for {
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Check for context cancellation while waiting
				select {
				case <-ctxDone:
					close(resultChan)
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
			}
			close(resultChan)
		}()
		
		for {
			// Check for context cancellation at the start of each loop iteration
			select {
			case <-ctxDone:
				return
			default:
				// Continue with execution
			}
			
			mu.Lock()
			readySteps := r.getReadyStepsLocked(dag, completed, failed, executing)
			mu.Unlock()
			
			if len(readySteps) == 0 {
				// Check if all steps are completed or executing
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if !completed[nodeName] && !executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Wait a bit before checking again, but check for cancellation
				select {
				case <-ctxDone:
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
				continue
			}
			
			// Start ready steps immediately without waiting
			for _, nodeName := range readySteps {
				// Check for context cancellation before starting each step
				select {
				case <-ctxDone:
					return
				default:
					// Continue with step execution
				}
				
				node := dag[nodeName]
				
				mu.Lock()
				executing[nodeName] = true
				mu.Unlock()
				
				// Skip if already failed and this node requires it
				if r.hasFailedDependency(node, failed) {
					failedDeps := r.getFailedDependencyNames(node, failed)
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: fmt.Sprintf("skipped (dependency failed: %s)", strings.Join(failedDeps, ", ")),
					}
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
					continue
				}
				
				// Check if step should be executed based on conditions
				if !r.shouldExecuteStep(ctx, node) {
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: "skipped (condition not met)",
					}
					
					// Call step callback for skipped step
					if r.opts.StepCallback != nil {
						r.opts.StepCallback.OnStepComplete(ctx, nodeName, StepStatusSkipped, "skipped (condition not met)", 0, "")
					}
					
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
					continue
				}
				
				// Execute the node in parallel with streaming output control
				go func(nodeName string, node *DAGNode) {
					// Execute matrix step
					result, err := r.executeActionForDAGWithStreamingControl(ctx, node.Action, streamingManager)
					result.Name = nodeName
					
					// Call OnStepError immediately if the step failed
					if err != nil && r.opts.StepCallback != nil {
						r.opts.StepCallback.OnStepError(ctx, nodeName, err)
					}
					
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
				}(nodeName, node)
				
				// Check if we can display the first step immediately after starting execution
				r.checkAndDisplayNextStep(ctx, steps, resultMap, displayed, completed, started)
			}
		}
	}()
	
	<-done
	
	// Display any remaining steps that weren't displayed yet
	r.displayRemainingSteps(ctx, steps, resultMap, displayed)
	
	return results, nil
}

// checkAndDisplayNextStep checks if the next step in declaration order can be displayed
func (r *Runner) checkAndDisplayNextStep(ctx context.Context, steps []Step, resultMap map[string]Result, displayed map[string]bool, completed map[string]bool, started map[string]bool) {
	// Find the next step that can be displayed
	for _, step := range steps {
		if !displayed[step.Action] {
			// Check if we can display this step (either completed or currently executing)
			if r.canDisplayStepInOrder(step, steps, displayed) {
				// Show step start message if not already shown
				if r.opts.StepCallback != nil && !started[step.Action] {
					r.opts.StepCallback.OnStepStart(ctx, step.Action)
					started[step.Action] = true
				}
				
				// If completed, also show completion message
				if completed[step.Action] {
					r.displayStepInOrder(ctx, step.Action, steps, resultMap, displayed, completed)
				}
				break // Only display one step at a time
			}
		}
	}
}

// displayStepInOrder displays a step only if it can be shown in declaration order
func (r *Runner) displayStepInOrder(ctx context.Context, stepName string, steps []Step, resultMap map[string]Result, displayed map[string]bool, completed map[string]bool) {
	// Find the step in declaration order
	for _, step := range steps {
		if step.Action == stepName {
			// Check if all previous steps in declaration order have been displayed
			if r.canDisplayStepInOrder(step, steps, displayed) {
				
				// Only show completion message if the step is actually completed
				if result, exists := resultMap[stepName]; exists && completed[stepName] {
					// Display any buffered output for this step
					if r.opts.StepCallback != nil {
						// In verbose mode, display output during execution
						// In quiet mode, output is buffered and only shown on failure in showStepCompletion
						if r.opts.VerboseLevel > 0 && result.Message != "" {
							r.opts.StepCallback.OnStepOutput(ctx, stepName, result.Message)
						}
						// Note: In quiet mode, buffered output is displayed in showStepCompletion, not here
					}
					
					if r.opts.StepCallback != nil {
						status := StepStatusOK
						message := "executed successfully"
						
						if result.Status == StatusWarn {
							status = StepStatusWarn
							message = result.Message
						} else if result.Status == StatusError {
							status = StepStatusError
							message = result.Message
						} else if result.Status == StatusSkipped {
							status = StepStatusSkipped
							message = result.Message
						}
						
						r.opts.StepCallback.OnStepComplete(ctx, stepName, status, message, result.Duration, result.BufferedOutput)
					}
					displayed[stepName] = true
				}
			}
			break
		}
	}
}

// canDisplayStepInOrder checks if a step can be displayed in declaration order
func (r *Runner) canDisplayStepInOrder(step Step, steps []Step, displayed map[string]bool) bool {
	// Find the position of this step in the declaration order
	stepIndex := -1
	for i, s := range steps {
		if s.Action == step.Action {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if all previous steps in declaration order have been displayed
	for i := 0; i < stepIndex; i++ {
		if !displayed[steps[i].Action] {
			return false
		}
	}
	
	return true
}

// displayRemainingSteps displays any steps that weren't displayed yet
func (r *Runner) displayRemainingSteps(ctx context.Context, steps []Step, resultMap map[string]Result, displayed map[string]bool) {
	for _, step := range steps {
		if !displayed[step.Action] {
			if result, exists := resultMap[step.Action]; exists {
				if r.opts.StepCallback != nil {
					status := StepStatusOK
					message := "executed successfully"
					
					if result.Status == StatusWarn {
						status = StepStatusWarn
						message = result.Message
					} else if result.Status == StatusError {
						status = StepStatusError
						message = result.Message
					} else if result.Status == StatusSkipped {
						status = StepStatusSkipped
						message = result.Message
					}
					
					r.opts.StepCallback.OnStepComplete(ctx, step.Action, status, message, result.Duration, result.BufferedOutput)
				}
				displayed[step.Action] = true
			}
		}
	}
}

// executeDAGWithParallel executes the DAG with parallel execution
func (r *Runner) executeDAGWithParallel(ctx context.Context, dag map[string]*DAGNode, steps []Step) ([]Result, error) {
	var results []Result
	completed := make(map[string]bool)
	failed := make(map[string]bool)
	executing := make(map[string]bool)
	
	// Create a map of results by step name for quick lookup
	resultMap := make(map[string]Result)
	
	// Create channels for communication
	resultChan := make(chan Result, len(dag))
	done := make(chan bool)
	ctxDone := ctx.Done()
	
	// Mutex for thread-safe access to shared state
	var mu sync.Mutex
	
	// Start a goroutine to handle results
	go func() {
		defer close(done)
		for result := range resultChan {
			mu.Lock()
			results = append(results, result)
			resultMap[result.Name] = result
			completed[result.Name] = true
			executing[result.Name] = false
			
			if result.Status == StatusError {
				failed[result.Name] = true
			}
			mu.Unlock()
		}
	}()
	
	// Start execution goroutine that continuously starts new steps
	go func() {
		defer func() {
			// Wait for all executing goroutines to complete before closing the channel
			for {
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Check for context cancellation while waiting
				select {
				case <-ctxDone:
					close(resultChan)
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
			}
			close(resultChan)
		}()
		
		for {
			// Check for context cancellation at the start of each loop iteration
			select {
			case <-ctxDone:
				return
			default:
				// Continue with execution
			}
			
			mu.Lock()
			readySteps := r.getReadyStepsLocked(dag, completed, failed, executing)
			mu.Unlock()
			
			if len(readySteps) == 0 {
				// Check if all steps are completed or executing
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if !completed[nodeName] && !executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Wait a bit before checking again, but check for cancellation
				select {
				case <-ctxDone:
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
				continue
			}
			
			// Start ready steps immediately without waiting
			for _, nodeName := range readySteps {
				// Check for context cancellation before starting each step
				select {
				case <-ctxDone:
					return
				default:
					// Continue with step execution
				}
				
				node := dag[nodeName]
				
				mu.Lock()
				executing[nodeName] = true
				mu.Unlock()
				
				// Skip if already failed and this node requires it
				if r.hasFailedDependency(node, failed) {
					failedDeps := r.getFailedDependencyNames(node, failed)
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: fmt.Sprintf("skipped (dependency failed: %s)", strings.Join(failedDeps, ", ")),
					}
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
					continue
				}
				
				// Check if step should be executed based on conditions
				if !r.shouldExecuteStep(ctx, node) {
					result := Result{
						Name:   nodeName,
						Status: StatusSkipped,
						Message: "skipped (condition not met)",
					}
					
					// Call step callback for skipped step
					if r.opts.StepCallback != nil {
						r.opts.StepCallback.OnStepComplete(ctx, nodeName, StepStatusSkipped, "skipped (condition not met)", 0, "")
					}
					
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
					continue
				}
				
				// Execute the node in parallel
				go func(nodeName string, node *DAGNode) {
					result, _ := r.executeActionForDAGWithStep(ctx, node.Action, &node.Step)
					result.Name = nodeName
					
					select {
					case resultChan <- result:
					case <-ctxDone:
						return
					}
				}(nodeName, node)
			}
		}
	}()
	
	<-done
	
	return results, nil
}

// getReadyStepsLocked returns steps that are ready to execute (thread-safe version)
func (r *Runner) getReadyStepsLocked(dag map[string]*DAGNode, completed map[string]bool, failed map[string]bool, executing map[string]bool) []string {
	var ready []string
	
	for nodeName, node := range dag {
		if completed[nodeName] || executing[nodeName] {
			continue
		}
		
		// Check if all dependencies are completed
		if r.allDependenciesCompleted(node, completed) {
			ready = append(ready, nodeName)
		}
	}
	
	return ready
}

// allDependenciesCompleted checks if all dependencies are completed
func (r *Runner) allDependenciesCompleted(node *DAGNode, completed map[string]bool) bool {
	for _, dep := range node.Dependencies {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// hasFailedDependency checks if any required dependency has failed
func (r *Runner) hasFailedDependency(node *DAGNode, failed map[string]bool) bool {
	for _, dep := range node.Dependencies {
		if failed[dep] {
			return true
		}
	}
	return false
}

// getFailedDependencyNames returns the names of failed dependencies
func (r *Runner) getFailedDependencyNames(node *DAGNode, failed map[string]bool) []string {
	var failedDeps []string
	for _, dep := range node.Dependencies {
		if failed[dep] {
			failedDeps = append(failedDeps, dep)
		}
	}
	return failedDeps
}

// getShellCommand returns the appropriate shell command and arguments
// If userShell is specified, use that; otherwise use platform defaults
func getShellCommand(userShell string, verboseLevel int) (string, []string, error) {
	// If user specified a shell, use it
	if userShell != "" {
		// For Windows, automatically add .exe if not present
		if runtime.GOOS == "windows" && !strings.HasSuffix(userShell, ".exe") {
			userShell = userShell + ".exe"
		}
		
		// Check if the specified shell is available
		if _, err := exec.LookPath(userShell); err != nil {
			return "", nil, fmt.Errorf("shell '%s' not found in PATH. Please install it or use a different shell", userShell)
		}
		
		// Return appropriate arguments based on shell type
		switch {
		case strings.Contains(userShell, "bash"):
			args := []string{"-euc"}
			if verboseLevel >= 2 {
				args = []string{"-xeuc"}
			}
			return userShell, args, nil
		case strings.Contains(userShell, "powershell") || strings.Contains(userShell, "pwsh"):
			return userShell, []string{"-NoProfile", "-Command"}, nil
		case strings.Contains(userShell, "cmd"):
			return userShell, []string{"/C"}, nil
		case strings.Contains(userShell, "sh") || strings.Contains(userShell, "zsh") || strings.Contains(userShell, "fish"):
			args := []string{"-euc"}
			if verboseLevel >= 2 {
				args = []string{"-xeuc"}
			}
			return userShell, args, nil
		default:
			// For unknown shells, try bash-style arguments first
			args := []string{"-euc"}
			if verboseLevel >= 2 {
				args = []string{"-xeuc"}
			}
			return userShell, args, nil
		}
	}
	
	// No user shell specified - use platform defaults
	if runtime.GOOS == "windows" {
		// Default Windows shell: bash (Git Bash)
		if _, err := exec.LookPath("bash.exe"); err == nil {
			args := []string{"-euc"}
			if verboseLevel >= 2 {
				args = []string{"-xeuc"}
			}
			return "bash.exe", args, nil
		}
		// Fallback to cmd
		return "cmd.exe", []string{"/C"}, nil
	}
	
	// Default Unix shell: sh
	args := []string{"-euc"}
	if verboseLevel >= 2 {
		args = []string{"-xeuc"}
	}
	return "sh", args, nil
}

// executeActionForDAGWithStreamingControl executes a single action for DAG execution with streaming control
func (r *Runner) executeActionForDAGWithStreamingControl(ctx context.Context, action Action, streamingManager *StreamingOutputManager) (Result, error) {
	// Step start callback will be handled by displayStepInOrder when the step becomes current

	var result Result
	var err error

	// Measure execution time from when the action actually starts to when it finishes
	start := time.Now()
	if action.Uses != "" {
		result, err = r.runBuiltInActionForDAG(ctx, action)
	} else {
		result, err = r.runCustomActionForDAGWithStreamingControl(ctx, action, streamingManager)
	}
	duration := time.Since(start)
	
	// Set the duration in the result
	result.Duration = duration

	// Step completion callback will be handled by displayStepInOrder when the step completes

	return result, err
}

// executeActionForDAG executes a single action for DAG execution
func (r *Runner) executeActionForDAG(ctx context.Context, action Action) (Result, error) {
	// Call step start callback if provided
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, action.Name)
	}

	var result Result
	var err error

	// Measure execution time from when the action actually starts to when it finishes
	start := time.Now()
	if action.Uses != "" {
		result, err = r.runBuiltInActionForDAG(ctx, action)
	} else {
		result, err = r.runCustomActionForDAG(ctx, action)
	}
	duration := time.Since(start)
	
	// Set the duration in the result
	result.Duration = duration

	return result, err
}

// executeActionForDAGWithStep executes a single action for DAG execution with step information
func (r *Runner) executeActionForDAGWithStep(ctx context.Context, action Action, step *Step) (Result, error) {
	// Call step start callback if provided
	stepName := action.Name
	if step != nil && step.Description != "" {
		stepName = step.Description // Use description for matrix steps
	}
	// For matrix steps, use the step action name (which includes the matrix suffix)
	if step != nil && step.Action != "" {
		stepName = step.Action // Use step action name for matrix steps (e.g., "container-platform-view.alpine:latest")
	}
	
	// Debug: Print step information
	if r.opts.Debug {
		fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] executeActionForDAGWithStep: action.Name=%s, step.Action=%s, stepName=%s\n", action.Name, step.Action, stepName)
	}
	
	if r.opts.StepCallback != nil {
		r.opts.StepCallback.OnStepStart(ctx, stepName)
	}

	// Execute the action (matrix steps now have interpolated actions)
	start := time.Now()
	var result Result
	var err error
	
	if action.Uses != "" {
		result, err = r.runBuiltInActionForDAG(ctx, action)
	} else {
		result, err = r.runCustomActionForDAG(ctx, action)
	}
	duration := time.Since(start)
	
	// Set the duration in the result
	result.Duration = duration

	return result, err
}

// runBuiltInActionForDAG executes a built-in action for DAG execution
func (r *Runner) runBuiltInActionForDAG(ctx context.Context, action Action) (Result, error) {
	if r.registry == nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("built-in action %s not supported: no action registry provided", action.Uses),
		}, fmt.Errorf("built-in action %s not supported: no action registry provided", action.Uses)
	}
	
	runner, exists := r.registry.GetRunner(action.Uses)
	if !exists {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("unknown built-in action: %s", action.Uses),
		}, fmt.Errorf("unknown built-in action: %s", action.Uses)
	}
	
	result, err := runner.Run(ctx)
	
	// Call step output callback if provided and verbose mode is enabled
	if r.opts.StepCallback != nil && r.opts.VerboseLevel > 0 && result.Message != "" {
		r.opts.StepCallback.OnStepOutput(ctx, action.Name, result.Message)
	}
	
	// Return error if action failed
	if result.Status == StatusError {
		return result, fmt.Errorf("built-in action failed: %s", result.Message)
	}
	
	return result, err
}

// runCustomActionForDAGWithStreamingControl executes a custom action for DAG execution with streaming control
func (r *Runner) runCustomActionForDAGWithStreamingControl(ctx context.Context, action Action, streamingManager *StreamingOutputManager) (Result, error) {
	if action.Run == "" {
		return Result{
			Status:  StatusError,
			Message: "no run command specified",
		}, fmt.Errorf("action %s has no run command", action.Name)
	}
	
	// Interpolate variables in the action
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("failed to interpolate variables: %v", err),
		}, fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	// Apply delay if specified in action options
	if delay, exists := action.Options["delay"]; exists {
		if delaySeconds, ok := delay.(int); ok && delaySeconds > 0 {
			// Wait for the specified delay before executing the command
			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}
	
	// Create command with error handling flags
	shell, shellArgs, err := getShellCommand(action.Shell, r.opts.VerboseLevel)
	if err != nil {
		return Result{
			Name:    action.Name,
			Status:  StatusError,
			Message: fmt.Sprintf("shell configuration error: %v", err),
		}, fmt.Errorf("shell configuration error: %w", err)
	}
	cmd := exec.CommandContext(ctx, shell, append(shellArgs, interpolatedAction.Run)...)
	cmd.Dir = r.opts.WorkingDir
	
	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range r.opts.Variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	var bufferedOutput string
	if r.opts.VerboseLevel > 0 && streamingManager.ShouldStreamOutput(action.Name) {
		// Use streaming output for verbose mode and if this step should stream
		err = r.executeCommandWithStreaming(ctx, cmd, action.Name)
	} else {
		// Use buffered output for non-verbose mode or if this step shouldn't stream
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		// Execute command
		err = cmd.Run()
		
		// Store the output for later display when this step becomes active
		if stdout.Len() > 0 {
			bufferedOutput += stdout.String()
		}
		if stderr.Len() > 0 {
			if bufferedOutput != "" {
				bufferedOutput += "\n"
			}
			bufferedOutput += stderr.String()
		}
		}
		
		if err != nil {
		// Provide better error message with reproduction instructions
		errorMessage := fmt.Sprintf("failed, to check run:\n  %s", action.Run)
		// Debug output
		fmt.Fprintf(os.Stderr, "[DEBUG] runCustomActionForDAGWithStreamingControl: bufferedOutput=%q\n", bufferedOutput)
		return Result{
			Status:         StatusError,
			Message:        errorMessage,
			BufferedOutput: bufferedOutput,
		}, fmt.Errorf("command failed: %w", err)
	}
	
	return Result{
		Status:         StatusOK,
		Message:        "executed successfully",
		BufferedOutput: bufferedOutput, // Store buffered output separately
	}, nil
}

// runCustomActionForDAG executes a custom action for DAG execution
func (r *Runner) runCustomActionForDAG(ctx context.Context, action Action) (Result, error) {
	// Check if action has container configuration
	if action.Container != nil {
		return r.runContainerAction(ctx, action)
	}
	
	if action.Run == "" {
		return Result{
			Status:  StatusError,
			Message: "no run command specified",
		}, fmt.Errorf("action %s has no run command", action.Name)
	}
	
	// Interpolate variables in the action
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("failed to interpolate variables: %v", err),
		}, fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	// Apply delay if specified in action options
	if delay, exists := action.Options["delay"]; exists {
		if delaySeconds, ok := delay.(int); ok && delaySeconds > 0 {
			// Wait for the specified delay before executing the command
			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}
	
	// Create command with error handling flags
	shell, shellArgs, err := getShellCommand(action.Shell, r.opts.VerboseLevel)
	if err != nil {
		return Result{
			Name:    action.Name,
			Status:  StatusError,
			Message: fmt.Sprintf("shell configuration error: %v", err),
		}, fmt.Errorf("shell configuration error: %w", err)
	}
	cmd := exec.CommandContext(ctx, shell, append(shellArgs, interpolatedAction.Run)...)
	cmd.Dir = r.opts.WorkingDir
	
	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range r.opts.Variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Declare stdout and stderr variables for buffered output
	var stdout, stderr strings.Builder
	
	if r.opts.VerboseLevel > 0 {
		// Use streaming output for verbose mode
		err = r.executeCommandWithStreaming(ctx, cmd, action.Name)
	} else {
		// Use buffered output for quiet mode
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		// Execute command
		err = cmd.Run()
		
		// Don't call step output callback in quiet mode - output is buffered
	}
	
	// Capture buffered output for quiet mode
	var bufferedOutput string
	if r.opts.VerboseLevel == 0 {
		if stdout.Len() > 0 {
			bufferedOutput += stdout.String()
		}
		if stderr.Len() > 0 {
			if bufferedOutput != "" {
				bufferedOutput += "\n"
			}
			bufferedOutput += stderr.String()
		}
	}
	
	if err != nil {
		// Provide better error message with reproduction instructions
		errorMessage := fmt.Sprintf("failed, to check run:\n  %s", action.Run)
		return Result{
			Status:         StatusError,
			Message:        errorMessage,
			BufferedOutput: bufferedOutput,
		}, fmt.Errorf("command failed: %w", err)
	}
	
	return Result{
		Status:         StatusOK,
		Message:        "command executed successfully",
		BufferedOutput: bufferedOutput,
	}, nil
}

// runActionInternal executes a single action
func (r *Runner) runActionInternal(ctx context.Context, action Action) error {
	// Select variant if action has variants
	variant, err := action.SelectVariant(r.opts.Variables)
	if err != nil {
		return err
	}
	
	// If variant is nil and action has variants, it means no variant matched - skip
	if variant == nil && len(action.Variants) > 0 {
		// Call step complete callback with skipped status if provided
		if r.opts.StepCallback != nil {
			r.opts.StepCallback.OnStepComplete(ctx, action.Name, StepStatusSkipped, "no matching variant", 0, "")
		}
		
		// Print skip message if verbose mode is enabled
		if r.opts.VerboseLevel > 0 {
			fmt.Fprintf(r.opts.Output, "→ %s: skipped (no matching variant)\n", action.Name)
		}
		
		return nil // Not an error, just skipped
	}
	
	// Use variant if available, otherwise use action directly
	effectiveAction := action
	if variant != nil {
		effectiveAction = Action{
			Name:      action.Name,
			Run:       variant.Run,
			Uses:      variant.Uses,
			Shell:     variant.Shell,
			Container: action.Container, // Preserve container configuration
		}
	}
	
	// Check if action has container configuration
	if effectiveAction.Container != nil {
		_, err := r.runContainerAction(ctx, effectiveAction)
		return err
	}
	
	if effectiveAction.Uses != "" {
		return r.runBuiltInAction(ctx, effectiveAction)
	}
	
	return r.runCustomAction(ctx, effectiveAction)
}

// runActionInternalDryRun simulates action execution for dry-run mode
func (r *Runner) runActionInternalDryRun(ctx context.Context, action Action) error {
	// Select variant if action has variants
	variant, err := action.SelectVariant(r.opts.Variables)
	if err != nil {
		return err
	}
	
	// If variant is nil and action has variants, it means no variant matched - skip
	if variant == nil && len(action.Variants) > 0 {
		// Print skip message if verbose mode is enabled
		if r.opts.VerboseLevel > 0 {
			fmt.Fprintf(r.opts.Output, "→ %s: would skip (no matching variant)\n", action.Name)
		}
		return nil // Not an error, just skipped
	}
	
	// Use variant if available, otherwise use action directly
	effectiveAction := action
	if variant != nil {
		effectiveAction = Action{
			Name:      action.Name,
			Run:       variant.Run,
			Uses:      variant.Uses,
			Shell:     variant.Shell,
			Container: action.Container, // Preserve container configuration
		}
	}
	
	if effectiveAction.Uses != "" {
		return r.runBuiltInActionDryRun(ctx, effectiveAction)
	}
	
	return r.runCustomActionDryRun(ctx, effectiveAction)
}

// runBuiltInActionDryRun simulates built-in action execution for dry-run mode
func (r *Runner) runBuiltInActionDryRun(ctx context.Context, action Action) error {
	if r.registry == nil {
		return fmt.Errorf("built-in action %s not supported: no action registry provided", action.Uses)
	}
	
	runner, exists := r.registry.GetRunner(action.Uses)
	if !exists {
		return fmt.Errorf("unknown built-in action: %s", action.Uses)
	}
	
	description := runner.Description()
	
	// Print what would be executed if verbose mode is enabled
	if r.opts.VerboseLevel > 0 {
		fmt.Fprintf(r.opts.Output, "✓ %s: would execute built-in action: %s\n", action.Name, description)
	}
	
	return nil
}

// runCustomActionDryRun simulates custom action execution for dry-run mode
func (r *Runner) runCustomActionDryRun(ctx context.Context, action Action) error {
	if action.Run == "" {
		return fmt.Errorf("action %s has no run command", action.Name)
	}
	
	// Interpolate variables in the action
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	// Get shell command info
	shell, shellArgs, err := getShellCommand(action.Shell, r.opts.VerboseLevel)
	if err != nil {
		return fmt.Errorf("shell configuration error for action %s: %w", action.Name, err)
	}
	
	// Build the full command
	fullCommand := append(shellArgs, interpolatedAction.Run)
	commandStr := shell + " " + strings.Join(fullCommand, " ")
	
	// Print what would be executed if verbose mode is enabled
	if r.opts.VerboseLevel > 0 {
		fmt.Fprintf(r.opts.Output, "✓ %s: would execute command: %s\n", action.Name, commandStr)
	}
	
	return nil
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
	if r.opts.StepCallback != nil && r.opts.VerboseLevel > 0 && result.Message != "" {
		r.opts.StepCallback.OnStepOutput(ctx, action.Name, result.Message)
	}
	
	// Print result if verbose mode is enabled
	if r.opts.VerboseLevel > 0 {
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
	
	// Interpolate variables in the action
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	// Create command with error handling flags
	shell, shellArgs, err := getShellCommand(action.Shell, r.opts.VerboseLevel)
	if err != nil {
		return fmt.Errorf("shell configuration error for action %s: %w", action.Name, err)
	}
	cmd := exec.CommandContext(ctx, shell, append(shellArgs, interpolatedAction.Run)...)
	cmd.Dir = r.opts.WorkingDir
	
	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range r.opts.Variables {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	if r.opts.VerboseLevel > 0 {
		// Use streaming output for verbose mode
		err = r.executeCommandWithStreaming(ctx, cmd, action.Name)
	} else {
		// Use buffered output for quiet mode - capture stdout/stderr
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		// Execute command
		err = cmd.Run()
		
		// Store buffered output for potential error display
		if r.opts.StepCallback != nil {
			// Store the buffered output in the callback for potential error display
			if callback, ok := r.opts.StepCallback.(*SimpleStepCallback); ok {
				callback.StoreBufferedOutput(action.Name, stdout.String(), stderr.String())
			}
		}
	}
	
	if err != nil {
		// Provide better error message with reproduction instructions
		return fmt.Errorf("failed, to check run:\n  %s", action.Run)
	}
	
	return nil
}

// executeCommandWithStreaming executes a command with real-time output streaming
func (r *Runner) executeCommandWithStreaming(ctx context.Context, cmd *exec.Cmd, actionName string) error {
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Create channels for goroutine communication
	done := make(chan error, 1)
	
	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Call step output callback if provided - this handles the printing
			if r.opts.StepCallback != nil {
				r.opts.StepCallback.OnStepOutput(ctx, actionName, line)
			}
		}
	}()
	
	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Call step output callback if provided - this handles the printing
			if r.opts.StepCallback != nil {
				r.opts.StepCallback.OnStepOutput(ctx, actionName, line)
			}
		}
	}()
	
	// Wait for command completion
	go func() {
		done <- cmd.Wait()
	}()
	
	// Wait for either completion or context cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Kill the command if context is cancelled
		cmd.Process.Kill()
		return ctx.Err()
	}
}

// runContainerAction executes an action inside a container
func (r *Runner) runContainerAction(ctx context.Context, action Action) (Result, error) {
	// Interpolate variables in the container configuration
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("failed to interpolate variables in container action: %v", err),
		}, fmt.Errorf("failed to interpolate variables in container action %s: %w", action.Name, err)
	}
	
	// Interpolate variables in the container configuration
	if interpolatedAction.Container != nil {
		interpolatedContainer, err := InterpolateContainerConfig(interpolatedAction.Container, r.opts.Variables)
		if err != nil {
			return Result{
				Status:  StatusError,
				Message: fmt.Sprintf("failed to interpolate variables in container config: %v", err),
			}, fmt.Errorf("failed to interpolate variables in container config %s: %w", action.Name, err)
		}
		interpolatedAction.Container = interpolatedContainer
	}
	
	// Show container command if verbose level is 2 or higher
	if r.opts.VerboseLevel >= 2 {
		// Create a temporary runner to prepare the configuration for display
		tempRunner, err := containerRunner.NewContainerRunnerWithVerbosity(r.opts.VerboseLevel)
		if err == nil {
			preparedConfig, prepErr := tempRunner.PrepareContainerConfig(*interpolatedAction.Container, r.opts.ConfigPath)
			if prepErr == nil {
				containerCmd := r.buildContainerCommandForDisplay(&preparedConfig)
				fmt.Fprintf(r.opts.ErrorOutput, "  🐳 Running container: %s\n", containerCmd)
			}
		}
	}

	// Create container runner with specified engine or default
	var runner *containerRunner.ContainerRunner
	
	if interpolatedAction.Container.Engine != "" {
		// Use specified engine
		runner, err = containerRunner.NewContainerRunnerWithEngine(interpolatedAction.Container.Engine)
		if err == nil {
			runner.VerbosityLevel = r.opts.VerboseLevel
		}
	} else {
		// Use default engine (Podman)
		runner, err = containerRunner.NewContainerRunnerWithVerbosity(r.opts.VerboseLevel)
	}
	
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("failed to create container runner: %v", err),
			Error:   err,
		}, fmt.Errorf("failed to create container runner: %w", err)
	}
	
	// Container command output is now displayed in OnStepStart callback
	
	// Execute container action using the runner's RunActionWithCallback method
	// This will properly prepare the container configuration for run_action/run_stage
	var containerResult *container.ContainerResult
	err = func() error {
		// Create a callback function for real-time output streaming
		outputCallback := func(line string) {
			if r.opts.StepCallback != nil && r.opts.VerboseLevel > 0 && line != "" {
				r.opts.StepCallback.OnStepOutput(ctx, action.Name, line)
			}
		}
		
			var err error
			containerResult, err = runner.RunActionWithCallback(ctx, *interpolatedAction.Container, r.opts.ConfigPath, outputCallback)
			return err
	}()
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("container action failed: %v", err),
			Error:   err,
		}, fmt.Errorf("container action failed: %w", err)
	}
	
	// Display container output based on verbosity level
	// Container output is now streamed in real-time via the callback

	// Check container exit code
	if containerResult != nil && containerResult.ExitCode != 0 {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("container exited with code %d", containerResult.ExitCode),
			Error:   fmt.Errorf("container exited with code %d", containerResult.ExitCode),
		}, fmt.Errorf("container exited with code %d", containerResult.ExitCode)
	}
	
	// Return successful result
	return Result{
		Status:         StatusOK,
		Message:        "container action executed successfully",
		BufferedOutput: containerResult.Output,
	}, nil
}

// buildContainerCommand builds a human-readable representation of the container command
func (r *Runner) buildContainerCommand(config *container.ContainerConfig) string {
	var parts []string
	
	// Use specified engine or default to podman
	engineName := "podman" // Default to podman
	if config.Engine != "" {
		engineName = config.Engine
	}
	
	// Add engine (Docker/Podman)
	parts = append(parts, engineName)
	
	// Add image
	parts = append(parts, "run", "--rm")
	
	// Add mount arguments
	for _, mount := range config.Mounts {
		mountArg := fmt.Sprintf("--mount=type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target)
		if mount.RO {
			mountArg += ",readonly"
		}
		parts = append(parts, mountArg)
	}
	
	// Add image
	parts = append(parts, config.Image.From)
	
	// Add command to run
	if config.Run != "" {
		parts = append(parts, "sh", "-c", config.Run)
	} else if config.RunAction != "" {
		parts = append(parts, "buildfab", "action", config.RunAction)
	} else if config.RunStage != "" {
		parts = append(parts, "buildfab", "run", config.RunStage)
	}
	
	return strings.Join(parts, " ")
}
