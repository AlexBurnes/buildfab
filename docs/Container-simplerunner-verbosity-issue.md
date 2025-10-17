# Container Hang Issue at Verbosity Levels 2-3 with SimpleRunner API

## Status: ✅ **FIXED**

This issue has been resolved in the latest version. The deadlock/race condition in `OrderedOutputManager` has been fixed.

## Problem Summary (Historical)

Previously, when using buildfab's `SimpleRunner` API with container actions, containers would hang indefinitely at verbosity levels 2 and 3, regardless of whether `run:` or `run_action:` was used.

## Test Results

### After Fix
| Verbosity | `run:` (shell) | `run_action:` (buildfab) |
|-----------|----------------|--------------------------|
| 0 (quiet) | ✅ **Works** | ❌ Fails (missing buildfab binary - see Container-simplerunner-issue.md) |
| 1 (-v) | ✅ **Works** | ❌ Fails (missing buildfab binary - see Container-simplerunner-issue.md) |
| 2 (-vv) | ✅ **Works** | ❌ Fails (missing buildfab binary - see Container-simplerunner-issue.md) |
| 3 (-vvv) | ✅ **Works** | ❌ Fails (missing buildfab binary - see Container-simplerunner-issue.md) |

### Before Fix (Historical)
| Verbosity | `run:` (shell) | `run_action:` (buildfab) |
|-----------|----------------|--------------------------|
| 0 (quiet) | ✓ Works | ✗ Fails (missing buildfab binary) |
| 1 (-v) | ✓ Works | ✗ Fails (missing buildfab binary) |
| 2 (-vv) | ✗ **HANGS** | ✗ **HANGS** |
| 3 (-vvv) | ✗ **HANGS** | ✗ **HANGS** |

## Symptoms

At verbosity levels 2-3:
1. The container action starts
2. Step name is displayed: `💻 container-action-name`
3. **Container hangs indefinitely** - no output, no completion
4. Requires Ctrl+C or timeout to stop

## Root Cause (Historical)

The hang occurred due to a deadlock in the `OrderedOutputManager`:

1. **At verbosity >= 1**: Uses `OrderedStepCallback` instead of `MultilineStepCallback`
2. **Mutex held during I/O**: The `OnStepOutput`, `OnStepComplete`, and related methods would hold a mutex while performing I/O operations
3. **Deadlock condition**: At verbosity levels 2-3, showing the container command (`showContainerCommand`) would be called while holding the mutex, and this function could then try to acquire the mutex again or block on I/O, causing a deadlock

### Code Location

In `pkg/buildfab/simple.go` (lines 173-190):
```go
if r.opts.VerboseLevel == 0 {
    stepCallback = NewMultilineStepCallbackWithActions(...)
} else {
    stepCallback = NewOrderedStepCallback(...)
}
```

In `pkg/buildfab/ordered_output.go` (lines 177-201):
```go
func (o *OrderedOutputManager) OnStepOutput(ctx context.Context, stepName string, output string) {
    if o.shouldStreamOutput(stepName) {
        // Stream output immediately
        ...
        return
    }
    // Buffer output for later display
    ...
}
```

## Fix Implemented

The fix involved three key changes to `pkg/buildfab/ordered_output.go`:

1. **`OnStepOutput`**: Release mutex before performing I/O operations
   - Determine streaming/buffering logic while holding the lock
   - Release the lock
   - Perform I/O operations without holding the lock

2. **`OnStepComplete`**: Release mutex before calling methods that do I/O
   - Update state while holding the lock
   - Release the lock before calling `checkAndShowCompletedSteps` and `checkAndShowNextStep`

3. **`checkAndShowCompletedSteps` and `checkAndShowNextStep`**: Acquire/release mutex properly
   - Acquire mutex to check state
   - Release mutex before calling `showStepStart` (which does I/O)
   - Re-acquire mutex as needed for state updates

This ensures that I/O operations are never performed while holding the mutex, preventing deadlocks.

## Verification

After the fix, all verbosity levels now work correctly:

```bash
cd examples/test-buildfab-api
go build -o test-api .

# Test verbose level 0 (works)
./test-api test-run-command.yml test 0

# Test verbose level 1 (works)
./test-api test-run-command.yml test 1

# Test verbose level 2 (NOW WORKS!)
./test-api test-run-command.yml test 2

# Test verbose level 3 (NOW WORKS!)
./test-api test-run-command.yml test 3
```

All tests complete successfully without hanging.

## Testing

A test application is provided in `examples/test-buildfab-api/` to reproduce the issue:

```bash
cd examples/test-buildfab-api
go build -o test-api .

# Test verbose level 0 (works)
./test-api test-run-command.yml test 0

# Test verbose level 1 (works)
./test-api test-run-command.yml test 1

# Test verbose level 2 (hangs - use timeout)
timeout 15 ./test-api test-run-command.yml test 2

# Test verbose level 3 (hangs - use timeout)
timeout 15 ./test-api test-run-command.yml test 3
```

## Impact

- **pre-push utility**: Cannot use verbosity levels 2-3 with container actions
- **Custom tools using SimpleRunner**: Must limit verbosity to 0-1 for container actions
- **CI/CD pipelines**: Must be aware of this limitation when using containers

## Related Issues

- Output buffering in `OrderedOutputManager`
- Container output streaming via callbacks
- Potential deadlock in `shouldStreamOutput` logic

## References

- Test application: `examples/test-buildfab-api/`
- Issue documentation: `docs/Container-simplerunner-issue.md` (for `run_action:` issue)
- Library documentation: `docs/Library.md`

