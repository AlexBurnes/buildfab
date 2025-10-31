# Matrix on Stage Feature - Implementation Plan

## Overview

This document outlines the implementation plan for enabling matrix configuration on stage steps, allowing entire stages to be executed multiple times with different matrix values. This feature enables use cases like cross-compiler builds, multi-platform testing, and parameterized stage execution.

## Problem Statement

### Current Limitation

Currently, matrix configuration can only be applied to action steps, not stage steps. This limitation prevents users from running entire stages (with multiple actions and dependencies) across different matrix values.

**Example use case that's not currently possible:**

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
      - stage: build              # ❌ Cannot apply matrix here
        matrix:
          values:
            compiler: ["gcc", "clang"]
        strategy:
          max_parallel: 1
```

### User Requirements

1. **Matrix on Stage Steps**: Apply matrix configuration to steps that reference other stages
2. **Sequential Execution with max_parallel**: When `max_parallel: 1`, ensure complete execution of one matrix job before starting the next
3. **Sliding Window Execution**: When `max_parallel: N`, ensure at most N matrix jobs run concurrently
4. **Preserve Stage Dependencies**: All internal dependencies within each matrix job must be preserved
5. **Matrix Variable Propagation**: Matrix variables must be available to all steps in the expanded stage

## Solution Design: Option A with Sliding Window Dependencies

### Approach: Matrix-First Expansion

**Execution Order:**
1. **Matrix Expansion First**: Expand matrix values to create matrix jobs
2. **Stage Expansion Second**: For each matrix combination, expand the referenced stage with matrix variables in context
3. **Dependency Injection**: Inject sliding window dependencies between matrix jobs based on `max_parallel`

### Key Design Decisions

#### 1. Matrix-First Expansion (Option A)

**Why Matrix-First?**
- Natural grouping: Steps are naturally grouped by matrix job during expansion
- Easier boundary tracking: We can identify which steps belong to which matrix job
- Matrix variables available: Matrix variables are available during stage expansion
- Simpler implementation: Clearer code flow and logic

**Execution Flow:**
```
Step with stage + matrix
  ↓
Expand matrix: [gcc, clang] → 2 matrix jobs
  ↓
For each matrix job:
  - Expand stage reference with matrix variables
  - Track first steps (entry points)
  - Track last steps (leaf nodes)
  ↓
Inject sliding window dependencies
  ↓
Build DAG and execute
```

#### 2. Sliding Window Dependency Algorithm

**Pattern:**
- **Jobs 1..N**: No cross-job dependencies (can start immediately)
- **Job i (where i > N)**: First step depends on last step of job (i - max_parallel)

**Example: max_parallel = 2, 4 jobs (v1, v2, v3, v4)**

```
Job 1 (v1): No dependency → starts immediately
Job 2 (v2): No dependency → starts immediately

Job 3 (v3): First step depends on last step of Job 1
Job 4 (v4): First step depends on last step of Job 2
```

**Execution Timeline:**
```
Time:  [=========] [=========]
Job 1: [running──]
Job 2:           [running──]
Job 3:            [waiting─] [running──]
Job 4:                       [waiting─] [running──]
```

**General Algorithm:**
```go
// For M matrix jobs with max_parallel = N
for i := N; i < M; i++ {
    previousJobIdx := i - N
    // First steps of job i depend on last steps of job (i - N)
    injectDependency(jobs[i].firstSteps, jobs[previousJobIdx].lastSteps)
}
```

#### 3. No DAG Executor Changes Required

The existing DAG executor already respects dependencies correctly:
- `buildDAG()` uses `step.Require` to build dependency graph
- `executeDAGWithOrderedStreaming()` starts steps only when dependencies are satisfied
- We just need to inject the right dependencies before DAG construction

## Implementation Details

### Phase 1: Matrix-First Expansion for Stage Steps

#### 1.1 Modify Matrix Expansion to Handle Stage Steps

**File**: `pkg/buildfab/matrix.go`

**New Function**: `ExpandMatrixStageStep()`
```go
func (me *MatrixExpander) ExpandMatrixStageStep(
    step *Step, 
    referencedStage Stage,
    matrixValues []map[string]interface{},
) ([]MatrixJob, error) {
    // 1. For each matrix combination, expand the stage reference
    // 2. Track first steps (steps with no dependencies within the job)
    // 3. Track last steps (steps with no dependents within the job)
    // 4. Return matrix jobs with all steps and boundary information
}
```

**Key Implementation Points:**
- Expand stage reference within each matrix combination context
- Inject matrix variables into all expanded steps
- Preserve all internal stage dependencies
- Track step boundaries (first/last steps) for each matrix job

#### 1.2 Update expandMatrixStepsWithPools()

**File**: `pkg/buildfab/buildfab.go`

**Modification**: Handle steps with `stage:` field
```go
func (r *Runner) expandMatrixStepsWithPools(steps []Step) ([]Step, map[string]*Action, map[string]int, error) {
    for _, step := range steps {
        if step.Matrix != nil {
            if step.Stage != "" {
                // Handle matrix on stage step
                matrixJobs, err := expandMatrixStageStep(...)
                // Process matrix jobs...
            } else if step.Action != "" {
                // Existing logic for action steps
            }
        }
    }
}
```

### Phase 2: Sliding Window Dependency Injection

#### 2.1 Identify Step Boundaries

**Function**: `findFirstSteps()` and `findLastSteps()`

**Algorithm:**
- **First Steps**: Steps with no `require` dependencies within the matrix job (or dependencies that reference steps outside the job)
- **Last Steps**: Steps that are not required by any other step within the matrix job

**Implementation:**
```go
func findFirstSteps(jobSteps []Step) []string {
    // Build dependency graph for this job
    // Find steps with no dependencies within the job
}

func findLastSteps(jobSteps []Step) []string {
    // Build reverse dependency graph
    // Find steps with no dependents within the job
}
```

#### 2.2 Inject Cross-Job Dependencies

**Function**: `injectSlidingWindowDependencies()`

**Implementation:**
```go
func injectSlidingWindowDependencies(jobs []MatrixJob, maxParallel int) {
    if maxParallel <= 0 {
        return // No limit, no dependencies needed
    }
    
    for i := maxParallel; i < len(jobs); i++ {
        previousJobIdx := i - maxParallel
        previousJobLastSteps := jobs[previousJobIdx].LastSteps
        currentJobFirstSteps := jobs[i].FirstSteps
        
        // Inject dependency: first steps of current job depend on last steps of previous job
        for _, firstStepName := range currentJobFirstSteps {
            firstStep := findStepInJob(jobs[i], firstStepName)
            for _, lastStepName := range previousJobLastSteps {
                if !contains(firstStep.Require, lastStepName) {
                    firstStep.Require = append(firstStep.Require, lastStepName)
                }
            }
        }
    }
}
```

### Phase 3: Integration with Execution Pipeline

#### 3.1 Update runStageInternal()

**File**: `pkg/buildfab/buildfab.go`

**Modification**: Change expansion order for matrix stage steps

**Current Order:**
```
Stage expansion → Matrix expansion
```

**New Order for Matrix Stage Steps:**
```
Matrix expansion (with stage expansion inside) → Dependency injection → DAG construction
```

**Implementation:**
```go
func (r *Runner) runStageInternal(ctx context.Context, stage Stage, stageName string) error {
    // ... setup ...
    
    // Check if any step has both stage: and matrix:
    hasMatrixStageSteps := false
    for _, step := range stage.Steps {
        if step.Stage != "" && step.Matrix != nil {
            hasMatrixStageSteps = true
            break
        }
    }
    
    if hasMatrixStageSteps {
        // Matrix-first expansion path
        expandedSteps, interpolatedActions, poolConfigs, err := r.expandMatrixStageStepsWithPools(stage.Steps)
        // Inject sliding window dependencies
        expandedSteps = r.injectMatrixJobDependencies(expandedSteps, poolConfigs)
    } else {
        // Existing path: stage expansion → matrix expansion
        stepsWithExpandedStages, err := r.expandStageReferences(stage.Steps)
        expandedSteps, interpolatedActions, poolConfigs, err := r.expandMatrixStepsWithPools(stepsWithExpandedStages)
    }
    
    // Continue with DAG construction and execution...
}
```

### Phase 4: Data Structures

#### 4.1 MatrixJob Structure

```go
type MatrixJob struct {
    Index      int                // Job index (0-based)
    MatrixVars map[string]string // Matrix variables for this job
    Steps      []Step             // All steps in this job
    FirstSteps []string           // Step names that can start the job
    LastSteps  []string           // Step names that complete the job
}
```

#### 4.2 Matrix Stage Expansion Result

```go
type MatrixStageExpansion struct {
    Jobs              []MatrixJob
    AllSteps          []Step
    InterpolatedActions map[string]*Action
    PoolConfigs       map[string]int
}
```

## Examples

### Example 1: Sequential Matrix Execution

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

**Expansion Result:**
- Matrix Job 1 (gcc): All 6 build steps with `matrix.compiler=gcc`
- Matrix Job 2 (clang): All 6 build steps with `matrix.compiler=clang`
- Dependency: First step of clang job depends on last step (cmake-test) of gcc job

**Execution Order:**
```
gcc:
  check-conan | check-cmake
    conan-install
      cmake-config
        cmake-build
          cmake-test  ← completes

clang:  ← starts after gcc completes
  check-conan | check-cmake
    conan-install
      cmake-config
        cmake-build
          cmake-test
```

### Example 2: Parallel Matrix Execution with Sliding Window

```yaml
compiler-build:
  steps:
    - stage: build
      matrix:
        values:
          compiler: ["gcc", "clang", "msvc", "icc"]
      strategy:
        max_parallel: 2
```

**Expansion Result:**
- 4 matrix jobs (gcc, clang, msvc, icc)
- Dependency injection:
  - Job 1 (gcc): No dependency
  - Job 2 (clang): No dependency
  - Job 3 (msvc): Depends on completion of Job 1 (gcc)
  - Job 4 (icc): Depends on completion of Job 2 (clang)

**Execution Timeline:**
```
Time:    [========] [========] [========] [========]
gcc:     [running─────────────────────────]
clang:              [running──────────────]
msvc:                         [waiting──] [running──]
icc:                                    [waiting─] [running─]
```

### Example 3: Matrix Variables in Stage Steps

```yaml
actions:
  - name: configure-compiler
    run: |
      echo "Setting up ${{ matrix.compiler }}"
      export CC=${{ matrix.compiler }}

stages:
  build:
    steps:
      - action: configure-compiler
      - action: build-source

  multi-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: ["gcc", "clang"]
```

**Matrix Variable Propagation:**
- Each expanded step in the build stage has `matrix.compiler` available
- Variable interpolation works in all action commands within the expanded stage

## Testing Strategy

### Unit Tests

1. **Matrix Stage Expansion Tests**
   - Test matrix expansion with stage references
   - Test matrix variable propagation
   - Test step boundary identification (first/last steps)

2. **Dependency Injection Tests**
   - Test sliding window dependency injection for various max_parallel values
   - Test sequential execution (max_parallel = 1)
   - Test parallel execution (max_parallel > 1)

3. **Integration Tests**
   - Test complete execution flow with matrix stage steps
   - Test error handling when matrix jobs fail
   - Test pool assignment and concurrency limits

### Example Test Cases

```go
func TestMatrixStageExpansion(t *testing.T) {
    // Test expanding stage with matrix
    // Verify matrix variables are available in all steps
    // Verify internal dependencies are preserved
}

func TestSlidingWindowDependencies(t *testing.T) {
    // Test max_parallel = 1: sequential execution
    // Test max_parallel = 2: parallel execution with dependencies
    // Test max_parallel = 0: no dependencies (unlimited)
}

func TestMatrixStageExecution(t *testing.T) {
    // End-to-end test: execute stage with matrix
    // Verify execution order matches dependencies
    // Verify all matrix jobs complete successfully
}
```

## Edge Cases and Considerations

### 1. Nested Stage References

**Scenario**: Matrix stage step references a stage that itself references another stage

**Solution**: Stage expansion should recurse, preserving matrix variables at each level

### 2. Matrix Jobs with Different Step Counts

**Scenario**: Stage expansion might result in different step counts per matrix job (due to conditional steps)

**Solution**: First/last step identification must handle variable step counts per job

### 3. Empty Matrix Values

**Scenario**: Matrix expansion results in empty or skipped jobs

**Solution**: Handle skipped jobs in dependency injection (skip them in the sliding window)

### 4. Matrix with Multiple Dimensions

**Scenario**: Future support for multi-dimensional matrices on stage steps

**Solution**: Current implementation supports single-dimension only, but structure allows extension

## Documentation Updates

### 1. Update Matrix Feature Documentation

**File**: `docs/Matrix-feature.md`

- Add section on matrix with stage steps
- Update limitations section (remove "stage-level only" limitation)
- Add examples of matrix on stage steps

### 2. Update YAML Syntax Reference

**File**: `docs/YAML-syntax-reference.md`

- Update stage references section to show matrix configuration
- Add examples of matrix on stage steps
- Document sliding window dependency behavior

### 3. Update Project Specification

**File**: `docs/Project-specification.md`

- Update matrix feature section
- Document matrix on stage step support
- Update examples and use cases

## Implementation Checklist

### Phase 1: Core Implementation
- [ ] Implement `ExpandMatrixStageStep()` function
- [ ] Update `expandMatrixStepsWithPools()` to handle stage steps
- [ ] Test matrix expansion with stage references
- [ ] Test matrix variable propagation

### Phase 2: Dependency Injection
- [ ] Implement `findFirstSteps()` and `findLastSteps()`
- [ ] Implement `injectSlidingWindowDependencies()`
- [ ] Test dependency injection algorithm
- [ ] Test various max_parallel scenarios

### Phase 3: Integration
- [ ] Update `runStageInternal()` with matrix-first expansion path
- [ ] Integrate with existing execution pipeline
- [ ] Test end-to-end execution flow
- [ ] Handle edge cases (nested stages, conditional steps)

### Phase 4: Testing and Documentation
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation
- [ ] Add examples to documentation

## Future Enhancements

1. **Multi-dimensional Matrix on Stages**: Support multiple matrix dimensions on stage steps
2. **Matrix Job Dependencies**: Allow explicit dependencies between matrix jobs
3. **Conditional Matrix Jobs**: Skip matrix jobs based on conditions
4. **Matrix Job Filtering**: Filter matrix jobs at runtime
5. **Matrix Includes/Excludes**: Include or exclude specific matrix combinations

## Conclusion

This implementation plan provides a comprehensive approach to enabling matrix configuration on stage steps. The matrix-first expansion approach combined with sliding window dependency injection ensures correct execution order while maintaining flexibility and performance. The design leverages the existing DAG executor, requiring no changes to the core execution engine.

