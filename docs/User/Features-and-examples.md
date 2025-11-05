# Features and Examples

This document provides comprehensive documentation of buildfab features with detailed examples and usage patterns.

## Table of Contents

- [Core Features](#core-features)
- [YAML Configuration Syntax](#yaml-configuration-syntax)
- [Action Variants](#action-variants)
- [Conditional Execution](#conditional-execution)
- [Matrix Feature](#matrix-feature)
  - [Multi-Dimensional Matrices](#multi-dimensional-matrices)
  - [Advanced Matrix Features (v0.32.0)](#advanced-matrix-features-v0320)
- [Container Support](#container-support)
- [Include System](#include-system)
- [Variable Interpolation](#variable-interpolation)
- [Built-in Actions](#built-in-actions)
- [Advanced Examples](#advanced-examples)
- [CLI Usage Examples](#cli-usage-examples)
- [Library API Examples](#library-api-examples)

## Core Features

### YAML-Driven Configuration

buildfab uses intuitive YAML configuration files to define your automation workflows:

```yaml
project:
  name: "my-project"
  modules: ["my-app"]
  bin: "bin"

actions:
  - name: test
    run: |
      go test ./...
  
  - name: build
    run: |
      go build -o bin/my-app ./cmd/my-app

stages:
  pre-push:
    steps:
      - action: test
      - action: build
```

### DAG-Based Execution

buildfab automatically builds a Directed Acyclic Graph (DAG) from your stage configuration, enabling parallel execution with proper dependency management:

```yaml
stages:
  build:
    steps:
      - action: install-deps
      - action: compile
        require: [install-deps]
      - action: test
        require: [compile]
      - action: package
        require: [compile]
      # test and package run in parallel after compile completes
```

### Cross-Platform Support

buildfab automatically detects your platform and provides platform-specific variables:

- **Linux**: `platform=linux`, `os=ubuntu|debian`, `arch=amd64|arm64`
- **Windows**: `platform=windows`, `os=windows`, `arch=amd64|arm64`
- **macOS**: `platform=darwin`, `os=darwin`, `arch=amd64|arm64`

## YAML Configuration Syntax

### Project Configuration

```yaml
project:
  name: "project-name"           # Required: Project name
  modules: ["module1", "module2"] # Optional: Go modules
  bin: "bin"                     # Optional: Binary directory
```

### Actions

Actions define executable units with two types:

#### Custom Actions (run)
```yaml
actions:
  - name: build
    run: |
      echo "Building project..."
      go build -o bin/app ./cmd/app
```

#### Built-in Actions (uses)
```yaml
actions:
  - name: git-check
    uses: git@untracked
```

### Stages and Steps

```yaml
stages:
  stage-name:
    steps:
      - action: action-name
        require: [dependency1, dependency2]  # Optional: Dependencies
        onerror: warn                         # Optional: warn|stop (default: stop)
        only: [release, production]          # Optional: Labels
        if: "os == 'linux'"                  # Optional: Condition
```

## Action Variants

Action variants allow platform-specific execution within a single action:

```yaml
actions:
  - name: build
    variants:
      - when: "${{ platform == 'linux' }}"
        run: |
          echo "Building for Linux..."
          cmake -S . -B build && cmake --build build
      
      - when: "${{ platform == 'windows' }}"
        shell: powershell
        run: |
          Write-Host "Building for Windows..."
          cmake -S . -B build -G "Visual Studio 17 2022"
          cmake --build build --config Release
      
      - when: "${{ platform == 'darwin' }}"
        run: |
          echo "Building for macOS..."
          cmake -S . -B build -G "Xcode"
          cmake --build build --config Release
```

### Variant Selection Logic

- **First matching variant**: buildfab selects the first variant whose `when` condition evaluates to true
- **No variants match**: Action is skipped with status "SKIP"
- **Mixed run/uses**: Variants can use different execution types

```yaml
actions:
  - name: git-check
    variants:
      - when: "${{ os == 'linux' }}"
        uses: git@untracked
      - when: "${{ os == 'windows' }}"
        run: git status --porcelain | findstr "^??"
```

## Conditional Execution

### Step-Level Conditions

Steps can use `if` conditions for conditional execution:

```yaml
stages:
  test:
    steps:
      - action: unit-tests
        if: "os == 'linux'"
      
      - action: integration-tests
        if: "platform == 'windows' && arch == 'amd64'"
      
      - action: always-runs
        # No condition - always executes
```

### Expression Language

buildfab supports a powerful expression language similar to GitHub Actions:

#### Variables
- **Platform variables**: `${{ platform }}`, `${{ arch }}`, `${{ os }}`, `${{ os_version }}`, `${{ cpu }}`
- **Environment variables**: `${{ env.VAR_NAME }}`
- **Input variables**: `${{ inputs.name }}`
- **Matrix variables**: `${{ matrix.os }}`
- **Boolean variables**: `${{ ci }}`, `${{ branch }}`

#### Operators
```yaml
# Comparison operators
if: "os == 'linux'"
if: "arch != 'arm64'"
if: "cpu >= 4"
if: "os_version < '20.04'"

# Logical operators
if: "os == 'linux' && arch == 'amd64'"
if: "platform == 'windows' || platform == 'darwin'"
if: "!(os == 'windows')"

# Parentheses for grouping
if: "(os == 'linux' || os == 'darwin') && cpu >= 2"
```

#### Helper Functions
```yaml
# String functions
if: "contains(os, 'ubuntu')"
if: "startsWith(arch, 'arm')"
if: "endsWith(os_version, '.0')"
if: "matches(platform, 'linux|darwin')"

# File system functions
if: "fileExists('package.json')"
if: "fileExists('CMakeLists.txt')"

# Version comparison
if: "semverCompare(os_version, '>=20.04')"
```

### Label-Based Execution (only)

Steps can be restricted to specific labels:

```yaml
stages:
  release:
    steps:
      - action: build
      - action: test
        only: [release]
      - action: deploy
        only: [release, production]
```

Run with labels:
```bash
buildfab run release --only release
buildfab run release --only release,production
```

## Matrix Feature

The Matrix feature enables parallel execution across multiple configurations, allowing you to run the same action with different parameter combinations.

### Basic Matrix Configuration

```yaml
stages:
  test-matrix:
    - action: test-action
      matrix:
        values:
          os: ["linux", "windows", "macos"]
        strategy:
          max_parallel: 2
          fail_fast: true
          continue_on_error: false
          order: "fifo"
```

### Matrix Variable Interpolation

Matrix values are available as variables in action commands:

```yaml
actions:
  - name: test-action
    run: echo "Testing on ${{ matrix.os }}"
```

### Cross-Platform Testing Example

```yaml
stages:
  cross-platform-test:
    - action: run-tests
      matrix:
        values:
          os: ["linux", "windows", "macos"]
        strategy:
          max_parallel: 3
          fail_fast: false
          continue_on_error: false

actions:
  - name: run-tests
    run: |
      echo "Running tests on ${{ matrix.os }}"
      # Your test commands here
```

### Multi-Dimensional Matrices

buildfab supports multi-dimensional matrices where dimensions can have nested sub-dimensions. This enables complex build matrices like testing multiple OS versions with different compiler sets.

#### Simple Multi-Dimensional Matrix

```yaml
stages:
  cross-compile:
    steps:
      - action: build
        matrix:
          values:
            platform:
              - linux:
                  compiler: ["gcc", "clang"]
              - windows:
                  compiler: ["msvc", "mingw"]
            build_type: ["Release", "Debug"]
          strategy:
            max_parallel: 4

actions:
  - name: build
    run: |
      echo "Building for ${{ matrix.platform }} with ${{ matrix.compiler }}"
      echo "Build type: ${{ matrix.build_type }}"
```

This generates **8 combinations**:
- `linux × gcc × Release`
- `linux × gcc × Debug`
- `linux × clang × Release`
- `linux × clang × Debug`
- `windows × msvc × Release`
- `windows × msvc × Debug`
- `windows × mingw × Release`
- `windows × mingw × Debug`

#### Complex Multi-Dimensional Matrix

For more complex scenarios with varying sub-dimensions per main dimension:

```yaml
stages:
  full-build:
    steps:
      - action: build-and-test
        matrix:
          values:
            images:
              - centos7:
                  compiler: "gcc"
              - centos8:
                  compiler: ["gcc", "clang"]
              - centos9:
                  compiler: ["gcc", "clang", "icc"]
            builds: ["Release", "Debug"]
          strategy:
            max_parallel: 6
            fail_fast: false

actions:
  - name: build-and-test
    run: |
      echo "Building on ${{ matrix.images }}"
      echo "Compiler: ${{ matrix.compiler }}"
      echo "Config: ${{ matrix.builds }}"
```

This generates **12 combinations**:
- CentOS 7: 1 compiler × 2 build types = 2 combinations
- CentOS 8: 2 compilers × 2 build types = 4 combinations  
- CentOS 9: 3 compilers × 2 build types = 6 combinations
- **Total: 12 combinations**

#### Flat Variable Naming

All matrix variables are flattened to a single level for easy access:
- `${{ matrix.images }}` - The main dimension value (e.g., "centos7", "centos8")
- `${{ matrix.compiler }}` - The sub-dimension value (e.g., "gcc", "clang", "icc")
- `${{ matrix.builds }}` - Another dimension value (e.g., "Release", "Debug")

**Note**: Variables use flat naming (`matrix.compiler`) instead of nested naming (`matrix.images.compiler`). This prevents naming conflicts and simplifies variable access in actions.

#### Step Naming with Multi-Dimensional Matrices

Generated steps include all matrix dimensions in their names for clear identification:

```
build-and-test.Release.gcc.centos7
build-and-test.Release.gcc.centos8
build-and-test.Release.clang.centos8
build-and-test.Debug.icc.centos9
```

Each step name follows the pattern: `action.value1.value2.value3` (alphabetically sorted dimensions).

#### Real-World Example: Container Testing

```yaml
stages:
  container-tests:
    steps:
      - action: container-test
        matrix:
          values:
            image:
              - python:
                  tag: ["3.9", "3.10", "3.11"]
                  suite: ["unit", "integration"]
              - node:
                  tag: ["18", "20"]
                  suite: ["unit", "e2e"]
              - alpine:
                  tag: ["3.18"]
                  suite: ["unit"]
          strategy:
            max_parallel: 6
            continue_on_error: true

actions:
  - name: container-test
    container:
      image:
        from: ${{ matrix.image }}:${{ matrix.tag }}
      run: |
        echo "Testing in ${{ matrix.image }}:${{ matrix.tag }}"
        echo "Running suite: ${{ matrix.suite }}"
```

This generates:
- Python: 3 versions × 2 suites = 6 combinations
- Node: 2 versions × 2 suites = 4 combinations
- Alpine: 1 version × 1 suite = 1 combination
- **Total: 11 container test combinations**

See `examples/matrix-multidimensional-*.yml` for complete working examples.

### Matrix Strategy Options

- **`max_parallel`**: Maximum concurrent jobs. Creates a dedicated execution pool for the matrix. If not specified, matrix jobs use the global pool. **Note**: Effective parallelism = `min(global_max_parallel, matrix_max_parallel)` when both are set.
- **`fail_fast`**: Stop all jobs on first failure (default: false)
- **`continue_on_error`**: Stage succeeds even if some jobs fail (default: false)
- **`order`**: Job scheduling order - "fifo" or "random" (default: "fifo")

### Global and Matrix Parallelism Control

buildfab provides two levels of concurrency control:

#### Global Concurrency Control

Set project-wide parallel execution limit:

```yaml
project:
  name: "my-project"
  max_parallel: 4  # Global limit for all concurrent execution
```

This creates a global execution pool that limits total concurrent tasks across the entire DAG execution.

#### Matrix-Specific Pools

Matrix steps with `max_parallel` get dedicated execution pools:

```yaml
project:
  max_parallel: 8  # Global pool size

stages:
  test:
    - action: matrix-test
      matrix:
        values:
          platform: ["linux", "windows", "macos", "freebsd"]
        strategy:
          max_parallel: 2  # Dedicated matrix pool with 2 workers
```

#### Min() Strategy - How Limits Interact

When both global and matrix limits are set, they interact using the min() strategy:

**Example 1: Global Restricts Matrix**
```yaml
project:
  max_parallel: 1  # Low global limit

stages:
  test:
    - action: matrix-action
      matrix:
        values:
          item: ["1", "2", "3", "4"]
        strategy:
          max_parallel: 2  # Matrix wants 2 concurrent
```
**Result**: Jobs run **one at a time** (effective = min(1, 2) = 1)

**Example 2: Matrix Self-Limits**
```yaml
project:
  max_parallel: 10  # High global limit

stages:
  test:
    - action: matrix-action
      matrix:
        values:
          item: ["1", "2", "3", "4"]
        strategy:
          max_parallel: 2  # Matrix restricts itself
```
**Result**: Jobs run **2 at a time** (effective = min(10, 2) = 2)

**Example 3: No Matrix Limit - Uses Global Pool**
```yaml
project:
  max_parallel: 4  # Global limit

stages:
  test:
    - action: matrix-action
      matrix:
        values:
          platform: ["linux", "windows", "macos", "freebsd"]
        # NO max_parallel - uses global pool
```
**Result**: All 4 jobs run in parallel using the global pool

### Matrix on Stage Steps

Matrix can now be applied to steps that reference other stages, enabling cross-compiler builds, multi-platform testing, and parameterized stage execution:

```yaml
stages:
  build:
    steps:
      - action: check-conan
      - action: check-cmake
      - action: conan-install
        require: [check-conan, check-cmake]
      - action: cmake-config
        require: [conan-install]
      - action: cmake-build
        require: [cmake-config]
      - action: cmake-test
        require: [cmake-build]

  compiler-build:
    steps:
      - stage: build
        matrix:
          values:
            compiler: ["gcc", "clang"]
          strategy:
            max_parallel: 1
```

**Key Features:**

1. **Sequential Execution (`max_parallel: 1`)**: 
   - Each matrix job runs completely before the next starts
   - Ensures clean separation between different configurations
   - Ideal for resource-intensive builds

2. **Parallel Execution (`max_parallel: N`)**:
   - Uses sliding window dependency pattern
   - At most N matrix jobs run concurrently
   - Automatic dependency injection ensures correct ordering
   - Maximizes resource utilization while maintaining order

3. **Matrix Variable Propagation**:
   - All steps in expanded stages have access to `${{ matrix.* }}` variables
   - Variables are available for interpolation in action commands
   - Enables configuration-specific builds and tests

**Complete Example: Cross-Platform Build**

```yaml
project:
  name: cross-platform-app

actions:
  - name: setup-platform
    run: |
      echo "Setting up for ${{ matrix.platform }}"
      export PLATFORM=${{ matrix.platform }}

  - name: build-source
    run: |
      echo "Building for ${{ matrix.platform }}"
      # Platform-specific build commands

  - name: run-tests
    run: |
      echo "Testing on ${{ matrix.platform }}"
      # Platform-specific tests

  - name: package-artifact
    run: |
      echo "Packaging for ${{ matrix.platform }}"
      # Platform-specific packaging

stages:
  build-platform:
    steps:
      - action: setup-platform
      - action: build-source
        require: [setup-platform]
      - action: run-tests
        require: [build-source]
      - action: package-artifact
        require: [run-tests]

  all-platforms:
    steps:
      - stage: build-platform
        matrix:
          values:
            platform: ["linux-amd64", "linux-arm64", "windows-amd64", "darwin-amd64", "darwin-arm64"]
          strategy:
            max_parallel: 2
            fail_fast: false
            continue_on_error: true
```

This configuration:
- Builds the complete pipeline for each platform
- Runs up to 2 platforms concurrently
- Continues even if some platforms fail
- All steps have access to `${{ matrix.platform }}` variable

For detailed matrix feature documentation, see [Matrix Feature Documentation](Matrix-feature.md).

### Advanced Matrix Features (v0.32.0)

buildfab v0.32.0 introduced significant improvements to matrix execution with the hierarchical DAG architecture.

#### Nested Matrix Support

Matrix on stage references now works correctly with proper job-based execution:

```yaml
stages:
  build:
    steps:
      - action: configure
      - action: compile
        require: [configure]
      - action: test
        require: [compile]

  cross-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: [gcc, clang]
            version: ["11", "12", "13"]
# Creates 6 jobs (gcc-11, gcc-12, gcc-13, clang-11, clang-12, clang-13)
# Each job runs sequentially: configure → compile → test
# All 6 jobs execute in parallel (respecting max_parallel)
```

**How It Works**:
1. Matrix expansion creates one **job** per combination
2. Each job contains all steps from the referenced stage
3. Steps within a job execute **sequentially**
4. Jobs execute in **parallel waves**

#### Sliding Window Dependencies

When using `max_parallel` with matrix builds, buildfab creates sliding window dependencies to control concurrency:

```yaml
stages:
  test:
    steps:
      - action: test-platform
        matrix:
          values:
            platform: [p1, p2, p3, p4, p5, p6]
          strategy:
            max_parallel: 3
```

**Execution Pattern**:
```
Time →
Wave 1:  p1  p2  p3  (3 jobs start)
Wave 2:  p4          (starts when p1 completes)
Wave 3:  p5          (starts when p2 completes)
Wave 4:  p6          (starts when p3 completes)
```

This prevents all 6 jobs from starting simultaneously, avoiding resource exhaustion.

#### Condition-Based Skips with Sliding Window

Condition skips don't block sliding window dependencies (v0.32.0 fix):

```yaml
stages:
  build:
    steps:
      - action: build
        matrix:
          values:
            os: [centos7, centos8, centos9]
            compiler: [gcc, clang]
        if: "!(matrix.os == 'centos7' && matrix.compiler == 'clang')"
        strategy:
          max_parallel: 2
# If centos7-clang is skipped due to condition:
# - Next job in sliding window can still start
# - Doesn't block centos8-gcc from running
# - Only 2 jobs run concurrently at any time
```

**Smart Skip Behavior** (v0.32.0):
- **Sliding window dependencies**: Condition skips don't block (parallelism maintained)
- **User dependencies** (`require`, `depends_on`): Condition skips do block (safety maintained)

#### Complex NOT Expressions

The NOT operator now correctly handles complex nested conditions:

```yaml
steps:
  - action: build
    if: "!(matrix.images == 'centos7' && matrix.compiler == 'clang')"
    # Skips only this specific combination
  
  - action: deploy
    if: "!((os == 'windows' || os == 'darwin') && env.CI == 'true')"
    # Skips deploy on Windows/macOS in CI

  - action: test
    if: "!(matrix.variant == 'debug') && version.type == 'release'"
    # Skips debug variant for release builds
```

**Operator Precedence** (v0.32.0 fix):
- Binary operators (`&&`, `||`) evaluated before unary (`!`)
- Parentheses properly respected
- Complex nested expressions work correctly

#### Matrix Variable Propagation

Matrix variables correctly propagate in all scenarios (v0.32.0 fix):

```yaml
stages:
  inner-build:
    steps:
      - action: compile
        run: gcc ${{ matrix.flags }} -o app main.c
      - action: test
        run: ./app --mode=${{ matrix.mode }}

  outer-matrix:
    steps:
      - stage: inner-build
        matrix:
          values:
            flags: ["-O0", "-O2", "-O3"]
            mode: ["debug", "release"]
# All steps in inner-build have access to matrix.flags and matrix.mode
# Variables propagate correctly through stage references
```

#### User Dependencies with Matrix Jobs

User dependencies (`require`, `depends_on`) are properly inherited by matrix jobs:

```yaml
stages:
  ci:
    steps:
      - action: prepare
      
      - action: build
        require: [prepare]
        matrix:
          values:
            platform: [linux, darwin, windows]
# Each of the 3 matrix jobs (linux, darwin, windows)
# properly depends on "prepare" step
# All 3 jobs wait for prepare to complete before starting
```

#### Global max_parallel Enforcement

Global `max_parallel` setting is properly enforced (v0.32.0 fix):

```yaml
project:
  max_parallel: 4

stages:
  test:
    steps:
      - action: test-variant
        matrix:
          values:
            variant: [v1, v2, v3, v4, v5, v6, v7, v8]
# Even though matrix has 8 combinations,
# only 4 jobs run concurrently (global limit enforced)
```

**Priority Rules**:
- Matrix `max_parallel` creates dedicated pool
- Effective limit = `min(global_max_parallel, matrix_max_parallel)`
- Global setting provides hard upper limit

## Container Support

buildfab provides native container integration with Docker and Podman for running actions in isolated environments.

### Basic Container Action

```yaml
actions:
  - name: test
    run: go test ./...

  - name: test-in-container
    container:
      image:
        from: golang:1.22
        pull: missing
      run_action: test
      volumes:
        - $PWD:/workspace
      environment:
        - GOMODCACHE=/workspace/.cache
```

Run the container action:
```bash
buildfab action test-in-container
```

### Container Configuration Reference

#### Image Configuration

```yaml
container:
  image:
    from: "golang:1.22"      # Container image name (required)
    pull: missing            # Pull policy: always, missing, never (default: missing)
```

**Pull Policies**:
- `always`: Always pull image before running (ensures latest)
- `missing`: Pull only if image not found locally (default, faster)
- `never`: Never pull, use local image only (offline builds)

#### Engine Selection

```yaml
container:
  engine: podman            # Explicitly use podman (default: auto-detect)
```

buildfab automatically detects available engines in order:
1. Docker (if `docker` command available)
2. Podman (if `podman` command available)

Specify `engine` to force a specific container runtime.

#### Resource Limits

```yaml
container:
  cpu: 2                    # CPU cores limit (default: unlimited)
  memory: "4G"              # Memory limit: 4G, 2048M, etc. (default: unlimited)
  network: host             # Network mode: host, bridge, none (default: bridge)
```

**Network Modes**:
- `host`: Use host network stack (fastest, less isolated)
- `bridge`: Bridged network (default, balanced)
- `none`: No network access (maximum isolation)

#### Volume Mounting

Mount host directories into the container:

```yaml
container:
  volumes:
    - $PWD:/workspace                      # Mount current directory
    - $HOME/.cache:/cache                  # Mount cache directory
    - ./config:/app/config:ro              # Read-only mount
    - /tmp:/tmp                            # Mount temp directory
```

**Volume Syntax**: `<host-path>:<container-path>[:ro]`
- `host-path`: Absolute or relative path on host (variables supported)
- `container-path`: Absolute path in container
- `:ro`: Optional read-only flag

**Variable Interpolation in Volumes**:
```yaml
volumes:
  - ${{ env.HOME }}/.cache:/cache
  - ${{ PWD }}/build:/workspace/build
```

#### Environment Variables

Pass environment variables to the container:

```yaml
container:
  environment:
    - GO111MODULE=on                       # Set specific value
    - GOPATH                               # Pass from host environment
    - GOCACHE=/workspace/.cache            # Set container-specific path
    - BUILD_TYPE=${{ version.build-type }} # Use buildfab variables
```

**Environment Syntax**:
- `KEY=value`: Set variable to specific value
- `KEY`: Pass variable from host environment
- Supports variable interpolation with `${{ }}`

#### Working Directory

```yaml
container:
  workdir: /workspace        # Working directory inside container (default: /)
```

#### Running Actions/Stages in Container

**Run a single action**:
```yaml
container:
  image:
    from: golang:1.22
  run_action: test           # Action name to run inside container
```

**Run a complete stage**:
```yaml
container:
  image:
    from: golang:1.22
  run_stage: integration     # Stage name to run inside container
```

### Advanced Container Examples

#### Multi-Platform Testing with Containers

```yaml
actions:
  - name: test
    run: go test ./...

  - name: test-distro
    container:
      image:
        from: ${{ matrix.distro }}
        pull: missing
      run_action: test
      volumes:
        - $PWD:/workspace
      workdir: /workspace
      environment:
        - CGO_ENABLED=0

stages:
  test-all:
    steps:
      - action: test-distro
        matrix:
          values:
            distro: [ubuntu:22.04, alpine:latest, debian:12]
          strategy:
            max_parallel: 2
# Tests run in 3 different distros, 2 concurrent at a time
```

#### Container Build with Caching

```yaml
actions:
  - name: build
    run: go build -o bin/app ./cmd/app

  - name: build-cached
    container:
      image:
        from: golang:1.22
        pull: missing
      run_action: build
      volumes:
        - $PWD:/workspace
        - $HOME/.cache/go-build:/root/.cache/go-build
        - $HOME/.cache/go-mod:/go/pkg/mod
      workdir: /workspace
      environment:
        - GOCACHE=/root/.cache/go-build
        - GOMODCACHE=/go/pkg/mod
        - CGO_ENABLED=0
```

#### Container with Resource Limits

```yaml
actions:
  - name: intensive-task
    container:
      image:
        from: buildpack:latest
      cpu: 4                          # Limit to 4 CPU cores
      memory: "8G"                    # Limit to 8GB RAM
      network: none                   # No network access
      run_action: compile
      volumes:
        - $PWD:/workspace
      workdir: /workspace
```

#### Container Matrix Build

```yaml
actions:
  - name: cross-compile
    run: GOOS=${{ matrix.os }} GOARCH=${{ matrix.arch }} go build -o bin/app-${{ matrix.os }}-${{ matrix.arch }}

  - name: build-matrix
    container:
      image:
        from: golang:1.22
      run_action: cross-compile
      volumes:
        - $PWD:/workspace
      workdir: /workspace

stages:
  release:
    steps:
      - action: build-matrix
        matrix:
          values:
            os: [linux, darwin, windows]
            arch: [amd64, arm64]
# Creates 6 container jobs (linux-amd64, linux-arm64, darwin-amd64, etc.)
# Each job runs cross-compile action in isolated container
```

### Container Requirements

**Buildfab Binary**:
- Must be available for `run_action` and `run_stage` to work
- Auto-discovered from: `/usr/local/bin`, `/usr/bin`, `$HOME/bin`, `./scripts`
- Or provide explicit path in configuration

**Container Engine**:
- Docker or Podman must be installed and available in PATH
- User must have permissions to run containers

### Container Troubleshooting

#### Container Engine Not Found

```
Error: no container engine found (docker or podman required)
```

**Solution**:
```bash
# Install Docker
sudo apt install docker.io         # Ubuntu/Debian
sudo systemctl start docker

# Or install Podman
sudo apt install podman             # Ubuntu/Debian
```

#### Buildfab Binary Not Found

```
Error: buildfab binary not found, required for run_action/run_stage
```

**Solution**:
```bash
# Install buildfab to standard directory
sudo cp bin/buildfab /usr/local/bin/

# Or add to PATH
export PATH=$PATH:$(pwd)/bin
```

#### Permission Denied

```
Error: permission denied while trying to connect to Docker daemon
```

**Solution**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Log out and log back in

# Or use Podman (rootless by default)
sudo apt install podman
```

#### Volume Mount Issues

```
Error: invalid mount specification
```

**Solution**:
- Use absolute paths or `$PWD` for host paths
- Ensure host directory exists
- Check permissions on host directory
- Use `:ro` for read-only mounts when possible

### Container Best Practices

1. **Use specific image tags**: `golang:1.22` not `golang:latest`
2. **Cache dependencies**: Mount cache directories for faster builds
3. **Set resource limits**: Prevent runaway processes
4. **Use pull: missing**: Faster for local development
5. **Minimize volumes**: Only mount what's needed
6. **Set working directory**: Ensure consistent execution context
7. **Test locally first**: Verify container config before CI

## Include System

The include system allows you to organize complex configurations into smaller, manageable files:

### Basic Include Usage

```yaml
# project.yml
project:
  name: "my-project"

include:
  - actions/common.yml
  - actions/build.yml
  - stages/ci.yml

# Main configuration can override included content
actions:
  - name: main-action
    run: echo "Main action"
```

### Include Patterns

```yaml
include:
  - "actions.yml"           # Exact file path (must exist)
  - "config/*.yml"         # Glob pattern (directory must exist)
  - "stages/common.yml"    # Subdirectory file
  - "platforms/*.yml"      # Multiple files matching pattern
```

### Include Behavior

- **Exact paths**: Must exist or configuration fails
- **Glob patterns**: Directory must exist, files optional
- **Merge order**: Later includes override earlier ones
- **Circular detection**: Prevents infinite loops
- **File types**: Only `.yml` and `.yaml` files processed

### Example: Modular Configuration

**Main file (`project.yml`)**:
```yaml
project:
  name: "my-project"

include:
  - actions/test.yml
  - actions/build.yml
  - stages/ci.yml

stages:
  main:
    steps:
      - action: test
      - action: build
```

**Actions file (`actions/test.yml`)**:
```yaml
actions:
  - name: test
    run: go test ./...
  - name: test-coverage
    run: go test -cover ./...
```

**Actions file (`actions/build.yml`)**:
```yaml
actions:
  - name: build
    run: go build ./...
  - name: build-static
    run: go build -ldflags="-s -w" ./...
```

**Stages file (`stages/ci.yml`)**:
```yaml
stages:
  ci:
    steps:
      - action: test
      - action: build
      - action: test-coverage
```

## Variable Interpolation

buildfab supports GitHub-style variable interpolation with `${{ }}` syntax:

### Default Values

Use a fallback value when a variable is undefined:

- Literal default: `${{ variable-default }}` (quotes optional)
- From another variable: `${{ variable-other.variable }}`

Examples:

```yaml
actions:
  - name: defaults-demo
    run: |
      echo "Compiler: ${{ matrix.compiler-\"gcc\" }}"
      echo "Image: ${{ matrix.image-ubuntu:24.04 }}"
      echo "Compiler via variable: ${{ matrix.compiler-variable.compiler_default }}"
```

Notes:
- If the primary variable exists, it is used as-is.
- If missing and default is provided, the default is used.
- Defaults can be quoted; quotes are stripped.
- Defaults can reference another variable; that reference is resolved recursively.
- Missing variables without defaults cause an error with a list of available variables.

### Platform Variables

```yaml
actions:
  - name: platform-info
    run: |
      echo "Platform: ${{ platform }}"
      echo "Architecture: ${{ arch }}"
      echo "OS: ${{ os }}"
      echo "OS Version: ${{ os_version }}"
      echo "CPU Cores: ${{ cpu }}"
```

### Git Variables

```yaml
actions:
  - name: git-info
    run: |
      echo "Current branch: ${{ branch }}"
      echo "Latest tag: ${{ tag }}"
```

### Version Variables (when using version-go integration)

```yaml
actions:
  - name: version-info
    run: |
      echo "Project version: ${{ version.version }}"
      echo "Project name: ${{ version.project }}"
```

### Environment Variables

```yaml
actions:
  - name: env-info
    run: |
      echo "Go version: ${{ env.GO_VERSION }}"
      echo "Build target: ${{ env.BUILD_TARGET }}"
```

Pass environment variables:
```bash
buildfab run build --env GO_VERSION=1.23.1 --env BUILD_TARGET=linux
```

## Built-in Actions

buildfab includes several built-in actions for common automation tasks:

### Git Actions

```yaml
actions:
  - name: git-untracked
    uses: git@untracked      # Fail if untracked files present
  
  - name: git-uncommitted
    uses: git@uncommitted    # Fail if staged/unstaged changes present
  
  - name: git-modified
    uses: git@modified       # Warn if working tree differs from HEAD
    onerror: warn
```

### Version Actions

```yaml
actions:
  - name: version-check
    uses: version@check           # Validate version format
  
  - name: version-greatest
    uses: version@check-greatest  # Ensure current version is greatest
```

### Using Built-in Actions

Built-in actions can be used in two ways:

1. **In YAML configuration**:
```yaml
actions:
  - name: git-untracked
    uses: git@untracked

stages:
  pre-push:
    steps:
      - action: git-untracked
```

2. **Directly via CLI**:
```bash
# Run built-in actions directly
buildfab action git@untracked
buildfab action version@check

# List all available built-in actions
buildfab list-actions
```

## Advanced Examples

### Multi-Platform Build Pipeline

```yaml
project:
  name: "cross-platform-app"

actions:
  - name: build-linux
    variants:
      - when: "${{ platform == 'linux' }}"
        run: |
          echo "Building for Linux..."
          GOOS=linux GOARCH=amd64 go build -o bin/app-linux-amd64 ./cmd/app
          GOOS=linux GOARCH=arm64 go build -o bin/app-linux-arm64 ./cmd/app
  
  - name: build-windows
    variants:
      - when: "${{ platform == 'windows' }}"
        run: |
          echo "Building for Windows..."
          GOOS=windows GOARCH=amd64 go build -o bin/app-windows-amd64.exe ./cmd/app
          GOOS=windows GOARCH=arm64 go build -o bin/app-windows-arm64.exe ./cmd/app
  
  - name: build-macos
    variants:
      - when: "${{ platform == 'darwin' }}"
        run: |
          echo "Building for macOS..."
          GOOS=darwin GOARCH=amd64 go build -o bin/app-darwin-amd64 ./cmd/app
          GOOS=darwin GOARCH=arm64 go build -o bin/app-darwin-arm64 ./cmd/app
  
  - name: test
    run: |
      go test ./...
  
  - name: package
    variants:
      - when: "${{ platform == 'linux' }}"
        run: |
          tar -czf app-linux.tar.gz bin/app-linux-*
      - when: "${{ platform == 'windows' }}"
        run: |
          powershell Compress-Archive -Path bin/app-windows-* -DestinationPath app-windows.zip
      - when: "${{ platform == 'darwin' }}"
        run: |
          tar -czf app-macos.tar.gz bin/app-darwin-*

stages:
  build:
    steps:
      - action: build-linux
      - action: build-windows
      - action: build-macos
      - action: test
      - action: package
```

### Environment-Specific Deployment

```yaml
project:
  name: "web-app"

actions:
  - name: deploy
    variants:
      - when: "${{ env.ENVIRONMENT == 'production' }}"
        run: |
          echo "Deploying to production..."
          kubectl apply -f k8s/production/
          kubectl rollout status deployment/web-app
      
      - when: "${{ env.ENVIRONMENT == 'staging' }}"
        run: |
          echo "Deploying to staging..."
          kubectl apply -f k8s/staging/
          kubectl rollout status deployment/web-app-staging
      
      - when: "${{ env.ENVIRONMENT == 'development' }}"
        run: |
          echo "Deploying to development..."
          docker-compose -f docker-compose.dev.yml up -d

stages:
  deploy:
    steps:
      - action: deploy
```

### Conditional Testing Pipeline

```yaml
project:
  name: "microservices"

actions:
  - name: unit-tests
    run: go test ./... -short
  
  - name: integration-tests
    run: go test ./... -tags=integration
    if: "contains(env.TEST_LEVEL, 'integration')"
  
  - name: e2e-tests
    run: go test ./... -tags=e2e
    if: "contains(env.TEST_LEVEL, 'e2e')"
  
  - name: performance-tests
    run: go test ./... -tags=performance -bench=.
    if: "contains(env.TEST_LEVEL, 'performance')"
  
  - name: security-scan
    variants:
      - when: "${{ os == 'linux' }}"
        run: |
          docker run --rm -v $(pwd):/app securecodewarrior/docker-security-scanner /app
      - when: "${{ os == 'windows' }}"
        run: |
          powershell -Command "Invoke-WebRequest -Uri 'https://security-scanner.exe' -OutFile 'scanner.exe'; .\scanner.exe"
  
  - name: coverage-report
    run: |
      go test ./... -coverprofile=coverage.out
      go tool cover -html=coverage.out -o coverage.html
    if: "env.COVERAGE == 'true'"

stages:
  test:
    steps:
      - action: unit-tests
      - action: integration-tests
      - action: e2e-tests
      - action: performance-tests
      - action: security-scan
      - action: coverage-report
```

## CLI Usage Examples

### Basic Commands

```bash
# Run a stage
buildfab run pre-push

# Run a specific action
buildfab action version@check

# Run with verbose output (default)
buildfab run build --verbose

# Run in quiet mode
buildfab run build --quiet

# Preview execution plan
buildfab run build --dry-run
```

### Advanced CLI Usage

```bash
# Run with custom configuration
buildfab run build --config my-project.yml

# Run with environment variables
buildfab run deploy --env ENVIRONMENT=production --env VERSION=v1.2.3

# Run only steps with specific labels
buildfab run release --only production,stable

# Run with custom working directory
buildfab run build --working-dir /path/to/project

# Run with custom parallel limit
buildfab run test --max-parallel 4

# Run with debug output
buildfab run build --debug
```

### Listing and Validation

```bash
# List all available actions (built-in and defined)
buildfab list-actions

# List all stages
buildfab list-stages

# List steps in a stage
buildfab list-steps pre-push

# Validate configuration
buildfab validate

# Validate specific configuration file
buildfab validate --config my-project.yml
```

## Library API Examples

### SimpleRunner (Recommended)

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        panic(err)
    }
    
    // Create simple run options
    opts := &buildfab.SimpleRunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
    }
    
    // Create simple runner
    runner := buildfab.NewSimpleRunner(cfg, opts)
    
    // Run a stage - all output is handled automatically!
    err = runner.RunStage(ctx, "pre-push")
    if err != nil {
        // Handle error
    }
}
```

### One-liner Usage

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Simple one-liner
    err := buildfab.RunStageSimple(ctx, ".project.yml", "pre-push", true)
    if err != nil {
        // Handle error
    }
}
```

### Running Individual Actions

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    cfg, _ := buildfab.LoadConfig(".project.yml")
    runner := buildfab.NewSimpleRunner(cfg, &buildfab.SimpleRunOptions{})
    
    // Run built-in action
    err := runner.RunAction(ctx, "version@check")
    if err != nil {
        // Handle error
    }
    
    // Run custom action
    err = runner.RunAction(ctx, "test")
    if err != nil {
        // Handle error
    }
}
```

### Running Specific Steps

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    cfg, _ := buildfab.LoadConfig(".project.yml")
    runner := buildfab.NewSimpleRunner(cfg, &buildfab.SimpleRunOptions{})
    
    // Run specific step from stage
    err := runner.RunStageStep(ctx, "pre-push", "version-check")
    if err != nil {
        // Handle error
    }
}
```

### Custom Variables and Options

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    cfg, _ := buildfab.LoadConfig(".project.yml")
    
    // Create options with custom variables
    opts := &buildfab.SimpleRunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
        Variables: map[string]string{
            "custom_var": "custom_value",
            "environment": "production",
        },
        Environment: map[string]string{
            "GO_VERSION": "1.23.1",
            "BUILD_TARGET": "linux",
        },
    }
    
    runner := buildfab.NewSimpleRunner(cfg, opts)
    err := runner.RunStage(ctx, "build")
    if err != nil {
        // Handle error
    }
}
```

## Best Practices

### Configuration Organization

1. **Use include system** for large configurations
2. **Group related actions** in separate files
3. **Use descriptive action names** that indicate their purpose
4. **Keep stage definitions** in the main configuration file

### Error Handling

1. **Use appropriate error policies**:
   - `onerror: stop` (default) for critical steps
   - `onerror: warn` for non-critical checks
2. **Provide helpful error messages** in your actions
3. **Use built-in actions** when possible for consistency

### Performance

1. **Leverage parallel execution** by structuring dependencies properly
2. **Use `--max-parallel`** to control resource usage
3. **Group related operations** in single actions when appropriate

### Platform Compatibility

1. **Use action variants** for platform-specific commands
2. **Test on multiple platforms** when possible
3. **Use built-in platform variables** for conditional logic
4. **Provide fallback variants** when appropriate

### Security

1. **Review YAML configurations** before committing
2. **Use `--dry-run`** to preview execution plans
3. **Avoid hardcoded secrets** in configuration files
4. **Use environment variables** for sensitive data

---

For more information, see:
- [Project Specification](Project-specification.md) - Complete technical specification
- [API Reference](Library.md) - Library API documentation
- [Developer Workflow](Developer-workflow.md) - Development setup and workflow
