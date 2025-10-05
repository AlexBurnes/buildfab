# Container Feature Implementation Status

## Overview

This document describes the current implementation status of the Docker and Podman container feature in buildfab, including what works, what's broken, and what's missing.

## Current Status: ~70% Complete

### ✅ Working Features

1. **Basic Container Execution**
   - Docker and Podman engine detection
   - Container image pulling and execution
   - Basic shell command execution via `run` field
   - **Shell Compatibility**: Uses `sh` shell for maximum compatibility (not bash-specific syntax)
   - **Real-time Streaming Output**: Container output streams in real-time using streaming pipes and callbacks
   - **Matrix Integration**: Matrix feature works with containers for parallel execution across different images/configurations

2. **Container Configuration**
   - Image specification (`from` field)
   - Working directory (`workdir`)
   - Environment variables (`env`)
   - Volume mounts (`mounts`)
   - Resource limits (`cpus`, `memory`)
   - User specification (`user`)
   - Network configuration (`network`)

3. **Integration with buildfab**
   - Container actions properly integrated into action execution system
   - Matrix + container combination works for basic execution
   - Proper error handling and output display

### ❌ Broken Features

1. **`run_action` and `run_stage` Execution**
   - **Problem**: These execution modes fail with exit status 125
   - **Root Cause**: The `prepareContainer()` method is not implemented (just a placeholder)
   - **Impact**: buildfab binary and configuration are not copied into containers
   - **Error**: When containers try to run `buildfab action <name>`, buildfab is not installed

2. **Artifact Collection**
   - **Problem**: `collectArtifacts()` method is not implemented (just a placeholder)
   - **Impact**: No artifacts can be collected from containers

### 🚧 Missing Features

1. **Image Building**
   - Dockerfile building support not implemented
   - `build` field in image configuration not supported

2. **Environment File Support**
   - `env_file` field not implemented
   - No support for loading environment from files

3. **Cache Management**
   - `cache` field not implemented
   - No cache mount support

4. **Advanced Configuration**
   - Build arguments for Dockerfile builds
   - Multi-stage builds
   - Container health checks

## Fixed Issues

### ✅ Container Configuration Schema (FIXED)

**Problem**: Container configuration used `Commands` field instead of `run` field
**Solution**: Updated all container implementations to use `run` field consistently
**Impact**: Container configuration now matches action configuration schema

**Files Updated**:
- `pkg/buildfab/container/types.go` - Changed `Commands []string` to `Run string`
- `pkg/buildfab/container/engines.go` - Updated Docker and Podman engines
- `internal/container/docker.go` - Updated Docker implementation
- `internal/container/podman.go` - Updated Podman implementation
- `pkg/buildfab/buildfab.go` - Updated main buildfab implementation
- All container examples updated to use `run` field

## Current Container Configuration Schema

```yaml
container:
  image:
    from: "alpine:latest"          # Required: Container image
  workdir: "/workspace"            # Optional: Working directory
  env:                            # Optional: Environment variables
    KEY: "value"
  user: "1000:1000"               # Optional: User/group
  network: "host"                 # Optional: Network mode
  run: "echo 'Hello from container'"  # Shell command to execute (WORKING)
  run_stage: "test"               # Buildfab stage to run (BROKEN)
  run_action: "lint"              # Buildfab action to run (BROKEN)
  mounts:                         # Optional: Volume mounts (WORKING)
    - type: bind
      source: "."
      target: "/workspace"
      ro: false
  resources:                      # Optional: Resource limits (WORKING)
    cpus: "2.0"
    memory: "1GB"
```

## Working Examples

### Basic Container Execution
```yaml
actions:
  - name: hello-container
    description: Say hello from container
    container:
      image:
        from: alpine:latest
      run: |
        echo "Hello from container!"
        echo "Container is working correctly"
```

### Container with Mounts and Environment
```yaml
actions:
  - name: test-with-mounts
    description: Test container with mounts and environment
    container:
      image:
        from: alpine:latest
      workdir: /test
      env:
        TEST_VAR: "container-test"
      mounts:
        - type: bind
          source: .
          target: /test
          ro: false
      run: |
        echo "Test variable: $TEST_VAR"
        ls -la /test
```

### Shell Compatibility Guidelines
```yaml
# ✅ GOOD: Use sh-compatible syntax
run: |
  i=1
  while [ $i -le 10 ]; do
    echo "Line $i"
    i=$((i + 1))
  done

# ❌ BAD: Don't use bash-specific syntax
run: |
  for i in {1..10}; do  # This won't work in sh
    echo "Line $i"
  done
```

### Matrix + Container (Working Example)
```yaml
stages:
  test-platforms:
    steps:
      - action: platform-test
        matrix:
          values:
            image: ["alpine:latest", "ubuntu:22.04", "debian:12"]
        strategy:
          max_parallel: 3
          fail_fast: false
          continue_on_error: true

actions:
  - name: platform-test
    description: Test platform detection in ${{ matrix.image }}
    container:
      image:
        from: ${{ matrix.image }}
      workdir: /test
      mounts:
        - type: bind
          source: .
          target: /test
          ro: false
      run: |
        echo "=== Platform Detection Test ==="
        echo "Container Image: ${{ matrix.image }}"
        echo "Platform: $(uname -m)"
        echo "Architecture: $(uname -m)"
        echo "OS: $(uname -s)"
        echo "OS Version: $(cat /etc/os-release 2>/dev/null | head -1 || echo 'unknown')"
        echo "CPU Count: $(nproc 2>/dev/null || echo 'unknown')"
        echo "================================"
```

**Matrix-Container Integration Features:**
- **Parallel Execution**: Multiple containers run simultaneously based on matrix values
- **Variable Interpolation**: Matrix variables (`${{ matrix.image }}`) are interpolated in container configurations
- **Real-time Streaming**: Each container's output streams in real-time
- **Strategy Control**: Matrix strategy controls parallel execution, fail-fast behavior, and error handling

## Next Steps for Implementation

### Immediate Fixes (High Priority)

1. **Implement `prepareContainer()` Method**
   - Copy buildfab binary into container
   - Copy configuration file into container
   - Set up proper working directory
   - This will fix `run_action` and `run_stage` execution

2. **Implement `collectArtifacts()` Method**
   - Copy artifacts from container to host
   - Support pattern-based artifact collection
   - Handle directory and file artifacts

### Medium Priority Features

3. **Image Building Support**
   - Implement Dockerfile building
   - Support build context and arguments
   - Add build cache support

4. **Environment File Support**
   - Load environment variables from files
   - Support multiple environment files

5. **Cache Management**
   - Implement cache mount support
   - Add cache key generation and validation

### Low Priority Features

6. **Advanced Configuration**
   - Multi-stage builds
   - Container health checks
   - Advanced networking options
   - Security contexts

## Testing Status

### ✅ Working Tests
- Basic container execution
- Container with mounts
- Container with environment variables
- Container with resource limits
- Matrix + container basic execution

### ❌ Broken Tests
- `run_action` execution (buildfab not in container)
- `run_stage` execution (buildfab not in container)
- Artifact collection tests

### 🚧 Missing Tests
- Image building tests
- Environment file tests
- Cache management tests
- Error handling tests
- Performance tests

## Conclusion

The container feature is approximately 70% complete with basic container execution working well. The main blocker is the missing implementation of `prepareContainer()` and `collectArtifacts()` methods, which prevents `run_action` and `run_stage` from working. Once these are implemented, the container feature will be fully functional for the core use cases.

The schema fix (Commands → run) has been completed and all examples have been updated to use the correct configuration format.
