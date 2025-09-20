package buildfab

import (
	"context"
	"fmt"
	"os"
)

// RunCLI executes the buildfab CLI with the given arguments
func RunCLI(ctx context.Context, args []string) error {
	// TODO: Implement CLI parsing and execution
	// This is a placeholder implementation
	fmt.Fprintf(os.Stderr, "buildfab CLI not yet implemented\n")
	fmt.Fprintf(os.Stderr, "Arguments: %v\n", args)
	return fmt.Errorf("not implemented")
}

// RunStage executes a specific stage from project.yml configuration
func RunStage(ctx context.Context, stageName string, opts *RunOptions) error {
	// TODO: Implement stage execution
	return fmt.Errorf("not implemented")
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
}