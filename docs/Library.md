# Library API Reference

buildfab provides a comprehensive Go library API for embedding automation workflows in other applications.

## Core API

### Main Functions

```go
package buildfab

import "context"

// RunStage executes a specific stage from project.yml configuration
func RunStage(ctx context.Context, stageName string, opts *RunOptions) error

// RunAction executes a standalone action
func RunAction(ctx context.Context, actionName string, opts *RunOptions) error

// RunStageStep executes a single step within a stage
func RunStageStep(ctx context.Context, stageName, stepName string, opts *RunOptions) error
```

### Configuration Options

```go
// RunOptions configures stage execution
type RunOptions struct {
    ConfigPath    string            // Path to project.yml (default: "project.yml")
    MaxParallel   int               // Maximum parallel execution (default: CPU count)
    Verbose       bool              // Enable verbose output
    Debug         bool              // Enable debug output
    Variables     map[string]string // Additional variables for interpolation
    WorkingDir    string            // Working directory for execution
    Output        io.Writer         // Output writer (default: os.Stdout)
    ErrorOutput   io.Writer         // Error output writer (default: os.Stderr)
    OnlyLabels    []string          // Labels for conditional execution
    WithRequires  bool              // Include dependencies when running single step
    StepCallback  StepCallback      // Optional callback for step execution events
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

### Basic Stage Execution

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/burnes/buildfab"
)

func main() {
    ctx := context.Background()
    
    opts := &buildfab.RunOptions{
        ConfigPath: "project.yml",
        Verbose:    true,
        WorkingDir: ".",
    }
    
    err := buildfab.RunStage(ctx, "pre-push", opts)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Stage execution failed: %v\n", err)
        os.Exit(1)
    }
}
```

### Step Callbacks for Real-time Progress

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    "github.com/burnes/buildfab"
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
    
    // Create run options with step callback
    opts := &buildfab.RunOptions{
        ConfigPath:   "project.yml",
        Verbose:      true,
        WorkingDir:   ".",
        StepCallback: &ExampleStepCallback{},
    }
    
    err := buildfab.RunStage(ctx, "pre-push", opts)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Stage execution failed: %v\n", err)
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

### Custom Variables

```go
opts := &buildfab.RunOptions{
    ConfigPath: "project.yml",
    Variables: map[string]string{
        "custom_var": "value",
        "environment": "production",
    },
}

err := buildfab.RunStage(ctx, "deploy", opts)
```

### Single Action Execution

```go
err := buildfab.RunAction(ctx, "run-tests", opts)
```

### Single Step Execution

```go
// Run just the version-check step from pre-push stage
err := buildfab.RunStageStep(ctx, "pre-push", "version-check", opts)

// Run with dependencies
opts.WithRequires = true
err := buildfab.RunStageStep(ctx, "pre-push", "version-check", opts)
```

## Integration with pre-push

The pre-push utility uses buildfab as its execution engine:

```go
package main

import (
    "context"
    "os"
    "github.com/burnes/buildfab"
)

func main() {
    ctx := context.Background()
    
    opts := &buildfab.RunOptions{
        ConfigPath:  "project.yml",
        Verbose:     true,
        WorkingDir:  ".",
    }
    
    err := buildfab.RunStage(ctx, "pre-push", opts)
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