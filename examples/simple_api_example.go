package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
	// Example 1: Using SimpleRunner with full configuration
	fmt.Println("=== Example 1: SimpleRunner with full configuration ===")
	
	// Load configuration
	cfg, err := buildfab.LoadConfig(".project.yml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	
	// Create simple run options
	opts := &buildfab.SimpleRunOptions{
		ConfigPath: ".project.yml",
		VerboseLevel: 1,
		Output:     os.Stdout,
		ErrorOutput: os.Stderr,
	}
	
	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)
	
	// Run a stage - all output is handled automatically!
	ctx := context.Background()
	err = runner.RunStage(ctx, "pre-push")
	if err != nil {
		fmt.Printf("Stage failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 2: Using convenience functions ===")
	
	// Example 2: Using convenience functions for minimal setup
	err = buildfab.RunStageSimple(ctx, ".project.yml", "pre-push", 1)
	if err != nil {
		fmt.Printf("Stage failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 3: Running individual actions ===")
	
	// Example 3: Running individual actions
	err = runner.RunAction(ctx, "version@check")
	if err != nil {
		fmt.Printf("Action failed: %v\n", err)
	}
	
	fmt.Println("\n=== Example 4: Running specific steps ===")
	
	// Example 4: Running specific steps from a stage
	err = runner.RunStageStep(ctx, "pre-push", "version-module")
	if err != nil {
		fmt.Printf("Step failed: %v\n", err)
	}
	
	fmt.Println("\n=== All examples completed! ===")
	fmt.Println("Notice how simple the API is now - no callbacks, no complex setup!")
	fmt.Println("Just load config, create runner, and call RunStage/RunAction/RunStageStep!")
}