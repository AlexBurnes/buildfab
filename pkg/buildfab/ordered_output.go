package buildfab

import (
    "context"
    "fmt"
    "io"
    "strings"
    "sync"
    "time"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab/container"
    containerRunner "github.com/AlexBurnes/buildfab/internal/container"
)

// OrderedOutputManager manages step output in proper order using a queue-based approach
// This implements the architecture where:
// 1. Executor only runs tasks and reports to output manager, no direct output
// 2. Output manager has a queue and outputs steps in proper order
// 3. Step execution reports to manager, not UI directly
// 4. Manager outputs step start → step output → step result in sequence
type OrderedOutputManager struct {
    steps       []Step                     // Steps in declaration order
    stepData    map[string]*StepOutputData // Buffered data for each step
    currentStep string                     // Currently active step for output
    mu          *sync.Mutex
    verboseLevel int
    debug       bool
    errorOutput io.Writer
    config      *Config                    // Configuration for command extraction
    configPath  string                     // Configuration file path for container commands
    interpolatedActions map[string]*Action // Interpolated actions for matrix steps
}

// StepOutputData contains all output data for a step
type StepOutputData struct {
    Started        bool
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
    }
}

// SetConfigPath sets the configuration file path
func (o *OrderedOutputManager) SetConfigPath(configPath string) {
    o.configPath = configPath
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
    defer o.mu.Unlock()

    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] OnStepStart: %s\n", stepName)
        o.debugPrintState()
    }

    if data, exists := o.stepData[stepName]; exists {
        data.Started = true
    }

    // Show step start message if this is the next step in order and not already shown
    if o.canShowStepStart(stepName) && !o.stepData[stepName].Shown {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Showing step start for: %s\n", stepName)
        }
        o.showStepStart(stepName)
        o.stepData[stepName].Shown = true
        o.currentStep = stepName
    } else {
        if o.debug {
            fmt.Fprintf(o.errorOutput, "[DEBUG] Cannot show step start for: %s (not ready or already shown)\n", stepName)
        }
    }
}

// OnStepComplete handles step completion events from executor
func (o *OrderedOutputManager) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    o.mu.Lock()
    defer o.mu.Unlock()

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

    // Check if any completed steps can now be shown in order
    o.checkAndShowCompletedSteps()

    // Check if next step can be shown
    o.checkAndShowNextStep()
}

// OnStepOutput handles step output events from executor
func (o *OrderedOutputManager) OnStepOutput(ctx context.Context, stepName string, output string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// For matrix steps, show output immediately if it's the current step
	// This enables real-time streaming for matrix execution
	if o.shouldStreamOutput(stepName) {
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Fprintf(o.errorOutput, "    %s\n", line)
			}
		}
		// Don't buffer output when streaming - it's already displayed
		return
	}

	// Otherwise, buffer output for later display when step completes
	if data, exists := o.stepData[stepName]; exists {
		data.Output = append(data.Output, output)
		if o.debug {
			fmt.Fprintf(o.errorOutput, "[DEBUG] Buffered output for %s: %s\n", stepName, output)
		}
	}
}

// shouldStreamOutput checks if the given step should have its output streamed
func (o *OrderedOutputManager) shouldStreamOutput(stepName string) bool {
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range o.steps {
		if step.Action == stepName {
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
		if data, exists := o.stepData[o.steps[i].Action]; !exists || !data.Completed {
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
        if step.Action == stepName {
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
        prevStepName := o.steps[i].Action
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
        for _, step := range o.steps {
            stepName := step.Action
            if data, exists := o.stepData[stepName]; exists && data.Completed && !data.Shown {
                // Check if all previous steps have been completed AND shown
                canShow := true
                for _, s := range o.steps {
                    if s.Action == stepName {
                        break
                    }
                    prevData, prevExists := o.stepData[s.Action]
                    if !prevExists || !prevData.Completed || !prevData.Shown {
                        canShow = false
                        break
                    }
                }

                if canShow {
                    if o.debug {
                        fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowCompletedSteps: showing completed step: %s\n", stepName)
                    }
                    o.showStepStart(stepName)
                    o.stepData[stepName].Shown = true
                    o.currentStep = stepName

                    // Flush any buffered output for this step
                    o.flushBufferedOutput(stepName)

                    o.showStepCompletion(stepName)

                    // Flush any remaining buffered output after step completion
                    o.flushBufferedOutput(stepName)

                    o.currentStep = ""

                    shownAny = true
                    break // Start over to check for more steps that can now be shown
                }
            }
        }
    }
}

// checkAndShowNextStep checks if the next step can be shown
func (o *OrderedOutputManager) checkAndShowNextStep() {
    if o.debug {
        fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: checking for next step to show\n")
    }

    // Find the next step that can be shown
    for _, step := range o.steps {
        stepName := step.Action
        if data, exists := o.stepData[stepName]; exists && data.Started && !data.Completed && !data.Shown {
            if o.canShowStepStart(stepName) {
                if o.debug {
                    fmt.Fprintf(o.errorOutput, "[DEBUG] checkAndShowNextStep: showing next step: %s\n", stepName)
                }
                o.showStepStart(stepName)
                o.stepData[stepName].Shown = true
                o.currentStep = stepName

                // Flush any buffered output for this step
                o.flushBufferedOutput(stepName)
                break
            }
        }
    }
}

// showStepStart shows the start message for a step
func (o *OrderedOutputManager) showStepStart(stepName string) {
    if o.verboseLevel > 0 {
        fmt.Fprintf(o.errorOutput, "  💻 %s\n", stepName)
        
        // Container command display is now handled in the actual execution
    } else {
        // In quiet mode, don't show individual step indicators
        // The summary will show the overall results
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
    
    if action.Container != nil {
        // Create a temporary runner to prepare the configuration for display
        tempRunner, err := containerRunner.NewContainerRunnerWithVerbosity(o.verboseLevel)
        if err == nil {
            // Use the actual config path
            configPath := o.configPath
            if configPath == "" {
                configPath = ".project.yml" // Fallback
            }
            // Use the interpolated container configuration with matrix variables already substituted
            preparedConfig, prepErr := tempRunner.PrepareContainerConfig(*action.Container, configPath)
            if prepErr == nil {
                containerCmd := o.buildContainerCommand(&preparedConfig)
                fmt.Fprintf(o.errorOutput, "  🐳 Running container: %s\n", containerCmd)
            }
        }
    }
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
        
        // Add mount arguments
        for _, mount := range config.Mounts {
            mountArg := fmt.Sprintf("--mount=type=%s,source=%s,target=%s", mount.Type, mount.Source, mount.Target)
            if mount.RO {
                mountArg += ",readonly"
            }
            parts = append(parts, mountArg)
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
            parts = append(parts, "sh", "-c", config.Run)
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
            fmt.Fprintf(o.errorOutput, "\r  %s%s%s %s execute failure\n", color, icon, colorReset, stepName)

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
        // In quiet mode, show step completion results but not step start messages
        fmt.Fprintf(o.errorOutput, "  %s%s%s %s %s\n", color, icon, colorReset, stepName, enhancedMessage)
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

    // Register all steps
    for _, step := range steps {
        manager.RegisterStep(step.Action)
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

    // Register all steps
    for _, step := range steps {
        manager.RegisterStep(step.Action)
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
        fmt.Fprintf(o.errorOutput, "%s", step.Action)
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
