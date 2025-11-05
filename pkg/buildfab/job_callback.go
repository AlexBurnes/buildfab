package buildfab

import (
    "context"
    "fmt"
    "io"
    "sync"
    "time"
)

// SimpleJobCallback implements JobExecutionCallback with simple output
type SimpleJobCallback struct {
    output       io.Writer
    errorOutput  io.Writer
    verboseLevel int
    debug        bool
    results      []StepResult
}

// NewSimpleJobCallback creates a new simple job callback
func NewSimpleJobCallback(output, errorOutput io.Writer, verboseLevel int, debug bool) *SimpleJobCallback {
    return &SimpleJobCallback{
        output:       output,
        errorOutput:  errorOutput,
        verboseLevel: verboseLevel,
        debug:        debug,
        results:      make([]StepResult, 0),
    }
}

// OnJobStart implements JobExecutionCallback
func (s *SimpleJobCallback) OnJobStart(ctx context.Context, job *JobNode) {
    if s.debug {
        fmt.Fprintf(s.errorOutput, "[DEBUG] Job %s (%s) started\n", job.ID, job.DisplayName)
    }
}

// OnJobComplete implements JobExecutionCallback
func (s *SimpleJobCallback) OnJobComplete(ctx context.Context, job *JobNode) {
    if s.debug {
        fmt.Fprintf(s.errorOutput, "[DEBUG] Job %s (%s) completed with status: %s\n", 
            job.ID, job.DisplayName, job.Status)
    }
}

// OnStepStart implements JobExecutionCallback
func (s *SimpleJobCallback) OnStepStart(ctx context.Context, job *JobNode, step *ExecutableStep) {
    if s.verboseLevel > 0 {
        timestamp := formatISO8601Timestamp(time.Now())
        // Use job display name + step display name for full context
        fullName := fmt.Sprintf("%s.%s", job.DisplayName, step.DisplayName)
        // If job has only one step, just use job display name
        if len(job.Steps) == 1 {
            fullName = job.DisplayName
        }
        fmt.Fprintf(s.errorOutput, "  💻 %s [%s]\n", fullName, timestamp)
    }
}

// OnStepComplete implements JobExecutionCallback
func (s *SimpleJobCallback) OnStepComplete(ctx context.Context, job *JobNode, step *ExecutableStep, result StepResult) {
    // Collect result
    s.results = append(s.results, result)
    
    // Display result based on verbosity
    if s.verboseLevel > 0 {
        icon, color := s.getStatusDisplay(result.Status)
        
        durationStr := ""
        if result.Status == StepStatusOK && result.Duration > 0 {
            durationStr = fmt.Sprintf(" - in '%.3fs'", result.Duration.Seconds())
        }
        
        // Use job display name + step display name for full context
        fullName := fmt.Sprintf("%s.%s", job.DisplayName, step.DisplayName)
        // If job has only one step, just use job display name
        if len(job.Steps) == 1 {
            fullName = job.DisplayName
        }
        
        statusText := "executed successfully"
        if result.Status == StepStatusSkipped || result.Status == StepStatusSkippedCondition {
            statusText = "skipped (condition not met)"
        } else if result.Status == StepStatusError {
            statusText = "failed"
        }
        
        fmt.Fprintf(s.errorOutput, "  %s%s%s %s %s%s\n", 
            color, icon, colorReset, fullName, statusText, durationStr)
        
        if result.Error != nil && s.verboseLevel >= 2 {
            fmt.Fprintf(s.errorOutput, "    Error: %v\n", result.Error)
        }
    }
}

// OnStepOutput implements JobExecutionCallback
func (s *SimpleJobCallback) OnStepOutput(ctx context.Context, job *JobNode, step *ExecutableStep, output string) {
    if s.verboseLevel >= 2 {
        fmt.Fprintf(s.errorOutput, "    %s\n", output)
    }
}

// GetResults returns collected results
func (s *SimpleJobCallback) GetResults() []StepResult {
    return s.results
}

// getStatusDisplay returns icon and color for a status
func (s *SimpleJobCallback) getStatusDisplay(status StepStatus) (string, string) {
    switch status {
    case StepStatusOK:
        return "✓", colorGreen
    case StepStatusWarn:
        return "!", colorYellow
    case StepStatusError:
        return "✗", colorRed
    case StepStatusSkipped, StepStatusSkippedCondition:
        return "→", colorGray
    default:
        return "?", colorGray
    }
}


// OrderedJobCallback implements JobExecutionCallback using OrderedOutputManager
// This ensures the same output behavior as the flat DAG executor
type OrderedJobCallback struct {
    manager       *OrderedOutputManager
    results       []StepResult
    mu            sync.Mutex
    stepIDToName  map[string]string // Map step ID to step name for manager
    stepCallback  StepCallback      // Optional user callback (for tests)
}

// NewOrderedJobCallback creates a new ordered job callback using OrderedOutputManager
func NewOrderedJobCallback(dag *HierarchicalDAG, output, errorOutput io.Writer, verboseLevel int, debug bool, config *Config, configPath string, variables map[string]string) *OrderedJobCallback {
    // Flatten hierarchical DAG to ordered step list
    flatSteps := dag.FlattenToSteps()
    
    // Convert ExecutableSteps to Steps for OrderedOutputManager
    steps := make([]Step, len(flatSteps))
    stepIDToName := make(map[string]string)
    
    for i, execStep := range flatSteps {
        // Find the job for this step to get display name
        displayName := execStep.DisplayName
        if job := dag.AllJobs[getJobIDFromStepID(execStep.ID)]; job != nil {
            if len(job.Steps) == 1 {
                // Single step job: use job display name only
                displayName = job.DisplayName
            } else {
                // Multi-step job: use job.step format
                displayName = fmt.Sprintf("%s.%s", job.DisplayName, execStep.DisplayName)
            }
        }
        
        // Create Step with the display name as the action
        steps[i] = Step{
            Action:    displayName,
            Variables: execStep.Variables,
        }
        
        // Map step ID to display name for OnStepStart/OnStepComplete calls
        stepIDToName[execStep.ID] = displayName
    }
    
    // Create OrderedOutputManager with the flat step list
    manager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)
    manager.SetConfigPath(configPath)
    manager.SetVariables(variables)
    
    // Register all steps
    for _, step := range steps {
        manager.RegisterStep(step.GetStepName())
    }
    
    return &OrderedJobCallback{
        manager:      manager,
        results:      make([]StepResult, 0),
        stepIDToName: stepIDToName,
    }
}

// getJobIDFromStepID extracts job ID from step ID (e.g., "1.2" -> "1.2", "0.0" -> "0")
func getJobIDFromStepID(stepID string) string {
    // Step ID format: jobID.stepIndex (e.g., "0.0", "1.2.0")
    // Job ID is all but the last segment
    lastDot := -1
    for i := len(stepID) - 1; i >= 0; i-- {
        if stepID[i] == '.' {
            lastDot = i
            break
        }
    }
    
    if lastDot > 0 {
        return stepID[:lastDot]
    }
    return stepID // Single segment, assume it's the job ID
}

// OnJobStart implements JobExecutionCallback
func (o *OrderedJobCallback) OnJobStart(ctx context.Context, job *JobNode) {
    // Nothing to do - OrderedOutputManager handles per-step output
}

// OnJobComplete implements JobExecutionCallback
func (o *OrderedJobCallback) OnJobComplete(ctx context.Context, job *JobNode) {
    // Nothing to do - OrderedOutputManager handles per-step output
}

// SetStepCallback sets an optional user-provided StepCallback (for tests)
func (o *OrderedJobCallback) SetStepCallback(callback StepCallback) {
    o.stepCallback = callback
}

// OnStepStart implements JobExecutionCallback
func (o *OrderedJobCallback) OnStepStart(ctx context.Context, job *JobNode, step *ExecutableStep) {
    stepName := o.stepIDToName[step.ID]
    o.manager.OnStepStart(ctx, stepName)
    
    // Delegate to user callback if provided
    if o.stepCallback != nil {
        o.stepCallback.OnStepStart(ctx, stepName)
    }
}

// OnStepComplete implements JobExecutionCallback
func (o *OrderedJobCallback) OnStepComplete(ctx context.Context, job *JobNode, step *ExecutableStep, result StepResult) {
    stepName := o.stepIDToName[step.ID]
    
    // Use message from result if available, otherwise build a default one
    message := result.Message
    if message == "" {
        // Build default message based on status
        if result.Status == StepStatusSkipped || result.Status == StepStatusSkippedCondition {
            message = "skipped (condition not met)"
        } else if result.Status == StepStatusError {
            message = "failed"
            if result.Error != nil {
                message = fmt.Sprintf("failed: %v", result.Error)
            }
        } else {
            message = "executed successfully"
        }
    }
    
    // Delegate to OrderedOutputManager
    o.manager.OnStepComplete(ctx, stepName, result.Status, message, result.Duration, "")
    
    // Delegate to user callback if provided
    if o.stepCallback != nil {
        o.stepCallback.OnStepComplete(ctx, stepName, result.Status, message, result.Duration, "")
    }
    
    // Collect result for summary
    o.mu.Lock()
    o.results = append(o.results, result)
    o.mu.Unlock()
}

// OnStepOutput implements JobExecutionCallback
func (o *OrderedJobCallback) OnStepOutput(ctx context.Context, job *JobNode, step *ExecutableStep, output string) {
    stepName := o.stepIDToName[step.ID]
    o.manager.OnStepOutput(ctx, stepName, output)
    
    // Delegate to user callback if provided
    if o.stepCallback != nil {
        o.stepCallback.OnStepOutput(ctx, stepName, output)
    }
}

// GetResults returns collected step results
func (o *OrderedJobCallback) GetResults() []StepResult {
    o.mu.Lock()
    defer o.mu.Unlock()
    resultsCopy := make([]StepResult, len(o.results))
    copy(resultsCopy, o.results)
    return resultsCopy
}

// MultilineJobCallback implements JobExecutionCallback using MultilineOutputManager
// This is used for quiet mode to display all steps upfront and update dynamically
type MultilineJobCallback struct {
    manager       *MultilineOutputManager
    fallback      *OrderedOutputManager
    results       []StepResult
    mu            sync.Mutex
    stepIDToName  map[string]string
    stepCallback  StepCallback      // Optional user callback (for tests)
}

// NewMultilineJobCallback creates a new multiline job callback for quiet mode
func NewMultilineJobCallback(dag *HierarchicalDAG, output, errorOutput io.Writer, verboseLevel int, debug bool, config *Config, configPath string, variables map[string]string) *MultilineJobCallback {
    // Flatten hierarchical DAG to ordered step list
    flatSteps := dag.FlattenToSteps()
    
    // Convert ExecutableSteps to Steps for MultilineOutputManager
    steps := make([]Step, len(flatSteps))
    stepIDToName := make(map[string]string)
    
    for i, execStep := range flatSteps {
        // Find the job for this step to get display name
        displayName := execStep.DisplayName
        if job := dag.AllJobs[getJobIDFromStepID(execStep.ID)]; job != nil {
            if len(job.Steps) == 1 {
                // Single step job: use job display name only
                displayName = job.DisplayName
            } else {
                // Multi-step job: use job.step format
                displayName = fmt.Sprintf("%s.%s", job.DisplayName, execStep.DisplayName)
            }
        }
        
        // Create Step with the display name as the action
        steps[i] = Step{
            Action:    displayName,
            Variables: execStep.Variables,
        }
        
        // Map step ID to display name for OnStepStart/OnStepComplete calls
        stepIDToName[execStep.ID] = displayName
    }
    
    // Create MultilineOutputManager for quiet mode
    multilineManager := NewMultilineOutputManager(steps, verboseLevel, debug, errorOutput, config)
    multilineManager.SetConfigPath(configPath)
    
    // Create fallback ordered manager for non-TTY environments
    fallbackManager := NewOrderedOutputManager(steps, verboseLevel, debug, errorOutput, config)
    fallbackManager.SetConfigPath(configPath)
    fallbackManager.SetVariables(variables)
    
    // Register all steps in fallback manager
    for _, step := range steps {
        fallbackManager.RegisterStep(step.GetStepName())
    }
    
    return &MultilineJobCallback{
        manager:      multilineManager,
        fallback:     fallbackManager,
        results:      make([]StepResult, 0),
        stepIDToName: stepIDToName,
    }
}

// OnJobStart implements JobExecutionCallback
func (m *MultilineJobCallback) OnJobStart(ctx context.Context, job *JobNode) {
    // Nothing to do - MultilineOutputManager handles per-step output
}

// OnJobComplete implements JobExecutionCallback
func (m *MultilineJobCallback) OnJobComplete(ctx context.Context, job *JobNode) {
    // Nothing to do - MultilineOutputManager handles per-step output
}

// SetStepCallback sets an optional user-provided StepCallback (for tests)
func (m *MultilineJobCallback) SetStepCallback(callback StepCallback) {
    m.stepCallback = callback
}

// Initialize initializes the multiline display (call before execution starts)
func (m *MultilineJobCallback) Initialize() {
    if m.manager.IsEnabled() {
        m.manager.InitializeDisplay()
    }
}

// Cleanup cleans up the display (call after execution completes)
func (m *MultilineJobCallback) Cleanup() {
    if m.manager.IsEnabled() {
        m.manager.Cleanup()
    }
}

// OnStepStart implements JobExecutionCallback
func (m *MultilineJobCallback) OnStepStart(ctx context.Context, job *JobNode, step *ExecutableStep) {
    stepName := m.stepIDToName[step.ID]
    
    if m.manager.IsEnabled() {
        // Use multiline display for quiet mode
        m.manager.UpdateJobStatus(stepName, JobStatusRunning, "(running...)", 0)
    } else {
        // Use fallback ordered manager for verbose mode or non-TTY
        if m.fallback != nil {
            m.fallback.OnStepStart(ctx, stepName)
        }
    }
    
    // Delegate to user callback if provided
    if m.stepCallback != nil {
        m.stepCallback.OnStepStart(ctx, stepName)
    }
}

// OnStepComplete implements JobExecutionCallback
func (m *MultilineJobCallback) OnStepComplete(ctx context.Context, job *JobNode, step *ExecutableStep, result StepResult) {
    stepName := m.stepIDToName[step.ID]
    
    // Use message from result if available, otherwise build a default one
    message := result.Message
    if message == "" {
        // Build default message based on status
        if result.Status == StepStatusSkipped || result.Status == StepStatusSkippedCondition {
            message = "skipped (condition not met)"
        } else if result.Status == StepStatusError {
            message = "failed"
            if result.Error != nil {
                message = fmt.Sprintf("failed: %v", result.Error)
            }
        } else {
            message = "executed successfully"
        }
    }
    
    if m.manager.IsEnabled() {
        // Convert StepStatus to JobStatus and update multiline display
        jobStatus := m.convertStepStatusToJobStatus(result.Status)
        m.manager.UpdateJobStatus(stepName, jobStatus, message, result.Duration)
    } else {
        // Use fallback ordered manager for verbose mode or non-TTY
        if m.fallback != nil {
            m.fallback.OnStepComplete(ctx, stepName, result.Status, message, result.Duration, "")
        }
    }
    
    // Delegate to user callback if provided
    if m.stepCallback != nil {
        m.stepCallback.OnStepComplete(ctx, stepName, result.Status, message, result.Duration, "")
    }
    
    // Collect result for summary
    m.mu.Lock()
    m.results = append(m.results, result)
    m.mu.Unlock()
}

// OnStepOutput implements JobExecutionCallback
func (m *MultilineJobCallback) OnStepOutput(ctx context.Context, job *JobNode, step *ExecutableStep, output string) {
    stepName := m.stepIDToName[step.ID]
    
    if m.manager.IsEnabled() {
        // In multiline mode, we don't show streaming output during execution
        return
    } else {
        // Use fallback ordered manager for verbose mode
        if m.fallback != nil {
            m.fallback.OnStepOutput(ctx, stepName, output)
        }
    }
    
    // Delegate to user callback if provided
    if m.stepCallback != nil {
        m.stepCallback.OnStepOutput(ctx, stepName, output)
    }
}

// convertStepStatusToJobStatus converts StepStatus to JobStatus
func (m *MultilineJobCallback) convertStepStatusToJobStatus(status StepStatus) JobStatus {
    switch status {
    case StepStatusOK:
        return JobStatusSuccess
    case StepStatusError:
        return JobStatusError
    case StepStatusSkipped, StepStatusSkippedCondition:
        return JobStatusSkipped
    default:
        return JobStatusPending
    }
}

// GetResults returns collected step results
func (m *MultilineJobCallback) GetResults() []StepResult {
    m.mu.Lock()
    defer m.mu.Unlock()
    resultsCopy := make([]StepResult, len(m.results))
    copy(resultsCopy, m.results)
    return resultsCopy
}
