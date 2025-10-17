# YAML Configuration Syntax Reference

This document provides a comprehensive reference for buildfab's YAML configuration syntax, including all supported fields, their types, and usage examples.

## Table of Contents

- [Configuration Structure](#configuration-structure)
- [Project Configuration](#project-configuration)
- [Include System](#include-system)
- [Actions](#actions)
- [Stages and Steps](#stages-and-steps)
- [Action Variants](#action-variants)
- [Expression Language](#expression-language)
- [Variable Interpolation](#variable-interpolation)
- [Built-in Actions](#built-in-actions)
- [Complete Examples](#complete-examples)

## Configuration Structure

A buildfab configuration file follows this basic structure:

```yaml
project:
  name: "project-name"
  modules: ["module1", "module2"]  # Optional
  bin: "bin"                       # Optional

include:                           # Optional
  - "file1.yml"
  - "patterns/*.yml"

actions:                           # Optional
  - name: "action-name"
    # Action definition

stages:                            # Optional
  stage-name:
    steps:
      - action: "action-name"
        # Step configuration
```

## Project Configuration

### Required Fields

```yaml
project:
  name: "my-project"              # Required: Project name (string)
```

### Optional Fields

```yaml
project:
  name: "my-project"              # Required
  modules:                        # Optional: Go modules list
    - "my-app"
    - "my-library"
  bin: "bin"                      # Optional: Binary directory (default: "bin")
  max_parallel: 4                 # Optional: Global concurrency limit (default: CPU count)
```

### max_parallel Field

The `max_parallel` field controls global concurrency for all step execution:

```yaml
project:
  name: "my-project"
  max_parallel: 4                 # Maximum 4 concurrent tasks
```

**Behavior**:
- **Default**: Number of CPU cores if not specified
- **Value = 0**: Uses CPU core count
- **Value > 0**: Creates global execution pool with specified size
- **Validation**: Must be >= 0

**Interaction with Matrix**:
- Steps without matrix `max_parallel` use global pool
- Steps with matrix `max_parallel` get dedicated pool
- **Effective parallelism** = `min(global_max_parallel, matrix_max_parallel)`

**Examples**:

```yaml
# Example 1: Limit total concurrency to 2
project:
  max_parallel: 2

stages:
  test:
    - action: build      # Uses global pool (max 2)
    - action: test       # Uses global pool (max 2)
    # Both steps compete for same 2 slots

# Example 2: Global + Matrix limits
project:
  max_parallel: 4       # Global limit

stages:
  test:
    - action: matrix-test
      matrix:
        values:
          platform: ["linux", "windows", "macos", "freebsd"]
        strategy:
          max_parallel: 2  # Matrix wants 2 concurrent
    # Effective = min(4, 2) = 2 concurrent jobs

# Example 3: High global, matrix self-limits
project:
  max_parallel: 10      # High global limit

stages:
  test:
    - action: matrix-test
      matrix:
        values:
          item: ["1", "2", "3"]
        strategy:
          max_parallel: 2  # Matrix restricts to 2
    # Effective = min(10, 2) = 2 concurrent jobs
```

## Include System

The include system allows you to organize configurations across multiple files:

```yaml
include:
  # Exact file paths (must exist)
  - "actions.yml"
  - "stages/ci.yml"
  - "config/common.yml"
  
  # Glob patterns (directory must exist, files optional)
  - "actions/*.yml"
  - "stages/*.yml"
  - "platforms/*.yml"
```

### Include Behavior

- **Exact paths**: File must exist or configuration fails
- **Glob patterns**: Directory must exist, files are optional
- **Merge order**: Later includes override earlier ones
- **File types**: Only `.yml` and `.yaml` files are processed
- **Circular detection**: Prevents infinite include loops

## Actions

Actions define executable units with two types: custom actions (`run`) and built-in actions (`uses`).

### Custom Actions

```yaml
actions:
  - name: "action-name"           # Required: Action name
    run: |                        # Required: Shell command (multiline string)
      echo "Hello, World!"
      go build ./...
    
    shell: "bash"                 # Optional: Shell to use (default: platform-specific)
    
    variants:                     # Optional: Action variants
      - when: "condition"
        run: "platform-specific command"
```

### Built-in Actions

```yaml
actions:
  - name: "action-name"           # Required: Action name
    uses: "git@untracked"         # Required: Built-in action identifier
    
    variants:                     # Optional: Built-in action variants
      - when: "condition"
        uses: "git@uncommitted"
```

### Shell Configuration

```yaml
actions:
  - name: "linux-action"
    shell: "bash"
    run: |
      echo "Running on Linux"
  
  - name: "windows-action"
    shell: "powershell"
    run: |
      Write-Host "Running on Windows"
  
  - name: "default-action"
    # No shell specified - uses platform default
    run: echo "Using default shell"
```

## Stages and Steps

Stages define workflows composed of steps that can reference either actions or other stages:

```yaml
stages:
  stage-name:                     # Stage identifier
    steps:
      - name: "unique-name"       # Optional: Unique name for this step (for DAG linking)
        action: "action-name"     # Option 1: Action to execute (mutually exclusive with stage)
        require: ["dep1", "dep2"] # Optional: Dependencies (list of action/stage/step names)
        onerror: "warn"           # Optional: Error policy (warn|stop, default: stop)
        only: ["label1", "label2"] # Optional: Execution labels (list)
        if: "condition"           # Optional: Conditional expression (string)
        variables:                # Optional: Step-level variable overrides (map[string]string]
          key: "value"
      
      - name: "unique-name"       # Optional: Unique name for this step
        stage: "stage-name"       # Option 2: Stage to execute (mutually exclusive with action)
        require: ["dep1"]         # Optional: Dependencies work same as actions
        onerror: "warn"           # Optional: Inherited by all steps in referenced stage
        variables:                # Optional: Variables inherited by all steps in referenced stage
          key: "value"
```

**Step Types:**
- **Action Steps**: Execute a single action defined in the `actions:` section
- **Stage Steps**: Execute all steps from another stage (composable workflows)
- Steps must have either `action` or `stage`, but not both

**Key Features:**
- **Nested Stages**: Stages can reference other stages, which can reference more stages
- **Cycle Detection**: Circular stage references and dependencies are automatically detected and rejected
- **Variable Inheritance**: Variables from stage steps are inherited by all steps in the referenced stage
- **Condition Inheritance**: `if` conditions from stage steps are combined with conditions in referenced steps
- **Error Policy Inheritance**: `onerror` settings are inherited if not overridden in referenced steps

### Step Names

The optional `name` field allows you to assign a unique identifier to a step for DAG linking. This is particularly useful when you need to use the same action multiple times within a stage:

**Default Behavior (without `name` field):**
- Steps are identified by their `action` or `stage` name
- You cannot use the same action twice in the same stage (causes duplicate name error)

**With `name` field:**
- Steps are identified by the custom name you provide
- You can use the same action multiple times with different names
- Matrix-expanded steps append matrix values to the name for uniqueness

**Example: Using the Same Action Multiple Times**

```yaml
actions:
  - name: sleep
    run: sleep ${{ duration }}

stages:
  pipeline:
    steps:
      # First use of sleep action with custom name
      - name: sleep-before-build
        action: sleep
        variables:
          duration: "2"
      
      # Build step
      - name: build
        action: build-app
        require: [sleep-before-build]  # References custom name
      
      # Second use of sleep action with different custom name
      - name: sleep-before-test
        action: sleep
        variables:
          duration: "1"
        require: [build]
      
      # Test step
      - name: test
        action: run-tests
        require: [sleep-before-test]  # References custom name
```

**Error Without `name` field:**

```yaml
stages:
  pipeline:
    steps:
      - action: sleep  # First sleep
      - action: sleep  # ERROR: Duplicate step name 'sleep'!
```

**Matrix Integration:**

When using `name` with matrix steps, the matrix values are appended to create unique names:

```yaml
stages:
  test:
    steps:
      - name: test-platform  # Custom base name
        action: run-tests
        matrix:
          values:
            os: [linux, windows, macos]
        # Creates: test-platform.linux, test-platform.windows, test-platform.macos
```

### Step Dependencies

```yaml
stages:
  build:
    steps:
      - action: "install-deps"
      - action: "compile"
        require: ["install-deps"]  # Single dependency
      - action: "test"
        require: ["compile"]       # Single dependency
      - action: "package"
        require: ["compile"]       # Parallel with test
      - action: "deploy"
        require: ["test", "package"] # Multiple dependencies
```

### Error Policies

```yaml
stages:
  checks:
    steps:
      - action: "critical-check"
        onerror: "stop"           # Default: Stop on error
      - action: "warning-check"
        onerror: "warn"           # Continue on error with warning
```

### Label-Based Execution

```yaml
stages:
  release:
    steps:
      - action: "build"
      - action: "test"
        only: ["release"]         # Only run with release label
      - action: "deploy"
        only: ["release", "production"] # Multiple labels (AND logic)
```

### Conditional Execution

```yaml
stages:
  test:
    steps:
      - action: "unit-tests"
      - action: "integration-tests"
        if: "env.TEST_LEVEL == 'integration'"
      - action: "e2e-tests"
        if: "os == 'linux' && cpu >= 4"
```

### Stage References

Steps can reference other stages to create reusable, composable workflows:

```yaml
stages:
  # Define reusable stage components
  git-checks:
    steps:
      - action: "check-untracked"
      - action: "check-uncommitted"
  
  build-checks:
    steps:
      - action: "check-version"
      - action: "lint"
  
  # Compose workflows from reusable stages
  pre-push:
    steps:
      - stage: "git-checks"        # Include all git checks
      - stage: "build-checks"      # Include all build checks
      - action: "run-tests"        # Add additional actions
```

**Nested Stage References:**

```yaml
stages:
  unit-tests:
    steps:
      - action: "test-parser"
      - action: "test-executor"
  
  integration-tests:
    steps:
      - action: "test-api"
      - action: "test-cli"
  
  all-tests:
    steps:
      - stage: "unit-tests"        # Nested: includes test-parser, test-executor
      - stage: "integration-tests" # Nested: includes test-api, test-cli
  
  ci-pipeline:
    steps:
      - action: "build"
      - stage: "all-tests"         # Multi-level nesting
      - action: "package"
```

**Stage References with Variables:**

```yaml
stages:
  docker-build:
    steps:
      - action: "build-image"
      - action: "push-image"
  
  build-production:
    steps:
      - stage: "docker-build"
        variables:
          REGISTRY: "registry.prod.com"
          TAG: "latest"
  
  build-staging:
    steps:
      - stage: "docker-build"
        variables:
          REGISTRY: "registry.staging.com"
          TAG: "dev"
```

**Stage References with Dependencies:**

```yaml
stages:
  prepare:
    steps:
      - action: "setup"
      - action: "install-deps"
  
  build:
    steps:
      - action: "compile"
  
  test:
    steps:
      - action: "unit-test"
      - action: "integration-test"
  
  full-workflow:
    steps:
      - stage: "prepare"
      - stage: "build"
        require: ["setup", "install-deps"]  # Depends on prepare stage steps
      - stage: "test"
        require: ["compile"]                # Depends on build stage steps
```

**Cycle Detection:**

```yaml
# This configuration will be rejected with a clear error message:
stages:
  stage-a:
    steps:
      - stage: "stage-b"  # ❌ Creates cycle
  
  stage-b:
    steps:
      - stage: "stage-c"
  
  stage-c:
    steps:
      - stage: "stage-a"  # ❌ Completes the cycle

# Error: circular stage reference detected (cycle: [stage-a, stage-b, stage-c, stage-a])
```

### Step-Level Variables

Step-level variables allow you to override global variables or provide step-specific values:

```yaml
stages:
  build:
    steps:
      - action: "build-image"
        variables:
          image: "registry.svc/burnes/production"
          tag: "v1.0.0"
      
      - action: "build-image"
        variables:
          image: "registry.svc/burnes/development"
          tag: "latest"
```

**Behavior**:
- Step variables override global variables with the same name
- Step variables are temporary and only apply during that specific step execution
- Global variables are restored after step completion
- Works with both regular steps and matrix steps

**Use Cases**:
- Override matrix variables for specific steps
- Provide step-specific configuration
- Reuse actions with different parameters
- Configure multiple environments in a single stage

**Example with Matrix**:
```yaml
stages:
  images-build:
    steps:
      - action: build-image
        matrix:
          values:
            image: [
              "registry.svc/clr-distr-centos8",
              "registry.svc/clr-distr-centos7"
            ]
        variables:
          tag: "latest"
          registry: "registry.svc"
      
      - action: test-image
        require: ["build-image"]
        variables:
          image: "registry.svc/clr-distr-centos8"
          tag: "latest"
```

## Action Variants

Action variants allow platform-specific or condition-specific execution:

```yaml
actions:
  - name: "build"
    variants:
      - when: "${{ platform == 'linux' }}"    # Condition (required)
        run: |                                 # Command (required)
          echo "Building for Linux"
          cmake -S . -B build && cmake --build build
        shell: "bash"                          # Optional: Shell override
        
      - when: "${{ platform == 'windows' }}"
        shell: "powershell"
        run: |
          Write-Host "Building for Windows"
          cmake -S . -B build -G "Visual Studio 17 2022"
          
      - when: "${{ platform == 'darwin' }}"
        run: |
          echo "Building for macOS"
          cmake -S . -B build -G "Xcode"
```

### Variant Selection

- **First match wins**: First variant with true condition is selected
- **No match**: Action is skipped with status "SKIP"
- **Mixed types**: Variants can use different execution types

```yaml
actions:
  - name: "git-check"
    variants:
      - when: "${{ os == 'linux' }}"
        uses: "git@untracked"      # Built-in action
      - when: "${{ os == 'windows' }}"
        run: "git status --porcelain | findstr \"^??\""  # Custom command
```

## Expression Language

buildfab supports a powerful expression language for conditions and variable interpolation.

### Variables

#### Platform Variables
```yaml
# Available platform variables
${{ platform }}      # linux, windows, darwin
${{ os }}            # ubuntu, debian, windows, darwin
${{ arch }}          # amd64, arm64
${{ os_version }}    # 24.04, 15.0, windows10
${{ cpu }}           # Number of CPU cores
```

#### Version Variables
```yaml
# Available version variables from version-go library
${{ version.version }}        # Current version (e.g., "v1.2.3")
${{ version.rawversion }}     # Raw version string (e.g., "v0.22.0-1-g1234abc")
${{ version.project }}        # Project name from .project.yml or go.mod
${{ version.commit }}         # Git commit hash
${{ version.date }}           # Build date
${{ version.type }}           # Version type (release, prerelease, etc.)
${{ version.major }}          # Major version number
${{ version.minor }}          # Minor version number
${{ version.patch }}          # Patch version number
${{ version.build-type }}     # CMake build type (Release, Debug, etc.)
${{ version.version-type }}   # Semantic version type (release, prerelease, etc.)
${{ version.modules }}        # Comma-separated list of Go modules
${{ version.tag }}            # Current Git tag (latest tag on branch)
${{ version.branch }}         # Current Git branch name
```

#### Environment Variables
```yaml
${{ env.VAR_NAME }}  # Environment variable
${{ env.PATH }}      # System PATH
${{ env.HOME }}      # User home directory
```

#### Input Variables
```yaml
${{ inputs.name }}   # Input variable
${{ inputs.version }} # Input version
```

#### Matrix Variables
```yaml
${{ matrix.os }}     # Matrix OS value
${{ matrix.arch }}   # Matrix architecture value
```

#### Boolean Variables
```yaml
${{ ci }}            # true if running in CI
${{ branch }}        # Current git branch
```

### Operators

#### Comparison Operators
```yaml
# Equality
if: "os == 'linux'"
if: "arch != 'arm64'"

# Numeric comparison
if: "cpu >= 4"
if: "cpu < 8"
if: "os_version <= '20.04'"
if: "os_version > '18.04'"
```

#### Logical Operators
```yaml
# AND operator
if: "os == 'linux' && arch == 'amd64'"
if: "platform == 'windows' && cpu >= 4"

# OR operator
if: "platform == 'windows' || platform == 'darwin'"
if: "os == 'ubuntu' || os == 'debian'"

# NOT operator
if: "!(os == 'windows')"
if: "!(platform == 'darwin' && arch == 'arm64')"

# Parentheses for grouping
if: "(os == 'linux' || os == 'darwin') && cpu >= 2"
if: "!(os == 'windows' || os == 'darwin')"
```

### Helper Functions

#### String Functions
```yaml
# Contains function
if: "contains(os, 'ubuntu')"
if: "contains(platform, 'linux')"

# Starts with function
if: "startsWith(arch, 'arm')"
if: "startsWith(os_version, '20')"

# Ends with function
if: "endsWith(os_version, '.04')"
if: "endsWith(platform, 'nix')"

# Matches function (regex)
if: "matches(platform, 'linux|darwin')"
if: "matches(os, 'ubuntu|debian')"
```

#### File System Functions
```yaml
# File exists function
if: "fileExists('package.json')"
if: "fileExists('CMakeLists.txt')"
if: "fileExists('go.mod')"
```

#### Version Functions
```yaml
# Semantic version comparison
if: "semverCompare(os_version, '>=20.04')"
if: "semverCompare(version, '>=1.2.0')"
if: "semverCompare(os_version, '<22.04')"
```

## Variable Interpolation

Variables can be interpolated in action commands using `${{ }}` syntax:

### Error Handling

When using undefined variables, buildfab provides clear error messages:

```yaml
# This will produce an error:
actions:
  - name: "example"
    run: echo "Value: ${{ undefined.variable }}"

# Error output:
# undefined variables: undefined.variable
# available variables: platform, arch, os, os_version, cpu, version.version, version.project, ...
```

### Basic Interpolation

```yaml
actions:
  - name: "platform-info"
    run: |
      echo "Platform: ${{ platform }}"
      echo "Architecture: ${{ arch }}"
      echo "OS: ${{ os }}"
      echo "CPU cores: ${{ cpu }}"
```

### Multi-line Commands

```yaml
actions:
  - name: "build"
    run: |
      echo "Building for ${{ platform }} on ${{ os }} ${{ os_version }}"
      echo "Using ${{ cpu }} CPU cores"
      
      # Platform-specific build commands
      if [ "${{ platform }}" = "linux" ]; then
        make build-linux
      elif [ "${{ platform }}" = "darwin" ]; then
        make build-macos
      fi
```

### Environment Variable Interpolation

```yaml
actions:
  - name: "deploy"
    run: |
      echo "Deploying to ${{ env.ENVIRONMENT }}"
      echo "Using version ${{ env.VERSION }}"
      kubectl set image deployment/web-app web-app=myapp:${{ env.VERSION }}
```

### Version Variable Interpolation

```yaml
actions:
  - name: "build-info"
    run: |
      echo "Project: ${{ version.project }}"
      echo "Version: ${{ version.version }}"
      echo "Build Type: ${{ version.build-type }}"
      echo "Version Type: ${{ version.version-type }}"
      echo "Modules: ${{ version.modules }}"
      echo "Current Tag: ${{ version.tag }}"
      echo "Current Branch: ${{ version.branch }}"
      echo "Commit: ${{ version.commit }}"
  
  - name: "build-with-version"
    run: |
      echo "Building ${{ version.project }} version ${{ version.version }}"
      echo "Branch: ${{ version.branch }}, Tag: ${{ version.tag }}"
      go build -ldflags "-X main.version=${{ version.version }} -X main.buildType=${{ version.build-type }} -X main.branch=${{ version.branch }}"
  
  - name: "conditional-build"
    variants:
      - when: "${{ version.build-type == 'Release' }}"
        run: |
          echo "Building release version ${{ version.version }} on branch ${{ version.branch }}"
          go build -ldflags "-s -w"
      - when: "${{ version.build-type == 'Debug' }}"
        run: |
          echo "Building debug version ${{ version.version }} on branch ${{ version.branch }}"
          go build -race -gcflags="all=-N -l"
  
  - name: "branch-specific-build"
    variants:
      - when: "${{ version.branch == 'master' || version.branch == 'main' }}"
        run: |
          echo "Building production version ${{ version.version }}"
          go build -ldflags "-s -w -X main.environment=production"
      - when: "${{ version.branch != 'master' && version.branch != 'main' }}"
        run: |
          echo "Building development version ${{ version.version }} on branch ${{ version.branch }}"
          go build -ldflags "-X main.environment=development -X main.branch=${{ version.branch }}"
```

## Built-in Actions

### Git Actions

```yaml
actions:
  - name: "git-untracked"
    uses: "git@untracked"        # Fail if untracked files present
  
  - name: "git-uncommitted"
    uses: "git@uncommitted"      # Fail if staged/unstaged changes present
  
  - name: "git-modified"
    uses: "git@modified"         # Warn if working tree differs from HEAD
    onerror: "warn"              # Recommended for git@modified
```

### Version Actions

```yaml
actions:
  - name: "version-check"
    uses: "version@check"        # Validate version format in VERSION file
  
  - name: "version-greatest"
    uses: "version@check-greatest" # Ensure current version is greatest tag
```

### Built-in Action Usage

```yaml
# In stage steps
stages:
  pre-push:
    steps:
      - action: "git-untracked"
      - action: "version-check"

# As standalone actions
actions:
  - name: "git-check"
    uses: "git@untracked"
  
  - name: "version-validation"
    uses: "version@check"
```

## Complete Examples

### Simple Project

```yaml
project:
  name: "hello-world"
  modules: ["hello"]
  bin: "bin"

actions:
  - name: "test"
    run: |
      go test ./...
  
  - name: "build"
    run: |
      go build -o bin/hello ./cmd/hello
  
  - name: "git-check"
    uses: "git@untracked"

stages:
  pre-push:
    steps:
      - action: "test"
      - action: "build"
      - action: "git-check"
```

### Cross-Platform Build

```yaml
project:
  name: "cross-platform-app"

actions:
  - name: "build"
    variants:
      - when: "${{ platform == 'linux' }}"
        run: |
          echo "Building for Linux ${{ arch }}..."
          GOOS=linux GOARCH=${{ arch }} go build -o bin/app-linux-${{ arch }} ./cmd/app
      
      - when: "${{ platform == 'windows' }}"
        run: |
          echo "Building for Windows ${{ arch }}..."
          GOOS=windows GOARCH=${{ arch }} go build -o bin/app-windows-${{ arch }}.exe ./cmd/app
      
      - when: "${{ platform == 'darwin' }}"
        run: |
          echo "Building for macOS ${{ arch }}..."
          GOOS=darwin GOARCH=${{ arch }} go build -o bin/app-darwin-${{ arch }} ./cmd/app
  
  - name: "test"
    run: |
      go test ./...
  
  - name: "package"
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
      - action: "build"
      - action: "test"
      - action: "package"
```

### Modular Configuration

**Main file (`project.yml`)**:
```yaml
project:
  name: "modular-project"

include:
  - "actions/test.yml"
  - "actions/build.yml"
  - "stages/ci.yml"

stages:
  main:
    steps:
      - action: "test"
      - action: "build"
```

**Actions file (`actions/test.yml`)**:
```yaml
actions:
  - name: "test"
    run: |
      go test ./...
  
  - name: "test-coverage"
    run: |
      go test -cover ./...
      go tool cover -html=coverage.out -o coverage.html
```

**Actions file (`actions/build.yml`)**:
```yaml
actions:
  - name: "build"
    run: |
      go build ./...
  
  - name: "build-static"
    run: |
      go build -ldflags="-s -w" ./...
```

**Stages file (`stages/ci.yml`)**:
```yaml
stages:
  ci:
    steps:
      - action: "test"
      - action: "build"
      - action: "test-coverage"
```

### Complex Conditional Pipeline

```yaml
project:
  name: "complex-pipeline"

actions:
  - name: "unit-tests"
    run: |
      go test ./... -short
  
  - name: "integration-tests"
    run: |
      go test ./... -tags=integration
    if: "contains(env.TEST_LEVEL, 'integration')"
  
  - name: "e2e-tests"
    run: |
      go test ./... -tags=e2e
    if: "contains(env.TEST_LEVEL, 'e2e')"
  
  - name: "performance-tests"
    run: |
      go test ./... -tags=performance -bench=.
    if: "contains(env.TEST_LEVEL, 'performance')"
  
  - name: "security-scan"
    variants:
      - when: "${{ os == 'linux' }}"
        run: |
          docker run --rm -v $(pwd):/app securecodewarrior/docker-security-scanner /app
      - when: "${{ os == 'windows' }}"
        run: |
          powershell -Command "Invoke-WebRequest -Uri 'https://security-scanner.exe' -OutFile 'scanner.exe'; .\scanner.exe"
  
  - name: "coverage-report"
    run: |
      go test ./... -coverprofile=coverage.out
      go tool cover -html=coverage.out -o coverage.html
    if: "env.COVERAGE == 'true'"

stages:
  test:
    steps:
      - action: "unit-tests"
      - action: "integration-tests"
      - action: "e2e-tests"
      - action: "performance-tests"
      - action: "security-scan"
      - action: "coverage-report"
```

## Validation Rules

### Required Fields
- `project.name` - Project name must be specified
- `actions[].name` - Action name must be specified
- `actions[].run` or `actions[].uses` - Action must have execution method
- `stages[].steps[].action` - Step must reference an action

### Validation Rules
- Action names must be unique within a configuration
- Stage names must be unique within a configuration
- Referenced actions in steps must exist
- Dependencies in `require` must reference existing actions
- No circular dependencies allowed
- Include files must exist (for exact paths)
- Include directories must exist (for glob patterns)

### Error Handling
- Configuration validation errors result in exit code 2
- Missing actions or circular dependencies are caught during validation
- Include file errors are reported during configuration loading
- Expression syntax errors are reported during evaluation

---

For more information, see:
- [Features and Examples](Features-and-examples.md) - Comprehensive features documentation
- [Project Specification](Project-specification.md) - Complete technical specification
- [API Reference](Library.md) - Library API documentation
