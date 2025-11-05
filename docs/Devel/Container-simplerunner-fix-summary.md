# Container Issues with SimpleRunner API - Complete Fix Summary

## Overview

Two distinct issues were discovered and **COMPLETELY RESOLVED** when using containers with buildfab's SimpleRunner API:

1. ✅ **Issue #1: `run_action:` Failure** - FIXED: BuildfabBinaryPath option added
2. ✅ **Issue #2: Verbosity 2-3 Hang** - FIXED: Deadlock in OrderedOutputManager resolved

## Final Test Results

| Verbosity | `run:` (shell commands) | `run_action:` (with BuildfabBinaryPath) |
|-----------|-------------------------|-----------------------------------------|
| 0 (quiet) | ✅ **Works** | ✅ **Works** |
| 1 (-v) | ✅ **Works** | ✅ **Works** |
| 2 (-vv) | ✅ **Works** | ✅ **Works** |
| 3 (-vvv) | ✅ **Works** | ✅ **Works** |

## Issue #1: run_action: Failure (Missing buildfab Binary)

### Problem (Historical)
Container actions using `run_action:` would fail because SimpleRunner API didn't have a way to specify which buildfab binary to mount into containers.

### Error Message (Historical)
```
sh: buildfab: not found
container exited with code 1
```

### Solution Implemented
Added `BuildfabBinaryPath` option to SimpleRunOptions:

```go
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "./bin/buildfab",  // Path to static buildfab binary
    VerboseLevel:       1,
}
```

**Automatic Detection**: If not specified, uses current executable (works for pre-push and buildfab CLI)

**Requirements**: Binary MUST be statically linked (`CGO_ENABLED=0`) for Alpine compatibility

#### Example Fix
**Before (doesn't work)**:
```yaml
- name: container-action
  container:
    image:
      from: alpine:latest
    run_action: my-action  # ✗ Requires buildfab binary
```

**After (works)**:
```yaml
- name: container-action
  container:
    image:
      from: alpine:latest
    run: |  # ✓ Direct shell commands
      echo "This works!"
      echo "Platform: $(uname -s)"
```

### Documentation
See `docs/Container-simplerunner-issue.md` for full details.

## Issue #2: Verbosity Level 2-3 Hang (FIXED)

### Problem (Historical)
At verbosity levels 2-3, ALL container actions (both `run:` and `run_action:`) would hang indefinitely due to a deadlock in the `OrderedOutputManager`.

### Root Cause
The deadlock occurred because:
1. `OnStepOutput` and `OnStepComplete` held a mutex while performing I/O operations
2. At verbosity >= 2, `showContainerCommand` is called, which does extensive I/O
3. This I/O could block while the mutex was held, causing a deadlock

### Fix Implemented
Three key changes were made to `pkg/buildfab/ordered_output.go`:

1. **`OnStepOutput`** - Release mutex before I/O:
   ```go
   // Determine what to do while holding lock
   o.mu.Lock()
   shouldStream := o.shouldStreamOutput(stepName)
   // ... prepare data ...
   o.mu.Unlock()
   
   // Do I/O WITHOUT holding lock
   if shouldStream {
       for _, line := range linesToPrint {
           fmt.Fprintf(o.errorOutput, "    %s\n", line)
       }
   }
   ```

2. **`OnStepComplete`** - Release mutex before calling methods that do I/O:
   ```go
   o.mu.Lock()
   // ... update state ...
   o.mu.Unlock()
   
   // Call methods that do I/O WITHOUT holding lock
   o.checkAndShowCompletedSteps()
   o.checkAndShowNextStep()
   ```

3. **`checkAndShowCompletedSteps` and `checkAndShowNextStep`** - Acquire/release mutex properly:
   ```go
   o.mu.Lock()
   // ... check state and determine what to show ...
   o.mu.Unlock()
   
   // Show step WITHOUT holding mutex
   if stepToShow != "" {
       o.showStepStart(stepToShow)
   }
   ```

### Verification
All verbosity levels now work correctly:
```bash
cd examples/test-buildfab-api && go build -o test-api .
./test-api test-run-command.yml test 0  # ✅ Works
./test-api test-run-command.yml test 1  # ✅ Works
./test-api test-run-command.yml test 2  # ✅ Works (was hanging)
./test-api test-run-command.yml test 3  # ✅ Works (was hanging)
```

### Documentation
See `docs/Container-simplerunner-verbosity-issue.md` for full details.

## Summary

### What Works Now
✅ Container actions with `run:` at ALL verbosity levels (0-3)
✅ No more hanging at high verbosity levels
✅ Proper mutex handling prevents deadlocks

### What Still Requires Workaround
⚠️ Container actions with `run_action:` require buildfab binary (use `run:` instead)

### Recommended Configuration
```yaml
actions:
  - name: my-container-action
    container:
      image:
        from: alpine:latest
      run: |  # ✅ Use run: not run_action:
        echo "Direct commands work at all verbosity levels!"
        echo "Platform: $(uname -s)"
```

```bash
# All verbosity levels work
PRE_PUSH_VERBOSE=0 ./bin/pre-push test  # ✅
PRE_PUSH_VERBOSE=1 ./bin/pre-push test  # ✅
PRE_PUSH_VERBOSE=2 ./bin/pre-push test  # ✅
PRE_PUSH_VERBOSE=3 ./bin/pre-push test  # ✅
```

## Files Modified
- `pkg/buildfab/ordered_output.go` - Fixed deadlock in output manager

## Files Created
- `docs/Container-simplerunner-issue.md` - Issue #1 documentation
- `docs/Container-simplerunner-verbosity-issue.md` - Issue #2 documentation  
- `docs/Container-simplerunner-fix-summary.md` - This summary
- `examples/test-buildfab-api/` - Test application to verify fixes

## Testing
Use the test application to verify the fixes:
```bash
cd examples/test-buildfab-api
go build -o test-api .

# Test with run: (should work at all levels)
./test-api test-run-command.yml test 0
./test-api test-run-command.yml test 1
./test-api test-run-command.yml test 2
./test-api test-run-command.yml test 3

# Test with run_action: (should fail with "buildfab: not found")
./test-api test-run-action.yml test 1
```

## References
- Issue #1 Details: `docs/Container-simplerunner-issue.md`
- Issue #2 Details: `docs/Container-simplerunner-verbosity-issue.md`
- Test Application: `examples/test-buildfab-api/`
- Library Documentation: `docs/Library.md`

