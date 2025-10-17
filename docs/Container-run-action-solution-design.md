# Solution Design: Supporting run_action/run_stage in SimpleRunner API

## Problem Analysis

When using SimpleRunner API from applications other than buildfab CLI (like `pre-push` or `test-api`), container actions with `run_action:` or `run_stage:` fail with "buildfab: not found".

### Root Cause

The `getBuildfabBinaryPath()` function in `internal/container/runner.go` uses `os.Executable()` to find the binary to mount:

```go
func (r *ContainerRunner) getCurrentExecutablePath() (string, error) {
    execPath, err := os.Executable()  // Returns path to CURRENT executable
    return resolvedPath, nil
}
```

**The Problem:**
- When running `buildfab` CLI → `os.Executable()` = `/path/to/buildfab` ✅
- When running `test-api` → `os.Executable()` = `/path/to/test-api` ❌
- When running `pre-push` → `os.Executable()` = `/path/to/pre-push` ❌

The container gets `test-api` or `pre-push` mounted, but tries to execute `buildfab` command!

## Solution Options

### Option 1: Add BuildfabBinaryPath to SimpleRunOptions (Recommended)

Allow users to explicitly specify the buildfab binary path:

```go
type SimpleRunOptions struct {
    ConfigPath       string
    BuildfabBinaryPath string  // Path to buildfab binary for run_action/run_stage
    // ... other fields
}
```

**Usage:**
```go
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "/path/to/buildfab",  // Explicit path
    // ... other options
}
```

**Pros:**
- Explicit and clear
- Allows using any buildfab binary (different versions, custom builds)
- Simple to implement
- Backward compatible (optional field)

**Cons:**
- Requires users to know where buildfab is installed
- Extra configuration needed

### Option 2: Smart Binary Detection (Fallback Chain)

Enhance `getBuildfabBinaryPath()` to search for buildfab even when current executable isn't buildfab:

```go
func (r *ContainerRunner) getBuildfabBinaryPath() (string, error) {
    // Try current executable first (for buildfab CLI)
    if currentBinaryPath, err := r.getCurrentExecutablePath(); err == nil {
        if isBuildfabBinary(currentBinaryPath) {  // Check if it's actually buildfab
            return currentBinaryPath, nil
        }
    }
    
    // Search for buildfab in standard locations
    searchPaths := []string{
        "./bin/buildfab",                    // Development
        "./buildfab",                        // Development
        exec.LookPath("buildfab"),          // System PATH
        "/usr/local/bin/buildfab",          // System install
        "/usr/bin/buildfab",                // System install
        "$GOPATH/bin/buildfab",             // Go install
    }
    
    for _, path := range searchPaths {
        if isValidBuildfabBinary(path) {
            return path, nil
        }
    }
    
    return "", fmt.Errorf("buildfab binary not found")
}

func isBuildfabBinary(path string) bool {
    // Check if binary name contains "buildfab"
    return strings.Contains(filepath.Base(path), "buildfab")
}
```

**Pros:**
- Automatic - no user configuration needed
- Works for most common cases
- Backward compatible

**Cons:**
- May find wrong buildfab version
- Heuristic-based detection could fail
- More complex implementation

### Option 3: Require buildfab in PATH (Simplest)

Document that `buildfab` must be in system PATH when using `run_action`/`run_stage`:

```go
func (r *ContainerRunner) getBuildfabBinaryPath() (string, error) {
    // Try current executable first
    if currentBinaryPath, err := r.getCurrentExecutablePath(); err == nil {
        if strings.Contains(filepath.Base(currentBinaryPath), "buildfab") {
            return currentBinaryPath, nil
        }
    }
    
    // Look in PATH
    path, err := exec.LookPath("buildfab")
    if err == nil {
        return path, nil
    }
    
    return "", fmt.Errorf("buildfab binary not found in PATH")
}
```

**Pros:**
- Simple implementation
- Standard Unix convention
- Automatic detection

**Cons:**
- Requires buildfab installation
- May not work in all environments

### Option 4: Hybrid Approach (Best of All)

Combine Option 1 and Option 2:

```go
// In SimpleRunOptions
type SimpleRunOptions struct {
    ConfigPath         string
    BuildfabBinaryPath string  // Optional: explicit path, takes precedence
    // ... other fields
}

// In getBuildfabBinaryPath
func (r *ContainerRunner) getBuildfabBinaryPath() (string, error) {
    // 1. Check if explicit path provided via options (highest priority)
    if r.buildfabPath != "" {
        if _, err := os.Stat(r.buildfabPath); err == nil {
            return r.buildfabPath, nil
        }
        return "", fmt.Errorf("specified buildfab binary not found: %s", r.buildfabPath)
    }
    
    // 2. Try current executable (for buildfab CLI)
    if currentPath, err := r.getCurrentExecutablePath(); err == nil {
        if strings.Contains(filepath.Base(currentPath), "buildfab") {
            return currentPath, nil
        }
    }
    
    // 3. Search in PATH
    if pathBinary, err := exec.LookPath("buildfab"); err == nil {
        return pathBinary, nil
    }
    
    // 4. Search common development locations
    wd, _ := r.getCurrentWorkingDir()
    searchPaths := []string{
        filepath.Join(wd, "bin", "buildfab"),
        filepath.Join(wd, "buildfab"),
    }
    
    for _, path := range searchPaths {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    
    return "", fmt.Errorf("buildfab binary not found (checked PATH, ./bin/, ./)")
}
```

**Pros:**
- Explicit path when needed
- Automatic detection when possible
- Works in most common scenarios
- Backward compatible

**Cons:**
- Slightly more complex

## Recommended Implementation

**Implement Hybrid Approach (Option 4)**

1. Add `BuildfabBinaryPath` field to `SimpleRunOptions` and `RunOptions`
2. Pass it through to `ContainerRunner`
3. Enhance `getBuildfabBinaryPath()` with smart search logic
4. Update documentation with usage examples

## Implementation Plan

### Phase 1: Add Configuration Fields
1. Add `BuildfabBinaryPath string` to `SimpleRunOptions` and `RunOptions`
2. Add `buildfabPath string` field to `ContainerRunner`
3. Pass the path when creating ContainerRunner

### Phase 2: Smart Binary Detection
1. Update `getBuildfabBinaryPath()` to check explicit path first
2. Add binary name validation to avoid mounting wrong executable
3. Add `exec.LookPath("buildfab")` for PATH search
4. Keep existing fallback search paths

### Phase 3: Documentation
1. Document `BuildfabBinaryPath` option in Library.md
2. Add examples showing explicit path usage
3. Update test-buildfab-api example to demonstrate usage
4. Document automatic detection behavior

### Phase 4: Testing
1. Test with explicit path specified
2. Test with buildfab in PATH
3. Test with buildfab in ./bin/
4. Test error messages when binary not found

## Usage Examples

### Automatic Detection (buildfab in PATH)
```go
// Just works if buildfab is in PATH
opts := &buildfab.SimpleRunOptions{
    ConfigPath:   ".project.yml",
    VerboseLevel: 1,
}
```

### Explicit Path
```go
// Specify exact buildfab binary to use
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "/usr/local/bin/buildfab",  // Explicit
    VerboseLevel:       1,
}
```

### Development Setup
```go
// Point to local development binary
opts := &buildfab.SimpleRunOptions{
    ConfigPath:         ".project.yml",
    BuildfabBinaryPath: "./bin/buildfab",  // Local build
    VerboseLevel:       1,
}
```

## Error Messages

Improve error messages to guide users:

```
Error: buildfab binary not found for run_action/run_stage

The container needs the buildfab binary to execute run_action and run_stage.

Solutions:
1. Install buildfab in PATH: 
   go install github.com/AlexBurnes/buildfab/cmd/buildfab@latest
   
2. Specify explicit path:
   opts.BuildfabBinaryPath = "/path/to/buildfab"
   
3. Use 'run:' instead of 'run_action:' to avoid needing buildfab:
   container:
     run: |
       echo "Direct commands don't need buildfab binary"

Searched locations:
- PATH
- ./bin/buildfab
- ./buildfab
```

## Benefits

1. **Flexibility**: Users can specify exact binary to use
2. **Convenience**: Automatic detection when possible
3. **Clarity**: Clear error messages when binary not found
4. **Compatibility**: Works with pre-push, test apps, and any tool using SimpleRunner

## Alternative: Don't Fix, Document Workaround

Simply document that `run:` should be used instead of `run_action:`/`run_stage:` when using SimpleRunner API.

**Pros:**
- No code changes needed
- Users learn to use direct commands
- Simpler container configurations

**Cons:**
- Less flexibility
- Can't reuse existing actions in containers
- Duplicates logic between regular and container actions

## Recommendation

**Implement Hybrid Approach (Option 4)** because:
- Provides both automatic detection and explicit control
- Enables full feature parity between CLI and SimpleRunner
- Relatively simple to implement
- Maintains backward compatibility
- Provides excellent user experience

