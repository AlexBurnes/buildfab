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

Stages define workflows composed of steps that reference actions:

```yaml
stages:
  stage-name:                     # Stage identifier
    steps:
      - action: "action-name"     # Required: Action to execute
        require: ["dep1", "dep2"] # Optional: Dependencies (list of action names)
        onerror: "warn"           # Optional: Error policy (warn|stop, default: stop)
        only: ["label1", "label2"] # Optional: Execution labels (list)
        if: "condition"           # Optional: Conditional expression (string)
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
