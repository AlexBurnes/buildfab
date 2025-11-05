# Buildfab Library API Documentation

## Overview

Buildfab is a Go library for building and executing DAG-based workflows with matrix builds, parallel execution, and advanced features. This document describes the public API for using buildfab as a library in your own Go applications.

**Current Version**: v0.32.0

## Table of Contents

1. [Quick Start](#quick-start)
2. [Core Concepts](#core-concepts)
3. [API Reference](#api-reference)
   - [Configuration](#configuration-api)
   - [Execution](#execution-api)
   - [Variables](#variables-api)
4. [Examples](#examples)
5. [Migration Guide](#migration-guide)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    // 1. Load configuration
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }
    
    // 2. Create execution options
    opts := buildfab.DefaultSimpleRunOptions()
    opts.VerboseLevel = 1
    opts.Debug = false
    opts.WorkingDir = "."
    opts.Output = os.Stdout
    opts.ErrorOutput = os.Stderr
    
    // 3. Add variables for interpolation
    opts.Variables = map[string]string{
        "version": "v1.0.0",
        "platform": "linux-amd64",
    }
    
    // 4. Create runner
    runner := buildfab.NewSimpleRunner(config, opts)
    
    // 5. Execute stage
    ctx := context.Background()
    if err := runner.RunStage(ctx, "build"); err != nil {
        fmt.Fprintf(os.Stderr, "Stage failed: %v\n", err)
        os.Exit(1)
    }
}
```

## Core Concepts

### Configuration

Buildfab uses YAML configuration files (`.project.yml`) to define:
- **Project**: Name, modules, settings
- **Actions**: Individual commands or built-in actions
- **Stages**: Collections of steps with dependencies

### Execution Model

**Hierarchical DAG (v0.32.0+)**:
- Matrix builds expand into **jobs** (groups of steps)
- Each job contains **steps** (sequential actions)
- Jobs execute in parallel waves with dependency resolution
- Steps within a job execute sequentially
- Supports nested matrices (matrix on stage references)

### Variable Interpolation

Variables are available in all run commands and conditions:
- Platform variables: `os`, `arch`, `cpu`
- Version variables: `version`, `project`, `module`
- Custom variables: any key-value pairs
- Environment variables: `env.*`
- Matrix variables: `matrix.*`

## API Reference

### Configuration API

#### LoadConfig

```go
func LoadConfig(path string) (*Config, error)
```

Loads and parses a YAML configuration file with include support.

**Parameters**:
- `path`: Path to configuration file (e.g., ".project.yml")

**Returns**:
- `*Config`: Parsed configuration
- `error`: Parse or validation error

**Features**:
- Recursive include processing
- Configuration validation
- Duplicate detection
- Cycle detection

**Example**:
```go
config, err := buildfab.LoadConfig(".project.yml")
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Validate configuration
if err := config.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}
```

#### Config Methods

##### GetStage

```go
func (c *Config) GetStage(name string) (Stage, bool)
```

Retrieves a stage by name.

**Parameters**:
- `name`: Stage name

**Returns**:
- `Stage`: Stage configuration
- `bool`: true if stage exists

**Example**:
```go
stage, exists := config.GetStage("build")
if !exists {
    return fmt.Errorf("stage not found: build")
}
```

##### GetAction

```go
func (c *Config) GetAction(name string) (Action, bool)
```

Retrieves an action by name.

**Parameters**:
- `name`: Action name

**Returns**:
- `Action`: Action configuration
- `bool`: true if action exists

**Example**:
```go
action, exists := config.GetAction("test")
if !exists {
    return fmt.Errorf("action not found: test")
}
```

### Execution API

#### DefaultSimpleRunOptions

```go
func DefaultSimpleRunOptions() *SimpleRunOptions
```

Creates default execution options.

**Returns**:
- `*SimpleRunOptions`: Default options

**Default Values**:
- `VerboseLevel`: 0 (quiet)
- `Debug`: false
- `WorkingDir`: "."
- `Output`: os.Stdout
- `ErrorOutput`: os.Stderr
- `Variables`: empty map

**Example**:
```go
opts := buildfab.DefaultSimpleRunOptions()
opts.VerboseLevel = 1  // Basic verbosity
opts.Debug = true      // Enable debug output
```

#### SimpleRunOptions

```go
type SimpleRunOptions struct {
    ConfigPath         string            // Path to config file
    MaxParallel        int               // Max parallel jobs (0=CPU count)
    VerboseLevel       int               // 0=quiet, 1=basic, 2=detailed, 3=max
    Debug              bool              // Enable debug output
    DryRun             bool              // Show what would run
    Variables          map[string]string // Variables for interpolation
    WorkingDir         string            // Working directory
    Input              io.Reader         // Standard input
    Output             io.Writer         // Standard output
    ErrorOutput        io.Writer         // Error output
    Only               []string          // Filter by labels
    WithRequires       bool              // Include dependencies
    BuildfabBinaryPath string            // Path to buildfab binary (containers)
    StepCallback       StepCallback      // Callback for step events (optional)
}
```

**Verbose Levels**:
- **0 (Quiet)**: Minimal output, multiline display with dynamic updates
- **1 (Basic)**: Step status with icons, basic command output
- **2 (Detailed)**: Detailed command output with full logs
- **3 (Maximum)**: Step-by-step execution with reproduction commands

**MaxParallel**:
- `0`: Use CPU count (default)
- `>0`: Limit concurrent job execution
- Respects `max_parallel` in matrix strategy (uses minimum)

#### NewSimpleRunner

```go
func NewSimpleRunner(config *Config, opts *SimpleRunOptions) *SimpleRunner
```

Creates a new runner for stage/action execution.

**Parameters**:
- `config`: Loaded configuration
- `opts`: Execution options

**Returns**:
- `*SimpleRunner`: Runner instance

**Architecture (v0.32.0+)**:
- Uses hierarchical DAG execution
- Matrix builds expand into jobs
- Jobs execute in parallel waves
- Steps within jobs execute sequentially

**Example**:
```go
opts := buildfab.DefaultSimpleRunOptions()
opts.VerboseLevel = 1
opts.Variables = map[string]string{
    "version": "v1.0.0",
}

runner := buildfab.NewSimpleRunner(config, opts)
```

#### SimpleRunner.RunStage

```go
func (r *SimpleRunner) RunStage(ctx context.Context, stageName string) error
```

Executes a complete stage with all its steps.

**Parameters**:
- `ctx`: Context for cancellation and timeout
- `stageName`: Name of stage to execute

**Returns**:
- `error`: Execution error (nil on success)

**Behavior**:
- Builds hierarchical DAG from stage steps
- Resolves dependencies between jobs
- Detects circular dependencies
- Executes jobs in parallel waves
- Executes steps sequentially within jobs
- Handles errors according to `onerror` policy
- Displays output (ordered or multiline based on verbosity)
- Streams results as they complete

**Error Policies**:
- `stop` (default): Fail immediately, block dependents
- `warn`: Continue execution, show warning

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

if err := runner.RunStage(ctx, "build"); err != nil {
    return fmt.Errorf("stage failed: %w", err)
}
```

#### SimpleRunner.RunAction

```go
func (r *SimpleRunner) RunAction(ctx context.Context, actionName string) error
```

Executes a single action directly (no DAG).

**Parameters**:
- `ctx`: Context for cancellation
- `actionName`: Name of action to execute

**Returns**:
- `error`: Execution error (nil on success)

**Behavior**:
- Executes single action without DAG
- Interpolates variables in run command
- Executes shell command or container action
- Captures output and exit code
- No dependency resolution

**Example**:
```go
ctx := context.Background()
if err := runner.RunAction(ctx, "test"); err != nil {
    return fmt.Errorf("action failed: %w", err)
}
```

### Variables API

#### GetPlatformVariables

```go
func GetPlatformVariables() *PlatformVariables
```

Returns platform-specific variables.

**Returns**:
- `*PlatformVariables`: Platform information

**Variables**:
```go
type PlatformVariables struct {
    Platform  string  // e.g., "linux-amd64", "darwin-arm64"
    Arch      string  // e.g., "amd64", "arm64"
    OS        string  // e.g., "linux", "darwin", "windows"
    OSVersion string  // OS version string
    CPU       int     // Number of CPU cores
}
```

**Example**:
```go
platformVars := buildfab.GetPlatformVariables()
variables := map[string]string{
    "platform":   platformVars.Platform,
    "arch":       platformVars.Arch,
    "os":         platformVars.OS,
    "os_version": platformVars.OSVersion,
    "cpu":        fmt.Sprintf("%d", platformVars.CPU),
}
```

#### AddPlatformVariables

```go
func AddPlatformVariables(vars map[string]string) map[string]string
```

Adds platform variables with "platform." prefix.

**Parameters**:
- `vars`: Existing variables map

**Returns**:
- `map[string]string`: Updated variables

**Added Variables**:
- `platform.os`: Operating system
- `platform.arch`: Architecture
- `platform.cpu`: CPU count
- `platform.version`: OS version

**Example**:
```go
variables := map[string]string{
    "version": "v1.0.0",
}
variables = buildfab.AddPlatformVariables(variables)
// Now has: platform.os, platform.arch, platform.cpu, platform.version
```

#### AddVersionVariables

```go
func AddVersionVariables(vars map[string]string) map[string]string
```

Adds detailed version variables from existing version info.

**Parameters**:
- `vars`: Variables map (must contain "version" key)

**Returns**:
- `map[string]string`: Updated variables

**Added Variables**:
- `version.rawversion`: Raw version string
- `version.major`: Major version number
- `version.minor`: Minor version number
- `version.patch`: Patch version number
- `version.prerelease`: Prerelease identifier
- `version.build`: Build metadata

**Example**:
```go
variables := map[string]string{
    "version": "v1.2.3-alpha+build123",
}
variables = buildfab.AddVersionVariables(variables)
// Now has: version.major=1, version.minor=2, version.patch=3,
//          version.prerelease=alpha, version.build=build123
```

## Examples

### Example 1: Basic Execution

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    // Load configuration
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    // Create options
    opts := buildfab.DefaultSimpleRunOptions()
    opts.VerboseLevel = 1
    
    // Create runner and execute
    runner := buildfab.NewSimpleRunner(config, opts)
    ctx := context.Background()
    
    if err := runner.RunStage(ctx, "build"); err != nil {
        fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("Build successful!")
}
```

### Example 2: With Custom Variables

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    // Create options with custom variables
    opts := buildfab.DefaultSimpleRunOptions()
    opts.VerboseLevel = 1
    opts.Variables = make(map[string]string)
    
    // Add platform variables
    platformVars := buildfab.GetPlatformVariables()
    opts.Variables["platform"] = platformVars.Platform
    opts.Variables["arch"] = platformVars.Arch
    opts.Variables["os"] = platformVars.OS
    opts.Variables["cpu"] = fmt.Sprintf("%d", platformVars.CPU)
    
    // Add custom variables
    opts.Variables["version"] = "v1.0.0"
    opts.Variables["environment"] = "production"
    
    // Add prefixed platform variables
    opts.Variables = buildfab.AddPlatformVariables(opts.Variables)
    
    // Add version details
    opts.Variables = buildfab.AddVersionVariables(opts.Variables)
    
    // Create runner and execute
    runner := buildfab.NewSimpleRunner(config, opts)
    ctx := context.Background()
    
    if err := runner.RunStage(ctx, "deploy"); err != nil {
        fmt.Fprintf(os.Stderr, "Deploy failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Example 3: With Timeout and Cancellation

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    opts := buildfab.DefaultSimpleRunOptions()
    opts.VerboseLevel = 1
    
    runner := buildfab.NewSimpleRunner(config, opts)
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    
    // Handle interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Fprintf(os.Stderr, "\nInterrupted, cancelling...\n")
        cancel()
    }()
    
    // Execute stage
    if err := runner.RunStage(ctx, "build"); err != nil {
        if ctx.Err() == context.Canceled {
            fmt.Fprintf(os.Stderr, "Build cancelled by user\n")
            os.Exit(130)
        } else if ctx.Err() == context.DeadlineExceeded {
            fmt.Fprintf(os.Stderr, "Build timeout\n")
            os.Exit(1)
        }
        fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Example 4: Custom Verbosity and Debugging

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    opts := buildfab.DefaultSimpleRunOptions()
    
    // Set verbosity based on environment
    if debug := os.Getenv("DEBUG"); debug == "1" {
        opts.Debug = true
        opts.VerboseLevel = 3
    } else if verbose := os.Getenv("VERBOSE"); verbose == "1" {
        opts.VerboseLevel = 2
    } else {
        opts.VerboseLevel = 0  // Quiet mode
    }
    
    // Separate output streams
    opts.Output = os.Stdout       // Action output
    opts.ErrorOutput = os.Stderr  // Status and errors
    
    runner := buildfab.NewSimpleRunner(config, opts)
    ctx := context.Background()
    
    if err := runner.RunStage(ctx, "test"); err != nil {
        fmt.Fprintf(os.Stderr, "Tests failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Example 5: Matrix Build Execution

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    config, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    opts := buildfab.DefaultSimpleRunOptions()
    opts.VerboseLevel = 1
    opts.MaxParallel = 4  // Limit concurrent matrix jobs
    
    // Add CLI matrix variables (optional override)
    opts.Variables = map[string]string{
        "matrix.platform": "linux",  // Override specific matrix value
    }
    
    runner := buildfab.NewSimpleRunner(config, opts)
    ctx := context.Background()
    
    // Execute stage with matrix build
    if err := runner.RunStage(ctx, "matrix-build"); err != nil {
        fmt.Fprintf(os.Stderr, "Matrix build failed: %v\n", err)
        os.Exit(1)
    }
}
```

## Migration Guide

### From v0.29.1 to v0.32.0

**Breaking Changes**:
- None for library API users
- All public APIs remain compatible

**New Features**:
- Hierarchical DAG execution (automatic)
- Improved matrix build support
- Nested matrix expansion
- Better parallelism control

**Recommended Updates**:

```go
// Old code (still works)
runner := buildfab.NewSimpleRunner(config, opts)
err := runner.RunStage(ctx, "build")

// New code (same API, better execution)
runner := buildfab.NewSimpleRunner(config, opts)
err := runner.RunStage(ctx, "build")
// Now uses hierarchical DAG automatically
```

**Deprecated (Removed in v0.32.0)**:
- Flat DAG execution (automatically migrated)
- Direct Runner usage (use SimpleRunner)

### Best Practices

1. **Always use SimpleRunner**: The old Runner is deprecated
2. **Set MaxParallel**: Control resource usage
3. **Use context for cancellation**: Support graceful shutdown
4. **Provide variables early**: Set all variables in SimpleRunOptions
5. **Handle errors properly**: Check context.Err() for timeouts
6. **Use appropriate verbosity**: 0=quiet, 1=normal, 2=detailed, 3=debug

## Advanced Topics

### Custom Step Callbacks

```go
type MyCallback struct {
    results []buildfab.StepResult
}

func (c *MyCallback) OnStepStart(ctx context.Context, stepName string) {
    fmt.Printf("Starting: %s\n", stepName)
}

func (c *MyCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration, output string) {
    c.results = append(c.results, buildfab.StepResult{
        StepName: stepName,
        Status:   status,
        Duration: duration,
    })
}

func (c *MyCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    // Handle streaming output
}

func (c *MyCallback) OnStepError(ctx context.Context, stepName string, err error) {
    fmt.Fprintf(os.Stderr, "Error in %s: %v\n", stepName, err)
}

func (c *MyCallback) GetResults() []buildfab.StepResult {
    return c.results
}

// Usage
opts := buildfab.DefaultSimpleRunOptions()
opts.StepCallback = &MyCallback{}
```

### Container Support

For container actions to work, ensure:
1. `BuildfabBinaryPath` is set in options
2. Docker or Podman is available
3. Buildfab binary is in standard locations

```go
opts := buildfab.DefaultSimpleRunOptions()
opts.BuildfabBinaryPath = "/usr/local/bin/buildfab"
```

## See Also

- [Project Specification](Project-specification.md) - Complete YAML syntax
- [Developer Workflow](Developer-workflow.md) - Development guide
- [Examples](../examples/) - Sample configurations
- [Tests](../tests/) - Integration tests

## Support

For issues and questions:
- GitHub: https://github.com/AlexBurnes/buildfab
- Documentation: https://github.com/AlexBurnes/buildfab/tree/master/docs

## Version History

- **v0.32.0** (2025-11-05): Hierarchical DAG architecture, improved matrix builds
- **v0.31.0**: Multi-dimensional matrix support
- **v0.29.1**: Production-ready release with container support

