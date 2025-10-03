package buildfab

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// MatrixExpander handles matrix expansion and job creation
type MatrixExpander struct {
	config        *Config
	cliMatrixVars map[string]string // CLI-provided matrix variables
}

// NewMatrixExpander creates a new matrix expander
func NewMatrixExpander(config *Config, cliMatrixVars ...map[string]string) *MatrixExpander {
	var matrixVars map[string]string
	if len(cliMatrixVars) > 0 {
		matrixVars = cliMatrixVars[0]
	} else {
		matrixVars = make(map[string]string)
	}
	
	return &MatrixExpander{
		config:        config,
		cliMatrixVars: matrixVars,
	}
}

// ExpandMatrix expands a matrix configuration into individual jobs
func (me *MatrixExpander) ExpandMatrix(step *Step, action *Action) ([]*MatrixJob, error) {
	if step.Matrix == nil {
		return nil, fmt.Errorf("step has no matrix configuration")
	}

	// Override matrix values with CLI-provided values if any
	matrixValues := make(map[string][]interface{})
	for key, values := range step.Matrix.Values {
		matrixValues[key] = values
	}

	// Override with CLI-provided matrix values
	for cliKey, cliValue := range me.cliMatrixVars {
		matrixValues[cliKey] = []interface{}{cliValue}
	}

	// Generate Cartesian product of all matrix values
	combinations := me.generateCombinations(matrixValues)
	
	jobs := make([]*MatrixJob, 0, len(combinations))
	for i, combination := range combinations {
		job := &MatrixJob{
			ID:     fmt.Sprintf("%s-%d", action.Name, i+1),
			Matrix: combination,
			Action: action,
			Status: StatusPending,
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// ExpandMatrixToSteps expands a matrix configuration into individual steps for DAG execution
func (me *MatrixExpander) ExpandMatrixToSteps(step *Step, action *Action) ([]Step, error) {
	if step.Matrix == nil {
		return nil, fmt.Errorf("step has no matrix configuration")
	}

	// Override matrix values with CLI-provided values if any
	matrixValues := make(map[string][]interface{})
	for key, values := range step.Matrix.Values {
		matrixValues[key] = values
	}

	// Override with CLI-provided matrix values
	for cliKey, cliValue := range me.cliMatrixVars {
		matrixValues[cliKey] = []interface{}{cliValue}
	}

	// Generate Cartesian product of all matrix values
	combinations := me.generateCombinations(matrixValues)
	
	steps := make([]Step, 0, len(combinations))
	totalJobs := len(combinations)
	
	for i, combination := range combinations {
		// Create step description
		description := me.generateStepDescription(action.Name, i+1, totalJobs)
		
		// Create matrix variables for this step
		matrixVars := make(map[string]string)
		for key, value := range combination {
			matrixVars[fmt.Sprintf("matrix.%s", key)] = fmt.Sprintf("%v", value)
		}
		
		// DEBUG: Add delay for testing parallel execution and output ordering
		// Uncomment the following lines to enable delays for debugging:
		// delay := i + 1  // Ascending order: 1, 2, 3, 4
		// delay := totalJobs - i  // Countdown order: 4, 3, 2, 1
		
		// Create a new action with matrix variables interpolated
		matrixAction := *action // Copy the original action
		matrixAction.Name = me.generateStepName(action.Name, combination, step.Matrix.Values) // Generate name with matrix values
		
		// Interpolate variables in the action
		if matrixAction.Run != "" {
			// Interpolate variables in the run command
			matrixAction.Run = me.interpolateVariables(matrixAction.Run, matrixVars)
		}
		
		// DEBUG: Add delay option to the action (uncomment for debugging)
		// if matrixAction.Options == nil {
		// 	matrixAction.Options = make(map[string]interface{})
		// }
		// matrixAction.Options["delay"] = delay
		
		// Add the interpolated action to the config
		me.config.Actions = append(me.config.Actions, matrixAction)
		
		// Create new step that references the interpolated action
		newStep := Step{
			Action:      matrixAction.Name, // Use the interpolated action name
			Description: description,
			Require:     step.Require, // Keep original dependencies
			OnError:     step.OnError,
			If:          step.If,
		}
		
		steps = append(steps, newStep)
	}

	return steps, nil
}

// interpolateVariables interpolates variables in a string
func (me *MatrixExpander) interpolateVariables(text string, variables map[string]string) string {
	result := text
	for key, value := range variables {
		placeholder := fmt.Sprintf("${{ %s }}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// generateStepName creates a step name with matrix values concatenated
func (me *MatrixExpander) generateStepName(actionName string, combination map[string]interface{}, originalValues map[string][]interface{}) string {
	var parts []string
	parts = append(parts, actionName)
	
	// Use the same sorted order as the Cartesian product generation
	keys := make([]string, 0, len(originalValues))
	for key := range originalValues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		if value, exists := combination[key]; exists {
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	}
	
	return strings.Join(parts, ".")
}

// generateStepDescription creates a step description with job index
func (me *MatrixExpander) generateStepDescription(actionName string, jobIndex, totalJobs int) string {
	return fmt.Sprintf("%s matrix (%d/%d)", actionName, jobIndex, totalJobs)
}

// generateCombinations creates a Cartesian product of all matrix values
func (me *MatrixExpander) generateCombinations(values map[string][]interface{}) []map[string]interface{} {
	if len(values) == 0 {
		return []map[string]interface{}{}
	}

	// Get all keys and their values in a deterministic order
	keys := make([]string, 0, len(values))
	valueLists := make([][]interface{}, 0, len(values))
	
	// Sort keys to ensure consistent, deterministic order
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	// Add values in sorted key order
	for _, key := range keys {
		valueLists = append(valueLists, values[key])
	}

	// Generate Cartesian product
	combinations := []map[string]interface{}{}
	me.cartesianProduct(keys, valueLists, 0, make(map[string]interface{}), &combinations)
	
	return combinations
}

// cartesianProduct recursively generates Cartesian product
func (me *MatrixExpander) cartesianProduct(keys []string, valueLists [][]interface{}, index int, current map[string]interface{}, result *[]map[string]interface{}) {
	if index == len(keys) {
		// Create a copy of current combination
		combination := make(map[string]interface{})
		for k, v := range current {
			combination[k] = v
		}
		*result = append(*result, combination)
		return
	}

	// Try each value for current key
	for _, value := range valueLists[index] {
		current[keys[index]] = value
		me.cartesianProduct(keys, valueLists, index+1, current, result)
	}
}

// MatrixScheduler handles matrix job execution and scheduling
type MatrixScheduler struct {
	jobs           []*MatrixJob
	strategy       MatrixStrategy
	maxParallel    int
	jobQueue       chan *MatrixJob
	completedJobs  chan *MatrixJob
	mu             sync.RWMutex
	runningJobs    map[string]*MatrixJob
	completedCount int
	failedCount    int
	ctx            context.Context
	cancel         context.CancelFunc
	stepCallback   StepCallback
	stepName       string
	opts           *RunOptions
}

// NewMatrixScheduler creates a new matrix scheduler
func NewMatrixScheduler(jobs []*MatrixJob, strategy MatrixStrategy, maxParallel int, opts *RunOptions) *MatrixScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &MatrixScheduler{
		jobs:          jobs,
		strategy:      strategy,
		maxParallel:   maxParallel,
		jobQueue:      make(chan *MatrixJob, len(jobs)),
		completedJobs: make(chan *MatrixJob, len(jobs)),
		runningJobs:   make(map[string]*MatrixJob),
		ctx:           ctx,
		cancel:        cancel,
		opts:          opts,
	}
}

// SetStepCallback sets the step callback and step name for matrix execution
func (ms *MatrixScheduler) SetStepCallback(callback StepCallback, stepName string) {
	ms.stepCallback = callback
	ms.stepName = stepName
}

// ScheduleJobs schedules and executes matrix jobs according to the strategy
func (ms *MatrixScheduler) ScheduleJobs(runner *Runner, options *RunOptions) error {
	// Call step start callback if provided
	if ms.stepCallback != nil {
		ms.stepCallback.OnStepStart(ms.ctx, ms.stepName)
	}
	
	// Set up job ordering
	ms.orderJobs()
	
	// Start job queue
	go ms.queueJobs()
	
	// Start job execution workers
	var wg sync.WaitGroup
	for i := 0; i < ms.maxParallel; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ms.executeJobs(runner, options, workerID)
		}(i)
	}
	
	// Monitor job completion
	go ms.monitorJobs()
	
	// Wait for all workers to complete
	wg.Wait()
	
	// Determine final status based on strategy
	err := ms.determineFinalStatus()
	
	// Call step complete callback if provided
	if ms.stepCallback != nil {
		status := StepStatusOK
		message := fmt.Sprintf("matrix execution completed: %d jobs", len(ms.jobs))
		if err != nil {
			status = StepStatusError
			message = err.Error()
		}
		ms.stepCallback.OnStepComplete(ms.ctx, ms.stepName, status, message, 0, "")
	}
	
	return err
}

// executeActionWithoutCallbacks executes an action without triggering step callbacks
func (ms *MatrixScheduler) executeActionWithoutCallbacks(runner *Runner, actionName string) error {
	// Check if it's a built-in action first
	if actionRunner, exists := runner.registry.GetRunner(actionName); exists {
		// Handle dry-run mode for built-in actions
		if ms.opts != nil && ms.opts.DryRun {
			// For dry-run, we don't need to do anything special
			return nil
		}

		// Execute the built-in action
		_, err := actionRunner.Run(ms.ctx)
		return err
	}

	// Handle custom action
	action, exists := runner.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	// Handle dry-run mode for custom actions
	if ms.opts != nil && ms.opts.DryRun {
		// For dry-run, we don't need to do anything special
		return nil
	}

	// Execute custom action using the internal method without callbacks
	return ms.runCustomActionWithoutCallbacks(runner, action)
}

// runCustomActionWithoutCallbacks executes a custom action without step callbacks
func (ms *MatrixScheduler) runCustomActionWithoutCallbacks(runner *Runner, action Action) error {
	// Select variant if action has variants
	var effectiveAction Action
	if len(action.Variants) > 0 {
		variant, err := action.SelectVariant(runner.opts.Variables)
		if err != nil {
			return fmt.Errorf("failed to select variant for action %s: %w", action.Name, err)
		}
		
		if variant == nil {
			// No variant matched, skip this action
			return nil
		}
		
		// Use the selected variant
		effectiveAction = Action{
			Name: action.Name,
			Run:  variant.Run,
			Uses: variant.Uses,
			Shell: variant.Shell,
		}
	} else {
		effectiveAction = action
	}
	
	// Execute the action based on its type
	if effectiveAction.Uses != "" {
		// Built-in action (should have been caught above, but just in case)
		if actionRunner, exists := runner.registry.GetRunner(effectiveAction.Uses); exists {
			_, err := actionRunner.Run(ms.ctx)
			return err
		}
		return fmt.Errorf("built-in action not found: %s", effectiveAction.Uses)
	} else {
		// Custom action - execute the run command
		return ms.runCustomCommand(effectiveAction)
	}
}

// runCustomCommand executes a custom command without step callbacks
func (ms *MatrixScheduler) runCustomCommand(action Action) error {
	if action.Run == "" {
		return fmt.Errorf("action %s has no run command", action.Name)
	}

	// Interpolate variables in the command
	interpolatedCmd, err := InterpolateVariables(action.Run, ms.opts.Variables)
	if err != nil {
		return fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}

	// Determine shell to use
	shell := action.Shell
	if shell == "" {
		shell = "sh" // Default shell
	}

	// Execute the command
	cmd := exec.CommandContext(ms.ctx, shell, "-c", interpolatedCmd)
	cmd.Stdout = ms.opts.Output
	cmd.Stderr = ms.opts.ErrorOutput
	
	return cmd.Run()
}

// orderJobs orders jobs according to the strategy
func (ms *MatrixScheduler) orderJobs() {
	switch ms.strategy.Order {
	case "random":
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(ms.jobs), func(i, j int) {
			ms.jobs[i], ms.jobs[j] = ms.jobs[j], ms.jobs[i]
		})
	case "fifo":
		// Jobs are already in FIFO order
	default:
		// Default to FIFO
	}
}

// queueJobs queues jobs for execution
func (ms *MatrixScheduler) queueJobs() {
	defer close(ms.jobQueue)
	
	for _, job := range ms.jobs {
		select {
		case ms.jobQueue <- job:
		case <-ms.ctx.Done():
			return
		}
	}
}

// executeJobs executes jobs from the queue
func (ms *MatrixScheduler) executeJobs(runner *Runner, options *RunOptions, workerID int) {
	for {
		select {
		case job, ok := <-ms.jobQueue:
			if !ok {
				return // Queue closed
			}
			
			// Check if we should stop due to fail_fast
			if ms.shouldStop() {
				return
			}
			
			ms.executeJob(job, runner, options, workerID)
			
		case <-ms.ctx.Done():
			return
		}
	}
}

// executeJob executes a single matrix job
func (ms *MatrixScheduler) executeJob(job *MatrixJob, runner *Runner, options *RunOptions, workerID int) {
	ms.mu.Lock()
	job.Status = StatusRunning
	job.StartTime = time.Now()
	ms.runningJobs[job.ID] = job
	ms.mu.Unlock()
	
	// Create matrix variables for this job
	matrixVars := make(map[string]string)
	for key, value := range job.Matrix {
		matrixVars[fmt.Sprintf("matrix.%s", key)] = fmt.Sprintf("%v", value)
	}
	
	// Store original variables and temporarily update with matrix variables
	originalVars := make(map[string]string)
	for k, v := range runner.opts.Variables {
		originalVars[k] = v
	}
	
	// Add matrix variables to runner's variables
	for k, v := range matrixVars {
		runner.opts.Variables[k] = v
	}
	
	// Execute the action without step callbacks (matrix scheduler handles callbacks)
	err := ms.executeActionWithoutCallbacks(runner, job.Action.Name)
	
	// Restore original variables
	runner.opts.Variables = originalVars
	
	ms.mu.Lock()
	job.EndTime = time.Now()
	
	if err != nil {
		job.Status = StatusError
		job.Result = &Result{
			Name:    job.Action.Name,
			Status:  StatusError,
			Message: err.Error(),
			Error:   err,
		}
		ms.failedCount++
	} else {
		job.Status = StatusOK
		job.Result = &Result{
			Name:    job.Action.Name,
			Status:  StatusOK,
			Message: "executed successfully",
		}
		ms.completedCount++
	}
	
	delete(ms.runningJobs, job.ID)
	ms.mu.Unlock()
	
	// Send to completed channel
	select {
	case ms.completedJobs <- job:
	case <-ms.ctx.Done():
	}
	
	// Check if we should stop due to fail_fast
	if ms.strategy.FailFast && job.Status == StatusError {
		if ms.opts != nil && ms.opts.Debug {
			fmt.Fprintf(ms.opts.ErrorOutput, "[DEBUG] Matrix fail_fast triggered, cancelling context\n")
		}
		ms.cancel()
	}
}

// monitorJobs monitors job completion and provides status updates
func (ms *MatrixScheduler) monitorJobs() {
	for {
		select {
		case job := <-ms.completedJobs:
			ms.printJobStatus(job)
		case <-ms.ctx.Done():
			return
		}
	}
}

// printJobStatus prints the status of a completed job
func (ms *MatrixScheduler) printJobStatus(job *MatrixJob) {
	ms.mu.RLock()
	totalJobs := len(ms.jobs)
	completed := ms.completedCount + ms.failedCount
	ms.mu.RUnlock()
	
	status := "✓"
	if job.Status == StatusError {
		status = "✗"
	}
	
	matrixStr := ms.formatMatrixValues(job.Matrix)
	fmt.Printf("[matrix] %s (%d/%d): %s - %s\n", status, completed, totalJobs, job.ID, matrixStr)
}

// formatMatrixValues formats matrix values for display
func (ms *MatrixScheduler) formatMatrixValues(matrix map[string]interface{}) string {
	var parts []string
	for key, value := range matrix {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// shouldStop checks if execution should stop due to fail_fast
func (ms *MatrixScheduler) shouldStop() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.strategy.FailFast && ms.failedCount > 0
}

// determineFinalStatus determines the final status based on strategy
func (ms *MatrixScheduler) determineFinalStatus() error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	if ms.strategy.ContinueOnError {
		// Stage succeeds even if some jobs failed
		return nil
	}
	
	if ms.failedCount > 0 {
		return fmt.Errorf("matrix execution failed: %d jobs failed", ms.failedCount)
	}
	
	return nil
}

// GetJobStatus returns the current status of all jobs
func (ms *MatrixScheduler) GetJobStatus() map[string]MatrixJobStatus {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	status := make(map[string]MatrixJobStatus)
	for _, job := range ms.jobs {
		switch job.Status {
		case StatusPending:
			status[job.ID] = MatrixJobPending
		case StatusRunning:
			status[job.ID] = MatrixJobRunning
		case StatusOK:
			status[job.ID] = MatrixJobCompleted
		case StatusError:
			status[job.ID] = MatrixJobFailed
		case StatusSkipped:
			status[job.ID] = MatrixJobSkipped
		}
	}
	
	return status
}

// GetJobResults returns the results of all completed jobs
func (ms *MatrixScheduler) GetJobResults() []*MatrixJob {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	results := make([]*MatrixJob, len(ms.jobs))
	copy(results, ms.jobs)
	return results
}
