# Buildfab CLI DRY Principle Violation Analysis

## Executive Summary

The current buildfab CLI **violates DRY (Don't Repeat Yourself) principles** by duplicating functionality that already exists in the public library API. The CLI should be a thin wrapper around the library, not a reimplementation of core functionality.

## Current Problems

### ❌ **Direct Internal Package Usage**

**Current CLI (main.go):**
```go
import (
    "github.com/AlexBurnes/buildfab/internal/config"     // ❌ Should use pkg/buildfab
    "github.com/AlexBurnes/buildfab/internal/executor"   // ❌ Should use pkg/buildfab
    "github.com/AlexBurnes/buildfab/internal/ui"         // ❌ Should use callbacks
    "github.com/AlexBurnes/buildfab/internal/version"    // ❌ Should use library
    "github.com/AlexBurnes/buildfab/internal/actions"    // ❌ Should use library
    "github.com/AlexBurnes/buildfab/pkg/buildfab"        // ✅ Correct
)
```

**Should be:**
```go
import (
    "github.com/AlexBurnes/buildfab/pkg/buildfab"  // ✅ Only library API
)
```

### ❌ **Duplicated Configuration Loading**

**Current CLI:**
```go
// Duplicated configuration loading logic
cfg, err := config.Load(configPath)
if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
}

// Duplicated variable detection
gitVars, err := config.DetectGitVariables(ctx)
versionDetector := version.New()
versionVars, err := versionDetector.GetVersionVariables(ctx)

// Duplicated variable merging
variables := make(map[string]string)
for k, v := range gitVars {
    variables[k] = v
}
for k, v := range versionVars {
    variables[k] = v
}

// Duplicated variable resolution
if err := config.ResolveVariables(cfg, variables); err != nil {
    return fmt.Errorf("failed to resolve variables: %w", err)
}
```

**Should be:**
```go
// Use library API
cfg, err := buildfab.LoadConfig(configPath)
if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
}
```

### ❌ **Duplicated Validation Logic**

**Current CLI:**
```go
// Manual validation in runValidate
fmt.Printf("Configuration is valid: %s\n", configPath)
fmt.Printf("Project: %s\n", cfg.Project.Name)
fmt.Printf("Actions: %d\n", len(cfg.Actions))
fmt.Printf("Stages: %d\n", len(cfg.Stages))
```

**Should be:**
```go
// Use library validation
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("configuration validation failed: %w", err)
}
```

### ❌ **Duplicated Action Execution**

**Current CLI:**
```go
// Manual built-in action handling
registry := actions.New()
if runner, exists := registry.GetRunner(actionName); exists {
    result, err := runner.Run(ctx)
    ui.PrintStepStatus(actionName, result.Status, result.Message)
    if err != nil {
        return err
    }
    return nil
}
// Manual custom action execution
return exec.RunAction(ctx, actionName)
```

**Should be:**
```go
// Use library API
return runner.RunAction(ctx, actionName)
```

### ❌ **Duplicated UI Logic**

**Current CLI:**
```go
// Manual UI creation and management
ui := ui.New(verbose, debug)
opts.Output = os.Stdout
exec := executor.New(cfg, opts, ui)
```

**Should be:**
```go
// Use library callbacks
opts.StepCallback = &CLIStepCallback{verbose: verbose, debug: debug}
```

## Refactored Solution

### ✅ **Proper Library Usage**

**Refactored CLI (main_refactored.go):**
```go
import (
    "github.com/AlexBurnes/buildfab/pkg/buildfab"  // ✅ Only library API
)

// All functionality uses library API
func createRunner() (*buildfab.Runner, error) {
    cfg, err := buildfab.LoadConfig(configPath)  // ✅ Library API
    if err != nil {
        return nil, fmt.Errorf("failed to load configuration: %w", err)
    }
    
    opts := &buildfab.RunOptions{  // ✅ Library API
        ConfigPath:   configPath,
        MaxParallel:  maxParallel,
        Verbose:      verbose,
        Debug:        debug,
        Variables:    variables,
        WorkingDir:   workingDir,
        Output:       os.Stdout,
        ErrorOutput:  os.Stderr,
        Only:         only,
        WithRequires: withRequires,
        StepCallback: &CLIStepCallback{verbose: verbose, debug: debug},  // ✅ Library callbacks
    }
    
    return buildfab.NewRunner(cfg, opts), nil  // ✅ Library API
}
```

### ✅ **Step Callback Implementation**

**CLI-specific UI via callbacks:**
```go
type CLIStepCallback struct {
    verbose bool
    debug   bool
}

func (c *CLIStepCallback) OnStepStart(ctx context.Context, stepName string) {
    if c.verbose {
        fmt.Printf("🔄 Running step: %s\n", stepName)
    }
}

func (c *CLIStepCallback) OnStepComplete(ctx context.Context, stepName string, status buildfab.StepStatus, message string, duration time.Duration) {
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
    
    if c.verbose || status == buildfab.StepStatusError {
        fmt.Printf("%s %s: %s (%v)\n", icon, stepName, message, duration)
    }
}
```

### ✅ **Simplified Command Handlers**

**All commands now use library API:**
```go
func runStage(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    
    runner, err := createRunner()  // ✅ Library API
    if err != nil {
        return err
    }
    
    if len(args) == 2 {
        return runner.RunStageStep(ctx, args[0], args[1])  // ✅ Library API
    }
    
    return runner.RunStage(ctx, args[0])  // ✅ Library API
}

func runAction(cmd *cobra.Command, args []string) error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    
    runner, err := createRunner()  // ✅ Library API
    if err != nil {
        return err
    }
    
    return runner.RunAction(ctx, args[0])  // ✅ Library API
}

func runValidate(cmd *cobra.Command, args []string) error {
    cfg, err := buildfab.LoadConfig(configPath)  // ✅ Library API
    if err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    if err := cfg.Validate(); err != nil {  // ✅ Library API
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    fmt.Printf("Configuration is valid: %s\n", configPath)
    return nil
}
```

## Benefits of Refactored Solution

### ✅ **DRY Compliance**
- **No duplication**: All functionality uses library API
- **Single source of truth**: Library contains all business logic
- **Maintainable**: Changes only need to be made in library

### ✅ **Proper Architecture**
- **Thin CLI layer**: CLI is just argument parsing + library calls
- **Separation of concerns**: CLI handles UI, library handles logic
- **Testable**: Library can be tested independently

### ✅ **Consistency**
- **Same behavior**: CLI and library have identical functionality
- **Same error handling**: Consistent error messages and handling
- **Same features**: All library features available in CLI

### ✅ **Extensibility**
- **Easy to add features**: Add to library, CLI automatically gets them
- **Easy to customize**: Use different callbacks for different CLI behaviors
- **Easy to embed**: Library API is the same for CLI and other tools

## Implementation Steps

1. **Replace main.go** with main_refactored.go
2. **Remove internal package imports** from CLI
3. **Test all CLI commands** to ensure they work identically
4. **Update tests** to use library API instead of internal packages
5. **Verify no functionality is lost** in the refactoring

## Conclusion

The refactored CLI properly follows DRY principles by:
- Using only the public library API
- Eliminating all code duplication
- Making the CLI a thin wrapper around the library
- Ensuring consistency between CLI and library functionality

This refactoring makes the codebase more maintainable, testable, and follows proper software architecture principles.