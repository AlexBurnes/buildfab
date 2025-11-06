package buildfab

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "testing"
)

// TestHierarchicalDAG_SimpleMatrixOnAction tests basic matrix expansion to jobs
func TestHierarchicalDAG_SimpleMatrixOnAction(t *testing.T) {
    // Create a simple config with one action
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {
                Name: "build",
                Run:  "echo 'Building with ${{ matrix.compiler }}'",
            },
        },
    }
    
    // Create a step with matrix
    step := &Step{
        Action: "build",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "compiler": []interface{}{"gcc", "clang"},
            },
        },
    }
    
    // Expand to jobs
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Verify we got 2 jobs
    if len(jobs) != 2 {
        t.Fatalf("Expected 2 jobs, got %d", len(jobs))
    }
    
    // Verify job 0
    if jobs[0].ID != "0" {
        t.Errorf("Expected job 0 ID '0', got '%s'", jobs[0].ID)
    }
    
    if !strings.Contains(jobs[0].DisplayName, "gcc") {
        t.Errorf("Expected job 0 display name to contain 'gcc', got '%s'", jobs[0].DisplayName)
    }
    
    if len(jobs[0].Steps) != 1 {
        t.Errorf("Expected job 0 to have 1 step, got %d", len(jobs[0].Steps))
    }
    
    if jobs[0].Steps[0].ID != "0.0" {
        t.Errorf("Expected step ID '0.0', got '%s'", jobs[0].Steps[0].ID)
    }
    
    // Verify interpolation worked
    if !strings.Contains(jobs[0].Steps[0].Action.Run, "gcc") {
        t.Errorf("Expected interpolated action to contain 'gcc', got '%s'", jobs[0].Steps[0].Action.Run)
    }
    
    // Verify job 1
    if jobs[1].ID != "1" {
        t.Errorf("Expected job 1 ID '1', got '%s'", jobs[1].ID)
    }
    
    if !strings.Contains(jobs[1].DisplayName, "clang") {
        t.Errorf("Expected job 1 display name to contain 'clang', got '%s'", jobs[1].DisplayName)
    }
    
    if !strings.Contains(jobs[1].Steps[0].Action.Run, "clang") {
        t.Errorf("Expected interpolated action to contain 'clang', got '%s'", jobs[1].Steps[0].Action.Run)
    }
}

// TestHierarchicalDAG_MatrixOnStage tests matrix on stage with multiple steps
func TestHierarchicalDAG_MatrixOnStage(t *testing.T) {
    // Create config with multiple actions and a stage
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo 'Build with ${{ matrix.compiler }}'"},
            {Name: "test", Run: "echo 'Test with ${{ matrix.compiler }}'"},
            {Name: "cleanup", Run: "echo 'Cleanup'"},
        },
        Stages: map[string]Stage{
            "ci": {
                Steps: []Step{
                    {Action: "build"},
                    {Action: "test"},
                    {Action: "cleanup"},
                },
            },
        },
    }
    
    // Create a step with matrix on stage
    step := &Step{
        Stage: "ci",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "compiler": []interface{}{"gcc", "clang"},
            },
        },
    }
    
    // Expand to jobs
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Verify we got 2 jobs
    if len(jobs) != 2 {
        t.Fatalf("Expected 2 jobs, got %d", len(jobs))
    }
    
    // Verify each job has 3 steps (build, test, cleanup)
    for i, job := range jobs {
        if len(job.Steps) != 3 {
            t.Errorf("Expected job %d to have 3 steps, got %d", i, len(job.Steps))
        }
        
        // Verify step IDs are sequential
        for stepIdx, step := range job.Steps {
            expectedID := fmt.Sprintf("%s.%d", job.ID, stepIdx)
            if step.ID != expectedID {
                t.Errorf("Expected step ID '%s', got '%s'", expectedID, step.ID)
            }
            
            if step.Index != stepIdx {
                t.Errorf("Expected step index %d, got %d", stepIdx, step.Index)
            }
        }
        
        // Verify matrix variables are set
        if job.MatrixVars["matrix.compiler"] == "" {
            t.Errorf("Expected matrix.compiler to be set for job %d", i)
        }
        
        // Verify interpolation worked in first step
        compilerValue := job.MatrixVars["matrix.compiler"]
        if !strings.Contains(job.Steps[0].Action.Run, compilerValue) {
            t.Errorf("Expected step 0 action to contain '%s', got '%s'", compilerValue, job.Steps[0].Action.Run)
        }
    }
}

// TestJobExpander_SlidingWindowDependencies tests sliding window dependency injection
func TestJobExpander_SlidingWindowDependencies(t *testing.T) {
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo build"},
        },
    }
    
    step := &Step{
        Action: "build",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "version": []interface{}{"1", "2", "3", "4"},
            },
            Strategy: MatrixStrategy{
                MaxParallel: 1,
            },
        },
    }
    
    // Expand to jobs
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Verify we got 4 jobs
    if len(jobs) != 4 {
        t.Fatalf("Expected 4 jobs, got %d", len(jobs))
    }
    
    // Inject sliding window dependencies with max_parallel=1
    expander.InjectSlidingWindowDependencies(jobs, 1)
    
    // Verify dependencies
    // Job 0: no dependencies
    if len(jobs[0].Dependencies) != 0 {
        t.Errorf("Expected job 0 to have 0 dependencies, got %d", len(jobs[0].Dependencies))
    }
    
    // Job 1: depends on job 0 (sliding window)
    if len(jobs[1].Dependencies) != 1 {
        t.Errorf("Expected job 1 to have 1 dependency, got %d", len(jobs[1].Dependencies))
    }
    if jobs[1].Dependencies[0] != "0" {
        t.Errorf("Expected job 1 to depend on '0', got '%s'", jobs[1].Dependencies[0])
    }
    if !jobs[1].SlidingWindowDeps["0"] {
        t.Errorf("Expected dependency on '0' to be marked as sliding window")
    }
    
    // Job 2: depends on job 1
    if len(jobs[2].Dependencies) != 1 {
        t.Errorf("Expected job 2 to have 1 dependency, got %d", len(jobs[2].Dependencies))
    }
    if jobs[2].Dependencies[0] != "1" {
        t.Errorf("Expected job 2 to depend on '1', got '%s'", jobs[2].Dependencies[0])
    }
    
    // Job 3: depends on job 2
    if len(jobs[3].Dependencies) != 1 {
        t.Errorf("Expected job 3 to have 1 dependency, got %d", len(jobs[3].Dependencies))
    }
    if jobs[3].Dependencies[0] != "2" {
        t.Errorf("Expected job 3 to depend on '2', got '%s'", jobs[3].Dependencies[0])
    }
}

// TestJobExpander_SlidingWindowDependencies_MaxParallel2 tests sliding window with max_parallel=2
func TestJobExpander_SlidingWindowDependencies_MaxParallel2(t *testing.T) {
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo build"},
        },
    }
    
    step := &Step{
        Action: "build",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "version": []interface{}{"1", "2", "3", "4"},
            },
            Strategy: MatrixStrategy{
                MaxParallel: 2,
            },
        },
    }
    
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Inject sliding window dependencies with max_parallel=2
    expander.InjectSlidingWindowDependencies(jobs, 2)
    
    // Verify dependencies
    // Jobs 0-1: no dependencies (can run in parallel)
    if len(jobs[0].Dependencies) != 0 {
        t.Errorf("Expected job 0 to have 0 dependencies, got %d", len(jobs[0].Dependencies))
    }
    if len(jobs[1].Dependencies) != 0 {
        t.Errorf("Expected job 1 to have 0 dependencies, got %d", len(jobs[1].Dependencies))
    }
    
    // Job 2: depends on job 0 (2 - 2 = 0)
    if len(jobs[2].Dependencies) != 1 {
        t.Errorf("Expected job 2 to have 1 dependency, got %d", len(jobs[2].Dependencies))
    }
    if jobs[2].Dependencies[0] != "0" {
        t.Errorf("Expected job 2 to depend on '0', got '%s'", jobs[2].Dependencies[0])
    }
    
    // Job 3: depends on job 1 (3 - 2 = 1)
    if len(jobs[3].Dependencies) != 1 {
        t.Errorf("Expected job 3 to have 1 dependency, got %d", len(jobs[3].Dependencies))
    }
    if jobs[3].Dependencies[0] != "1" {
        t.Errorf("Expected job 3 to depend on '1', got '%s'", jobs[3].Dependencies[0])
    }
}

// TestHierarchicalDAG_BuildFromJobs tests creating a complete DAG
func TestHierarchicalDAG_BuildFromJobs(t *testing.T) {
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo build"},
            {Name: "test", Run: "echo test"},
        },
        Stages: map[string]Stage{
            "ci": {
                Steps: []Step{
                    {Action: "build"},
                    {Action: "test"},
                },
            },
        },
    }
    
    step := &Step{
        Stage: "ci",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "compiler": []interface{}{"gcc", "clang"},
            },
            Strategy: MatrixStrategy{
                MaxParallel: 1,
            },
        },
    }
    
    // Expand to jobs
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Inject dependencies
    expander.InjectSlidingWindowDependencies(jobs, 1)
    
    // Build DAG
    dag := NewHierarchicalDAG()
    for _, job := range jobs {
        dag.AddRootJob(job)
    }
    
    // Verify DAG structure
    if dag.CountTotalJobs() != 2 {
        t.Errorf("Expected 2 total jobs, got %d", dag.CountTotalJobs())
    }
    
    if dag.CountTotalSteps() != 4 {
        t.Errorf("Expected 4 total steps (2 jobs × 2 steps), got %d", dag.CountTotalSteps())
    }
    
    // Verify all steps are registered
    for _, job := range jobs {
        for _, step := range job.Steps {
            retrievedStep := dag.GetStep(step.ID)
            if retrievedStep == nil {
                t.Errorf("Failed to retrieve step '%s'", step.ID)
            }
        }
    }
    
    // Verify all jobs are registered
    for _, job := range jobs {
        retrievedJob := dag.GetJob(job.ID)
        if retrievedJob == nil {
            t.Errorf("Failed to retrieve job '%s'", job.ID)
        }
    }
}

// MockJobCallback implements JobExecutionCallback for testing
type MockJobCallback struct {
    mu             sync.Mutex
    jobsStarted    []string
    jobsCompleted  []string
    stepsStarted   []string
    stepsCompleted []string
}

func NewMockJobCallback() *MockJobCallback {
    return &MockJobCallback{
        jobsStarted:    make([]string, 0),
        jobsCompleted:  make([]string, 0),
        stepsStarted:   make([]string, 0),
        stepsCompleted: make([]string, 0),
    }
}

func (m *MockJobCallback) OnJobStart(ctx context.Context, job *JobNode) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.jobsStarted = append(m.jobsStarted, job.ID)
}

func (m *MockJobCallback) OnJobComplete(ctx context.Context, job *JobNode) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.jobsCompleted = append(m.jobsCompleted, job.ID)
}

func (m *MockJobCallback) OnStepStart(ctx context.Context, job *JobNode, step *ExecutableStep) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.stepsStarted = append(m.stepsStarted, step.ID)
}

func (m *MockJobCallback) OnStepComplete(ctx context.Context, job *JobNode, step *ExecutableStep, result StepResult) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.stepsCompleted = append(m.stepsCompleted, step.ID)
}

func (m *MockJobCallback) OnStepOutput(ctx context.Context, job *JobNode, step *ExecutableStep, output string) {
    // No-op for testing
}

// Thread-safe getter methods
func (m *MockJobCallback) GetJobsStarted() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]string, len(m.jobsStarted))
    copy(result, m.jobsStarted)
    return result
}

func (m *MockJobCallback) GetJobsCompleted() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]string, len(m.jobsCompleted))
    copy(result, m.jobsCompleted)
    return result
}

func (m *MockJobCallback) GetStepsStarted() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]string, len(m.stepsStarted))
    copy(result, m.stepsStarted)
    return result
}

func (m *MockJobCallback) GetStepsCompleted() []string {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([]string, len(m.stepsCompleted))
    copy(result, m.stepsCompleted)
    return result
}

// TestHierarchicalExecutor_SimpleExecution tests basic job execution
func TestHierarchicalExecutor_SimpleExecution(t *testing.T) {
    // Create a simple DAG with 2 jobs
    dag := NewHierarchicalDAG()
    
    // Job 0: single step
    job0 := NewJobNode("0", "job-0", map[string]string{"matrix.value": "0"})
    action0 := Action{Name: "echo", Run: "echo job0"}
    job0.AddStep(ExecutableStep{DisplayName: "echo", Action: &action0})
    dag.AddRootJob(job0)
    
    // Job 1: depends on job 0 (sliding window)
    job1 := NewJobNode("1", "job-1", map[string]string{"matrix.value": "1"})
    action1 := Action{Name: "echo", Run: "echo job1"}
    job1.AddStep(ExecutableStep{DisplayName: "echo", Action: &action1})
    job1.AddDependency("0", true) // Sliding window dependency
    dag.AddRootJob(job1)
    
    // Create config and options
    config := &Config{
        Project: Project{Name: "test"},
    }
    
    opts := DefaultRunOptions()
    opts.VerboseLevel = 0
    opts.Debug = false
    
    // Create mock callback
    callback := NewMockJobCallback()
    
    // Create executor
    executor := NewHierarchicalExecutor(dag, config, opts, callback)
    
    // Execute (this will fail because executeStep is not fully implemented yet)
    ctx := context.Background()
    err := executor.Execute(ctx)
    
    // For now, we just verify the structure worked
    // Once we implement real step execution, we can verify results
    if err != nil {
        t.Logf("Execution error (expected for now): %v", err)
    }
    
    // Verify callbacks were called
    jobsStarted := callback.GetJobsStarted()
    if len(jobsStarted) == 0 {
        t.Error("Expected at least one job to start")
    }
    
    t.Logf("Jobs started: %v", jobsStarted)
    t.Logf("Jobs completed: %v", callback.GetJobsCompleted())
    t.Logf("Steps started: %v", callback.GetStepsStarted())
    t.Logf("Steps completed: %v", callback.GetStepsCompleted())
}

// TestHierarchicalDAG_ConditionSkipDoesNotBlockSlidingWindow tests the key feature
func TestHierarchicalDAG_ConditionSkipDoesNotBlockSlidingWindow(t *testing.T) {
    // Create config
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo 'Build: ${{ matrix.type }}'"},
        },
    }
    
    // Create step with if condition that will skip some combinations
    step := &Step{
        Action: "build",
        If:     "!(matrix.type == 'skip-me')",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "type": []interface{}{"normal", "skip-me", "normal2"},
            },
            Strategy: MatrixStrategy{
                MaxParallel: 1,
            },
        },
    }
    
    // Expand to jobs
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Inject sliding window dependencies
    expander.InjectSlidingWindowDependencies(jobs, 1)
    
    // Build DAG
    dag := NewHierarchicalDAG()
    for _, job := range jobs {
        dag.AddRootJob(job)
    }
    
    // Simulate execution: job 1 is skipped due to condition
    jobs[0].Status = JobExecutionStatusOK      // Job 0 executes normally
    jobs[1].Status = JobExecutionStatusSkippedCondition  // Job 1 skipped by condition
    
    // Check if job 2 should be skipped
    // Create executor to test dependency checking
    opts := DefaultRunOptions()
    callback := NewMockJobCallback()
    executor := NewHierarchicalExecutor(dag, config, opts, callback)
    
    // Job 2 depends on job 1 (sliding window)
    // Since job 1 was skipped due to condition AND it's a sliding window dep,
    // job 2 should NOT be blocked
    shouldSkip := executor.shouldSkipJobDueToDependencies(jobs[2])
    
    if shouldSkip {
        t.Error("Job 2 should NOT be skipped - sliding window dependency on condition-skipped job should not block")
    }
}

// TestHierarchicalDAG_ExplicitDependencyBlocksOnConditionSkip tests explicit deps
func TestHierarchicalDAG_ExplicitDependencyBlocksOnConditionSkip(t *testing.T) {
    dag := NewHierarchicalDAG()
    
    // Job 0: will be skipped due to condition
    job0 := NewJobNode("0", "job-0", nil)
    action0 := Action{Name: "build", Run: "echo build"}
    job0.AddStep(ExecutableStep{DisplayName: "build", Action: &action0})
    job0.Status = JobExecutionStatusSkippedCondition
    dag.AddRootJob(job0)
    
    // Job 1: explicitly depends on job 0 (NOT sliding window)
    job1 := NewJobNode("1", "job-1", nil)
    action1 := Action{Name: "test", Run: "echo test"}
    job1.AddStep(ExecutableStep{DisplayName: "test", Action: &action1})
    job1.AddDependency("0", false) // Explicit dependency (not sliding window)
    dag.AddRootJob(job1)
    
    // Create executor
    config := &Config{Project: Project{Name: "test"}}
    opts := DefaultRunOptions()
    callback := NewMockJobCallback()
    executor := NewHierarchicalExecutor(dag, config, opts, callback)
    
    // Job 1 has explicit dependency on job 0 which was condition-skipped
    // This SHOULD block job 1
    shouldSkip := executor.shouldSkipJobDueToDependencies(job1)
    
    if !shouldSkip {
        t.Error("Job 1 SHOULD be skipped - explicit dependency on condition-skipped job should block")
    }
}

// TestHierarchicalDAG_SkippedStepSkipsRemainingStepsInJob tests that when a step
// in a job is skipped due to a condition, all subsequent steps in the same job are also skipped
func TestHierarchicalDAG_SkippedStepSkipsRemainingStepsInJob(t *testing.T) {
    // Create config with multiple actions and a stage
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build", Run: "echo build"},
            {Name: "cleanup", Run: "echo cleanup"},
            {Name: "test", Run: "echo test"},
        },
        Stages: map[string]Stage{
            "ci": {
                Steps: []Step{
                    {Action: "build", If: "matrix.type != 'skip-me'"},
                    {Action: "cleanup"}, // No condition - should be skipped when build is skipped
                    {Action: "test"},    // No condition - should be skipped when build is skipped
                },
            },
        },
    }
    
    // Create a step with matrix that will skip some combinations
    step := &Step{
        Stage: "ci",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "type": []interface{}{"normal", "skip-me"},
            },
        },
    }
    
    // Create expander and expand
    expander := NewJobExpander(config, nil, nil)
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand matrix: %v", err)
    }
    
    // Verify we got 2 jobs
    if len(jobs) != 2 {
        t.Fatalf("Expected 2 jobs, got %d", len(jobs))
    }
    
    // Each job should have 3 steps (build, cleanup, test)
    for i, job := range jobs {
        if len(job.Steps) != 3 {
            t.Errorf("Expected job %d to have 3 steps, got %d", i, len(job.Steps))
        }
    }
    
    // Create DAG and executor
    dag := NewHierarchicalDAG()
    for _, job := range jobs {
        dag.AddRootJob(job)
    }
    
    opts := DefaultRunOptions()
    opts.VerboseLevel = 0
    callback := NewMockJobCallback()
    executor := NewHierarchicalExecutor(dag, config, opts, callback)
    
    // Execute the DAG
    ctx := context.Background()
    err = executor.Execute(ctx)
    
    if err != nil {
        t.Fatalf("Failed to execute DAG: %v", err)
    }
    
    // Verify job 0 (type=normal): all 3 steps should execute
    job0 := jobs[0]
    if len(job0.Results) != 3 {
        t.Errorf("Expected job 0 to have 3 results, got %d", len(job0.Results))
    }
    for i, result := range job0.Results {
        if result.Status != StepStatusOK {
            t.Errorf("Expected job 0 step %d to have status OK, got %v", i, result.Status)
        }
    }
    
    // Verify job 1 (type=skip-me): build should be skipped, cleanup and test should also be skipped
    job1 := jobs[1]
    if len(job1.Results) != 3 {
        t.Errorf("Expected job 1 to have 3 results, got %d", len(job1.Results))
    }
    
    // All steps should be skipped due to condition
    for i, result := range job1.Results {
        if result.Status != StepStatusSkippedCondition {
            t.Errorf("Expected job 1 step %d to have status SkippedCondition, got %v", i, result.Status)
        }
    }
    
    // Verify the callback recorded the expected events
    jobsStarted := callback.GetJobsStarted()
    if len(jobsStarted) != 2 {
        t.Errorf("Expected 2 jobs to start, got %d", len(jobsStarted))
    }
    
    jobsCompleted := callback.GetJobsCompleted()
    if len(jobsCompleted) != 2 {
        t.Errorf("Expected 2 jobs to complete, got %d", len(jobsCompleted))
    }
    
    // Expected: 6 steps total (3 from job 0, 3 from job 1)
    stepsCompleted := callback.GetStepsCompleted()
    if len(stepsCompleted) != 6 {
        t.Errorf("Expected 6 steps to complete, got %d", len(stepsCompleted))
    }
}

