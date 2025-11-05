package buildfab

import (
    "fmt"
    "strings"
    "testing"
)

// TestHierarchicalDAG_SolvesMatrixImageBuildProblem demonstrates how the hierarchical
// DAG solves the original problem from test_matrix_skiped.yml
func TestHierarchicalDAG_SolvesMatrixImageBuildProblem(t *testing.T) {
    // Recreate the problematic scenario:
    // - Stage with 3 steps: build-action (has if condition), cleanup-action, test-action
    // - Matrix with 2 compilers × 3 builds = 6 jobs
    // - max_parallel: 1 (sequential job execution)
    // - build-action has condition: !(matrix.builds == 'DebugWithRelInfo')
    
    config := &Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "build-action", Run: "echo 'Build: ${{ matrix.builds }} / ${{ matrix.compiler }}'"},
            {Name: "cleanup-action", Run: "echo 'Cleanup'"},
            {Name: "test-action", Run: "echo 'Test'"},
        },
        Stages: map[string]Stage{
            "build-stage": {
                Steps: []Step{
                    {Action: "build-action", If: "!(matrix.builds == 'DebugWithRelInfo')"},
                    {Action: "cleanup-action"},
                    {Action: "test-action"},
                },
            },
        },
    }
    
    // Matrix step
    step := &Step{
        Stage: "build-stage",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "compiler": []interface{}{"gcc", "clang"},
                "builds":   []interface{}{"Release", "Debug", "DebugWithRelInfo"},
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
    
    // Verify we got 6 jobs (2 compilers × 3 builds)
    if len(jobs) != 6 {
        t.Fatalf("Expected 6 jobs, got %d", len(jobs))
    }
    
    // Verify each job has 3 steps
    for i, job := range jobs {
        if len(job.Steps) != 3 {
            t.Errorf("Expected job %d to have 3 steps, got %d", i, len(job.Steps))
        }
        
        // Steps should be in order: build-action, cleanup-action, test-action
        if job.Steps[0].DisplayName != "build-action" {
            t.Errorf("Job %d step 0 should be 'build-action', got '%s'", i, job.Steps[0].DisplayName)
        }
        if job.Steps[1].DisplayName != "cleanup-action" {
            t.Errorf("Job %d step 1 should be 'cleanup-action', got '%s'", i, job.Steps[1].DisplayName)
        }
        if job.Steps[2].DisplayName != "test-action" {
            t.Errorf("Job %d step 2 should be 'test-action', got '%s'", i, job.Steps[2].DisplayName)
        }
    }
    
    // Inject sliding window dependencies
    expander.InjectSlidingWindowDependencies(jobs, 1)
    
    // Verify sliding window chain
    for i := 1; i < len(jobs); i++ {
        if len(jobs[i].Dependencies) != 1 {
            t.Errorf("Job %d should have 1 dependency, got %d", i, len(jobs[i].Dependencies))
        }
        
        expectedDep := fmt.Sprintf("%d", i-1)
        if jobs[i].Dependencies[0] != expectedDep {
            t.Errorf("Job %d should depend on '%s', got '%s'", i, expectedDep, jobs[i].Dependencies[0])
        }
        
        if !jobs[i].SlidingWindowDeps[expectedDep] {
            t.Errorf("Job %d dependency on '%s' should be sliding window", i, expectedDep)
        }
    }
    
    // Build complete DAG
    dag := NewHierarchicalDAG()
    for _, job := range jobs {
        dag.AddRootJob(job)
    }
    
    t.Logf("✅ Created hierarchical DAG with %d jobs and %d total steps", 
        dag.CountTotalJobs(), dag.CountTotalSteps())
    
    // Simulate execution scenario where job 1 is skipped due to condition
    // Expected: Job 2 should still execute (sliding window dependency)
    
    // Set up: Job 0 completes, Job 1 skipped by condition
    jobs[0].Status = JobExecutionStatusOK
    jobs[1].Status = JobExecutionStatusSkippedCondition
    
    // Create executor
    opts := DefaultRunOptions()
    callback := NewMockJobCallback()
    executor := NewHierarchicalExecutor(dag, config, opts, callback)
    
    // Check if job 2 should skip
    shouldSkip := executor.shouldSkipJobDueToDependencies(jobs[2])
    
    if shouldSkip {
        t.Error("❌ FAIL: Job 2 should NOT skip (sliding window dep on condition-skipped job)")
    } else {
        t.Log("✅ PASS: Job 2 correctly allowed to execute despite job 1 being condition-skipped")
    }
}

// TestHierarchicalDAG_ShowsCorrectArchitecture demonstrates the architecture
func TestHierarchicalDAG_ShowsCorrectArchitecture(t *testing.T) {
    config := &Config{
        Project: Project{Name: "demo"},
        Actions: []Action{
            {Name: "build", Run: "echo build"},
            {Name: "test", Run: "echo test"},
            {Name: "deploy", Run: "echo deploy"},
        },
        Stages: map[string]Stage{
            "ci": {
                Steps: []Step{
                    {Action: "build"},
                    {Action: "test"},
                    {Action: "deploy"},
                },
            },
        },
    }
    
    step := &Step{
        Stage: "ci",
        Matrix: &MatrixConfig{
            Values: map[string][]interface{}{
                "env": []interface{}{"dev", "prod"},
            },
            Strategy: MatrixStrategy{
                MaxParallel: 2, // Both can run in parallel
            },
        },
    }
    
    expander := NewJobExpander(config, map[string]string{}, map[string]string{})
    jobs, err := expander.ExpandMatrixToJobs(step)
    
    if err != nil {
        t.Fatalf("Failed to expand: %v", err)
    }
    
    expander.InjectSlidingWindowDependencies(jobs, 2)
    
    dag := NewHierarchicalDAG()
    for _, job := range jobs {
        dag.AddRootJob(job)
    }
    
    // Print the hierarchical structure
    t.Log("=" + strings.Repeat("=", 70))
    t.Log("HIERARCHICAL DAG STRUCTURE")
    t.Log("=" + strings.Repeat("=", 70))
    
    for _, job := range dag.RootJobs {
        printJobTree(t, job, 0)
    }
    
    t.Log("=" + strings.Repeat("=", 70))
    t.Logf("Total: %d jobs, %d steps", dag.CountTotalJobs(), dag.CountTotalSteps())
    t.Log("=" + strings.Repeat("=", 70))
}

// printJobTree prints a job and its children in tree format
func printJobTree(t *testing.T, job *JobNode, indent int) {
    prefix := strings.Repeat("  ", indent)
    
    // Print job
    deps := ""
    if len(job.Dependencies) > 0 {
        depList := make([]string, 0)
        for _, dep := range job.Dependencies {
            if job.SlidingWindowDeps[dep] {
                depList = append(depList, fmt.Sprintf("%s[SW]", dep))
            } else {
                depList = append(depList, dep)
            }
        }
        deps = fmt.Sprintf(" [depends on: %s]", strings.Join(depList, ", "))
    }
    
    t.Logf("%s📦 Job %s (%s)%s", prefix, job.ID, job.DisplayName, deps)
    
    // Print steps
    for _, step := range job.Steps {
        ifCond := ""
        if step.If != "" {
            ifCond = fmt.Sprintf(" [if: %s]", step.If)
        }
        t.Logf("%s  ├─ Step %d: %s (ID: %s)%s", prefix, step.Index, step.DisplayName, step.ID, ifCond)
    }
    
    // Print child jobs
    if len(job.ChildJobs) > 0 {
        t.Logf("%s  └─ Child Jobs:", prefix)
        for _, child := range job.ChildJobs {
            printJobTree(t, child, indent+2)
        }
    }
}

