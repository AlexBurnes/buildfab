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
		
		// Handle case where first argument is a flag and second argument is a stage name
		if !isKnownCommand && strings.HasPrefix(args[0], "-") && len(args) > 1 && !strings.HasPrefix(args[1], "-") {
			// Check if second argument is a known command
			isSecondArgKnownCommand := false
			for _, cmd := range knownCommands {
				if args[1] == cmd {
					isSecondArgKnownCommand = true
					break
				}
			}
			
			// If second argument is not a known command, treat it as a stage name
			if !isSecondArgKnownCommand {
				// Insert "run" before the second argument
				newArgs := make([]string, 0, len(args)+1)
				newArgs = append(newArgs, args[0]) // Keep the flag
				newArgs = append(newArgs, "run")   // Insert "run"
				newArgs = append(newArgs, args[1:]...) // Add the rest
				os.Args = append([]string{os.Args[0]}, newArgs...)
			}
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

// CLIStepCallback implements StepCallback for CLI output with v0.5.0 style formatting
type CLIStepCallback struct {
	verbose bool
	debug   bool
	results []buildfab.StepResult
	displayed map[string]bool // Track which steps have been displayed
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

func (c *CLIStepCallback) OnStepStart(ctx context.Context, stepName string) {
	if c.verbose {
		fmt.Printf("  💻 %s\n", stepName)
	}
}

func (c *CLIStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration) {
	// Initialize displayed map if not already done
	if c.displayed == nil {
		c.displayed = make(map[string]bool)
	}
	
	// Only display each step once
	if !c.displayed[stepName] {
		var icon, color string
		switch status {
		case buildfab.StepStatusOK:
			icon = "✓"
			color = colorGreen
		case buildfab.StepStatusWarn:
			icon = "!"
			color = colorYellow
		case buildfab.StepStatusError:
			icon = "✗"
			color = colorRed
		case buildfab.StepStatusSkipped:
			icon = "→"
			color = colorGray
		default:
			icon = "?"
			color = colorGray
		}
		
		// Show step results
		fmt.Printf("  %s%s%s %s %s\n", color, icon, colorReset, stepName, message)
		c.displayed[stepName] = true
	}
	
	// Collect result for summary (avoid duplicates)
	// Check if we already have a result for this step
	found := false
	for i, result := range c.results {
		if result.StepName == stepName {
			// Update existing result
			c.results[i] = buildfab.StepResult{
				StepName: stepName,
				Status:   status,
				Duration: duration,
			}
			found = true
			break
		}
	}
	if !found {
		c.results = append(c.results, buildfab.StepResult{
			StepName: stepName,
			Status:   status,
			Duration: duration,
		})
	}
}

func (c *CLIStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
	if c.verbose && output != "" {
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		for _, line := range lines {
			fmt.Printf("    %s\n", line)
		}
	}
}

func (c *CLIStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
	// Don't display here - OnStepComplete should handle all display
	// This prevents duplicate display when both OnStepComplete and OnStepError are called
}

// GetResults returns the collected step results
func (c *CLIStepCallback) GetResults() []buildfab.StepResult {
	return c.results
}

// getSkippedSteps determines which steps should be skipped due to failed dependencies
func getSkippedSteps(cfg *buildfab.Config, stageName string, executedResults []buildfab.StepResult) []string {
	stage, exists := cfg.GetStage(stageName)
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
		if result.Status == buildfab.StepStatusError {
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

// printHeader prints the v0.5.0 style header
func printHeader(projectName, version string) {
	// Handle version that already has 'v' prefix
	versionDisplay := version
	if !strings.HasPrefix(version, "v") {
		versionDisplay = "v" + version
	}
	fmt.Printf("🚀 %s %s\n", projectName, versionDisplay)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📦 Project: %s\n", projectName)
	fmt.Printf("🏷️  Version: %s\n", versionDisplay)
	fmt.Printf("\n")
}

// printStageHeader prints the stage header
func printStageHeader(stageName string) {
	fmt.Printf("▶️  Running stage: %s\n", stageName)
	fmt.Printf("\n")
}

// printStageResult prints the stage result with summary
func printStageResult(stageName string, success bool, duration time.Duration, results []buildfab.StepResult) {
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
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
	
	fmt.Printf("%s %s%s%s - %s (%.2fs)\n", icon, color, status, colorReset, stageName, duration.Seconds())
	
	// Print summary
	if len(results) > 0 {
		fmt.Printf("\n")
		fmt.Printf("📊 Summary:\n")
		
		statusCounts := make(map[buildfab.StepStatus]int)
		for _, result := range results {
			statusCounts[result.Status]++
		}
		
		// Define status order for consistent display
		statusOrder := []buildfab.StepStatus{
			buildfab.StepStatusError,
			buildfab.StepStatusWarn,
			buildfab.StepStatusOK,
			buildfab.StepStatusSkipped,
		}
		
		for _, status := range statusOrder {
			count := statusCounts[status]
			var icon, color string
			
			switch status {
			case buildfab.StepStatusOK:
				icon = "✓"
				if count > 0 {
					color = colorGreen
				} else {
					color = colorGray
				}
			case buildfab.StepStatusWarn:
				icon = "!"
				if count > 0 {
					color = colorYellow
				} else {
					color = colorGray
				}
			case buildfab.StepStatusError:
				icon = "✗"
				if count > 0 {
					color = colorRed
				} else {
					color = colorGray
				}
			case buildfab.StepStatusSkipped:
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
			
			fmt.Printf("   %s%s%s %s%-8s %3d%s\n", color, icon, colorReset, color, status.String(), count, colorReset)
		}
	}
}

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
	
	// Print header
	printHeader(projectName, version)
	
	stageName := args[0]
	
	// Print stage header
	printStageHeader(stageName)
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Create step callback to collect results
	stepCallback := &CLIStepCallback{verbose: verbose, debug: debug}
	
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
		StepCallback: stepCallback,
	}
	
	// Create runner using library API
	runner := buildfab.NewRunner(cfg, opts)
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		start := time.Now()
		err := runner.RunStageStep(ctx, stageName, stepName)
		duration := time.Since(start)
		
		// Get collected results
		results := stepCallback.GetResults()
		if len(results) == 0 {
			// Fallback if no results collected
			if err != nil {
				results = []buildfab.StepResult{
					{StepName: stepName, Status: buildfab.StepStatusError, Duration: duration, Error: err},
				}
			} else {
				results = []buildfab.StepResult{
					{StepName: stepName, Status: buildfab.StepStatusOK, Duration: duration},
				}
			}
		}
		
		printStageResult(stageName, err == nil, duration, results)
		if err != nil {
			os.Exit(1)
		}
		return nil
	}
	
	// Run the entire stage using library API
	start := time.Now()
	err = runner.RunStage(ctx, stageName)
	duration := time.Since(start)
	
	// Get collected results from step callbacks
	results := stepCallback.GetResults()
	
	// Handle skipped steps that weren't executed due to dependencies
	// This is a workaround for the library not handling DAG execution properly
	skippedSteps := getSkippedSteps(cfg, stageName, results)
	for _, stepName := range skippedSteps {
		// Call step callbacks for skipped steps
		stepCallback.OnStepStart(ctx, stepName)
		stepCallback.OnStepComplete(ctx, stepName, buildfab.StepStatusSkipped, "skipped (dependency failed)", 0)
	}
	
	// Get updated results after handling skipped steps
	results = stepCallback.GetResults()
	
	printStageResult(stageName, err == nil, duration, results)
	if err != nil {
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
	
	// Create run options using library API
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
		StepCallback: &CLIStepCallback{verbose: verbose, debug: debug},
	}
	
	// Create runner using library API
	runner := buildfab.NewRunner(cfg, opts)
	
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