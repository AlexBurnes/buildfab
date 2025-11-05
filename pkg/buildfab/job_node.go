package buildfab

import (
    "fmt"
    "time"
)

// JobNode represents a unit of work in the hierarchical DAG
// A job is a collection of steps that execute sequentially by default
// Jobs are linked via dependencies and can be executed in parallel subject to constraints
type JobNode struct {
    // Identity
    ID          string // Unique identifier: "0", "1", "0.1.2" for nested jobs
    DisplayName string // User-facing name: "gcc+Release", "centos8.clang"
    
    // Content
    Steps     []ExecutableStep // Sequential list of steps to execute in this job
    ChildJobs []*JobNode       // Nested jobs (for matrix within matrix scenarios)
    
    // Dependencies
    Dependencies      []string        // Job IDs this job depends on
    SlidingWindowDeps map[string]bool // Which dependencies are sliding window (for parallelism control)
    
    // Configuration
    MatrixVars  map[string]string // Matrix variables for this job and its steps
    MaxParallel int               // Parallelism limit for child jobs (0 = unlimited)
    FailFast    bool              // Stop on first child job failure
    
    // Runtime state
    Status      JobExecutionStatus    // Current execution status
    Results     []StepResult // Results of executed steps
    StartTime   time.Time    // When job execution started
    EndTime     time.Time    // When job execution completed
    Duration    time.Duration // Total execution time
    Error       error        // Error if job failed
}

// ExecutableStep represents a single action within a job
// Steps within a job execute sequentially by declaration order
type ExecutableStep struct {
    // Identity
    Index       int    // 0-based index within parent job
    ID          string // Unique: "jobID.stepIndex" (e.g., "0.2", "1.0.3")
    DisplayName string // User-facing: "build-action", "cleanup"
    
    // Configuration
    Action    *Action               // The action to execute (pointer to allow nil for nested jobs)
    Variables map[string]string     // Step-level variables (includes matrix vars from job)
    If        string                // Step-level condition expression
    OnError   string                // Error policy: "warn" or "stop" (default)
    
    // Dependencies within job (rarely used - steps are sequential by default)
    // These are step indices within the same job, not step IDs
    RequireSteps []int // Step indices this step depends on: [0, 1]
    
    // Runtime state
    Status   StepStatus
    Duration time.Duration
    Output   string
    Error    error
}

// JobExecutionStatus represents job execution status
type JobExecutionStatus int

const (
    JobExecutionStatusPending JobExecutionStatus = iota
    JobExecutionStatusRunning
    JobExecutionStatusOK                 // All steps completed successfully
    JobExecutionStatusWarn               // Some steps completed with warnings
    JobExecutionStatusError              // One or more steps failed
    JobExecutionStatusSkipped            // Skipped due to dependency failure
    JobExecutionStatusSkippedCondition   // Skipped due to if condition (doesn't block dependents)
    JobExecutionStatusPartial            // Some steps executed, some skipped
)

// String returns the string representation of JobExecutionStatus
func (s JobExecutionStatus) String() string {
    switch s {
    case JobExecutionStatusPending:
        return "pending"
    case JobExecutionStatusRunning:
        return "running"
    case JobExecutionStatusOK:
        return "ok"
    case JobExecutionStatusWarn:
        return "warn"
    case JobExecutionStatusError:
        return "error"
    case JobExecutionStatusSkipped:
        return "skipped"
    case JobExecutionStatusSkippedCondition:
        return "skipped_condition"
    case JobExecutionStatusPartial:
        return "partial"
    default:
        return "unknown"
    }
}

// HierarchicalDAG represents the complete execution plan as a hierarchy of jobs
type HierarchicalDAG struct {
    RootJobs []*JobNode                    // Top-level jobs
    AllSteps map[string]*ExecutableStep    // Quick lookup by step ID
    AllJobs  map[string]*JobNode           // Quick lookup by job ID
}

// NewJobNode creates a new job node
func NewJobNode(id, displayName string, matrixVars map[string]string) *JobNode {
    return &JobNode{
        ID:                id,
        DisplayName:       displayName,
        Steps:             make([]ExecutableStep, 0),
        ChildJobs:         make([]*JobNode, 0),
        Dependencies:      make([]string, 0),
        SlidingWindowDeps: make(map[string]bool),
        MatrixVars:        matrixVars,
        Status:            JobExecutionStatusPending,
        Results:           make([]StepResult, 0),
    }
}

// AddStep adds a step to the job
func (j *JobNode) AddStep(step ExecutableStep) {
    step.Index = len(j.Steps)
    step.ID = fmt.Sprintf("%s.%d", j.ID, step.Index)
    j.Steps = append(j.Steps, step)
}

// AddChildJob adds a nested job
func (j *JobNode) AddChildJob(child *JobNode) {
    j.ChildJobs = append(j.ChildJobs, child)
}

// AddDependency adds a job dependency
func (j *JobNode) AddDependency(jobID string, isSlidingWindow bool) {
    j.Dependencies = append(j.Dependencies, jobID)
    if isSlidingWindow {
        j.SlidingWindowDeps[jobID] = true
    }
}

// GetStepByIndex returns a step by its index within the job
func (j *JobNode) GetStepByIndex(index int) *ExecutableStep {
    if index < 0 || index >= len(j.Steps) {
        return nil
    }
    return &j.Steps[index]
}

// GetStepByID returns a step by its full ID
func (j *JobNode) GetStepByID(stepID string) *ExecutableStep {
    for i := range j.Steps {
        if j.Steps[i].ID == stepID {
            return &j.Steps[i]
        }
    }
    
    // Check child jobs recursively
    for _, child := range j.ChildJobs {
        if step := child.GetStepByID(stepID); step != nil {
            return step
        }
    }
    
    return nil
}

// IsComplete returns true if the job has completed (any terminal status)
func (j *JobNode) IsComplete() bool {
    switch j.Status {
    case JobExecutionStatusOK, JobExecutionStatusWarn, JobExecutionStatusError, JobExecutionStatusSkipped, JobExecutionStatusSkippedCondition, JobExecutionStatusPartial:
        return true
    default:
        return false
    }
}

// ShouldBlock returns true if this job's status should block dependent jobs
// Condition-based skips on sliding window dependencies don't block
func (j *JobNode) ShouldBlock() bool {
    switch j.Status {
    case JobExecutionStatusError, JobExecutionStatusSkipped:
        return true
    case JobExecutionStatusSkippedCondition:
        return false // Condition skips don't block sliding window deps
    default:
        return false
    }
}

// NewHierarchicalDAG creates a new hierarchical DAG
func NewHierarchicalDAG() *HierarchicalDAG {
    return &HierarchicalDAG{
        RootJobs: make([]*JobNode, 0),
        AllSteps: make(map[string]*ExecutableStep),
        AllJobs:  make(map[string]*JobNode),
    }
}

// AddRootJob adds a top-level job to the DAG
func (h *HierarchicalDAG) AddRootJob(job *JobNode) {
    h.RootJobs = append(h.RootJobs, job)
    h.registerJob(job)
}

// registerJob registers a job and all its steps/children in the lookup maps
func (h *HierarchicalDAG) registerJob(job *JobNode) {
    h.AllJobs[job.ID] = job
    
    // Register all steps
    for i := range job.Steps {
        h.AllSteps[job.Steps[i].ID] = &job.Steps[i]
    }
    
    // Recursively register child jobs
    for _, child := range job.ChildJobs {
        h.registerJob(child)
    }
}

// GetJob returns a job by ID
func (h *HierarchicalDAG) GetJob(jobID string) *JobNode {
    return h.AllJobs[jobID]
}

// GetStep returns a step by ID
func (h *HierarchicalDAG) GetStep(stepID string) *ExecutableStep {
    return h.AllSteps[stepID]
}

// GetAllJobs returns all jobs in the DAG (flattened from hierarchy)
func (h *HierarchicalDAG) GetAllJobs() []*JobNode {
    jobs := make([]*JobNode, 0, len(h.AllJobs))
    for _, job := range h.AllJobs {
        jobs = append(jobs, job)
    }
    return jobs
}

// CountTotalSteps returns the total number of steps across all jobs
func (h *HierarchicalDAG) CountTotalSteps() int {
    return len(h.AllSteps)
}

// CountTotalJobs returns the total number of jobs (including nested)
func (h *HierarchicalDAG) CountTotalJobs() int {
    return len(h.AllJobs)
}


// FlattenToSteps converts a hierarchical DAG to a flat ordered list of steps
// Steps are ordered by: job order (topological), then step index within each job
func (h *HierarchicalDAG) FlattenToSteps() []ExecutableStep {
    var flatSteps []ExecutableStep
    
    // Get jobs in topological order (respecting dependencies)
    orderedJobs := h.getJobsInTopologicalOrder()
    
    // For each job, add its steps in sequential order
    for _, job := range orderedJobs {
        flatSteps = append(flatSteps, h.flattenJobSteps(job)...)
    }
    
    return flatSteps
}

// getJobsInTopologicalOrder returns jobs in execution order (topological sort)
func (h *HierarchicalDAG) getJobsInTopologicalOrder() []*JobNode {
    // Simple topological sort using wave assignment
    jobMap := make(map[string]*JobNode)
    for _, job := range h.RootJobs {
        h.addJobToMap(job, jobMap)
    }
    
    assigned := make(map[string]bool)
    var ordered []*JobNode
    
    // Keep assigning jobs until all are assigned
    for len(assigned) < len(jobMap) {
        var currentWave []*JobNode
        
        // Find jobs where all dependencies are assigned
        for _, job := range jobMap {
            if assigned[job.ID] {
                continue
            }
            
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
            }
        }
        
        if len(currentWave) == 0 && len(assigned) < len(jobMap) {
            // No progress - cycle detected, break to avoid infinite loop
            break
        }
        
        // Add current wave to ordered list (maintain ID order within wave for determinism)
        ordered = append(ordered, currentWave...)
    }
    
    return ordered
}

// addJobToMap recursively adds a job and its children to the map
func (h *HierarchicalDAG) addJobToMap(job *JobNode, jobMap map[string]*JobNode) {
    jobMap[job.ID] = job
    for _, child := range job.ChildJobs {
        h.addJobToMap(child, jobMap)
    }
}

// flattenJobSteps flattens a job's steps (including nested jobs) to a list
func (h *HierarchicalDAG) flattenJobSteps(job *JobNode) []ExecutableStep {
    var steps []ExecutableStep
    
    // If job has child jobs, recurse into them
    if len(job.ChildJobs) > 0 {
        orderedChildren := h.getChildJobsInOrder(job.ChildJobs)
        for _, child := range orderedChildren {
            steps = append(steps, h.flattenJobSteps(child)...)
        }
    } else {
        // Job has direct steps
        steps = append(steps, job.Steps...)
    }
    
    return steps
}

// getChildJobsInOrder returns child jobs in dependency order
func (h *HierarchicalDAG) getChildJobsInOrder(childJobs []*JobNode) []*JobNode {
    // For now, return in ID order (could be enhanced with topological sort)
    // Child jobs typically don't have inter-dependencies
    return childJobs
}

// CountTotalJobs returns the total number of jobs in the DAG
