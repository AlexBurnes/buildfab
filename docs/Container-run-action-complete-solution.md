# Complete Solution: run_action/run_stage Support in SimpleRunner API

## Status: ✅ **FULLY IMPLEMENTED**

Container actions with `run_action:` and `run_stage:` now work with SimpleRunner API!

## Solution Overview

Added `BuildfabBinaryPath` option to SimpleRunOptions that allows:
1. **Automatic detection**: Uses current executable by default (works for pre-push, buildfab CLI)
2. **Explicit path**: Specify exact buildfab binary to use when needed
3. **Smart search**: Falls back to PATH and common locations

## Requirements

**Critical**: The binary MUST be statically linked to work in Alpine containers:

```bash
# Build statically-linked binary
VERSION=$(cat VERSION)
CGO_ENABLED=0 go build -ldflags "-X main.appVersion=${VERSION} -s -w -extldflags '-static'" -o bin/buildfab ./cmd/buildfab
```

## Implementation Details

### Added Fields

```go
// SimpleRunOptions
type SimpleRunOptions struct {
    // ... existing fields ...
    BuildfabBinaryPath string  // Path to buildfab binary (optional, auto-detected)
}

// RunOptions  
type RunOptions struct {
    // ... existing fields ...
    BuildfabBinaryPath string  // Path to buildfab binary (optional, auto-detected)
}

// ContainerRunner
type ContainerRunner struct {
    // ... existing fields ...
    buildfabPath string  // Path to buildfab-compatible binary
}
```

### Binary Detection Logic

Search order in `getBuildfabBinaryPath()`:

1. **Explicit path** (if `BuildfabBinaryPath` set) - Highest priority
2. **Current executable** (default) - Works for pre-push, buildfab CLI, any app with CLI interface
3. **System PATH** (`exec.LookPath("buildfab")`) - For installed buildfab
4. **Common locations** (`./bin/buildfab`, `./buildfab`, `/usr/local/bin/buildfab`) - Development/system installs

### Container Command Generation

The container uses the **actual binary name**, not hardcoded "buildfab":

```bash
# If using buildfab:
export PATH=${PATH}:/tmp/buildfab-bin && buildfab -c /tmp/buildfab-workspace/.project.yml action my-action

# If using pre-push:
export PATH=${PATH}:/tmp/buildfab-bin && pre-push -c /tmp/buildfab-workspace/.project.yml action my-action

# If using test-api with explicit buildfab:
export PATH=${PATH}:/tmp/buildfab-bin && buildfab -c /tmp/buildfab-workspace/.project.yml action my-action
```

## Usage Examples

### Example 1: Pre-push (Automatic - No Configuration Needed)

Pre-push implements the full buildfab CLI interface, so it works automatically:

```go
// In pre-push code
opts := &buildfab.SimpleRunOptions{
    ConfigPath:   ".project.yml",
    VerboseLevel: 1,
    // BuildfabBinaryPath NOT needed - uses current executable (pre-push)
}

runner := buildfab.NewSimpleRunner(cfg, opts)
runner.RunStage(ctx, "pre-push")  // Works with run_action in containers!
```

### Example 2: Custom App with Explicit Path

For simple test apps that don't implement CLI commands:

```go
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "./bin/buildfab",  // Explicit path to buildfab
    VerboseLevel:       1,
}

runner := buildfab.NewSimpleRunner(cfg, opts)
runner.RunStage(ctx, "test")  // Works with run_action in containers!
```

### Example 3: Using buildfab from PATH

If buildfab is installed system-wide:

```go
opts := &buildfab.SimpleRunOptions{
    ConfigPath:   ".project.yml",
    VerboseLevel: 1,
    // BuildfabBinaryPath NOT needed - auto-detected from PATH
}
```

## Configuration Examples

### YAML with run_action (Now Works!)

```yaml
actions:
  - name: platform-view
    run: |
      echo "Platform: $(uname -s)"

  - name: container-test
    container:
      image:
        from: alpine:latest
      run_action: platform-view  # ✅ NOW WORKS with SimpleRunner!
```

### YAML with run_stage (Now Works!)

```yaml
stages:
  local-test:
    steps:
      - action: test-unit
      - action: test-integration

actions:
  - name: container-full-test
    container:
      image:
        from: alpine:latest
      run_stage: local-test  # ✅ NOW WORKS with SimpleRunner!
```

## Test Results

| Verbosity | `run:` | `run_action:` (with path) | `run_stage:` (with path) |
|-----------|--------|---------------------------|--------------------------|
| 0 (quiet) | ✅ Works | ✅ **Works** | ✅ **Works** |
| 1 (-v) | ✅ Works | ✅ **Works** | ✅ **Works** |
| 2 (-vv) | ✅ Works | ✅ **Works** | ✅ **Works** |
| 3 (-vvv) | ✅ Works | ✅ **Works** | ✅ **Works** |

## Error Messages

If binary not found, users get helpful error message:

```
Error: buildfab-compatible binary not found for run_action/run_stage

The container needs a buildfab-compatible binary to execute run_action and run_stage.

Requirements:
- Binary MUST be statically linked (CGO_ENABLED=0) to work in Alpine containers
- Binary can be buildfab CLI or any app that embeds buildfab library (pre-push, etc.)

Solutions:
1. The current executable should work automatically (if statically linked)

2. Specify explicit path to a statically-linked binary:
   opts.BuildfabBinaryPath = "./bin/buildfab"

3. Build static binary:
   CGO_ENABLED=0 go build -ldflags "-extldflags '-static'" -o bin/buildfab ./cmd/buildfab

4. Use 'run:' instead of 'run_action:' (doesn't need binary):
   container:
     run: |
       echo "Direct commands work without any binary"

Searched locations: PATH, ./bin/buildfab, ./buildfab, /usr/local/bin/buildfab, /usr/bin/buildfab
```

## Files Modified

- `pkg/buildfab/simple.go` - Added BuildfabBinaryPath to SimpleRunOptions, pass to RunOptions
- `pkg/buildfab/buildfab.go` - Added BuildfabBinaryPath to RunOptions, pass to ContainerRunner
- `pkg/buildfab/ordered_output.go` - Added buildfabBinaryPath field, SetBuildfabBinaryPath method, pass to tempRunner
- `internal/container/runner.go` - Added buildfabPath field, SetBuildfabPath method, enhanced getBuildfabBinaryPath with smart detection, use actual binary name in commands

## Backward Compatibility

✅ Fully backward compatible:
- `BuildfabBinaryPath` is optional (empty string by default)
- Automatic detection works for most cases
- Existing code continues to work without changes

## Production Ready

This solution enables:
- ✅ pre-push to use `run_action`/`run_stage` without configuration
- ✅ Custom apps to specify buildfab path explicitly
- ✅ Automatic detection from PATH
- ✅ Clear error messages when binary not found
- ✅ Static linking requirement enforced
- ✅ Works at all verbosity levels (0-3)

## References

- Design Document: `docs/Container-run-action-solution-design.md`
- Issue Analysis: `docs/Container-simplerunner-issue.md`
- Test Application: `examples/test-buildfab-api/`

