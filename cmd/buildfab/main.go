package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
	versiongo "github.com/AlexBurnes/version-go/pkg/version"
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

// getProjectVersion returns the project version using the version-go library
func getProjectVersion() string {
	// Use version-go library GetVersion() to get version WITHOUT 'v' prefix
	// This is correct for most use cases like Docker tags: buildfab:0.21.1
	if ver, err := versiongo.GetVersion(); err == nil && ver != "" {
		return ver
	}
	
	// Fallback to VERSION file if version library fails
	// Remove 'v' prefix if present to match GetVersion() behavior
	if data, err := os.ReadFile("VERSION"); err == nil {
		ver := strings.TrimSpace(string(data))
		if ver != "" {
			// Remove 'v' prefix if present
			return strings.TrimPrefix(ver, "v")
		}
	}
	
	// Final fallback
	return "unknown"
}

// displayVersionInfo displays buildfab version and project version information
func displayVersionInfo(cfg *buildfab.Config) {
	buildfabVersion := getVersion()
	projectVersion := getProjectVersion()
	
	// Display version information
	fmt.Printf("%s %s\n", appName, buildfabVersion)
	if cfg != nil {
		fmt.Printf("Project %s (%s)\n", cfg.Project.Name, projectVersion)
	}
}

// Global flags
var (
	verboseLevel  int
	quiet         bool
	debug         bool
	dryRun        bool
	configPath    string
	maxParallel   int
	workingDir    string
	only          []string
	withRequires  bool
	envVars       []string
	showGraph     bool
	matrixVars    map[string]string // Dynamic matrix variables from CLI flags
	versionFlag   bool              // Version flag
	versionOnlyFlag bool            // Version-only flag
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
	// Disable flag parsing to allow custom matrix flag handling
	DisableFlagParsing: true,
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

// listVariablesCmd represents the list-variables command
var listVariablesCmd = &cobra.Command{
	Use:   "list-variables",
	Short: "List all available variables with their values",
	Long:  `List all available variables that can be used in configuration, sorted by name.`,
	RunE:  runListVariables,
}

func init() {
	listStepsCmd.Flags().BoolVarP(&showGraph, "graph", "g", false, "show steps as a dependency graph")
}

// parseFlags parses both regular flags and matrix flags from command line arguments
func parseFlags(args []string) ([]string, error) {
	var remainingArgs []string
	i := 0
	
	for i < len(args) {
		arg := args[i]
		
		// Handle matrix flags
		if strings.HasPrefix(arg, "--matrix.") {
			// Extract matrix key and value
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				matrixKey := strings.TrimPrefix(parts[0], "--matrix.")
				matrixVars[matrixKey] = parts[1]
			} else {
				// Handle case where value might be in next argument
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
					matrixKey := strings.TrimPrefix(arg, "--matrix.")
					matrixVars[matrixKey] = args[i+1]
					i++ // Skip the value argument
				}
			}
		} else if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			// Handle regular flags
			if err := handleRegularFlag(arg, &i, args); err != nil {
				return nil, err
			}
		} else {
			// Non-flag argument
			remainingArgs = append(remainingArgs, arg)
		}
		i++
	}
	
	return remainingArgs, nil
}

// handleRegularFlag handles regular CLI flags
func handleRegularFlag(arg string, i *int, args []string) error {
	switch arg {
	case "--help", "-h":
		// Help flag is handled by cobra automatically, just return nil
		return nil
	case "--version":
		versionFlag = true
	case "--version-only", "-V":
		versionOnlyFlag = true
	case "--verbose":
		verboseLevel++
	case "-v":
		verboseLevel++
	case "-vv":
		verboseLevel += 2
	case "-vvv":
		verboseLevel += 3
	case "--quiet", "-q":
		quiet = true
	case "--debug", "-d":
		debug = true
	case "--dry-run":
		dryRun = true
	case "--config", "-c":
		if *i+1 < len(args) {
			*i++
			configPath = args[*i]
		} else {
			return fmt.Errorf("flag --config requires a value")
		}
	case "--max-parallel":
		if *i+1 < len(args) {
			*i++
			if val, err := strconv.Atoi(args[*i]); err == nil {
				maxParallel = val
			} else {
				return fmt.Errorf("invalid value for --max-parallel: %s", args[*i])
			}
		} else {
			return fmt.Errorf("flag --max-parallel requires a value")
		}
	case "--working-dir", "-w":
		if *i+1 < len(args) {
			*i++
			workingDir = args[*i]
		} else {
			return fmt.Errorf("flag --working-dir requires a value")
		}
	case "--only":
		if *i+1 < len(args) {
			*i++
			only = append(only, args[*i])
		} else {
			return fmt.Errorf("flag --only requires a value")
		}
	case "--with-requires":
		withRequires = true
	case "--env":
		if *i+1 < len(args) {
			*i++
			envVars = append(envVars, args[*i])
		} else {
			return fmt.Errorf("flag --env requires a value")
		}
	default:
		// Unknown flag - let Cobra handle it or ignore it
		if !strings.HasPrefix(arg, "--matrix.") {
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}
	return nil
}

// parseMatrixFlags parses matrix flags from command line arguments (legacy function)
func parseMatrixFlags(args []string) {
	// This function is now handled by parseFlags
}

func main() {
	// Initialize matrix variables map
	matrixVars = make(map[string]string)
	
	// Add global flags
	rootCmd.PersistentFlags().CountVarP(&verboseLevel, "verbose", "v", "increase verbosity level (-v, -vv, -vvv)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable verbose output (silence mode)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be executed without running commands")
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
	rootCmd.AddCommand(listVariablesCmd)
	rootCmd.AddCommand(validateCmd)
	
	// Parse matrix flags before executing
	parseMatrixFlags(os.Args[1:])
	
	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runRoot handles the root command
func runRoot(cmd *cobra.Command, args []string) error {
	// Parse flags manually since we disabled automatic flag parsing
	remainingArgs, err := parseFlags(args)
	if err != nil {
		return err
	}
	
	// Check if version flags were set
	if versionFlag {
		fmt.Printf("%s version %s\n", appName, getVersion())
		return nil
	}
	if versionOnlyFlag {
		fmt.Printf("%s\n", getVersion())
		return nil
	}
	
	// If no arguments, show help
	if len(remainingArgs) == 0 {
		return cmd.Help()
	}
	
	// Load configuration to check if argument is a stage or action
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		// Check if it's a validation error or YAML parsing error
		if strings.Contains(err.Error(), "failed to parse YAML") ||
		   strings.Contains(err.Error(), "field") && strings.Contains(err.Error(), "not found") ||
		   strings.Contains(err.Error(), "step") && strings.Contains(err.Error(), "must have an action") ||
		   strings.Contains(err.Error(), "duplicate action name") ||
		   strings.Contains(err.Error(), "stage") && strings.Contains(err.Error(), "must have at least one step") {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			// Print error with line number and exit directly to avoid duplication with cobra error handling
			enhancedError := enhanceValidationError(configPath, err)
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", enhancedError)
			os.Exit(1)
		}
		// If config loading fails, treat as stage name (fallback behavior)
		return runStageDirect(cmd, remainingArgs)
	}
	
	stageOrActionName := remainingArgs[0]
	
	// Check if it's a stage name first (higher priority)
	if _, isStage := cfg.Stages[stageOrActionName]; isStage {
	// It's a stage, run it directly
	return runStageDirect(cmd, remainingArgs)
	}
	
	// Check if it's an action name
	if _, isAction := cfg.GetAction(stageOrActionName); isAction {
		// It's an action, run it directly
		return runActionDirect(cmd, remainingArgs)
	}
	
	// Check if it's a built-in action
	opts := buildfab.DefaultRunOptions()
	runner := buildfab.NewRunner(cfg, opts)
	builtinActions := runner.ListBuiltInActions()
	if _, isBuiltinAction := builtinActions[stageOrActionName]; isBuiltinAction {
		// It's a built-in action, run it directly
		return runActionDirect(cmd, remainingArgs)
	}
	
	// If not found as stage or action, treat as stage name (fallback behavior)
	// This allows for dynamic stage names or better error messages from run command
	return runStageDirect(cmd, remainingArgs)
}

// runStageDirect runs a stage directly without going through cobra command execution
func runStageDirect(cmd *cobra.Command, args []string) error {
	// Create context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		// Check if it's a validation error or YAML parsing error
		if strings.Contains(err.Error(), "failed to parse YAML") ||
		   strings.Contains(err.Error(), "field") && strings.Contains(err.Error(), "not found") ||
		   strings.Contains(err.Error(), "step") && strings.Contains(err.Error(), "must have an action") ||
		   strings.Contains(err.Error(), "duplicate action name") ||
		   strings.Contains(err.Error(), "stage") && strings.Contains(err.Error(), "must have at least one step") {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			// Print colored error with line number and exit directly
			enhancedError := enhanceValidationError(configPath, err)
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", enhancedError)
			os.Exit(1)
		}
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	stageName := args[0]
	
	// Display version information before running stage
	displayVersionInfo(cfg)
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Add matrix variables from CLI flags
	for key, value := range matrixVars {
		variables[fmt.Sprintf("matrix.%s", key)] = value
	}
	
	// Add platform variables
	variables = buildfab.AddPlatformVariables(variables)
	
	// Add version variables
	variables = buildfab.AddVersionVariables(variables)
	
	// Add direct project and version variables for convenience
	if cfg != nil {
		variables["project"] = cfg.Project.Name
		// Add first module as the primary module
		if len(cfg.Project.Modules) > 0 {
			variables["module"] = cfg.Project.Modules[0]
		}
	}
	// Add direct version variable from VERSION file
	if versionStr := getProjectVersion(); versionStr != "unknown" {
		variables["version"] = versionStr
	}
	
	// If quiet is set, override verbose level to 0
	// Otherwise, default to verbose level 1 if no verbose flags were provided
	effectiveVerboseLevel := verboseLevel
	if quiet {
		effectiveVerboseLevel = 0
	} else if verboseLevel == 0 {
		// Default to verbose level 1 if no -v flags were provided
		effectiveVerboseLevel = 1
	}
	
	// Create simple run options
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configPath,
		MaxParallel: maxParallel,
		VerboseLevel: effectiveVerboseLevel,
		Debug:       debug,
		DryRun:      dryRun,
		Variables:   variables,
		WorkingDir:  workingDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        only,
		WithRequires: withRequires,
	}
	
	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)
	
	// Check if running a specific step
	if len(args) == 2 {
		stepName := args[1]
		err := runner.RunStageStep(ctx, stageName, stepName)
		if err != nil {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			// Only show usage hints for stage/step not found errors, not for execution errors
			if strings.Contains(err.Error(), "stage not found") || strings.Contains(err.Error(), "step not found") {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "To see available stages run: buildfab list-stages\n")
				fmt.Fprintf(os.Stderr, "To see available actions run: buildfab list-actions\n")
			} else {
				// For execution errors, just show the error without usage hints
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			os.Exit(1)
		}
		return nil
	}
	
	// Run the entire stage using simple runner
	err = runner.RunStage(ctx, stageName)
	if err != nil {
		// In test mode, return the error instead of exiting
		if testing.Testing() {
			return err
		}
		// Only show usage hints for stage not found errors, not for execution errors
		if strings.Contains(err.Error(), "stage not found") {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "To see available stages run: buildfab list-stages\n")
			fmt.Fprintf(os.Stderr, "To see available actions run: buildfab list-actions\n")
		} else {
			// For execution errors, SimpleRunner already handled the output via step callbacks
			// Just exit with error code without printing the error again
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
		// Check if it's a validation error
		if strings.Contains(err.Error(), "step") && strings.Contains(err.Error(), "must have an action") ||
		   strings.Contains(err.Error(), "duplicate action name") ||
		   strings.Contains(err.Error(), "stage") && strings.Contains(err.Error(), "must have at least one step") {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			// Print colored error with line number and exit directly
			enhancedError := enhanceValidationError(configPath, err)
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", enhancedError)
			os.Exit(1)
		}
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
	
	// Add matrix variables from CLI flags
	for key, value := range matrixVars {
		variables[fmt.Sprintf("matrix.%s", key)] = value
	}
	
	// Add platform variables
	variables = buildfab.AddPlatformVariables(variables)
	
	// Add version variables
	variables = buildfab.AddVersionVariables(variables)
	
	// Add direct project and version variables for convenience
	if cfg != nil {
		variables["project"] = cfg.Project.Name
		// Add first module as the primary module
		if len(cfg.Project.Modules) > 0 {
			variables["module"] = cfg.Project.Modules[0]
		}
	}
	// Add direct version variable from VERSION file
	if versionStr := getProjectVersion(); versionStr != "unknown" {
		variables["version"] = versionStr
	}
	
	// Create simple run options
	// If quiet is set, override verbose level to 0
	// Otherwise, default to verbose level 1 if no verbose flags were provided
	effectiveVerboseLevel := verboseLevel
	if quiet {
		effectiveVerboseLevel = 0
	} else if verboseLevel == 0 {
		// Default to verbose level 1 if no -v flags were provided
		effectiveVerboseLevel = 1
	}
	opts := &buildfab.SimpleRunOptions{
		ConfigPath:  configPath,
		MaxParallel: maxParallel,
		VerboseLevel: effectiveVerboseLevel,
		Debug:       debug,
		DryRun:      dryRun,
		Variables:   variables,
		WorkingDir:  workingDir,
		Output:      os.Stdout,
		ErrorOutput: os.Stderr,
		Only:        only,
	}
	
	// Create simple runner
	runner := buildfab.NewSimpleRunner(cfg, opts)
	
	actionName := args[0]
	
	// Display version information before running action
	displayVersionInfo(cfg)
	
	// Run action using simple API
	err = runner.RunAction(ctx, actionName)
	if err != nil {
		// In test mode, return the error instead of exiting
		if testing.Testing() {
			return err
		}
		// Only show usage hints for action not found errors, not for execution errors
		if strings.Contains(err.Error(), "action not found") {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "To see available stages run: buildfab list-stages\n")
			fmt.Fprintf(os.Stderr, "To see available actions run: buildfab list-actions\n")
		} else {
			// For execution errors, SimpleRunner already handled the output via step callbacks
			// Just exit with error code without printing the error again
		}
		os.Exit(1)
	}
	return nil
}

// Custom output functions removed - now using library UI system

// runStage handles the run command
func runStage(cmd *cobra.Command, args []string) error {
	return runStageDirect(cmd, args)
}

// runAction handles the action command
func runAction(cmd *cobra.Command, args []string) error {
	return runActionDirect(cmd, args)
}

// runListActions handles the list-actions command
func runListActions(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return handleConfigLoadError(configPath, err)
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
		// Check if it's a validation error
		if strings.Contains(err.Error(), "step") && strings.Contains(err.Error(), "must have an action") ||
		   strings.Contains(err.Error(), "duplicate action name") ||
		   strings.Contains(err.Error(), "stage") && strings.Contains(err.Error(), "must have at least one step") {
			// In test mode, return the error instead of exiting
			if testing.Testing() {
				return err
			}
			// Print error with line number and exit directly to avoid duplication with cobra error handling
			enhancedError := enhanceValidationError(configPath, err)
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", enhancedError)
			os.Exit(1)
		}
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
		return handleConfigLoadError(configPath, err)
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

// runListVariables handles the list-variables command
func runListVariables(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return handleConfigLoadError(configPath, err)
	}
	
	// Create variables map from environment variables
	variables := make(map[string]string)
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		}
	}
	
	// Add matrix variables from CLI flags
	for key, value := range matrixVars {
		variables[fmt.Sprintf("matrix.%s", key)] = value
	}
	
	// Add platform variables
	variables = buildfab.AddPlatformVariables(variables)
	
	// Add version variables
	variables = buildfab.AddVersionVariables(variables)
	
	// Add direct project and version variables for convenience
	if cfg != nil {
		variables["project"] = cfg.Project.Name
		// Add first module as the primary module
		if len(cfg.Project.Modules) > 0 {
			variables["module"] = cfg.Project.Modules[0]
		}
	}
	
	// Add OS environment variables with env. prefix
	// Only add commonly used ones to avoid clutter
	commonEnvVars := []string{"PATH", "HOME", "USER", "SHELL", "TERM", "LANG", "PWD", "GOPATH", "GOROOT"}
	for _, envName := range commonEnvVars {
		if envValue := os.Getenv(envName); envValue != "" {
			variables[fmt.Sprintf("env.%s", envName)] = envValue
		}
	}
	
	fmt.Println("Available variables:")
	fmt.Println()
	
	if len(variables) == 0 {
		fmt.Println("  No variables available")
		return nil
	}
	
	// Sort variables by name
	var names []string
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)
	
	// Find the longest variable name for alignment
	maxNameLen := 0
	for _, name := range names {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}
	
	// Print variables in aligned format
	for _, name := range names {
		fmt.Printf("  %-*s = %s\n", maxNameLen, name, variables[name])
	}
	
	return nil
}

// printStepsGraph prints steps as a dependency graph
func printStepsGraph(stageName string, stage buildfab.Stage) error {
	fmt.Printf("Dependency graph for stage '%s':\n", stageName)
	fmt.Println()
	
	if len(stage.Steps) == 0 {
		fmt.Println("  No steps defined")
		return nil
	}
	
	// Create a map of step names to their dependencies
	stepDeps := make(map[string][]string)
	stepNames := make([]string, 0, len(stage.Steps))
	
	for _, step := range stage.Steps {
		stepNames = append(stepNames, step.Action)
		if len(step.Require) > 0 {
			stepDeps[step.Action] = step.Require
		} else {
			stepDeps[step.Action] = []string{}
		}
	}
	
	// Print the graph as a proper tree
	for i, stepName := range stepNames {
		deps := stepDeps[stepName]
		
		// Print the step
		fmt.Printf("  %2d. %s", i+1, stepName)
		if len(deps) > 0 {
			fmt.Printf(" (depends on: %s)", strings.Join(deps, ", "))
		}
		fmt.Println()
		
		// Print dependency tree
		if len(deps) > 0 {
			for j, dep := range deps {
				if j == len(deps)-1 {
					// Last dependency
					fmt.Printf("      └── %s\n", dep)
				} else {
					// Not last dependency
					fmt.Printf("      ├── %s\n", dep)
				}
			}
		}
		
		// Add spacing between steps
		if i < len(stepNames)-1 {
			fmt.Println()
		}
	}
	
	return nil
}

// runListSteps handles the list-steps command
func runListSteps(cmd *cobra.Command, args []string) error {
	// Load configuration using library API
	cfg, err := buildfab.LoadConfig(configPath)
	if err != nil {
		return handleConfigLoadError(configPath, err)
	}
	
	stageName := args[0]
	
	// Find the stage
	stage, exists := cfg.Stages[stageName]
	if !exists {
		return fmt.Errorf("stage '%s' not found in configuration", stageName)
	}
	
	if showGraph {
		return printStepsGraph(stageName, stage)
	}
	
	fmt.Printf("Steps for stage '%s':\n", stageName)
	fmt.Println()
	
	if len(stage.Steps) == 0 {
		fmt.Println("  No steps defined")
		return nil
	}
	
	for i, step := range stage.Steps {
		fmt.Printf("  %2d. %s\n", i+1, step.Action)
	}
	
	return nil
}

// handleConfigLoadError handles configuration loading errors with enhanced validation error messages
func handleConfigLoadError(configPath string, err error) error {
	// Check if it's a validation error
	if strings.Contains(err.Error(), "step") && strings.Contains(err.Error(), "must have an action") ||
	   strings.Contains(err.Error(), "duplicate action name") ||
	   strings.Contains(err.Error(), "stage") && strings.Contains(err.Error(), "must have at least one step") {
		// In test mode, return the error instead of exiting
		if testing.Testing() {
			return err
		}
		// Print error with line number and exit directly to avoid duplication with cobra error handling
		enhancedError := enhanceValidationError(configPath, err)
		fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", enhancedError)
		os.Exit(1)
	}
	return fmt.Errorf("failed to load configuration: %w", err)
}

// enhanceValidationError adds line number information to validation errors
func enhanceValidationError(configPath string, err error) string {
	// Try to extract step number from error message
	// Error format: "step X in stage Y must have an action"
	errorStr := err.Error()
	
	// Read the config file to find the actual line number
	content, readErr := os.ReadFile(configPath)
	if readErr != nil {
		// If we can't read the file, return original error
		return errorStr
	}
	
	lines := strings.Split(string(content), "\n")
	
	// Look for step validation errors
	if strings.Contains(errorStr, "step") && strings.Contains(errorStr, "must have an action") {
		// Extract step number from error message
		parts := strings.Fields(errorStr)
		var stepNum int
		for i, part := range parts {
			if part == "step" && i+1 < len(parts) {
				fmt.Sscanf(parts[i+1], "%d", &stepNum)
				break
			}
		}
		
		if stepNum > 0 {
			// Find the build stage and count steps
			inBuildStage := false
			stepCount := 0
			for lineNum, line := range lines {
				line = strings.TrimSpace(line)
				
				if strings.HasPrefix(line, "build:") || strings.HasPrefix(line, "test:") {
					inBuildStage = true
					continue
				}
				
				if inBuildStage {
					if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
						// This is a new stage or section, we're out of build stage
						if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "    ") {
							// Still in build stage
						} else {
							break
						}
					}
					
					if strings.HasPrefix(line, "- action:") {
						stepCount++
						if stepCount == stepNum {
							// Found the problematic line
							return fmt.Sprintf("%s:%d: %s", configPath, lineNum+1, errorStr)
						}
					} else if strings.HasPrefix(line, "- require:") && stepCount == stepNum-1 {
						// This is a require line without action - this is the problem
						return fmt.Sprintf("%s:%d: %s", configPath, lineNum+1, errorStr)
					}
				}
			}
		}
	}
	
	// For other validation errors, try to find relevant lines
	if strings.Contains(errorStr, "duplicate action name") {
		// Extract action name from error
		parts := strings.Fields(errorStr)
		var actionName string
		for i, part := range parts {
			if part == "duplicate" && i+2 < len(parts) && parts[i+1] == "action" && parts[i+2] == "name:" {
				if i+3 < len(parts) {
					actionName = parts[i+3]
					break
				}
			}
		}
		
		if actionName != "" {
			// Find the duplicate action definition
			for lineNum, line := range lines {
				if strings.Contains(line, fmt.Sprintf("name: %s", actionName)) {
					return fmt.Sprintf("%s:%d: %s", configPath, lineNum+1, errorStr)
				}
			}
		}
	}
	
	// If we can't find specific line, return original error
	return errorStr
}