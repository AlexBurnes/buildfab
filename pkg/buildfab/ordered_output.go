package buildfab

import (
    "context"
    "fmt"
    "io"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/AlexBurnes/buildfab/pkg/buildfab/container"
    containerRunner "github.com/AlexBurnes/buildfab/internal/container"
    "golang.org/x/term"
)

// formatISO8601Timestamp formats a time in ISO8601 format with fractional seconds
// Format: 2025-10-31T14:30:45.123456Z
func formatISO8601Timestamp(t time.Time) string {
    return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

// OrderedOutputManager manages step output in proper order using a queue-based approach
// This implements the architecture where:
// 1. Executor only runs tasks and reports to output manager, no direct output
// 2. Output manager has a queue and outputs steps in proper order
// 3. Step execution reports to manager, not UI directly
// 4. Manager outputs step start → step output → step result in sequence
type OrderedOutputManager struct {
    steps              []Step                           // Steps in declaration order
    stepData           map[string]*StepOutputData       // Buffered data for each step
    currentStep        string                           // Currently active step for output
    mu                 *sync.Mutex
    verboseLevel       int
    debug              bool
    errorOutput        io.Writer
    config             *Config                          // Configuration for command extraction
    configPath         string                           // Configuration file path for container commands
    interpolatedActions map[string]*Action              // Interpolated actions for matrix steps
    variables          map[string]string                // Global variables for interpolation
    stepVariables      map[string]map[string]string    // Step-specific variables for interpolation
    buildfabBinaryPath string                           // Path to buildfab binary for run_action/run_stage
}

// StepOutputData contains all output data for a step
type StepOutputData struct {
    Started        bool
    StartTime      time.Time // Actual time when step execution started
    Completed      bool
    Shown          bool  // Track if step start message has been shown
    Status         StepStatus
    Message        string
    Duration       time.Duration
    Output         []string
    Error          error
    BufferedOutput string // Store buffered stdout/stderr for quiet mode
}

// NewOrderedOutputManager creates a new ordered output manager
func NewOrderedOutputManager(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config) *OrderedOutputManager {
    return &OrderedOutputManager{
        steps:       steps,
        stepData:    make(map[string]*StepOutputData),
        mu:          &sync.Mutex{},
        verboseLevel: verboseLevel,
        debug:       debug,
        errorOutput: errorOutput,
        config:      config,
        interpolatedActions: make(map[string]*Action),
        stepVariables: make(map[string]map[string]string),
    }
}

// SetConfigPath sets the configuration file path
func (o *OrderedOutputManager) SetConfigPath(configPath string) {
    o.configPath = configPath
}

// SetVariables sets the global variables for interpolation
func (o *OrderedOutputManager) SetVariables(variables map[string]string) {
    o.variables = variables
}

// SetBuildfabBinaryPath sets the buildfab binary path for run_action/run_stage
func (o *OrderedOutputManager) SetBuildfabBinaryPath(path string) {
    o.buildfabBinaryPath = path
}

// SetStepVariables sets the step-specific variables for interpolation
func (o *OrderedOutputManager) SetStepVariables(stepName string, variables map[string]string) {
    o.mu.Lock()
    defer o.mu.Unlock()
    o.stepVariables[stepName] = variables
}

// SetInterpolatedAction sets the interpolated action for a step
func (o *OrderedOutputManager) SetInterpolatedAction(stepName string, action *Action) {
    o.mu.Lock()
    defer o.mu.Unlock()
    o.interpolatedActions[stepName] = action
}

// RegisterStep registers a step for execution
func (o *OrderedOutputManager) RegisterStep(stepName string) {
    o.mu.Lock()
    defer o.mu.Unlock()

    o.stepData[stepName] = &StepOutputData{}
}

// OnStepStart handles step start events from executor
func (o *OrderedOutputManager) OnStepStart(ctx context.Context, stepName string) {
    o.mu.Lock()
    
    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] OnStepStart: %s\n", stepName)
        o.debugPrintState()
    }

    // Capture actual start time when execution begins, not when displayed
    actualStartTime := time.Now()
    if data, exists := o.stepData[stepName]; exists {
        data.Started = true
        data.StartTime = actualStartTime
    } else {
        // Create entry if it doesn't exist
        o.stepData[stepName] = &StepOutputData{
            Started:   true,
            StartTime: actualStartTime,
        }
    }

    // Check if we can show step start (with mutex locked)
    shouldShow := o.canShowStepStart(stepName) && !o.stepData[stepName].Shown
    
    // Release mutex before calling display functions to avoid deadlock
    o.mu.Unlock()
    
    // Show step start message if ready (without holding mutex)
    if shouldShow {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Showing step start for: %s\n", stepName)
        }
        o.showStepStart(stepName)
        
        // Re-acquire mutex to update state
        o.mu.Lock()
        o.stepData[stepName].Shown = true
        o.currentStep = stepName
        o.mu.Unlock()
    } else {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Cannot show step start for: %s (not ready or already shown)\n", stepName)
        }
    }
}

// OnStepComplete handles step completion events from executor
func (o *OrderedOutputManager) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    o.mu.Lock()

    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] OnStepComplete: %s (status: %s)\n", stepName, status)
        o.debugPrintState()
    }

    if data, exists := o.stepData[stepName]; exists {
        data.Completed = true
        data.Status = status
        data.Message = message
        data.Duration = duration
        data.BufferedOutput = bufferedOutput
    }

    // Show step completion message if this is the current step
    if o.currentStep == stepName {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Showing step completion for: %s\n", stepName)
        }
        // Flush any buffered output for this step before showing completion
        o.flushBufferedOutput(stepName)
        o.showStepCompletion(stepName)
        o.currentStep = ""
        // Mark this step as shown so it doesn't get processed again in checkAndShowCompletedSteps
        if data, exists := o.stepData[stepName]; exists {
            data.Shown = true
        }
    }

    // Release mutex BEFORE checking for next steps to avoid deadlock
    // (checkAndShowCompletedSteps and checkAndShowNextStep may call showStepStart
    //  which does I/O and may try to acquire the mutex again)
    o.mu.Unlock()

    // Check if any completed steps can now be shown in order
    o.checkAndShowCompletedSteps()

    // Check if next step can be shown
    o.checkAndShowNextStep()
}

// OnStepOutput handles step output events from executor
func (o *OrderedOutputManager) OnStepOutput(ctx context.Context, stepName string, output string) {
    // Determine what to do while holding the lock, then release before I/O
    var shouldStream bool
    var linesToPrint []string
    var debugMsg string
    
    o.mu.Lock()
    
    // For matrix steps, show output immediately if it's the current step
    // This enables real-time streaming for matrix execution
    shouldStream = o.shouldStreamOutput(stepName)
    
    if shouldStream {
        // Prepare lines to print (but don't print yet - release lock first)
        lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
        for _, line := range lines {
            if line != "" {
                linesToPrint = append(linesToPrint, line)
            }
        }
    } else {
        // Buffer output for later display when step completes
        if data, exists := o.stepData[stepName]; exists {
            data.Output = append(data.Output, output)
            if o.debug {
                debugMsg = fmt.Sprintf("[DEBUG] Buffered output for %s: %s\n", stepName, output)
            }
        }
    }
    
    o.mu.Unlock()
    
    // Now do I/O operations WITHOUT holding the lock to avoid deadlock
    if shouldStream {
        for _, line := range linesToPrint {
            fmt.Fprintf(o.errorOutput, "    %s\n", line)
        }
    } else if debugMsg != "" {
        fmt.Fprint(o.errorOutput, debugMsg)
    }
}

// shouldStreamOutput checks if the given step should have its output streamed
func (o *OrderedOutputManager) shouldStreamOutput(stepName string) bool {
    // Find the step in declaration order
    stepIndex := -1
    for i, step := range o.steps {
        if step.GetStepName() == stepName {
            stepIndex = i
            break
        }
    }

    if stepIndex == -1 {
        return false
    }

    // Only allow streaming for the first step in declaration order that hasn't been completed yet
    // Check if all previous steps in declaration order have been completed
    for i := 0; i < stepIndex; i++ {
        if data, exists := o.stepData[o.steps[i].GetStepName()]; !exists || !data.Completed {
            return false
        }
    }

    // Check if this step itself has been completed - if so, don't stream
    if data, exists := o.stepData[stepName]; exists && data.Completed {
        return false
    }

    return true
}

// OnStepError handles step error events from executor
func (o *OrderedOutputManager) OnStepError(ctx context.Context, stepName string, err error) {
    o.mu.Lock()
    defer o.mu.Unlock()

    if data, exists := o.stepData[stepName]; exists {
        data.Error = err
    }
}

// canShowStepStart checks if a step can show its start message
func (o *OrderedOutputManager) canShowStepStart(stepName string) bool {
    // Find step index in declaration order
    stepIndex := -1
    for i, step := range o.steps {
        if step.GetStepName() == stepName {
            stepIndex = i
            break
        }
    }

    if stepIndex == -1 {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: step %s not found in steps\n", stepName)
        }
        return false
    }

    // First step can always show start
    if stepIndex == 0 {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s is first step, can show start\n", stepName)
        }
        return true
    }

    // Check if all previous steps have been completed
    for i := 0; i < stepIndex; i++ {
        prevStepName := o.steps[i].GetStepName()
        if data, exists := o.stepData[prevStepName]; !exists || !data.Completed {
            if o.debug {
                fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s cannot show start, previous step %s not completed (exists: %v, completed: %v)\n",
                    stepName, prevStepName, exists, exists && data.Completed)
            }
            return false
        }
    }

    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] canShowStepStart: %s can show start, all previous steps completed\n", stepName)
    }
    return true
}

// checkAndShowCompletedSteps checks if any completed steps can now be shown in order
func (o *OrderedOutputManager) checkAndShowCompletedSteps() {
    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowCompletedSteps: checking for completed steps to show\n")
    }

    // Show all steps that can be shown in order
    shownAny := true
    for shownAny {
        shownAny = false
        
        // Acquire mutex to check state
        o.mu.Lock()
        var stepToShow string
        for _, step := range o.steps {
            stepName := step.GetStepName()
            if data, exists := o.stepData[stepName]; exists && data.Completed && !data.Shown {
                // Check if all previous steps have been completed AND shown
                canShow := true
                for _, s := range o.steps {
                    if s.GetStepName() == stepName {
                        break
                    }
                    prevData, prevExists := o.stepData[s.GetStepName()]
                    if !prevExists || !prevData.Completed || !prevData.Shown {
                        canShow = false
                        break
                    }
                }

                if canShow {
                    stepToShow = stepName
                    o.stepData[stepName].Shown = true
                    o.currentStep = stepName
                    shownAny = true
                    break // Start over to check for more steps that can now be shown
                }
            }
        }
        o.mu.Unlock()
        
        // Show step WITHOUT holding mutex to avoid deadlock
        if stepToShow != "" {
            if o.debug {
                fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowCompletedSteps: showing completed step: %s\n", stepToShow)
            }
            o.showStepStart(stepToShow)

            // Re-acquire mutex for state updates
            o.mu.Lock()
            // Flush any buffered output for this step
            o.flushBufferedOutput(stepToShow)
            o.mu.Unlock()

            o.showStepCompletion(stepToShow)

            // Re-acquire mutex for final state updates
            o.mu.Lock()
            // Flush any remaining buffered output after step completion
            o.flushBufferedOutput(stepToShow)
            o.currentStep = ""
            o.mu.Unlock()
        }
    }
}

// checkAndShowNextStep checks if the next step can be shown
func (o *OrderedOutputManager) checkAndShowNextStep() {
    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: checking for next step to show\n")
    }

    // Acquire mutex to check state
    o.mu.Lock()
    var stepToShow string
    
    // Find the next step that can be shown
    for _, step := range o.steps {
        stepName := step.GetStepName()
        if data, exists := o.stepData[stepName]; exists && data.Started && !data.Completed && !data.Shown {
            if o.canShowStepStart(stepName) {
                stepToShow = stepName
                o.stepData[stepName].Shown = true
                o.currentStep = stepName
                break
            }
        }
    }
    o.mu.Unlock()
    
    // Show step WITHOUT holding mutex to avoid deadlock
    if stepToShow != "" {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: showing next step: %s\n", stepToShow)
        }
        o.showStepStart(stepToShow)

        // Re-acquire mutex to flush buffered output
        o.mu.Lock()
        // Flush any buffered output for this step
        o.flushBufferedOutput(stepToShow)
        o.mu.Unlock()
    }
}

// showStepStart shows the start message for a step
func (o *OrderedOutputManager) showStepStart(stepName string) {
    if o.verboseLevel > 0 {
        // Use actual start time captured when execution began, not display time
        var timestamp string
        if data, exists := o.stepData[stepName]; exists && !data.StartTime.IsZero() {
            timestamp = formatISO8601Timestamp(data.StartTime)
        } else {
            // Fallback to current time if start time not captured yet
            timestamp = formatISO8601Timestamp(time.Now())
        }
        fmt.Fprintf(o.errorOutput, "  💻 %s [%s]\n", stepName, timestamp)

        // Show container command for container actions (verbose level 2+)
        if o.verboseLevel >= 2 {
            o.showContainerCommand(stepName)
        }
    } else {
        // In quiet mode, show a simple running indicator that will be overwritten by completion message
        fmt.Fprintf(o.errorOutput, "  %s◯%s %s (running...)\r", colorCyan, colorReset, stepName)
    }
}

// showContainerCommand shows the container command for container actions
func (o *OrderedOutputManager) showContainerCommand(stepName string) {
    // Check if we have an interpolated action for this step
    action, exists := o.interpolatedActions[stepName]
    if !exists {
        // Fallback to base action name for non-matrix steps
        baseActionName := stepName
        if dotIndex := strings.Index(stepName, "."); dotIndex != -1 {
            baseActionName = stepName[:dotIndex]
        }
        var ok bool
        actionValue, ok := o.config.GetAction(baseActionName)
        if !ok {
            return
        }
        action = &actionValue
    }

    if action.Container == nil {
        return
    }
    
    // Create a temporary runner to prepare the configuration for display
    tempRunner, err := containerRunner.NewContainerRunnerWithVerbosity(o.verboseLevel)
    if err != nil {
        // Silently ignore runner creation errors for display purposes
        return
    }
    
    // Set the buildfab binary path for run_action/run_stage
    if o.buildfabBinaryPath != "" {
        tempRunner.SetBuildfabPath(o.buildfabBinaryPath)
    }
    
    // Use the actual config path
    configPath := o.configPath
    if configPath == "" {
        configPath = ".project.yml" // Fallback
    }
    
    // Use the already-interpolated container configuration from the action
    // For matrix steps, the action.Container already has matrix variables substituted
    preparedConfig, prepErr := tempRunner.PrepareContainerConfig(*action.Container, configPath)
    if prepErr != nil {
        // Silently ignore preparation errors for display purposes
        // (artifacts might not be mounted yet, which is OK for display)
        return
    }
    
    // Interpolate variables based on step type:
    // 1. Matrix steps (exists == true): already fully interpolated, no additional interpolation needed
    // 2. Steps with step variables: use step-specific variables
    // 3. Regular steps: use global variables
    if !exists {
        // Check if this step has step-specific variables
        o.mu.Lock()
        stepVars, hasStepVars := o.stepVariables[stepName]
        o.mu.Unlock()
        
        var varsToUse map[string]string
        if hasStepVars {
            // Use step-specific variables (which should already be merged with global vars)
            varsToUse = stepVars
        } else if o.variables != nil {
            // Use global variables
            varsToUse = o.variables
        }
        
        if varsToUse != nil {
            interpolatedConfig, err := InterpolateContainerConfig(&preparedConfig, varsToUse)
            if err == nil {
                preparedConfig = *interpolatedConfig
            }
        }
    }
    
    containerCmd := o.buildContainerCommand(&preparedConfig)
    
    // Determine the appropriate prefix based on the container operation type
    prefix := "Running container"
    if preparedConfig.Image.Build != nil {
        prefix = "Building image"
    } else if preparedConfig.Image.Slim != nil {
        prefix = "Slimming image"
    }
    
    fmt.Fprintf(o.errorOutput, "  🐳 %s: %s\n", prefix, containerCmd)
}

// buildContainerCommand builds a human-readable representation of the container command
func (o *OrderedOutputManager) buildContainerCommand(config *container.ContainerConfig) string {
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
        // Regular container execution
        parts = append(parts, "run", "--rm")

        hasWorkspaceMount := false

        // Add environment variables
        for key, value := range config.Env {
            parts = append(parts, "-e", fmt.Sprintf("\"%s=%s\"", key, value))
        }

        // Add mount arguments
        for _, mount := range config.Mounts {
            if mount.Target == "/tmp/buildfab-workspace" {
                hasWorkspaceMount = true
            }
            mountArg := fmt.Sprintf("--mount=type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target)
            if mount.RO {
                mountArg += ",readonly"
            }
            parts = append(parts, mountArg)
        }

        if hasWorkspaceMount {
            parts = append(parts, "-w", "/tmp/buildfab-workspace")
        }

        // Add cache mounts
        for cacheName, cachePath := range config.Cache {
            targetPath := fmt.Sprintf("/tmp/buildfab-cache-%s", cacheName)
            cacheMountArg := fmt.Sprintf("--mount=type=bind,source=%s,target=%s", cachePath, targetPath)
            parts = append(parts, cacheMountArg)
        }

        // Add CPU and memory limits
        if config.CPU > 0 {
            parts = append(parts, "--cpus", fmt.Sprintf("%d.0", config.CPU))

            // Generate CPU set: 2 -> "0,1", 3 -> "0,1,2", etc.
            cpuSet := ""
            for i := 0; i < config.CPU; i++ {
                if i > 0 {
                    cpuSet += ","
                }
                cpuSet += fmt.Sprintf("%d", i)
            }
            parts = append(parts, "--cpuset-cpus", cpuSet)
        }

        if config.Memory != "" {
            parts = append(parts, "-m", config.Memory)
        }

        // Add user if specified
        if config.User != "" {
            parts = append(parts, "-u", config.User)
        }

        // Add network if specified
        if config.Network != "" {
            parts = append(parts, "--network", config.Network)
        }

        // Add image
        parts = append(parts, config.Image.From)

        // Add command to run
        if config.Run != "" {
            parts = append(parts, "sh", "-c", "'", config.Run, "'")
        } else if config.RunAction != "" {
            parts = append(parts, "buildfab", "action", config.RunAction)
        } else if config.RunStage != "" {
            parts = append(parts, "buildfab", "run", config.RunStage)
        }
    }

    return strings.Join(parts, " ")
}

// showStepCompletion shows the completion message for a step
func (o *OrderedOutputManager) showStepCompletion(stepName string) {
    if data, exists := o.stepData[stepName]; exists {
        // In quiet mode (level 0), show buffered output on failure/warning before the result message
        if o.verboseLevel == 0 && (data.Status == StepStatusError || data.Status == StepStatusWarn) && data.BufferedOutput != "" {
            // Show step name and "execute failure" message first
            var icon, color string
            if data.Status == StepStatusError {
                icon = "✗"
                color = colorRed
            } else { // StepStatusWarn
                icon = "!"
                color = colorYellow
            }
            // Clear the running indicator line before showing failure message
            fmt.Fprintf(o.errorOutput, "\033[2K\r  %s%s%s %s execute failure\n", color, icon, colorReset, stepName)

            // Display the buffered output (which contains stdout/stderr)
            lines := strings.Split(strings.TrimRight(data.BufferedOutput, "\n"), "\n")
            for _, line := range lines {
                fmt.Fprintf(o.errorOutput, "    %s\n", line)
            }

            // Show "to check run:" and command lines only for level 3 (vvv)
            if o.verboseLevel == 3 {
                fmt.Fprintf(o.errorOutput, "    🔧 to check run:\n")
                // Use extractCommand to properly format the command with correct indentation
                command := o.extractCommand(stepName, data.Message)
                fmt.Fprintf(o.errorOutput, "%s\n", command)
            }
        } else {
            // For other cases (success, verbose mode), show the normal result message
            o.showStepResult(stepName, data.Status, data.Message, data.Duration)
        }
    }
}

// showStepOutput shows step output
func (o *OrderedOutputManager) showStepOutput(output string) {
    if o.verboseLevel > 0 {
        fmt.Fprintf(o.errorOutput, "    %s\n", output)
    }
}

// flushBufferedOutput flushes all buffered output for a step
func (o *OrderedOutputManager) flushBufferedOutput(stepName string) {
    if data, exists := o.stepData[stepName]; exists {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Flushing %d buffered outputs for %s\n", len(data.Output), stepName)
        }
        for _, output := range data.Output {
            o.showStepOutput(output)
        }
        // Clear the buffered output after flushing
        data.Output = nil
    }
}

// showStepResult shows the result message for a step
func (o *OrderedOutputManager) showStepResult(stepName string, status StepStatus, message string, duration time.Duration) {
    // Enhance error messages with reproduction instructions
    enhancedMessage := o.enhanceMessage(stepName, status, message)

    // Add execution time for successful actions
    executionTime := formatExecutionTime(status, duration)
    enhancedMessage += executionTime

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
    case StepStatusSkipped:
        icon = "→"
        color = colorGray
    default:
        icon = "?"
        color = colorGray
    }

    if o.verboseLevel > 0 {
        // In verbose mode, just print the result
        fmt.Fprintf(o.errorOutput, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, enhancedMessage)
    } else {
        // In quiet mode, clear the running indicator line and show completion
        // Use ANSI escape code to clear line, then carriage return to start of line
        fmt.Fprintf(o.errorOutput, "\033[2K\r  %s%s%s %s %s\n", color, icon, colorReset, stepName, enhancedMessage)
    }
}

// OrderedStepCallback implements StepCallback interface using the ordered output manager
type OrderedStepCallback struct {
    manager *OrderedOutputManager
    results []StepResult
    mu      *sync.Mutex
}

// NewOrderedStepCallback creates a new ordered step callback
func NewOrderedStepCallback(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config) *OrderedStepCallback {
    manager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)

    // Register all steps using their unique names
    for _, step := range steps {
        manager.RegisterStep(step.GetStepName())
    }

    return &OrderedStepCallback{
        manager: manager,
        results: make([]StepResult, 0),
        mu:      &sync.Mutex{},
    }
}

// NewOrderedStepCallbackWithActions creates a new ordered step callback with interpolated actions
func NewOrderedStepCallbackWithActions(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config, configPath string, interpolatedActions map[string]*Action) *OrderedStepCallback {
    manager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)

    // Set config path
    manager.SetConfigPath(configPath)

    // Set interpolated actions
    for stepName, action := range interpolatedActions {
        manager.SetInterpolatedAction(stepName, action)
    }

    // Register all steps using their unique names
    for _, step := range steps {
        manager.RegisterStep(step.GetStepName())
    }

    return &OrderedStepCallback{
        manager: manager,
        results: make([]StepResult, 0),
        mu:      &sync.Mutex{},
    }
}

// UpdateInterpolatedActions updates the interpolated actions from RunOptions
func (c *OrderedStepCallback) UpdateInterpolatedActions(interpolatedActions map[string]*Action) {
    for stepName, action := range interpolatedActions {
        c.manager.SetInterpolatedAction(stepName, action)
    }
}

// OnStepStart implements StepCallback interface
func (c *OrderedStepCallback) OnStepStart(ctx context.Context, stepName string) {
    c.manager.OnStepStart(ctx, stepName)
}

// OnStepComplete implements StepCallback interface
func (c *OrderedStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    c.manager.OnStepComplete(ctx, stepName, status, message, duration, bufferedOutput)

    // Collect result for summary (thread-safe)
    c.mu.Lock()
    c.results = append(c.results, StepResult{
        StepName: stepName,
        Status:   status,
        Duration: duration,
    })
    c.mu.Unlock()
}

// OnStepOutput implements StepCallback interface
func (c *OrderedStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    c.manager.OnStepOutput(ctx, stepName, output)
}

// OnStepError implements StepCallback interface
func (c *OrderedStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
    c.manager.OnStepError(ctx, stepName, err)
}

// GetResults returns the collected step results
func (c *OrderedStepCallback) GetResults() []StepResult {
    c.mu.Lock()
    defer c.mu.Unlock()
    // Return a copy to avoid race conditions
    result := make([]StepResult, len(c.results))
    copy(result, c.results)
    return result
}

// debugPrintState prints the current state of the output manager
func (o *OrderedOutputManager) debugPrintState() {
    fmt.Fprintf(o.errorOutput, "[DEBUG] Output Manager State:\n")
    fmt.Fprintf(o.errorOutput, "  Current Step: %s\n", o.currentStep)
    fmt.Fprintf(o.errorOutput, "  Steps in order: ")
    for i, step := range o.steps {
        if i > 0 {
            fmt.Fprintf(o.errorOutput, ", ")
        }
        fmt.Fprintf(o.errorOutput, "%s", step.GetStepName())
    }
    fmt.Fprintf(o.errorOutput, "\n")
    fmt.Fprintf(o.errorOutput, "  Step Data:\n")
    for stepName, data := range o.stepData {
        fmt.Fprintf(o.errorOutput, "    %s: started=%v, completed=%v, status=%s\n",
            stepName, data.Started, data.Completed, data.Status)
    }
}

// enhanceMessage improves error messages to match v0.5.0 style
func (o *OrderedOutputManager) enhanceMessage(stepName string, status StepStatus, message string) string {
    switch status {
    case StepStatusError, StepStatusWarn:
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
                command := o.extractCommand(stepName, strings.Join(reproductionLines, "\n"))
                if o.verboseLevel == 3 {
                    return fmt.Sprintf("execute failure\n    🔧 to check run:\n%s", command)
                } else {
                    return "execute failure"
                }
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
                command := o.extractCommand(stepName, message)
                if o.verboseLevel == 3 {
                    return fmt.Sprintf("execute failure\n    🔧 to check run:\n%s", command)
                } else {
                    return "execute failure"
                }
            }
        }
        // Enhance error messages with reproduction instructions
        if strings.Contains(message, "command failed: exit status") {
            // Extract the command from the step name or message
            command := o.extractCommand(stepName, message)
            if o.verboseLevel == 3 {
                return fmt.Sprintf("execute failure\n    🔧 to check run:\n%s", command)
            } else {
                return "execute failure"
            }
        }
        return message
    case StepStatusSkipped:
        // Enhance skipped messages with dependency information
        if strings.Contains(message, "dependency failed") {
            // Try to extract which dependency failed
            failedDep := o.extractFailedDependency(stepName)
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
func (o *OrderedOutputManager) extractCommand(stepName, message string) string {
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
    
    if o.config == nil {
        return fmt.Sprintf("      %s", stepName)
    }

    // Try to find the action and extract the actual command
    action, exists := o.config.GetAction(stepName)
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

        // If all lines were empty, use 0
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
func (o *OrderedOutputManager) extractFailedDependency(stepName string) string {
    // Look through results to find failed dependencies
    for _, data := range o.stepData {
        if data.Status == StepStatusError {
            // This is a simple heuristic - in a real implementation,
            // we'd need to track the dependency graph
            return stepName
        }
    }
    return ""
}

// ============================================================================
// MultilineOutputManager - New multiline display system for quiet mode
// ============================================================================

// ANSI escape codes for terminal control
const (
    hideCursor     = "\x1b[?25l" // Hide cursor
    showCursor     = "\x1b[?25h" // Show cursor
    clearLine      = "\x1b[2K"   // Clear current line
    saveCursor     = "\x1b7"     // Save cursor position
    restoreCursor  = "\x1b8"     // Restore cursor position
    getCursorPos   = "\x1b[6n"   // Get cursor position (returns \x1b[ROW;COL R)
)

// moveTo moves cursor to specific row and column (1-based)
func moveTo(row, col int) string {
    return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// moveUp moves cursor up by n lines
func moveUp(n int) string {
    if n <= 0 {
        return ""
    }
    return fmt.Sprintf("\x1b[%dA", n)
}

// moveDown moves cursor down by n lines
func moveDown(n int) string {
    if n <= 0 {
        return ""
    }
    return fmt.Sprintf("\x1b[%dB", n)
}

// moveToColumn moves cursor to specific column (1-based)
func moveToColumn(col int) string {
    return fmt.Sprintf("\x1b[%dG", col)
}

// JobStatus represents the status of a job in multiline display
type JobStatus int

const (
    JobStatusPending JobStatus = iota
    JobStatusRunning
    JobStatusSuccess
    JobStatusWarning
    JobStatusError
    JobStatusSkipped
)

// JobDisplay represents a job in the multiline display
type JobDisplay struct {
    Name        string        // Job name
    Status      JobStatus     // Current status
    Message     string        // Status message
    Duration    time.Duration // Execution duration
    Row         int           // Display row number (1-based)
    Started     bool          // Whether job has started
    Completed   bool          // Whether job has completed
    StartTime   time.Time     // When job started
}

// MultilineOutputManager manages multiline job status display using ANSI escape codes
type MultilineOutputManager struct {
    jobs              []JobDisplay             // Jobs in declaration order
    jobMap            map[string]*JobDisplay   // Fast lookup by job name
    verboseLevel      int                      // Verbosity level
    debug             bool                     // Debug mode
    errorOutput       io.Writer                // Output writer
    config            *Config                  // Configuration
    configPath        string                   // Configuration file path
    interpolatedActions map[string]*Action     // Interpolated actions for matrix steps
    mu                *sync.Mutex              // Mutex for thread safety
    isTTY             bool                     // Whether output is to a terminal
    initialized       bool                     // Whether display has been initialized
    lastDisplay       string                   // Last displayed content for comparison
}

// NewMultilineOutputManager creates a new multiline output manager
func NewMultilineOutputManager(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config) *MultilineOutputManager {
    // Check if output is to a terminal
    isTTY := false
    if file, ok := errorOutput.(*os.File); ok {
        isTTY = term.IsTerminal(int(file.Fd()))
    }

    // Create job displays from steps
    jobs := make([]JobDisplay, len(steps))
    jobMap := make(map[string]*JobDisplay)
    
    for i, step := range steps {
        stepName := step.GetStepName()
        job := JobDisplay{
            Name:     stepName,
            Status:   JobStatusPending,
            Message:  "(pending)",
            Row:      i + 1, // 1-based row numbering
            Started:  false,
            Completed: false,
        }
        jobs[i] = job
        jobMap[stepName] = &jobs[i]
    }

    return &MultilineOutputManager{
        jobs:              jobs,
        jobMap:            jobMap,
        verboseLevel:      verboseLevel,
        debug:             debug,
        errorOutput:       errorOutput,
        config:            config,
        interpolatedActions: make(map[string]*Action),
        mu:                &sync.Mutex{},
        isTTY:             isTTY,
        initialized:       false,
    }
}

// SetConfigPath sets the configuration file path
func (m *MultilineOutputManager) SetConfigPath(configPath string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.configPath = configPath
}

// SetInterpolatedAction sets the interpolated action for a job
func (m *MultilineOutputManager) SetInterpolatedAction(jobName string, action *Action) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.interpolatedActions[jobName] = action
}

// InitializeDisplay initializes the multiline display
func (m *MultilineOutputManager) InitializeDisplay() {
    m.mu.Lock()
    defer m.mu.Unlock()

    if !m.isTTY || m.initialized {
        return
    }

    if m.debug {
        fmt.Fprintf(m.errorOutput, "[DEBUG] Initializing multiline display for %d jobs\n", len(m.jobs))
    }

    // Hide cursor
    fmt.Fprint(m.errorOutput, hideCursor)
    
    // Initial display
    m.redrawDisplay()
    
    m.initialized = true
    
    if m.debug {
        fmt.Fprintf(m.errorOutput, "[DEBUG] Multiline display initialized with %d jobs\n", len(m.jobs))
    }
}

// UpdateJobStatus updates the status of a specific job (called via callback system from DAG executor)
func (m *MultilineOutputManager) UpdateJobStatus(jobName string, status JobStatus, message string, duration time.Duration) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if !m.isTTY || !m.initialized {
        return
    }

    job, exists := m.jobMap[jobName]
    if !exists {
        if m.debug {
            fmt.Fprintf(m.errorOutput, "[DEBUG] Job %s not found in multiline display\n", jobName)
        }
        return
    }

    if m.debug {
        fmt.Fprintf(m.errorOutput, "[DEBUG] Updating job %s: status=%v, message=%s, duration=%v\n", jobName, status, message, duration)
    }

    // Update job data
    job.Status = status
    job.Message = message
    job.Duration = duration
    if !job.Started {
        // Capture start time when job first starts
        job.StartTime = time.Now()
    }
    job.Started = true
    
    if status == JobStatusSuccess || status == JobStatusWarning || status == JobStatusError || status == JobStatusSkipped {
        job.Completed = true
    }

    // Redraw the entire display
    m.redrawDisplay()
}

// redrawDisplay redraws the entire multiline display
func (m *MultilineOutputManager) redrawDisplay() {
    if !m.isTTY {
        return
    }

    // Build the display content
    var display strings.Builder
    for i := range m.jobs {
        job := &m.jobs[i]
        // Get status icon and color
        icon, color := m.getStatusIconAndColor(job.Status)
        
        // Format duration if provided
        durationStr := ""
        if job.Duration > 0 {
            durationStr = fmt.Sprintf(" - in '%.3fs'", job.Duration.Seconds())
        }
        
        // Format timestamp if job has started
        timestampStr := ""
        if job.Started && !job.StartTime.IsZero() {
            timestampStr = " [" + formatISO8601Timestamp(job.StartTime) + "]"
        }
        
        // Build the job line
        line := fmt.Sprintf("  %s%s%s %s %s%s%s", color, icon, colorReset, job.Name, job.Message, durationStr, timestampStr)
        display.WriteString(line)
        display.WriteString("\n")
    }
    
    newDisplay := display.String()
    
    // Only redraw if content has changed
    if newDisplay != m.lastDisplay {
        // Clear previous display by moving up and clearing lines
        if m.lastDisplay != "" {
            lines := strings.Count(m.lastDisplay, "\n")
            if lines > 0 {
                fmt.Fprint(m.errorOutput, moveUp(lines))
            }
        }
        
        // Draw new content
        fmt.Fprint(m.errorOutput, newDisplay)
        m.lastDisplay = newDisplay
        
        // Flush output
        if f, ok := m.errorOutput.(interface{ Flush() }); ok {
            f.Flush()
        }
    }
}

// getStatusIconAndColor returns the appropriate icon and color for a job status
func (m *MultilineOutputManager) getStatusIconAndColor(status JobStatus) (string, string) {
    switch status {
    case JobStatusPending:
        return "○", colorGray
    case JobStatusRunning:
        return "◯", colorCyan
    case JobStatusSuccess:
        return "✓", colorGreen
    case JobStatusWarning:
        return "!", colorYellow
    case JobStatusError:
        return "✗", colorRed
    case JobStatusSkipped:
        return "→", colorGray
    default:
        return "?", colorGray
    }
}

// Cleanup cleans up the multiline display
func (m *MultilineOutputManager) Cleanup() {
    m.mu.Lock()
    defer m.mu.Unlock()

    if !m.isTTY || !m.initialized {
        return
    }

    if m.debug {
        fmt.Fprintf(m.errorOutput, "[DEBUG] Cleaning up multiline display\n")
    }

    // Show cursor
    fmt.Fprint(m.errorOutput, showCursor)
    
    // Don't clear the display - let it remain visible
    // Just ensure we're positioned correctly for subsequent output
    fmt.Fprint(m.errorOutput, "\n")
    
    m.initialized = false
}

// IsEnabled returns whether multiline display is enabled
func (m *MultilineOutputManager) IsEnabled() bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.isTTY && m.verboseLevel == 0 // Only enabled in quiet mode (level 0)
}

// GetJobStatus returns the current status of a job
func (m *MultilineOutputManager) GetJobStatus(jobName string) (JobStatus, bool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    job, exists := m.jobMap[jobName]
    if !exists {
        return JobStatusPending, false
    }
    
    return job.Status, true
}

// ============================================================================
// MultilineStepCallback - StepCallback implementation using MultilineOutputManager
// ============================================================================

// MultilineStepCallback implements StepCallback interface using the multiline output manager
type MultilineStepCallback struct {
    manager        *MultilineOutputManager
    results        []StepResult
    mu             *sync.Mutex
    fallbackManager *OrderedOutputManager // Fallback for verbose mode or non-TTY
}

// NewMultilineStepCallback creates a new multiline step callback
func NewMultilineStepCallback(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config) *MultilineStepCallback {
    multilineManager := NewMultilineOutputManager(steps, verboseLevel, debug, errorOutput, config)
    
    // Create fallback ordered manager for verbose mode or non-TTY environments
    fallbackManager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)
    
    // Register all steps in the fallback manager using their unique names
    for _, step := range steps {
        fallbackManager.RegisterStep(step.GetStepName())
    }

    return &MultilineStepCallback{
        manager:        multilineManager,
        results:        make([]StepResult, 0),
        mu:             &sync.Mutex{},
        fallbackManager: fallbackManager,
    }
}

// NewMultilineStepCallbackWithActions creates a new multiline step callback with interpolated actions
func NewMultilineStepCallbackWithActions(steps []Step, verboseLevel int, debug bool, errorOutput io.Writer, config *Config, configPath string, interpolatedActions map[string]*Action) *MultilineStepCallback {
    multilineManager := NewMultilineOutputManager(steps, verboseLevel, debug, errorOutput, config)
    multilineManager.SetConfigPath(configPath)
    
    // Set interpolated actions for multiline manager
    for stepName, action := range interpolatedActions {
        multilineManager.SetInterpolatedAction(stepName, action)
    }
    
    // Create fallback ordered manager with interpolated actions
    fallbackManager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)
    fallbackManager.SetConfigPath(configPath)
    for stepName, action := range interpolatedActions {
        fallbackManager.SetInterpolatedAction(stepName, action)
    }
    
    // Register all steps in the fallback manager using their unique names
    for _, step := range steps {
        fallbackManager.RegisterStep(step.GetStepName())
    }

    return &MultilineStepCallback{
        manager:        multilineManager,
        results:        make([]StepResult, 0),
        mu:             &sync.Mutex{},
        fallbackManager: fallbackManager,
    }
}

// UpdateInterpolatedActions updates the interpolated actions from RunOptions
func (c *MultilineStepCallback) UpdateInterpolatedActions(interpolatedActions map[string]*Action) {
    for stepName, action := range interpolatedActions {
        c.manager.SetInterpolatedAction(stepName, action)
        c.fallbackManager.SetInterpolatedAction(stepName, action)
    }
}

// Initialize initializes the display (call this before execution starts)
func (c *MultilineStepCallback) Initialize() {
    if c.manager.IsEnabled() {
        c.manager.InitializeDisplay()
    }
}

// Cleanup cleans up the display (call this after execution completes)
func (c *MultilineStepCallback) Cleanup() {
    if c.manager.IsEnabled() {
        c.manager.Cleanup()
    }
}

// OnStepStart implements StepCallback interface
func (c *MultilineStepCallback) OnStepStart(ctx context.Context, stepName string) {
    if c.manager.IsEnabled() {
        // Use multiline display for quiet mode
        c.manager.UpdateJobStatus(stepName, JobStatusRunning, "(running...)", 0)
    } else {
        // Use fallback ordered manager for verbose mode or non-TTY
        if c.fallbackManager != nil {
            c.fallbackManager.OnStepStart(ctx, stepName)
        }
    }
}

// OnStepComplete implements StepCallback interface
func (c *MultilineStepCallback) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    if c.manager.IsEnabled() {
        // Convert StepStatus to JobStatus and update multiline display
        jobStatus := c.convertStepStatusToJobStatus(status)
        c.manager.UpdateJobStatus(stepName, jobStatus, message, duration)
    } else {
        // Use fallback ordered manager for verbose mode or non-TTY
        if c.fallbackManager != nil {
            c.fallbackManager.OnStepComplete(ctx, stepName, status, message, duration, bufferedOutput)
        }
    }

    // Collect result for summary (thread-safe)
    c.mu.Lock()
    c.results = append(c.results, StepResult{
        StepName: stepName,
        Status:   status,
        Duration: duration,
    })
    c.mu.Unlock()
}

// OnStepOutput implements StepCallback interface
func (c *MultilineStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    if c.manager.IsEnabled() {
        // In multiline mode, we don't show streaming output during execution
        // The status updates are shown instead
        return
    } else {
        // Use fallback ordered manager for verbose mode
        if c.fallbackManager != nil {
            c.fallbackManager.OnStepOutput(ctx, stepName, output)
        }
    }
}

// OnStepError implements StepCallback interface
func (c *MultilineStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
    if c.manager.IsEnabled() {
        // Convert error to job status and update multiline display
        c.manager.UpdateJobStatus(stepName, JobStatusError, "execute failure", 0)
    } else {
        // Use fallback ordered manager for verbose mode or non-TTY
        if c.fallbackManager != nil {
            c.fallbackManager.OnStepError(ctx, stepName, err)
        }
    }
}

// convertStepStatusToJobStatus converts StepStatus to JobStatus
func (c *MultilineStepCallback) convertStepStatusToJobStatus(status StepStatus) JobStatus {
    switch status {
    case StepStatusPending:
        return JobStatusPending
    case StepStatusRunning:
        return JobStatusRunning
    case StepStatusOK:
        return JobStatusSuccess
    case StepStatusWarn:
        return JobStatusWarning
    case StepStatusError:
        return JobStatusError
    case StepStatusSkipped:
        return JobStatusSkipped
    default:
        return JobStatusPending
    }
}

// GetResults returns the collected step results
func (c *MultilineStepCallback) GetResults() []StepResult {
    c.mu.Lock()
    defer c.mu.Unlock()
    // Return a copy to avoid race conditions
    result := make([]StepResult, len(c.results))
    copy(result, c.results)
    return result
}
