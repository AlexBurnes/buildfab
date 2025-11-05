# ADR 005: Hierarchical DAG Architecture for Matrix Execution

## Status
Proposed

## Context

The current flat DAG architecture has fundamental limitations when handling:
1. **Nested matrices**: Matrix on stage referencing stages with matrix steps
2. **Multi-step matrix jobs**: No implicit ordering between steps in expanded stages
3. **Sliding window dependencies**: Must inject dependencies between individual steps
4. **Condition-based skipping**: Skipped steps block unrelated steps via dependency chains

### Current Architecture Problems

**Flat DAG Example:**
```
DAG = [step1, step2, step3, step4, step5, step6, ...]
         ↓      ↓      ↓      ↓      ↓      ↓
    All steps at same level, dependencies manually linked
```

**Issues:**
- When expanding `stage: build-stage` with matrix, steps within each job have no inherent relationship
- `findFirstSteps`/`findLastSteps` must guess job boundaries based on dependencies
- Sliding window injects dependencies between ALL first/last steps, not jobs
- A condition-skipped step can block unrelated steps in other jobs

### Real-World Example That Fails

```yaml
stages:
  build-stage:
    steps:
      - action: build
        if: "!(matrix.builds == 'DebugWithRelInfo')"
      - action: cleanup
      - action: test

  matrix-image-build:
    steps:
      - stage: build-stage
        matrix:
          values:
            compiler: ["gcc", "clang"]
            builds: ["Release", "Debug", "DebugWithRelInfo"]
          strategy:
            max_parallel: 1
```

**Expected**: 6 jobs, each with 3 steps (18 total)
- Jobs with `builds=DebugWithRelInfo` skip the build step but run cleanup/test
- Other jobs run all 3 steps

**Current Behavior**: Hangs because flat DAG can't properly represent this

## Decision

Implement a **hierarchical DAG** where jobs are first-class citizens containing sequential steps.

### New Architecture

```
HierarchicalDAG
├─ Job 0 (gcc+Release)
│   ├─ Step 0: build
│   ├─ Step 1: cleanup (implicitly depends on Step 0)
│   └─ Step 2: test (implicitly depends on Step 1)
│
├─ Job 1 (clang+Release) [sliding window: depends on Job 0]
│   ├─ Step 0: build
│   ├─ Step 1: cleanup
│   └─ Step 2: test
│
└─ Job 2 (gcc+Debug) [sliding window: depends on Job 1]
    └─ ChildJob 2.0 (nested matrix)
        └─ ...
```

### Key Principles

1. **Jobs are the unit of parallelism**: Sliding window controls job concurrency
2. **Steps within jobs are sequential**: Implicit ordering by declaration
3. **Two types of dependencies**:
   - **Explicit** (user-defined via `require:`): Block on any failure/skip
   - **Sliding window** (auto-injected): Don't block on condition-based skips
4. **Hierarchical composition**: Jobs can contain child jobs for nested matrices

## Implementation Plan

### Phase 1: Core Data Structures
- New file: `pkg/buildfab/job_node.go`
- Define `JobNode`, `ExecutableStep`, `JobStatus`, `HierarchicalDAG`
- Basic constructors and helper methods

### Phase 2: Job-Based Matrix Expansion
- New file: `pkg/buildfab/job_expander.go`
- `ExpandMatrixToJobs()` - creates job nodes from matrix config
- `ExpandMatrixStageToJobs()` - creates multi-step jobs
- Handle nested matrices by creating child jobs

### Phase 3: Hierarchical Executor
- New file: `pkg/buildfab/hierarchical_executor.go`
- Execute jobs in topological order with wave-based parallelism
- Execute steps sequentially within each job
- Respect `max_parallel` at job level
- Implement `fail_fast` logic

### Phase 4: Job-Aware Output Manager
- New file: `pkg/buildfab/job_output_manager.go`
- Display jobs with collapsible step output
- Use step IDs internally, display names for users
- Show job status summary

### Phase 5: Integration
- Add feature flag: `--experimental-hierarchical-dag`
- Update `RunStage()` to use hierarchical DAG when flag is set
- Comprehensive integration tests
- Performance comparison with flat DAG

### Phase 6: Migration
- Make hierarchical DAG the default
- Keep flat DAG as `--legacy-dag` fallback
- Update all documentation
- Eventually remove flat DAG code

## Consequences

### Positive

✅ **Correctness**: Proper semantics for nested matrices and multi-step jobs
✅ **Clarity**: Job boundaries are explicit, not inferred
✅ **Maintainability**: Cleaner code with better separation of concerns
✅ **Extensibility**: Easy to add job-level features (retries, timeouts, resource limits)
✅ **Performance**: Better parallelism control, less overhead
✅ **User Experience**: Clearer output showing job hierarchy

### Negative

❌ **Breaking change**: Requires significant refactoring
❌ **Migration effort**: Existing code must be updated
❌ **Testing burden**: New architecture needs extensive testing
❌ **Learning curve**: Contributors must understand hierarchical model

### Neutral

⚪ **Code size**: Similar LOC but different organization
⚪ **Performance**: Similar for simple cases, better for complex ones

## Alternatives Considered

### Alternative 1: Add Implicit Sequential Dependencies

**Approach**: When expanding stages, automatically add `require:` between steps

**Pros**: Minimal code changes, works within existing architecture

**Cons**: 
- Still fundamentally flat
- Doesn't solve nested matrix issues
- Doesn't fix sliding window problems
- Technical debt accumulates

### Alternative 2: Job Wrapper Around Flat DAG

**Approach**: Group steps into "virtual jobs" but keep flat execution

**Pros**: Smaller refactoring, preserves some existing code

**Cons**:
- Complexity of maintaining two mental models
- Doesn't fully solve the problems
- Confusing abstraction layer

### Alternative 3: Document Limitations

**Approach**: Mark nested matrices as unsupported, focus on simple cases

**Pros**: Zero implementation cost

**Cons**:
- Limits product capabilities
- User frustration with edge cases
- Technical debt forever

## Decision Rationale

The hierarchical DAG architecture is the **correct long-term solution** because:

1. **Semantic Correctness**: Jobs are a real concept in matrix execution, not an implementation detail
2. **Industry Standard**: GitHub Actions, GitLab CI, and other CI systems use job-based models
3. **Future-Proof**: Enables features like job-level retries, resource allocation, distributed execution
4. **Clean Architecture**: Separation between job orchestration and step execution

## References

- GitHub Actions: Jobs contain steps, run in parallel by default
- GitLab CI: Stages contain jobs,  each job runs sequentially
- Current issues: `tests/test_matrix_skiped.yml` matrix-image-build hangs

## Timeline

- **Design & ADR**: 1-2 hours ← **WE ARE HERE**
- **Phase 1-2 (Data structures & expansion)**: 3-4 hours
- **Phase 3-4 (Executor & output)**: 3-4 hours
- **Phase 5-6 (Integration & migration)**: 2-3 hours
- **Total**: ~8-12 hours

## Next Steps

1. ✅ Create design document
2. ✅ Create this ADR
3. ⏭️ Implement `JobNode` and `HierarchicalDAG` data structures
4. ⏭️ Implement job-based matrix expander
5. ⏭️ Implement hierarchical executor
6. ⏭️ Update output manager for jobs
7. ⏭️ Integration and testing
8. ⏭️ Documentation updates

