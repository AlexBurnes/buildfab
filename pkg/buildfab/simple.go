package buildfab

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// SimpleRunner provides a simplified interface for running stages and actions
// without requiring callback setup. All output is handled internally.
type SimpleRunner struct {
	config *Config
	opts   *SimpleRunOptions
}

// SimpleRunOptions configures simple stage execution
type SimpleRunOptions struct {
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

// DefaultSimpleRunOptions returns default simple run options
func DefaultSimpleRunOptions() *SimpleRunOptions {
	return &SimpleRunOptions{
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

// NewSimpleRunner creates a new simple buildfab runner
func NewSimpleRunner(config *Config, opts *SimpleRunOptions) *SimpleRunner {
	if opts == nil {
		opts = DefaultSimpleRunOptions()
	}
	return &SimpleRunner{
		config: config,
		opts:   opts,
	}
}

// RunStage executes a specific stage with automatic output handling
func (r *SimpleRunner) RunStage(ctx context.Context, stageName string) error {
	_, exists := r.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// Print stage header
	fmt.Fprintf(r.opts.Output, "▶️  Running stage: %s\n\n", stageName)

	// Create step callback to collect results
	stepCallback := &SimpleStepCallback{
		verbose: r.opts.Verbose,
		debug:   r.opts.Debug,
		output:  r.opts.Output,
		errorOutput: r.opts.ErrorOutput,
		config:  r.config,
	}

	// Convert to complex options and use internal runner
	complexOpts := &RunOptions{
		ConfigPath:   r.opts.ConfigPath,
		MaxParallel:  r.opts.MaxParallel,
		Verbose:      r.opts.Verbose,
		Debug:        r.opts.Debug,
		Variables:    r.opts.Variables,
		WorkingDir:   r.opts.WorkingDir,
		Output:       r.opts.Output,
		ErrorOutput:  r.opts.ErrorOutput,
		Only:         r.opts.Only,
		WithRequires: r.opts.WithRequires,
		StepCallback: stepCallback,
	}

	runner := NewRunner(r.config, complexOpts)
	err := runner.RunStage(ctx, stageName)
	
	// Get collected results
	results := stepCallback.GetResults()
	
	// Handle skipped steps that weren't executed due to dependencies
	skippedSteps := r.getSkippedSteps(stageName, results)
	for _, stepName := range skippedSteps {
		// Call step callbacks for skipped steps
		stepCallback.OnStepStart(ctx, stepName)
		stepCallback.OnStepComplete(ctx, stepName, StepStatusSkipped, "skipped (dependency failed)", 0)
	}
	
	// Get updated results after handling skipped steps
	results = stepCallback.GetResults()
	
	// Print summary
	r.printSummary(stageName, err == nil, results)
	
	return err
}

// RunAction executes a specific action with automatic output handling
func (r *SimpleRunner) RunAction(ctx context.Context, actionName string) error {
	// Print action header
	fmt.Fprintf(r.opts.Output, "▶️  Running action: %s\n\n", actionName)

	// Convert to complex options and use internal runner
	complexOpts := &RunOptions{
		ConfigPath:   r.opts.ConfigPath,
		MaxParallel:  r.opts.MaxParallel,
		Verbose:      r.opts.Verbose,
		Debug:        r.opts.Debug,
		Variables:    r.opts.Variables,
		WorkingDir:   r.opts.WorkingDir,
		Output:       r.opts.Output,
		ErrorOutput:  r.opts.ErrorOutput,
		Only:         r.opts.Only,
		WithRequires: r.opts.WithRequires,
		StepCallback: &SimpleStepCallback{
			verbose: r.opts.Verbose,
			debug:   r.opts.Debug,
			output:  r.opts.Output,
			errorOutput: r.opts.ErrorOutput,
			config:  r.config,
		},
	}

	runner := NewRunner(r.config, complexOpts)
	return runner.RunAction(ctx, actionName)
}

// RunStageStep executes a specific step within a stage with automatic output handling
func (r *SimpleRunner) RunStageStep(ctx context.Context, stageName, stepName string) error {
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

	// Print step header
	fmt.Fprintf(r.opts.Output, "▶️  Running step: %s (from stage: %s)\n\n", stepName, stageName)

	// Convert to complex options and use internal runner
	complexOpts := &RunOptions{
		ConfigPath:   r.opts.ConfigPath,
		MaxParallel:  r.opts.MaxParallel,
		Verbose:      r.opts.Verbose,
		Debug:        r.opts.Debug,
		Variables:    r.opts.Variables,
		WorkingDir:   r.opts.WorkingDir,
		Output:       r.opts.Output,
		ErrorOutput:  r.opts.ErrorOutput,
		Only:         r.opts.Only,
		WithRequires: r.opts.WithRequires,
		StepCallback: &SimpleStepCallback{
			verbose: r.opts.Verbose,
			debug:   r.opts.Debug,
			output:  r.opts.Output,
			errorOutput: r.opts.ErrorOutput,
			config:  r.config,
		},
	}

	runner := NewRunner(r.config, complexOpts)
	return runner.RunStageStep(ctx, stageName, stepName)
}

// SimpleStepCallback implements StepCallback for simple output
type SimpleStepCallback struct {
	verbose     bool
	debug       bool
	output      io.Writer
	errorOutput io.Writer
	results     []StepResult
	displayed   map[string]bool
	config      *Config // Store config to access action details
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func (c *SimpleStepCallback) OnStepStart(ctx context.Context, stepName string) {
	if c.verbose {
		fmt.Fprintf(c.output, "  💻 %s\n", stepName)
	}
}

func (c *SimpleStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	// Initialize displayed map if not already done
	if c.displayed == nil {
		c.displayed = make(map[string]bool)
	}
	
	// Only display each step once
	if !c.displayed[stepName] {
		var icon, color string
		switch status {
		case StepStatusOK:
			icon = "✓"
			color = colorGreen
		case StepStatusWarn:
			icon = "!"
			color = colorYellow
		case StepStatusError:
			icon = "✗"
			color = colorRed
		case StepStatusSkipped:
			icon = "→"
			color = colorGray
		default:
			icon = "?"
			color = colorGray
		}
		
		// Show step results with enhanced error messages
		displayMessage := c.enhanceMessage(stepName, status, message)
		fmt.Fprintf(c.output, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, displayMessage)
		c.displayed[stepName] = true
	}
	
	// Collect result for summary (avoid duplicates)
	found := false
	for i, result := range c.results {
		if result.StepName == stepName {
			c.results[i] = StepResult{
				StepName: stepName,
				Status:   status,
				Duration: duration,
			}
			found = true
			break
		}
	}
	if !found {
		c.results = append(c.results, StepResult{
			StepName: stepName,
			Status:   status,
			Duration: duration,
		})
	}
}

func (c *SimpleStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	if c.verbose && output != "" {
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		for _, line := range lines {
			fmt.Fprintf(c.output, "    %s\n", line)
		}
	}
}

func (c *SimpleStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	// Don't display here - OnStepComplete should handle all display
}

// GetResults returns the collected step results
func (c *SimpleStepCallback) GetResults() []StepResult {
	return c.results
}

// enhanceMessage improves error messages to match v0.5.0 style
func (c *SimpleStepCallback) enhanceMessage(stepName string, status StepStatus, message string) string {
	switch status {
	case StepStatusError:
		// Check if message already contains reproduction instructions
		if strings.Contains(message, "to check run:") {
			// Extract just the reproduction part and reformat with proper indentation
			lines := strings.Split(message, "\n")
			var reproductionLines []string
			foundReproduction := false
			for _, line := range lines {
				if strings.Contains(line, "to check run:") {
					foundReproduction = true
					reproductionLines = append(reproductionLines, line)
				} else if foundReproduction {
					reproductionLines = append(reproductionLines, line)
				}
			}
			if len(reproductionLines) > 0 {
				// Reformat the command with proper indentation
				command := c.extractCommand(stepName, strings.Join(reproductionLines, "\n"))
				return fmt.Sprintf("to check run:\n%s", command)
			}
		}
		// Check if message is wrapped by runStageInternal
		if strings.Contains(message, "step ") && strings.Contains(message, " failed: ") {
			// Extract the original error message after "failed: "
			parts := strings.Split(message, " failed: ")
			if len(parts) > 1 {
				originalMessage := parts[1]
				// If original message already has reproduction instructions, use it
				if strings.Contains(originalMessage, "to check run:") {
					return originalMessage
				}
				// If original message doesn't have reproduction instructions, add them
				command := c.extractCommand(stepName, message)
				return fmt.Sprintf("to check run:\n%s", command)
			}
		}
		// Enhance error messages with reproduction instructions
		if strings.Contains(message, "command failed: exit status") {
			// Extract the command from the step name or message
			command := c.extractCommand(stepName, message)
			return fmt.Sprintf("to check run:\n%s", command)
		}
		return message
	case StepStatusSkipped:
		// Enhance skipped messages with dependency information
		if strings.Contains(message, "dependency failed") {
			// Try to extract which dependency failed
			failedDep := c.extractFailedDependency(stepName)
			if failedDep != "" {
				return fmt.Sprintf("skipped (dependency failed: %s)", failedDep)
			}
		}
		return message
	default:
		return message
	}
}

// extractCommand tries to extract the command that failed
func (c *SimpleStepCallback) extractCommand(stepName, message string) string {
	if c.config == nil {
		return fmt.Sprintf("      %s", stepName)
	}
	
		// Try to find the action and extract the actual command
		action, exists := c.config.GetAction(stepName)
		if exists && action.Run != "" {
			lines := strings.Split(action.Run, "\n")
			var alignedLines []string
			
			// Find minimum leading spaces across all lines
			minLeadingSpaces := 999 // Start with a large number
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue // Skip empty lines
				}
				lineLeadingSpaces := 0
				for _, char := range line {
					if char == ' ' {
						lineLeadingSpaces++
					} else {
						break
					}
				}
				if lineLeadingSpaces < minLeadingSpaces {
					minLeadingSpaces = lineLeadingSpaces
				}
			}
			
			// If no non-empty lines found, use 0
			if minLeadingSpaces == 999 {
				minLeadingSpaces = 0
			}
			
			for _, line := range lines {
				// Remove minimum leading spaces to preserve relative indentation
				trimmedLine := line
				if len(line) >= minLeadingSpaces {
					trimmedLine = line[minLeadingSpaces:]
				}
				
				// Add 6 spaces indentation
				alignedLines = append(alignedLines, "      "+trimmedLine)
			}
			return strings.Join(alignedLines, "\n")
		}
	
	// Fallback to step name
	return fmt.Sprintf("      %s", stepName)
}

// extractFailedDependency tries to determine which dependency failed
func (c *SimpleStepCallback) extractFailedDependency(stepName string) string {
	// Look through results to find failed dependencies
	for _, result := range c.results {
		if result.Status == StepStatusError {
			// This is a simple heuristic - in a real implementation,
			// we'd need to track the dependency graph
			return result.StepName
		}
	}
	return ""
}

// getSkippedSteps determines which steps should be skipped due to failed dependencies
func (r *SimpleRunner) getSkippedSteps(stageName string, executedResults []StepResult) []string {
	stage, exists := r.config.GetStage(stageName)
	if !exists {
		return nil
	}
	
	// Create a map of executed steps
	executedSteps := make(map[string]bool)
	for _, result := range executedResults {
		executedSteps[result.StepName] = true
	}
	
	// Create a map of failed steps
	failedSteps := make(map[string]bool)
	for _, result := range executedResults {
		if result.Status == StepStatusError {
			failedSteps[result.StepName] = true
		}
	}
	
	var skippedSteps []string
	
	// Check each step in the stage
	for _, step := range stage.Steps {
		stepName := step.Action
		
		// Skip if already executed
		if executedSteps[stepName] {
			continue
		}
		
		// Check if any required dependencies failed
		shouldSkip := false
		for _, requiredStep := range step.Require {
			if failedSteps[requiredStep] {
				shouldSkip = true
				break
			}
		}
		
		if shouldSkip {
			skippedSteps = append(skippedSteps, stepName)
		}
	}
	
	return skippedSteps
}

// printSummary prints the stage result with summary
func (r *SimpleRunner) printSummary(stageName string, success bool, results []StepResult) {
	fmt.Fprintf(r.opts.Output, "\n")
	fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	var icon, color, status string
	if success {
		icon = "🎉"
		color = colorGreen
		status = "SUCCESS"
	} else {
		icon = "💥"
		color = colorRed
		status = "FAILED"
	}
	
	fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s\n", icon, color, status, colorReset, stageName)
	
	// Print summary
	if len(results) > 0 {
		fmt.Fprintf(r.opts.Output, "\n")
		fmt.Fprintf(r.opts.Output, "📊 Summary:\n")
		
		statusCounts := make(map[StepStatus]int)
		for _, result := range results {
			statusCounts[result.Status]++
		}
		
		// Define status order for consistent display
		statusOrder := []StepStatus{
			StepStatusError,
			StepStatusWarn,
			StepStatusOK,
			StepStatusSkipped,
		}
		
		for _, status := range statusOrder {
			count := statusCounts[status]
			var icon, color string
			
			switch status {
			case StepStatusOK:
				icon = "✓"
				if count > 0 {
					color = colorGreen
				} else {
					color = colorGray
				}
			case StepStatusWarn:
				icon = "!"
				if count > 0 {
					color = colorYellow
				} else {
					color = colorGray
				}
			case StepStatusError:
				icon = "✗"
				if count > 0 {
					color = colorRed
				} else {
					color = colorGray
				}
			case StepStatusSkipped:
				icon = "→"
				if count > 0 {
					color = colorGray
				} else {
					color = colorGray
				}
			default:
				icon = "?"
				if count > 0 {
					color = colorGray
				} else {
					color = colorGray
				}
			}
			
			fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", color, icon, colorReset, color, status.String(), count, colorReset)
		}
	}
}

// Simple convenience functions for even easier usage

// RunStageSimple executes a stage with minimal configuration
func RunStageSimple(ctx context.Context, configPath, stageName string, verbose bool) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	opts := &SimpleRunOptions{
		ConfigPath: configPath,
		Verbose:    verbose,
		Output:     os.Stdout,
		ErrorOutput: os.Stderr,
	}

	runner := NewSimpleRunner(cfg, opts)
	return runner.RunStage(ctx, stageName)
}

// RunActionSimple executes an action with minimal configuration
func RunActionSimple(ctx context.Context, configPath, actionName string, verbose bool) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	opts := &SimpleRunOptions{
		ConfigPath: configPath,
		Verbose:    verbose,
		Output:     os.Stdout,
		ErrorOutput: os.Stderr,
	}

	runner := NewSimpleRunner(cfg, opts)
	return runner.RunAction(ctx, actionName)
}