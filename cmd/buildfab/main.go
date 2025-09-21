package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/AlexBurnes/buildfab/internal/config"
	"github.com/AlexBurnes/buildfab/internal/executor"
	"github.com/AlexBurnes/buildfab/internal/ui"
	"github.com/AlexBurnes/buildfab/internal/version"
	"github.com/AlexBurnes/buildfab/internal/actions"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

const (
	appName = "buildfab"
)

// getVersion reads the version from the VERSION file
func getVersion() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "unknown"
	}
	
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "unknown"
	}
	
	return version
}

// Global flags
var (
	verbose       bool
	debug         bool
	configPath    string
	maxParallel   int
	workingDir    string
	only          []string
	withRequires  bool
	envVars       []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "buildfab",
	Short: "Buildfab CLI tool for project automation",
	Long: `buildfab is a Go-based runner for project automations defined in a YAML file.
It executes stages composed of steps (actions), supports parallel and sequential
execution via dependencies, and provides a library API for embedding.`,
	RunE: runRoot,
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <stage> [step]",
	Short: "Run a stage or specific step",
	Long: `Run a stage or specific step from the project configuration.
If a step is specified, only that step will be run (with dependencies if --with-requires is set).`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runStage,
}

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action <action>",
	Short: "Run a standalone action",
	Long:  `Run a standalone action directly without stage context.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAction,
}

// listActionsCmd represents the list-actions command
var listActionsCmd = &cobra.Command{
	Use:   "list-actions",
	Short: "List available built-in actions",
	Long:  `List all available built-in actions that can be used in the 'uses' field.`,
	RunE:  runListActions,
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate project configuration",
	Long:  `Validate the project.yml configuration file for errors.`,
	RunE:  runValidate,
}

func main() {
	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug output")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", ".project.yml", "path to configuration file")
	rootCmd.PersistentFlags().IntVar(&maxParallel, "max-parallel", 0, "maximum parallel execution (default: CPU count)")
	rootCmd.PersistentFlags().StringVarP(&workingDir, "working-dir", "w", ".", "working directory for execution")
	rootCmd.PersistentFlags().StringSliceVar(&only, "only", []string{}, "only run steps matching these labels")
	rootCmd.PersistentFlags().BoolVar(&withRequires, "with-requires", false, "include required dependencies when running single step")
	rootCmd.PersistentFlags().StringSliceVar(&envVars, "env", []string{}, "export environment variables to actions")
	
	// Add version flags
	rootCmd.Flags().BoolP("version", "", false, "print version and module name")
	rootCmd.Flags().BoolP("version-only", "V", false, "print version only")
	
	// Add subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(actionCmd)
	rootCmd.AddCommand(listActionsCmd)
	rootCmd.AddCommand(validateCmd)
	
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runRoot handles the root command
func runRoot(cmd *cobra.Command, args []string) error {
	// Check if version flags were set
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Printf("%s version %s\n", appName, getVersion())
		return nil
	}
	if versionOnlyFlag, _ := cmd.Flags().GetBool("version-only"); versionOnlyFlag {
		fmt.Printf("%s\n", getVersion())
		return nil
	}
	
	// Show help if no arguments
	return cmd.Help()
}

// runStage handles the run command
func runStage(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Detect Git variables
	gitVars, err := config.DetectGitVariables(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect Git variables: %w", err)
	}
	
	// Detect version variables
	versionDetector := version.New()
	versionVars, err := versionDetector.GetVersionVariables(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect version variables: %w", err)
	}
	
	// Merge variables
	variables := make(map[string]string)
	for k, v := range gitVars {
		variables[k] = v
	}
	for k, v := range versionVars {
		variables[k] = v
	}
	
	// Add environment variables
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Resolve variables in configuration
	if err := config.ResolveVariables(cfg, variables); err != nil {
		return fmt.Errorf("failed to resolve variables: %w", err)
	}
	
	// Create run options
	opts := &buildfab.RunOptions{
		ConfigPath:   configPath,
		MaxParallel:  maxParallel,
		Verbose:      verbose,
		Debug:        debug,
		Variables:    variables,
		WorkingDir:   workingDir,
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
		Only:         only,
		WithRequires: withRequires,
	}
	
	// Create UI
	ui := ui.New(verbose, debug)
	opts.Output = ui
	
	// Create executor
	exec := executor.New(cfg, opts, ui)
	
	stageName := args[0]
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		return exec.RunStageStep(ctx, stageName, stepName)
	}
	
	// Run the entire stage
	return exec.RunStage(ctx, stageName)
}

// runAction handles the action command
func runAction(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Detect Git variables
	gitVars, err := config.DetectGitVariables(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect Git variables: %w", err)
	}
	
	// Detect version variables
	versionDetector := version.New()
	versionVars, err := versionDetector.GetVersionVariables(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect version variables: %w", err)
	}
	
	// Merge variables
	variables := make(map[string]string)
	for k, v := range gitVars {
		variables[k] = v
	}
	for k, v := range versionVars {
		variables[k] = v
	}
	
	// Add environment variables
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Resolve variables in configuration
	if err := config.ResolveVariables(cfg, variables); err != nil {
		return fmt.Errorf("failed to resolve variables: %w", err)
	}
	
	// Create run options
	opts := &buildfab.RunOptions{
		ConfigPath:  configPath,
		MaxParallel: maxParallel,
		Verbose:     verbose,
		Debug:       debug,
		Variables:   variables,
		WorkingDir:  workingDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        only,
	}
	
	// Create UI
	ui := ui.New(verbose, debug)
	opts.Output = ui
	
	// Create executor
	exec := executor.New(cfg, opts, ui)
	
	actionName := args[0]
	
	// Check if it's a built-in action first
	registry := actions.New()
	if runner, exists := registry.GetRunner(actionName); exists {
		// Execute built-in action directly
		result, err := runner.Run(ctx)
		ui.PrintStepStatus(actionName, result.Status, result.Message)
		if err != nil {
			return err
		}
		return nil
	}
	
	// Otherwise, try to run as configuration action
	return exec.RunAction(ctx, actionName)
}

// runListActions handles the list-actions command
func runListActions(cmd *cobra.Command, args []string) error {
	registry := actions.New()
	actions := registry.ListActions()
	
	fmt.Println("Available built-in actions:")
	fmt.Println()
	
	for name, description := range actions {
		fmt.Printf("  %-20s %s\n", name, description)
	}
	
	return nil
}

// runValidate handles the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	fmt.Printf("Configuration is valid: %s\n", configPath)
	fmt.Printf("Project: %s\n", cfg.Project.Name)
	fmt.Printf("Actions: %d\n", len(cfg.Actions))
	fmt.Printf("Stages: %d\n", len(cfg.Stages))
	
	return nil
}