package buildfab

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// HierarchicalExecutor executes a hierarchical DAG of jobs
type HierarchicalExecutor struct {
    dag      *HierarchicalDAG
    config   *Config
    opts     *RunOptions
    callback JobExecutionCallback
    registry ActionRegistry // Custom action registry (for tests)
    mu       sync.Mutex
}

// JobExecutionCallback defines callbacks for job execution events
type JobExecutionCallback interface {
    // OnJobStart is called when a job starts
    OnJobStart(ctx context.Context, job *JobNode)
    
    // OnJobComplete is called when a job completes
    OnJobComplete(ctx context.Context, job *JobNode)
    
    // OnStepStart is called when a step within a job starts
    OnStepStart(ctx context.Context, job *JobNode, step *ExecutableStep)
    
    // OnStepComplete is called when a step completes
    OnStepComplete(ctx context.Context, job *JobNode, step *ExecutableStep, result StepResult)
    
    // OnStepOutput is called for step output
    OnStepOutput(ctx context.Context, job *JobNode, step *ExecutableStep, output string)
}

// NewHierarchicalExecutor creates a new hierarchical executor
func NewHierarchicalExecutor(dag *HierarchicalDAG, config *Config, opts *RunOptions, callback JobExecutionCallback) *HierarchicalExecutor {
    return &HierarchicalExecutor{
        dag:      dag,
        config:   config,
        opts:     opts,
        callback: callback,
    }
}

// Execute executes the hierarchical DAG
func (he *HierarchicalExecutor) Execute(ctx context.Context) error {
    // Execute root jobs
    return he.executeJobs(ctx, he.dag.RootJobs)
}

// executeJobs executes a list of jobs with parallelism control
func (he *HierarchicalExecutor) executeJobs(ctx context.Context, jobs []*JobNode) error {
    if len(jobs) == 0 {
        return nil
    }
    
    // Build execution waves based on job dependencies
    waves := he.buildExecutionWaves(jobs)
    
    // Execute each wave, respecting max_parallel constraints within the wave
    for waveIdx, wave := range waves {
        if he.opts.Debug {
            fmt.Fprintf(he.opts.ErrorOutput, "[DEBUG] Executing wave %d with %d jobs\n", waveIdx, len(wave))
        }
        
        // Use semaphore to control parallelism within wave
        maxParallel := he.opts.MaxParallel
        if maxParallel <= 0 {
            maxParallel = len(wave) // No limit, all jobs in wave run in parallel
        }
        
        semaphore := make(chan struct{}, maxParallel)
        var wg sync.WaitGroup
        errorChan := make(chan error, len(wave))
        
        for _, job := range wave {
            wg.Add(1)
            
            // Acquire semaphore slot (blocks if max_parallel slots are in use)
            semaphore <- struct{}{}
            
            go func(j *JobNode) {
                defer wg.Done()
                defer func() { <-semaphore }() // Release semaphore slot
                
                // Wait for this job's dependencies to complete before executing
                he.waitForDependencies(j)
                
                err := he.executeJob(ctx, j)
                if err != nil && j.Status == JobExecutionStatusError {
                    errorChan <- err
                }
            }(job)
        }
        
        wg.Wait()
        close(errorChan)
        
        // Collect errors from this wave
        var waveErrors []error
        for err := range errorChan {
            if err != nil {
                waveErrors = append(waveErrors, err)
                if he.opts.Debug {
                    fmt.Fprintf(he.opts.ErrorOutput, "[DEBUG] Job in wave %d failed: %v\n", waveIdx, err)
                }
            }
        }
        
        // Return first error if any job failed (fail-fast behavior)
        if len(waveErrors) > 0 {
            return waveErrors[0]
        }
    }
    
    return nil
}

// buildExecutionWaves groups jobs into waves based on dependencies
func (he *HierarchicalExecutor) buildExecutionWaves(jobs []*JobNode) [][]*JobNode {
    // Build job dependency map
    jobMap := make(map[string]*JobNode)
    for _, job := range jobs {
        jobMap[job.ID] = job
    }
    
    // Assign each job to a wave (topological sort by wave)
    jobWaves := make(map[string]int)
    var waves [][]*JobNode
    
    // Keep assigning jobs to waves until all are assigned
    assigned := make(map[string]bool)
    waveIdx := 0
    
    for len(assigned) < len(jobs) {
        var currentWave []*JobNode
        
        // Find jobs where all dependencies are assigned to earlier waves
        for _, job := range jobs {
            if assigned[job.ID] {
                continue
            }
            
            // Check if all dependencies are in earlier waves
            allDepsAssigned := true
            for _, depID := range job.Dependencies {
                if !assigned[depID] {
                    allDepsAssigned = false
                    break
                }
            }
            
            if allDepsAssigned {
                currentWave = append(currentWave, job)
                assigned[job.ID] = true
                jobWaves[job.ID] = waveIdx
            }
        }
        
        if len(currentWave) == 0 && len(assigned) < len(jobs) {
            // No progress - there must be a cycle
            return nil
        }
        
        if len(currentWave) > 0 {
            waves = append(waves, currentWave)
            waveIdx++
        }
    }
    
    return waves
}

// executeJob executes a single job node
func (he *HierarchicalExecutor) executeJob(ctx context.Context, job *JobNode) error {
    he.mu.Lock()
    job.Status = JobExecutionStatusRunning
    job.StartTime = time.Now()
    he.mu.Unlock()
    
    // Notify callback
    if he.callback != nil {
        he.callback.OnJobStart(ctx, job)
    }
    
    defer func() {
        he.mu.Lock()
        job.EndTime = time.Now()
        job.Duration = job.EndTime.Sub(job.StartTime)
        he.mu.Unlock()
        
        if he.callback != nil {
            he.callback.OnJobComplete(ctx, job)
        }
    }()
    
    // Check job-level if condition (from parent step)
    if job.If != "" {
        // Merge job's matrix variables with global variables for condition evaluation
        mergedVars := make(map[string]string)
        for k, v := range he.opts.Variables {
            mergedVars[k] = v
        }
        for k, v := range job.MatrixVars {
            mergedVars[k] = v
        }
        
        shouldExecute, err := he.evaluateJobCondition(job.If, mergedVars)
        if err != nil {
            if he.opts.VerboseLevel > 0 {
                fmt.Fprintf(he.opts.ErrorOutput, "Warning: failed to evaluate job condition for %s: %v\n", job.DisplayName, err)
            }
            shouldExecute = false
        }
        
        if !shouldExecute {
            he.mu.Lock()
            job.Status = JobExecutionStatusSkippedCondition
            he.mu.Unlock()
            
            // Notify callback about all steps being skipped due to job condition
            for i := range job.Steps {
                step := &job.Steps[i]
                result := StepResult{
                    StepName: step.ID,
                    Status:   StepStatusSkippedCondition,
                    Duration: 0,
                }
                job.Results = append(job.Results, result)
                
                if he.callback != nil {
                    he.callback.OnStepComplete(ctx, job, step, result)
                }
            }
            
            return nil
        }
    }
    
    // Check if job should be skipped due to dependencies
    if he.shouldSkipJobDueToDependencies(job) {
        he.mu.Lock()
        job.Status = JobExecutionStatusSkipped
        he.mu.Unlock()
        
        // Notify callback about all steps being skipped
        for i := range job.Steps {
            step := &job.Steps[i]
            result := StepResult{
                StepName: step.ID,
                Status:   StepStatusSkipped,
                Duration: 0,
            }
            job.Results = append(job.Results, result)
            
            if he.callback != nil {
                he.callback.OnStepComplete(ctx, job, step, result)
            }
        }
        
        return nil
    }
    
    // If job has child jobs (nested matrix), execute them instead of steps
    if len(job.ChildJobs) > 0 {
        err := he.executeJobs(ctx, job.ChildJobs)
        
        // Determine job status from child jobs
        he.mu.Lock()
        job.Status = he.determineJobExecutionStatusFromChildren(job)
        he.mu.Unlock()
        
        return err
    }
    
    // Execute steps sequentially within this job
    stepSkipped := false  // Track if a step was skipped to skip remaining steps
    for i := range job.Steps {
        step := &job.Steps[i]
        
        // If a previous step was skipped, skip all remaining steps
        if stepSkipped {
            result := StepResult{
                StepName: step.ID,
                Status:   StepStatusSkippedCondition,
                Duration: 0,
            }
            job.Results = append(job.Results, result)
            
            if he.callback != nil {
                he.callback.OnStepComplete(ctx, job, step, result)
            }
            continue
        }
        
        // Check step-level if condition
        if step.If != "" {
            shouldExecute, err := he.evaluateStepCondition(step, job)
            if err != nil {
                if he.opts.VerboseLevel > 0 {
                    fmt.Fprintf(he.opts.ErrorOutput, "Warning: failed to evaluate condition for step %s: %v\n", step.DisplayName, err)
                }
                shouldExecute = false
            }
            
            if !shouldExecute {
                // Skip this step and mark that subsequent steps should also be skipped
                result := StepResult{
                    StepName: step.ID,
                    Status:   StepStatusSkippedCondition,
                    Duration: 0,
                }
                job.Results = append(job.Results, result)
                
                if he.callback != nil {
                    he.callback.OnStepComplete(ctx, job, step, result)
                }
                
                stepSkipped = true  // Mark to skip remaining steps
                continue
            }
        }
        
        // Execute the step (passing onError policy for proper status handling)
        result := he.executeStepWithPolicy(ctx, job, step)
        job.Results = append(job.Results, result)
        
        // Check if we should stop on error (after policy has been applied)
        if result.Status == StepStatusError {
            // Error status means onError was "stop" or default, so stop execution
            he.mu.Lock()
            job.Status = JobExecutionStatusError
            job.Error = result.Error
            he.mu.Unlock()
            return result.Error
        }
        // If status is Warn, continue execution
    }
    
    // Determine final job status
    he.mu.Lock()
    job.Status = he.determineJobExecutionStatusFromSteps(job)
    he.mu.Unlock()
    
    return nil
}

// executeStepWithPolicy executes a single step within a job, applying onError policy
func (he *HierarchicalExecutor) executeStepWithPolicy(ctx context.Context, job *JobNode, step *ExecutableStep) StepResult {
    startTime := time.Now()
    
    // Notify callback
    if he.callback != nil {
        he.callback.OnStepStart(ctx, job, step)
    }
    
    // Create modified options with StepCallback bridge and merged variables
    modifiedOpts := *he.opts
    
    // Merge step variables (including matrix vars) with global variables
    mergedVars := make(map[string]string)
    for k, v := range he.opts.Variables {
        mergedVars[k] = v
    }
    for k, v := range job.MatrixVars {
        mergedVars[k] = v
    }
    for k, v := range step.Variables {
        mergedVars[k] = v
    }
    modifiedOpts.Variables = mergedVars
    
    if he.callback != nil {
        // Create bridge callback to connect StepCallback interface to JobExecutionCallback
        bridge := &stepCallbackBridge{
            job:      job,
            step:     step,
            callback: he.callback,
            ctx:      ctx,
        }
        modifiedOpts.StepCallback = bridge
    }
    
    // Execute the action using existing action execution logic
    runner := NewRunner(he.config, &modifiedOpts)
    // Transfer custom registry if exists (for tests with custom built-in actions)
    if he.registry != nil {
        runner.registry = he.registry
    }
    
    var err error
    var actionResult Result
    var resultMessage string
    
    // Check if this is a built-in action (Uses field) to get custom Result
    if step.Action.Uses != "" {
        actionResult, err = runner.runBuiltInActionForDAG(ctx, *step.Action)
        resultMessage = actionResult.Message
    } else {
        // Custom action execution
        err = runner.runActionInternal(ctx, *step.Action)
        resultMessage = "executed successfully"
    }
    
    duration := time.Since(startTime)
    
    result := StepResult{
        StepName: step.ID,
        Status:   StepStatusOK,
        Duration: duration,
        Error:    err,
    }
    
    // Set status based on error and onError policy
    if err != nil {
        if step.OnError == "warn" {
            result.Status = StepStatusWarn
            if resultMessage == "executed successfully" {
                resultMessage = "failed"
            }
        } else {
            result.Status = StepStatusError
            if resultMessage == "executed successfully" {
                resultMessage = "failed"
            }
        }
    } else if step.Action.Uses != "" {
        // For built-in actions, use the Result status and message
        if actionResult.Status == StatusWarn {
            result.Status = StepStatusWarn
        } else if actionResult.Status == StatusError {
            result.Status = StepStatusError
        }
    }
    
    // Store the message in both Message and Output (for backward compatibility with tests)
    result.Message = resultMessage
    result.Output = resultMessage
    
    // Notify callback of completion (only once, with final status)
    if he.callback != nil {
        he.callback.OnStepComplete(ctx, job, step, result)
    }
    
    return result
}

// shouldSkipJobDueToDependencies checks if job should be skipped due to failed dependencies
func (he *HierarchicalExecutor) shouldSkipJobDueToDependencies(job *JobNode) bool {
    for _, depID := range job.Dependencies {
        depJob := he.dag.GetJob(depID)
        if depJob == nil {
            continue
        }
        
        // Check if this dependency blocks execution
        switch depJob.Status {
        case JobExecutionStatusError, JobExecutionStatusSkipped:
            // Always block on errors and dependency-based skips
            return true
            
        case JobExecutionStatusSkippedCondition:
            // Only block if this is NOT a sliding window dependency
            if job.SlidingWindowDeps == nil || !job.SlidingWindowDeps[depID] {
                return true // Explicit dependency, block
            }
            // Sliding window dependency, don't block
            
        case JobExecutionStatusPartial:
            // If some steps failed, treat as error
            if job.SlidingWindowDeps == nil || !job.SlidingWindowDeps[depID] {
                return true
            }
        }
    }
    
    return false
}

// evaluateStepCondition evaluates a step's if condition
func (he *HierarchicalExecutor) evaluateStepCondition(step *ExecutableStep, job *JobNode) (bool, error) {
    // Merge job variables with step variables
    mergedVars := he.mergeVariables(he.opts.Variables, job.MatrixVars)
    mergedVars = he.mergeVariables(mergedVars, step.Variables)
    
    // Evaluate the condition
    ctx := NewExpressionContext(mergedVars)
    return EvaluateExpression(step.If, ctx)
}

// evaluateJobCondition evaluates a job-level condition expression
func (he *HierarchicalExecutor) evaluateJobCondition(condition string, variables map[string]string) (bool, error) {
    // Evaluate the condition using the expression evaluator
    ctx := NewExpressionContext(variables)
    return EvaluateExpression(condition, ctx)
}

// mergeVariables merges variable maps (later override earlier)
func (he *HierarchicalExecutor) mergeVariables(base, override map[string]string) map[string]string {
    merged := make(map[string]string)
    for k, v := range base {
        merged[k] = v
    }
    for k, v := range override {
        merged[k] = v
    }
    return merged
}

// determineJobExecutionStatusFromSteps determines job status from step results
func (he *HierarchicalExecutor) determineJobExecutionStatusFromSteps(job *JobNode) JobExecutionStatus {
    if len(job.Results) == 0 {
        return JobExecutionStatusOK // No steps executed, consider OK
    }
    
    hasError := false
    hasWarn := false
    hasSkipped := false
    
    for _, result := range job.Results {
        switch result.Status {
        case StepStatusError:
            hasError = true
        case StepStatusWarn:
            hasWarn = true
        case StepStatusSkipped, StepStatusSkippedCondition:
            hasSkipped = true
        }
    }
    
    if hasError {
        return JobExecutionStatusError
    }
    if hasWarn {
        return JobExecutionStatusWarn
    }
    if hasSkipped && len(job.Results) < len(job.Steps) {
        return JobExecutionStatusPartial
    }
    
    return JobExecutionStatusOK
}

// determineJobExecutionStatusFromChildren determines job status from child job results
func (he *HierarchicalExecutor) determineJobExecutionStatusFromChildren(job *JobNode) JobExecutionStatus {
    if len(job.ChildJobs) == 0 {
        return JobExecutionStatusOK
    }
    
    hasError := false
    hasWarn := false
    hasSkipped := false
    
    for _, child := range job.ChildJobs {
        switch child.Status {
        case JobExecutionStatusError:
            hasError = true
        case JobExecutionStatusWarn:
            hasWarn = true
        case JobExecutionStatusSkipped, JobExecutionStatusSkippedCondition, JobExecutionStatusPartial:
            hasSkipped = true
        }
    }
    
    if hasError {
        return JobExecutionStatusError
    }
    if hasWarn {
        return JobExecutionStatusWarn
    }
    if hasSkipped {
        return JobExecutionStatusPartial
    }
    
    return JobExecutionStatusOK
}


// stepCallbackBridge bridges StepCallback to JobExecutionCallback
type stepCallbackBridge struct {
    job      *JobNode
    step     *ExecutableStep
    callback JobExecutionCallback
    ctx      context.Context
}

// OnStepStart implements StepCallback
func (b *stepCallbackBridge) OnStepStart(ctx context.Context, stepName string) {
    // Already handled by HierarchicalExecutor
}

// OnStepComplete implements StepCallback  
func (b *stepCallbackBridge) OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration, bufferedOutput string) {
    // Already handled by HierarchicalExecutor
}

// OnStepOutput implements StepCallback
func (b *stepCallbackBridge) OnStepOutput(ctx context.Context, stepName string, output string) {
    if b.callback != nil {
        b.callback.OnStepOutput(b.ctx, b.job, b.step, output)
    }
}

// OnStepError implements StepCallback
func (b *stepCallbackBridge) OnStepError(ctx context.Context, stepName string, err error) {
    // Error handling is done in HierarchicalExecutor
}

// GetResults implements StepCallback
func (b *stepCallbackBridge) GetResults() []StepResult {
    // Not used by bridge - results are collected by OrderedJobCallback
    return nil
}

// waitForDependencies waits for all dependencies of a job to complete
func (he *HierarchicalExecutor) waitForDependencies(job *JobNode) {
    if len(job.Dependencies) == 0 {
        return
    }
    
    // Poll dependencies until all are complete
    for {
        allComplete := true
        
        for _, depID := range job.Dependencies {
            depJob := he.dag.GetJob(depID)
            if depJob == nil {
                continue
            }
            
            he.mu.Lock()
            depComplete := depJob.IsComplete()
            he.mu.Unlock()
            
            if !depComplete {
                allComplete = false
                break
            }
        }
        
        if allComplete {
            break
        }
        
        // Small sleep to avoid busy-waiting
        time.Sleep(10 * time.Millisecond)
    }
}
