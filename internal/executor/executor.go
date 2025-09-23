// Package executor provides DAG-based execution functionality for buildfab
package executor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/AlexBurnes/buildfab/internal/actions"
	"github.com/AlexBurnes/buildfab/internal/version"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// Executor handles execution of buildfab stages and actions
type Executor struct {
	config         *buildfab.Config
	opts           *buildfab.RunOptions
	ui             UI
	registry       *actions.Registry
	versionDetector *version.Detector
}

// UI defines the interface for user interface operations
type UI interface {
	PrintCLIHeader(name, version string)
	PrintProjectCheck(projectName, version string)
	PrintStepStatus(stepName string, status buildfab.Status, message string)
	PrintStageHeader(stageName string)
	PrintStageResult(stageName string, success bool, duration time.Duration)
	PrintStageTerminated(stageName string, duration time.Duration)
	PrintCommand(command string)
	PrintStepName(stepName string)
	PrintCommandOutput(output string)
	PrintRepro(stepName, repro string)
	PrintReproInline(stepName, repro string)
	PrintSummary(results []buildfab.Result)
	IsVerbose() bool
	IsDebug() bool
}

// New creates a new executor
func New(config *buildfab.Config, opts *buildfab.RunOptions, ui UI) *Executor {
	return &Executor{
		config:         config,
		opts:           opts,
		ui:             ui,
		registry:       actions.New(),
		versionDetector: version.New(),
	}
}

// RunStage executes a specific stage
func (e *Executor) RunStage(ctx context.Context, stageName string) error {
	stage, exists := e.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// Print stage header
	if e.ui != nil {
		e.ui.PrintStageHeader(stageName)
	}

	start := time.Now()

	// Build execution DAG
	dag, err := e.buildDAG(stage.Steps)
	if err != nil {
		return fmt.Errorf("failed to build execution DAG: %w", err)
	}

	// Execute DAG with streaming output in declaration order
	results, err := e.executeDAGWithStreaming(ctx, dag, stage.Steps)
	
	duration := time.Since(start)
	
	// Check if execution was terminated due to context cancellation
	terminated := ctx.Err() != nil
	success := err == nil && !hasErrors(results) && !terminated
	
	
	if e.ui != nil {
		if terminated {
			e.ui.PrintStageTerminated(stageName, duration)
		} else {
			e.ui.PrintStageResult(stageName, success, duration)
		}
		e.ui.PrintSummary(results)
	}

	return err
}

// RunAction executes a specific action
func (e *Executor) RunAction(ctx context.Context, actionName string) error {
	action, exists := e.config.GetAction(actionName)
	if !exists {
		return fmt.Errorf("action not found: %s", actionName)
	}

	result, _ := e.executeAction(ctx, action, nil)
	if e.ui != nil {
		e.ui.PrintStepStatus(actionName, result.Status, result.Message)
	}
	
	if result.Error != nil {
		return result.Error
	}
	
	return nil
}

// RunStageStep executes a specific step within a stage
func (e *Executor) RunStageStep(ctx context.Context, stageName, stepName string) error {
	stage, exists := e.config.GetStage(stageName)
	if !exists {
		return fmt.Errorf("stage not found: %s", stageName)
	}

	// Find the step
	var targetStep *buildfab.Step
	for i, step := range stage.Steps {
		if step.Action == stepName {
			targetStep = &stage.Steps[i]
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step not found: %s in stage %s", stepName, stageName)
	}

	// Get the action
	action, exists := e.config.GetAction(targetStep.Action)
	if !exists {
		return fmt.Errorf("action not found: %s", targetStep.Action)
	}

	// Execute the action
	result, _ := e.executeAction(ctx, action, nil)
	if e.ui != nil {
		e.ui.PrintStepStatus(stepName, result.Status, result.Message)
	}
	
	if result.Error != nil {
		return result.Error
	}
	
	return nil
}

// ListActions returns all available actions
func (e *Executor) ListActions() []buildfab.Action {
	return e.config.Actions
}

// DAGNode represents a node in the execution DAG
type DAGNode struct {
	Step         buildfab.Step
	Action       buildfab.Action
	Dependencies []string
	Dependents   []string
}

// buildDAG builds the execution DAG from stage steps
func (e *Executor) buildDAG(steps []buildfab.Step) (map[string]*DAGNode, error) {
	dag := make(map[string]*DAGNode)
	
	// Create nodes for each step
	for _, step := range steps {
		action, exists := e.config.GetAction(step.Action)
		if !exists {
			return nil, fmt.Errorf("action not found: %s", step.Action)
		}
		
		node := &DAGNode{
			Step:         step,
			Action:       action,
			Dependencies: step.Require,
			Dependents:   []string{},
		}
		
		dag[step.Action] = node
	}
	
	// Build dependency relationships
	for _, node := range dag {
		for _, dep := range node.Dependencies {
			if depNode, exists := dag[dep]; exists {
				depNode.Dependents = append(depNode.Dependents, node.Step.Action)
			} else {
				return nil, fmt.Errorf("dependency not found: %s", dep)
			}
		}
	}
	
	// Check for cycles
	if err := e.detectCycles(dag); err != nil {
		return nil, fmt.Errorf("circular dependency detected: %w", err)
	}
	
	return dag, nil
}

// detectCycles detects cycles in the DAG using DFS
func (e *Executor) detectCycles(dag map[string]*DAGNode) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var dfs func(string) error
	dfs = func(nodeName string) error {
		if recStack[nodeName] {
			return fmt.Errorf("cycle detected involving node: %s", nodeName)
		}
		if visited[nodeName] {
			return nil
		}
		
		visited[nodeName] = true
		recStack[nodeName] = true
		defer func() { recStack[nodeName] = false }()
		
		node := dag[nodeName]
		for _, dep := range node.Dependencies {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		
		return nil
	}
	
	for nodeName := range dag {
		if !visited[nodeName] {
			if err := dfs(nodeName); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// StreamingOutputManager manages which step's output should be streamed
type StreamingOutputManager struct {
	steps     []buildfab.Step
	displayed map[string]bool
	started   map[string]bool
	doneStreaming map[string]bool
	buffers   map[string][]string // Buffer output for steps that can't stream yet
	mu        *sync.Mutex
}

// ShouldStreamOutput checks if the given step should have its output streamed
func (s *StreamingOutputManager) ShouldStreamOutput(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if this step itself has finished streaming - if so, don't stream
	if s.doneStreaming[stepName] {
		return false
	}
	
	// Allow streaming if this is the first step, or if all previous steps have finished streaming
	if stepIndex == 0 {
		return true
	}
	
	// Check if all previous steps in declaration order have finished streaming
	for i := 0; i < stepIndex; i++ {
		if !s.doneStreaming[s.steps[i].Action] {
			return false
		}
	}
	
	return true
}

// ShouldShowStepStart checks if the given step should show its start message
func (s *StreamingOutputManager) ShouldShowStepStart(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Allow all steps to start immediately, but only show start message for the first one
	// Check if this step itself has been started - if so, don't show start
	if s.started[stepName] {
		return false
	}
	
	// Only show start message for the first step in declaration order
	return stepIndex == 0
}

// MarkStepStarted marks a step as started
func (s *StreamingOutputManager) MarkStepStarted(stepName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started[stepName] = true
}

// MarkStepDisplayed marks a step as displayed
func (s *StreamingOutputManager) MarkStepDisplayed(stepName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.displayed[stepName] = true
}

// IsStepDisplayed checks if a step has been displayed
func (s *StreamingOutputManager) IsStepDisplayed(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.displayed[stepName]
}

// MarkStepDoneStreaming marks a step as done streaming
func (s *StreamingOutputManager) MarkStepDoneStreaming(stepName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doneStreaming[stepName] = true
}

// BufferOutput buffers output for a step that can't stream yet
func (s *StreamingOutputManager) BufferOutput(stepName, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffers[stepName] = append(s.buffers[stepName], line)
}

// FlushBufferedOutput flushes all buffered output for a step
func (s *StreamingOutputManager) FlushBufferedOutput(stepName string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	output := make([]string, len(s.buffers[stepName]))
	copy(output, s.buffers[stepName])
	s.buffers[stepName] = nil // Clear the buffer
	return output
}

// ShouldShowStepStartWhenActive checks if a step should show its start message when it becomes active
func (s *StreamingOutputManager) ShouldShowStepStartWhenActive(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if all previous steps in declaration order have finished streaming
	for i := 0; i < stepIndex; i++ {
		if !s.doneStreaming[s.steps[i].Action] {
			return false
		}
	}
	
	// Check if this step itself has been started - if so, don't show start
	if s.started[stepName] {
		return false
	}
	
	return true
}

// ShouldShowStepSuccess checks if a step should show its success message
func (s *StreamingOutputManager) ShouldShowStepSuccess(stepName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Find the step in declaration order
	stepIndex := -1
	for i, step := range s.steps {
		if step.Action == stepName {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if this step itself has been displayed - if so, don't show success
	if s.displayed[stepName] {
		return false
	}
	
	// For the first step, always show success when it completes
	if stepIndex == 0 {
		return true
	}
	
	// For subsequent steps, check if all previous steps in declaration order have been displayed
	for i := 0; i < stepIndex; i++ {
		if !s.displayed[s.steps[i].Action] {
			return false
		}
	}
	
	return true
}

// executeDAGWithStreaming executes the DAG with streaming output in declaration order
func (e *Executor) executeDAGWithStreaming(ctx context.Context, dag map[string]*DAGNode, steps []buildfab.Step) ([]buildfab.Result, error) {
	var results []buildfab.Result
	completed := make(map[string]bool)
	failed := make(map[string]bool)
	displayed := make(map[string]bool)
	executing := make(map[string]bool)
	started := make(map[string]bool)
	
	// Create a map of results by step name for quick lookup
	resultMap := make(map[string]buildfab.Result)
	
	// Create channels for communication
	resultChan := make(chan buildfab.Result, len(dag))
	done := make(chan bool)
	ctxDone := ctx.Done()
	
	// Mutex for thread-safe access to shared state
	var mu sync.Mutex
	var channelClosed sync.Once
	
	// Create a streaming output manager
	streamingManager := &StreamingOutputManager{
		steps:     steps,
		displayed: displayed,
		started:   started,
		doneStreaming: make(map[string]bool),
		buffers:   make(map[string][]string),
		mu:        &mu,
	}
	
	// Safe send function that checks if channel is closed
	safeSend := func(result buildfab.Result) {
		// Check if context is done first
		select {
		case <-ctxDone:
			// Context cancelled, don't send
			return
		default:
			// Context not cancelled, try to send
		}
		
		// Try to send with timeout
		select {
		case resultChan <- result:
		case <-ctxDone:
			// Context cancelled while trying to send
		case <-time.After(100 * time.Millisecond):
			// Timeout, channel might be closed
		}
	}
	
	// Start a goroutine to handle results and display them
	go func() {
		defer close(done)
		for result := range resultChan {
			mu.Lock()
			results = append(results, result)
			resultMap[result.Name] = result
			completed[result.Name] = true
			executing[result.Name] = false
			
			if result.Status == buildfab.StatusError {
				failed[result.Name] = true
			}
			mu.Unlock()
			
			// Display immediately if it's ready in declaration order
			e.displayStepImmediately(result.Name, steps, resultMap, displayed, completed, streamingManager)
		}
	}()
	
	// Start execution goroutine that continuously starts new steps
	go func() {
		defer func() {
			// Wait for all executing goroutines to complete before closing the channel
			for {
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Check for context cancellation while waiting
				select {
				case <-ctxDone:
					channelClosed.Do(func() { close(resultChan) })
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
			}
			close(resultChan)
		}()
		
		for {
			// Check for context cancellation at the start of each loop iteration
			select {
			case <-ctx.Done():
				// Context cancelled, exit immediately
				return
			default:
				// Continue with execution
			}
			
			mu.Lock()
			readySteps := e.getReadyStepsLocked(dag, completed, failed, executing)
			mu.Unlock()
			
			if len(readySteps) == 0 {
				// Check if all steps are completed or executing
				mu.Lock()
				allDone := true
				for nodeName := range dag {
					if !completed[nodeName] && !executing[nodeName] {
						allDone = false
						break
					}
				}
				mu.Unlock()
				
				if allDone {
					break
				}
				
				// Wait a bit before checking again, but check for cancellation
				select {
				case <-ctx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					// Continue waiting
				}
				continue
			}
			
			// Start ready steps immediately without waiting
			for _, nodeName := range readySteps {
				// Check for context cancellation before starting each step
				select {
				case <-ctx.Done():
					return
				default:
					// Continue with step execution
				}
				
				node := dag[nodeName]
				
				mu.Lock()
				executing[nodeName] = true
				mu.Unlock()
				
				// Skip if already failed and this node requires it
				if e.hasFailedDependency(node, failed) {
					failedDeps := e.getFailedDependencyNames(node, failed)
					result := buildfab.Result{
						Name:   nodeName,
						Status: buildfab.StatusSkipped,
						Message: fmt.Sprintf("skipped (dependency failed: %s)", strings.Join(failedDeps, ", ")),
					}
					safeSend(result)
					continue
				}
				
				// Check if step should be executed based on conditions
				if !e.shouldExecuteStep(ctx, node) {
					result := buildfab.Result{
						Name:   nodeName,
						Status: buildfab.StatusOK,
						Message: "skipped (condition not met)",
					}
					safeSend(result)
					continue
				}
				
				// Execute the node in parallel
				go func(nodeName string, node *DAGNode) {
					result, _ := e.executeAction(ctx, node.Action, streamingManager)
					result.Name = nodeName
					// Check if context was cancelled during execution
					if ctx.Err() != nil {
						result.Status = buildfab.StatusError
						result.Message = "cancelled"
						result.Error = ctx.Err()
					}
					safeSend(result)
				}(nodeName, node)
			}
		}
	}()
	
	<-done
	
	// Display any remaining steps that weren't displayed yet
	e.displayRemainingSteps(steps, resultMap, displayed, streamingManager)
	
	return results, nil
}

// displayStepImmediately displays a step immediately if it can be shown (streaming)
func (e *Executor) displayStepImmediately(stepName string, steps []buildfab.Step, resultMap map[string]buildfab.Result, displayed map[string]bool, completed map[string]bool, streamingManager *StreamingOutputManager) {
	// Find the step in declaration order
	for _, step := range steps {
		if step.Action == stepName {
			// Check if all previous steps in declaration order have been completed
			if e.canDisplayStepImmediately(step, steps, displayed, completed, streamingManager) {
				if result, exists := resultMap[stepName]; exists {
					// Use streaming manager to control when success messages are displayed
					if streamingManager == nil || streamingManager.ShouldShowStepSuccess(stepName) {
						if e.ui != nil {
							e.ui.PrintStepStatus(stepName, result.Status, result.Message)
						}
						// Use streaming manager to mark as displayed to avoid race conditions
						if streamingManager != nil {
							streamingManager.MarkStepDisplayed(stepName)
						} else {
							displayed[stepName] = true
						}
					}
				}
			}
			break
		}
	}
}

// canDisplayStepImmediately checks if a step can be displayed immediately (streaming)
func (e *Executor) canDisplayStepImmediately(step buildfab.Step, steps []buildfab.Step, displayed map[string]bool, completed map[string]bool, streamingManager *StreamingOutputManager) bool {
	// Find the position of this step in the declaration order
	stepIndex := -1
	for i, s := range steps {
		if s.Action == step.Action {
			stepIndex = i
			break
		}
	}
	
	if stepIndex == -1 {
		return false
	}
	
	// Check if all previous steps in declaration order have been displayed
	// This allows true streaming - steps can complete out of order but display in order
	for i := 0; i < stepIndex; i++ {
		// Use streaming manager if available, otherwise use local map
		isDisplayed := false
		if streamingManager != nil {
			isDisplayed = streamingManager.IsStepDisplayed(steps[i].Action)
		} else {
			isDisplayed = displayed[steps[i].Action]
		}
		if !isDisplayed {
			return false
		}
	}
	
	return true
}

// getReadyStepsLocked returns steps that are ready to execute (thread-safe version)
func (e *Executor) getReadyStepsLocked(dag map[string]*DAGNode, completed map[string]bool, failed map[string]bool, executing map[string]bool) []string {
	var ready []string
	
	for nodeName, node := range dag {
		if completed[nodeName] || executing[nodeName] {
			continue
		}
		
		// Check if all dependencies are completed
		if e.allDependenciesCompleted(node, completed) {
			ready = append(ready, nodeName)
		}
	}
	
	return ready
}

// displayRemainingSteps displays any steps that weren't displayed yet
func (e *Executor) displayRemainingSteps(steps []buildfab.Step, resultMap map[string]buildfab.Result, displayed map[string]bool, streamingManager *StreamingOutputManager) {
	for _, step := range steps {
		// Check if step is already displayed using streaming manager or local map
		isDisplayed := false
		if streamingManager != nil {
			isDisplayed = streamingManager.IsStepDisplayed(step.Action)
		} else {
			isDisplayed = displayed[step.Action]
		}
		
		if !isDisplayed {
			if result, exists := resultMap[step.Action]; exists {
				// Use streaming manager to control when success messages are displayed
				if streamingManager == nil || streamingManager.ShouldShowStepSuccess(step.Action) {
					if e.ui != nil {
						e.ui.PrintStepStatus(step.Action, result.Status, result.Message)
					}
					// Use streaming manager to mark as displayed to avoid race conditions
					if streamingManager != nil {
						streamingManager.MarkStepDisplayed(step.Action)
					} else {
						displayed[step.Action] = true
					}
				}
			}
		}
	}
}

// allDependenciesCompleted checks if all dependencies are completed
func (e *Executor) allDependenciesCompleted(node *DAGNode, completed map[string]bool) bool {
	for _, dep := range node.Dependencies {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// hasFailedDependency checks if any required dependency has failed
func (e *Executor) hasFailedDependency(node *DAGNode, failed map[string]bool) bool {
	for _, dep := range node.Dependencies {
		if failed[dep] {
			return true
		}
	}
	return false
}

// getFailedDependencyNames returns the names of failed dependencies
func (e *Executor) getFailedDependencyNames(node *DAGNode, failed map[string]bool) []string {
	var failedDeps []string
	for _, dep := range node.Dependencies {
		if failed[dep] {
			failedDeps = append(failedDeps, dep)
		}
	}
	return failedDeps
}

// executeAction executes a single action
func (e *Executor) executeAction(ctx context.Context, action buildfab.Action, streamingManager *StreamingOutputManager) (buildfab.Result, error) {
	// Check if context is already cancelled before starting
	if ctx.Err() != nil {
		return buildfab.Result{
			Status:  buildfab.StatusError,
			Message: "cancelled",
			Error:   ctx.Err(),
		}, ctx.Err()
	}

	// Call step start callback if provided
	if e.opts.StepCallback != nil {
		e.opts.StepCallback.OnStepStart(ctx, action.Name)
	}

	var result buildfab.Result
	var err error

	if action.Uses != "" {
		result, err = e.executeBuiltInAction(ctx, action)
	} else {
		result, err = e.executeCustomAction(ctx, action, streamingManager)
	}

	// Call step complete callback if provided
	if e.opts.StepCallback != nil {
		status := buildfab.StepStatusOK
		message := "executed successfully"
		
		if err != nil {
			status = buildfab.StepStatusError
			message = err.Error()
			e.opts.StepCallback.OnStepError(ctx, action.Name, err)
		} else if result.Status == buildfab.StatusWarn {
			status = buildfab.StepStatusWarn
			message = result.Message
		} else if result.Status == buildfab.StatusError {
			status = buildfab.StepStatusError
			message = result.Message
		} else if result.Status == buildfab.StatusSkipped {
			status = buildfab.StepStatusSkipped
			message = result.Message
		}
		
		e.opts.StepCallback.OnStepComplete(ctx, action.Name, status, message, result.Duration)
	}

	return result, err
}

// executeBuiltInAction executes a built-in action
func (e *Executor) executeBuiltInAction(ctx context.Context, action buildfab.Action) (buildfab.Result, error) {
	runner, exists := e.registry.GetRunner(action.Uses)
	if !exists {
		return buildfab.Result{
			Status:  buildfab.StatusError,
			Message: fmt.Sprintf("unknown built-in action: %s", action.Uses),
		}, fmt.Errorf("unknown built-in action: %s", action.Uses)
	}
	
	result, err := runner.Run(ctx)
	
	// Print output using UI interface
	if e.ui != nil && result.Message != "" {
		e.ui.PrintCommandOutput(result.Message)
	}
	
	return result, err
}

// executeCustomAction executes a custom action with run command
func (e *Executor) executeCustomAction(ctx context.Context, action buildfab.Action, streamingManager *StreamingOutputManager) (buildfab.Result, error) {
	// Check if context is already cancelled before starting
	if ctx.Err() != nil {
		return buildfab.Result{
			Status:  buildfab.StatusError,
			Message: "cancelled",
			Error:   ctx.Err(),
		}, ctx.Err()
	}

	if action.Run == "" {
		return buildfab.Result{
			Status:  buildfab.StatusError,
			Message: "no run command specified",
		}, fmt.Errorf("no run command specified for action %s", action.Name)
	}

	// Print step name only if this step should show its start message
	if e.ui != nil && (streamingManager == nil || streamingManager.ShouldShowStepStart(action.Name)) {
		e.ui.PrintStepName(action.Name)
		if streamingManager != nil {
			streamingManager.MarkStepStarted(action.Name)
		}
	}

	// Execute the command with streaming output
	cmd := exec.CommandContext(ctx, "sh", "-c", action.Run)
	
	var err error
	if e.opts.Verbose {
		// Use streaming output for verbose mode
		err = e.executeCommandWithStreaming(ctx, cmd, action.Name, streamingManager)
	} else {
		// Use buffered output for non-verbose mode
		output, cmdErr := cmd.CombinedOutput()
		err = cmdErr
		
		// Print output using UI interface or buffer it
		if e.ui != nil && len(output) > 0 {
			lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					if streamingManager == nil || streamingManager.ShouldStreamOutput(action.Name) {
						e.ui.PrintCommandOutput(line)
					} else {
						// Buffer output for later when this step becomes active
						streamingManager.BufferOutput(action.Name, line)
					}
				}
			}
		}
		
		// Mark as done streaming for non-verbose mode too
		if streamingManager != nil {
			streamingManager.MarkStepDoneStreaming(action.Name)
			// Check if this step should display its success message
			e.checkAndDisplayStepSuccess(action.Name, streamingManager)
			// Check if next step should now start streaming
			e.checkAndStartNextStep(action.Name, streamingManager)
		}
	}
	
	if err != nil {
		// For custom actions, provide the exact command to run manually
		return buildfab.Result{
			Status:  buildfab.StatusError,
			Message: fmt.Sprintf("Command failed. To debug run:\n%s", action.Run),
		}, fmt.Errorf("command failed: %w", err)
	}
	
	return buildfab.Result{
		Status:  buildfab.StatusOK,
		Message: "command executed successfully",
	}, nil
}

// executeCommandWithStreaming executes a command with real-time output streaming
func (e *Executor) executeCommandWithStreaming(ctx context.Context, cmd *exec.Cmd, actionName string, streamingManager *StreamingOutputManager) error {
	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Check if this step should show its start message when it becomes active
	if e.ui != nil && streamingManager != nil && streamingManager.ShouldShowStepStartWhenActive(actionName) {
		e.ui.PrintStepName(actionName)
		streamingManager.MarkStepStarted(actionName)
	}
	
	// Create channels for goroutine communication
	done := make(chan error, 1)
	streamingDone := make(chan bool, 2) // One for stdout, one for stderr
	
	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Check if this step should show its start message when it becomes active
			if e.ui != nil && streamingManager != nil && streamingManager.ShouldShowStepStartWhenActive(actionName) {
				e.ui.PrintStepName(actionName)
				streamingManager.MarkStepStarted(actionName)
			}
			
			// Check streaming status dynamically for each line
			if e.ui != nil {
				if streamingManager == nil || streamingManager.ShouldStreamOutput(actionName) {
					e.ui.PrintCommandOutput(line)
				} else {
					// Buffer output for later when this step becomes active
					streamingManager.BufferOutput(actionName, line)
				}
			}
		}
		// Signal that stdout is done
		streamingDone <- true
	}()
	
	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Check if this step should show its start message when it becomes active
			if e.ui != nil && streamingManager != nil && streamingManager.ShouldShowStepStartWhenActive(actionName) {
				e.ui.PrintStepName(actionName)
				streamingManager.MarkStepStarted(actionName)
			}
			
			// Check streaming status dynamically for each line
			if e.ui != nil {
				if streamingManager == nil || streamingManager.ShouldStreamOutput(actionName) {
					e.ui.PrintCommandOutput(line)
				} else {
					// Buffer output for later when this step becomes active
					streamingManager.BufferOutput(actionName, line)
				}
			}
		}
		// Signal that stderr is done
		streamingDone <- true
	}()
	
	// Wait for both stdout and stderr to finish streaming
	go func() {
		<-streamingDone // Wait for stdout
		<-streamingDone // Wait for stderr
		// Mark as done streaming when both are finished
		if streamingManager != nil {
			streamingManager.MarkStepDoneStreaming(actionName)
			// Check if this step should display its success message
			e.checkAndDisplayStepSuccess(actionName, streamingManager)
			// Check if next step should now start streaming
			e.checkAndStartNextStep(actionName, streamingManager)
		}
	}()
	
	// Wait for command completion
	go func() {
		done <- cmd.Wait()
	}()
	
	// Wait for either completion or context cancellation
	select {
	case err := <-done:
		// Command completed normally
		return err
	case <-ctx.Done():
		// Context cancelled, kill the command
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return ctx.Err()
	}
}

// checkAndStartNextStep checks if the next step should start streaming and shows its start message
func (e *Executor) checkAndStartNextStep(completedStepName string, streamingManager *StreamingOutputManager) {
	// Find the next step in declaration order
	nextStepIndex := -1
	for i, step := range streamingManager.steps {
		if step.Action == completedStepName {
			nextStepIndex = i + 1
			break
		}
	}
	
	if nextStepIndex >= len(streamingManager.steps) {
		return // No next step
	}
	
	nextStepName := streamingManager.steps[nextStepIndex].Action
	
	// Check if the next step should show its start message and start streaming
	if streamingManager.ShouldShowStepStartWhenActive(nextStepName) {
		if e.ui != nil {
			e.ui.PrintStepName(nextStepName)
			streamingManager.MarkStepStarted(nextStepName)
		}
	}
	
	// Flush any buffered output for the next step
	if e.ui != nil {
		bufferedOutput := streamingManager.FlushBufferedOutput(nextStepName)
		for _, line := range bufferedOutput {
			e.ui.PrintCommandOutput(line)
		}
	}
}

// checkAndDisplayStepSuccess checks if a step should display its success message and displays it
func (e *Executor) checkAndDisplayStepSuccess(stepName string, streamingManager *StreamingOutputManager) {
	if streamingManager == nil || !streamingManager.ShouldShowStepSuccess(stepName) {
		return
	}
	
	// Find the step result
	for _, result := range streamingManager.steps {
		if result.Action == stepName {
			// Display the success message
			if e.ui != nil {
				e.ui.PrintStepStatus(stepName, buildfab.StatusOK, "command executed successfully")
			}
			streamingManager.MarkStepDisplayed(stepName)
			break
		}
	}
}

// getVersion returns the current version from the VERSION file
func (e *Executor) getVersion() string {
	// Try to read from VERSION file first
	if version, err := e.readVersionFile(); err == nil {
		return version
	}
	
	// Fallback to git tag detection
	if version, err := e.versionDetector.DetectCurrentVersion(context.Background()); err == nil {
		return version
	}
	
	// Final fallback
	return "unknown"
}

// readVersionFile reads the version from the VERSION file
func (e *Executor) readVersionFile() (string, error) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "", err
	}
	
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}
	
	return version, nil
}

// shouldExecuteStep determines if a step should be executed based on conditions
func (e *Executor) shouldExecuteStep(ctx context.Context, node *DAGNode) bool {
	// Check 'if' condition
	if node.Step.If != "" {
		if !e.evaluateCondition(ctx, node.Step.If) {
			return false
		}
	}
	
	// Check 'only' conditions
	if len(node.Step.Only) > 0 {
		if !e.matchesOnlyConditions(ctx, node.Step.Only) {
			return false
		}
	}
	
	return true
}

// evaluateCondition evaluates a condition string
func (e *Executor) evaluateCondition(ctx context.Context, condition string) bool {
	// For now, support simple conditions like "version.type == 'release'"
	// This is a simplified implementation - in a real system, you'd want a proper expression parser
	
	if condition == "version.type == 'release'" {
		versionType, err := e.getVersionType(ctx)
		return err == nil && versionType == "release"
	}
	
	if condition == "version.type == 'prerelease'" {
		versionType, err := e.getVersionType(ctx)
		return err == nil && versionType == "prerelease"
	}
	
	// Default to true for unknown conditions
	return true
}

// matchesOnlyConditions checks if current context matches any of the 'only' conditions
func (e *Executor) matchesOnlyConditions(ctx context.Context, onlyConditions []string) bool {
	versionType, err := e.getVersionType(ctx)
	if err != nil {
		// If we can't determine version type, don't execute steps with 'only' conditions
		return false
	}
	
	for _, condition := range onlyConditions {
		if condition == versionType {
			return true
		}
	}
	
	return false
}

// getVersionType determines the type of version (release, prerelease, patch, minor, major)
func (e *Executor) getVersionType(ctx context.Context) (string, error) {
	// Use version-go library to determine type
	parsedVersion, err := e.versionDetector.DetectVersionInfo(ctx)
	if err != nil {
		return "", err
	}
	
	return parsedVersion.Type, nil
}

// hasErrors checks if any results have errors
func hasErrors(results []buildfab.Result) bool {
	for _, result := range results {
		if result.Status == buildfab.StatusError {
			return true
		}
	}
	return false
}