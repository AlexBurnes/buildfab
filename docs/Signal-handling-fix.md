# Signal Handling Fix for DAG Executor

## Problem Description

The DAG executor was not terminating running jobs when receiving Ctrl+C or INT signal. Users would press Ctrl+C but the running jobs would continue executing indefinitely, requiring forceful termination.

## Root Cause

The issue was caused by **context isolation** in the ExecutionPool implementation. Here's what was happening:

### 1. Signal Handler Context (Correct)

In `cmd/buildfab/main.go:393`:
```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer cancel()
```

This context was properly set up to receive INT/TERM signals.

### 2. Pool Context Isolation (Problem)

In `pkg/buildfab/pool.go:40-41` (OLD CODE):
```go
func NewExecutionPool(name string, maxWorkers int) *ExecutionPool {
    ctx, cancel := context.WithCancel(context.Background())  // ❌ ISOLATED CONTEXT!
```

**The pool was creating its own independent context from `context.Background()`**, breaking the cancellation chain!

### 3. Task Execution (Problem)

In `pkg/buildfab/pool.go:102` (OLD CODE):
```go
err := task.Execute(p.ctx)  // Uses pool's isolated context, not parent context!
```

### Why Jobs Didn't Terminate

When Ctrl+C was pressed:

1. ✅ Main context gets cancelled (signal received)
2. ❌ Pool context remains active (isolated from main context)  
3. ❌ Tasks receive pool's context, not the cancelled main context
4. ❌ Running commands don't receive cancellation signal
5. ❌ Jobs continue running indefinitely

## Solution

The fix involves passing the parent context through the execution pool hierarchy, preserving the cancellation chain.

### Changes Made

#### 1. Modified `NewExecutionPool` Signature

**File**: `pkg/buildfab/pool.go`

```go
// NEW: Accept parent context parameter
func NewExecutionPool(name string, maxWorkers int, parentCtx context.Context) *ExecutionPool {
    // If no parent context provided, use background context as fallback
    if parentCtx == nil {
        parentCtx = context.Background()
    }
    
    // Create pool context derived from parent - this preserves cancellation chain
    ctx, cancel := context.WithCancel(parentCtx)  // ✅ DERIVED FROM PARENT!
    
    pool := &ExecutionPool{
        name:       name,
        maxWorkers: maxWorkers,
        taskQueue:  make(chan Task, maxWorkers*2),
        ctx:        ctx,
        cancel:     cancel,
        running:    false,
    }
    
    return pool
}
```

#### 2. Modified `NewPoolManager` Signature

**File**: `pkg/buildfab/pool.go`

```go
// NEW: Accept parent context parameter
func NewPoolManager(globalMaxWorkers int, parentCtx context.Context) *PoolManager {
    // Create global pool with parent context for proper cancellation propagation
    globalPool := NewExecutionPool("global", globalMaxWorkers, parentCtx)
    globalPool.Start()
    
    pm := &PoolManager{
        globalPool:  globalPool,
        matrixPools: make(map[string]*ExecutionPool),
        parentCtx:   parentCtx,  // Store for creating matrix pools
    }
    
    return pm
}
```

#### 3. Added parentCtx to PoolManager Struct

**File**: `pkg/buildfab/pool.go`

```go
type PoolManager struct {
    globalPool  *ExecutionPool
    matrixPools map[string]*ExecutionPool
    parentCtx   context.Context  // NEW: Parent context for creating new pools
    mu          sync.RWMutex
}
```

#### 4. Updated Matrix Pool Creation

**File**: `pkg/buildfab/pool.go`

```go
func (pm *PoolManager) GetOrCreateMatrixPool(name string, maxWorkers int) *ExecutionPool {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    if pool, exists := pm.matrixPools[name]; exists {
        return pool
    }
    
    // Create pool with parent context for proper cancellation propagation
    pool := NewExecutionPool(name, maxWorkers, pm.parentCtx)  // ✅ USE PARENT CONTEXT
    pool.Start()
    pm.matrixPools[name] = pool
    
    return pool
}
```

#### 5. Recreate PoolManager with Execution Context

**File**: `pkg/buildfab/buildfab.go`

```go
func (r *Runner) runStageInternal(ctx context.Context, stageName string) error {
    // ... other code ...
    
    // Recreate pool manager with execution context for proper signal handling
    maxParallel := r.opts.MaxParallel
    if maxParallel <= 0 && r.config.Project.MaxParallel > 0 {
        maxParallel = r.config.Project.MaxParallel
    }
    if maxParallel <= 0 {
        maxParallel = GetDefaultMaxParallel()
    }
    r.poolManager = NewPoolManager(maxParallel, ctx)  // ✅ PASS EXECUTION CONTEXT
    
    // ... rest of execution ...
}
```

## How It Works Now

### Context Propagation Chain

```
Signal Handler Context (from main.go)
    ↓
Runner.runStageInternal(ctx) 
    ↓
PoolManager(ctx) 
    ↓
ExecutionPool(ctx)
    ↓
Task.Execute(poolCtx) where poolCtx derives from ctx
    ↓
exec.CommandContext(taskCtx, ...) 
    ↓
Running Process (receives cancellation)
```

### When Ctrl+C is Pressed

1. ✅ Signal handler context gets cancelled
2. ✅ Pool context (derived from signal handler context) gets cancelled
3. ✅ Pool workers detect `<-p.ctx.Done()` and stop accepting new tasks
4. ✅ Running tasks receive cancelled context via `task.Execute(p.ctx)`
5. ✅ Commands created with `exec.CommandContext(ctx, ...)` receive cancellation
6. ✅ Running processes are killed via `cmd.Process.Kill()`
7. ✅ Execution terminates properly with "TERMINATED" status

## Testing

### Manual Testing

1. **Create a long-running test**:
   ```yaml
   actions:
     - name: long-action
       run: |
         echo "Starting long task..."
         sleep 30
         echo "Finished"
   
   stages:
     test:
       - action: long-action
   ```

2. **Run and interrupt**:
   ```bash
   ./bin/buildfab run test
   # Press Ctrl+C during sleep
   ```

3. **Expected behavior**:
   - Job terminates immediately
   - Shows "⏹️ TERMINATED" status
   - No hanging processes

### Automated Testing

Run the context cancellation test:
```bash
go test ./pkg/buildfab -run TestExecutionPool_ContextCancellation -v
```

This test verifies that:
- Pool respects context cancellation
- Tasks are not executed when context is already cancelled
- Running tasks are interrupted properly

## Impact

### Fixed Issues

- ✅ Ctrl+C now properly terminates all running jobs
- ✅ INT and TERM signals are handled correctly
- ✅ No hanging processes after signal
- ✅ Proper "TERMINATED" status display
- ✅ Clean resource cleanup

### Backward Compatibility

- ✅ All existing tests pass
- ✅ API changes are internal (pool creation)
- ✅ No breaking changes to public API
- ✅ Existing configurations work without modification

## Additional Fix: Container Execution

### Container-Specific Issue

Even after fixing the pool context propagation, container commands were still not responding to Ctrl+C. Investigation revealed that `RunContainerWithCallback` methods in both Docker and Podman engines were using `cmd.Wait()` without monitoring `ctx.Done()`.

**File**: `pkg/buildfab/container/engines.go`

**OLD CODE** (lines 584, 1189):
```go
// Wait for command to complete
err = cmd.Wait()
```

This blocked until the container command completed, ignoring context cancellation.

**NEW CODE**:
```go
// Wait for command to complete or context cancellation
cmdDone := make(chan error, 1)
go func() {
    cmdDone <- cmd.Wait()
}()

var cmdErr error
select {
case cmdErr = <-cmdDone:
    // Command completed normally
case <-ctx.Done():
    // Context cancelled - kill the process
    if cmd.Process != nil {
        cmd.Process.Kill()
    }
    cmdErr = ctx.Err()
}
```

This properly monitors both command completion and context cancellation, killing the container process immediately when Ctrl+C is pressed.

### Changes Made

1. **Docker Engine** (`dockerEngineImpl.RunContainerWithCallback`)
   - Added context cancellation monitoring with select statement
   - Three-step termination: SIGTERM → 100ms grace period → SIGKILL
   - Ensures process group termination for nested processes

2. **Podman Engine** (`podmanEngineImpl.RunContainerWithCallback`)
   - Added context cancellation monitoring with select statement
   - Three-step termination: SIGTERM → 100ms grace period → SIGKILL
   - Ensures process group termination for nested processes

3. **DAG Executor** (all three executor variants)
   - Added context monitoring when waiting for completion
   - Calls `poolManager.StopAll()` immediately on cancellation
   - Returns with partial results instead of hanging forever

## Related Files

- `pkg/buildfab/pool.go` - Pool implementation with context propagation
- `pkg/buildfab/buildfab.go` - Runner integration with context passing
- `pkg/buildfab/container/engines.go` - Container engine context cancellation handling (**NEW**)
- `pkg/buildfab/pool_phase1_test.go` - Updated tests with context parameter
- `cmd/buildfab/main.go` - Signal handler setup (unchanged)

## Version

This fix is included in version 0.19.1 and later.

