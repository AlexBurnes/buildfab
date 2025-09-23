package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
	"github.com/AlexBurnes/buildfab/internal/ui"
	"github.com/AlexBurnes/buildfab/internal/executor"
)

const (
	appName = "buildfab"
)

// appVersion is set at build time via ldflags
var appVersion = ""

// getVersion returns the version from build-time variable
func getVersion() string {
	// Use build-time variable only
	if appVersion != "" {
		return appVersion
	}
	
	// If not set at build time, return unknown
	return "unknown"
}

// Global flags
var (
	verbose       bool
	quiet         bool
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
	Use:   "buildfab [flags] [stage]",
	Short: "Buildfab CLI tool for project automation",
	Long: `buildfab is a Go-based runner for project automations defined in a YAML file.
It executes stages composed of steps (actions), supports parallel and sequential
execution via dependencies, and provides a library API for embedding.

When no command is specified, the first argument is treated as a stage name for the run command.
For example: buildfab pre-push is equivalent to buildfab run pre-push`,
	RunE: runRoot,
	// Disable automatic command suggestions to allow custom argument handling
	DisableSuggestions: true,
	// Allow unknown commands to be passed to RunE
	Args: cobra.ArbitraryArgs,
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

// listStagesCmd represents the list-stages command
var listStagesCmd = &cobra.Command{
	Use:   "list-stages",
	Short: "List defined stages in project configuration",
	Long:  `List all stages defined in the project configuration file.`,
	RunE:  runListStages,
}

// listStepsCmd represents the list-steps command
var listStepsCmd = &cobra.Command{
	Use:   "list-steps <stage>",
	Short: "List steps for a specific stage",
	Long:  `List all steps defined for a specific stage in the project configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runListSteps,
}

func main() {
	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", true, "enable verbose output (default)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable verbose output (silence mode)")
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
	rootCmd.AddCommand(listStagesCmd)
	rootCmd.AddCommand(listStepsCmd)
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
	
	// If no arguments, show help
	if len(args) == 0 {
		return cmd.Help()
	}
	
	// Load configuration to check if argument is a stage or action
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		// If config loading fails, treat as stage name (fallback behavior)
		return runStageDirect(cmd, args)
	}
	
	stageOrActionName := args[0]
	
	// Check if it's a stage name first (higher priority)
	if _, isStage := cfg.Stages[stageOrActionName]; isStage {
		// It's a stage, run it directly
		return runStageDirect(cmd, args)
	}
	
	// Check if it's an action name
	if _, isAction := cfg.GetAction(stageOrActionName); isAction {
		// It's an action, run it directly
		return runActionDirect(cmd, args)
	}
	
	// Check if it's a built-in action
	opts := buildfab.DefaultRunOptions()
	runner := buildfab.NewRunner(cfg, opts)
	builtinActions := runner.ListBuiltInActions()
	if _, isBuiltinAction := builtinActions[stageOrActionName]; isBuiltinAction {
		// It's a built-in action, run it directly
		return runActionDirect(cmd, args)
	}
	
	// If not found as stage or action, treat as stage name (fallback behavior)
	// This allows for dynamic stage names or better error messages from run command
	return runStageDirect(cmd, args)
}

// runStageDirect runs a stage directly without going through cobra command execution
func runStageDirect(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Get project info
	projectName := "buildfab" // Default project name
	if cfg.Project.Name != "" {
		projectName = cfg.Project.Name
	}
	version := getVersion()
	
	// Print header using library UI
	uiInstance := ui.New(verbose && !quiet, debug)
	uiInstance.PrintCLIHeader(projectName, version)
	uiInstance.PrintProjectCheck(projectName, version)
	
	stageName := args[0]
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create run options for internal executor
	// If quiet is set, override verbose to false
	effectiveVerbose := verbose && !quiet
	opts := &buildfab.RunOptions{
		ConfigPath:   configPath,
		MaxParallel:  maxParallel,
		Verbose:      effectiveVerbose,
		Debug:        debug,
		Variables:    variables,
		WorkingDir:   workingDir,
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
		Only:         only,
		WithRequires: withRequires,
	}
	
	// Create internal executor
	exec := executor.New(cfg, opts, uiInstance)
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		err := exec.RunAction(ctx, stepName)
		if err != nil {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			os.Exit(1)
		}
		return nil
	}
	
	// Run the entire stage using internal executor
	err = exec.RunStage(ctx, stageName)
	if err != nil {
		// In test mode, return the error instead of exiting
		if testing.Testing() {
			return err
		}
		os.Exit(1)
	}
	return nil
}

// runActionDirect runs an action directly without going through cobra command execution
func runActionDirect(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create simple run options
	// If quiet is set, override verbose to false
	effectiveVerbose := verbose && !quiet
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configPath,
		MaxParallel: maxParallel,
		Verbose:     effectiveVerbose,
		Debug:       debug,
		Variables:   variables,
		WorkingDir:  workingDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        only,
	}
	
	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)
	
	actionName := args[0]
	
	// Run action using simple API
	return runner.RunAction(ctx, actionName)
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// Custom output functions removed - now using library UI system

// runStage handles the run command
func runStage(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Get project info
	projectName := "buildfab" // Default project name
	if cfg.Project.Name != "" {
		projectName = cfg.Project.Name
	}
	version := getVersion()
	
	// Print header using library UI
	uiInstance := ui.New(verbose && !quiet, debug)
	uiInstance.PrintCLIHeader(projectName, version)
	uiInstance.PrintProjectCheck(projectName, version)
	
	stageName := args[0]
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create run options for internal executor
	// If quiet is set, override verbose to false
	effectiveVerbose := verbose && !quiet
	opts := &buildfab.RunOptions{
		ConfigPath:   configPath,
		MaxParallel:  maxParallel,
		Verbose:      effectiveVerbose,
		Debug:        debug,
		Variables:    variables,
		WorkingDir:   workingDir,
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
		Only:         only,
		WithRequires: withRequires,
	}
	
	// Reuse the UI instance created above
	
	// Create internal executor
	exec := executor.New(cfg, opts, uiInstance)
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		err := exec.RunAction(ctx, stepName)
		if err != nil {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			os.Exit(1)
		}
		return nil
	}
	
	// Run the entire stage using internal executor
	err = exec.RunStage(ctx, stageName)
	if err != nil {
		// In test mode, return the error instead of exiting
		if testing.Testing() {
			return err
		}
		os.Exit(1)
	}
	return nil
}

// runAction handles the action command
func runAction(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create simple run options
	// If quiet is set, override verbose to false
	effectiveVerbose := verbose && !quiet
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configPath,
		MaxParallel: maxParallel,
		Verbose:     effectiveVerbose,
		Debug:       debug,
		Variables:   variables,
		WorkingDir:  workingDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        only,
	}
	
	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)
	
	actionName := args[0]
	
	// Run action using simple API
	return runner.RunAction(ctx, actionName)
}

// runListActions handles the list-actions command
func runListActions(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Get built-in actions using library API
	opts := buildfab.DefaultRunOptions()
	runner := buildfab.NewRunner(cfg, opts)
	builtinActions := runner.ListBuiltInActions()
	
	fmt.Println("Available actions:")
	fmt.Println()
	
	// Show defined actions from configuration
	if len(cfg.Actions) > 0 {
		fmt.Println("Defined actions in project configuration:")
		for _, action := range cfg.Actions {
			description := "Custom action"
			if action.Uses != "" {
				description = fmt.Sprintf("Uses: %s", action.Uses)
			} else if action.Run != "" {
				description = "Custom run command"
			}
			fmt.Printf("  %-20s %s\n", action.Name, description)
		}
		fmt.Println()
	}
	
	// Show built-in actions
	fmt.Println("Built-in actions:")
	for name, description := range builtinActions {
		fmt.Printf("  %-20s %s\n", name, description)
	}
	
	return nil
}

// runValidate handles the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	fmt.Printf("Configuration is valid: %s\n", configPath)
	fmt.Printf("Project: %s\n", cfg.Project.Name)
	fmt.Printf("Actions: %d\n", len(cfg.Actions))
	fmt.Printf("Stages: %d\n", len(cfg.Stages))
	
	return nil
}

// runListStages handles the list-stages command
func runListStages(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	fmt.Println("Defined stages in project configuration:")
	fmt.Println()
	
	if len(cfg.Stages) == 0 {
		fmt.Println("  No stages defined")
		return nil
	}
	
	for name, stage := range cfg.Stages {
		stepCount := len(stage.Steps)
		description := fmt.Sprintf("%d step(s)", stepCount)
		fmt.Printf("  %-20s %s\n", name, description)
	}
	
	return nil
}

// runListSteps handles the list-steps command
func runListSteps(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	stageName := args[0]
	
	// Find the stage
	stage, exists := cfg.Stages[stageName]
	if !exists {
		return fmt.Errorf("stage '%s' not found in configuration", stageName)
	}
	
	fmt.Printf("Steps for stage '%s':\n", stageName)
	fmt.Println()
	
	if len(stage.Steps) == 0 {
		fmt.Println("  No steps defined")
		return nil
	}
	
	for i, step := range stage.Steps {
		description := step.Action
		if step.If != "" {
			description += fmt.Sprintf(" (if: %s)", step.If)
		}
		if len(step.Only) > 0 {
			description += fmt.Sprintf(" (only: %s)", strings.Join(step.Only, ","))
		}
		fmt.Printf("  %-3d %-20s %s\n", i+1, step.Action, description)
	}
	
	return nil
}