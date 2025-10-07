package buildfab

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPhase1_GlobalPoolLimit tests that global max_parallel limits concurrent execution
func TestPhase1_GlobalPoolLimit(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 2, // Global limit
		},
		Actions: []Action{
			{Name: "action-1", Run: "sleep 2"},
			{Name: "action-2", Run: "sleep 2"},
			{Name: "action-3", Run: "sleep 2"},
			{Name: "action-4", Run: "sleep 2"},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{Action: "action-1"},
					{Action: "action-2"},
					{Action: "action-3"},
					{Action: "action-4"},
				},
			},
		},
	}
	
	opts := &RunOptions{
		ConfigPath:   ".project.yml",
		VerboseLevel: 0,
		Debug:        true, // Enable debug to see pool activity
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
	
	// With max_parallel=2 and 4 actions of 2s each:
	// Wave 1: actions 1,2 (2s)
	// Wave 2: actions 3,4 (2s)
	// Total: ~4s (allow 0.5s overhead)
	minExpected := 3500 * time.Millisecond
	maxExpected := 4500 * time.Millisecond
	
	if duration < minExpected {
		t.Errorf("Execution too fast (%v), pool limit may not be working", duration)
	}
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), expected ~4s with max_parallel=2", duration)
	}
	
	t.Logf("Execution time: %v (expected ~4s with max_parallel=2)", duration)
}

// TestPhase1_UnlimitedParallel tests execution with unlimited parallel
func TestPhase1_UnlimitedParallel(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test",
			MaxParallel: 0, // Unlimited (use CPU cores)
		},
		Actions: []Action{
			{Name: "action-1", Run: "sleep 2"},
			{Name: "action-2", Run: "sleep 2"},
			{Name: "action-3", Run: "sleep 2"},
			{Name: "action-4", Run: "sleep 2"},
		},
		Stages: map[string]Stage{
			"test": {
				Steps: []Step{
					{Action: "action-1"},
					{Action: "action-2"},
					{Action: "action-3"},
					{Action: "action-4"},
				},
			},
		},
	}
	
	opts := &RunOptions{
		MaxParallel:  0, // Use config default
		VerboseLevel: 0,
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
	
	// All actions should run in parallel: ~2s total (allow overhead)
	maxExpected := 2500 * time.Millisecond
	
	if duration > maxExpected {
		t.Errorf("Execution too slow (%v), all actions should run in parallel (~2s)", duration)
	}
	
	t.Logf("Execution time: %v (expected ~2s with unlimited parallel)", duration)
}

// TestExecutionPool_BasicExecution tests basic pool task execution
func TestExecutionPool_BasicExecution(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var executed int32
	var wg sync.WaitGroup
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(ctx context.Context) error {
				atomic.AddInt32(&executed, 1)
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	
	wg.Wait()
	
	if executed != 10 {
		t.Errorf("Expected 10 tasks executed, got %d", executed)
	}
	
	stats := pool.GetStats()
	if stats.TasksCompleted != 10 {
		t.Errorf("Expected 10 completed tasks, got %d", stats.TasksCompleted)
	}
}

// TestExecutionPool_MaxParallelLimit tests that pool enforces max_parallel limit
func TestExecutionPool_MaxParallelLimit(t *testing.T) {
	maxParallel := 2
	pool := NewExecutionPool("test", maxParallel, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var currentRunning int32
	var maxObserved int32
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(ctx context.Context) error {
				running := atomic.AddInt32(&currentRunning, 1)
				
				mu.Lock()
				if running > maxObserved {
					maxObserved = running
				}
				mu.Unlock()
				
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&currentRunning, -1)
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	
	wg.Wait()
	
	if maxObserved > int32(maxParallel) {
		t.Errorf("Max parallel limit violated: observed %d concurrent tasks, limit is %d", maxObserved, maxParallel)
	}
	
	t.Logf("Max concurrent tasks observed: %d (limit: %d)", maxObserved, maxParallel)
}

// TestPoolManager_GlobalPool tests basic PoolManager functionality
func TestPoolManager_GlobalPool(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	defer pm.StopAll()
	
	globalPool := pm.GetGlobalPool()
	if globalPool == nil {
		t.Fatal("Global pool should not be nil")
	}
	
	if globalPool.GetName() != "global" {
		t.Errorf("Expected global pool name 'global', got '%s'", globalPool.GetName())
	}
	
	// Test that we can submit tasks to global pool
	var wg sync.WaitGroup
	wg.Add(1)
	task := Task{
		ID: "test-task",
		Execute: func(ctx context.Context) error {
			return nil
		},
		OnComplete: func(err error) {
			wg.Done()
		},
	}
	
	if err := globalPool.Submit(task); err != nil {
		t.Fatalf("Failed to submit task to global pool: %v", err)
	}
	
	wg.Wait()
}

// TestPoolManager_MatrixPools tests matrix pool creation and retrieval
func TestPoolManager_MatrixPools(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	defer pm.StopAll()
	
	// Create matrix pools
	buildPool := pm.GetOrCreateMatrixPool("matrix-build", 2)
	testPool := pm.GetOrCreateMatrixPool("matrix-test", 3)
	
	if buildPool == nil || testPool == nil {
		t.Fatal("Failed to create matrix pools")
	}
	
	// Verify pools are different
	if buildPool == testPool {
		t.Error("Expected different pools for different matrices")
	}
	
	// Verify pool retrieval
	retrieved := pm.GetPool("matrix-build")
	if retrieved != buildPool {
		t.Error("Failed to retrieve correct pool")
	}
	
	// Verify global pool retrieval
	global := pm.GetPool("global")
	if global != pm.GetGlobalPool() {
		t.Error("GetPool('global') should return global pool")
	}
}

// TestExecutionPool_ContextCancellation tests that pool respects context cancellation
func TestExecutionPool_ContextCancellation(t *testing.T) {
	testCtx, testCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer testCancel()
	
	pool := NewExecutionPool("test", 2, testCtx)
	pool.Start()
	defer pool.Stop()
	
	poolCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	var executed int32
	var wg sync.WaitGroup
	
	// Submit tasks that would take longer than context timeout
	for i := 0; i < 5; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(taskCtx context.Context) error {
				select {
				case <-poolCtx.Done():
					return poolCtx.Err()
				case <-time.After(500 * time.Millisecond):
					atomic.AddInt32(&executed, 1)
					return nil
				}
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	
	wg.Wait()
	
	// Some tasks should be cancelled
	if executed >= 5 {
		t.Errorf("Expected some tasks to be cancelled, but all %d executed", executed)
	}
	
	t.Logf("Executed %d tasks before context cancellation", executed)
}

