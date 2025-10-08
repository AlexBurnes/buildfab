package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// ExampleStepCallback demonstrates how to implement step callbacks
type ExampleStepCallback struct {
	verbose bool
}

// OnStepStart is called when a step starts execution
func (c *ExampleStepCallback) OnStepStart(ctx context.Context, stepName string) {
	fmt.Printf("🔄 Running step: %s\n", stepName)
}

// OnStepComplete is called when a step completes (success, warning, or error)
func (c *ExampleStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration, bufferedOutput string) {
	var icon string
	switch status {
	case buildfab.StepStatusOK:
		icon = "✔"
	case buildfab.StepStatusWarn:
		icon = "⚠"
	case buildfab.StepStatusError:
		icon = "✖"
	case buildfab.StepStatusSkipped:
		icon = "○"
	default:
		icon = "?"
	}
	
	fmt.Printf("%s %s: %s (%v)\n", icon, stepName, message, duration)
}

// OnStepOutput is called for step output (when verbose mode is enabled)
func (c *ExampleStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	if output != "" {
		fmt.Printf("📤 %s output:\n%s\n", stepName, output)
	}
}

// OnStepError is called for step errors
func (c *ExampleStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	fmt.Printf("❌ %s failed: %v\n", stepName, err)
}

// GetResults returns the collected step results
func (c *ExampleStepCallback) GetResults() []buildfab.StepResult {
	return nil // ExampleStepCallback doesn't collect results
}

// VerboseStepCallback provides more detailed output
type VerboseStepCallback struct {
	startTime time.Time
}

// OnStepStart is called when a step starts execution
func (c *VerboseStepCallback) OnStepStart(ctx context.Context, stepName string) {
	c.startTime = time.Now()
	fmt.Printf("[%s] 🔄 Starting step: %s\n", time.Now().Format("15:04:05"), stepName)
}

// OnStepComplete is called when a step completes
func (c *VerboseStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration, bufferedOutput string) {
	timestamp := time.Now().Format("15:04:05")
	var icon string
	var color string
	
	switch status {
	case buildfab.StepStatusOK:
		icon = "✔"
		color = "\033[32m" // Green
	case buildfab.StepStatusWarn:
		icon = "⚠"
		color = "\033[33m" // Yellow
	case buildfab.StepStatusError:
		icon = "✖"
		color = "\033[31m" // Red
	case buildfab.StepStatusSkipped:
		icon = "○"
		color = "\033[90m" // Gray
	default:
		icon = "?"
		color = "\033[37m" // White
	}
	
	reset := "\033[0m"
	fmt.Printf("[%s] %s%s %s: %s%s (%v)\n", timestamp, color, icon, stepName, message, reset, duration)
}

// OnStepOutput is called for step output
func (c *VerboseStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	if output != "" {
		fmt.Printf("[%s] 📤 %s output:\n%s\n", time.Now().Format("15:04:05"), stepName, output)
	}
}

// OnStepError is called for step errors
func (c *VerboseStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	fmt.Printf("[%s] ❌ %s failed: %v\n", time.Now().Format("15:04:05"), stepName, err)
}

// GetResults returns the collected step results
func (c *VerboseStepCallback) GetResults() []buildfab.StepResult {
	return nil // VerboseStepCallback doesn't collect results
}

// SilentStepCallback provides minimal output
type SilentStepCallback struct{}

// OnStepStart is called when a step starts execution
func (c *SilentStepCallback) OnStepStart(ctx context.Context, stepName string) {
	// Silent - no output
}

// OnStepComplete is called when a step completes
func (c *SilentStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration, bufferedOutput string) {
	// Only show errors
	if status == buildfab.StepStatusError {
		fmt.Printf("Error in %s: %s\n", stepName, message)
	}
}

// OnStepOutput is called for step output
func (c *SilentStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	// Silent - no output
}

// OnStepError is called for step errors
func (c *SilentStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	fmt.Printf("Error in %s: %v\n", stepName, err)
}

// GetResults returns the collected step results
func (c *SilentStepCallback) GetResults() []buildfab.StepResult {
	return nil // SilentStepCallback doesn't collect results
}

func mainStepCallbacks() {
	// Example 1: Basic usage with step callbacks
	fmt.Println("=== Example 1: Basic Step Callbacks ===")
	
	// Load configuration
	config, err := buildfab.LoadConfig(".project.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	
	// Create run options with step callback
	opts := buildfab.DefaultRunOptions()
	opts.StepCallback = &ExampleStepCallback{verbose: true}
	opts.VerboseLevel = 1
	
	// Create runner
	runner := buildfab.NewRunner(config, opts)
	
	// Run a stage with step callbacks
	ctx := context.Background()
	err = runner.RunStage(ctx, "pre-push")
	if err != nil {
		fmt.Printf("Stage execution failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 2: Verbose Step Callbacks ===")
	
	// Example 2: Verbose callbacks with timestamps
	opts2 := buildfab.DefaultRunOptions()
	opts2.StepCallback = &VerboseStepCallback{}
	opts2.VerboseLevel = 1
	
	runner2 := buildfab.NewRunner(config, opts2)
	err = runner2.RunStage(ctx, "pre-push")
	if err != nil {
		fmt.Printf("Stage execution failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 3: Silent Step Callbacks ===")
	
	// Example 3: Silent callbacks (only show errors)
	opts3 := buildfab.DefaultRunOptions()
	opts3.StepCallback = &SilentStepCallback{}
	
	runner3 := buildfab.NewRunner(config, opts3)
	err = runner3.RunStage(ctx, "pre-push")
	if err != nil {
		fmt.Printf("Stage execution failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 4: No Callbacks (Default Behavior) ===")
	
	// Example 4: No callbacks - default behavior
	opts4 := buildfab.DefaultRunOptions()
	opts4.VerboseLevel = 1
	
	runner4 := buildfab.NewRunner(config, opts4)
	err = runner4.RunStage(ctx, "pre-push")
	if err != nil {
		fmt.Printf("Stage execution failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 5: Custom Action with Callbacks ===")
	
	// Example 5: Running individual actions with callbacks
	opts5 := buildfab.DefaultRunOptions()
	opts5.StepCallback = &ExampleStepCallback{verbose: true}
	opts5.VerboseLevel = 1
	
	runner5 := buildfab.NewRunner(config, opts5)
	
	// Run individual actions
	actions := []string{"version-check", "git-untracked", "git-uncommitted"}
	for _, actionName := range actions {
		fmt.Printf("\n--- Running action: %s ---\n", actionName)
		err = runner5.RunAction(ctx, actionName)
		if err != nil {
			fmt.Printf("Action %s failed: %v\n", actionName, err)
		}
	}
}

// Uncomment the main function below to run this example
// func main() {
// 	mainStepCallbacks()
// }