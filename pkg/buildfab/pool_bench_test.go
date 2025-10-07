package buildfab

import (
	"context"
	"runtime"
	"sync"
	"testing"
)

// BenchmarkPool_Submit benchmarks task submission overhead
func BenchmarkPool_Submit(b *testing.B) {
	pool := NewExecutionPool("bench", runtime.NumCPU(), context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		task := Task{
			ID: "task",
			Execute: func(ctx context.Context) error {
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	wg.Wait()
	b.StopTimer()
}

// BenchmarkPool_ExecuteNoOp benchmarks execution of no-op tasks
func BenchmarkPool_ExecuteNoOp(b *testing.B) {
	pool := NewExecutionPool("bench", runtime.NumCPU(), context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		task := Task{
			ID: "task",
			Execute: func(ctx context.Context) error {
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	wg.Wait()
	b.StopTimer()
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "tasks/sec")
}

// BenchmarkPool_ExecuteParallel benchmarks parallel task execution
func BenchmarkPool_ExecuteParallel(b *testing.B) {
	numWorkers := runtime.NumCPU()
	pool := NewExecutionPool("bench", numWorkers, context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		task := Task{
			ID: "task",
			Execute: func(ctx context.Context) error {
				// Simulate minimal work
				_ = 1 + 1
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	wg.Wait()
	b.StopTimer()
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "tasks/sec")
}

// BenchmarkPool_SubmitConcurrent benchmarks concurrent submissions from multiple goroutines
func BenchmarkPool_SubmitConcurrent(b *testing.B) {
	pool := NewExecutionPool("bench", runtime.NumCPU(), context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	numGoroutines := 4
	tasksPerGoroutine := b.N / numGoroutines
	
	b.ResetTimer()
	for g := 0; g < numGoroutines; g++ {
		go func() {
			for i := 0; i < tasksPerGoroutine; i++ {
				wg.Add(1)
				task := Task{
					ID: "task",
					Execute: func(ctx context.Context) error {
						return nil
					},
					OnComplete: func(err error) {
						wg.Done()
					},
				}
				pool.Submit(task)
			}
		}()
	}
	
	wg.Wait()
	b.StopTimer()
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "tasks/sec")
}

// BenchmarkPoolManager_GetPool benchmarks pool retrieval
func BenchmarkPoolManager_GetPool(b *testing.B) {
	pm := NewPoolManager(runtime.NumCPU(), context.Background())
	defer pm.StopAll()
	
	// Create some matrix pools
	pm.GetOrCreateMatrixPool("pool1", 2)
	pm.GetOrCreateMatrixPool("pool2", 2)
	pm.GetOrCreateMatrixPool("pool3", 2)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 4 {
		case 0:
			pm.GetPool("global")
		case 1:
			pm.GetPool("pool1")
		case 2:
			pm.GetPool("pool2")
		case 3:
			pm.GetPool("pool3")
		}
	}
}

// BenchmarkPoolManager_GetOrCreateMatrixPool benchmarks pool creation/retrieval
func BenchmarkPoolManager_GetOrCreateMatrixPool(b *testing.B) {
	pm := NewPoolManager(runtime.NumCPU(), context.Background())
	defer pm.StopAll()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will create once and retrieve the rest of the times
		pm.GetOrCreateMatrixPool("bench-pool", 2)
	}
}

// BenchmarkPool_SmallTasks benchmarks many small tasks
func BenchmarkPool_SmallTasks(b *testing.B) {
	pool := NewExecutionPool("bench", runtime.NumCPU(), context.Background())
	pool.Start()
	defer pool.Stop()
	
	var wg sync.WaitGroup
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		task := Task{
			ID: "small-task",
			Execute: func(ctx context.Context) error {
				// Very small work unit
				sum := 0
				for j := 0; j < 10; j++ {
					sum += j
				}
				_ = sum
				return nil
			},
			OnComplete: func(err error) {
				wg.Done()
			},
		}
		pool.Submit(task)
	}
	
	wg.Wait()
	b.StopTimer()
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "tasks/sec")
}

