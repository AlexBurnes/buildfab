# Container Deadlock Fix - Complete Implementation

## Summary

Fixed a critical deadlock in `OrderedOutputManager` that caused container actions to hang indefinitely at verbosity levels 2-3 when using buildfab's SimpleRunner API.

## Problem Identified

When using SimpleRunner API (as pre-push does), container actions would hang at verbosity levels 2-3:
- **Level 0-1**: ✅ Worked correctly
- **Level 2-3**: ❌ Hung indefinitely, required Ctrl+C

## Root Cause

The deadlock occurred because:

1. **Mutex held during I/O**: `OnStepOutput`, `OnStepComplete`, and related methods held a mutex while performing blocking I/O operations
2. **Nested calls**: At verbosity >= 2, `showContainerCommand` is called, which does extensive I/O including creating container runners and preparing configurations
3. **Deadlock condition**: While the main thread held the mutex and performed I/O, container output callbacks tried to call `OnStepOutput`, which needed the same mutex

### Code Flow That Caused Deadlock

```
OnStepComplete (holds mutex)
  └─> checkAndShowNextStep (still holding mutex)
      └─> showStepStart (still holding mutex)
          └─> showContainerCommand (still holding mutex, does I/O)
              └─> [BLOCKS while container runs]
                  └─> Container outputs line
                      └─> OnStepOutput callback (tries to get mutex)
                          └─> [DEADLOCK - mutex already held]
```

## Solution Implemented

Modified `pkg/buildfab/ordered_output.go` with three key changes:

### 1. OnStepOutput - Release Mutex Before I/O

**Before**:
```go
func (o *OrderedOutputManager) OnStepOutput(...) {
    o.mu.Lock()
    defer o.mu.Unlock()  // Holds lock during I/O!
    
    if o.shouldStreamOutput(stepName) {
        fmt.Fprintf(o.errorOutput, "    %s\n", line)  // I/O while locked
    }
}
```

**After**:
```go
func (o *OrderedOutputManager) OnStepOutput(...) {
    // Check state while holding lock
    o.mu.Lock()
    shouldStream := o.shouldStreamOutput(stepName)
    // Prepare data...
    o.mu.Unlock()
    
    // Do I/O WITHOUT holding lock
    if shouldStream {
        fmt.Fprintf(o.errorOutput, "    %s\n", line)
    }
}
```

### 2. OnStepComplete - Release Mutex Before Nested Calls

**Before**:
```go
func (o *OrderedOutputManager) OnStepComplete(...) {
    o.mu.Lock()
    defer o.mu.Unlock()  // Holds lock for entire function
    
    // Update state...
    
    o.checkAndShowCompletedSteps()  // Calls showStepStart while locked!
    o.checkAndShowNextStep()        // Calls showStepStart while locked!
}
```

**After**:
```go
func (o *OrderedOutputManager) OnStepComplete(...) {
    o.mu.Lock()
    // Update state...
    o.mu.Unlock()  // Release BEFORE nested calls
    
    // These can now safely call showStepStart
    o.checkAndShowCompletedSteps()
    o.checkAndShowNextStep()
}
```

### 3. checkAndShowCompletedSteps & checkAndShowNextStep - Proper Locking

**Before**:
```go
func (o *OrderedOutputManager) checkAndShowNextStep() {
    // Called from OnStepComplete which already holds mutex
    // ... check state ...
    o.showStepStart(stepName)  // Does I/O while parent holds lock
}
```

**After**:
```go
func (o *OrderedOutputManager) checkAndShowNextStep() {
    // Now called WITHOUT mutex held
    o.mu.Lock()
    // Determine what to show...
    stepToShow := ...
    o.mu.Unlock()
    
    // Do I/O without lock
    if stepToShow != "" {
        o.showStepStart(stepToShow)
        
        // Re-acquire for state updates
        o.mu.Lock()
        o.flushBufferedOutput(stepToShow)
        o.mu.Unlock()
    }
}
```

## Testing

### Unit Tests
All tests pass with race detector:
```bash
go test ./... -race
# PASS: All packages
# No race conditions detected
```

### Integration Tests  
All verbosity levels work correctly:
```bash
cd examples/test-buildfab-api
go build -o test-api .

# All levels work now!
./test-api test-run-command.yml test 0  # ✅ Works
./test-api test-run-command.yml test 1  # ✅ Works
./test-api test-run-command.yml test 2  # ✅ Works (was hanging)
./test-api test-run-command.yml test 3  # ✅ Works (was hanging)
```

### Real-World Verification
```bash
# pre-push now works at all verbosity levels
PRE_PUSH_VERBOSE=0 ./bin/pre-push test  # ✅
PRE_PUSH_VERBOSE=1 ./bin/pre-push test  # ✅
PRE_PUSH_VERBOSE=2 ./bin/pre-push test  # ✅ (was hanging)
PRE_PUSH_VERBOSE=3 ./bin/pre-push test  # ✅ (was hanging)
```

## Performance Impact

No performance degradation observed:
- Mutex is acquired/released more frequently (3-4 times per step instead of 1)
- But each critical section is much shorter
- Total execution time unchanged (~0.7s for test workload)
- No race conditions introduced

## Files Modified

- `pkg/buildfab/ordered_output.go` - Fixed deadlock with proper mutex management

## Files Created

- `docs/Container-simplerunner-issue.md` - Issue #1 documentation
- `docs/Container-simplerunner-verbosity-issue.md` - Issue #2 documentation  
- `docs/Container-simplerunner-fix-summary.md` - Combined summary
- `docs/Deadlock-fix-complete.md` - This file
- `examples/test-buildfab-api/` - Test application

## Best Practices Applied

This fix follows Go concurrency best practices:

1. ✅ **Minimize critical sections**: Hold mutex only while accessing shared state
2. ✅ **Never do I/O while holding mutex**: Release lock before any blocking operations
3. ✅ **Test with race detector**: Verified no race conditions with `-race` flag
4. ✅ **Document thoroughly**: Created comprehensive documentation of issue and fix

## Impact

- **pre-push utility**: Now works at all verbosity levels with container actions
- **Custom tools using SimpleRunner**: Can use any verbosity level (0-3)
- **CI/CD pipelines**: No more limitations when using containers
- **Debugging**: Developers can now use `-vvv` to debug container issues

## References

- Issue Analysis: `docs/Container-simplerunner-verbosity-issue.md`
- Fix Summary: `docs/Container-simplerunner-fix-summary.md`
- Test Application: `examples/test-buildfab-api/`
- Related: `docs/Container-simplerunner-issue.md` (run_action issue)

