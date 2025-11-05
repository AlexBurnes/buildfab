# Bug Fixes Required Before Matrix-on-Stage Implementation

## Overview

Two critical bugs must be fixed before implementing the matrix-on-stage feature:

1. **Variable Interpolation with Default Values**: Support default value syntax
2. **DAG Skip Propagation**: Ensure skipped steps propagate to all dependents

## Issue 1: Variable Interpolation with Default Values

### Problem

Currently, when a variable is undefined (e.g., `${{matrix.compiler}}`), buildfab raises an error. Users need the ability to provide default values similar to bash `${VAR:-default}` syntax.

### Requirements

Support two forms of default values:

1. **Literal Default**: `${{variable-default}}` or `${{variable-"default"}}`
   - If `variable` is undefined, use literal string `"default"`
   
2. **Variable Default**: `${{variable-otherVariable}}`
   - If `variable` is undefined, use value of `otherVariable`

### Syntax Specification

- **Format**: `${{ variableName-defaultValue }}`
- **Default can be**: 
  - Literal string (with or without quotes)
  - Another variable reference
- **Parsing**: Split on first `-` character after variable name

### Examples

```yaml
actions:
  - name: build
    run: |
      echo "Compiler: ${{matrix.compiler-gcc}}"
      echo "Platform: ${{matrix.platform-platform.default}}"
      
  # With quotes for literal
  - name: test
    run: |
      echo "Version: ${{version.version-"unknown"}}"
```

### Implementation Plan

**Files to Modify:**
1. `pkg/buildfab/variables.go` - `InterpolateVariables()` function
2. `internal/config/config.go` - `resolveString()` function (if still used)

**Changes Required:**

1. **Parse variable reference with default**:
   ```go
   // Parse: "variableName-defaultValue"
   // Extract: variableName and defaultValue (which may be a variable reference)
   ```

2. **Evaluate default value**:
   - If variable exists → use variable value
   - If variable doesn't exist → evaluate default
     - If default looks like a variable (contains `.` or is a known variable) → resolve it
     - Otherwise → use as literal string

3. **Update validation**: Don't error on undefined variables if default is provided

### Algorithm

```go
func parseVariableWithDefault(ref string) (varName, defaultValue string, hasDefault bool) {
    // Split on first '-'
    parts := strings.SplitN(ref, "-", 2)
    if len(parts) == 2 {
        return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
    }
    return ref, "", false
}

func resolveVariableWithDefault(varName, defaultValue string, variables map[string]string) (string, error) {
    // Check if variable exists
    if value, exists := variables[varName]; exists {
        return value, nil
    }
    
    // Variable doesn't exist, use default
    if defaultValue == "" {
        return "", fmt.Errorf("undefined variable: %s", varName)
    }
    
    // Check if default is a variable reference
    if strings.HasPrefix(defaultValue, "${{") && strings.HasSuffix(defaultValue, "}}") {
        // Recursively resolve default variable
        return InterpolateVariables(defaultValue, variables)
    }
    
    // Check if default looks like a variable (e.g., "variable.name")
    if strings.Contains(defaultValue, ".") {
        if value, exists := variables[defaultValue]; exists {
            return value, nil
        }
    }
    
    // Remove quotes if present
    defaultValue = strings.Trim(defaultValue, `"'`)
    
    // Use as literal string
    return defaultValue, nil
}
```

## Issue 2: DAG Skip Propagation

### Problem

When a step is skipped due to dependency failure, dependent steps are not properly skipped. The DAG executor checks if dependencies are "completed" but doesn't distinguish between "successfully completed" and "skipped".

**Example:**
```yaml
stages:
  test:
    steps:
      - action: step1        # Fails
      - action: step2        # Skipped (depends on step1)
        require: [step1]
      - action: step3        # ❌ Currently runs (should be skipped)
        require: [step2]
```

### Current Behavior

1. `step1` fails → marked as `failed`
2. `step2` becomes ready → checks `hasFailedDependency()` → marks as `StatusSkipped` → added to `completed`
3. `step3` becomes ready → `allDependenciesCompleted()` checks if `step2` is in `completed` → **TRUE** → step3 runs ❌

### Root Cause

`allDependenciesCompleted()` only checks if dependencies are in the `completed` map, but doesn't verify they completed **successfully**. A skipped step is also "completed", so the check passes incorrectly.

### Solution

**Track step status separately** and modify dependency checks to verify dependencies completed successfully (not skipped/failed).

### Implementation Plan

**Files to Modify:**
1. `pkg/buildfab/buildfab.go` - DAG execution functions

**Changes Required:**

1. **Track step status separately from completion**:
   ```go
   // Instead of just: completed map[string]bool
   // Use: status map[string]Status  (where Status = OK, Error, Skipped)
   ```

2. **Modify `allDependenciesCompleted()`**:
   ```go
   func (r *Runner) allDependenciesCompleted(node *DAGNode, status map[string]Status) bool {
       for _, dep := range node.Dependencies {
           depStatus := status[dep]
           // Dependency must be OK (not Error, not Skipped, and must exist)
           if depStatus != StatusOK {
               return false
           }
       }
       return true
   }
   ```

3. **Propagate skips**:
   - When a step is skipped due to failed dependency, mark it as skipped
   - When checking if a step can run, verify all dependencies are OK (not skipped)
   - Automatically skip steps that depend on skipped steps

4. **Update execution loops**:
   - Replace `completed` map with `status` map in all execution functions
   - Check for `StatusOK` instead of just completion
   - Skip dependent steps when their dependencies are skipped

### Modified Functions

1. **`getReadyStepsLocked()`**: Check dependencies are OK, not just completed
2. **`allDependenciesCompleted()`**: Verify dependencies completed successfully
3. **`executeDAGWithOrderedStreaming()`**: Track status instead of just completion
4. **`executeDAGWithCallback()`**: Same status tracking
5. **`executeDAGWithParallel()`**: Same status tracking

### Algorithm

```go
// Track status for each step
status := make(map[string]Status)  // Status: OK, Error, Skipped

// When processing ready steps:
for _, nodeName := range readySteps {
    node := dag[nodeName]
    
    // Check if any dependency failed or was skipped
    if r.hasFailedOrSkippedDependency(node, status) {
        status[nodeName] = StatusSkipped
        // Mark result as skipped and continue
        continue
    }
    
    // Check if all dependencies completed successfully
    if !r.allDependenciesCompleted(node, status) {
        continue  // Not ready yet
    }
    
    // Step is ready to execute
    // ... execute step ...
    
    // After execution:
    if err != nil {
        status[nodeName] = StatusError
    } else {
        status[nodeName] = StatusOK
    }
}

func (r *Runner) hasFailedOrSkippedDependency(node *DAGNode, status map[string]Status) bool {
    for _, dep := range node.Dependencies {
        depStatus := status[dep]
        if depStatus == StatusError || depStatus == StatusSkipped {
            return true
        }
    }
    return false
}

func (r *Runner) allDependenciesCompleted(node *DAGNode, status map[string]Status) bool {
    for _, dep := range node.Dependencies {
        if status[dep] != StatusOK {
            return false
        }
    }
    return true
}
```

## Testing Strategy

### Issue 1: Variable Default Values

**Test Cases:**
1. Variable exists → use variable value
2. Variable undefined, literal default → use default
3. Variable undefined, variable default → resolve default variable
4. Variable undefined, default variable undefined → error or empty?
5. Nested defaults: `${{var1-var2-var3}}` → should this work?
6. Default with quotes: `${{var-"default"}}`
7. Default without quotes: `${{var-default}}`

### Issue 2: Skip Propagation

**Test Cases:**
1. step1 fails → step2 skipped → step3 skipped
2. step1 succeeds → step2 runs → step3 runs
3. step1 skipped (condition) → step2 skipped → step3 skipped
4. Multiple dependencies: step3 depends on [step1, step2], step1 fails, step2 succeeds → step3 skipped
5. Complex DAG: Multiple paths, some succeed, some fail → verify correct skip propagation

## Implementation Order

1. **Fix Issue 1** (Variable defaults) - Less complex, independent change
2. **Fix Issue 2** (Skip propagation) - More complex, affects core execution logic
3. **Test both fixes together** - Ensure no regressions
4. **Proceed with matrix-on-stage implementation** - After both fixes verified

## Files to Modify

### Issue 1
- `pkg/buildfab/variables.go`
- `internal/config/config.go` (if still used)
- `pkg/buildfab/variables_test.go` (tests)

### Issue 2
- `pkg/buildfab/buildfab.go` (multiple functions)
- `pkg/buildfab/buildfab_test.go` (tests)

## Success Criteria

### Issue 1
- ✅ `${{var-default}}` works when `var` is undefined
- ✅ `${{var-otherVar}}` resolves to `otherVar` when `var` is undefined
- ✅ Existing variable interpolation still works
- ✅ Error messages are clear when defaults can't be resolved

### Issue 2
- ✅ Steps depending on failed steps are skipped
- ✅ Steps depending on skipped steps are skipped (transitive)
- ✅ Steps depending on successful steps run normally
- ✅ Complex DAGs correctly propagate skips through all paths

