# Container Stdin Fix - Issue Resolution

## Problem Summary
Container actions were hanging indefinitely when executed via SimpleRunner (used by pre-push utility). The containers were waiting for stdin input that was never provided, causing the execution to hang.

## Root Cause
When containers were started using `exec.CommandContext`, the stdin was implicitly connected by default. When buildfab was called programmatically via SimpleRunner (as pre-push does), stdin was not properly handled, causing containers to wait indefinitely for input.

## Solution Implemented
Added stdin control throughout the execution chain:

### 1. Added Input Field to Options Structs
- **SimpleRunOptions**: Added `Input io.Reader` field to control stdin for simple API
- **RunOptions**: Added `Input io.Reader` field to pass stdin through full execution pipeline

### 2. Updated Container Execution Chain
- **ContainerRunner**: Added `Stdin io.Reader` field to container runner
- **Manager**: Updated `ExecuteActionWithCallback` to accept stdin parameter
- **Engine Interface**: Updated `RunContainerWithCallback` to accept stdin parameter

### 3. Updated Engine Implementations
- **DockerEngine**: Set `cmd.Stdin = stdin` in `RunContainerWithCallback`
- **PodmanEngine**: Set `cmd.Stdin = stdin` in `RunContainerWithCallback`

### 4. Default Behavior
When `Input` is `nil` (default for programmatic usage), containers run in non-interactive mode without waiting for stdin. This is the correct behavior for automated execution.

## Files Modified
1. `pkg/buildfab/simple.go` - Added Input field to SimpleRunOptions
2. `pkg/buildfab/buildfab.go` - Added Input field to RunOptions
3. `internal/container/runner.go` - Added Stdin field and passed it to manager
4. `pkg/buildfab/container/manager.go` - Updated to accept and pass stdin parameter
5. `pkg/buildfab/container/engine.go` - Updated Engine interface
6. `pkg/buildfab/container/engines.go` - Updated docker and podman implementations
7. `internal/container/docker.go` - Removed unused io import

## Usage Examples

### Programmatic Usage (Non-Interactive)
```go
// Default behavior - no stdin (non-interactive)
opts := buildfab.DefaultSimpleRunOptions()
// opts.Input is nil by default - containers won't wait for stdin

runner := buildfab.NewSimpleRunner(config, opts)
err := runner.RunStage(ctx, "pre-push")
// Container actions execute without hanging
```

### Custom Stdin (If Needed)
```go
// Provide custom stdin if interactive mode is desired
opts := buildfab.DefaultSimpleRunOptions()
opts.Input = os.Stdin // or any other io.Reader

runner := buildfab.NewSimpleRunner(config, opts)
err := runner.RunStage(ctx, "interactive-stage")
```

## Testing
The fix was validated by:
1. Building the binary successfully
2. Running all unit tests - all passed
3. Running container integration tests - all passed without hanging
4. Testing with actual container actions in pre-push stage

## Impact
- **Before Fix**: Container actions hang indefinitely when called via SimpleRunner
- **After Fix**: Container actions complete normally in non-interactive mode
- **Backward Compatibility**: No breaking changes - default behavior is sensible (no stdin)
- **API Enhancement**: Users can now control stdin if needed

## Related Issue
This fix resolves the issue described in "Buildfab Container Hang Issue Report" where `pre-push test` would hang waiting for stdin when container actions were present.

