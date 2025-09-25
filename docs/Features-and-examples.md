# Features and Examples

This document provides comprehensive documentation of buildfab features with detailed examples and usage patterns.

## Table of Contents

- [Core Features](#core-features)
- [YAML Configuration Syntax](#yaml-configuration-syntax)
- [Action Variants](#action-variants)
- [Conditional Execution](#conditional-execution)
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
