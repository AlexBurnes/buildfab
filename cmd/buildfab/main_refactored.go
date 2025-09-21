package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
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
	Use:   "buildfab [flags] [command]",
	Short: "Buildfab CLI tool for project automation",
	Long: `buildfab is a Go-based runner for project automations defined in a YAML file.
It executes stages composed of steps (actions), supports parallel and sequential
execution via dependencies, and provides a library API for embedding.

When no command is specified, the first argument is treated as a stage name for the run command.
For example: buildfab pre-push is equivalent to buildfab run pre-push`,
	RunE: runRoot,
	// Disable automatic command suggestions to allow custom argument handling
	DisableSuggestions: true,
	// Allow unknown commands to be handled by runRoot
	DisableFlagParsing: false,
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
	rootCmd.AddCommand(listStagesCmd)
	rootCmd.AddCommand(listStepsCmd)
	rootCmd.AddCommand(validateCmd)
	
	// Check if first argument is a known subcommand
	args := os.Args[1:]
	if len(args) > 0 {
		// Check if first argument is a known subcommand
		knownCommands := []string{"run", "action", "list-actions", "list-stages", "list-steps", "validate", "completion", "help"}
		isKnownCommand := false
		for _, cmd := range knownCommands {
			if args[0] == cmd {
				isKnownCommand = true
				break
			}
		}
		
		// If not a known command and not a flag, treat as stage name
		if !isKnownCommand && !strings.HasPrefix(args[0], "-") {
			// Insert "run" as the first argument
			args = append([]string{"run"}, args...)
			os.Args = append([]string{os.Args[0]}, args...)
		}
	}
	
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
	
	// If arguments provided, treat first argument as stage name for run command
	// This implements the default behavior: buildfab pre-push -> buildfab run pre-push
	return runStage(cmd, args)
}

// CLIStepCallback implements StepCallback for CLI output
type CLIStepCallback struct {
	verbose bool
	debug   bool
}

func (c *CLIStepCallback) OnStepStart(ctx context.Context, stepName string) {
	if c.verbose {
		fmt.Printf("🔄 Running step: %s\n", stepName)
	}
}

func (c *CLIStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration) {
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
	
	if c.verbose || status == buildfab.StepStatusError {
		fmt.Printf("%s %s: %s (%v)\n", icon, stepName, message, duration)
	}
}

func (c *CLIStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	if c.verbose && output != "" {
		fmt.Printf("📤 %s output:\n%s\n", stepName, output)
	}
}

func (c *CLIStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	fmt.Printf("❌ %s failed: %v\n", stepName, err)
}

// createRunner creates a buildfab runner with proper configuration
func createRunner() (*buildfab.Runner, error) {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create run options using library API
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
		StepCallback: &CLIStepCallback{verbose: verbose, debug: debug},
	}
	
	// Create runner using library API
	runner := buildfab.NewRunner(cfg, opts)
	return runner, nil
}

// runStage handles the run command
func runStage(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Create runner using library API
	runner, err := createRunner()
	if err != nil {
		return err
	}
	
	stageName := args[0]
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		return runner.RunStageStep(ctx, stageName, stepName)
	}
	
	// Run the entire stage using library API
	return runner.RunStage(ctx, stageName)
}

// runAction handles the action command
func runAction(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Create runner using library API
	runner, err := createRunner()
	if err != nil {
		return err
	}
	
	actionName := args[0]
	
	// Run action using library API
	return runner.RunAction(ctx, actionName)
}

// runListActions handles the list-actions command
func runListActions(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Create runner to get built-in actions using library API
	opts := buildfab.DefaultRunOptions()
	runner := buildfab.NewRunner(cfg, opts)
	builtinActions := runner.ListBuiltInActions()
	
	fmt.Println("Available actions:")
	fmt.Println()
	
	// Show defined actions from configuration using library API
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
	
	// Show built-in actions using library API
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
	
	// Validate using library API
	if err := cfg.Validate(); err != nil {
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
	
	// Find the stage using library API
	stage, exists := cfg.GetStage(stageName)
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