package buildfab

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegration_MatrixMaxParallel2 tests matrix execution with max_parallel=2
func TestIntegration_MatrixMaxParallel2(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 8, // High global limit
		},
		Actions: []Action{
			{
				Name: "matrix-action",
				Run:  "sleep 2",
			},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{
						Action: "matrix-action",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"job": {"job1", "job2", "job3", "job4"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 2, // Only 2 concurrent
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// 4 jobs with max_parallel=2:
	// Wave 1: jobs 1,2 (2s)
	// Wave 2: jobs 3,4 (2s)
	// Total: ~4s (allow overhead)
	minExpected := 3500 * time.Millisecond
	maxExpected := 4500 * time.Millisecond
	
	if duration < minExpected {
		t.Errorf("Execution too fast (%v), max_parallel=2 may not be enforced", duration)
	}
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~4s with max_parallel=2", duration)
	}
	
	t.Logf("Integration test passed: 4 jobs with max_parallel=2 took %v (expected ~4s)", duration)
}

// TestIntegration_MatrixSequential tests sequential matrix execution with max_parallel=1
func TestIntegration_MatrixSequential(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test",
		},
		Actions: []Action{
			{
				Name: "matrix-action",
				Run:  "sleep 2",
			},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{
						Action: "matrix-action",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"job": {"seqA", "seqB", "seqC"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 1, // Sequential
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// 3 jobs sequential: 3 × 2s = ~6s
	minExpected := 5500 * time.Millisecond
	maxExpected := 6500 * time.Millisecond
	
	if duration < minExpected {
		t.Errorf("Execution too fast (%v), sequential execution may not be working", duration)
	}
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~6s sequential", duration)
	}
	
	t.Logf("Integration test passed: 3 jobs sequential took %v (expected ~6s)", duration)
}

// TestIntegration_MatrixUsesGlobalPool tests matrix without max_parallel uses global pool
func TestIntegration_MatrixUsesGlobalPool(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 8, // Global limit
		},
		Actions: []Action{
			{
				Name: "matrix-action",
				Run:  "sleep 2",
			},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{
						Action: "matrix-action",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"job": {"globalA", "globalB", "globalC", "globalD"},
							},
							Strategy: MatrixStrategy{
								// NO MaxParallel - uses global pool
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// All 4 jobs should run in parallel: ~2s total
	maxExpected := 2500 * time.Millisecond
	
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), all jobs should run in parallel (~2s)", duration)
	}
	
	t.Logf("Integration test passed: 4 jobs in parallel took %v (expected ~2s)", duration)
}

// TestIntegration_MixedRegularAndMatrix tests regular steps + matrix steps
func TestIntegration_MixedRegularAndMatrix(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 4, // Global limit
		},
		Actions: []Action{
			{Name: "regular-1", Run: "sleep 1"},
			{Name: "regular-2", Run: "sleep 1"},
			{Name: "matrix-action", Run: "sleep 1"},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{Action: "regular-1"},
					{Action: "regular-2"},
					{
						Action: "matrix-action",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"item": {"m1", "m2"},
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
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// 2 regular steps + 2 matrix jobs, all can run in parallel with global=4
	// Total: ~1s (all concurrent)
	maxExpected := 1500 * time.Millisecond
	
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~1s with mixed workload", duration)
	}
	
	t.Logf("Integration test passed: mixed workload took %v (expected ~1s)", duration)
}

// TestIntegration_GlobalLimitRestrictsMatrix tests that low global limit restricts matrix
func TestIntegration_GlobalLimitRestrictsMatrix(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 2, // Low global limit
		},
		Actions: []Action{
			{Name: "matrix-action", Run: "sleep 1"},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{
						Action: "matrix-action",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"item": {"1", "2", "3", "4"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 4, // Wants 4 concurrent
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        true, // Enable to see pool assignment
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// 4 jobs with effective max_parallel=min(2,4)=2:
	// Wave 1: jobs 1,2 (1s)
	// Wave 2: jobs 3,4 (1s)
	// Total: ~2s
	minExpected := 1500 * time.Millisecond
	maxExpected := 2500 * time.Millisecond
	
	if duration < minExpected {
		t.Errorf("Execution too fast (%v), global limit may not be enforced", duration)
	}
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~2s with global limit=2", duration)
	}
	
	t.Logf("Integration test passed: global limit restricts matrix, took %v (expected ~2s)", duration)
}

// TestIntegration_DependenciesWithPools tests that dependencies work correctly with pools
func TestIntegration_DependenciesWithPools(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 4,
		},
		Actions: []Action{
			{Name: "setup", Run: "sleep 1"},
			{Name: "matrix-action", Run: "sleep 1"},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{Action: "setup"}, // Runs first
					
					// Matrix jobs depend on setup
					{
						Action:  "matrix-action",
						Require: []string{"setup"},
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"item": {"1", "2", "3"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 3,
							},
						},
					},
				},
			},
		},
	}
	
	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}
	
	runner := NewRunner(config, opts)
	
	start := time.Now()
	ctx := context.Background()
	err := runner.RunStage(ctx, "test")
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}
	
	// Execution:
	// 1. setup (1s)
	// 2. matrix jobs all 3 concurrent (1s)
	// Total: ~2s
	minExpected := 1500 * time.Millisecond
	maxExpected := 2500 * time.Millisecond
	
	if duration < minExpected {
		t.Errorf("Execution too fast (%v), dependencies may not be working", duration)
	}
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~2s with dependencies", duration)
	}
	
	t.Logf("Integration test passed: dependencies with pools took %v (expected ~2s)", duration)
}

