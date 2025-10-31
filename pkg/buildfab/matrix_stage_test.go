package buildfab

import (
	"testing"
)

// TestMatrixStageExpansion_Basic tests basic matrix stage expansion
func TestMatrixStageExpansion_Basic(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "echo step1"},
			{Name: "step2", Run: "echo step2"},
			{Name: "step3", Run: "echo step3"},
		},
		Stages: map[string]Stage{
			"build": {
				Steps: []Step{
					{Action: "step1"},
					{Action: "step2", Require: []string{"step1"}},
					{Action: "step3", Require: []string{"step2"}},
				},
			},
			"compiler-build": {
				Steps: []Step{
					{
						Stage: "build",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"compiler": {"gcc", "clang"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 1,
							},
						},
					},
				},
			},
		},
	}

	opts := DefaultRunOptions()
	opts.Debug = true
	runner := NewRunner(config, opts)

	// Test expansion
	matrixSteps, interpolatedActions, poolConfigs, matrixJobs, err := runner.expandMatrixStageStepsWithPools(config.Stages["compiler-build"].Steps)
	if err != nil {
		t.Fatalf("Failed to expand matrix stage steps: %v", err)
	}

	// Should have 2 matrix jobs (gcc, clang), each with 3 steps
	if len(matrixJobs) != 2 {
		t.Errorf("Expected 2 matrix jobs, got %d", len(matrixJobs))
	}

	// Should have 6 total steps (2 jobs × 3 steps per job)
	if len(matrixSteps) != 6 {
		t.Errorf("Expected 6 expanded steps, got %d", len(matrixSteps))
	}

	// Check matrix job structure
	for i, job := range matrixJobs {
		if len(job.Steps) != 3 {
			t.Errorf("Matrix job %d: expected 3 steps, got %d", i, len(job.Steps))
		}
		
		// Check that matrix variables are set
		expectedCompiler := "gcc"
		if i == 1 {
			expectedCompiler = "clang"
		}
		
		matrixVar := job.MatrixVars["matrix.compiler"]
		if matrixVar != expectedCompiler {
			t.Errorf("Matrix job %d: expected matrix.compiler=%s, got %s", i, expectedCompiler, matrixVar)
		}
		
		// Check first and last steps
		if len(job.FirstSteps) == 0 {
			t.Errorf("Matrix job %d: expected at least one first step", i)
		}
		if len(job.LastSteps) == 0 {
			t.Errorf("Matrix job %d: expected at least one last step", i)
		}
	}

	// Check pool configs
	if len(poolConfigs) != 1 {
		t.Errorf("Expected 1 pool config, got %d", len(poolConfigs))
	}

	// Check interpolated actions
	if len(interpolatedActions) != 0 {
		// For stage expansion, interpolated actions may be empty
		t.Logf("Note: %d interpolated actions (may be 0 for stage expansion)", len(interpolatedActions))
	}
}

// TestMatrixStageExpansion_MaxParallel2 tests matrix stage expansion with max_parallel=2
func TestMatrixStageExpansion_MaxParallel2(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-project",
		},
		Actions: []Action{
			{Name: "step1", Run: "echo step1"},
		},
		Stages: map[string]Stage{
			"build": {
				Steps: []Step{
					{Action: "step1"},
				},
			},
			"multi-build": {
				Steps: []Step{
					{
						Stage: "build",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"compiler": {"gcc", "clang", "msvc", "icc"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 2,
							},
						},
					},
				},
			},
		},
	}

	opts := DefaultRunOptions()
	opts.Debug = true
	runner := NewRunner(config, opts)

	// Test expansion
	matrixSteps, _, poolConfigs, matrixJobs, err := runner.expandMatrixStageStepsWithPools(config.Stages["multi-build"].Steps)
	if err != nil {
		t.Fatalf("Failed to expand matrix stage steps: %v", err)
	}

	// Should have 4 matrix jobs
	if len(matrixJobs) != 4 {
		t.Errorf("Expected 4 matrix jobs, got %d", len(matrixJobs))
	}

	// Should have 4 total steps (4 jobs × 1 step per job)
	if len(matrixSteps) != 4 {
		t.Errorf("Expected 4 expanded steps, got %d", len(matrixSteps))
	}

	// Inject sliding window dependencies
	maxParallel := 2
	stepsWithDeps := runner.injectSlidingWindowDependencies(matrixSteps, matrixJobs, maxParallel)

	// Job 0 (gcc) and Job 1 (clang): no dependencies (first maxParallel jobs)
	// Job 2 (msvc): should depend on Job 0's last step
	// Job 3 (icc): should depend on Job 1's last step

	// Find step for job 2 (msvc) - should be named "build.2.step1"
	msvcStepName := matrixJobs[2].FirstSteps[0] // Third job (index 2, msvc)
	
	// Find the step in stepsWithDeps
	msvcStep := (*Step)(nil)
	for i := range stepsWithDeps {
		if stepsWithDeps[i].GetStepName() == msvcStepName {
			msvcStep = &stepsWithDeps[i]
			break
		}
	}

	if msvcStep == nil {
		t.Fatalf("Could not find msvc step: %s", msvcStepName)
	}

	// Check that msvc step has a dependency on gcc step (job 0's last step)
	gccLastStep := matrixJobs[0].LastSteps[0] // First job's last step (gcc)
	hasGccDep := false
	
	for _, dep := range msvcStep.Require {
		if dep == gccLastStep {
			hasGccDep = true
			break
		}
	}

	if !hasGccDep {
		t.Errorf("Expected msvc step (%s) to depend on gcc step (%s), but dependencies are: %v", 
			msvcStepName, gccLastStep, msvcStep.Require)
	}
	
	// Also check that icc step (job 3) depends on clang step (job 1)
	iccStepName := matrixJobs[3].FirstSteps[0]
	clangLastStep := matrixJobs[1].LastSteps[0]
	
	iccStep := (*Step)(nil)
	for i := range stepsWithDeps {
		if stepsWithDeps[i].GetStepName() == iccStepName {
			iccStep = &stepsWithDeps[i]
			break
		}
	}
	
	if iccStep == nil {
		t.Fatalf("Could not find icc step: %s", iccStepName)
	}
	
	hasClangDep := false
	for _, dep := range iccStep.Require {
		if dep == clangLastStep {
			hasClangDep = true
			break
		}
	}
	
	if !hasClangDep {
		t.Errorf("Expected icc step (%s) to depend on clang step (%s), but dependencies are: %v",
			iccStepName, clangLastStep, iccStep.Require)
	}

	// Check pool configs
	if len(poolConfigs) != 1 {
		t.Errorf("Expected 1 pool config, got %d", len(poolConfigs))
	}
	
	poolID := "matrix-build"
	if maxParallel, exists := poolConfigs[poolID]; !exists {
		t.Errorf("Expected pool config for %s", poolID)
	} else if maxParallel != 2 {
		t.Errorf("Expected pool max_parallel=2, got %d", maxParallel)
	}
}

// TestFindFirstSteps tests the findFirstSteps function
func TestFindFirstSteps(t *testing.T) {
	config := &Config{
		Project: Project{Name: "test"},
	}
	opts := DefaultRunOptions()
	runner := NewRunner(config, opts)

	tests := []struct {
		name      string
		steps     []Step
		wantFirst []string
	}{
		{
			name: "linear_dependencies",
			steps: []Step{
				{Action: "step1"},
				{Action: "step2", Require: []string{"step1"}},
				{Action: "step3", Require: []string{"step2"}},
			},
			wantFirst: []string{"step1"}, // Only step1 has no dependencies
		},
		{
			name: "parallel_steps",
			steps: []Step{
				{Action: "step1"},
				{Action: "step2"},
				{Action: "step3", Require: []string{"step1", "step2"}},
			},
			wantFirst: []string{"step1", "step2"}, // Both step1 and step2 have no dependencies
		},
		{
			name: "all_dependent",
			steps: []Step{
				{Action: "step1", Require: []string{"external"}},
				{Action: "step2", Require: []string{"external"}},
			},
			// Should return all steps if dependencies are external
			wantFirst: []string{"step1", "step2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runner.findFirstSteps(tt.steps)
			if len(got) != len(tt.wantFirst) {
				t.Errorf("findFirstSteps() returned %d steps, want %d: got=%v, want=%v", 
					len(got), len(tt.wantFirst), got, tt.wantFirst)
			}
			
			// Check that all expected first steps are present
			gotMap := make(map[string]bool)
			for _, step := range got {
				gotMap[step] = true
			}
			for _, want := range tt.wantFirst {
				if !gotMap[want] {
					t.Errorf("findFirstSteps() missing expected step: %s", want)
				}
			}
		})
	}
}

// TestFindLastSteps tests the findLastSteps function
func TestFindLastSteps(t *testing.T) {
	config := &Config{
		Project: Project{Name: "test"},
	}
	opts := DefaultRunOptions()
	runner := NewRunner(config, opts)

	tests := []struct {
		name     string
		steps    []Step
		wantLast []string
	}{
		{
			name: "linear_dependencies",
			steps: []Step{
				{Action: "step1"},
				{Action: "step2", Require: []string{"step1"}},
				{Action: "step3", Require: []string{"step2"}},
			},
			wantLast: []string{"step3"}, // Only step3 has no dependents
		},
		{
			name: "converging_dependencies",
			steps: []Step{
				{Action: "step1"},
				{Action: "step2"},
				{Action: "step3", Require: []string{"step1", "step2"}},
			},
			wantLast: []string{"step3"}, // Only step3 has no dependents
		},
		{
			name: "parallel_last_steps",
			steps: []Step{
				{Action: "step1"},
				{Action: "step2", Require: []string{"step1"}},
				{Action: "step3", Require: []string{"step1"}},
			},
			wantLast: []string{"step2", "step3"}, // Both step2 and step3 have no dependents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runner.findLastSteps(tt.steps)
			if len(got) != len(tt.wantLast) {
				t.Errorf("findLastSteps() returned %d steps, want %d: got=%v, want=%v", 
					len(got), len(tt.wantLast), got, tt.wantLast)
			}
			
			// Check that all expected last steps are present
			gotMap := make(map[string]bool)
			for _, step := range got {
				gotMap[step] = true
			}
			for _, want := range tt.wantLast {
				if !gotMap[want] {
					t.Errorf("findLastSteps() missing expected step: %s", want)
				}
			}
		})
	}
}

// TestMatrixStageExecution_Sequential tests sequential execution with max_parallel=1
func TestMatrixStageExecution_Sequential(t *testing.T) {
	// This is an integration-style test that would require more setup
	// For now, we test the expansion logic
	t.Log("Matrix stage execution test would require full runner setup")
	t.Log("This test verifies expansion logic; execution testing should be done in integration tests")
}

// TestMatrixStageExecution_Parallel tests parallel execution with max_parallel=2
func TestMatrixStageExecution_Parallel(t *testing.T) {
	// This is an integration-style test that would require more setup
	t.Log("Matrix stage execution test would require full runner setup")
	t.Log("This test verifies expansion logic; execution testing should be done in integration tests")
}

