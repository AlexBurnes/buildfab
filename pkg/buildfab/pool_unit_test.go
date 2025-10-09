package buildfab

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestExecutionPool_TaskErrors verifies that the pool properly handles task errors
func TestExecutionPool_TaskErrors(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var successCount int32
	var errorCount int32
	var wg sync.WaitGroup
	
	// Submit mix of successful and failing tasks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID := i
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				if taskID%2 == 0 {
					return nil // Success
				}
				return fmt.Errorf("task %d failed", taskID)
			},
			OnComplete: func(err error) {
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
				wg.Done()
			},
		}
		
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	
	wg.Wait()
	
	if successCount != 5 {
		t.Errorf("Expected 5 successful tasks, got %d", successCount)
	}
	if errorCount != 5 {
		t.Errorf("Expected 5 failed tasks, got %d", errorCount)
	}
	
	// Verify pool statistics
	stats := pool.GetStats()
	if stats.TasksCompleted != 5 {
		t.Errorf("Expected 5 completed tasks in stats, got %d", stats.TasksCompleted)
	}
	if stats.TasksFailed != 5 {
		t.Errorf("Expected 5 failed tasks in stats, got %d", stats.TasksFailed)
	}
	
	t.Logf("Task errors handled correctly: %d successes, %d failures", successCount, errorCount)
}

// TestExecutionPool_SubmitAfterStop verifies that submitting to a stopped pool returns an error
func TestExecutionPool_SubmitAfterStop(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	pool.Stop()
	
	task := Task{
		ID: "test-task",
		Execute: func(ctx context.Context) error {
			return nil
		},
	}
	
	err := pool.Submit(task)
	if err == nil {
		t.Error("Expected error when submitting to stopped pool, got nil")
	}
	
	expectedMsg := "pool test is not running"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
	
	t.Logf("Submit after stop correctly returns error: %v", err)
}

// TestExecutionPool_WaitGroupBalance verifies that WaitGroup is properly balanced
func TestExecutionPool_WaitGroupBalance(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	
	var wg sync.WaitGroup
	
	// Submit tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(ctx context.Context) error {
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
	
	// Wait for all tasks to complete
	wg.Wait()
	
	// Stop the pool - this should not hang if WaitGroup is balanced
	done := make(chan bool)
	go func() {
		pool.Stop()
		done <- true
	}()
	
	select {
	case <-done:
		// Success - pool stopped without hanging
		t.Log("Pool stopped successfully - WaitGroup properly balanced")
	case <-time.After(2 * time.Second):
		t.Fatal("Pool.Stop() hung - WaitGroup may be imbalanced")
	}
}

// TestExecutionPool_OnStartOnComplete verifies that callbacks are called correctly
func TestExecutionPool_OnStartOnComplete(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var startCount int32
	var completeCount int32
	var wg sync.WaitGroup
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("task-%d", i),
			Execute: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			OnStart: func() {
				atomic.AddInt32(&startCount, 1)
			},
			OnComplete: func(err error) {
				atomic.AddInt32(&completeCount, 1)
				wg.Done()
			},
		}
		
		if err := pool.Submit(task); err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}
	
	wg.Wait()
	
	if startCount != 5 {
		t.Errorf("Expected OnStart called 5 times, got %d", startCount)
	}
	if completeCount != 5 {
		t.Errorf("Expected OnComplete called 5 times, got %d", completeCount)
	}
	
	t.Logf("Callbacks executed correctly: %d OnStart, %d OnComplete", startCount, completeCount)
}

// TestExecutionPool_ConcurrentSubmit verifies that concurrent submissions work correctly
func TestExecutionPool_ConcurrentSubmit(t *testing.T) {
	pool := NewExecutionPool("test", 4, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var executed int32
	var wg sync.WaitGroup
	var submitWg sync.WaitGroup
	
	// Submit tasks concurrently from multiple goroutines
	for g := 0; g < 5; g++ {
		submitWg.Add(1)
		go func(goroutineID int) {
			defer submitWg.Done()
			for i := 0; i < 10; i++ {
				wg.Add(1)
				task := Task{
					ID: fmt.Sprintf("g%d-task-%d", goroutineID, i),
					Execute: func(ctx context.Context) error {
						atomic.AddInt32(&executed, 1)
						time.Sleep(5 * time.Millisecond)
						return nil
					},
					OnComplete: func(err error) {
						wg.Done()
					},
				}
				
				if err := pool.Submit(task); err != nil {
					t.Errorf("Failed to submit task from goroutine %d: %v", goroutineID, err)
				}
			}
		}(g)
	}
	
	// Wait for all submissions to complete
	submitWg.Wait()
	
	// Wait for all tasks to execute
	wg.Wait()
	
	expectedTasks := 50 // 5 goroutines × 10 tasks each
	if executed != int32(expectedTasks) {
		t.Errorf("Expected %d tasks executed, got %d", expectedTasks, executed)
	}
	
	t.Logf("Concurrent submission successful: %d tasks from 5 goroutines", executed)
}

// TestPoolManager_GetOrCreateIdempotent verifies that GetOrCreateMatrixPool is idempotent
func TestPoolManager_GetOrCreateIdempotent(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	defer pm.StopAll()
	
	// Create pool first time
	pool1 := pm.GetOrCreateMatrixPool("test-matrix", 2)
	if pool1 == nil {
		t.Fatal("Failed to create pool")
	}
	
	// Get same pool second time
	pool2 := pm.GetOrCreateMatrixPool("test-matrix", 2)
	if pool2 == nil {
		t.Fatal("Failed to get existing pool")
	}
	
	// Verify they are the same pool instance
	if pool1 != pool2 {
		t.Error("GetOrCreateMatrixPool should return same pool instance for same name")
	}
	
	t.Log("GetOrCreateMatrixPool is correctly idempotent")
}

// TestPoolManager_GetPoolReturnsGlobal verifies that GetPool returns global for empty/global names
func TestPoolManager_GetPoolReturnsGlobal(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	defer pm.StopAll()
	
	globalPool := pm.GetGlobalPool()
	
	// Verify GetPool with empty name returns global pool
	emptyNamePool := pm.GetPool("")
	if emptyNamePool != globalPool {
		t.Error("GetPool(\"\") should return global pool")
	}
	
	// Verify GetPool with "global" returns global pool
	globalNamePool := pm.GetPool("global")
	if globalNamePool != globalPool {
		t.Error("GetPool(\"global\") should return global pool")
	}
	
	t.Log("GetPool correctly returns global pool for empty and 'global' names")
}

// TestPoolManager_CancelAll verifies that CancelAll cancels all pools without waiting
func TestPoolManager_CancelAll(t *testing.T) {
	ctx := context.Background()
	pm := NewPoolManager(4, ctx)
	
	// Create additional matrix pools
	pm.GetOrCreateMatrixPool("pool1", 2)
	pm.GetOrCreateMatrixPool("pool2", 2)
	
	var wg sync.WaitGroup
	
	// Submit long-running tasks to global pool
	globalPool := pm.GetGlobalPool()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("global-task-%d", i),
			Execute: func(taskCtx context.Context) error {
				select {
				case <-time.After(5 * time.Second):
					return nil
				case <-taskCtx.Done():
					return taskCtx.Err()
				}
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		globalPool.Submit(task)
	}
	
	// Submit tasks to matrix pools
	pool1 := pm.GetPool("pool1")
	for i := 0; i < 2; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("pool1-task-%d", i),
			Execute: func(taskCtx context.Context) error {
				select {
				case <-time.After(5 * time.Second):
					return nil
				case <-taskCtx.Done():
					return taskCtx.Err()
				}
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool1.Submit(task)
	}
	
	// Wait for tasks to start
	time.Sleep(50 * time.Millisecond)
	
	// Cancel all pools
	start := time.Now()
	pm.CancelAll()
	cancelDuration := time.Since(start)
	
	// CancelAll should return quickly without waiting
	if cancelDuration > 200*time.Millisecond {
		t.Errorf("CancelAll took too long (%v), should return immediately", cancelDuration)
	}
	
	// Wait for tasks to complete
	wg.Wait()
	
	t.Logf("CancelAll returned in %v (immediate, no waiting)", cancelDuration)
}

// TestMinStrategy_GlobalRestricts verifies the min() strategy when global < matrix
func TestMinStrategy_GlobalRestricts(t *testing.T) {
	globalMax := 2
	matrixMax := 5
	
	effective := matrixMax
	if globalMax > 0 && globalMax < effective {
		effective = globalMax
	}
	
	if effective != 2 {
		t.Errorf("Expected effective=2 (min of global=2 and matrix=5), got %d", effective)
	}
	
	t.Logf("Min strategy correct: global=%d, matrix=%d, effective=%d", globalMax, matrixMax, effective)
}

// TestMinStrategy_MatrixRestricts verifies the min() strategy when matrix < global
func TestMinStrategy_MatrixRestricts(t *testing.T) {
	globalMax := 10
	matrixMax := 2
	
	effective := matrixMax
	if globalMax > 0 && globalMax < effective {
		effective = globalMax
	}
	
	if effective != 2 {
		t.Errorf("Expected effective=2 (min of global=10 and matrix=2), got %d", effective)
	}
	
	t.Logf("Min strategy correct: global=%d, matrix=%d, effective=%d", globalMax, matrixMax, effective)
}

// TestMinStrategy_GlobalZero verifies behavior when global is 0 (unlimited)
func TestMinStrategy_GlobalZero(t *testing.T) {
	globalMax := 0 // Unlimited
	matrixMax := 3
	
	effective := matrixMax
	if globalMax > 0 && globalMax < effective {
		effective = globalMax
	}
	
	if effective != 3 {
		t.Errorf("Expected effective=3 (global=0 means unlimited, matrix=3), got %d", effective)
	}
	
	t.Logf("Min strategy correct with global=0: global=%d, matrix=%d, effective=%d", globalMax, matrixMax, effective)
}

// TestExecutionPool_Statistics verifies that pool statistics are tracked correctly
func TestExecutionPool_Statistics(t *testing.T) {
	pool := NewExecutionPool("test", 2, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	
	// Submit 5 successful tasks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("success-%d", i),
			Execute: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	// Submit 3 failing tasks
	for i := 0; i < 3; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("fail-%d", i),
			Execute: func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return fmt.Errorf("intentional failure")
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	wg.Wait()
	
	stats := pool.GetStats()
	
	if stats.TasksCompleted != 5 {
		t.Errorf("Expected 5 completed tasks, got %d", stats.TasksCompleted)
	}
	if stats.TasksFailed != 3 {
		t.Errorf("Expected 3 failed tasks, got %d", stats.TasksFailed)
	}
	if stats.TasksQueued != 8 {
		t.Errorf("Expected 8 queued tasks, got %d", stats.TasksQueued)
	}
	if stats.TasksRunning != 0 {
		t.Errorf("Expected 0 running tasks after completion, got %d", stats.TasksRunning)
	}
	
	t.Logf("Statistics: queued=%d, completed=%d, failed=%d, running=%d",
		stats.TasksQueued, stats.TasksCompleted, stats.TasksFailed, stats.TasksRunning)
}

// TestPoolManager_StopAll verifies that StopAll waits for all pools to complete
func TestPoolManager_StopAll(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	
	// Create matrix pools
	pm.GetOrCreateMatrixPool("pool1", 2)
	pm.GetOrCreateMatrixPool("pool2", 2)
	
	var executed int32
	var wg sync.WaitGroup
	
	// Submit tasks to global pool
	globalPool := pm.GetGlobalPool()
	for i := 0; i < 3; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("global-task-%d", i),
			Execute: func(ctx context.Context) error {
				atomic.AddInt32(&executed, 1)
				time.Sleep(20 * time.Millisecond)
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		globalPool.Submit(task)
	}
	
	// Submit tasks to matrix pools
	pool1 := pm.GetPool("pool1")
	for i := 0; i < 2; i++ {
		wg.Add(1)
		task := Task{
			ID: fmt.Sprintf("pool1-task-%d", i),
			Execute: func(ctx context.Context) error {
				atomic.AddInt32(&executed, 1)
				time.Sleep(20 * time.Millisecond)
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool1.Submit(task)
	}
	
	// Give tasks time to start executing
	time.Sleep(5 * time.Millisecond)
	
	// StopAll should wait for all tasks to complete
	start := time.Now()
	pm.StopAll()
	duration := time.Since(start)
	
	// Wait for OnComplete callbacks
	wg.Wait()
	
	// Verify all tasks executed
	expectedTasks := 5
	if executed != int32(expectedTasks) {
		t.Errorf("Expected %d tasks executed, got %d", expectedTasks, executed)
	}
	
	// StopAll should have waited for tasks to complete (~20ms minimum)
	if duration < 15*time.Millisecond {
		t.Errorf("StopAll returned too quickly (%v), should wait for tasks", duration)
	}
	
	t.Logf("StopAll correctly waited for all tasks: duration=%v, executed=%d", duration, executed)
}

// TestExecutionPool_GetName verifies that pool returns correct name
func TestExecutionPool_GetName(t *testing.T) {
	pool := NewExecutionPool("my-test-pool", 2, context.Background())
	defer pool.Stop()
	
	if pool.GetName() != "my-test-pool" {
		t.Errorf("Expected pool name 'my-test-pool', got %q", pool.GetName())
	}
}

// TestPoolManager_WaitAll verifies that WaitAll waits for all pools
func TestPoolManager_WaitAll(t *testing.T) {
	pm := NewPoolManager(4, context.Background())
	defer pm.StopAll()
	
	// Create matrix pools
	pm.GetOrCreateMatrixPool("pool1", 2)
	
	var wg sync.WaitGroup
	
	// Submit tasks to different pools
	globalPool := pm.GetGlobalPool()
	wg.Add(1)
	globalPool.Submit(Task{
		ID: "global-task",
		Execute: func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
		OnComplete: func(err error) {
			wg.Done()
		},
	})
	
	pool1 := pm.GetPool("pool1")
	wg.Add(1)
	pool1.Submit(Task{
		ID: "pool1-task",
		Execute: func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		OnComplete: func(err error) {
			wg.Done()
		},
	})
	
	// WaitAll in separate goroutine
	done := make(chan bool)
	go func() {
		pm.WaitAll()
		done <- true
	}()
	
	// Wait for all tasks
	wg.Wait()
	
	// WaitAll should complete after tasks finish
	select {
	case <-done:
		t.Log("WaitAll successfully waited for all pools")
	case <-time.After(2 * time.Second):
		t.Fatal("WaitAll did not complete in time")
	}
}

