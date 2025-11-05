# Feature Documentation Review

## Overview

This document compares features mentioned in `docs/Release-announcement.md` with their documentation in user-facing documents to identify gaps and ensure all features are properly documented.

**Review Date**: November 5, 2025  
**Current Version**: v0.32.0

## Comparison Summary

| Feature | Release Announcement | User Docs | Status | Notes |
|---------|---------------------|-----------|--------|-------|
| **v0.32.0 Features** |
| Hierarchical DAG Architecture | ✅ Highlighted | ❌ Not documented | **GAP** | Major feature, needs user docs |
| Code Cleanup (internal) | ✅ Mentioned | N/A | ✅ OK | Internal change, not user-facing |
| Library Documentation | ✅ Mentioned | ✅ docs/Library.md | ✅ OK | Complete |
| Documentation Organization | ✅ Mentioned | ✅ docs/User.md | ✅ OK | Complete |
| Expression Parsing (NOT operator) | ✅ Mentioned | ❓ Partial | **UPDATE** | Exists but NOT fix not highlighted |
| Enhanced Matrix Builds | ✅ Mentioned | ✅ Partial | **UPDATE** | Nested matrix not explicitly called out |
| **Core Features** |
| Matrix Builds | ✅ Yes | ✅ Yes | ✅ OK | Well documented |
| Multi-Dimensional Matrices | ✅ Yes | ✅ Yes | ✅ OK | Comprehensive docs |
| Matrix on Stages | ✅ Yes | ✅ Yes | ✅ OK | Documented |
| Container Support | ✅ Yes | ❌ Limited | **GAP** | Mentioned but not detailed |
| Pool-Based Parallelism | ✅ Yes | ❓ Partial | **UPDATE** | `max_parallel` documented, pools not explained |
| File Caching | ✅ Yes | ✅ Yes | ✅ OK | docs/User/Caching.md exists |
| Artifact Collection | ✅ Yes | ❓ Unknown | **CHECK** | Need to verify |
| Expression Language | ✅ Yes | ✅ Yes | ✅ OK | Documented in Features |
| Platform Detection | ✅ Yes | ✅ Yes | ✅ OK | Documented |
| Include System | ✅ Yes | ✅ Yes | ✅ OK | Documented |
| Action Variants | ✅ Yes | ✅ Yes | ✅ OK | Documented |
| Built-in Actions | ✅ Yes | ✅ Yes | ✅ OK | Documented |
| Library API | ✅ Yes | ✅ Yes | ✅ OK | docs/Library.md complete |

## Detailed Gap Analysis

### 🔴 Critical Gaps (User-Facing Features Not Documented)

#### 1. Hierarchical DAG Architecture

**In Release Announcement**:
- Job-based execution model
- Jobs execute in parallel waves
- Steps within jobs execute sequentially
- Nested matrix support
- Proper dependency waiting

**In User Docs**:
- ❌ Not mentioned
- DAG execution described as "parallel execution with dependencies"
- No explanation of job vs step execution model

**Recommendation**: Add section to Project-specification.md explaining:
- What is a "job" vs a "step"
- How matrix builds create jobs
- How hierarchical execution improves nested matrices
- Execution model diagram

---

#### 2. Container Support (Detailed)

**In Release Announcement**:
- Native Docker/Podman integration
- Resource management
- Container configuration
- Volume mounting
- Environment passing

**In User Docs**:
- ❌ Not detailed
- Mentioned in core features list
- No comprehensive container configuration reference
- No examples of container usage

**Recommendation**: Add to Features-and-examples.md:
- Container configuration syntax
- Docker/Podman engine selection
- Volume mounting patterns
- Environment variable passing
- Resource limits (CPU, memory)
- Image pull policies
- Container networking

---

### 🟡 Partial Documentation (Needs Enhancement)

#### 3. Pool-Based Parallelism

**In Release Announcement**:
- Global and matrix-specific concurrency control
- Worker pool system
- Task queues

**In User Docs**:
- ✅ `max_parallel` setting documented
- ❌ Pool concept not explained
- ❌ Matrix pool isolation not described

**Recommendation**: Add to Project-specification.md:
- How pools control concurrency
- Global pool vs matrix pools
- How `min(global, matrix)` strategy works

---

#### 4. Expression Language - NOT Operator Fix

**In Release Announcement**:
- Fixed `!` operator parsing
- Complex expressions like `!(matrix.images == 'centos7' && matrix.compiler == 'clang')`
- Operator precedence

**In User Docs**:
- ✅ Expression language documented
- ✅ NOT operator mentioned
- ❌ v0.32.0 improvements not highlighted
- ❌ Complex nested expressions not emphasized

**Recommendation**: Add example in Features-and-examples.md:
- Complex NOT expressions
- Operator precedence examples
- Matrix variable conditions

---

#### 5. Enhanced Matrix Builds (v0.32.0 Improvements)

**In Release Announcement**:
- Matrix variables propagate in nested scenarios
- Condition-based skips don't block sliding window
- User dependencies inherited by matrix jobs
- Global max_parallel enforced

**In User Docs**:
- ✅ Matrix builds documented
- ✅ Multi-dimensional matrices documented
- ❌ v0.32.0 enhancements not highlighted
- ❌ Sliding window dependency concept not explained

**Recommendation**: Add to Features-and-examples.md:
- Section on "Advanced Matrix Features (v0.32.0)"
- Explain sliding window dependencies
- Show nested matrix examples
- Condition skip behavior with dependencies

---

### 🟢 Well Documented (No Action Needed)

These features are well documented:

1. ✅ **Matrix Builds**: Comprehensive documentation with examples
2. ✅ **Multi-Dimensional Matrices**: Detailed in Features-and-examples.md
3. ✅ **Matrix on Stages**: Documented with examples
4. ✅ **Variable Interpolation**: Complete syntax and examples
5. ✅ **Built-in Actions**: All documented with usage examples
6. ✅ **Include System**: Syntax and behavior documented
7. ✅ **Action Variants**: Platform-specific execution documented
8. ✅ **Conditional Execution**: Expression language documented
9. ✅ **Library API**: Complete docs/Library.md (782 lines)
10. ✅ **DAG Execution**: Basic concept documented
11. ✅ **Error Policies**: stop/warn documented
12. ✅ **Cross-Platform**: Platform detection documented

---

## Recommendations

### High Priority Updates

#### 1. Add Hierarchical DAG Section to Project-specification.md

```markdown
### Hierarchical DAG Architecture (v0.32.0+)

buildfab uses a hierarchical DAG execution model:

**Execution Model**:
- **Jobs**: Matrix combinations create individual job nodes
- **Steps**: Each job contains a sequence of steps to execute
- **Parallelism**: Jobs execute in parallel waves, steps sequentially within jobs
- **Dependencies**: Jobs depend on other jobs, not individual steps

**Benefits**:
- Better handling of nested matrices (matrix on stage reference)
- Improved parallelism control
- No more hanging scenarios in complex matrix builds
- Clearer execution model

**Example**:
```yaml
stages:
  cross-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: [gcc, clang]
# Creates 2 jobs, each job runs all steps from "build" stage
```
```

#### 2. Add Container Support Section to Features-and-examples.md

```markdown
## Container Support

buildfab provides native container integration with Docker and Podman:

### Basic Container Action

```yaml
actions:
  - name: test-in-container
    container:
      image:
        from: golang:1.22
        pull: missing
      run_action: test
      volumes:
        - $PWD:/workspace
      environment:
        - GOMODCACHE=/workspace/.cache
```

### Container Configuration Options

**Image**:
- `from`: Container image name
- `pull`: Pull policy (always, missing, never)

**Resources**:
- `cpu`: CPU limit
- `memory`: Memory limit
- `network`: Network mode (host, bridge, none)

**Volumes**:
- Mount workspace and cache directories
- Syntax: `<host-path>:<container-path>`

**Environment**:
- Pass environment variables to container
- Syntax: `KEY=value` or `KEY` (from host)

### Engine Selection

buildfab automatically detects available engines:
1. Docker
2. Podman

Specify engine explicitly:
```yaml
actions:
  - name: build-image
    container:
      engine: podman
      ...
```
```

#### 3. Add Advanced Matrix Section to Features-and-examples.md

```markdown
## Advanced Matrix Features (v0.32.0)

### Nested Matrix Support

Matrix on stage references now works correctly:

```yaml
stages:
  build:
    steps:
      - action: configure
      - action: compile
      - action: test

  multi-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: [gcc, clang]
            version: ["11", "12"]
# Creates 4 jobs (gcc-11, gcc-12, clang-11, clang-12)
# Each job runs: configure → compile → test
```

### Sliding Window Dependencies

When using `max_parallel` with matrix, buildfab creates sliding window dependencies:

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
# Only 2 platforms test at once, third waits for first to complete
```

### Condition-Based Skips

Condition skips don't block sliding window dependencies:

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
# centos7-clang skips, but doesn't block centos8-gcc from starting
```
```

---

### Medium Priority Updates

#### 4. Update YAML-syntax-reference.md

Add sections for:
- Container configuration syntax
- Matrix strategy options
- Job vs Step execution model

#### 5. Update Comparison-with-others.md

Add v0.32.0 advantages:
- Hierarchical DAG vs flat DAG (GitHub Actions)
- Better nested matrix support than competitors

---

## Documentation Update Plan

### Phase 1: Critical Gaps (Immediate)

1. ✅ Create this review document
2. **TODO**: Add "Hierarchical DAG" section to Project-specification.md
3. **TODO**: Add "Container Support" section to Features-and-examples.md
4. **TODO**: Add "Advanced Matrix Features" section to Features-and-examples.md

### Phase 2: Enhancements (Next)

5. **TODO**: Update YAML-syntax-reference.md with container syntax
6. **TODO**: Add pool/concurrency explanation to Project-specification.md
7. **TODO**: Update Comparison-with-others.md with v0.32.0 advantages

### Phase 3: Polish (Future)

8. **TODO**: Add diagrams for hierarchical DAG execution
9. **TODO**: Add more container examples
10. **TODO**: Cross-reference between docs (internal links)

---

## Feature Coverage Matrix

### Release Announcement Features vs Documentation

| Feature Category | Announcement | Spec | Features | Syntax | Examples |
|-----------------|--------------|------|----------|--------|----------|
| **v0.32.0 New** |
| Hierarchical DAG | ✅ | ❌ | ❌ | ❌ | ❌ |
| Code Cleanup | ✅ | N/A | N/A | N/A | N/A |
| Library Docs | ✅ | ✅ | ✅ | N/A | ✅ |
| Docs Organization | ✅ | ✅ | ✅ | N/A | ✅ |
| Expression NOT Fix | ✅ | ❌ | ✅ | ✅ | ❓ |
| Enhanced Matrix | ✅ | ❌ | ✅ | ✅ | ❓ |
| **Core Features** |
| Matrix Builds | ✅ | ✅ | ✅ | ✅ | ✅ |
| Multi-Dim Matrix | ✅ | ✅ | ✅ | ✅ | ✅ |
| Matrix on Stages | ✅ | ✅ | ✅ | ✅ | ✅ |
| Container Support | ✅ | ❌ | ❌ | ❌ | ❓ |
| Pool Parallelism | ✅ | ❓ | ❓ | ❓ | ❌ |
| File Caching | ✅ | ✅ | ✅ | ✅ | ✅ |
| Artifact Collection | ✅ | ❓ | ❓ | ❓ | ❓ |
| Expression Language | ✅ | ✅ | ✅ | ✅ | ✅ |
| Platform Detection | ✅ | ✅ | ✅ | ✅ | ✅ |
| Include System | ✅ | ✅ | ✅ | ✅ | ✅ |
| Action Variants | ✅ | ✅ | ✅ | ✅ | ✅ |
| Built-in Actions | ✅ | ✅ | ✅ | ✅ | ✅ |
| DAG Execution | ✅ | ✅ | ✅ | ✅ | ✅ |
| Error Policies | ✅ | ✅ | ✅ | ✅ | ✅ |

**Legend**:
- ✅ Fully documented
- ❓ Partially documented
- ❌ Not documented
- N/A Not applicable (internal/meta)

---

## Detailed Findings

### 🔴 Critical Documentation Gaps

#### 1. Hierarchical DAG Architecture (v0.32.0)

**Why Important**: Major architectural change, affects how users understand execution

**What's Missing**:
- Explanation of job-based execution model
- How matrix builds create jobs
- Why nested matrices now work correctly
- Execution wave concept
- Job vs step distinction

**Where to Add**: 
- `docs/User/Project-specification.md` - Section 2.5 "Execution Model"
- `docs/User/Features-and-examples.md` - Section "Hierarchical DAG Execution"

**Priority**: HIGH (major v0.32.0 feature)

---

#### 2. Container Support Details

**Why Important**: Key differentiator vs other tools

**What's Missing**:
- Comprehensive container configuration reference
- Docker vs Podman engine details
- Volume mounting syntax and patterns
- Environment variable passing
- Resource limits configuration
- Image pull policies
- Network modes
- Real-world container examples

**Where to Add**:
- `docs/User/Features-and-examples.md` - New section "Container Support"
- `docs/User/YAML-syntax-reference.md` - Add "container:" syntax reference

**Priority**: HIGH (frequently used feature)

---

### 🟡 Features Needing Enhancement

#### 3. Pool-Based Parallelism Explanation

**Currently**: `max_parallel` setting is documented  
**Missing**: How pools actually work

**What to Add**:
- Concept of execution pools
- Global pool vs matrix-specific pools
- How `min(global, matrix)` strategy works
- Worker queue and semaphore explanation

**Where**: `docs/User/Project-specification.md` - Section "Concurrency Control"

**Priority**: MEDIUM (helps understand parallelism)

---

#### 4. Advanced Matrix Features (v0.32.0 Enhancements)

**Currently**: Basic matrix documented  
**Missing**: v0.32.0 improvements

**What to Add**:
- Sliding window dependencies concept
- Condition-based skip behavior with dependencies
- User dependency inheritance in matrix jobs
- Nested matrix variable propagation

**Where**: `docs/User/Features-and-examples.md` - New section "Advanced Matrix Features"

**Priority**: MEDIUM (improves understanding of complex scenarios)

---

#### 5. Artifact Collection

**Uncertain**: Not sure if documented

**Need to Check**:
- Container artifact collection
- Artifact patterns and wildcards
- Artifact storage and retrieval

**Action**: Search for artifact documentation

---

### 🟢 Well Documented Features

These features are comprehensively documented:

1. ✅ **Matrix Builds**: Multiple sections with examples
2. ✅ **Multi-Dimensional Matrices**: Dedicated section in Features-and-examples.md
3. ✅ **Matrix on Stages**: Explained with examples
4. ✅ **Expression Language**: Syntax, operators, functions documented
5. ✅ **Variable Interpolation**: Complete reference with examples
6. ✅ **Built-in Actions**: All actions documented with usage
7. ✅ **Include System**: Behavior and syntax documented
8. ✅ **Action Variants**: Platform-specific execution documented
9. ✅ **Conditional Execution**: If conditions and expressions
10. ✅ **DAG Execution**: Basic dependency and parallel execution
11. ✅ **Error Policies**: stop/warn documented
12. ✅ **Platform Detection**: Variables and usage documented
13. ✅ **Library API**: Complete docs/Library.md with examples

---

## Recommended Updates

### Immediate (Before Release)

#### Update 1: Add Hierarchical DAG Section

**File**: `docs/User/Project-specification.md`  
**Section**: After "Core Features", add "Execution Model"

```markdown
### Execution Model (v0.32.0)

buildfab uses a **hierarchical DAG architecture** for executing workflows:

#### Job-Based Execution

- **Jobs**: Matrix combinations and non-matrix steps form individual job nodes
- **Steps**: Each job contains an ordered sequence of steps
- **Parallelism**: Jobs execute in parallel waves, steps sequentially within jobs
- **Dependencies**: Jobs wait for dependent jobs, not individual steps

#### Execution Waves

Jobs execute in dependency-ordered waves:
1. All jobs with no dependencies start first (Wave 1)
2. Once Wave 1 completes, jobs depending only on Wave 1 start (Wave 2)
3. Process continues until all jobs complete

#### Matrix Build Execution

When a step has a matrix:
- Each matrix combination creates one job
- Each job runs the same sequence of steps
- Jobs execute in parallel (respecting max_parallel)
- Sliding window dependencies control concurrency

Example:
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
```

#### Nested Matrix Support

Matrix on stage references expands hierarchically:
```yaml
stages:
  build:
    steps:
      - action: configure
      - action: compile

  cross-compiler:
    steps:
      - stage: build
        matrix:
          values:
            compiler: [gcc, clang]
# Creates 2 jobs (gcc, clang)
# Each job runs: configure → compile (sequential)
# Jobs run in parallel
```
```

---

#### Update 2: Add Container Support Section

**File**: `docs/User/Features-and-examples.md`  
**Location**: After "Matrix Feature" section

```markdown
## Container Support

buildfab provides native integration with Docker and Podman for containerized builds.

### Basic Container Action

```yaml
actions:
  - name: test-in-container
    container:
      image:
        from: golang:1.22
        pull: missing
      run_action: test
      volumes:
        - $PWD:/workspace
      environment:
        - GO111MODULE=on
```

### Container Configuration Reference

#### Image Configuration

```yaml
container:
  image:
    from: "image:tag"        # Container image to use
    pull: missing            # Pull policy: always, missing, never
```

**Pull Policies**:
- `always`: Always pull image before running
- `missing`: Pull only if image not found locally
- `never`: Never pull, use local image only

#### Engine Selection

```yaml
container:
  engine: podman             # Explicitly use podman (default: auto-detect)
```

buildfab automatically detects:
1. Docker (if available)
2. Podman (if Docker not found)

#### Resource Limits

```yaml
container:
  cpu: 2                     # CPU cores limit
  memory: "4G"               # Memory limit
  network: host              # Network mode: host, bridge, none
```

#### Volume Mounting

```yaml
container:
  volumes:
    - $PWD:/workspace                    # Mount current directory
    - $HOME/.cache:/cache                # Mount cache directory
    - ./config:/app/config:ro            # Read-only mount
```

Volume syntax: `<host-path>:<container-path>[:ro]`

#### Environment Variables

```yaml
container:
  environment:
    - GO111MODULE=on                     # Set specific value
    - GOPATH                             # Pass from host
    - GOCACHE=/workspace/.cache          # Set container-specific
```

### Advanced Container Examples

#### Multi-Platform Testing

```yaml
actions:
  - name: test-distro
    container:
      image:
        from: ${{ matrix.distro }}
        pull: missing
      run_action: test
      volumes:
        - $PWD:/workspace
      workdir: /workspace

stages:
  test-all:
    steps:
      - action: test-distro
        matrix:
          values:
            distro: [ubuntu:22.04, alpine:latest, centos:8]
```

#### Container with Caching

```yaml
actions:
  - name: build-with-cache
    container:
      image:
        from: golang:1.22
      run_action: build
      volumes:
        - $PWD:/workspace
        - $HOME/.cache/go-build:/root/.cache/go-build
        - $HOME/.cache/go-mod:/go/pkg/mod
      environment:
        - GOCACHE=/root/.cache/go-build
        - GOMODCACHE=/go/pkg/mod
```

### Container Requirements

- Buildfab binary must be in PATH or standard directories
- Required for `run_action` and `run_stage` inside containers
- Auto-discovered from: `/usr/local/bin`, `/usr/bin`, `$HOME/bin`

### Troubleshooting

**Container not found**:
- Ensure Docker or Podman is installed
- Check engine availability: `docker info` or `podman info`

**Buildfab binary not found**:
- Install buildfab in standard directory
- Or set explicit path in configuration

**Permission denied**:
- Add user to docker group: `sudo usermod -aG docker $USER`
- Or use Podman (rootless by default)
```

---

#### Update 3: Add v0.32.0 Expression Improvements Note

**File**: `docs/User/Features-and-examples.md`  
**Location**: In "Conditional Execution" section

Add note:
```markdown
### Complex NOT Expressions (v0.32.0)

The NOT operator now works correctly with complex nested conditions:

```yaml
steps:
  - action: build
    if: "!(matrix.images == 'centos7' && matrix.compiler == 'clang')"
# Skips only the specific combination, not all centos7 or all clang
```

Multiple conditions with NOT:
```yaml
steps:
  - action: deploy
    if: "!(env.CI == 'true') && version.type == 'release'"
# Deploys only on non-CI release builds
```
```

---

### Future Enhancements (v0.33.0+)

4. Add execution flow diagrams
5. Add troubleshooting guide
6. Add performance tuning guide
7. Add migration guide from Make/Taskfile

---

## Summary

### Documentation Coverage

**Overall**: ~75% of features well documented

**Gaps**:
- Hierarchical DAG architecture (v0.32.0 major feature)
- Container support (detailed configuration)
- Pool-based parallelism (concept explanation)
- v0.32.0 enhancements (NOT operator, matrix improvements)

**Next Steps**:
1. Add hierarchical DAG section (HIGH priority)
2. Add container support section (HIGH priority)
3. Add advanced matrix section (MEDIUM priority)
4. Enhance YAML reference with container syntax (MEDIUM priority)

**Estimated Effort**: 2-3 hours of documentation writing

---

## Conclusion

The buildfab documentation is **generally comprehensive** for core features, but needs updates for:

1. **v0.32.0 specific features** (hierarchical DAG, improvements)
2. **Container support details** (configuration reference)
3. **Advanced matrix scenarios** (nested, sliding window)

These additions will make the documentation complete and current with the v0.32.0 release.

**Recommendation**: Add the three high-priority sections before tagging release.

