package buildfab

import (
    "testing"
)

func TestJobNode_Creation(t *testing.T) {
    matrixVars := map[string]string{
        "matrix.compiler": "gcc",
        "matrix.build":    "Release",
    }
    
    job := NewJobNode("0", "gcc+Release", matrixVars)
    
    if job.ID != "0" {
        t.Errorf("Expected job ID '0', got '%s'", job.ID)
    }
    
    if job.DisplayName != "gcc+Release" {
        t.Errorf("Expected display name 'gcc+Release', got '%s'", job.DisplayName)
    }
    
    if len(job.Steps) != 0 {
        t.Errorf("Expected 0 steps, got %d", len(job.Steps))
    }
    
    if job.Status != JobExecutionStatusPending {
        t.Errorf("Expected status Pending, got %s", job.Status)
    }
}

func TestJobNode_AddStep(t *testing.T) {
    job := NewJobNode("0", "test-job", nil)
    
    action := Action{Name: "build", Run: "echo hello"}
    step := ExecutableStep{
        DisplayName: "build",
        Action:      &action,
        Variables:   map[string]string{"key": "value"},
    }
    
    job.AddStep(step)
    
    if len(job.Steps) != 1 {
        t.Errorf("Expected 1 step, got %d", len(job.Steps))
    }
    
    if job.Steps[0].Index != 0 {
        t.Errorf("Expected step index 0, got %d", job.Steps[0].Index)
    }
    
    if job.Steps[0].ID != "0.0" {
        t.Errorf("Expected step ID '0.0', got '%s'", job.Steps[0].ID)
    }
}

func TestJobNode_AddDependency(t *testing.T) {
    job := NewJobNode("2", "test-job", nil)
    
    // Add sliding window dependency
    job.AddDependency("1", true)
    
    if len(job.Dependencies) != 1 {
        t.Errorf("Expected 1 dependency, got %d", len(job.Dependencies))
    }
    
    if job.Dependencies[0] != "1" {
        t.Errorf("Expected dependency '1', got '%s'", job.Dependencies[0])
    }
    
    if !job.SlidingWindowDeps["1"] {
        t.Errorf("Expected sliding window dependency for '1'")
    }
    
    // Add explicit dependency
    job.AddDependency("0", false)
    
    if len(job.Dependencies) != 2 {
        t.Errorf("Expected 2 dependencies, got %d", len(job.Dependencies))
    }
    
    if job.SlidingWindowDeps["0"] {
        t.Errorf("Expected '0' to NOT be sliding window dependency")
    }
}

func TestHierarchicalDAG_Creation(t *testing.T) {
    dag := NewHierarchicalDAG()
    
    if len(dag.RootJobs) != 0 {
        t.Errorf("Expected 0 root jobs, got %d", len(dag.RootJobs))
    }
    
    if len(dag.AllJobs) != 0 {
        t.Errorf("Expected 0 jobs in map, got %d", len(dag.AllJobs))
    }
    
    if len(dag.AllSteps) != 0 {
        t.Errorf("Expected 0 steps in map, got %d", len(dag.AllSteps))
    }
}

func TestHierarchicalDAG_AddRootJob(t *testing.T) {
    dag := NewHierarchicalDAG()
    
    job := NewJobNode("0", "test-job", nil)
    action := Action{Name: "build", Run: "echo hello"}
    step := ExecutableStep{DisplayName: "build", Action: &action}
    job.AddStep(step)
    
    dag.AddRootJob(job)
    
    if len(dag.RootJobs) != 1 {
        t.Errorf("Expected 1 root job, got %d", len(dag.RootJobs))
    }
    
    if len(dag.AllJobs) != 1 {
        t.Errorf("Expected 1 job in map, got %d", len(dag.AllJobs))
    }
    
    if len(dag.AllSteps) != 1 {
        t.Errorf("Expected 1 step in map, got %d", len(dag.AllSteps))
    }
    
    // Verify lookup works
    retrievedJob := dag.GetJob("0")
    if retrievedJob == nil {
        t.Fatal("Expected to retrieve job '0'")
    }
    
    if retrievedJob.ID != "0" {
        t.Errorf("Expected job ID '0', got '%s'", retrievedJob.ID)
    }
    
    retrievedStep := dag.GetStep("0.0")
    if retrievedStep == nil {
        t.Fatal("Expected to retrieve step '0.0'")
    }
    
    if retrievedStep.ID != "0.0" {
        t.Errorf("Expected step ID '0.0', got '%s'", retrievedStep.ID)
    }
}

func TestJobNode_NestedJobs(t *testing.T) {
    parentJob := NewJobNode("0", "parent-job", nil)
    childJob := NewJobNode("0.0", "child-job", nil)
    
    action := Action{Name: "build", Run: "echo hello"}
    step := ExecutableStep{DisplayName: "build", Action: &action}
    childJob.AddStep(step)
    
    parentJob.AddChildJob(childJob)
    
    if len(parentJob.ChildJobs) != 1 {
        t.Errorf("Expected 1 child job, got %d", len(parentJob.ChildJobs))
    }
    
    // Test hierarchical step lookup
    retrievedStep := parentJob.GetStepByID("0.0.0")
    if retrievedStep == nil {
        t.Fatal("Expected to retrieve step '0.0.0'")
    }
    
    if retrievedStep.ID != "0.0.0" {
        t.Errorf("Expected step ID '0.0.0', got '%s'", retrievedStep.ID)
    }
}

func TestJobExecutionStatus_String(t *testing.T) {
    tests := []struct {
        status JobExecutionStatus
        want   string
    }{
        {JobExecutionStatusPending, "pending"},
        {JobExecutionStatusRunning, "running"},
        {JobExecutionStatusOK, "ok"},
        {JobExecutionStatusWarn, "warn"},
        {JobExecutionStatusError, "error"},
        {JobExecutionStatusSkipped, "skipped"},
        {JobExecutionStatusSkippedCondition, "skipped_condition"},
        {JobExecutionStatusPartial, "partial"},
    }
    
    for _, tt := range tests {
        t.Run(tt.want, func(t *testing.T) {
            if got := tt.status.String(); got != tt.want {
                t.Errorf("JobExecutionStatus.String() = %v, want %v", got, tt.want)
            }
        })
    }
}

