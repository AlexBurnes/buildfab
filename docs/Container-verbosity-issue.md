# Container Verbosity Issue

## Problem Description

Container actions do not respect verbosity flags (`-v`, `-vv`, `-vvv`) when executing. The container output is not displayed even when verbose flags are used.

## Current Behavior

```bash
$ ./bin/buildfab action hello-container -c examples/container-simple-test.yml -vvv
buildfab unknown
Project container-simple-test (v0.16.11)
▶️  Running action: hello-container

  💻 hello-container
  ✓ hello-container executed successfully - in '0.297s'
```

**Expected Behavior:**
- Container output should be displayed when using verbose flags
- Container execution details should be shown with `-vv` and `-vvv`
- Container command execution should be visible

## Root Cause Analysis

The issue is in the container execution flow:

1. **Container Runner** (`internal/container/runner.go`) executes the container
2. **Container Manager** (`pkg/buildfab/container/manager.go`) calls the engine
3. **Engine Implementation** (`pkg/buildfab/container/engines.go`) runs the container
4. **Output Handling** - Container output is captured but not passed back to the main buildfab output system

## Technical Details

### Current Flow
```
buildfab action -> runActionInternal -> runContainerAction -> ContainerRunner.RunAction -> Manager.ExecuteAction -> Engine.RunContainer
```

### Output Handling
- Container output is captured in `ContainerResult.Output`
- Output is not passed back to buildfab's output manager
- Verbosity flags are not propagated to container execution

## Files Affected

- `internal/container/runner.go` - Container execution wrapper
- `pkg/buildfab/container/manager.go` - Container management
- `pkg/buildfab/container/engines.go` - Engine implementations
- `pkg/buildfab/buildfab.go` - Main action execution

## Proposed Solution

### Phase 1: Basic Output Display
1. **Update ContainerRunner** to return container output
2. **Modify runContainerAction** to display container output
3. **Add verbosity support** to container execution

### Phase 2: Enhanced Verbosity
1. **Add verbosity levels** to container execution
2. **Display container command** being executed
3. **Show container engine** being used
4. **Display container configuration** details

## Implementation Steps

### Step 1: Update ContainerRunner
```go
func (r *ContainerRunner) RunAction(ctx context.Context, config container.ContainerConfig) (*container.ContainerResult, error) {
    // Execute container and return result
    result, err := r.manager.ExecuteAction(ctx, config)
    if err != nil {
        return nil, err
    }
    return result, nil
}
```

### Step 2: Update runContainerAction
```go
func (r *Runner) runContainerAction(ctx context.Context, action Action) error {
    runner, err := containerRunner.NewContainerRunner()
    if err != nil {
        return fmt.Errorf("failed to create container runner: %w", err)
    }

    result, err := runner.RunAction(ctx, *action.Container)
    if err != nil {
        return fmt.Errorf("container action failed: %w", err)
    }

    // Display container output based on verbosity
    if result.Output != "" {
        fmt.Println(result.Output)
    }

    return nil
}
```

### Step 3: Add Verbosity Support
- Pass verbosity level to container execution
- Display container command with `-vv`
- Show container engine with `-vvv`
- Display container configuration with `-vvv`

## Testing

### Test Cases
1. **Basic Output**: Container output should be displayed
2. **Verbosity Levels**: Different verbosity levels should show different details
3. **Error Handling**: Container errors should be properly displayed
4. **Multiple Commands**: Multiple container commands should be shown

### Test Commands
```bash
# Basic output
./bin/buildfab action hello-container -c examples/container-simple-test.yml

# Verbose output
./bin/buildfab action hello-container -c examples/container-simple-test.yml -v

# Very verbose output
./bin/buildfab action hello-container -c examples/container-simple-test.yml -vvv
```

## Priority

**High** - This affects the basic usability of the container feature. Users expect to see container output when using verbose flags.

## Related Issues

- Container output not displayed
- Verbosity flags not respected
- Container execution details not shown
- Debugging container actions difficult

## Status

**Open** - Issue identified and documented, awaiting implementation.

## Assignee

TBD - To be assigned during Phase 2 implementation.

## Labels

- `bug`
- `container`
- `verbosity`
- `output`
- `phase2`
