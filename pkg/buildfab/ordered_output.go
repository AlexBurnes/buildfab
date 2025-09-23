package buildfab

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// OrderedOutputManager manages step output in proper order using a queue-based approach
// This implements the architecture where:
// 1. Executor only runs tasks and reports to output manager, no direct output
// 2. Output manager has a queue and outputs steps in proper order
// 3. Step execution reports to manager, not UI directly
// 4. Manager outputs step start → step output → step result in sequence
type OrderedOutputManager struct {
	steps       []Step                     // Steps in declaration order
	stepData    map[string]*StepOutputData // Buffered data for each step
	currentStep string                     // Currently active step for output
	mu          *sync.Mutex
	verbose     bool
	debug       bool
	errorOutput io.Writer
	config      *Config                    // Configuration for command extraction
}

// StepOutputData contains all output data for a step
type StepOutputData struct {
	Started   bool
	Completed bool
	Shown     bool  // Track if step start message has been shown
	Status    StepStatus
	Message   string
	Duration  time.Duration
	Output    []string
	Error     error
}

// NewOrderedOutputManager creates a new ordered output manager
func NewOrderedOutputManager(steps []Step, verbose bool, debug bool, errorOutput io.Writer, config *Config) *OrderedOutputManager {
	return &OrderedOutputManager{
		steps:       steps,
		stepData:    make(map[string]*StepOutputData),
		mu:          &sync.Mutex{},
		verbose:     verbose,
		debug:       debug,
		errorOutput: errorOutput,
		config:      config,
	}
}

// RegisterStep registers a step for execution
func (o *OrderedOutputManager) RegisterStep(stepName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	o.stepData[stepName] = &StepOutputData{}
}

// OnStepStart handles step start events from executor
func (o *OrderedOutputManager) OnStepStart(ctx context.Context, stepName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if o.debug {
		fmt.Fprintf(o.errorOutput, "[DEBUG] OnStepStart: %s\n", stepName)
		o.debugPrintState()
	}
	
	if data, exists := o.stepData[stepName]; exists {
		data.Started = true
	}
	
	// Show step start message if this is the next step in order and not already shown
	if o.canShowStepStart(stepName) && !o.stepData[stepName].Shown {
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] Showing step start for: %s\n", stepName)
		}
		o.showStepStart(stepName)
		o.stepData[stepName].Shown = true
		o.currentStep = stepName
	} else {
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] Cannot show step start for: %s (not ready or already shown)\n", stepName)
		}
	}
}

// OnStepComplete handles step completion events from executor
func (o *OrderedOutputManager) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if o.debug {
		fmt.Fprintf(o.errorOutput, "[DEBUG] OnStepComplete: %s (status: %s)\n", stepName, status)
		o.debugPrintState()
	}
	
	if data, exists := o.stepData[stepName]; exists {
		data.Completed = true
		data.Status = status
		data.Message = message
		data.Duration = duration
	}
	
	// Show step completion message if this is the current step
	if o.currentStep == stepName {
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] Showing step completion for: %s\n", stepName)
		}
		o.showStepCompletion(stepName)
		o.currentStep = ""
	}
	
	// Check if any completed steps can now be shown in order
	o.checkAndShowCompletedSteps()
	
	// Check if next step can be shown
	o.checkAndShowNextStep()
}

// OnStepOutput handles step output events from executor
func (o *OrderedOutputManager) OnStepOutput(ctx context.Context, stepName string, output string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if data, exists := o.stepData[stepName]; exists {
		data.Output = append(data.Output, output)
	}
	
	// Stream output immediately if this is the current active step
	if o.currentStep == stepName {
		o.showStepOutput(output)
	}
}

// OnStepError handles step error events from executor
func (o *OrderedOutputManager) OnStepError(ctx context.Context, stepName string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if data, exists := o.stepData[stepName]; exists {
		data.Error = err
	}
}

// canShowStepStart checks if a step can show its start message
func (o *OrderedOutputManager) canShowStepStart(stepName string) bool {
	// Find step index in declaration order
	stepIndex := -1
	for i, step := range o.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: step %s not found in steps\n", stepName)
		}
		return false
	}
	
	// First step can always show start
	if stepIndex == 0 {
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s is first step, can show start\n", stepName)
		}
		return true
	}
	
	// Check if all previous steps have been completed
	for i := 0; i < stepIndex; i++ {
		prevStepName := o.steps[i].Action
		if data, exists := o.stepData[prevStepName]; !exists || !data.Completed {
			if o.debug {
				fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s cannot show start, previous step %s not completed (exists: %v, completed: %v)\n", 
					stepName, prevStepName, exists, exists && data.Completed)
			}
			return false
		}
	}
	
	if o.debug {
		fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s can show start, all previous steps completed\n", stepName)
	}
	return true
}

// checkAndShowCompletedSteps checks if any completed steps can now be shown in order
func (o *OrderedOutputManager) checkAndShowCompletedSteps() {
	if o.debug {
		fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowCompletedSteps: checking for completed steps to show\n")
	}
	
	// Find completed steps that can be shown in order
	for _, step := range o.steps {
		stepName := step.Action
		if data, exists := o.stepData[stepName]; exists && data.Completed && !data.Shown {
			// Check if all previous steps have been completed AND shown
			canShow := true
			for _, s := range o.steps {
				if s.Action == stepName {
					break
				}
				prevData, prevExists := o.stepData[s.Action]
				if !prevExists || !prevData.Completed || !prevData.Shown {
					canShow = false
					break
				}
			}
			
			if canShow {
				if o.debug {
					fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowCompletedSteps: showing completed step: %s\n", stepName)
				}
				o.showStepStart(stepName)
				o.stepData[stepName].Shown = true
				o.currentStep = stepName
				
				// Flush any buffered output for this step
				o.flushBufferedOutput(stepName)
				
				o.showStepCompletion(stepName)
				o.currentStep = ""
			}
		}
	}
}

// checkAndShowNextStep checks if the next step can be shown
func (o *OrderedOutputManager) checkAndShowNextStep() {
	if o.debug {
		fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: checking for next step to show\n")
	}
	
	// Find the next step that can be shown
	for _, step := range o.steps {
		stepName := step.Action
		if data, exists := o.stepData[stepName]; exists && data.Started && !data.Completed && !data.Shown {
			if o.canShowStepStart(stepName) {
				if o.debug {
					fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: showing next step: %s\n", stepName)
				}
				o.showStepStart(stepName)
				o.stepData[stepName].Shown = true
				o.currentStep = stepName
				
				// Flush any buffered output for this step
				o.flushBufferedOutput(stepName)
				break
			}
		}
	}
}

// showStepStart shows the start message for a step
func (o *OrderedOutputManager) showStepStart(stepName string) {
	if o.verbose {
		fmt.Fprintf(o.errorOutput, "  💻 %s\n", stepName)
	} else {
		// In silence mode, show running indicator
		fmt.Fprintf(o.errorOutput, "  %s%s%s %s running...\r", colorCyan, "○", colorReset, stepName)
	}
}

// showStepCompletion shows the completion message for a step
func (o *OrderedOutputManager) showStepCompletion(stepName string) {
	if data, exists := o.stepData[stepName]; exists {
		// Output is now streamed immediately, so no need to show buffered output here
		// Just show the completion message
		o.showStepResult(stepName, data.Status, data.Message, data.Duration)
	}
}

// showStepOutput shows step output
func (o *OrderedOutputManager) showStepOutput(output string) {
	if o.verbose {
		fmt.Fprintf(o.errorOutput, "    %s\n", output)
	}
}

// flushBufferedOutput flushes all buffered output for a step
func (o *OrderedOutputManager) flushBufferedOutput(stepName string) {
	if data, exists := o.stepData[stepName]; exists {
		for _, output := range data.Output {
			o.showStepOutput(output)
		}
		// Clear the buffered output after flushing
		data.Output = nil
	}
}

// showStepResult shows the result message for a step
func (o *OrderedOutputManager) showStepResult(stepName string, status StepStatus, message string, duration time.Duration) {
	// Enhance error messages with reproduction instructions
	enhancedMessage := o.enhanceMessage(stepName, status, message)
	
	// Add execution time for successful actions
	executionTime := formatExecutionTime(status, duration)
	enhancedMessage += executionTime
	
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
	
	if o.verbose {
		// In verbose mode, just print the result
		fmt.Fprintf(o.errorOutput, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, enhancedMessage)
	} else {
		// In silence mode, replace the running indicator with the result
		fmt.Fprintf(o.errorOutput, "\r  %s%s%s %s %s\n", color, icon, colorReset, stepName, enhancedMessage)
	}
}

// OrderedStepCallback implements StepCallback interface using the ordered output manager
type OrderedStepCallback struct {
	manager *OrderedOutputManager
	results []StepResult
	mu      *sync.Mutex
}

// NewOrderedStepCallback creates a new ordered step callback
func NewOrderedStepCallback(steps []Step, verbose bool, debug bool, errorOutput io.Writer, config *Config) *OrderedStepCallback {
	manager := NewOrderedOutputManager(steps, verbose, debug, errorOutput, config)
	
	// Register all steps
	for _, step := range steps {
		manager.RegisterStep(step.Action)
	}
	
	return &OrderedStepCallback{
		manager: manager,
		results: make([]StepResult, 0),
		mu:      &sync.Mutex{},
	}
}

// OnStepStart implements StepCallback interface
func (c *OrderedStepCallback) OnStepStart(ctx context.Context, stepName string) {
	c.manager.OnStepStart(ctx, stepName)
}

// OnStepComplete implements StepCallback interface
func (c *OrderedStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration) {
	c.manager.OnStepComplete(ctx, stepName, status, message, duration)
	
	// Collect result for summary (thread-safe)
	c.mu.Lock()
	c.results = append(c.results, StepResult{
		StepName: stepName,
		Status:   status,
		Duration: duration,
	})
	c.mu.Unlock()
}

// OnStepOutput implements StepCallback interface
func (c *OrderedStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	c.manager.OnStepOutput(ctx, stepName, output)
}

// OnStepError implements StepCallback interface
func (c *OrderedStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	c.manager.OnStepError(ctx, stepName, err)
}

// GetResults returns the collected step results
func (c *OrderedStepCallback) GetResults() []StepResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Return a copy to avoid race conditions
	result := make([]StepResult, len(c.results))
	copy(result, c.results)
	return result
}

// debugPrintState prints the current state of the output manager
func (o *OrderedOutputManager) debugPrintState() {
	fmt.Fprintf(o.errorOutput, "[DEBUG] Output Manager State:\n")
	fmt.Fprintf(o.errorOutput, "  Current Step: %s\n", o.currentStep)
	fmt.Fprintf(o.errorOutput, "  Steps in order: ")
	for i, step := range o.steps {
		if i > 0 {
			fmt.Fprintf(o.errorOutput, ", ")
		}
		fmt.Fprintf(o.errorOutput, "%s", step.Action)
	}
	fmt.Fprintf(o.errorOutput, "\n")
	fmt.Fprintf(o.errorOutput, "  Step Data:\n")
	for stepName, data := range o.stepData {
		fmt.Fprintf(o.errorOutput, "    %s: started=%v, completed=%v, status=%s\n", 
			stepName, data.Started, data.Completed, data.Status)
	}
}

// enhanceMessage improves error messages to match v0.5.0 style
func (o *OrderedOutputManager) enhanceMessage(stepName string, status StepStatus, message string) string {
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
				command := o.extractCommand(stepName, strings.Join(reproductionLines, "\n"))
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
				command := o.extractCommand(stepName, message)
				return fmt.Sprintf("to check run:\n%s", command)
			}
		}
		// Enhance error messages with reproduction instructions
		if strings.Contains(message, "command failed: exit status") {
			// Extract the command from the step name or message
			command := o.extractCommand(stepName, message)
			return fmt.Sprintf("to check run:\n%s", command)
		}
		return message
	case StepStatusSkipped:
		// Enhance skipped messages with dependency information
		if strings.Contains(message, "dependency failed") {
			// Try to extract which dependency failed
			failedDep := o.extractFailedDependency(stepName)
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
func (o *OrderedOutputManager) extractCommand(stepName, message string) string {
	if o.config == nil {
		return fmt.Sprintf("      %s", stepName)
	}
	
	// Try to find the action and extract the actual command
	action, exists := o.config.GetAction(stepName)
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
		
		// If all lines were empty, use 0
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
func (o *OrderedOutputManager) extractFailedDependency(stepName string) string {
	// Look through results to find failed dependencies
	for _, data := range o.stepData {
		if data.Status == StepStatusError {
			// This is a simple heuristic - in a real implementation,
			// we'd need to track the dependency graph
			return stepName
		}
	}
	return ""
}
