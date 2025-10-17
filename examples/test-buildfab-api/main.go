package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file> [stage-name]\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	stageName := "test"
	if len(os.Args) >= 3 {
		stageName = os.Args[2]
	}

	// Load configuration
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create simple run options (similar to pre-push)
	verboseLevel := 0 // Start with 0 to see fast failure
	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &verboseLevel)
	}
	
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:         configPath,
		VerboseLevel:       verboseLevel,
		Debug:              verboseLevel > 0,
		WorkingDir:         ".",
		Output:             os.Stdout,
		ErrorOutput:        os.Stderr,
		BuildfabBinaryPath: "../../bin/buildfab",  // Explicit path (test-api doesn't implement CLI commands)
		// Note: For apps that implement full buildfab CLI interface (like pre-push),
		// BuildfabBinaryPath can be omitted and current executable will be used automatically
	}

	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)

	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Printf("Running stage '%s' using SimpleRunner API...\n", stageName)
	fmt.Printf("Config: %s\n", configPath)
	fmt.Printf("Verbose Level: %d\n", opts.VerboseLevel)
	fmt.Println()

	// Run the stage
	err = runner.RunStage(ctx, stageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nStage failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nStage completed successfully!")
}

