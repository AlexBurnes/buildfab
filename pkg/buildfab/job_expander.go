package buildfab

import (
    "fmt"
    "sort"
    "strings"
)

// JobExpander handles expansion of matrix configurations into job nodes
type JobExpander struct {
    config        *Config
    globalVars    map[string]string
    cliMatrixVars map[string]string
}

// NewJobExpander creates a new job expander
func NewJobExpander(config *Config, cliMatrixVars, globalVars map[string]string) *JobExpander {
    return &JobExpander{
        config:        config,
        globalVars:    globalVars,
        cliMatrixVars: cliMatrixVars,
    }
}

// ExpandMatrixToJobs expands a matrix step into job nodes
// For action steps, each job contains a single step
// For stage steps, each job contains all steps from the stage
func (je *JobExpander) ExpandMatrixToJobs(step *Step) ([]*JobNode, error) {
    if step.Matrix == nil {
        return nil, fmt.Errorf("step has no matrix configuration")
    }
    
    // Determine if this is an action or stage step
    if step.Action != "" {
        return je.expandActionMatrixToJobs(step)
    } else if step.Stage != "" {
        return je.expandStageMatrixToJobs(step)
    }
    
    return nil, fmt.Errorf("step must have either action or stage")
}

// expandActionMatrixToJobs expands a matrix on an action step
func (je *JobExpander) expandActionMatrixToJobs(step *Step) ([]*JobNode, error) {
    action, exists := je.config.GetAction(step.Action)
    if !exists {
        return nil, fmt.Errorf("action not found: %s", step.Action)
    }
    
    // Generate matrix combinations
    matrixValues := je.getEffectiveMatrixValues(step.Matrix)
    combinations := je.generateCombinations(matrixValues)
    
    var jobs []*JobNode
    for jobIdx, combination := range combinations {
        // Create matrix variables for this job
        matrixVars := je.combinationToVariables(combination)
        
        // Merge with global variables
        jobVars := je.mergeVariables(je.globalVars, matrixVars)
        
        // Generate job ID and display name
        jobID := fmt.Sprintf("%d", jobIdx)
        displayName := je.generateJobDisplayName(step.GetStepName(), combination, step.Matrix.Values)
        
        // Create job node
        job := NewJobNode(jobID, displayName, matrixVars)
        
        // Interpolate action with matrix variables
        interpolatedAction := je.interpolateAction(&action, jobVars)
        
        // Make a stable copy for the step (avoid loop variable address issues)
        actionForStep := new(Action)
        *actionForStep = interpolatedAction
        
        // Create single step for this job
        execStep := ExecutableStep{
            Index:       0,
            ID:          fmt.Sprintf("%s.0", jobID),
            DisplayName: step.GetStepName(),
            Action:      actionForStep,
            Variables:   matrixVars,
            If:          step.If,
            OnError:     step.OnError,
        }
        
        job.AddStep(execStep)
        
        // Add user dependencies to job (from the original step)
        for _, dep := range step.Require {
            job.AddDependency(dep, false) // Not a sliding window dependency
        }
        for _, dep := range step.DependsOn {
            job.AddDependency(dep, false)
        }
        
        jobs = append(jobs, job)
    }
    
    return jobs, nil
}

// expandStageMatrixToJobs expands a matrix on a stage reference
func (je *JobExpander) expandStageMatrixToJobs(step *Step) ([]*JobNode, error) {
    stage, exists := je.config.GetStage(step.Stage)
    if !exists {
        return nil, fmt.Errorf("stage not found: %s", step.Stage)
    }
    
    // Generate matrix combinations for outer matrix
    matrixValues := je.getEffectiveMatrixValues(step.Matrix)
    combinations := je.generateCombinations(matrixValues)
    
    var jobs []*JobNode
    for jobIdx, combination := range combinations {
        // Create matrix variables for this job
        matrixVars := je.combinationToVariables(combination)
        
        // Merge with global variables
        jobVars := je.mergeVariables(je.globalVars, matrixVars)
        
        // Generate job ID and display name
        jobID := fmt.Sprintf("%d", jobIdx)
        displayName := je.generateJobDisplayName(step.Stage, combination, step.Matrix.Values)
        
        // Create job node
        job := NewJobNode(jobID, displayName, matrixVars)
        
        // Store parent step's if condition in the job
        // The executor will evaluate this and skip the job if condition is not met
        job.If = step.If
        
        // Expand stage steps into this job's steps
        for stepIdx, stageStep := range stage.Steps {
            // Check if this stage step has its own matrix (nested matrix)
            if stageStep.Matrix != nil && stageStep.Action != "" {
                // This is a nested matrix - create child jobs
                childJobs, err := je.expandNestedMatrixToJobs(&stageStep, job.ID, jobVars)
                if err != nil {
                    return nil, fmt.Errorf("failed to expand nested matrix in job %s: %w", job.ID, err)
                }
                
                // Add child jobs to this job
                for _, childJob := range childJobs {
                    job.AddChildJob(childJob)
                }
            } else if stageStep.Action != "" {
                // Regular action step
                action, exists := je.config.GetAction(stageStep.Action)
                if !exists {
                    return nil, fmt.Errorf("action not found: %s", stageStep.Action)
                }
                
                // Interpolate action with job variables
                interpolatedAction := je.interpolateAction(&action, jobVars)
                
                // Create executable step
                execStep := ExecutableStep{
                    Index:       stepIdx,
                    DisplayName: stageStep.GetStepName(),
                    Action:      &interpolatedAction,
                    Variables:   je.mergeVariables(matrixVars, stageStep.Variables),
                    If:          stageStep.If,
                    OnError:     stageStep.OnError,
                }
                
                job.AddStep(execStep)
            }
            // TODO: Handle stageStep.Stage (nested stage reference)
        }
        
        jobs = append(jobs, job)
    }
    
    return jobs, nil
}

// expandNestedMatrixToJobs expands a nested matrix step into child jobs
func (je *JobExpander) expandNestedMatrixToJobs(step *Step, parentJobID string, parentVars map[string]string) ([]*JobNode, error) {
    action, exists := je.config.GetAction(step.Action)
    if !exists {
        return nil, fmt.Errorf("action not found: %s", step.Action)
    }
    
    // Generate matrix combinations for nested matrix
    matrixValues := je.getEffectiveMatrixValues(step.Matrix)
    combinations := je.generateCombinations(matrixValues)
    
    var childJobs []*JobNode
    for childIdx, combination := range combinations {
        // Create matrix variables for this child job
        childMatrixVars := je.combinationToVariables(combination)
        
        // Merge parent vars with child matrix vars (child overrides parent)
        jobVars := je.mergeVariables(parentVars, childMatrixVars)
        
        // Generate child job ID: parentID.childIndex
        childJobID := fmt.Sprintf("%s.%d", parentJobID, childIdx)
        
        // Generate display name that includes BOTH parent and child matrix values for uniqueness
        // Extract all matrix values from merged jobVars
        allCombination := make(map[string]interface{})
        for k, v := range jobVars {
            if strings.HasPrefix(k, "matrix.") {
                key := strings.TrimPrefix(k, "matrix.")
                allCombination[key] = v
            }
        }
        displayName := je.generateJobDisplayName(step.GetStepName(), allCombination, nil)
        
        // Create child job node
        childJob := NewJobNode(childJobID, displayName, childMatrixVars)
        
        // Interpolate action with merged variables
        interpolatedAction := je.interpolateAction(&action, jobVars)
        
        // Create single step for this child job
        execStep := ExecutableStep{
            Index:       0,
            DisplayName: step.GetStepName(),
            Action:      &interpolatedAction,
            Variables:   jobVars,
            If:          step.If,
            OnError:     step.OnError,
        }
        
        childJob.AddStep(execStep)
        childJobs = append(childJobs, childJob)
    }
    
    return childJobs, nil
}

// getEffectiveMatrixValues returns matrix values considering CLI overrides
func (je *JobExpander) getEffectiveMatrixValues(matrix *MatrixConfig) map[string][]interface{} {
    matrixValues := make(map[string][]interface{})
    for key, values := range matrix.Values {
        matrixValues[key] = values
    }
    
    // Override with CLI-provided matrix values
    for cliKey, cliValue := range je.cliMatrixVars {
        // Remove "matrix." prefix for key matching
        keyWithoutPrefix := strings.TrimPrefix(cliKey, "matrix.")
        if _, exists := matrixValues[keyWithoutPrefix]; exists {
            matrixValues[keyWithoutPrefix] = []interface{}{cliValue}
        }
    }
    
    return matrixValues
}

// generateCombinations creates Cartesian product of matrix values
func (je *JobExpander) generateCombinations(matrixValues map[string][]interface{}) []map[string]interface{} {
    if len(matrixValues) == 0 {
        return []map[string]interface{}{{}}
    }
    
    // Get sorted keys for deterministic ordering
    keys := make([]string, 0, len(matrixValues))
    for key := range matrixValues {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    
    // Generate Cartesian product
    var combinations []map[string]interface{}
    je.generateCombinationsRecursive(keys, 0, make(map[string]interface{}), matrixValues, &combinations)
    
    return combinations
}

// generateCombinationsRecursive generates combinations recursively
func (je *JobExpander) generateCombinationsRecursive(keys []string, keyIndex int, current map[string]interface{}, matrixValues map[string][]interface{}, combinations *[]map[string]interface{}) {
    if keyIndex >= len(keys) {
        // Make a copy of current combination
        combo := make(map[string]interface{})
        for k, v := range current {
            combo[k] = v
        }
        *combinations = append(*combinations, combo)
        return
    }
    
    key := keys[keyIndex]
    values := matrixValues[key]
    
    for _, value := range values {
        current[key] = value
        je.generateCombinationsRecursive(keys, keyIndex+1, current, matrixValues, combinations)
    }
}

// evaluateCondition evaluates a condition expression with given variables
func (je *JobExpander) evaluateCondition(condition string, variables map[string]string) (bool, error) {
    // Create expression context with variables
    ctx := NewExpressionContext(variables)
    
    // Evaluate the expression using the expression evaluator
    return EvaluateExpression(condition, ctx)
}

// combinationToVariables converts a combination map to matrix variables
func (je *JobExpander) combinationToVariables(combination map[string]interface{}) map[string]string {
    matrixVars := make(map[string]string)
    for key, value := range combination {
        matrixVars[fmt.Sprintf("matrix.%s", key)] = fmt.Sprintf("%v", value)
    }
    return matrixVars
}

// mergeVariables merges two variable maps (second overrides first)
func (je *JobExpander) mergeVariables(base, override map[string]string) map[string]string {
    merged := make(map[string]string)
    for k, v := range base {
        merged[k] = v
    }
    for k, v := range override {
        merged[k] = v
    }
    return merged
}

// generateJobDisplayName creates a display name for a job
func (je *JobExpander) generateJobDisplayName(baseName string, combination map[string]interface{}, originalValues map[string][]interface{}) string {
    var parts []string
    
    // Get sorted keys for consistent ordering
    keys := make([]string, 0, len(combination))
    for key := range combination {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    
    // Add all values in sorted order
    for _, key := range keys {
        if value, exists := combination[key]; exists {
            parts = append(parts, fmt.Sprintf("%v", value))
        }
    }
    
    if len(parts) > 0 {
        return fmt.Sprintf("%s.%s", baseName, strings.Join(parts, "."))
    }
    
    return baseName
}

// interpolateAction interpolates variables in an action
func (je *JobExpander) interpolateAction(action *Action, variables map[string]string) Action {
    interpolated := *action // Copy the action
    
    // Interpolate Run command
    if interpolated.Run != "" {
        interpolated.Run, _ = InterpolateVariables(interpolated.Run, variables)
    }
    
    // Interpolate container configuration if present
    if interpolated.Container != nil {
        // Create a copy of the container config
        containerCopy := *interpolated.Container
        
        // Interpolate container fields
        if containerCopy.Image.From != "" {
            containerCopy.Image.From, _ = InterpolateVariables(containerCopy.Image.From, variables)
        }
        if containerCopy.Run != "" {
            containerCopy.Run, _ = InterpolateVariables(containerCopy.Run, variables)
        }
        
        // Copy and interpolate env map
        if len(containerCopy.Env) > 0 {
            envCopy := make(map[string]string)
            for key, value := range containerCopy.Env {
                envCopy[key], _ = InterpolateVariables(value, variables)
            }
            containerCopy.Env = envCopy
        }
        
        interpolated.Container = &containerCopy
    }
    
    return interpolated
}

// InjectSlidingWindowDependencies adds dependencies between jobs for parallelism control
func (je *JobExpander) InjectSlidingWindowDependencies(jobs []*JobNode, maxParallel int) {
    if maxParallel <= 0 || len(jobs) <= maxParallel {
        // No dependencies needed - all jobs can run in parallel
        return
    }
    
    // Job i depends on job (i - maxParallel)
    for i := maxParallel; i < len(jobs); i++ {
        previousJobIdx := i - maxParallel
        previousJob := jobs[previousJobIdx]
        currentJob := jobs[i]
        
        // Add sliding window dependency: current job depends on previous job
        currentJob.AddDependency(previousJob.ID, true) // true = is sliding window
    }
}

