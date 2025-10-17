# Test Buildfab API with Containers

This example demonstrates the container issue with buildfab's SimpleRunner API and provides the solution.

## Problem

When using `run_action:` in container configurations with SimpleRunner API (as used by tools like pre-push), the buildfab binary is not automatically copied into the container, causing the action to fail with error: **`sh: buildfab: not found`**

## Files

- `main.go` - Test application using SimpleRunner API (similar to pre-push)
- `test-run-action.yml` - Configuration with PROBLEMATIC `run_action:` (will hang)
- `test-run-command.yml` - Configuration with WORKING `run:` command (works correctly)

## Building the Test Application

```bash
# Build the test application
cd examples/test-buildfab-api
go build -o test-api main.go
```

## Testing the Issue

### Test 1: Problematic run_action (will fail at all verbose levels)

```bash
# Verbose level 0 - fast failure with generic error
./test-api test-run-action.yml test 0
# Output: ✗ container-platform-view-run-action container exited with code 1

# Verbose level 1 - fast failure with detailed error
./test-api test-run-action.yml test 1
# Output: sh: buildfab: not found
#         ✗ container-platform-view-run-action container exited with code 1
```

### Test 2: Working run: command

```bash
# Verbose level 0 - works silently
./test-api test-run-command.yml test 0
# Output: ✓ container-platform-view-run-command executed successfully

# Verbose level 1 - works with detailed output
./test-api test-run-command.yml test 1
# Output: === Container Platform Detection Test ===
#         Platform: Linux
#         Architecture: x86_64
#         ✓ container-platform-view-run-command executed successfully
```

## Root Cause

The issue occurs because:

1. **buildfab CLI** has special logic to prepare containers:
   - Copies buildfab binary into the container
   - Sets up execution environment
   - Handles stdin/tty configuration

2. **SimpleRunner API** doesn't expose this container preparation functionality:
   - When `run_action:` tries to execute `buildfab action <name>` inside container
   - Container doesn't have buildfab binary (no preparation was done)
   - Command fails or hangs trying to execute non-existent command

## Solution

**Use `run:` with shell commands instead of `run_action:`**

### Before (doesn't work with SimpleRunner):
```yaml
- name: container-platform-view
  container:
    image:
      from: alpine:latest
    run_action: platform-view  # ✗ Fails - needs buildfab binary
```

### After (works with SimpleRunner):
```yaml
- name: container-platform-view
  container:
    image:
      from: alpine:latest
    run: |
      echo "=== Platform Detection Test ==="
      echo "Platform: $(uname -s)"
      echo "Architecture: $(uname -m)"
      echo "================================"
```

## Verification

After applying the fix to your configuration:

1. Update container actions to use `run:` instead of `run_action:`
2. Test with the buildfab CLI: `buildfab run stage-name`
3. Test with SimpleRunner-based tools: `pre-push test`

Both should work correctly with the `run:` approach.

