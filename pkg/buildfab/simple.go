package buildfab

import (
    "context"
    "fmt"
    "io"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab/container"
    containerRunner "github.com/AlexBurnes/buildfab/internal/container"
)

// formatExecutionTime formats a duration in the requested format (e.g., '20s' or '1m 20s')
// Only shows time for successful actions, returns empty string for errors
func formatExecutionTime(status StepStatus, duration time.Duration) string {
    // Only show execution time on success (OK status)
    if status != StepStatusOK {
        return ""
    }

    // Format duration as requested: '20s' or '1m 20s'
    totalSeconds := int(duration.Seconds())
    minutes := totalSeconds / 60
    seconds := totalSeconds % 60

    if minutes > 0 {
        return fmt.Sprintf(" - in '%dm %ds'", minutes, seconds)
    } else if seconds > 0 {
        return fmt.Sprintf(" - in '%ds'", seconds)
    } else {
        // For sub-second durations, show as fractional seconds (e.g., '0.002s')
        return fmt.Sprintf(" - in '%.3fs'", duration.Seconds())
    }
}

// SimpleRunner provides a simplified interface for running stages and actions
// without requiring callback setup. All output is handled internally.
type SimpleRunner struct {
    config   *Config
    opts     *SimpleRunOptions
    registry ActionRegistry
}

// SimpleRunOptions configures simple stage execution
type SimpleRunOptions struct {
    ConfigPath         string            // Path to project.yml (default: ".project.yml")
    MaxParallel        int               // Maximum parallel execution (default: CPU count)
    VerboseLevel       int               // Verbosity level: 0=quiet, 1=-v, 2=-vv, 3=-vvv
    Debug              bool              // Enable debug output
    DryRun             bool              // Show what would be executed without running commands
    Variables          map[string]string // Additional variables for interpolation
    WorkingDir         string            // Working directory for execution
    Input              io.Reader         // Input reader (default: nil for non-interactive)
    Output             io.Writer         // Output writer (default: os.Stdout)
    ErrorOutput        io.Writer         // Error output writer (default: os.Stderr)
    Only               []string          // Only run steps matching these labels
    WithRequires       bool              // Include required dependencies when running single step
    BuildfabBinaryPath string            // Path to buildfab binary for run_action/run_stage (optional, auto-detected if not specified)
    StepCallback       StepCallback      // Optional callback for step execution (mainly for tests)
}

// DefaultSimpleRunOptions returns default simple run options
func DefaultSimpleRunOptions() *SimpleRunOptions {
    variables := make(map[string]string)
    // Add platform variables by default
    variables = AddPlatformVariables(variables)
    // Add version variables by default
    variables = AddVersionVariables(variables)

    return &SimpleRunOptions{
        ConfigPath:  ".project.yml",
        MaxParallel: runtime.NumCPU(),
        VerboseLevel: 1,
        Debug:       false,
        Variables:   variables,
        WorkingDir:  ".",
        Output:      os.Stdout,
        ErrorOutput: os.Stderr,
        Only:        []string{},
        WithRequires: false,
    }
}

// NewSimpleRunner creates a new simple buildfab runner
func NewSimpleRunner(config *Config, opts *SimpleRunOptions) *SimpleRunner {
    if opts == nil {
        opts = DefaultSimpleRunOptions()
    }
    return &SimpleRunner{
        config:   config,
        opts:     opts,
        registry: NewDefaultActionRegistry(),
    }
}

// RunStage executes a specific stage with automatic output handling
func (r *SimpleRunner) RunStage(ctx context.Context, stageName string) error {
    if r.opts.Debug {
        fmt.Fprintf(os.Stderr, "[DEBUG] SimpleRunner.RunStage called with stageName=%s\n", stageName)
    }
    stage, exists := r.config.GetStage(stageName)
    if !exists {
        return fmt.Errorf("stage not found: %s", stageName)
    }

    // Debug output
    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] SimpleRunner.RunStage: stageName=%s (using hierarchical DAG)\n", stageName)
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Steps: %+v\n", stage.Steps)
        for i, step := range stage.Steps {
            fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Step %d: %+v\n", i, step)
            if step.Matrix != nil {
                fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix config: %+v\n", step.Matrix)
            }
        }
    }

    // Always use hierarchical DAG execution (the correct architecture)
    return r.runStageWithHierarchicalDAG(ctx, stageName, stage)
}

// RunAction executes a specific action with automatic output handling
func (r *SimpleRunner) RunAction(ctx context.Context, actionName string) error {
    // Print action header
    fmt.Fprintf(r.opts.Output, "▶️  Running action: %s\n\n", actionName)

    // Create step callback to collect results
    stepCallback := &SimpleStepCallback{
        verboseLevel: r.opts.VerboseLevel,
        debug:   r.opts.Debug,
        output:  r.opts.ErrorOutput,  // Use errorOutput for step results
        errorOutput: r.opts.ErrorOutput,
        config:  r.config,
        configPath: r.opts.ConfigPath,
        bufferedOutput: make(map[string]*BufferedOutput),
        variables: r.opts.Variables,
    }

    // Convert to complex options and use internal runner
    complexOpts := &RunOptions{
        ConfigPath:         r.opts.ConfigPath,
        MaxParallel:        r.opts.MaxParallel,
        VerboseLevel:       r.opts.VerboseLevel,
        Debug:              r.opts.Debug,
        DryRun:             r.opts.DryRun,
        Variables:          r.opts.Variables,
        WorkingDir:         r.opts.WorkingDir,
        Input:              r.opts.Input,
        Output:             r.opts.Output,
        ErrorOutput:        r.opts.ErrorOutput,
        Only:               r.opts.Only,
        WithRequires:       r.opts.WithRequires,
        BuildfabBinaryPath: r.opts.BuildfabBinaryPath,
        StepCallback:       stepCallback,
    }

    runner := NewRunner(r.config, complexOpts)
    err := runner.RunAction(ctx, actionName)

    // Get collected results
    results := stepCallback.GetResults()

    // Show final status summary for single actions
    if len(results) > 0 {
        // Check if execution was terminated due to context cancellation
        terminated := ctx.Err() != nil

        // Check if any step failed or has warnings
        hasError := false
        hasWarning := false
        for _, result := range results {
            if result.Status == StepStatusError {
                hasError = true
                break
            } else if result.Status == StepStatusWarn {
                hasWarning = true
            }
        }

        // Show final status
        fmt.Fprintf(r.opts.Output, "\n")
        fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
        if terminated {
            fmt.Fprintf(r.opts.Output, "⏹️ %s%s%s - %s\n", colorYellow, "TERMINATED", colorReset, actionName)
        } else if hasError {
            fmt.Fprintf(r.opts.Output, "💥 %s%s%s - %s\n", colorRed, "FAILED", colorReset, actionName)
        } else if hasWarning {
            fmt.Fprintf(r.opts.Output, "⚠️ %s%s%s - %s\n", colorYellow, "WARNING", colorReset, actionName)
        } else {
            fmt.Fprintf(r.opts.Output, "🎉 %s%s%s - %s\n", colorGreen, "SUCCESS", colorReset, actionName)
        }
    }

    return err
}

// RunStageStep executes a specific step within a stage with automatic output handling
func (r *SimpleRunner) RunStageStep(ctx context.Context, stageName, stepName string) error {
    stage, exists := r.config.GetStage(stageName)
    if !exists {
        return fmt.Errorf("stage not found: %s", stageName)
    }

    // Find the step
    var targetStep *Step
    for i, step := range stage.Steps {
        if step.GetStepName() == stepName {
            targetStep = &stage.Steps[i]
            break
        }
    }

    if targetStep == nil {
        return fmt.Errorf("step not found: %s in stage %s", stepName, stageName)
    }

    // Print step header
    fmt.Fprintf(r.opts.Output, "▶️  Running step: %s (from stage: %s)\n\n", stepName, stageName)

    // Convert to complex options and use internal runner
    complexOpts := &RunOptions{
        ConfigPath:         r.opts.ConfigPath,
        MaxParallel:        r.opts.MaxParallel,
        VerboseLevel:       r.opts.VerboseLevel,
        Debug:              r.opts.Debug,
        DryRun:             r.opts.DryRun,
        Variables:          r.opts.Variables,
        WorkingDir:         r.opts.WorkingDir,
        Input:              r.opts.Input,
        Output:             r.opts.Output,
        ErrorOutput:        r.opts.ErrorOutput,
        Only:               r.opts.Only,
        WithRequires:       r.opts.WithRequires,
        BuildfabBinaryPath: r.opts.BuildfabBinaryPath,
        StepCallback:       &SimpleStepCallback{
            verboseLevel: r.opts.VerboseLevel,
            debug:   r.opts.Debug,
            output:  r.opts.ErrorOutput,  // Use errorOutput for step results
            errorOutput: r.opts.ErrorOutput,
            config:  r.config,
            configPath: r.opts.ConfigPath,
            bufferedOutput: make(map[string]*BufferedOutput),
            variables: r.opts.Variables,
        },
    }

    runner := NewRunner(r.config, complexOpts)
    return runner.RunStageStep(ctx, stageName, stepName)
}

// SimpleStepCallback implements StepCallback for simple output
type SimpleStepCallback struct {
    verboseLevel int
    debug       bool
    output      io.Writer
    errorOutput io.Writer
    results     []StepResult
    displayed   map[string]bool
    config      *Config // Store config to access action details
    configPath  string  // Store config path for container actions
    bufferedOutput map[string]*BufferedOutput // Store buffered output for quiet mode
    variables   map[string]string // Store variables for interpolation
    stepStartTimes map[string]time.Time // Store actual start times for steps
    mu           sync.Mutex // Protect stepStartTimes map
}

// BufferedOutput stores captured stdout and stderr for a step
type BufferedOutput struct {
    Stdout string
    Stderr string
}


// StoreBufferedOutput stores captured stdout and stderr for a step
func (c *SimpleStepCallback) StoreBufferedOutput(stepName, stdout, stderr string) {
    if c.bufferedOutput == nil {
        c.bufferedOutput = make(map[string]*BufferedOutput)
    }
    c.bufferedOutput[stepName] = &BufferedOutput{
        Stdout: stdout,
        Stderr: stderr,
    }
}

func (c *SimpleStepCallback) OnStepStart(ctx context.Context, stepName string) {
    // Capture actual start time when execution begins
    actualStartTime := time.Now()
    c.mu.Lock()
    if c.stepStartTimes == nil {
        c.stepStartTimes = make(map[string]time.Time)
    }
    c.stepStartTimes[stepName] = actualStartTime
    c.mu.Unlock()
    
    if c.verboseLevel > 0 {
        timestamp := formatISO8601Timestamp(actualStartTime)
        fmt.Fprintf(c.errorOutput, "  💻 %s [%s]\n", stepName, timestamp)
        
        // Show container command for container actions (verbose level 2+)
        if c.verboseLevel >= 2 {
            c.showContainerCommand(stepName)
        }
    } else {
        // In silence mode, show running indicator
        fmt.Fprintf(c.errorOutput, "  %s%s%s %s running...\r", colorCyan, "○", colorReset, stepName)
    }
}

// showContainerCommand shows the container command for container actions
func (c *SimpleStepCallback) showContainerCommand(stepName string) {
    // Get the action
    action, exists := c.config.GetAction(stepName)
    if !exists {
        return
    }

    if action.Container != nil {
        // Create a temporary runner to prepare the configuration for display
        tempRunner, err := containerRunner.NewContainerRunnerWithVerbosity(c.verboseLevel)
        if err != nil {
            // Silently ignore runner creation errors for display purposes
            return
        }
        
        // Use the actual config path
        configPath := c.configPath
        if configPath == "" {
            configPath = ".project.yml" // Fallback
        }
        
        // Prepare the container configuration
        preparedConfig, prepErr := tempRunner.PrepareContainerConfig(*action.Container, configPath)
        if prepErr != nil {
            // Silently ignore preparation errors for display purposes
            // (artifacts might not be mounted yet, which is OK for display)
            return
        }
        
        // Interpolate variables in the container configuration for display
        if c.variables != nil {
            interpolatedConfig, err := InterpolateContainerConfig(&preparedConfig, c.variables)
            if err == nil {
                preparedConfig = *interpolatedConfig
            }
        }
        
        containerCmd := c.buildContainerCommand(&preparedConfig)
        
        // Determine the appropriate prefix based on the container operation type
        prefix := "Running container"
        if preparedConfig.Image.Build != nil {
            prefix = "Building image"
        } else if preparedConfig.Image.Slim != nil {
            prefix = "Slimming image"
        }
        
        fmt.Fprintf(c.errorOutput, "  🐳 %s: %s\n", prefix, containerCmd)
    }
}

// buildContainerCommand builds a human-readable representation of the container command
func (c *SimpleStepCallback) buildContainerCommand(config *container.ContainerConfig) string {
    var parts []string

    // Use specified engine or default to podman
    engineName := "podman" // Default to podman
    if config.Engine != "" {
        engineName = config.Engine
    }

    // Add engine (Docker/Podman)
    parts = append(parts, engineName)

    // Handle build operations differently
    if config.Image.Build != nil {
        // For build operations, we run docker/podman build command
        parts = append(parts, "build")

        // Add build args
        for key, value := range config.Image.Build.Args {
            parts = append(parts, "--build-arg", fmt.Sprintf("%s=%s", key, value))
        }

        // Add tags
        for _, tag := range config.Image.Build.Tags {
            parts = append(parts, "--tag", tag)
        }

        // Add network if specified
        if config.Image.Build.Network != "" {
            parts = append(parts, "--network", config.Image.Build.Network)
        }

        // Add progress if specified
        if config.Image.Build.Progress != "" {
            parts = append(parts, "--progress", config.Image.Build.Progress)
        }

        // Add dockerfile if specified
        if config.Image.Build.Dockerfile != "" {
            parts = append(parts, "-f", config.Image.Build.Dockerfile)
        }

        // Add context (default to current directory)
        context := config.Image.Build.Context
        if context == "" {
            context = "."
        }
        parts = append(parts, context)
    } else if config.Image.Slim != nil {
        // For slim operations, we need to mount the Docker socket
        if engineName == "docker" {
            parts = append(parts, "-v", "/var/run/docker.sock:/var/run/docker.sock")
        } else if engineName == "podman" {
            parts = append(parts, "-v", "/run/podman/podman.sock:/run/podman/podman.sock")
        }

        // For slim operations, we run the dslim/slim container with specific arguments
        parts = append(parts, "dslim/slim:latest")
        parts = append(parts, "slim", "build")

        // Add slim-specific flags
        if !config.Image.Slim.HttpProbe {
            parts = append(parts, "--http-probe=false")
            parts = append(parts, "--continue-after=exit")
        }

        parts = append(parts, config.Image.Slim.Target)

        // Add exec command if specified
        if config.Image.Slim.Exec != "" {
            parts = append(parts, "--exec", config.Image.Slim.Exec)
        }

        // Add tags for the slim image
        for _, tag := range config.Image.Slim.Tags {
            parts = append(parts, "--tag", tag)
        }
    } else {
        // Regular container run command
        parts = append(parts, "run", "--rm")

        // Add environment variables
        for key, value := range config.Env {
            parts = append(parts, "-e", fmt.Sprintf("\"%s=%s\"", key, value))
        }

        // Add mounts for workspace and binary
        for _, mount := range config.Mounts {
            parts = append(parts, fmt.Sprintf("--mount=type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target))
        }

        // Add working directory
        if config.Workdir != "" {
            parts = append(parts, "-w", config.Workdir)
        }

        // Add CPU settings
        if config.CPU > 0 {
            parts = append(parts, "--cpus", fmt.Sprintf("%d.0", config.CPU))
        }

        // Add memory limit
        if config.Memory != "" {
            parts = append(parts, "-m", config.Memory)
        }

        // Add network mode
        if config.Network != "" {
            parts = append(parts, "--network", config.Network)
        }

        // Add image name
        parts = append(parts, config.Image.From)

        // Add command
        if config.Run != "" {
            parts = append(parts, "sh", "-c", fmt.Sprintf("'%s'", config.Run))
        }
    }

    return strings.Join(parts, " ")
}

func (c *SimpleStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    // Initialize displayed map if not already done
    if c.displayed == nil {
        c.displayed = make(map[string]bool)
    }

    // Only display each step once
    if !c.displayed[stepName] {
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
        case StepStatusSkipped, StepStatusSkippedCondition:
            icon = "→"
            color = colorGray
        default:
            icon = "?"
            color = colorGray
        }

        // Show step results with enhanced error messages
        displayMessage := c.enhanceMessage(stepName, status, message)

        // Add execution time for successful actions
        executionTime := formatExecutionTime(status, duration)
        displayMessage += executionTime

        if c.verboseLevel > 0 {
            // In verbose mode, just print the result
            fmt.Fprintf(c.errorOutput, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, displayMessage)
        } else {
            // In silence mode, replace the running indicator with the result
            // Clear the running line and print the final result
            fmt.Fprintf(c.errorOutput, "\r  %s%s%s %s %s\n", color, icon, colorReset, stepName, displayMessage)

            // In quiet mode, if the step failed, display buffered output
            if status == StepStatusError && c.bufferedOutput != nil {
                if buffered, exists := c.bufferedOutput[stepName]; exists {
                    c.displayBufferedOutput(stepName, buffered)
                }
            }
        }
        c.displayed[stepName] = true
    }

    // Collect result for summary (avoid duplicates)
    found := false
    for i, result := range c.results {
        if result.StepName == stepName {
            c.results[i] = StepResult{
                StepName: stepName,
                Status:   status,
                Duration: duration,
            }
            found = true
            break
        }
    }
    if !found {
        c.results = append(c.results, StepResult{
            StepName: stepName,
            Status:   status,
            Duration: duration,
        })
    }
}

func (c *SimpleStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    if output != "" {
        lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
        for _, line := range lines {
            fmt.Fprintf(c.output, "    %s\n", line)
        }
    }
}

func (c *SimpleStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
    // Don't display here - OnStepComplete should handle all display
}

// displayBufferedOutput displays the buffered stdout and stderr for a failed step
func (c *SimpleStepCallback) displayBufferedOutput(stepName string, buffered *BufferedOutput) {
    if buffered.Stdout != "" || buffered.Stderr != "" {
        fmt.Fprintf(c.errorOutput, "\n")
        if buffered.Stdout != "" {
            lines := strings.Split(strings.TrimRight(buffered.Stdout, "\n"), "\n")
            for _, line := range lines {
                fmt.Fprintf(c.errorOutput, "    %s\n", line)
            }
        }
        if buffered.Stderr != "" {
            lines := strings.Split(strings.TrimRight(buffered.Stderr, "\n"), "\n")
            for _, line := range lines {
                fmt.Fprintf(c.errorOutput, "    %s\n", line)
            }
        }
    }
}

// GetResults returns the collected step results
func (c *SimpleStepCallback) GetResults() []StepResult {
    return c.results
}

// enhanceMessage improves error messages to match v0.5.0 style
func (c *SimpleStepCallback) enhanceMessage(stepName string, status StepStatus, message string) string {
    switch status {
    case StepStatusOK:
        // For dry-run messages, return as-is
        if strings.Contains(message, "would execute") {
            return message
        }
        return message
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
                command := c.extractCommand(stepName, strings.Join(reproductionLines, "\n"))
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
                command := c.extractCommand(stepName, message)
                return fmt.Sprintf("to check run:\n%s", command)
            }
        }
        // Enhance error messages with reproduction instructions
        if strings.Contains(message, "command failed: exit status") {
            // Extract the command from the step name or message
            command := c.extractCommand(stepName, message)
            return fmt.Sprintf("to check run:\n%s", command)
        }
        return message
    case StepStatusSkipped:
        // Enhance skipped messages with dependency information
        if strings.Contains(message, "dependency failed") {
            // Try to extract which dependency failed
            failedDep := c.extractFailedDependency(stepName)
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
func (c *SimpleStepCallback) extractCommand(stepName, message string) string {
    // First, check if the message already contains the command (from error message)
    // This will have interpolated variables
    if strings.Contains(message, "to check run:") {
        lines := strings.Split(message, "\n")
        var commandLines []string
        foundCommand := false
        for _, line := range lines {
            if strings.Contains(line, "to check run:") {
                foundCommand = true
                continue // Skip the "to check run:" line itself
            } else if foundCommand && strings.TrimSpace(line) != "" {
                commandLines = append(commandLines, line)
            }
        }
        if len(commandLines) > 0 {
            // Reformat with proper indentation
            var alignedLines []string
            for _, line := range commandLines {
                // Trim existing indentation and add 6 spaces
                alignedLines = append(alignedLines, "      "+strings.TrimLeft(line, " \t"))
            }
            return strings.Join(alignedLines, "\n")
        }
    }
    
    if c.config == nil {
        return fmt.Sprintf("      %s", stepName)
    }

        // Try to find the action and extract the actual command
        action, exists := c.config.GetAction(stepName)
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

            // If no non-empty lines found, use 0
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
func (c *SimpleStepCallback) extractFailedDependency(stepName string) string {
    // Look through results to find failed dependencies
    for _, result := range c.results {
        if result.Status == StepStatusError {
            // This is a simple heuristic - in a real implementation,
            // we'd need to track the dependency graph
            return result.StepName
        }
    }
    return ""
}

// getSkippedSteps determines which steps should be skipped due to failed dependencies
func (r *SimpleRunner) getSkippedSteps(stageName string, executedResults []StepResult) []string {
    stage, exists := r.config.GetStage(stageName)
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
        if result.Status == StepStatusError {
            failedSteps[result.StepName] = true
        }
    }

    var skippedSteps []string

    // Check each step in the stage
    for _, step := range stage.Steps {
        stepName := step.GetStepName()

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

// printTerminatedSummary prints the stage result when execution was terminated
func (r *SimpleRunner) printTerminatedSummary(stageName string, results []StepResult, duration time.Duration) {
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

    icon := "⏹️"
    color := colorYellow
    status := "TERMINATED"

    // Format duration to match step execution time format: 'in 0.021s'
    var durationStr string
    if duration.Seconds() < 1 {
        // For sub-second durations, show as fractional seconds (e.g., 'in 0.021s')
        durationStr = fmt.Sprintf(" in %.3fs", duration.Seconds())
    } else {
        // For durations >= 1 second, show as whole seconds (e.g., 'in 3s')
        totalSeconds := int(duration.Seconds())
        minutes := totalSeconds / 60
        seconds := totalSeconds % 60

        if minutes > 0 {
            durationStr = fmt.Sprintf(" in %dm %ds", minutes, seconds)
        } else {
            durationStr = fmt.Sprintf(" in %ds", seconds)
        }
    }

    fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s%s\n", icon, color, status, colorReset, stageName, durationStr)

    // Print summary
    if len(results) > 0 {
        fmt.Fprintf(r.opts.Output, "\n")
        fmt.Fprintf(r.opts.Output, "📊 Summary:\n")

        statusCounts := make(map[StepStatus]int)
        for _, result := range results {
            statusCounts[result.Status]++
        }

        // Combine both types of skips for display (they show the same to users)
        totalSkipped := statusCounts[StepStatusSkipped] + statusCounts[StepStatusSkippedCondition]
        statusCounts[StepStatusSkipped] = totalSkipped
        delete(statusCounts, StepStatusSkippedCondition)

        // Define status order for consistent display
        statusOrder := []StepStatus{
            StepStatusError,
            StepStatusWarn,
            StepStatusOK,
            StepStatusSkipped,
        }

        for _, status := range statusOrder {
            count := statusCounts[status]
            var icon, color string

            switch status {
            case StepStatusOK:
                icon = "✓"
                if count > 0 {
                    color = colorGreen
                } else {
                    color = colorGray
                }
            case StepStatusWarn:
                icon = "!"
                if count > 0 {
                    color = colorYellow
                } else {
                    color = colorGray
                }
            case StepStatusError:
                icon = "✗"
                if count > 0 {
                    color = colorRed
                } else {
                    color = colorGray
                }
            case StepStatusSkipped:
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

            fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", color, icon, colorReset, color, status.String(), count, colorReset)
        }
    }
}

// printSummary prints the stage result with summary
func (r *SimpleRunner) printSummary(stageName string, success bool, results []StepResult, duration time.Duration) {
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

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

    // Format duration to match step execution time format: 'in 0.021s'
    var durationStr string
    if duration.Seconds() < 1 {
        // For sub-second durations, show as fractional seconds (e.g., 'in 0.021s')
        durationStr = fmt.Sprintf(" in %.3fs", duration.Seconds())
    } else {
        // For durations >= 1 second, show as whole seconds (e.g., 'in 3s')
        totalSeconds := int(duration.Seconds())
        minutes := totalSeconds / 60
        seconds := totalSeconds % 60

        if minutes > 0 {
            durationStr = fmt.Sprintf(" in %dm %ds", minutes, seconds)
        } else {
            durationStr = fmt.Sprintf(" in %ds", seconds)
        }
    }

    fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s%s\n", icon, color, status, colorReset, stageName, durationStr)

    // Print summary
    if len(results) > 0 {
        fmt.Fprintf(r.opts.Output, "\n")
        fmt.Fprintf(r.opts.Output, "📊 Summary:\n")

        statusCounts := make(map[StepStatus]int)
        for _, result := range results {
            statusCounts[result.Status]++
        }

        // Combine both types of skips for display (they show the same to users)
        totalSkipped := statusCounts[StepStatusSkipped] + statusCounts[StepStatusSkippedCondition]
        statusCounts[StepStatusSkipped] = totalSkipped
        delete(statusCounts, StepStatusSkippedCondition)

        // Define status order for consistent display
        statusOrder := []StepStatus{
            StepStatusError,
            StepStatusWarn,
            StepStatusOK,
            StepStatusSkipped,
        }

        for _, status := range statusOrder {
            count := statusCounts[status]
            var icon, color string

            switch status {
            case StepStatusOK:
                icon = "✓"
                if count > 0 {
                    color = colorGreen
                } else {
                    color = colorGray
                }
            case StepStatusWarn:
                icon = "!"
                if count > 0 {
                    color = colorYellow
                } else {
                    color = colorGray
                }
            case StepStatusError:
                icon = "✗"
                if count > 0 {
                    color = colorRed
                } else {
                    color = colorGray
                }
            case StepStatusSkipped:
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

            fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", color, icon, colorReset, color, status.String(), count, colorReset)
        }
    }
}

// executeStageDryRun simulates stage execution for dry-run mode
func (r *SimpleRunner) executeStageDryRun(ctx context.Context, stageName string, steps []Step) error {
    // Print stage header
    fmt.Fprintf(r.opts.Output, "▶️  Dry run stage: %s\n\n", stageName)

    // Count total steps
    totalSteps := len(steps)
    skippedSteps := 0
    executedSteps := 0

    // Process each step
    for _, step := range steps {
        // Check if step should be executed based on conditions
        shouldExecute, err := r.shouldExecuteStepByCondition(ctx, step)
        if err != nil {
            return fmt.Errorf("failed to evaluate step condition: %w", err)
        }

        if !shouldExecute {
            skippedSteps++
            if r.opts.VerboseLevel > 0 {
                fmt.Fprintf(r.opts.Output, "→ %s: would skip (condition not met)\n", step.Action)
            }
            continue
        }

        // Check if step should be executed based on only filter
        if len(r.opts.Only) > 0 {
            stepMatches := false
            for _, label := range r.opts.Only {
                for _, stepLabel := range step.Only {
                    if stepLabel == label {
                        stepMatches = true
                        break
                    }
                }
            }
            if !stepMatches {
                skippedSteps++
                if r.opts.VerboseLevel > 0 {
                    fmt.Fprintf(r.opts.Output, "→ %s: would skip (not in only filter)\n", step.Action)
                }
                continue
            }
        }

        executedSteps++

        // Simulate action execution
        action, exists := r.config.GetAction(step.Action)
        if !exists {
            // Check if it's a built-in action
            if runner, exists := r.registry.GetRunner(step.Action); exists {
                description := runner.Description()
                if r.opts.VerboseLevel > 0 {
                    fmt.Fprintf(r.opts.Output, "  ✓ %s would execute built-in action: %s\n", step.Action, description)
                }
            } else {
                if r.opts.VerboseLevel > 0 {
                    fmt.Fprintf(r.opts.Output, "  ✗ %s would fail (action not found)\n", step.Action)
                }
            }
        } else {
            // Simulate custom action execution
            err := r.runActionInternalDryRun(ctx, action)
            if err != nil {
                if r.opts.VerboseLevel > 0 {
                    fmt.Fprintf(r.opts.Output, "  ✗ %s would fail (%v)\n", step.Action, err)
                }
            }
        }
    }

    // Print summary
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    fmt.Fprintf(r.opts.Output, "🔍 %s%s%s - %s\n", colorCyan, "DRY RUN", colorReset, stageName)
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "📊 Summary:\n")
    fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGreen, "✓", colorReset, colorGreen, "would run", executedSteps, colorReset)
    fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGray, "→", colorReset, colorGray, "skipped", skippedSteps, colorReset)
    fmt.Fprintf(r.opts.Output, "   %s%s%s %s%-8s %3d%s\n", colorGray, "?", colorReset, colorGray, "total", totalSteps, colorReset)

    return nil
}

// shouldExecuteStepByCondition determines if a step should be executed based on its if condition
func (r *SimpleRunner) shouldExecuteStepByCondition(ctx context.Context, step Step) (bool, error) {
	// If no if condition is specified, execute the step
	if step.If == "" {
		return true, nil
	}
	
	// Merge step-level variables with global variables for condition evaluation
	// Step variables (including matrix variables) take precedence
	mergedVars := make(map[string]string)
	for k, v := range r.opts.Variables {
		mergedVars[k] = v
	}
	for k, v := range step.Variables {
		mergedVars[k] = v
	}
	
	// Evaluate the if condition using the expression evaluator with merged variables
	shouldExecute, err := evaluateCondition(step.If, mergedVars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate if condition for step %s: %w", step.Action, err)
	}
	
	return shouldExecute, nil
}

// runActionInternalDryRun simulates action execution for dry-run mode
func (r *SimpleRunner) runActionInternalDryRun(ctx context.Context, action Action) error {
    // Select variant if action has variants
    variant, err := action.SelectVariant(r.opts.Variables)
    if err != nil {
        return err
    }

    // If variant is nil and action has variants, it means no variant matched - skip
    if variant == nil && len(action.Variants) > 0 {
        return nil // Not an error, just skipped
    }

    // Use variant if available, otherwise use action directly
    effectiveAction := action
    if variant != nil {
        effectiveAction = Action{
            Name:  action.Name,
            Run:   variant.Run,
            Uses:  variant.Uses,
            Shell: variant.Shell,
        }
    }

    if effectiveAction.Uses != "" {
        return r.runBuiltInActionDryRun(ctx, effectiveAction)
    }

    return r.runCustomActionDryRun(ctx, effectiveAction)
}

// runBuiltInActionDryRun simulates built-in action execution for dry-run mode
func (r *SimpleRunner) runBuiltInActionDryRun(ctx context.Context, action Action) error {
    if r.registry == nil {
        return fmt.Errorf("built-in action %s not supported: no action registry provided", action.Uses)
    }

    runner, exists := r.registry.GetRunner(action.Uses)
    if !exists {
        return fmt.Errorf("unknown built-in action: %s", action.Uses)
    }

    description := runner.Description()

    // Print what would be executed if verbose mode is enabled
    if r.opts.VerboseLevel > 0 {
        fmt.Fprintf(r.opts.Output, "  ✓ %s would execute built-in action: %s\n", action.Name, description)
    }

    return nil
}

// runCustomActionDryRun simulates custom action execution for dry-run mode
func (r *SimpleRunner) runCustomActionDryRun(ctx context.Context, action Action) error {
    if action.Run == "" {
        return fmt.Errorf("action %s has no run command", action.Name)
    }

    // Interpolate variables in the action
    interpolatedAction, err := InterpolateAction(action, r.opts.Variables)
    if err != nil {
        return fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
    }

    // Get shell command info
    shell, shellArgs, err := getShellCommand(action.Shell, r.opts.VerboseLevel)
    if err != nil {
        return fmt.Errorf("shell configuration error for action %s: %w", action.Name, err)
    }

    // Build the full command
    fullCommand := append(shellArgs, interpolatedAction.Run)
    commandStr := shell + " " + strings.Join(fullCommand, " ")

    // Print what would be executed if verbose mode is enabled
    if r.opts.VerboseLevel > 0 {
        fmt.Fprintf(r.opts.Output, "  💻 %s\n", action.Name)
        fmt.Fprintf(r.opts.Output, "   ✓ %s, would execute command:\n", action.Name)

        // Handle multiline commands with proper indentation
        lines := strings.Split(commandStr, "\n")
        for i, line := range lines {
            if i == 0 {
                // First line with 6 spaces indentation
                fmt.Fprintf(r.opts.Output, "      %s\n", line)
            } else {
                // Subsequent lines with 6 spaces indentation
                fmt.Fprintf(r.opts.Output, "      %s\n", line)
            }
        }
    }

    return nil
}

// Simple convenience functions for even easier usage

// RunStageSimple executes a stage with minimal configuration
func RunStageSimple(ctx context.Context, configPath, stageName string, verboseLevel int) error {
    cfg, err := LoadConfig(configPath)
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    opts := &SimpleRunOptions{
        ConfigPath: configPath,
        VerboseLevel: verboseLevel,
        Output:     os.Stdout,
        ErrorOutput: os.Stderr,
    }

    runner := NewSimpleRunner(cfg, opts)
    return runner.RunStage(ctx, stageName)
}

// RunActionSimple executes an action with minimal configuration
func RunActionSimple(ctx context.Context, configPath, actionName string, verboseLevel int) error {
    cfg, err := LoadConfig(configPath)
    if err != nil {
        return fmt.Errorf("failed to load configuration: %w", err)
    }

    opts := &SimpleRunOptions{
        ConfigPath: configPath,
        VerboseLevel: verboseLevel,
        Output:     os.Stdout,
        ErrorOutput: os.Stderr,
    }

    runner := NewSimpleRunner(cfg, opts)
    return runner.RunAction(ctx, actionName)
}

// expandMatrixSteps expands matrix steps into individual steps for DAG execution
// expandStageReferences expands stage references in steps to their actual steps
func (r *SimpleRunner) expandStageReferences(steps []Step) ([]Step, error) {
    var expandedSteps []Step
    
    var expandStep func(step Step, depth int) ([]Step, error)
    expandStep = func(step Step, depth int) ([]Step, error) {
        // Check recursion depth to prevent stack overflow
        if depth > 100 {
            return nil, fmt.Errorf("maximum recursion depth exceeded when expanding stage references")
        }
        
        // If step references an action, return it as-is
        if step.Action != "" {
            return []Step{step}, nil
        }
        
        // If step references a stage, expand it
        if step.Stage != "" {
            // If step has both Stage and Matrix, skip expansion here - it will be handled by expandMatrixSteps
            if step.Matrix != nil {
                return []Step{step}, nil
            }
            
            // Get the referenced stage
            referencedStage, exists := r.config.GetStage(step.Stage)
            if !exists {
                return nil, fmt.Errorf("referenced stage not found: %s", step.Stage)
            }
            
            var stageSteps []Step
            for _, refStep := range referencedStage.Steps {
                // Recursively expand nested stage references
                subSteps, err := expandStep(refStep, depth+1)
                if err != nil {
                    return nil, err
                }
                
                // Inherit properties from the parent step where appropriate
                for i := range subSteps {
                    // If parent step has variables, merge with child (parent takes precedence)
                    if len(step.Variables) > 0 {
                        if subSteps[i].Variables == nil {
                            subSteps[i].Variables = make(map[string]string)
                        }
                        for k, v := range step.Variables {
                            if _, exists := subSteps[i].Variables[k]; !exists {
                                subSteps[i].Variables[k] = v
                            }
                        }
                    }
                    
                    // Inherit onerror if not set in child
                    if step.OnError != "" && subSteps[i].OnError == "" {
                        subSteps[i].OnError = step.OnError
                    }
                    
                    // Inherit if condition if not set in child (combine with AND)
                    if step.If != "" {
                        if subSteps[i].If == "" {
                            subSteps[i].If = step.If
                        } else {
                            subSteps[i].If = fmt.Sprintf("(%s) && (%s)", step.If, subSteps[i].If)
                        }
                    }
                }
                
                stageSteps = append(stageSteps, subSteps...)
            }
            
            return stageSteps, nil
        }
        
        // Should not reach here (validation ensures action or stage is set)
        return nil, fmt.Errorf("step has neither action nor stage")
    }
    
    // Expand each step
    for _, step := range steps {
        expanded, err := expandStep(step, 0)
        if err != nil {
            return nil, err
        }
        expandedSteps = append(expandedSteps, expanded...)
    }
    
    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandStageReferences: %d steps → %d expanded steps\n", 
            len(steps), len(expandedSteps))
    }
    
    return expandedSteps, nil
}

func (r *SimpleRunner) expandMatrixSteps(steps []Step) ([]Step, []MatrixStageJob, map[string]*Action, error) {
    var expandedSteps []Step
    var matrixJobs []MatrixStageJob
    allInterpolatedActions := make(map[string]*Action)

    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandMatrixSteps: processing %d steps\n", len(steps))
    }

    for i, step := range steps {
        if r.opts.Debug {
            fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Step %d: Action=%s, Matrix=%v\n", i, step.Action, step.Matrix != nil)
            if step.Matrix != nil {
                fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix values: %+v\n", step.Matrix.Values)
            }
        }

        // Check if this step has matrix configuration
        if step.Matrix != nil {
            // Handle matrix stage steps (step with both Stage and Matrix)
            if step.Stage != "" {
                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanding matrix stage step: %s\n", step.Stage)
                }

                // Get the referenced stage
                referencedStage, exists := r.config.GetStage(step.Stage)
                if !exists {
                    return nil, nil, nil, fmt.Errorf("stage not found: %s", step.Stage)
                }

                // Extract matrix variables from CLI variables
                cliMatrixVars := make(map[string]string)
                for key, value := range r.opts.Variables {
                    if strings.HasPrefix(key, "matrix.") {
                        cliMatrixVars[key] = value
                    }
                }

                // Create matrix expander with CLI matrix variables AND all global variables
                expander := NewMatrixExpander(r.config, cliMatrixVars, r.opts.Variables)

                // Generate matrix combinations
                matrixValues := make(map[string][]interface{})
                for key, values := range step.Matrix.Values {
                    matrixValues[key] = values
                }

                // Override with CLI-provided matrix values
                for cliKey, cliValue := range cliMatrixVars {
                    // Remove "matrix." prefix for key matching
                    keyWithoutPrefix := strings.TrimPrefix(cliKey, "matrix.")
                    if _, existsInMatrix := matrixValues[keyWithoutPrefix]; existsInMatrix {
                        matrixValues[keyWithoutPrefix] = []interface{}{cliValue}
                    }
                }

                combinations := expander.generateCombinations(matrixValues)

                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Generated %d matrix combinations for stage %s\n", len(combinations), step.Stage)
                }

                // Determine base name for pool ID and logging
                baseName := step.Stage
                if step.Name != "" {
                    baseName = step.Name
                }

                // For each matrix combination, expand the stage with matrix variables
                for jobIdx, combination := range combinations {
                    // Store start index to track steps added for this job
                    jobStartIdx := len(expandedSteps)
                    // Create matrix variables for this job (with "matrix." prefix)
                    matrixVars := make(map[string]string)
                    for key, value := range combination {
                        matrixVars[fmt.Sprintf("matrix.%s", key)] = fmt.Sprintf("%v", value)
                    }

                    // Merge with global variables (matrix vars override global)
                    jobVariables := make(map[string]string)
                    for k, v := range r.opts.Variables {
                        jobVariables[k] = v
                    }
                    for k, v := range matrixVars {
                        jobVariables[k] = v
                    }

                    // Temporarily set variables for stage expansion
                    originalVars := r.opts.Variables
                    r.opts.Variables = jobVariables

                    // Expand stage reference with matrix variables in context
                    expandedStageSteps, err := r.expandStageReferences(referencedStage.Steps)
                    if err != nil {
                        r.opts.Variables = originalVars
                        return nil, nil, nil, fmt.Errorf("failed to expand stage %s for matrix job %d: %w", step.Stage, jobIdx, err)
                    }

                    // Restore original variables
                    r.opts.Variables = originalVars

                    // Check if any expanded steps have matrix configurations and expand them
                    // This handles nested matrix scenarios where both the outer stage reference
                    // and inner steps have matrix configurations
                    if r.opts.Debug {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Checking %d expanded steps for nested matrix configurations\n", len(expandedStageSteps))
                        for i, s := range expandedStageSteps {
                            fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG]   Step %d: Action=%s, Stage=%s, HasMatrix=%v\n", i, s.Action, s.Stage, s.Matrix != nil)
                        }
                    }
                    var fullyExpandedSteps []Step
                    for _, expandedStep := range expandedStageSteps {
                        if expandedStep.Matrix != nil && expandedStep.Action != "" {
                            if r.opts.Debug {
                                fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanding nested matrix for step %s\n", expandedStep.Action)
                            }
                            // This step has a matrix configuration - expand it
                            action, exists := r.config.GetAction(expandedStep.Action)
                            if !exists {
                                return nil, nil, nil, fmt.Errorf("action not found: %s", expandedStep.Action)
                            }
                            
                            // Create a temporary expander with the job's matrix variables already set
                            tempExpander := NewMatrixExpander(r.config, cliMatrixVars, r.opts.Variables)
                            tempExpander.globalVars = jobVariables // Use job variables that include outer matrix vars
                            
                            nestedMatrixSteps, nestedInterpolatedActions, err := tempExpander.ExpandMatrixToStepsWithActions(&expandedStep, &action)
                            if err != nil {
                                return nil, nil, nil, fmt.Errorf("failed to expand nested matrix for step %s in job %d: %w", expandedStep.Action, jobIdx, err)
                            }
                            
                            // Add nested matrix variables to each expanded step's variables
                            // This ensures the outer matrix variables are preserved
                            for i := range nestedMatrixSteps {
                                if nestedMatrixSteps[i].Variables == nil {
                                    nestedMatrixSteps[i].Variables = make(map[string]string)
                                }
                                // Add outer matrix variables first
                                for k, v := range matrixVars {
                                    if _, exists := nestedMatrixSteps[i].Variables[k]; !exists {
                                        nestedMatrixSteps[i].Variables[k] = v
                                    }
                                }
                            }
                            
                            // Store interpolated actions with unique names to avoid collisions
                            // Each nested matrix step needs a unique action name across all jobs
                            for i, nestedStep := range nestedMatrixSteps {
                                // Original action name from nested expansion
                                originalActionName := nestedStep.Action
                                // Create unique action name: <baseName>.<jobIdx>.<originalActionName>
                                uniqueActionName := fmt.Sprintf("%s.%d.%s", baseName, jobIdx, originalActionName)
                                
                                // Find the corresponding interpolated action
                                if interpolatedAction, exists := nestedInterpolatedActions[originalActionName]; exists {
                                    // Create a copy of the action with the unique name
                                    uniqueAction := *interpolatedAction
                                    uniqueAction.Name = uniqueActionName
                                    
                                    // Store using the unique name
                                    allInterpolatedActions[uniqueActionName] = &uniqueAction
                                    
                                    // Update the step's Action field to match
                                    nestedMatrixSteps[i].Action = uniqueActionName
                                    
                                    if r.opts.Debug {
                                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Stored interpolated action: %s → %s\n", originalActionName, uniqueActionName)
                                    }
                                } else if r.opts.Debug {
                                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Warning: No interpolated action found for step %d: %s\n", i, originalActionName)
                                }
                            }
                            
                            fullyExpandedSteps = append(fullyExpandedSteps, nestedMatrixSteps...)
                        } else {
                            // No matrix on this step - keep as-is
                            fullyExpandedSteps = append(fullyExpandedSteps, expandedStep)
                        }
                    }
                    
                    // Replace expandedStageSteps with fully expanded steps
                    expandedStageSteps = fullyExpandedSteps

                    // Apply matrix variables to all expanded steps and ensure unique names
                    for stepIdx := range expandedStageSteps {
                        // Merge matrix variables into step variables
                        if expandedStageSteps[stepIdx].Variables == nil {
                            expandedStageSteps[stepIdx].Variables = make(map[string]string)
                        }
                        for k, v := range matrixVars {
                            expandedStageSteps[stepIdx].Variables[k] = v
                        }

                        // Ensure unique step names across matrix jobs
                        originalStepName := expandedStageSteps[stepIdx].GetStepName()
                        uniqueName := fmt.Sprintf("%s.%d.%s", baseName, jobIdx, originalStepName)

                        // Keep the Action field as-is (it's the key to find the interpolated action)
                        // Only update the Name field for uniqueness
                        if expandedStageSteps[stepIdx].Name == "" {
                            expandedStageSteps[stepIdx].Name = uniqueName
                        } else {
                            expandedStageSteps[stepIdx].Name = fmt.Sprintf("%s.%d.%s", baseName, jobIdx, expandedStageSteps[stepIdx].Name)
                        }

                        // Inherit properties from parent step
                        if step.OnError != "" && expandedStageSteps[stepIdx].OnError == "" {
                            expandedStageSteps[stepIdx].OnError = step.OnError
                        }
                        // Inherit if condition from parent step (combine with AND if child has one)
                        if step.If != "" {
                            if expandedStageSteps[stepIdx].If == "" {
                                expandedStageSteps[stepIdx].If = step.If
                            } else {
                                combinedIf := fmt.Sprintf("(%s) && (%s)", step.If, expandedStageSteps[stepIdx].If)
                                if r.opts.Debug {
                                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Combining if conditions:\n")
                                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG]   Parent: %s\n", step.If)
                                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG]   Child:  %s\n", expandedStageSteps[stepIdx].If)
                                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG]   Combined: %s\n", combinedIf)
                                }
                                expandedStageSteps[stepIdx].If = combinedIf
                            }
                        }
                    }

                    // Update dependencies within the same matrix job
                    stepNameMap := make(map[string]string)
                    for stepIdx := range expandedStageSteps {
                        originalStepName := expandedStageSteps[stepIdx].Action
                        if expandedStageSteps[stepIdx].Stage != "" {
                            originalStepName = expandedStageSteps[stepIdx].Stage
                        }
                        newStepName := expandedStageSteps[stepIdx].Name
                        stepNameMap[originalStepName] = newStepName
                    }

                    // Update Require and DependsOn to use new names
                    for stepIdx := range expandedStageSteps {
                        updatedRequire := make([]string, 0, len(expandedStageSteps[stepIdx].Require))
                        for _, dep := range expandedStageSteps[stepIdx].Require {
                            if newName, exists := stepNameMap[dep]; exists {
                                updatedRequire = append(updatedRequire, newName)
                            } else {
                                updatedRequire = append(updatedRequire, dep)
                            }
                        }
                        expandedStageSteps[stepIdx].Require = updatedRequire

                        updatedDependsOn := make([]string, 0, len(expandedStageSteps[stepIdx].DependsOn))
                        for _, dep := range expandedStageSteps[stepIdx].DependsOn {
                            if newName, exists := stepNameMap[dep]; exists {
                                updatedDependsOn = append(updatedDependsOn, newName)
                            } else {
                                updatedDependsOn = append(updatedDependsOn, dep)
                            }
                        }
                        expandedStageSteps[stepIdx].DependsOn = updatedDependsOn
                    }

                    // Assign pool ID if max_parallel is specified
                    strategy := step.Matrix.Strategy
                    if strategy.MaxParallel > 0 {
                        poolID := fmt.Sprintf("matrix-%s", baseName)
                        
                        // Assign pool ID to all expanded steps from this matrix job
                        for stepIdx := range expandedStageSteps {
                            expandedStageSteps[stepIdx].PoolID = poolID
                        }
                    }

                    // Add expanded steps for this matrix job
                    expandedSteps = append(expandedSteps, expandedStageSteps...)
                    
                    // Find first and last steps for this job and create MatrixStageJob
                    firstSteps := r.findFirstSteps(expandedStageSteps)
                    lastSteps := r.findLastSteps(expandedStageSteps)
                    jobSteps := expandedSteps[jobStartIdx:]
                    
                    matrixJob := MatrixStageJob{
                        Index:      jobIdx,
                        MatrixVars: matrixVars,
                        Steps:      jobSteps,
                        FirstSteps: firstSteps,
                        LastSteps:  lastSteps,
                    }
                    matrixJobs = append(matrixJobs, matrixJob)
                    
                    if r.opts.Debug {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix job %d: %d steps, first=%v, last=%v\n", 
                            jobIdx, len(expandedStageSteps), firstSteps, lastSteps)
                    }
                }
                
                // Log pool assignment after all jobs are processed
                strategy := step.Matrix.Strategy
                if strategy.MaxParallel > 0 && r.opts.Debug {
                    poolID := fmt.Sprintf("matrix-%s", baseName)
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Assigned matrix stage steps to pool %s (max_parallel=%d)\n",
                        poolID, strategy.MaxParallel)
                }

                continue
            }

            // Handle matrix action steps (step with Action and Matrix)
            if step.Action != "" {
                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanding matrix action step: %s\n", step.Action)
                }

                // Get the action for this step
                action, exists := r.config.GetAction(step.Action)
                if !exists {
                    return nil, nil, nil, fmt.Errorf("action not found: %s", step.Action)
                }

                // Debug: Print action details before expansion
                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Original action %s: Run=%q, Uses=%q\n", action.Name, action.Run, action.Uses)
                }

                // Extract matrix variables from CLI variables
                matrixVars := make(map[string]string)
                for key, value := range r.opts.Variables {
                    if strings.HasPrefix(key, "matrix.") {
                        matrixKey := strings.TrimPrefix(key, "matrix.")
                        matrixVars[matrixKey] = value
                    }
                }
                

                // Create matrix expander with CLI matrix variables AND all global variables
                expander := NewMatrixExpander(r.config, matrixVars, r.opts.Variables)

                // Expand matrix into individual steps with interpolated actions
                matrixSteps, interpolatedActions, err := expander.ExpandMatrixToStepsWithActions(&step, &action)
                if err != nil {
                    return nil, nil, nil, fmt.Errorf("failed to expand matrix for step %s: %w", step.Action, err)
                }
                
                // Debug: Print interpolated actions
                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Created %d interpolated actions\n", len(interpolatedActions))
                    for actionName, interpolatedAction := range interpolatedActions {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Interpolated action %s: Run=%q, Uses=%q\n", 
                            actionName, interpolatedAction.Run, interpolatedAction.Uses)
                    }
                }
                
                // Store interpolated actions in the overall map
                for actionName, interpolatedAction := range interpolatedActions {
                    allInterpolatedActions[actionName] = interpolatedAction
                }

                // Assign pool ID if max_parallel is specified
                strategy := step.Matrix.Strategy
                if strategy.MaxParallel > 0 {
                    poolID := fmt.Sprintf("matrix-%s", step.Action)
                    
                    // Assign pool ID to all expanded steps
                    for i := range matrixSteps {
                        matrixSteps[i].PoolID = poolID
                    }
                    
                    if r.opts.Debug {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Assigned %d matrix steps to pool %s (max_parallel=%d)\n",
                            len(matrixSteps), poolID, strategy.MaxParallel)
                    }
                }
                
                if r.opts.Debug {
                    fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Expanded into %d steps\n", len(matrixSteps))
                    for j, matrixStep := range matrixSteps {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Matrix step %d: Action=%s, PoolID=%s, Description=%s\n", 
                            j, matrixStep.Action, matrixStep.PoolID, matrixStep.Description)
                    }
                }

                // Add expanded steps
                expandedSteps = append(expandedSteps, matrixSteps...)
            }
        } else {
            // Regular step - add as is
            expandedSteps = append(expandedSteps, step)
        }
    }

    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] expandMatrixSteps: returning %d expanded steps, %d matrix jobs, %d interpolated actions\n", 
            len(expandedSteps), len(matrixJobs), len(allInterpolatedActions))
    }

    return expandedSteps, matrixJobs, allInterpolatedActions, nil
}

// findFirstSteps finds steps that can start the matrix job (no dependencies within the job)
func (r *SimpleRunner) findFirstSteps(steps []Step) []string {
    // Build a set of all step names in this job
    stepNames := make(map[string]bool)
    for _, step := range steps {
        stepNames[step.GetStepName()] = true
    }
    
    var firstSteps []string
    for _, step := range steps {
        stepName := step.GetStepName()
        // Check if this step has any dependencies within the job
        hasInternalDep := false
        for _, dep := range step.Require {
            if stepNames[dep] {
                hasInternalDep = true
                break
            }
        }
        for _, dep := range step.DependsOn {
            if stepNames[dep] {
                hasInternalDep = true
                break
            }
        }
        
        // If no internal dependencies, this is a first step
        if !hasInternalDep {
            firstSteps = append(firstSteps, stepName)
        }
    }
    
    // Handle edge cases:
    // - If no first steps found (all have dependencies): they depend on external steps, return all
    // - If all steps are first steps (no internal deps) AND no external deps: sequential, return first only
    if len(firstSteps) == 0 {
        // All steps have dependencies - assume external, return all
        for _, step := range steps {
            firstSteps = append(firstSteps, step.GetStepName())
        }
    } else if len(firstSteps) == len(steps) {
        // All steps are independent - check if they have ANY dependencies
        hasAnyDeps := false
        for _, step := range steps {
            if len(step.Require) > 0 || len(step.DependsOn) > 0 {
                hasAnyDeps = true
                break
            }
        }
        
        // If no dependencies at all, use sequential (first step only)
        if !hasAnyDeps && len(steps) > 0 {
            firstSteps = []string{steps[0].GetStepName()}
        }
        // Otherwise keep all (they have external deps)
    }
    
    return firstSteps
}

// findLastSteps finds steps that complete the matrix job (no dependents within the job)
func (r *SimpleRunner) findLastSteps(steps []Step) []string {
    // Build a reverse dependency graph: which steps depend on each step
    dependents := make(map[string][]string)
    stepNames := make(map[string]bool)
    
    for _, step := range steps {
        stepName := step.GetStepName()
        stepNames[stepName] = true
        dependents[stepName] = []string{}
    }
    
    for _, step := range steps {
        stepName := step.GetStepName()
        for _, dep := range step.Require {
            if stepNames[dep] {
                dependents[dep] = append(dependents[dep], stepName)
            }
        }
        for _, dep := range step.DependsOn {
            if stepNames[dep] {
                dependents[dep] = append(dependents[dep], stepName)
            }
        }
    }
    
    var lastSteps []string
    for _, step := range steps {
        stepName := step.GetStepName()
        // If this step has no dependents within the job, it's a last step
        if len(dependents[stepName]) == 0 {
            lastSteps = append(lastSteps, stepName)
        }
    }
    
    // Handle edge cases:
    // - If no last steps found OR all steps are last steps (no internal dependents),
    //   check if they have external dependents vs no dependents at all
    if len(lastSteps) == 0 || len(lastSteps) == len(steps) {
        if len(lastSteps) == len(steps) {
            // All steps are last steps - use declaration order for sequential execution
            if len(steps) > 0 {
                lastSteps = []string{steps[len(steps)-1].GetStepName()}
            }
        } else {
            // No last steps found (shouldn't happen) - return all
            for _, step := range steps {
                lastSteps = append(lastSteps, step.GetStepName())
            }
        }
    }
    
    return lastSteps
}

// injectSlidingWindowDependencies injects dependencies between matrix jobs based on max_parallel
func (r *SimpleRunner) injectSlidingWindowDependencies(steps []Step, matrixJobs []MatrixStageJob, maxParallel int) []Step {
    if maxParallel <= 0 || len(matrixJobs) <= maxParallel {
        // No dependencies needed
        return steps
    }
    
    // Create a map from step name to step index for quick lookup
    stepNameToIndex := make(map[string]int)
    for i, step := range steps {
        stepNameToIndex[step.GetStepName()] = i
    }
    
    // Inject dependencies: job i's first steps depend on job (i - maxParallel)'s last steps
    for i := maxParallel; i < len(matrixJobs); i++ {
        previousJobIdx := i - maxParallel
        previousJob := matrixJobs[previousJobIdx]
        currentJob := matrixJobs[i]
        
        // Find indices of first steps in current job
        var currentFirstIndices []int
        for _, firstStepName := range currentJob.FirstSteps {
            if idx, exists := stepNameToIndex[firstStepName]; exists {
                currentFirstIndices = append(currentFirstIndices, idx)
            }
        }
        
        // Add dependencies from previous job's last steps
        for _, firstIdx := range currentFirstIndices {
            firstStep := &steps[firstIdx]
            for _, lastStepName := range previousJob.LastSteps {
                // Check if dependency already exists
                hasDep := false
                for _, dep := range firstStep.Require {
                    if dep == lastStepName {
                        hasDep = true
                        break
                    }
                }
                for _, dep := range firstStep.DependsOn {
                    if dep == lastStepName {
                        hasDep = true
                        break
                    }
                }
                
                if !hasDep {
                    firstStep.Require = append(firstStep.Require, lastStepName)
                    
                    // Mark this dependency as a sliding window dependency
                    if firstStep.SlidingWindowDependencies == nil {
                        firstStep.SlidingWindowDependencies = make(map[string]bool)
                    }
                    firstStep.SlidingWindowDependencies[lastStepName] = true
                    
                    if r.opts.Debug {
                        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Injected sliding window dependency: %s depends on %s (job %d → job %d)\n",
                            firstStep.GetStepName(), lastStepName, i, previousJobIdx)
                    }
                }
            }
        }
    }
    
    return steps
}

// runStageWithHierarchicalDAG executes a stage using the hierarchical DAG architecture
func (r *SimpleRunner) runStageWithHierarchicalDAG(ctx context.Context, stageName string, stage Stage) error {
    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Using hierarchical DAG execution for stage: %s\n", stageName)
    }
    
    // Start timing
    stageStart := time.Now()
    timestamp := formatISO8601Timestamp(stageStart)
    fmt.Fprintf(r.opts.Output, "▶️  Running stage: %s [%s]\n\n", stageName, timestamp)
    
    // Build hierarchical DAG from stage steps
    dag, err := r.buildHierarchicalDAG(stage.Steps)
    if err != nil {
        return fmt.Errorf("failed to build hierarchical DAG: %w", err)
    }
    
    if r.opts.Debug {
        fmt.Fprintf(r.opts.ErrorOutput, "[DEBUG] Created hierarchical DAG with %d jobs and %d steps\n",
            dag.CountTotalJobs(), dag.CountTotalSteps())
    }
    
    // Create callback based on verbosity level
    var callback JobExecutionCallback
    var multilineCallback *MultilineJobCallback
    var orderedCallback *OrderedJobCallback
    
    if r.opts.VerboseLevel == 0 {
        // Quiet mode: use multiline callback (shows all steps upfront, updates dynamically)
        multilineCallback = NewMultilineJobCallback(dag, r.opts.Output, r.opts.ErrorOutput, r.opts.VerboseLevel, r.opts.Debug, r.config, r.opts.ConfigPath, r.opts.Variables)
        // Connect user-provided StepCallback if exists (mainly for tests)
        if r.opts.StepCallback != nil {
            multilineCallback.SetStepCallback(r.opts.StepCallback)
        }
        callback = multilineCallback
    } else {
        // Verbose mode: use ordered callback (shows steps in order as they complete)
        orderedCallback = NewOrderedJobCallback(dag, r.opts.Output, r.opts.ErrorOutput, r.opts.VerboseLevel, r.opts.Debug, r.config, r.opts.ConfigPath, r.opts.Variables)
        // Connect user-provided StepCallback if exists (mainly for tests)
        if r.opts.StepCallback != nil {
            orderedCallback.SetStepCallback(r.opts.StepCallback)
        }
        callback = orderedCallback
    }
    
    // Create RunOptions for executor
    execOpts := &RunOptions{
        ConfigPath:         r.opts.ConfigPath,
        MaxParallel:        r.opts.MaxParallel,
        VerboseLevel:       r.opts.VerboseLevel,
        Debug:              r.opts.Debug,
        DryRun:             r.opts.DryRun,
        Variables:          r.opts.Variables,
        WorkingDir:         r.opts.WorkingDir,
        Input:              r.opts.Input,
        Output:             r.opts.Output,
        ErrorOutput:        r.opts.ErrorOutput,
        Only:               r.opts.Only,
        WithRequires:       r.opts.WithRequires,
        BuildfabBinaryPath: r.opts.BuildfabBinaryPath,
    }
    
    // Initialize multiline display if using multiline callback
    if multilineCallback != nil {
        multilineCallback.Initialize()
        defer multilineCallback.Cleanup()
    }
    
    // Create and run hierarchical executor
    executor := NewHierarchicalExecutor(dag, r.config, execOpts, callback)
    // Transfer custom registry if exists (for tests)
    if r.registry != nil {
        executor.registry = r.registry
    }
    err = executor.Execute(ctx)
    
    // Calculate stage duration
    stageDuration := time.Since(stageStart)
    
    // Get results from callback
    var results []StepResult
    if multilineCallback != nil {
        results = multilineCallback.GetResults()
    } else if orderedCallback, ok := callback.(*OrderedJobCallback); ok {
        results = orderedCallback.GetResults()
    }
    
    // Print results
    r.printHierarchicalResults(stageName, stageDuration, results)
    
    return err
}

// buildHierarchicalDAG builds a hierarchical DAG from stage steps
func (r *SimpleRunner) buildHierarchicalDAG(steps []Step) (*HierarchicalDAG, error) {
    dag := NewHierarchicalDAG()
    
    // Extract matrix variables from CLI
    cliMatrixVars := make(map[string]string)
    for key, value := range r.opts.Variables {
        if strings.HasPrefix(key, "matrix.") {
            cliMatrixVars[key] = value
        }
    }
    
    // Create job expander
    expander := NewJobExpander(r.config, cliMatrixVars, r.opts.Variables)
    
    // Expand each step into jobs
    jobCounter := 0
    for _, step := range steps {
        // Check if step has matrix
        if step.Matrix != nil {
            // Expand matrix to jobs
            jobs, err := expander.ExpandMatrixToJobs(&step)
            if err != nil {
                return nil, fmt.Errorf("failed to expand matrix for step %s: %w", step.GetStepName(), err)
            }
            
            // Inject sliding window dependencies if needed
            if step.Matrix.Strategy.MaxParallel > 0 {
                expander.InjectSlidingWindowDependencies(jobs, step.Matrix.Strategy.MaxParallel)
            }
            
            // Add jobs to DAG
            for _, job := range jobs {
                dag.AddRootJob(job)
            }
            jobCounter += len(jobs)
        } else {
            // Non-matrix step: create a single job with one step
            // Use step name as job ID so dependencies can be resolved
            jobID := step.GetStepName()
            job := NewJobNode(jobID, step.GetStepName(), nil)
            
            if step.Action != "" {
                action, exists := r.config.GetAction(step.Action)
                if !exists {
                    return nil, fmt.Errorf("action not found: %s", step.Action)
                }
                
                execStepID := fmt.Sprintf("%s.0", jobID)
                execStep := ExecutableStep{
                    ID:          execStepID,
                    DisplayName: step.GetStepName(),
                    Action:      &action,
                    Variables:   step.Variables,
                    If:          step.If,
                    OnError:     step.OnError,
                }
                job.AddStep(execStep)
            }
            
            // Add user dependencies to job
            for _, dep := range step.Require {
                job.AddDependency(dep, false) // Not a sliding window dependency
            }
            for _, dep := range step.DependsOn {
                job.AddDependency(dep, false)
            }
            
            dag.AddRootJob(job)
            jobCounter++
        }
    }
    
    return dag, nil
}

// printHierarchicalResults prints execution results for hierarchical DAG
func (r *SimpleRunner) printHierarchicalResults(stageName string, duration time.Duration, results []StepResult) {
    // Count results by status
    statusCounts := make(map[StepStatus]int)
    for _, result := range results {
        statusCounts[result.Status]++
    }
    
    // Combine both skip types
    totalSkipped := statusCounts[StepStatusSkipped] + statusCounts[StepStatusSkippedCondition]
    
    // Determine overall status
    hasError := statusCounts[StepStatusError] > 0
    hasWarn := statusCounts[StepStatusWarn] > 0
    
    var statusIcon, statusText, statusColor string
    if hasError {
        statusIcon = "💥"
        statusText = "FAILED"
        statusColor = colorRed
    } else if hasWarn {
        statusIcon = "⚠️"
        statusText = "WARNING"
        statusColor = colorYellow
    } else {
        statusIcon = "🎉"
        statusText = "SUCCESS"
        statusColor = colorGreen
    }
    
    // Print summary
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    fmt.Fprintf(r.opts.Output, "%s %s%s%s - %s in %.3fs\n", statusIcon, statusColor, statusText, colorReset, stageName, duration.Seconds())
    fmt.Fprintf(r.opts.Output, "\n")
    fmt.Fprintf(r.opts.Output, "📊 Summary:\n")
    
    // Print status counts
    fmt.Fprintf(r.opts.Output, "   %s✗%s %serror%s      %d\n", 
        colorGray, colorReset, colorGray, colorReset, statusCounts[StepStatusError])
    fmt.Fprintf(r.opts.Output, "   %s!%s %swarn%s       %d\n", 
        colorGray, colorReset, colorGray, colorReset, statusCounts[StepStatusWarn])
    
    okCount := statusCounts[StepStatusOK]
    okColor := colorGray
    if okCount > 0 {
        okColor = colorGreen
    }
    fmt.Fprintf(r.opts.Output, "   %s✓%s %sok%s         %d\n", 
        okColor, colorReset, okColor, colorReset, okCount)
    
    fmt.Fprintf(r.opts.Output, "   %s→%s %sskipped%s    %d\n", 
        colorGray, colorReset, colorGray, colorReset, totalSkipped)
}
