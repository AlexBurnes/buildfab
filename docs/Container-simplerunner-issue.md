# Container Issues with SimpleRunner API

## Status: ✅ **ALL ISSUES FIXED**

Both container issues with SimpleRunner API have been completely resolved:

1. ✅ **`run_action:` Issue**: FIXED - BuildfabBinaryPath option added
2. ✅ **Verbosity Level 2-3 Hang Issue**: FIXED - Deadlock resolved (see `Container-simplerunner-verbosity-issue.md`)

## Overview

There were **TWO DISTINCT ISSUES** when using containers with SimpleRunner API:

1. **`run_action:` Issue**: Container actions failed because buildfab binary was not available in containers
2. **Verbosity Level 2-3 Hang Issue**: ALL container actions would hang at verbosity levels 2-3

## Issue 1: run_action: Fails (Missing buildfab Binary)

When using `run_action:` in container configurations with SimpleRunner API (as used by tools like `pre-push`), the buildfab binary is not available inside the container.

##Root Cause

The issue occurs because:

1. **buildfab CLI** has special container preparation logic:
   - Automatically copies the buildfab binary into the container
   - Sets up the execution environment
   - Configures stdin/tty handling properly

2. **SimpleRunner API** doesn't expose this container preparation functionality:
   - When `run_action: action-name` tries to execute `buildfab action action-name` inside the container
   - Container doesn't have the buildfab binary (no preparation was done)
   - Command fails with `sh: buildfab: not found`

## Behavior by Verbosity Level (for run_action:)

### Verbose Level 0 (Quiet)
```bash
PRE_PUSH_VERBOSE=0 ./bin/pre-push test
```
- **Result**: Fast failure with generic error
- **Output**: `✗ container-platform-view container exited with code 1`
- **No details about root cause**

### Verbose Level 1 (Debug)
```bash
PRE_PUSH_VERBOSE=1 ./bin/pre-push test
```
- **Result**: Fast failure with detailed error
- **Output**: `sh: buildfab: not found`
- **Shows the actual problem**

### Verbose Level 2-3
✅ **Fixed**: The hang issue at verbosity levels 2-3 has been resolved. See `Container-simplerunner-verbosity-issue.md` for details.

### Direct buildfab CLI
```bash
buildfab -vvv pre-push
```
- **Result**: Works correctly at all verbose levels
- **Reason**: CLI does container preparation automatically

## Solution (for run_action: Issue)

### Option 1: Use BuildfabBinaryPath (Recommended - Full Feature Support)

Specify the path to buildfab binary in SimpleRunOptions:

```go
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "./bin/buildfab",  // Path to statically-linked buildfab
    VerboseLevel:       1,
}
```

**Requirements:**
- Buildfab binary MUST be statically linked (`CGO_ENABLED=0`)
- Use explicit path or ensure buildfab is in PATH

**Build static binary:**
```bash
VERSION=$(cat VERSION)
CGO_ENABLED=0 go build -ldflags "-X main.appVersion=${VERSION} -s -w -extldflags '-static'" -o bin/buildfab ./cmd/buildfab
```

### Option 2: Use `run:` with shell commands (Simpler - No buildfab needed)

Use direct shell commands instead of `run_action:`:

```yaml
container:
  image:
    from: alpine:latest
  run: |
    echo "Direct shell commands work without buildfab binary"
```

This approach works because it doesn't depend on the buildfab binary being present in the container.

✅ **Note**: Both approaches now work at all verbosity levels (0-3). The verbosity hang issue has been fixed (see `Container-simplerunner-verbosity-issue.md`).

### Before (Doesn't Work with SimpleRunner)

```yaml
actions:
  - name: platform-view
    description: Display platform information
    run: |
      echo "Platform: $(uname -s)"
      echo "Architecture: $(uname -m)"

  - name: container-platform-view
    description: Test platform detection in container
    container:
      image:
        from: alpine:latest
      run_action: platform-view  # ✗ Fails - needs buildfab binary
```

### After (Works with SimpleRunner)

```yaml
actions:
  - name: platform-view
    description: Display platform information
    run: |
      echo "Platform: $(uname -s)"
      echo "Architecture: $(uname -m)"

  - name: container-platform-view
    description: Test platform detection in container
    container:
      image:
        from: alpine:latest
      run: |
        # ✓ Works - direct shell commands
        echo "Platform: $(uname -s)"
        echo "Architecture: $(uname -m)"
```

## Testing

A test application is provided in `examples/test-buildfab-api/` to demonstrate the issue and solution.

### Build Test Application

```bash
cd examples/test-buildfab-api
go build -o test-api .
```

### Test Problematic Configuration (run_action)

```bash
# Verbose level 0 - fast failure with generic error
./test-api test-run-action.yml test 0

# Verbose level 1 - fast failure with detailed error
./test-api test-run-action.yml test 1
# Output: sh: buildfab: not found
```

### Test Working Configuration (run)

```bash
# Verbose level 0 - works
./test-api test-run-command.yml test 0

# Verbose level 1 - works with output
./test-api test-run-command.yml test 1
```

## Files Affected

The following files use `run_action:` in container configurations and may need updates:

1. **examples/container-matrix-platform-test.yml**
   - Line 61: `run_action: platform-view`
   
2. **examples/container-run-test.yml**
   - Line 38: `run_action: hello-action`

3. **tests/test-container-artifacts-run-action.yml**
   - Line 21: `run_action: create-artifact-file`

## Recommended Actions

1. **Update existing configurations** to use `run:` instead of `run_action:` for container actions
2. **Document the limitation** in library API documentation
3. **Consider enhancing SimpleRunner** to support container preparation (future improvement)

## Workarounds (if run: doesn't work)

### Option 1: Use buildfab CLI directly
```bash
# Instead of using SimpleRunner-based tools
buildfab pre-push
```

### Option 2: Pre-build container with buildfab
```yaml
container:
  image:
    build:
      dockerfile: Dockerfile
      context: .
  run_action: my-action  # Works if Dockerfile includes buildfab binary
```

## Related Issues

- Container stdin/tty handling in SimpleRunner API
- Container preparation API not exposed to library users
- Verbose mode differences between CLI and SimpleRunner

## References

- Test application: `examples/test-buildfab-api/`
- Library documentation: `docs/Library.md`
- Container feature: `docs/Container-feature-implementation-steps.md`

