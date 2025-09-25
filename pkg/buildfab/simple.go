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

// formatExecutionTime formats a duration in the requested format (e.g., '20s' or '1m 20s')
// Only shows time for successful actions, returns empty string for errors
func formatExecutionTime(status StepStatus, duration time.Duration) string {
	// Only show execution time on success (OK status)
	if status != StepStatusOK {
		return ""
	}
	
	// Format duration as requested: '20s' or '1m 20s'
	totalSeconds := int(duration.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	
	if minutes > 0 {
		return fmt.Sprintf(" - in '%dm %ds'", minutes, seconds)
	} else if seconds > 0 {
		return fmt.Sprintf(" - in '%ds'", seconds)
	} else {
		// For sub-second durations, show as fractional seconds (e.g., '0.002s')
		return fmt.Sprintf(" - in '%.3fs'", duration.Seconds())
	}
}

// SimpleRunner provides a simplified interface for running stages and actions
// without requiring callback setup. All output is handled internally.
type SimpleRunner struct {
	config   *Config
	opts     *SimpleRunOptions
	registry ActionRegistry
}

// SimpleRunOptions configures simple stage execution
type SimpleRunOptions struct {
	ConfigPath  string            // Path to project.yml (default: ".project.yml")
	MaxParallel int               // Maximum parallel execution (default: CPU count)
	Verbose     bool              // Enable verbose output
	Debug       bool              // Enable debug output
	DryRun      bool              // Show what would be executed without running commands
	Variables   map[string]string // Additional variables for interpolation
	WorkingDir  string            // Working directory for execution
	Output      io.Writer         // Output writer (default: os.Stdout)
	ErrorOutput io.Writer         // Error output writer (default: os.Stderr)
	Only        []string          // Only run steps matching these labels
	WithRequires bool             // Include required dependencies when running single step
}

// DefaultSimpleRunOptions returns default simple run options
func DefaultSimpleRunOptions() *SimpleRunOptions {
	variables := make(map[string]string)
	// Add platform variables by default
	variables = AddPlatformVariables(variables)
	
	return &SimpleRunOptions{
		ConfigPath:  ".project.yml",
		MaxParallel: runtime.NumCPU(),
		Verbose:     false,
		Debug:       false,
		Variables:   variables,
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
		config:   config,
		opts:     opts,
		registry: NewDefaultActionRegistry(),
	}
}

// RunStage executes a specific stage with automatic output handling
func (r *SimpleRunner) RunStage(ctx context.Context, stageName string) error {
	stage, exists := r.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// Handle dry-run mode differently
	if r.opts.DryRun {
		return r.executeStageDryRun(ctx, stageName, stage.Steps)
	}

	// Print stage start message
	fmt.Fprintf(r.opts.Output, "▶️  Running stage: %s\n\n", stageName)

	// Start timing the stage execution
	stageStart := time.Now()

	// Create ordered step callback to collect results with proper ordering
	stepCallback := NewOrderedStepCallback(stage.Steps, r.opts.Verbose, r.opts.Debug, r.opts.ErrorOutput, r.config)

	// Convert to complex options for internal executor
	complexOpts := &RunOptions{
		ConfigPath:   r.opts.ConfigPath,
		MaxParallel:  r.opts.MaxParallel,
		Verbose:      r.opts.Verbose,
		Debug:        r.opts.Debug,
		DryRun:       r.opts.DryRun,
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
	
	// Calculate stage execution duration
	stageDuration := time.Since(stageStart)
	
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
	
	// Check if execution was terminated due to context cancellation
	terminated := ctx.Err() != nil
	success := err == nil && !terminated
	
	// Print summary with termination handling
	if terminated {
		r.printTerminatedSummary(stageName, results, stageDuration)
	} else {
		r.printSummary(stageName, success, results, stageDuration)
	}
	
	return err
}

// RunAction executes a specific action with automatic output handling
func (r *SimpleRunner) RunAction(ctx context.Context, actionName string) error {
	// Print action header
	fmt.Fprintf(r.opts.Output, "▶️  Running action: %s\n\n", actionName)

	// Create step callback to collect results
	stepCallback := &SimpleStepCallback{
		verbose: r.opts.Verbose,
		debug:   r.opts.Debug,
		output:  r.opts.ErrorOutput,  // Use errorOutput for step results
		errorOutput: r.opts.ErrorOutput,
		config:  r.config,
	}

	// Convert to complex options and use internal runner
	complexOpts := &RunOptions{
		ConfigPath:   r.opts.ConfigPath,
		MaxParallel:  r.opts.MaxParallel,
		Verbose:      r.opts.Verbose,
		Debug:        r.opts.Debug,
		DryRun:       r.opts.DryRun,
		Variables:    r.opts.Variables,
		WorkingDir:   r.opts.WorkingDir,
		Output:       r.opts.Output,
		ErrorOutput:  r.opts.ErrorOutput,
		Only:         r.opts.Only,
		WithRequires: r.opts.WithRequires,
		StepCallback: stepCallback,
	}

	runner := NewRunner(r.config, complexOpts)
	err := runner.RunAction(ctx, actionName)
	
	// Get collected results
	results := stepCallback.GetResults()
	
	// Show final status summary for single actions
	if len(results) > 0 {
		// Check if execution was terminated due to context cancellation
		terminated := ctx.Err() != nil
		
		// Check if any step failed or has warnings
		hasError := false
		hasWarning := false
		for _, result := range results {
			if result.Status == StepStatusError {
				hasError = true
				break
			} else if result.Status == StepStatusWarn {
				hasWarning = true
			}
		}
		
		// Show final status
		fmt.Fprintf(r.opts.Output, "\n")
		fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		if terminated {
			fmt.Fprintf(r.opts.Output, "⏹️ %s%s%s - %s\n", colorYellow, "TERMINATED", colorReset, actionName)
		} else if hasError {
			fmt.Fprintf(r.opts.Output, "💥 %s%s%s - %s\n", colorRed, "FAILED", colorReset, actionName)
		} else if hasWarning {
			fmt.Fprintf(r.opts.Output, "⚠️ %s%s%s - %s\n", colorYellow, "WARNING", colorReset, actionName)
		} else {
			fmt.Fprintf(r.opts.Output, "🎉 %s%s%s - %s\n", colorGreen, "SUCCESS", colorReset, actionName)
		}
	}
	
	return err
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
		DryRun:       r.opts.DryRun,
		Variables:    r.opts.Variables,
		WorkingDir:   r.opts.WorkingDir,
		Output:       r.opts.Output,
		ErrorOutput:  r.opts.ErrorOutput,
		Only:         r.opts.Only,
		WithRequires: r.opts.WithRequires,
		StepCallback: &SimpleStepCallback{
			verbose: r.opts.Verbose,
			debug:   r.opts.Debug,
			output:  r.opts.ErrorOutput,  // Use errorOutput for step results
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


func (c *SimpleStepCallback) OnStepStart(ctx context.Context, stepName string) {
	if c.verbose {
		fmt.Fprintf(c.errorOutput, "  💻 %s\n", stepName)
	} else {
		// In silence mode, show running indicator
		fmt.Fprintf(c.errorOutput, "  %s%s%s %s running...\r", colorCyan, "○", colorReset, stepName)
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
		
		// Add execution time for successful actions
		executionTime := formatExecutionTime(status, duration)
		displayMessage += executionTime
		
		if c.verbose {
			// In verbose mode, just print the result
			fmt.Fprintf(c.errorOutput, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, displayMessage)
		} else {
			// In silence mode, replace the running indicator with the result
			// Clear the running line and print the final result
			fmt.Fprintf(c.errorOutput, "\r  %s%s%s %s %s\n", color, icon, colorReset, stepName, displayMessage)
		}
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
	case StepStatusOK:
		// For dry-run messages, return as-is
		if strings.Contains(message, "would execute") {
			return message
		}
		return message
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

// printTerminatedSummary prints the stage result when execution was terminated
func (r *SimpleRunner) printTerminatedSummary(stageName string, results []StepResult, duration time.Duration) {
	fmt.Fprintf(r.opts.Output, "\n")
	fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	icon := "⏹️"
	color := colorYellow
	status := "TERMINATED"
	
	// Format duration to match step execution time format: 'in 0.021s'
	var durationStr string
	if duration.Seconds() < 1 {
		// For sub-second durations, show as fractional seconds (e.g., 'in 0.021s')
		durationStr = fmt.Sprintf(" in %.3fs", duration.Seconds())
	} else {
		// For durations >= 1 second, show as whole seconds (e.g., 'in 3s')
		totalSeconds := int(duration.Seconds())
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		
		if minutes > 0 {
			durationStr = fmt.Sprintf(" in %dm %ds", minutes, seconds)
		} else {
			durationStr = fmt.Sprintf(" in %ds", seconds)
		}
	}
	
	fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s%s\n", icon, color, status, colorReset, stageName, durationStr)
	
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

// printSummary prints the stage result with summary
func (r *SimpleRunner) printSummary(stageName string, success bool, results []StepResult, duration time.Duration) {
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
	
	// Format duration to match step execution time format: 'in 0.021s'
	var durationStr string
	if duration.Seconds() < 1 {
		// For sub-second durations, show as fractional seconds (e.g., 'in 0.021s')
		durationStr = fmt.Sprintf(" in %.3fs", duration.Seconds())
	} else {
		// For durations >= 1 second, show as whole seconds (e.g., 'in 3s')
		totalSeconds := int(duration.Seconds())
		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		
		if minutes > 0 {
			durationStr = fmt.Sprintf(" in %dm %ds", minutes, seconds)
		} else {
			durationStr = fmt.Sprintf(" in %ds", seconds)
		}
	}
	
	fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s%s\n", icon, color, status, colorReset, stageName, durationStr)
	
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

// executeStageDryRun simulates stage execution for dry-run mode
func (r *SimpleRunner) executeStageDryRun(ctx context.Context, stageName string, steps []Step) error {
	// Print stage header
	fmt.Fprintf(r.opts.Output, "▶️  Dry run stage: %s\n\n", stageName)
	
	// Count total steps
	totalSteps := len(steps)
	skippedSteps := 0
	executedSteps := 0
	
	// Process each step
	for _, step := range steps {
		// Check if step should be executed based on conditions
		shouldExecute, err := r.shouldExecuteStepByCondition(ctx, step)
		if err != nil {
			return fmt.Errorf("failed to evaluate step condition: %w", err)
		}
		
		if !shouldExecute {
			skippedSteps++
			if r.opts.Verbose {
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
				if r.opts.Verbose {
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
				if r.opts.Verbose {
					fmt.Fprintf(r.opts.Output, "  ✓ %s would execute built-in action: %s\n", step.Action, description)
				}
			} else {
				if r.opts.Verbose {
					fmt.Fprintf(r.opts.Output, "  ✗ %s would fail (action not found)\n", step.Action)
				}
			}
		} else {
			// Simulate custom action execution
			err := r.runActionInternalDryRun(ctx, action)
			if err != nil {
				if r.opts.Verbose {
					fmt.Fprintf(r.opts.Output, "  ✗ %s would fail (%v)\n", step.Action, err)
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

// shouldExecuteStepByCondition determines if a step should be executed based on its if condition
func (r *SimpleRunner) shouldExecuteStepByCondition(ctx context.Context, step Step) (bool, error) {
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

// runActionInternalDryRun simulates action execution for dry-run mode
func (r *SimpleRunner) runActionInternalDryRun(ctx context.Context, action Action) error {
	// Select variant if action has variants
	variant, err := action.SelectVariant(r.opts.Variables)
	if err != nil {
		return err
	}
	
	// If variant is nil and action has variants, it means no variant matched - skip
	if variant == nil && len(action.Variants) > 0 {
		return nil // Not an error, just skipped
	}
	
	// Use variant if available, otherwise use action directly
	effectiveAction := action
	if variant != nil {
		effectiveAction = Action{
			Name:  action.Name,
			Run:   variant.Run,
			Uses:  variant.Uses,
			Shell: variant.Shell,
		}
	}
	
	if effectiveAction.Uses != "" {
		return r.runBuiltInActionDryRun(ctx, effectiveAction)
	}
	
	return r.runCustomActionDryRun(ctx, effectiveAction)
}

// runBuiltInActionDryRun simulates built-in action execution for dry-run mode
func (r *SimpleRunner) runBuiltInActionDryRun(ctx context.Context, action Action) error {
	if r.registry == nil {
		return fmt.Errorf("built-in action %s not supported: no action registry provided", action.Uses)
	}
	
	runner, exists := r.registry.GetRunner(action.Uses)
	if !exists {
		return fmt.Errorf("unknown built-in action: %s", action.Uses)
	}
	
	description := runner.Description()
	
	// Print what would be executed if verbose mode is enabled
	if r.opts.Verbose {
		fmt.Fprintf(r.opts.Output, "  ✓ %s would execute built-in action: %s\n", action.Name, description)
	}
	
	return nil
}

// runCustomActionDryRun simulates custom action execution for dry-run mode
func (r *SimpleRunner) runCustomActionDryRun(ctx context.Context, action Action) error {
	if action.Run == "" {
		return fmt.Errorf("action %s has no run command", action.Name)
	}
	
	// Interpolate variables in the action
	interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
	if err != nil {
		return fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	// Get shell command info
	shell, shellArgs, err := getShellCommand(action.Shell)
	if err != nil {
		return fmt.Errorf("shell configuration error for action %s: %w", action.Name, err)
	}
	
	// Build the full command
	fullCommand := append(shellArgs, interpolatedAction.Run)
	commandStr := shell + " " + strings.Join(fullCommand, " ")
	
	// Print what would be executed if verbose mode is enabled
	if r.opts.Verbose {
		fmt.Fprintf(r.opts.Output, "  💻 %s\n", action.Name)
		fmt.Fprintf(r.opts.Output, "   ✓ %s, would execute command:\n", action.Name)
		
		// Handle multiline commands with proper indentation
		lines := strings.Split(commandStr, "\n")
		for i, line := range lines {
			if i == 0 {
				// First line with 6 spaces indentation
				fmt.Fprintf(r.opts.Output, "      %s\n", line)
			} else {
				// Subsequent lines with 6 spaces indentation
				fmt.Fprintf(r.opts.Output, "      %s\n", line)
			}
		}
	}
	
	return nil
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