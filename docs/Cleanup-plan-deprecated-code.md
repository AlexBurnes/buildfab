# Cleanup Plan: Deprecated Flat DAG Code

## Overview

After implementing the hierarchical DAG architecture in v0.32.0, several functions in `pkg/buildfab/buildfab.go` are no longer used. This document provides a detailed plan for safely removing this deprecated code.

## Status

**Current Version**: v0.32.0  
**Cleanup Status**: ✅ Complete (Unreleased)  
**Risk Level**: Low (code was verified unused)
**Actual Removal**: 1,697 lines total (1,317 from buildfab.go + 380 from test file)

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

1. ✅ **v0.32.0** (Complete): Mark as deprecated, document in code, create cleanup plan
2. ✅ **v0.32.0** (Complete - Unreleased): Remove all deprecated code (1,697 lines)
3. 🎯 **Next Release**: Publish cleaned codebase
4. **v1.0.0** (Future major): Stable API with long-term support

## Recommendations

### ✅ Completed Actions (v0.32.0 - Unreleased)

All cleanup tasks completed:
1. ✅ Removed deprecated code from `simple.go` (196 lines)
2. ✅ Removed 11 deprecated functions from `buildfab.go` (1,317 lines)
3. ✅ Removed deprecated test file `matrix_stage_test.go` (380 lines)
4. ✅ All tests passing with race detection
5. ✅ Zero race conditions detected
6. ✅ Code compiles successfully
7. ✅ Updated CHANGELOG.md with cleanup details
8. ✅ Updated cleanup plan document
9. ✅ Verified zero external dependencies on removed code

**Total Cleanup**: 1,697 lines removed

### Next Steps (Next Release)

1. **Tag and release** the cleaned codebase
   - All deprecated code already removed
   - No breaking changes for documented API users
   - Potential breaking change only if external code used undocumented `RunStageWithSteps()`

2. **Monitor feedback** after release
   - Check for any issues from external users
   - Provide migration guidance if needed (though unlikely)

## Verification Checklist

✅ **All verification steps completed**:

- [x] Search codebase for direct calls to deprecated functions ✅ None found
- [x] Check pre-push project for usage ✅ Uses only documented API
- [x] Run full test suite: `go test ./... -v -race` ✅ All tests pass
- [x] Build all packages: `go build ./...` ✅ Compiles successfully
- [x] Test example configurations ✅ Working correctly
- [x] Update CHANGELOG.md ✅ Updated with cleanup details
- [x] Update Library-API.md (if needed) ✅ Already documented current API
- [x] Update Cleanup-plan-deprecated-code.md ✅ Completed

## Files to Update

✅ **All files updated successfully**:

### Code Files
- [x] `pkg/buildfab/simple.go` - ✅ Cleaned (196 lines removed)
- [x] `pkg/buildfab/buildfab.go` - ✅ Cleaned (1,317 lines removed)

### Test Files
- [x] `pkg/buildfab/matrix_stage_test.go` - ✅ Deleted (380 lines, deprecated tests)

### Documentation Files
- [x] `docs/Library-API.md` - ✅ Created with current API
- [x] `docs/Cleanup-plan-deprecated-code.md` - ✅ Created and updated
- [x] `CHANGELOG.md` - ✅ Updated with cleanup details

## Success Criteria

✅ **Phase 1 (v0.32.0 - Complete)**:
- [x] Removed deprecated code from `simple.go` (196 lines)
- [x] All tests pass with race detection
- [x] Created Library-API.md (782 lines)
- [x] Created cleanup plan document (376 lines)

✅ **Phase 2 (v0.32.0 - Complete)**:
- [x] Remove all deprecated functions from `buildfab.go` ✅ 11 functions removed
- [x] Reduce codebase significantly ✅ 1,697 lines removed total
- [x] All tests still pass ✅ 100% passing with race detection
- [x] No compilation errors ✅ Compiles successfully
- [x] Examples still work ✅ All examples functional
- [x] Documentation updated ✅ CHANGELOG and cleanup plan updated

**Final Results**: Both phases completed successfully in v0.32.0 (unreleased)

## Cleanup Completion Summary

✅ **Cleanup completed successfully** (Unreleased, to be included in next version):

### What Was Removed
- **11 deprecated functions** from `pkg/buildfab/buildfab.go` (1,317 lines)
- **1 deprecated test file** `pkg/buildfab/matrix_stage_test.go` (380 lines)  
- **Total**: 1,697 lines removed

### Results
- ✅ **All tests passing** with race detection
- ✅ **Zero race conditions** detected
- ✅ **Code compiles** successfully
- ✅ **27% reduction** in buildfab.go (4,845 → 3,528 lines)
- ✅ **Zero external dependencies** on removed code (verified)

### Impact
- **Significantly reduced code complexity**
- **Improved maintainability** (single execution path)
- **Eliminated confusion** about which execution path to use
- **No breaking changes** for documented API users

## Original Plan vs Actual

**Planned**: ~2,210 lines (estimated with supporting functions)  
**Actual**: 1,697 lines (11 functions + 1 test file)

**Difference**: The original estimate included some lines that were part of comments and documentation. The actual removal was more precise, focusing on the deprecated functions themselves.

## Conclusion

The deprecated flat DAG code has been successfully removed:
- **1,697 lines** of unused code eliminated
- **Zero external dependencies** verified
- **All tests passing** with new hierarchical DAG
- **Clean codebase** with single execution path

The removal significantly reduced code complexity, improved maintainability, and eliminated confusion about which execution path to use.

**Status**: ✅ Cleanup complete, ready for next release.

