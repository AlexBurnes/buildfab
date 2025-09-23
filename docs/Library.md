# Library API Reference

buildfab provides a comprehensive Go library API for embedding automation workflows in other applications.

## Core API

### SimpleRunner (Recommended)

The `SimpleRunner` provides a simplified interface for most use cases:

```go
package buildfab

import "context"

// SimpleRunner provides a simplified interface for running stages and actions
type SimpleRunner struct {
    config *Config
    opts   *SimpleRunOptions
}

// NewSimpleRunner creates a new simple buildfab runner
func NewSimpleRunner(config *Config, opts *SimpleRunOptions) *SimpleRunner

// RunStage executes a specific stage with automatic output handling
func (r *SimpleRunner) RunStage(ctx context.Context, stageName string) error

// RunAction executes a specific action with automatic output handling
func (r *SimpleRunner) RunAction(ctx context.Context, actionName string) error

// RunStageStep executes a specific step within a stage
func (r *SimpleRunner) RunStageStep(ctx context.Context, stageName, stepName string) error
```

### Advanced Runner

For advanced use cases with custom callbacks:

```go
// Runner provides the main execution interface
type Runner struct {
    config   *Config
    opts     *RunOptions
    registry ActionRegistry
}

// NewRunner creates a new buildfab runner with default built-in actions
func NewRunner(config *Config, opts *RunOptions) *Runner

// RunStage executes a specific stage
func (r *Runner) RunStage(ctx context.Context, stageName string) error

// RunAction executes a specific action
func (r *Runner) RunAction(ctx context.Context, actionName string) error

// RunStageStep executes a specific step within a stage
func (r *Runner) RunStageStep(ctx context.Context, stageName, stepName string) error
```

### Convenience Functions

```go
// RunStageSimple executes a stage with minimal configuration
func RunStageSimple(ctx context.Context, configPath, stageName string, verbose bool) error

// RunActionSimple executes an action with minimal configuration
func RunActionSimple(ctx context.Context, configPath, actionName string, verbose bool) error
```

### Configuration Options

#### SimpleRunOptions (Recommended)

```go
// SimpleRunOptions configures simple stage execution
type SimpleRunOptions struct {
    ConfigPath   string            // Path to project.yml (default: ".project.yml")
    MaxParallel  int               // Maximum parallel execution (default: CPU count)
    Verbose      bool              // Enable verbose output
    Debug        bool              // Enable debug output
    Variables    map[string]string // Additional variables for interpolation
    WorkingDir   string            // Working directory for execution
    Output       io.Writer         // Output writer (default: os.Stdout)
    ErrorOutput  io.Writer         // Error output writer (default: os.Stderr)
    Only         []string          // Only run steps matching these labels
    WithRequires bool              // Include required dependencies when running single step
}
```

#### RunOptions (Advanced)

```go
// RunOptions configures stage execution
type RunOptions struct {
    ConfigPath   string            // Path to project.yml (default: ".project.yml")
    MaxParallel  int               // Maximum parallel execution (default: CPU count)
    Verbose      bool              // Enable verbose output
    Debug        bool              // Enable debug output
    Variables    map[string]string // Additional variables for interpolation
    WorkingDir   string            // Working directory for execution
    Output       io.Writer         // Output writer (default: os.Stdout)
    ErrorOutput  io.Writer         // Error output writer (default: os.Stderr)
    Only         []string          // Only run steps matching these labels
    WithRequires bool              // Include required dependencies when running single step
    StepCallback StepCallback      // Optional callback for step execution events
}
```

### Step Callbacks

```go
// StepCallback defines the interface for step execution callbacks
type StepCallback interface {
    // OnStepStart is called when a step starts execution
    OnStepStart(ctx context.Context, stepName string)
    
    // OnStepComplete is called when a step completes (success, warning, or error)
    OnStepComplete(ctx context.Context, stepName string, status StepStatus, message string, duration time.Duration)
    
    // OnStepOutput is called for step output (when verbose mode is enabled)
    OnStepOutput(ctx context.Context, stepName string, output string)
    
    // OnStepError is called for step errors
    OnStepError(ctx context.Context, stepName string, err error)
}
```

## Result Types

### StageResult

```go
// StageResult contains execution results for a stage
type StageResult struct {
    StageName string
    Success   bool
    Steps     []StepResult
    Duration  time.Duration
    Error     error
}
```

### StepResult

```go
// StepResult contains execution results for a step
type StepResult struct {
    StepName   string
    ActionName string
    Status     StepStatus
    Duration   time.Duration
    Output     string
    Error      error
}
```

### StepStatus

```go
// StepStatus represents the execution status of a step
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

## Error Types

### ConfigurationError

```go
// ConfigurationError represents errors in project.yml configuration
type ConfigurationError struct {
    Message string
    Path    string
    Line    int
    Column  int
}
```

### ExecutionError

```go
// ExecutionError represents errors during step execution
type ExecutionError struct {
    StepName string
    Action   string
    Message  string
    Output   string
}
```

### DependencyError

```go
// DependencyError represents errors in dependency resolution
type DependencyError struct {
    Message string
    Cycle   []string
}
```

### VariableError

```go
// VariableError represents errors in variable interpolation
type VariableError struct {
    Variable string
    Message  string
}
```

## Usage Examples

### Basic Stage Execution (SimpleRunner - Recommended)

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Printf("Error loading config: %v\n", err)
        return
    }
    
    // Create simple run options
    opts := &buildfab.SimpleRunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
        Output:     os.Stdout,
        ErrorOutput: os.Stderr,
    }
    
    // Create simple runner
    runner := buildfab.NewSimpleRunner(cfg, opts)
    
    // Run a stage - all output is handled automatically!
    err = runner.RunStage(ctx, "pre-push")
    if err != nil {
        fmt.Printf("Stage failed: %v\n", err)
        os.Exit(1)
    }
}
```

### One-liner Stage Execution

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

### Step Callbacks for Real-time Progress (Advanced Runner)

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// ExampleStepCallback demonstrates step-by-step progress reporting
type ExampleStepCallback struct{}

func (c *ExampleStepCallback) OnStepStart(ctx context.Context, stepName string) {
    fmt.Printf("🔄 Running step: %s\n", stepName)
}

func (c *ExampleStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration) {
    var icon string
    switch status {
    case buildfab.StepStatusOK:
        icon = "✔"
    case buildfab.StepStatusWarn:
        icon = "⚠"
    case buildfab.StepStatusError:
        icon = "✖"
    case buildfab.StepStatusSkipped:
        icon = "○"
    default:
        icon = "?"
    }
    
    fmt.Printf("%s %s: %s (%v)\n", icon, stepName, message, duration)
}

func (c *ExampleStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    if output != "" {
        fmt.Printf("📤 %s output:\n%s\n", stepName, output)
    }
}

func (c *ExampleStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
    fmt.Printf("❌ %s failed: %v\n", stepName, err)
}

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        fmt.Printf("Error loading config: %v\n", err)
        return
    }
    
    // Create run options with step callback
    opts := buildfab.DefaultRunOptions()
    opts.StepCallback = &ExampleStepCallback{}
    opts.Verbose = true
    
    // Create runner
    runner := buildfab.NewRunner(cfg, opts)
    
    // Run a stage with step callbacks
    err = runner.RunStage(ctx, "pre-push")
    if err != nil {
        fmt.Printf("Stage execution failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Silent Step Callbacks (Errors Only)

```go
// SilentStepCallback provides minimal output, only showing errors
type SilentStepCallback struct{}

func (c *SilentStepCallback) OnStepStart(ctx context.Context, stepName string) {
    // Silent - no output
}

func (c *SilentStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration) {
    // Only show errors
    if status == buildfab.StepStatusError {
        fmt.Printf("Error in %s: %s\n", stepName, message)
    }
}

func (c *SilentStepCallback) OnStepOutput(ctx context.Context, stepName string, output string) {
    // Silent - no output
}

func (c *SilentStepCallback) OnStepError(ctx context.Context, stepName string, err error) {
    fmt.Printf("Error in %s: %v\n", stepName, err)
}

// Usage
opts := &buildfab.RunOptions{
    ConfigPath:   "project.yml",
    StepCallback: &SilentStepCallback{},
}
```

### Custom Variables (SimpleRunner)

```go
cfg, err := buildfab.LoadConfig(".project.yml")
if err != nil {
    // Handle error
}

opts := &buildfab.SimpleRunOptions{
    ConfigPath: ".project.yml",
    Variables: map[string]string{
        "custom_var": "value",
        "environment": "production",
    },
}

runner := buildfab.NewSimpleRunner(cfg, opts)
err = runner.RunStage(ctx, "deploy")
```

### Single Action Execution

```go
// Using SimpleRunner
runner := buildfab.NewSimpleRunner(cfg, opts)
err := runner.RunAction(ctx, "run-tests")

// Using convenience function
err := buildfab.RunActionSimple(ctx, ".project.yml", "run-tests", true)
```

### Single Step Execution

```go
// Using SimpleRunner
runner := buildfab.NewSimpleRunner(cfg, opts)

// Run just the version-check step from pre-push stage
err := runner.RunStageStep(ctx, "pre-push", "version-check")

// Run with dependencies
opts.WithRequires = true
runner = buildfab.NewSimpleRunner(cfg, opts)
err = runner.RunStageStep(ctx, "pre-push", "version-check")
```

## Integration with pre-push

The pre-push utility uses buildfab as its execution engine:

```go
package main

import (
    "context"
    "os"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        os.Exit(1)
    }
    
    // Create simple run options
    opts := &buildfab.SimpleRunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
        Output:     os.Stdout,
        ErrorOutput: os.Stderr,
    }
    
    // Create simple runner
    runner := buildfab.NewSimpleRunner(cfg, opts)
    
    // Run pre-push stage
    err = runner.RunStage(ctx, "pre-push")
    if err != nil {
        os.Exit(1)
    }
}
```

## Advanced Usage

### Custom Output Writers

```go
var buf bytes.Buffer

opts := &buildfab.RunOptions{
    ConfigPath:  "project.yml",
    Output:      &buf,
    ErrorOutput: os.Stderr,
}

err := buildfab.RunStage(ctx, "build", opts)
fmt.Printf("Output: %s\n", buf.String())
```

### Debug Mode

```go
opts := &buildfab.RunOptions{
    ConfigPath: "project.yml",
    Debug:      true,
    Verbose:    true,
}

err := buildfab.RunStage(ctx, "test", opts)
```

### Conditional Execution

```go
opts := &buildfab.RunOptions{
    ConfigPath: "project.yml",
    OnlyLabels: []string{"release", "production"},
}

err := buildfab.RunStage(ctx, "deploy", opts)
```

## Error Handling

### Checking Error Types

```go
err := buildfab.RunStage(ctx, "pre-push", opts)
if err != nil {
    switch e := err.(type) {
    case *buildfab.ConfigurationError:
        fmt.Printf("Configuration error: %s at line %d\n", e.Message, e.Line)
    case *buildfab.ExecutionError:
        fmt.Printf("Execution error in %s: %s\n", e.StepName, e.Message)
    case *buildfab.DependencyError:
        fmt.Printf("Dependency error: %s\n", e.Message)
    default:
        fmt.Printf("Unknown error: %v\n", err)
    }
}
```

### Using Errors Package

```go
import "errors"

err := buildfab.RunStage(ctx, "pre-push", opts)
if errors.Is(err, &buildfab.ConfigurationError{}) {
    // Handle configuration error
}
```