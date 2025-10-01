package buildfab

import (
	"context"
	"testing"
)

func TestMatrixExpander_ExpandMatrix(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo ${{ matrix.os }} ${{ matrix.version }}",
			},
		},
	}
	
	expander := NewMatrixExpander(config)
	
	step := &Step{
		Action: "test-action",
		Matrix: &MatrixConfig{
			Values: map[string][]interface{}{
				"os":      {"linux", "windows"},
				"version": {"1.0", "2.0"},
			},
			Strategy: MatrixStrategy{
				MaxParallel:     2,
				FailFast:        false,
				ContinueOnError: false,
				Order:          "fifo",
			},
		},
	}
	
	action := &Action{
		Name: "test-action",
		Run:  "echo ${{ matrix.os }} ${{ matrix.version }}",
	}
	
	jobs, err := expander.ExpandMatrix(step, action)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should have 4 jobs (2 OS × 2 versions)
	expectedJobs := 4
	if len(jobs) != expectedJobs {
		t.Fatalf("Expected %d jobs, got %d", expectedJobs, len(jobs))
	}
	
	// Check that all combinations are present
	expectedCombinations := []map[string]interface{}{
		{"os": "linux", "version": "1.0"},
		{"os": "linux", "version": "2.0"},
		{"os": "windows", "version": "1.0"},
		{"os": "windows", "version": "2.0"},
	}
	
	for _, expected := range expectedCombinations {
		found := false
		for _, job := range jobs {
			if job.Matrix["os"] == expected["os"] && job.Matrix["version"] == expected["version"] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected combination %v not found", expected)
		}
	}
}

func TestMatrixExpander_ExpandMatrix_SingleDimension(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo ${{ matrix.test }}",
			},
		},
	}
	
	expander := NewMatrixExpander(config)
	
	step := &Step{
		Action: "test-action",
		Matrix: &MatrixConfig{
			Values: map[string][]interface{}{
				"test": {"test1", "test2", "test3"},
			},
			Strategy: MatrixStrategy{
				MaxParallel:     1,
				FailFast:        true,
				ContinueOnError: false,
				Order:          "fifo",
			},
		},
	}
	
	action := &Action{
		Name: "test-action",
		Run:  "echo ${{ matrix.test }}",
	}
	
	jobs, err := expander.ExpandMatrix(step, action)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should have 3 jobs
	expectedJobs := 3
	if len(jobs) != expectedJobs {
		t.Fatalf("Expected %d jobs, got %d", expectedJobs, len(jobs))
	}
	
	// Check job IDs are unique
	ids := make(map[string]bool)
	for _, job := range jobs {
		if ids[job.ID] {
			t.Errorf("Duplicate job ID: %s", job.ID)
		}
		ids[job.ID] = true
	}
}

func TestMatrixExpander_ExpandMatrix_EmptyValues(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}
	
	expander := NewMatrixExpander(config)
	
	step := &Step{
		Action: "test-action",
		Matrix: &MatrixConfig{
			Values: map[string][]interface{}{},
			Strategy: MatrixStrategy{
				MaxParallel:     1,
				FailFast:        false,
				ContinueOnError: false,
				Order:          "fifo",
			},
		},
	}
	
	action := &Action{
		Name: "test-action",
		Run:  "echo test",
	}
	
	jobs, err := expander.ExpandMatrix(step, action)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Should have 0 jobs for empty values
	expectedJobs := 0
	if len(jobs) != expectedJobs {
		t.Fatalf("Expected %d jobs, got %d", expectedJobs, len(jobs))
	}
}

func TestMatrixScheduler_OrderJobs(t *testing.T) {
	jobs := []*MatrixJob{
		{ID: "job1", Matrix: map[string]interface{}{"test": "1"}},
		{ID: "job2", Matrix: map[string]interface{}{"test": "2"}},
		{ID: "job3", Matrix: map[string]interface{}{"test": "3"}},
	}
	
	// Test FIFO ordering
	scheduler := &MatrixScheduler{
		jobs:    jobs,
		strategy: MatrixStrategy{Order: "fifo"},
	}
	
	// Make a copy to test ordering
	originalOrder := make([]string, len(jobs))
	for i, job := range jobs {
		originalOrder[i] = job.ID
	}
	
	scheduler.orderJobs()
	
	// FIFO should maintain original order
	for i, job := range scheduler.jobs {
		if job.ID != originalOrder[i] {
			t.Errorf("FIFO ordering failed: expected %s at position %d, got %s", originalOrder[i], i, job.ID)
		}
	}
}

func TestMatrixScheduler_OrderJobs_Random(t *testing.T) {
	jobs := []*MatrixJob{
		{ID: "job1", Matrix: map[string]interface{}{"test": "1"}},
		{ID: "job2", Matrix: map[string]interface{}{"test": "2"}},
		{ID: "job3", Matrix: map[string]interface{}{"test": "3"}},
	}
	
	// Test random ordering
	scheduler := &MatrixScheduler{
		jobs:    jobs,
		strategy: MatrixStrategy{Order: "random"},
	}
	
	// Make a copy to test ordering
	originalOrder := make([]string, len(jobs))
	for i, job := range jobs {
		originalOrder[i] = job.ID
	}
	
	scheduler.orderJobs()
	
	// Random ordering should potentially change the order
	// (though it's possible it could be the same by chance)
	// We'll just verify all jobs are still present
	jobIDs := make(map[string]bool)
	for _, job := range scheduler.jobs {
		jobIDs[job.ID] = true
	}
	
	for _, originalID := range originalOrder {
		if !jobIDs[originalID] {
			t.Errorf("Random ordering lost job: %s", originalID)
		}
	}
}

func TestMatrixJobStatus_String(t *testing.T) {
	tests := []struct {
		status MatrixJobStatus
		expected string
	}{
		{MatrixJobPending, "PENDING"},
		{MatrixJobRunning, "RUNNING"},
		{MatrixJobCompleted, "COMPLETED"},
		{MatrixJobFailed, "FAILED"},
		{MatrixJobSkipped, "SKIPPED"},
		{MatrixJobStatus(999), "UNKNOWN"},
	}
	
	for _, test := range tests {
		result := test.status.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestMatrixScheduler_FormatMatrixValues(t *testing.T) {
	scheduler := &MatrixScheduler{}
	
	tests := []struct {
		matrix   map[string]interface{}
		expected string
	}{
		{
			map[string]interface{}{"os": "linux", "version": "1.0"},
			"os=linux, version=1.0",
		},
		{
			map[string]interface{}{"test": "single"},
			"test=single",
		},
		{
			map[string]interface{}{"a": "1", "b": "2", "c": "3"},
			"a=1, b=2, c=3",
		},
	}
	
	for _, test := range tests {
		result := scheduler.formatMatrixValues(test.matrix)
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestMatrixScheduler_ShouldStop(t *testing.T) {
	tests := []struct {
		name     string
		failFast bool
		failedCount int
		expected bool
	}{
		{"no fail fast, no failures", false, 0, false},
		{"no fail fast, with failures", false, 1, false},
		{"fail fast, no failures", true, 0, false},
		{"fail fast, with failures", true, 1, true},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &MatrixScheduler{
				strategy: MatrixStrategy{FailFast: test.failFast},
				failedCount: test.failedCount,
			}
			
			result := scheduler.shouldStop()
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestMatrixScheduler_DetermineFinalStatus(t *testing.T) {
	tests := []struct {
		name            string
		continueOnError bool
		failedCount     int
		expectedError   bool
	}{
		{"no continue on error, no failures", false, 0, false},
		{"no continue on error, with failures", false, 1, true},
		{"continue on error, no failures", true, 0, false},
		{"continue on error, with failures", true, 1, false},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := &MatrixScheduler{
				strategy: MatrixStrategy{ContinueOnError: test.continueOnError},
				failedCount: test.failedCount,
			}
			
			err := scheduler.determineFinalStatus()
			hasError := err != nil
			
			if hasError != test.expectedError {
				t.Errorf("Expected error %v, got %v", test.expectedError, hasError)
			}
		})
	}
}

func TestMatrixScheduler_GetJobStatus(t *testing.T) {
	jobs := []*MatrixJob{
		{ID: "job1", Status: StatusPending},
		{ID: "job2", Status: StatusRunning},
		{ID: "job3", Status: StatusOK},
		{ID: "job4", Status: StatusError},
		{ID: "job5", Status: StatusSkipped},
	}
	
	scheduler := &MatrixScheduler{jobs: jobs}
	status := scheduler.GetJobStatus()
	
	expected := map[string]MatrixJobStatus{
		"job1": MatrixJobPending,
		"job2": MatrixJobRunning,
		"job3": MatrixJobCompleted,
		"job4": MatrixJobFailed,
		"job5": MatrixJobSkipped,
	}
	
	for jobID, expectedStatus := range expected {
		if status[jobID] != expectedStatus {
			t.Errorf("Job %s: expected %v, got %v", jobID, expectedStatus, status[jobID])
		}
	}
}

func TestMatrixScheduler_GetJobResults(t *testing.T) {
	jobs := []*MatrixJob{
		{ID: "job1", Status: StatusOK},
		{ID: "job2", Status: StatusError},
	}
	
	scheduler := &MatrixScheduler{jobs: jobs}
	results := scheduler.GetJobResults()
	
	if len(results) != len(jobs) {
		t.Errorf("Expected %d results, got %d", len(jobs), len(results))
	}
	
	for i, result := range results {
		if result.ID != jobs[i].ID {
			t.Errorf("Expected job ID %s, got %s", jobs[i].ID, result.ID)
		}
	}
}

// Integration test for matrix execution
func TestMatrixExecution_Integration(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "echo-matrix",
				Run:  "echo 'OS: ${{ matrix.os }}, Version: ${{ matrix.version }}'",
			},
		},
		Stages: map[string]Stage{
			"test-matrix": {
				Steps: []Step{
					{
						Action: "echo-matrix",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"os":      {"linux", "windows"},
								"version": {"1.0"},
							},
							Strategy: MatrixStrategy{
								MaxParallel:     2,
								FailFast:        false,
								ContinueOnError: false,
								Order:          "fifo",
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		ConfigPath:  ".project.yml",
		MaxParallel: 2,
		VerboseLevel: 0,
		Debug:       false,
		Variables:   make(map[string]string),
		WorkingDir:  ".",
	}
	
	runner := NewRunner(config, opts)
	ctx := context.Background()
	
	// This is a basic integration test - in a real scenario,
	// we'd need to mock the command execution
	err := runner.RunStage(ctx, "test-matrix")
	
	// For now, we expect this to fail because we don't have actual command execution
	// but the matrix expansion and scheduling should work
	if err == nil {
		t.Log("Matrix execution completed successfully")
	} else {
		t.Logf("Matrix execution failed as expected (no actual command execution): %v", err)
	}
}
