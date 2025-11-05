# Hierarchical DAG Architecture Design

## Problem Statement

The current flat DAG architecture has fundamental issues with nested matrices and multi-step stages:

1. **No implicit ordering**: Steps within a stage have no inherent sequential relationship
2. **Sliding window complexity**: Dependencies are injected between individual steps, not jobs
3. **Guess-based detection**: `findFirstSteps`/`findLastSteps` must guess job boundaries
4. **Cascading failures**: Condition-based skips in one step block unrelated steps
5. **Nested matrix failure**: Matrix on stage with matrix steps creates ambiguous dependencies

## Proposed Solution: Hierarchical DAG

### Core Concept

Instead of a flat list of steps, use a **tree of job nodes**, where:
- Each **job node** represents a matrix combination
- Each job contains a **sequential list of steps**
- Jobs are linked via **job-level dependencies**
- Steps within a job execute **sequentially by default**

### Data Structures

```go
// JobNode represents a unit of work in the hierarchical DAG
type JobNode struct {
    // Identity
    ID          string              // Unique identifier: "0", "0.1", "0.1.2" for nested
    DisplayName string              // User-facing name: "gcc+Release", "centos8+clang"
    
    // Content
    Steps       []ExecutableStep    // Sequential list of steps to execute
    ChildJobs   []JobNode           // Nested jobs (for matrix within matrix)
    
    // Dependencies
    Dependencies        []string           // Job IDs this depends on
    SlidingWindowDeps   map[string]bool    // Which deps are sliding window (for parallelism)
    
    // Configuration
    MatrixVars  map[string]string   // Matrix variables for this job
    MaxParallel int                 // Parallelism limit for child jobs
    FailFast    bool                // Stop on first child job failure
    
    // Runtime state
    Status      JobStatus           // Pending, Running, OK, Warn, Error, Skipped
    Results     []StepResult        // Results of executed steps
}

// ExecutableStep represents a single action within a job
type ExecutableStep struct {
    // Identity
    Index       int                 // 0-based index within job
    ID          string              // Unique: "jobID.stepIndex" (e.g., "0.2", "1.0.3")
    DisplayName string              // User-facing: "build-action", "cleanup"
    
    // Configuration
    Action      Action              // The action to execute
    Variables   map[string]string   // Step-level variables (includes matrix vars)
    If          string              // Step-level condition
    OnError     string              // Error policy: "warn" or "stop"
    
    // Dependencies (within job - rarely used, jobs are sequential by default)
    Require     []string            // Step indices within same job: ["0", "1"]
}

// JobStatus represents job execution status
type JobStatus int

const (
    JobStatusPending JobStatus = iota
    JobStatusRunning
    JobStatusOK                    // All steps OK
    JobStatusWarn                  // Some steps warned
    JobStatusError                 // Some steps failed
    JobStatusSkipped               // Skipped due to dependency
    JobStatusSkippedCondition      // Skipped due to if condition (doesn't block)
    JobStatusPartial               // Some steps executed, some skipped
)

// HierarchicalDAG represents the complete execution plan
type HierarchicalDAG struct {
    RootJobs    []JobNode           // Top-level jobs
    AllSteps    map[string]*ExecutableStep  // Quick lookup by step ID
    AllJobs     map[string]*JobNode // Quick lookup by job ID
}
```

### Matrix Expansion Algorithm

#### Single-Level Matrix (Action)

```yaml
- action: build
  matrix:
    values:
      compiler: ["gcc", "clang"]
```

**Expansion**:
```
Job 0 (gcc):
  - Step 0: build [with matrix.compiler=gcc]

Job 1 (clang):
  - Step 0: build [with matrix.compiler=clang]
```

#### Single-Level Matrix (Stage)

```yaml
- stage: build-stage  # Contains: action1, action2, action3
  matrix:
    values:
      compiler: ["gcc", "clang"]
```

**Expansion**:
```
Job 0 (gcc):
  - Step 0: action1 [with matrix.compiler=gcc]
  - Step 1: action2 [with matrix.compiler=gcc]
  - Step 2: action3 [with matrix.compiler=gcc]

Job 1 (clang):
  - Step 0: action1 [with matrix.compiler=clang]
  - Step 1: action2 [with matrix.compiler=clang]
  - Step 2: action3 [with matrix.compiler=clang]
```

#### Nested Matrix (Stage with Matrix referencing Stage with Matrix)

```yaml
stages:
  inner-stage:
    steps:
      - action: build
        matrix:
          values:
            type: ["Debug", "Release"]
  
  outer-stage:
    steps:
      - stage: inner-stage
        matrix:
          values:
            compiler: ["gcc", "clang"]
```

**Expansion**:
```
Job 0 (gcc):
  ChildJob 0.0 (gcc+Debug):
    - Step 0.0.0: build [compiler=gcc, type=Debug]
  ChildJob 0.1 (gcc+Release):
    - Step 0.1.0: build [compiler=gcc, type=Release]

Job 1 (clang):
  ChildJob 1.0 (clang+Debug):
    - Step 1.0.0: build [compiler=clang, type=Debug]
  ChildJob 1.1 (clang+Release):
    - Step 1.1.0: build [compiler=clang, type=Release]
```

### DAG Execution Algorithm

```go
func (h *HierarchicalDAG) Execute(ctx context.Context) error {
    // Execute root jobs with sliding window dependencies
    return h.executeJobs(ctx, h.RootJobs)
}

func (h *HierarchicalDAG) executeJobs(ctx context.Context, jobs []JobNode) error {
    // Build job dependency graph
    jobDAG := h.buildJobDAG(jobs)
    
    // Execute jobs in topological order with parallelism control
    for wave := range h.getExecutionWaves(jobDAG) {
        // Execute jobs in this wave in parallel
        for _, job := range wave {
            go h.executeJob(ctx, job)
        }
        // Wait for wave to complete
    }
}

func (h *HierarchicalDAG) executeJob(ctx context.Context, job *JobNode) error {
    // Check job-level if condition
    if job has if condition && !evaluate(job.If) {
        job.Status = JobStatusSkippedCondition
        return nil  // Don't block dependent jobs
    }
    
    // Check job dependencies
    if hasFailedDependency(job) {
        job.Status = JobStatusSkipped
        return error // Block dependent jobs
    }
    
    // Execute steps sequentially within job
    for _, step := range job.Steps {
        // Check step-level if condition
        if step.If != "" && !evaluate(step.If) {
            continue // Skip this step, continue with next
        }
        
        // Execute step
        result := executeStep(ctx, step)
        job.Results = append(job.Results, result)
        
        // Handle errors per step.OnError policy
        if result.Error != nil {
            if step.OnError == "stop" {
                job.Status = JobStatusError
                return result.Error
            }
            // Continue with warnings
        }
    }
    
    // If job has child jobs (nested matrix), execute them
    if len(job.ChildJobs) > 0 {
        return h.executeJobs(ctx, job.ChildJobs)
    }
    
    job.Status = JobStatusOK
    return nil
}
```

### Sliding Window Dependencies

**At Job Level:**
```go
if maxParallel == 1:
    Job 1 depends on Job 0
    Job 2 depends on Job 1
    Job 3 depends on Job 2
```

**Key Insight:** If Job 1 is entirely skipped due to condition, Job 2 still executes because it's a sliding window dependency!

### Step Naming

**Step ID** (for DAG execution):
- Format: `{jobID}.{stepIndex}`
- Examples: `"0.0"`, `"0.1"`, `"1.0"`, `"0.1.2"` (nested)
- Used internally for dependency resolution

**Step Display Name** (for user output):
- Format: `{jobDisplayName}.{stepName}`
- Examples: `"gcc+Release.build-action"`, `"clang+Debug.cleanup"`
- Used in output manager for user-facing messages

### Migration Strategy

1. **Phase 1**: Implement new data structures alongside old ones
2. **Phase 2**: Add `--experimental-hierarchical-dag` flag
3. **Phase 3**: Run both implementations in parallel, compare results
4. **Phase 4**: Switch default to hierarchical, keep flat as fallback
5. **Phase 5**: Remove old flat implementation

## Implementation Steps

### Step 1: Core Data Structures (`pkg/buildfab/job_node.go`)
- Define `JobNode`, `ExecutableStep`, `JobStatus`
- Define `HierarchicalDAG`
- Basic constructor functions

### Step 2: Job-Based Matrix Expansion (`pkg/buildfab/job_expander.go`)
- `ExpandMatrixToJobs(step, action)` → creates job nodes
- `ExpandMatrixStageToJobs(step, stage)` → creates jobs with multiple steps
- Handle nested matrices → child jobs

### Step 3: Hierarchical DAG Builder (`pkg/buildfab/hierarchical_dag.go`)
- Build job dependency graph
- Inject sliding window dependencies at job level
- Detect cycles in job graph
- Generate execution waves

### Step 4: Hierarchical Executor (`pkg/buildfab/hierarchical_executor.go`)
- Execute jobs in topological waves
- Execute steps sequentially within each job
- Handle job-level and step-level conditions separately
- Proper error propagation with OnError policies

### Step 5: Job-Aware Output Manager (`pkg/buildfab/job_output_manager.go`)
- Display jobs with their steps
- Use step IDs internally, display names for users
- Group output by job
- Show job status separately from step status

### Step 6: Integration & Testing
- Wire up hierarchical DAG in `RunStage`
- Add feature flag for gradual rollout
- Comprehensive tests for all scenarios
- Update documentation

## Benefits

✅ **Correctness**: Proper semantics for nested matrices
✅ **Clarity**: Job boundaries are explicit, not inferred
✅ **Performance**: Better parallelism control at job level
✅ **Extensibility**: Easy to add job-level features (timeouts, retries, etc.)
✅ **Debugging**: Clear job hierarchy in output

---

Ready to proceed?

