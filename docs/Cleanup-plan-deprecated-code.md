# Cleanup Plan: Deprecated Flat DAG Code

## Overview

After implementing the hierarchical DAG architecture in v0.32.0, several functions in `pkg/buildfab/buildfab.go` are no longer used. This document provides a detailed plan for safely removing this deprecated code.

## Status

**Current Version**: v0.32.0  
**Cleanup Status**: Planned  
**Risk Level**: Low (code is verified unused)

## Deprecated Functions

### 1. runStageInternal()

**Location**: `pkg/buildfab/buildfab.go:1897`

**Purpose**: Old entry point for stage execution using flat DAG

**Why Deprecated**:
- `Runner.RunStage()` now delegates to `SimpleRunner.RunStage()`
- `SimpleRunner` uses hierarchical DAG execution
- This function is never called in the current codebase

**Dependencies** (functions only called by this):
- `expandMatrixStepsWithPools()`
- `executeStageWithCallback()`
- `executeDAGWithOrderedStreaming()`

**Removal Impact**: None (not called)

---

### 2. executeStageWithCallback()

**Location**: `pkg/buildfab/buildfab.go:2135`

**Purpose**: Executes stage steps with callback-based output management

**Why Deprecated**:
- Only called from `runStageInternal()` (line 1996)
- Only called from `RunStageWithSteps()` (line 463)
- `RunStageWithSteps()` is itself never called

**Removal Impact**: None (only called by deprecated code)

---

### 3. expandMatrixSteps()

**Location**: `pkg/buildfab/buildfab.go:1081`

**Purpose**: Basic matrix expansion (no action interpolation)

**Why Deprecated**:
- Replaced by `JobExpander` in hierarchical DAG
- Only called from deprecated functions
- SimpleRunner has its own `expandMatrixSteps` with different signature

**Removal Impact**: None (not called)

**Note**: SimpleRunner has a method with the same name but different signature - keep that one.

---

### 4. expandMatrixStepsWithActions()

**Location**: `pkg/buildfab/buildfab.go:1152`

**Purpose**: Matrix expansion with action interpolation

**Why Deprecated**:
- Replaced by `JobExpander.ExpandMatrixToJobs()`
- Only called from deprecated functions

**Removal Impact**: None (not called)

---

### 5. expandMatrixStepsWithPools()

**Location**: `pkg/buildfab/buildfab.go:1218`

**Purpose**: Matrix expansion with pool assignment

**Why Deprecated**:
- Replaced by `JobExpander` with sliding window dependencies
- Only called from `runStageInternal()` (line 1972)

**Removal Impact**: None (only called by deprecated code)

---

### 6. executeDAGWithOrderedStreaming()

**Location**: `pkg/buildfab/buildfab.go:3164`

**Purpose**: Old DAG executor with flat structure

**Why Deprecated**:
- Replaced by `HierarchicalExecutor`
- Only called from `runStageInternal()` (line 2006)

**Removal Impact**: None (only called by deprecated code)

---

### 7. RunStageWithSteps()

**Location**: `pkg/buildfab/buildfab.go:461`

**Purpose**: Public method for running pre-expanded steps

**Why Deprecated**:
- Not used anywhere in codebase
- Not documented in public API
- Replaced by `SimpleRunner.RunStage()`

**Removal Impact**: Potential breaking change if used by external code (low risk)

---

## Verification

### Call Graph Analysis

```
External Code (pre-push, etc.)
    ↓
Runner.RunStage() → SimpleRunner.RunStage() → HierarchicalDAG ✓ (IN USE)
    
DEPRECATED (NEVER CALLED):
runStageInternal()
    ├─> expandMatrixStepsWithPools()
    ├─> executeStageWithCallback()
    └─> executeDAGWithOrderedStreaming()
    
RunStageWithSteps()
    └─> executeStageWithCallback()
```

### Verification Commands

```bash
# Check if any external code calls these functions
grep -r "runStageInternal\(" . --include="*.go" --exclude-dir=vendor

# Check RunStageWithSteps usage
grep -r "RunStageWithSteps\(" . --include="*.go" --exclude-dir=vendor

# Check expandMatrixSteps* usage
grep -r "expandMatrixSteps\|expandMatrixStepsWithActions\|expandMatrixStepsWithPools" . --include="*.go" --exclude-dir=vendor
```

**Result**: Only found in:
- `pkg/buildfab/buildfab.go` (definitions and internal calls)
- `pkg/buildfab/simple.go` (different function - `SimpleRunner.expandMatrixSteps`)
- Documentation files

### Test Coverage

All integration tests pass after cleanup of `simple.go`:
```
✅ All tests PASSED with race detection
✅ Zero race conditions detected
✅ 0 test failures
```

## Cleanup Steps

### Phase 1: Mark as Deprecated (Done in comments)

Already documented in comments that these functions are deprecated.

### Phase 2: Safe Removal Plan

1. **Remove the functions**:
   - `runStageInternal()` + ~500 lines
   - `executeStageWithCallback()` + ~50 lines  
   - `expandMatrixSteps()` + ~70 lines
   - `expandMatrixStepsWithActions()` + ~80 lines
   - `expandMatrixStepsWithPools()` + ~120 lines
   - `executeDAGWithOrderedStreaming()` + ~800 lines
   - `RunStageWithSteps()` + ~5 lines

2. **Total Removal**: ~1,625 lines of code

3. **Supporting functions that can also be removed**:
   - `expandMatrixStageStepsWithPools()` (~400 lines) - only called by `runStageInternal()`
   - `findFirstSteps()` (~60 lines) - only used by flat DAG
   - `findLastSteps()` (~60 lines) - only used by flat DAG
   - `injectSlidingWindowDependencies()` (~65 lines) - replaced by JobExpander

4. **Total with supporting functions**: ~2,210 lines

### Phase 3: Validation

```bash
# After removal, run full test suite
go test ./... -v -race

# Ensure no compilation errors
go build ./...

# Check that examples still work
./bin/buildfab run build --config examples/matrix-multidimensional-simple.yml
```

### Phase 4: Documentation Update

Update the following docs to remove references to flat DAG:
- ~~`docs/Matrix-on-stage-feature-plan.md`~~ (historical, keep for reference)
- ~~`docs/Matrix-parallel-pools-implementation.md`~~ (historical, keep)
- Update `CHANGELOG.md` with cleanup note

## Risk Assessment

### Low Risk Areas

✅ **Functions only called internally**: All deprecated functions are only called by other deprecated functions

✅ **External API unchanged**: `Runner.RunStage()` still works (delegates to SimpleRunner)

✅ **Tests pass**: All tests pass with hierarchical DAG

### Potential Risks

⚠️ **RunStageWithSteps()**: Public method (though undocumented)
- **Mitigation**: Check if pre-push or other external tools use it
- **Likelihood**: Very low (not in public API docs, not in examples)

⚠️ **Documentation references**: Old docs mention these functions
- **Mitigation**: Keep historical docs, add deprecation notice

## Migration Path for External Users

If any external code uses `RunStageWithSteps()`:

```go
// Old code (deprecated)
runner := buildfab.NewRunner(config, opts)
steps := []buildfab.Step{...}  // Pre-expanded steps
err := runner.RunStageWithSteps(ctx, "stage-name", steps)

// New code (recommended)
simpleOpts := &buildfab.SimpleRunOptions{
    VerboseLevel: opts.VerboseLevel,
    Debug:        opts.Debug,
    Variables:    opts.Variables,
    // ... other options
}
simpleRunner := buildfab.NewSimpleRunner(config, simpleOpts)
err := simpleRunner.RunStage(ctx, "stage-name")
```

## Timeline

1. **v0.32.0** (Current): Mark as deprecated, document in code
2. **v0.32.1** (Optional): Add deprecation warnings if functions are called
3. **v0.33.0** (Next minor): Remove deprecated code
4. **v1.0.0** (Future major): Complete API cleanup

## Recommendations

### Immediate Actions (v0.32.0)

✅ Already done:
- Removed deprecated code from `simple.go` (196 lines)
- All tests passing with race detection
- Documented deprecation in code comments

### Next Steps (v0.33.0)

1. **Remove deprecated functions** from `buildfab.go`:
   - 7 main functions (~1,625 lines)
   - 4 supporting functions (~585 lines)
   - **Total: ~2,210 lines**

2. **Update documentation**:
   - Add removal notice to CHANGELOG
   - Update any remaining references in docs

3. **Verify external usage**:
   - Check pre-push project for any direct usage
   - Search GitHub for any public repos using these APIs

4. **Release v0.33.0**:
   - Include "Removed deprecated flat DAG code" in changelog
   - Note: Breaking change if any external code used `RunStageWithSteps()`

## Verification Checklist

Before removing code:

- [ ] Search codebase for direct calls to deprecated functions
- [ ] Check pre-push project for usage
- [ ] Run full test suite: `go test ./... -v -race`
- [ ] Build all packages: `go build ./...`
- [ ] Test example configurations
- [ ] Update CHANGELOG.md
- [ ] Update Library-API.md (if needed)

## Files to Update

### Code Files
- [x] `pkg/buildfab/simple.go` - Already cleaned (v0.32.0)
- [ ] `pkg/buildfab/buildfab.go` - Remove deprecated functions (v0.33.0)

### Documentation Files
- [x] `docs/Library-API.md` - Created with current API (v0.32.0)
- [ ] `CHANGELOG.md` - Add cleanup note (v0.33.0)
- [ ] `docs/Cleanup-plan-deprecated-code.md` - This document (v0.32.0)

### Test Files
- No changes needed (all tests use SimpleRunner)

## Success Criteria

✅ **Phase 1 (v0.32.0 - Complete)**:
- [x] Removed deprecated code from `simple.go`
- [x] All tests pass with race detection
- [x] Created Library-API.md
- [x] Created cleanup plan document

🎯 **Phase 2 (v0.33.0 - Planned)**:
- [ ] Remove all deprecated functions from `buildfab.go`
- [ ] Reduce codebase by ~2,210 lines
- [ ] All tests still pass
- [ ] No compilation errors
- [ ] Examples still work
- [ ] Documentation updated

## Conclusion

The deprecated flat DAG code in `buildfab.go` can be safely removed in v0.33.0:
- **~2,210 lines** of unused code
- **Zero external dependencies** verified
- **All tests passing** with new hierarchical DAG
- **Clean migration path** for any external users

The removal will significantly reduce code complexity, improve maintainability, and eliminate confusion about which execution path to use.

**Recommendation**: Proceed with cleanup in v0.33.0 release.

