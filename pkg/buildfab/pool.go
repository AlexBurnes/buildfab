package buildfab

import (
	"context"
	"fmt"
	"sync"
)

// Task represents a unit of work for the execution pool
type Task struct {
	ID         string                         // Unique task identifier
	Execute    func(context.Context) error    // Function to execute
	OnStart    func()                         // Called when task starts
	OnComplete func(error)                    // Called when task completes
	Priority   int                            // For future use
}

// PoolStats tracks statistics for an execution pool
type PoolStats struct {
	TasksQueued    int
	TasksRunning   int
	TasksCompleted int
	TasksFailed    int
}

// ExecutionPool manages concurrent task execution with a worker pool
type ExecutionPool struct {
	name       string
	maxWorkers int
	taskQueue  chan Task
	activeJobs sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	running    bool
	stats      PoolStats
}

// NewExecutionPool creates a new execution pool
func NewExecutionPool(name string, maxWorkers int) *ExecutionPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &ExecutionPool{
		name:       name,
		maxWorkers: maxWorkers,
		taskQueue:  make(chan Task, maxWorkers*2), // Buffered queue
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}
	
	return pool
}

// Start starts the worker pool
func (p *ExecutionPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		return
	}
	
	p.running = true
	
	// Start worker goroutines
	for i := 0; i < p.maxWorkers; i++ {
		go p.worker(i)
	}
}

// worker processes tasks from the queue
func (p *ExecutionPool) worker(id int) {
	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				return // Channel closed
			}
			
			p.executeTask(task)
			
		case <-p.ctx.Done():
			return
		}
	}
}

// executeTask executes a single task
func (p *ExecutionPool) executeTask(task Task) {
	p.mu.Lock()
	p.stats.TasksRunning++
	p.mu.Unlock()
	
	p.activeJobs.Add(1)
	defer p.activeJobs.Done()
	
	if task.OnStart != nil {
		task.OnStart()
	}
	
	err := task.Execute(p.ctx)
	
	p.mu.Lock()
	p.stats.TasksRunning--
	if err != nil {
		p.stats.TasksFailed++
	} else {
		p.stats.TasksCompleted++
	}
	p.mu.Unlock()
	
	if task.OnComplete != nil {
		task.OnComplete(err)
	}
}

// Submit submits a task to the pool
func (p *ExecutionPool) Submit(task Task) error {
	p.mu.RLock()
	if !p.running {
		p.mu.RUnlock()
		return fmt.Errorf("pool %s is not running", p.name)
	}
	p.mu.RUnlock()
	
	p.mu.Lock()
	p.stats.TasksQueued++
	p.mu.Unlock()
	
	select {
	case p.taskQueue <- task:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("pool %s is shutting down", p.name)
	}
}

// Wait waits for all submitted tasks to complete
func (p *ExecutionPool) Wait() {
	p.activeJobs.Wait()
}

// Stop stops the pool and waits for completion
func (p *ExecutionPool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	
	close(p.taskQueue)
	p.cancel()
	p.running = false
	p.mu.Unlock()
	
	// Wait for all active jobs to complete
	p.activeJobs.Wait()
}

// GetStats returns current pool statistics
func (p *ExecutionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// GetName returns the pool name
func (p *ExecutionPool) GetName() string {
	return p.name
}

// PoolManager coordinates multiple execution pools
type PoolManager struct {
	globalPool  *ExecutionPool
	matrixPools map[string]*ExecutionPool
	mu          sync.RWMutex
}

// NewPoolManager creates a new pool manager
func NewPoolManager(globalMaxWorkers int) *PoolManager {
	globalPool := NewExecutionPool("global", globalMaxWorkers)
	globalPool.Start()
	
	return &PoolManager{
		globalPool:  globalPool,
		matrixPools: make(map[string]*ExecutionPool),
	}
}

// GetOrCreateMatrixPool gets or creates a matrix-specific pool
func (pm *PoolManager) GetOrCreateMatrixPool(name string, maxWorkers int) *ExecutionPool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pool, exists := pm.matrixPools[name]; exists {
		return pool
	}
	
	pool := NewExecutionPool(name, maxWorkers)
	pool.Start()
	pm.matrixPools[name] = pool
	
	return pool
}

// GetPool returns a pool by name (global or matrix-specific)
func (pm *PoolManager) GetPool(name string) *ExecutionPool {
	if name == "" || name == "global" {
		return pm.globalPool
	}
	
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return pm.matrixPools[name]
}

// GetGlobalPool returns the global pool
func (pm *PoolManager) GetGlobalPool() *ExecutionPool {
	return pm.globalPool
}

// StopAll stops all pools
func (pm *PoolManager) StopAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.globalPool.Stop()
	
	for _, pool := range pm.matrixPools {
		pool.Stop()
	}
}

// WaitAll waits for all pools to complete their tasks
func (pm *PoolManager) WaitAll() {
	pm.globalPool.Wait()
	
	pm.mu.RLock()
	pools := make([]*ExecutionPool, 0, len(pm.matrixPools))
	for _, pool := range pm.matrixPools {
		pools = append(pools, pool)
	}
	pm.mu.RUnlock()
	
	for _, pool := range pools {
		pool.Wait()
	}
}

