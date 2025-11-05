# Technical Specification: buildfab

**Project:** `buildfab`
**Language:** Go
**Scope:** CLI utility and library for building and running project automation stages and actions defined in a YAML configuration file.

## 1) Overview

Buildfab is a Go library and CLI utility that provides a flexible framework for executing project automation stages and actions defined in a `project.yml` configuration file. It serves as the underlying engine for tools like `pre-push` that need to run complex, dependency-aware automation workflows.

### Core Features
- **YAML-driven configuration**: Define stages and actions in `project.yml`
- **Hierarchical DAG execution**: Job-based parallel execution with dependency resolution (v0.32.0)
- **Built-in action registry**: Extensible system for common automation tasks
- **Custom action support**: Execute shell commands and external tools
- **Variable interpolation**: Dynamic configuration with `${{ }}` syntax
- **Matrix execution**: Run actions and stages across multiple configurations (compilers, platforms, versions)
- **Matrix on stages**: Apply matrix to entire stages for cross-compiler builds and multi-platform testing
- **Container support**: Native Docker/Podman integration with resource management
- **Error policy management**: Configurable stop/warn behavior per step
- **Cross-platform compatibility**: Linux, Windows, macOS (amd64/arm64)

### Execution Model (v0.32.0)

buildfab uses a **hierarchical DAG architecture** for executing workflows:

#### Job-Based Execution

- **Jobs**: Matrix combinations and non-matrix steps form individual job nodes
- **Steps**: Each job contains an ordered sequence of steps to execute
- **Parallelism**: Jobs execute in parallel waves based on dependencies
- **Sequential**: Steps within a job execute sequentially
- **Dependencies**: Jobs depend on other jobs (not individual steps)

#### Execution Waves

Jobs are organized into dependency-ordered waves:

1. **Wave 1**: All jobs with no dependencies start first
2. **Wave 2**: Once Wave 1 completes, jobs depending only on Wave 1 start
3. **Wave N**: Process continues until all jobs complete

Within each wave, jobs execute in parallel (respecting `max_parallel` limit).

#### Matrix Build Execution

When a step has a matrix configuration:
- Each matrix combination creates one **job**
- Each job runs the same sequence of **steps**
- Jobs execute in parallel waves
- Sliding window dependencies control concurrency (via `max_parallel`)

**Example**:
```yaml
stages:
  test:
    steps:
      - action: test-platform
        matrix:
          values:
            platform: [linux, darwin, windows]
          strategy:
            max_parallel: 2
# Creates 3 jobs: linux, darwin, windows
# Only 2 run concurrently (3rd waits for first to complete)
# All jobs run the same action: test-platform
```

#### Nested Matrix Support

Matrix on stage references expands hierarchically:

```yaml
stages:
  build:
    steps:
      - action: configure
      - action: compile
      - action: test

  cross-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: [gcc, clang]
# Creates 2 jobs: compiler=gcc, compiler=clang
# Each job runs sequentially: configure → compile → test
# Both jobs execute in parallel
```

**Benefits of Hierarchical DAG** (v0.32.0):
- ✅ Better handling of nested matrices (matrix on stage reference)
- ✅ Improved parallelism control with sliding window dependencies
- ✅ No more hanging scenarios in complex matrix builds
- ✅ Clearer execution model (jobs are the unit of parallelism)
- ✅ Proper dependency resolution between matrix combinations

#### Sliding Window Dependencies

When `max_parallel` is set on a matrix, buildfab automatically injects sliding window dependencies:

```yaml
stages:
  build:
    steps:
      - action: build-variant
        matrix:
          values:
            variant: [v1, v2, v3, v4, v5]
          strategy:
            max_parallel: 2
# Execution pattern:
# Wave 1: v1, v2 start
# Wave 2: v3 starts when v1 completes
# Wave 3: v4 starts when v2 completes
# Wave 4: v5 starts when v3 completes
```

This prevents resource exhaustion while maintaining throughput.

#### Condition-Based Skips

Condition skips (`if` conditions) don't block sliding window dependencies:

```yaml
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
# - It doesn't block centos8-gcc from starting (sliding window continues)
# - Explicit user dependencies would still wait for it
```

**Key Distinction**:
- **Sliding window dependencies**: Auto-injected for parallelism control (condition skips don't block)
- **User dependencies** (`require`, `depends_on`): Explicit dependencies (condition skips do block)

## 2) Library API

### Primary Interface

```go
package buildfab

import "context"

// RunStage executes a specific stage from project.yml configuration
func RunStage(ctx context.Context, stageName string, opts *RunOptions) error

// RunOptions configures stage execution
type RunOptions struct {
    ConfigPath    string            // Path to project.yml (default: ".project.yml")
    MaxParallel   int               // Maximum parallel execution (default: CPU count)
    Verbose       bool              // Enable verbose output
    Debug         bool              // Enable debug output
    Variables     map[string]string // Additional variables for interpolation
    WorkingDir    string            // Working directory for execution
    Output        io.Writer         // Output writer (default: os.Stdout)
    ErrorOutput   io.Writer         // Error output writer (default: os.Stderr)
}

// StageResult contains execution results for a stage
type StageResult struct {
    StageName string
    Success   bool
    Steps     []StepResult
    Duration  time.Duration
    Error     error
}

// StepResult contains execution results for a step
type StepResult struct {
    StepName   string
    ActionName string
    Status     StepStatus
    Duration   time.Duration
    Output     string
    Error      error
}

type StepStatus int

const (
    StepStatusPending StepStatus = iota
    StepStatusRunning
    StepStatusOK
    StepStatusWarn
    StepStatusError
    StepStatusSkipped
)
```

### Action Interface

```go
// Action defines the interface for executable actions
type Action interface {
    Run(ctx context.Context, opts *ActionOptions) error
    GetName() string
    GetHelp() string
    GetRepro() string
}

// ActionOptions provides context for action execution
type ActionOptions struct {
    Variables   map[string]string
    WorkingDir  string
    Verbose     bool
    Debug       bool
    Output      io.Writer
    ErrorOutput io.Writer
}
```

## 3) Configuration Format: project.yml

### Structure

```yaml
project:
  name: "project-name"
  modules: ["module1", "module2"]
  bin: "bin"  # Optional: directory for module binaries

actions:
  - name: action-name
    run: |
      shell command
      multi-line supported
    # OR
    uses: builtin@action-type
    
  - name: another-action
    run: echo "Hello ${{ variable }}"
    only: [release, prerelease]  # Optional: run only for specific version types
    if: "condition expression"   # Optional: conditional execution

stages:
  stage-name:
    steps:
      - action: action-name
        require: [dependency1, dependency2]  # Optional: dependencies
        onerror: warn  # Optional: stop (default) or warn
        only: [release]  # Optional: version type filter
        if: "condition"  # Optional: conditional execution
        variables:  # Optional: step-level variable overrides
          key: "value"
```

### Variable Interpolation

Variables available in `${{ }}` syntax:
- **Git state**: `tag`, `branch` (current repository state)
- **Version info**: `version.version`, `version.rawversion`, `version.project`, `version.module`, `version.modules`
- **Custom variables**: Provided via `RunOptions.Variables`
- **Step variables**: Defined in `step.variables` field (override global variables for that step)
- **Matrix variables**: Defined in matrix configuration (e.g., `matrix.image`, `matrix.os`)
- **Environment**: `env.VAR_NAME` for environment variables

#### Default Values

You can provide fallback values for missing variables using a dash `-` separator inside the expression:

- Literal default: `${{ variable-default }}`
- Default from another variable: `${{ variable-other.variable }}`

Examples:

```yaml
actions:
  - name: echo-compiler
    run: |
      echo "Compiler: ${{ matrix.compiler-\"gcc\" }}"
      echo "Default from var: ${{ matrix.compiler-variable.compiler_default }}"
```

Rules:
- If `variable` is defined, its value is used.
- If undefined and a default is provided, the default is used.
- Defaults may be quoted (single or double) and quotes are stripped.
- Defaults may reference another variable; that reference is resolved.
- If no default is provided and the variable is undefined, interpolation fails with a clear error listing available variables.

#### Variable Precedence

When the same variable name is defined at multiple levels, the following precedence applies (highest to lowest):
1. **Step variables** - Defined in `step.variables`
2. **Matrix variables** - Generated during matrix expansion
3. **Global variables** - Provided via `RunOptions.Variables`
4. **Built-in variables** - Platform, version, environment variables

### Built-in Actions

#### Git Actions
- `git@untracked` - Fail if untracked files present
- `git@uncommitted` - Fail if staged/unstaged changes present  
- `git@modified` - Fail if working tree differs from HEAD

#### Version Actions
- `version@check` - Validate version format and consistency
- `version@check-greatest` - Ensure current version is greatest

#### Custom Actions
- `run:` - Execute shell commands with full variable interpolation
- Support for any external tool or script execution

## 4) Library Architecture

### Package Structure

```
buildfab/
├── cmd/buildfab/           # CLI application
│   └── main.go
├── pkg/buildfab/           # Public API
│   ├── buildfab.go         # Main API functions
│   ├── types.go            # Public types and interfaces
│   └── errors.go           # Error types
├── internal/
│   ├── config/             # YAML parsing and validation
│   ├── executor/           # DAG execution engine
│   ├── actions/            # Built-in action implementations
│   ├── variables/          # Variable interpolation
│   └── ui/                 # Output formatting and display
└── examples/               # Usage examples
```

### Core Components

#### 1. Configuration Parser (`internal/config/`)
- YAML file parsing and validation
- Schema validation for project.yml format
- Variable resolution and interpolation
- Action and stage definition parsing

#### 2. DAG Executor (`internal/executor/`)
- Dependency graph construction
- Cycle detection and validation
- Parallel execution scheduling
- Error policy enforcement
- Result aggregation and reporting

#### 3. Action Registry (`internal/actions/`)
- Built-in action implementations
- Action discovery and instantiation
- Custom action execution (run: commands)
- Action result formatting

#### 4. Variable System (`internal/variables/`)
- Variable interpolation engine
- Git state detection (tag, branch)
- Version information integration
- Custom variable support

#### 5. UI System (`internal/ui/`)
- Colored output formatting
- Progress indication
- Error reporting and reproduction hints
- Verbose and debug output modes

## 5) CLI Interface

### Commands

```bash
buildfab [options] [command]

Commands:
  run <stage>     Run a specific stage from project.yml
  list-actions    List available built-in actions
  validate        Validate project.yml configuration
  version         Show version information

Options:
  -c, --config string    Path to project.yml (default: ".project.yml")
  -j, --max-parallel int Maximum parallel execution (default: CPU count)
  -v, --verbose          Enable verbose output
  -d, --debug            Enable debug output
  -w, --working-dir string Working directory
  -h, --help             Show help information
  -V, --version          Show version only
```

### Example Usage

```bash
# Run pre-push stage
buildfab run pre-push

# Run with custom config and verbose output
buildfab -c my-project.yml -v run pre-push

# List available actions
buildfab list-actions

# Validate configuration
buildfab validate
```

## 6) Integration with pre-push

The pre-push utility will use buildfab as its execution engine:

```go
package main

import (
    "context"
    "os"
    "github.com/user/buildfab"
)

func main() {
    ctx := context.Background()
    
    opts := &buildfab.RunOptions{
        ConfigPath:  ".project.yml",
        Verbose:     true,
        WorkingDir:  ".",
    }
    
    err := buildfab.RunStage(ctx, "pre-push", opts)
    if err != nil {
        os.Exit(1)
    }
}
```

## 7) Error Handling

### Error Policies
- **stop** (default): Halt execution on step failure
- **warn**: Continue execution but mark step as warning

### Error Types
- **ConfigurationError**: Invalid project.yml format or content
- **ExecutionError**: Step execution failure
- **DependencyError**: Circular dependencies or missing dependencies
- **VariableError**: Unresolved variable interpolation

### Error Reporting
- Clear error messages with context
- Reproduction hints for failed actions
- Step-by-step execution trace in debug mode
- Exit codes: 0 (success), 1 (error), 2 (configuration error)

## 8) Testing Strategy

### Test Types
- **Unit tests**: Individual component testing
- **Integration tests**: End-to-end stage execution
- **E2E tests**: Full workflow testing with temporary projects
- **Race detection**: Concurrency safety validation

### Test Coverage
- Configuration parsing and validation
- DAG construction and execution
- Variable interpolation
- Action execution (built-in and custom)
- Error handling and policies
- CLI interface and output formatting

## 9) Build and Packaging

**Note**: Build and packaging for buildfab will follow the same system as the pre-push project:

### Build System
- **CMake**: Cross-platform build configuration
- **Conan**: Go toolchain and dependency management
- **GoReleaser**: Automated release and packaging
- **GitHub Actions**: CI/CD pipeline

### Packaging Targets
- **Linux**: tar.gz archives with install.sh script
- **Windows**: Scoop manifest for package manager
- **macOS**: Homebrew formula
- **Cross-platform**: Static binaries (amd64/arm64)

### Release Process
1. Version bump in VERSION file
2. Update CHANGELOG.md
3. Create git tag
4. Push to repository
5. GoReleaser builds and publishes releases
6. Package managers updated automatically

## 10) Dependencies

### Core Dependencies
- `gopkg.in/yaml.v3`: YAML configuration parsing
- `golang.org/x/sync/errgroup`: Parallel execution management
- `github.com/spf13/cobra`: CLI framework (optional)

### Development Dependencies
- `golangci-lint`: Code linting and quality checks
- `go test`: Testing framework
- `gofmt`: Code formatting
- `goimports`: Import organization

### Build Dependencies
- `conanfile-golang.py`: Go toolchain via Conan
- `CMakeLists.txt`: Cross-platform build configuration
- `.goreleaser.yml`: Release automation

## 11) Performance Considerations

### Optimization Targets
- **Fast startup**: Minimal initialization overhead
- **Efficient execution**: Parallel processing where possible
- **Memory usage**: Stream processing for large outputs
- **I/O efficiency**: Minimize file system operations

### Scalability
- Support for large dependency graphs
- Efficient parallel execution scheduling
- Memory-efficient variable interpolation
- Streaming output for long-running actions

## 12) Security Considerations

### Input Validation
- YAML file validation and sanitization
- Variable interpolation safety
- Command execution sandboxing
- Path traversal prevention

### Safe Execution
- No arbitrary code execution in built-in actions
- Command injection prevention
- Secure file handling
- Environment variable sanitization

## 13) Future Extensions

### Planned Features
- **Multi-dimensional matrix**: Support multiple matrix dimensions simultaneously
- **Conditional execution**: Advanced condition expressions (currently supports basic if conditions)
- **Action composition**: Reusable action definitions (currently via stage references)
- **Plugin system**: External action registration
- **Webhook integration**: Remote trigger support
- **Metrics collection**: Execution timing and statistics

### API Evolution
- Backward compatibility for major versions
- Deprecation warnings for removed features
- Clear migration paths for breaking changes
- Comprehensive documentation for all changes

## 14) Documentation Requirements

### Required Documentation
- **README.md**: Installation, usage, and examples
- **API documentation**: Complete GoDoc comments
- **Configuration reference**: project.yml format specification
- **Migration guide**: Upgrading between versions
- **CHANGELOG.md**: Version history and changes

### Example Projects
- **Basic usage**: Simple project.yml examples
- **Advanced workflows**: Complex dependency scenarios
- **Integration examples**: Using with other tools
- **Custom actions**: Creating project-specific actions

This specification provides a comprehensive foundation for the buildfab library that will serve as the execution engine for pre-push and other automation tools requiring flexible, dependency-aware workflow execution.