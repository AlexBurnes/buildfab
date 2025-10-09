# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [Unreleased]

## [0.23.1] - 2025-10-09

### Fixed
- **Container Command Display**: Fixed container build/slim/run command display at verbose level >= 2
  - Container commands now show proper prefix based on operation type:
    - "Building image" for build operations
    - "Slimming image" for slim operations
    - "Running container" for regular run operations
  - Variables like `${{version}}` and `${{project}}` are now properly interpolated before display
  - Fixed issue where build and slim commands were not showing at all at `-vv` level
  - Applies to both direct action execution and stage execution
  - Files modified: `pkg/buildfab/ordered_output.go`, `pkg/buildfab/simple.go`

### Technical Details
- **SimpleStepCallback Enhancement**: Added `showContainerCommand` and `buildContainerCommand` methods
  - Displays container commands at verbose level >= 2
  - Supports all container operation types (build, slim, run)
  - Includes variable interpolation for display
- **OrderedOutputManager Enhancement**: Added variable interpolation for container command display
  - Added `variables` field and `SetVariables` method
  - Interpolates variables in container configuration before building display command
- **Variable Interpolation**: Uses existing `InterpolateContainerConfig` function to interpolate variables
  - Properly handles project variables like `${{version}}`, `${{project}}`
  - Correctly expands matrix variables in matrix jobs

## [0.23.0] - 2025-10-08

### Added
- **New Version Variable**: Added `version.rawversion` variable using `GetRawVersion()` method from version-go library v1.5.0
  - Returns raw version string exactly as detected (e.g., "v0.22.0-1-g1234abc")
  - Provides full git describe output including commits since tag and commit hash
  - Useful for development builds and detailed version tracking
  - Accessible in all contexts (stages, actions, matrix, containers)
  - Added to documentation in `docs/YAML-syntax-reference.md` and `docs/Project-specification.md`

### Changed
- **Version Library**: Updated version-go from v1.4.2 to v1.5.0
  - New `GetRawVersion()` method for raw version string access
  - Enhanced version detection with full git describe support
  - Latest version parsing improvements

### Technical Details
- **File Modified**: `go.mod` - Upgraded version-go to v1.5.0
- **File Modified**: `internal/version/version.go` - Added `version.rawversion` variable using `GetRawVersion()` method
- **File Modified**: `internal/version/version_test.go` - Added `version.rawversion` to expected keys list
- **File Modified**: `docs/YAML-syntax-reference.md` - Added `version.rawversion` to version variables documentation
- **File Modified**: `docs/Project-specification.md` - Added `version.rawversion` to variable interpolation documentation

### Testing
- **All 13 Version Variables Verified**: version, version.version, version.rawversion, version.tag, version.project, version.major, version.minor, version.patch, version.type, version.build-type, version.version-type, version.branch, version.commit
- **Verified Raw Version Access**: Tested that `version.rawversion` returns full git describe output
- **All Tests Passing**: `go test ./internal/version/... -v` passes with new variable

## [0.22.0] - 2025-10-08

### Added
- **New Version Variable**: Added `version.tag` variable using `GetRawTag()` method from version-go library
  - Returns raw git tag WITH 'v' prefix (e.g., "v0.22.0")
  - Useful for git operations requiring full tag format
  - Accessible in all contexts (stages, actions, matrix, containers)

### Changed
- **Version Detection**: Fixed version variable detection to use correct `GetVersion()` API from version-go library
  - `version` and `version.version` now return version WITHOUT 'v' prefix (e.g., "0.22.0")
  - Properly formatted for Docker tags (e.g., `myapp:0.22.0`)
  - Modified `getProjectVersion()` in `cmd/buildfab/main.go` to use `version.GetVersion()`
  - Modified `DetectCurrentVersion()` in `internal/version/version.go` to use `version.GetVersion()`
  - Both functions now return consistent values without 'v' prefix

- **Version Library**: Updated version-go from v1.4.0 to v1.4.2
  - New `GetRawTag()` method for raw git tag access
  - Latest version detection improvements
  - Enhanced version type classification

### Fixed
- **Variable Consistency**: Both `version` and `version.version` now return the same value
  - Previously returned different values due to inconsistent detection methods
  - Now both use `GetVersion()` from version-go library
  - Ensures consistent behavior across all use cases

### Technical Details
- **File Modified**: `cmd/buildfab/main.go` - Added version-go import, updated `getProjectVersion()` to use `GetVersion()`
- **File Modified**: `internal/version/version.go` - Updated `DetectCurrentVersion()` to use `GetVersion()`, added `version.tag` variable using `GetRawTag()`
- **File Modified**: `internal/version/version_test.go` - Updated test expectations to match new behavior (version without 'v' prefix)
- **File Modified**: `go.mod` - Upgraded version-go to v1.4.2
- **File Modified**: `go.sum` - Updated checksums for version-go v1.4.2

### Testing
- **All 12 Version Variables Verified**: version, version.version, version.tag, version.project, version.major, version.minor, version.patch, version.type, version.build-type, version.version-type, version.branch, version.commit
- **Complete Context Testing**: Verified all variables work correctly in:
  - Regular stage execution
  - Direct action execution
  - Matrix-expanded steps
  - Container execution
  - Container + matrix combined execution
- **Unit Tests**: All version tests pass with updated expectations

## [0.21.1] - 2025-10-08

### Fixed
- **Race Condition in DAG Executor**: Fixed concurrent map access race conditions in `executeDAGWithOrderedStreaming`
  - Added mutex synchronization to all shared state map accesses (`displayed`, `completed`, `started`)
  - Created thread-safe helper functions: `checkAndDisplayNextStep`, `displayStepInOrder`, `displayStepInOrderLocked`, `canDisplayStepInOrderLocked`, `displayRemainingSteps`
  - Implemented proper unlock/lock patterns around callbacks to prevent deadlocks
  - All tests now pass with race detector enabled (`go test ./... -race`)
  
- **Pool Context Cancellation Hang**: Fixed `ExecutionPool` hanging on context cancellation
  - Added `drainQueuedTasks()` method to handle tasks still in queue when context is cancelled
  - Implemented `sync.Once` pattern to ensure queue draining happens exactly once even with multiple workers
  - Workers now properly drain queued tasks and call `OnComplete` with cancellation error
  - Fixed WaitGroup balance by decrementing for drained tasks
  - Test `TestExecutionPool_ContextCancellation` now completes successfully without hanging

### Technical Details
- **File Modified**: `pkg/buildfab/buildfab.go` - Added mutex parameters to display functions and proper locking
- **File Modified**: `pkg/buildfab/pool.go` - Added `drainOnce sync.Once` field and `drainQueuedTasks()` method
- **Testing**: All tests pass with race detector, including previously hanging `TestExecutionPool_ContextCancellation`

## [0.21.0] - 2025-10-08

### Added
- **Multiline Output Feature**: Enhanced quiet mode with real-time job status display
  - Real-time status updates showing all jobs simultaneously (pending → running → success/error)
  - ANSI escape codes for cursor control and dynamic display updates
  - Consistent job display order matching declaration order
  - Parallel execution support with max_parallel constraints
  - Proper error handling with red ✗ indicators for failed jobs
  - Clean terminal state management and cursor positioning
  - Event-driven updates via existing DAG executor callback system
  - Thread-safe concurrent updates with mutex protection
  - TTY detection for terminal-only activation with graceful fallback for non-TTY environments

### Changed
- **Quiet Mode UX**: Enhanced user experience by showing all jobs simultaneously instead of single-line running indicator
- **Output Manager**: Added MultilineOutputManager for quiet mode with OrderedOutputManager fallback for verbose mode
- **Step Callback**: Extended StepCallback interface with GetResults() method for result collection
- **Terminal Compatibility**: Improved cursor management and terminal state handling for better user experience

### Technical Details
- **New Components**: MultilineOutputManager, MultilineStepCallback integrating with existing DAG executor
- **Backward Compatibility**: Verified verbose mode continues using OrderedOutputManager with zero breaking changes
- **Testing**: Validated error scenarios, cursor positioning, and Ctrl+C termination handling
- **Production Ready**: Feature enhances quiet mode UX while maintaining familiar verbose mode behavior

### Documentation
- **Feature Specification**: Created comprehensive PRD in `docs/Multiline-output-feature-specification.md`
- **Implementation Details**: Documented technical architecture, ANSI escape codes, and integration points
- **Testing Strategy**: Added test files for matrix streaming and error handling scenarios

### Documentation
- **Multiline Output Feature Specification**
  - **Created comprehensive PRD** - `docs/Multiline-output-feature-specification.md` with detailed technical specification for enhancing quiet mode user experience
  - **Current Implementation Analysis** - Analyzed existing OrderedOutputManager and OrderedStepCallback with single-line cyan circle running indicator (`◯ step-name (running...)`)
  - **Feature Requirements** - Identified limitations: users can only see one job at a time, no overall progress visibility, limited status information during execution
  - **Proposed Solution** - Multiline display showing all jobs simultaneously with real-time status updates using ANSI escape codes for cursor control
  - **Technical Implementation** - MultilineOutputManager with ANSI escape codes (`\x1b[?25l`, `\x1b[2K`, `\x1b[n;mH`) for cursor control, job status tracking (pending, running, success, warning, error, skipped)
  - **Integration Points** - Replace quiet mode logic in OrderedOutputManager.showStepStart(), integrate with existing DAG executor and step callback system
  - **User Experience** - All jobs visible simultaneously with status icons (○ pending, ◯ running, ✓ success, ✗ error, → skipped), real-time status updates, execution order preservation
  - **Implementation Plan** - 4 phases: Core Infrastructure (2-3 days), Integration (2-3 days), Testing and Refinement (1-2 days), Documentation and Release (1 day)
  - **Testing Strategy** - Unit tests, integration tests, manual testing with real buildfab stages, edge case handling
  - **Success Criteria** - All jobs visible, real-time updates, proper ordering, status accuracy, minimal overhead, responsive updates
  - **Risk Mitigation** - TTY detection and fallback mode, terminal resizing handling, performance profiling, accessibility considerations

## [0.20.1] - 2025-10-08

### Documentation
- **Documentation Translation and Reorganization**
  - **Translated Russian documents to English**:
    - `docs/Practical_applications.md` → `docs/Practical-applications.md` - comprehensive production usage guide
    - `docs/slim_support_added.md` → `docs/Slim-support-added.md` - slim image feature documentation
  - **Renamed files according to naming conventions** (First-word-second-word.md format):
    - `Analysis_summary.md` → `Analysis-summary.md`
    - `CLI_DRY_analysis.md` → `CLI-dry-analysis.md`
    - `Documentation_complete.md` → `Documentation-complete.md`
    - `Documentation_files.md` → `Documentation-files.md`
    - `Git_integration_info.md` → `Git-integration-info.md`
    - `Signal_handling_fix_summary.md` → `Signal-handling-fix-summary.md`
  - **Project Specification Updates**:
    - Replaced `docs/Project-specification.md` with more comprehensive root version (409 lines vs 175 lines)
    - Removed duplicate `buildfab-project-specification.md` from root directory
  - **README.md Enhancements**:
    - Added comprehensive **Project Vision** section explaining buildfab's origins and frustrations it solves
    - Added **Philosophy** section describing build orchestration approach
    - Added **Architecture Overview** with high-level and detailed diagrams
    - Added **Team** section introducing Alex Burnes, Buzzy (AI Assistant), and Coddy (AI Agent)
    - Fixed **Installation** section structure - removed duplicate information
    - Clarified **Git Hook Setup** - buildfab runs pre-push stages but doesn't install hooks
    - Added link to [pre-push utility](https://github.com/AlexBurnes/pre-push) for Git hook installation
    - Updated **Version Utility** description - semantic version validation and platform/git detection variables
    - Updated **macOS Installation** to use Homebrew instead of curl script
    - Updated **Linux Installation** to use platform-specific install scripts
  - **Documentation Organization**:
    - All documentation now follows consistent naming conventions
    - All documents properly located in `docs/` directory (except memory bank files in root)
    - No broken links found - all references already correct or updated

### Fixed
- **Comparison Document Corrections**: Fixed multiple inaccuracies in Comparison-with-others.md based on real implementation
  - **Container Support**: Corrected to show real `container:` block syntax instead of incorrect `uses: docker@build`
  - **Caching**: Clarified that caching is not a built-in feature, but recommended via container bind mounts (ccache, Conan, vcpkg)
  - **Artifacts**: Corrected to indicate artifacts are container-only feature with full path preservation
  - **Real Examples**: Updated all examples to match actual YAML syntax from examples/ and tests/
  - **Accurate Capabilities**: Container support includes `image.from`, `image.build`, `image.slim`, mounts, env, resources, artifacts
  - **Performance Claims**: Verified all performance metrics against actual implementation
  - **Git Integration**: Enhanced Git integration section with comprehensive pre-push utility documentation
    - Added architecture diagram showing Git → pre-push → buildfab library flow
    - Documented shared `.project.yml` configuration between buildfab and pre-push
    - Clarified pre-push utility as separate project that embeds buildfab as library
    - Added installation instructions for pre-push utility: `pre-push install`
    - Explained automatic execution (Git hook) vs manual execution (buildfab CLI)
    - Added link to pre-push project: https://github.com/AlexBurnes/pre-push
  - **Slim Image Support**: Enhanced container support section with comprehensive slim image documentation
    - Added `image.slim` capability for image size optimization using dslim/slim tool
    - Documented 30x+ image size reduction capabilities
    - Added example workflow: build → slim → artifacts
    - Included slim configuration options: target, tags, network, http_probe, exec
    - Explained slim image benefits: removes unnecessary files, creates minimal production images
    - Added complete 3-step example showing image.build, image.slim, and artifact collection

### Documentation
- **Core Purpose Documentation**: Enhanced all documentation with buildfab's core mission and value proposition
  - **README.md**: Added "Why buildfab?" section explaining build fragmentation problem
    - Documented the problem: scattered bash scripts, different CI YAML, custom Dockerfiles
    - Explained the solution: single `.project.yml` file working everywhere (locally, CI, containers, hooks)
    - Added "What is buildfab?" section with clear positioning as universal task orchestrator
  - **Comparison-with-others.md**: Enhanced Executive Summary with problem/solution/result structure
    - Added "The Problem: Build Fragmentation" section explaining scattered scripts issue
    - Added "The Solution: Single Source of Truth" section with unified YAML approach
    - Added "The Result" section highlighting benefits: single source of truth, unified execution, reproducible builds
  - **Release-announcement.md**: Updated introduction with problem/solution framework
    - Added clear problem statement about build fragmentation
    - Documented solution with single `.project.yml` working everywhere
  - **productContext.md**: Added "Core Purpose" section to memory bank
    - Documented key concept: one configuration file for all environments
    - Explained single source of truth approach for build stages, dependencies, variants, matrices
- **Practical Applications**: Added comprehensive real-world usage documentation across all main documents
  - **README.md**: Added "About" section with self-hosting and real-world usage
    - Self-hosting: buildfab builds itself using its own .project.yml
    - Go projects: buildfab building itself locally and in GitHub Actions
    - C++ modules: complex projects on GitLab CI with multi-distro support
    - Container workflows: building applications and creating slim images (30x+ reduction)
  - **Comparison-with-others.md**: Added "Practical Applications" section with detailed examples
    - Self-hosting example with .project.yml configuration
    - Go projects: cross-platform compilation with multi-platform matrix builds
    - C++ modules: real production usage with GitLab CI, CMake/Conan, ccache integration
    - Container workflows: build → slim → artifacts pipeline with concrete YAML examples
  - **Release-announcement.md**: Added "Practical Applications" section
    - Self-hosting proof of concept
    - Go projects with GoReleaser workflow
    - C++ modules with multi-distro support
    - Container workflows with slim image optimization
- **Comprehensive Comparison Document**: Created detailed comparison with alternative tools (Taskfile, GitHub Actions, Earthly, Make, Just)
  - **Comparison-with-others.md**: 850+ line comprehensive analysis document
    - Feature-by-feature comparison table with 25+ criteria
    - Detailed comparisons of matrix builds, containers, artifacts, parallelism
    - Performance benchmarks (startup time, overhead, memory usage)
    - Use case recommendations with migration guides
    - Scoring summary across 10 evaluation criteria
    - Accurate examples from real project configuration
  - **Overall Assessment**: buildfab scores 85/100, highest among local automation tools
- **Release Announcement**: Created comprehensive release announcement document
  - **Release-announcement.md**: 500+ line feature overview and technical highlights
    - Key features and capabilities overview
    - Real-world usage examples
    - Performance metrics and technical highlights
    - Comparison summary with alternatives
    - Getting started guide and roadmap
- **README.md Updates**: Added links to new documentation
  - Added Comparison with Others link in documentation section
  - Added Release Announcement link in documentation section
  - Improved documentation navigation and discovery

## [0.20.0] - 2025-10-07

### Added - Sprint 3 Documentation & Release (2025-10-07)
- **Comprehensive Documentation Updates**: Updated all user documentation with parallel pool feature details
  - **Matrix-feature.md**: Added parallel pool execution section with detailed explanations
    - How matrix parallelism works with pool-based execution
    - Global concurrency control with `project.max_parallel`
    - Matrix-specific pools with dedicated `max_parallel` settings
    - Interaction between limits using min() strategy
    - Performance metrics showing ~0.75μs overhead (1000x better than requirement)
    - Troubleshooting guide with debug output examples
    - Pool statistics for advanced monitoring
  - **Features-and-examples.md**: Added pool configuration examples
    - Global and matrix parallelism control section
    - Min() strategy examples showing limit interactions
    - Three practical scenarios: global restricts matrix, matrix self-limits, no matrix limit
  - **YAML-syntax-reference.md**: Added `project.max_parallel` field documentation
    - Complete field reference with behavior explanation
    - Validation rules (must be >= 0)
    - Interaction with matrix pools
    - Three detailed configuration examples
  - **README.md**: Updated with parallel pool feature
    - Added pool-based concurrency control to feature list
    - Updated Quick Start with `max_parallel` example
    - Enhanced execution control documentation
    - Added note on parallelism CLI flag behavior
- **Release Preparation**: Final testing and version bump
  - All tests pass with `-race` flag
  - Cross-platform compatibility verified
  - Version bumped to v0.20.0
  - Release notes prepared with feature highlights
- **Feature Highlights**:
  - Fixed critical bug where matrix `max_parallel` was not enforced
  - Implemented pool-based execution system with exceptional performance
  - 26 comprehensive tests (20 unit + 6 integration) with 100% pass rate
  - Performance: 1.3M tasks/sec, ~0.75μs overhead, < 10MB for 1000 tasks
  - Thread-safe operations with no goroutine leaks

### Documentation
- **Matrix-feature.md**: Added 150+ lines of parallel pool documentation
- **Features-and-examples.md**: Added 90+ lines of pool configuration examples
- **YAML-syntax-reference.md**: Added 60+ lines of `max_parallel` field documentation
- **README.md**: Updated feature list and CLI usage with parallelism notes

### Added - Sprint 2 Testing & Benchmarks (2025-10-07)
- **Comprehensive Unit Tests**: Added 20 unit tests for pool system with 100% coverage of core functionality
  - ExecutionPool tests: basic execution, max parallel limit, context cancellation, task errors, callbacks, concurrent submit
  - PoolManager tests: global pool, matrix pools, idempotent creation, pool retrieval, stop/cancel operations
  - Min() strategy tests: validation of effective parallelism calculation with various configurations
  - WaitGroup balance verification: ensures proper cleanup without deadlocks
  - Statistics tracking validation: verifies accurate task counting and status reporting
- **Integration Tests**: Added 6 end-to-end integration tests with timing validation
  - Matrix max_parallel=2: verified 4 jobs take ~4s (2 waves of 2 jobs)
  - Sequential execution (max_parallel=1): verified 3 jobs take ~6s (sequential)
  - Global pool usage: verified 4 jobs run in parallel (~2s with global limit=8)
  - Mixed workload: regular steps + matrix steps with combined pool usage
  - Global limit restriction: verified min() strategy enforces global as hard limit
  - Dependencies with pools: verified dependency resolution works across pool boundaries
- **Performance Benchmarks**: Added 7 benchmarks verifying excellent pool performance
  - Submit overhead: ~0.75μs (1000x better than 1ms requirement)
  - Throughput: 1.3 million tasks/second with CPU-core workers
  - GetPool: ~57ns lookup time
  - GetOrCreateMatrixPool: ~87ns creation/retrieval time
  - Concurrent submit: 151M tasks/sec from multiple goroutines
  - Small tasks: 469K tasks/sec with minimal work
- **Files Added**:
  - `pkg/buildfab/pool_unit_test.go` - 20 comprehensive unit tests (602 lines)
  - `pkg/buildfab/pool_integration_test.go` - 6 integration tests with timing validation (399 lines)
  - `pkg/buildfab/pool_bench_test.go` - 7 performance benchmarks (212 lines)
- **Test Results**:
  - All 20 unit tests pass in < 1s
  - All 6 integration tests pass with perfect timing
  - All benchmarks confirm < 1ms overhead (actually < 0.001ms)

### Fixed - Sprint 1 Critical Fixes (2025-10-07)
- **WaitGroup Management**: Fixed potential deadlock in ExecutionPool by moving `Add(1)` to Submit() before queueing
  - Previously called Add(1) in executeTask() after dequeue, causing imbalance on cancellation
  - Now increments WaitGroup before queueing task, decrements on context cancellation if task doesn't execute
  - Prevents WaitGroup panic and ensures proper cleanup during pool shutdown
- **Debug Output Consistency**: Fixed debug messages to show correct pool name (matrix vs global)
  - Previously all debug messages showed "global pool" even for matrix-specific pools
  - Now correctly identifies pool name from stepConfig.PoolID
  - Debug output format: `[DEBUG] Pool: Starting step <name> in <pool-name> pool`
- **Config Validation**: Verified Project.MaxParallel validation already implemented (>= 0 check)
  - Validation ensures max_parallel is non-negative (0 means use CPU cores)
  - Error message: "project.max_parallel must be >= 0 (0 means use CPU cores)"

### Added - Parallel Pool Feature (2025-10-07)
- **Parallel Pool Execution System**: Implemented comprehensive pool-based execution system to enforce matrix `max_parallel` limits
  - **ExecutionPool Infrastructure**: Worker pool with task queue, context-aware cancellation, and statistics tracking
  - **PoolManager**: Coordinates global and matrix-specific pools with proper lifecycle management
  - **Project.MaxParallel Configuration**: Global concurrency limit in project YAML (default: CPU cores)
  - **Matrix Pool Assignment**: Matrix steps with `max_parallel` get dedicated pools via PoolID field
  - **Min() Strategy**: Effective parallelism = min(global_max, matrix_max) properly enforced
  - **DAG Integration**: Pool-based task submission in executeDAGWithCallback() replaces unlimited goroutines
  - **Context Propagation**: Pools derive context from parent for proper Ctrl+C signal handling
  - **Test Coverage**: Created test-parallel-pool-matrix.yml with 3 comprehensive test scenarios
- **Files Added**:
  - `pkg/buildfab/pool.go` - ExecutionPool and PoolManager implementation (296 lines)
  - `tests/test-parallel-pool-matrix.yml` - Test scenarios for matrix limits and global pool usage
  - `docs/Parallel-pool-next-steps.md` - Refinement plan and implementation roadmap
- **Files Modified**:
  - `pkg/buildfab/buildfab.go` - expandMatrixStepsWithPools(), createTaskForStep(), pool-based DAG execution
  - `pkg/buildfab/simple.go` - Min() strategy implementation for effective parallelism calculation
  - `internal/config/config.go` - Added Project.MaxParallel field
- **Documentation Updated**:
  - `docs/Matrix-parallel-pools-implementation.md` - Updated with implementation status and phase completion
  - `activeContext.md` - Documented parallel pool feature completion
  - `progress.md` - Updated with parallel pool refinement phase details

### Fixed - Matrix Max Parallel Enforcement (2025-10-07)
- **Critical Bug**: Fixed matrix `max_parallel` not being enforced - all matrix jobs were running in unlimited parallel
- **Root Cause**: DAG executor was spawning unlimited goroutines for all ready nodes, completely bypassing max_parallel setting
- **Solution**: Implemented pool-based execution where steps are submitted to ExecutionPool instead of spawning goroutines
- **Impact**: Matrix builds now properly respect max_parallel limits, preventing resource exhaustion
- **Validation**: Global limit acts as hard upper bound, matrix-specific limits cannot exceed global limit

### Fixed
- **Critical: Signal Handling for DAG Executor and Container Execution** - Fixed issue where Ctrl+C/INT signal did not terminate running jobs
  - **Pool Context Issue**: ExecutionPool was creating an isolated context from `context.Background()` instead of deriving from parent context
  - **Container Execution Issue**: Container engines were using `cmd.Wait()` without monitoring `ctx.Done()` for cancellation
  - **DAG Executor Wait Issue**: DAG executors were blocking on `<-done` without monitoring `ctx.Done()`, causing hang even after context cancellation
  - **Solution Part 1**: Modified `NewExecutionPool` and `NewPoolManager` to accept and use parent context, preserving cancellation chain
  - **Solution Part 2**: Modified `RunContainerWithCallback` in both Docker and Podman engines to monitor context cancellation with three-step termination (SIGTERM → 100ms → SIGKILL)
  - **Solution Part 3**: Modified all three DAG executors to monitor `ctx.Done()` when waiting for completion, calling `poolManager.StopAll()` immediately on cancellation
  - **Impact**: All running jobs (regular and container) now properly terminate when Ctrl+C is pressed within ~150ms, no hanging processes
  - **Verified**: Automated test confirms process terminates properly after receiving INT signal
  - **Technical details**: See `docs/Signal-handling-fix.md` for comprehensive explanation
  - **Modified files**: `pkg/buildfab/pool.go`, `pkg/buildfab/buildfab.go`, `pkg/buildfab/container/engines.go`, `pkg/buildfab/pool_phase1_test.go`

## [0.19.0] - 2025-10-07

### Added
- **Container Artifact Collection**: Complete implementation of hybrid artifact collection for container feature
  - **Hybrid Approach**: Different strategies for run commands vs build-only images
    - Run commands: Pre-mounted volume approach with automatic copy inside container
    - Build-only images: Docker/Podman cp approach with temporary container creation
  - **Full Path Preservation**: All artifacts preserve complete directory structure
    - `/app/binary` → `./dist/app/binary`
    - `/usr/local/bin/myapp` → `./dist/usr/local/bin/myapp`
    - `/build/output/` → `./dist/build/output/` (directories)
  - **Pre-mounted Volume for Run Commands**: Host output directory mounted as `/buildfab-artifacts`
  - **Automatic Copy Commands**: Artifact copy commands automatically added to run scripts
  - **Docker CP for Built Images**: Temporary container created from built image for artifact extraction
  - **Unique Container Naming**: Timestamp + random component prevents parallel execution conflicts
  - **Directory Support**: Complete directory structures copied with nested files
  - **Cross-Platform**: Works with both Docker and Podman engines
  - **No Overhead for Run Commands**: Artifacts written directly to host via mount (no docker cp needed)

### Changed
- **Artifact Collection Functions**: Implemented `collectArtifacts()`, `collectArtifactsFromImage()`, `copyArtifactFromContainer()`
- **Container Configuration Preparation**: Added `addArtifactMount()` and `addArtifactCopyCommands()` helpers
- **Path Handling**: Full path structure preservation using `strings.TrimPrefix()` and `filepath.Join()`

### Documentation
- **Container Artifact Collection Guide**: Created comprehensive `docs/Container-artifact-collection.md` with implementation details
- **Artifact Examples**: Created `examples/container-artifacts-example.yml` with 5 real-world usage examples
- **Test Suite**: Added comprehensive tests in `tests/test-container-artifacts.yml` and `tests/test-container-build-artifacts.yml`
- **Dockerfiles**: Created test Dockerfiles demonstrating artifact creation in images

## [0.18.8] - 2025-10-07

### Fixed
- **Container Output Ordering**: Fixed critical output ordering issue in matrix container builds where container commands and outputs were displayed out of order
  - Moved container command display from `runContainerAction()` to ordered output manager's `showStepStart()` function
  - Container commands now display in proper sequence: step start → container command → output → completion
  - Fixed output for matrix builds to show each container's output sequentially instead of all commands first then mixed outputs
  - Modified `showStepStart()` in `ordered_output.go` to call `showContainerCommand()` when verbosity >= 2
  - Removed direct container command printing from `runContainerAction()` in buildfab.go that bypassed ordered output manager

### Changed
- **Container Command Display**: Enhanced container command display for better accuracy and copy-paste usability
  - Added workspace mount detection and working directory `-w` option to display command
  - Changed from `alias buildfab=...` to `export PATH=...` in container commands for non-interactive shell compatibility
  - Added proper shell command quoting in display to make commands ready for copy-paste execution
  - Updated `buildContainerCommand()` to show accurate working directory when workspace is mounted

## [0.18.7] - 2025-10-07

### Added
- **Container Feature Architecture Enhancement**: Implemented comprehensive container feature enhancements with conditional `-w` option and optimized command execution
  - Added conditional `-w /tmp/buildfab-workspace` option that only applies when bind mount with `target=/tmp/buildfab-workspace` exists
  - Implemented `cd <workdir>` command integration before other run commands when `workdir` is defined
  - Centralized all command preparation in `PrepareContainerConfig` function for better maintainability
  - Optimized command strings by shortening `exec /tmp/buildfab-bin/buildfab` to `buildfab` due to alias being set
  - Corrected environment file path from `. .docker-env` to `. ./.docker-env` for proper sourcing

### Changed
- **Container Engine Simplification**: Simplified container engines (Docker and Podman) to focus on execution only
  - Removed complex command construction logic from engines
  - Engines now simply execute prepared commands from `PrepareContainerConfig`
  - Eliminated command construction duplication between engines and preparation logic
  - Improved architecture with single responsibility principle

### Fixed
- **Display Consistency**: Fixed `buildContainerCommandForDisplay` function to accurately reflect actual executed commands
  - Display string now matches real execution command exactly
  - Added conditional `-w` option logic to display function
  - Applied `cd workdir` logic to display function for consistency
  - Users now see exactly what command will be executed

## [0.18.6] - 2025-10-06

### Added
- **Enhanced Binary Search**: Added system path support for buildfab binary detection
  - Added `/usr/local/bin/buildfab` and `/usr/bin/buildfab` to search paths
  - Implemented current executable detection using `os.Executable()` with symlink resolution
  - Enhanced search priority: current executable → `./bin/buildfab` → `./buildfab` → system paths
  - Better error messages showing all searched locations for easier debugging
  - Perfect container support for both development and system-installed binaries

### Fixed
- **Container Error Handling**: Fixed critical bug in `PrepareContainerConfig` where errors were silently ignored
  - Modified `PrepareContainerConfig` to return `(container.ContainerConfig, error)` instead of silently falling back
  - Updated all callers to properly handle error return value
  - Users now get clear error messages when buildfab binary is not found or working directory cannot be accessed
  - Prevents confusing runtime failures by failing fast with descriptive error messages

## [0.18.5] - 2025-10-06

### Fixed
- **Container Pull Streaming**: Fixed container image pull progress to stream in real-time
  - Added `PullImageWithCallback` method to Engine interface for streaming pull operations
  - Implemented streaming pull in both Docker and Podman engines using `streamingWriter` helper
  - Updated container manager to use streaming pull methods when callbacks are enabled
  - Users now see complete pull progress: "Trying to pull...", "Copying blob" for each layer, etc.
  - Provides same visibility as manual `podman pull` or `docker pull` commands
  - Comprehensive testing verified fix works correctly with fresh image pulls

## [0.18.4] - 2025-10-06

### Fixed
- **Container Build-Only Execution**: Fixed critical bug where buildfab was automatically running Docker containers after building them, even when no run commands were specified in the YAML configuration
  - **Problem Identified** - Container actions with only `image.build` configuration were being executed with Dockerfile's default `CMD` instruction, causing unwanted container execution and errors
  - **Root Cause Found** - The `ExecuteAction` method in `container/manager.go` was always calling `RunContainer` after building images, regardless of whether run commands were specified
  - **Solution Implemented** - Added conditional check to only run containers when `run`, `run_action`, or `run_stage` commands are explicitly specified in the YAML configuration
  - **Fixed Both Execution Paths** - Updated both `ExecuteAction` and `ExecuteActionWithCallback` methods to check for run commands before attempting container execution
  - **Perfect User Experience** - Container build actions now only build/pull images without automatically running them, which is the expected behavior when no run commands are defined
  - **Comprehensive Testing** - Verified fix works correctly with container build examples showing clean build-only execution without unwanted container runs

### Changed
- **Container Execution Logic**: Container actions now only execute containers when run commands are explicitly specified
- **Build-Only Behavior**: Container build actions return success after building/pulling images without running containers

## [0.18.3] - 2025-10-06

### Fixed
- **CLI Help Flag Parsing**: Fixed "Error: unknown flag: --help" by adding missing `--help` and `-h` flag handling
  - Added proper flag handling in `handleRegularFlag` function for both short and long help flags
  - Users can now use `buildfab --help` and `buildfab -h` without errors
  - Help flag is properly recognized and handled by cobra's built-in help system

### Added
- **CLI Flags Integration Tests**: Comprehensive integration tests for all CLI flags
  - Added `TestIntegration_CLIFlags` test covering `-h`, `--help`, `--version`, and `-V` flags
  - Proper output validation for help content and version information
  - Ensures CLI flag functionality is maintained and catches future regressions
  - Tests both short and long flag forms with comprehensive validation

### Changed
- **CLI Flag Handling**: Enhanced flag parsing to properly handle help flags in custom parsing logic
- **Test Coverage**: Extended integration test suite with CLI flag validation

## [0.18.2] - 2025-10-06

### Fixed
- **Container Streaming Output**: Fixed container build and slim operations to stream output in real-time instead of buffering all output until completion
  - Added `BuildImageWithCallback` method to Engine interface with streaming implementation for both Docker and Podman engines
  - Added `SlimImageWithCallback` method with streaming implementation for both Docker and Podman engines
  - Updated `ExecuteActionWithCallback` to use streaming methods for both build and slim operations
  - Users now see complete container process output as it happens: Docker build steps, layer downloads, slim inspection phases, minification results, and security profile generation
  - Real-time feedback makes it easy to monitor build progress, debug issues, and understand optimization results

## [0.18.1] - 2025-10-06

### Added
- **Module Variables**: Added comprehensive module variable support for container builds
  - Added `module` variable (direct from current config file)
  - Added `version.module` variable (first module from main project.yml)
  - Enhanced `version.modules` variable (comma-separated modules from project.yml)
  - Updated `internal/version/version.go` to detect and provide module variables
  - Updated `cmd/buildfab/main.go` to load module variables in CLI

### Fixed
- **Container Command Display**: Fixed container command display to show interpolated variables instead of raw syntax
  - Analyzed call graph to identify root cause of "${{project}}:${{version}}" display issue
  - Moved container command display from `OnStepStart` to `runContainerAction` where variables are properly interpolated
  - Removed duplicate container command display from `OrderedOutputManager`
  - Created `buildContainerCommandForDisplay` method in Runner for proper command string building
  - Container commands now show actual values like `buildfab:v0.18.1` instead of `${{project}}:${{version}}`

### Enhanced
- **Variable Interpolation**: Enhanced container configuration variable interpolation
  - Implemented comprehensive `InterpolateContainerConfig()` function in `pkg/buildfab/variables.go`
  - Added `interpolateContainerBuild()` function for build configuration interpolation
  - Added `interpolateContainerSlim()` function for slim configuration interpolation
  - All container configuration fields now support variable interpolation (tags, args, targets, exec commands)

### Documentation
- **Memory Bank Updates**: Updated project documentation to reflect variable module implementation
  - Updated `activeContext.md` with comprehensive variable module implementation details
  - Updated `progress.md` with variable module completion status
  - Documented call graph analysis and container display fix methodology

## [0.18.0] - 2025-10-06

### Added
- **Version Variables Extension**: Extended version variables with Git detection functionality
  - Added `version.tag` variable for current Git tag using `git describe --tags --abbrev=0`
  - Added `version.branch` variable for current Git branch with comprehensive fallback handling for detached HEAD state
  - Implemented robust Git detection in `internal/version/version.go` with multiple fallback strategies
  - Enhanced `GetVersionVariables()` method to include new Git-related variables

### Enhanced
- **Variable Interpolation Error Handling**: Comprehensive error handling for undefined variables
  - Updated `pkg/buildfab/variables.go` and `internal/config/config.go` to perform two-pass validation
  - First pass collects all missing variables, second pass performs actual interpolation
  - Clear error messages showing exactly which variables are undefined
  - Available variables list displayed when errors occur for better debugging experience
  - Multiple missing variables detected and reported in single error message

### Documentation
- **YAML Syntax Reference**: Enhanced documentation with new version variables and error handling examples
  - Added `version.tag` and `version.branch` variables to available variables list
  - Added comprehensive error handling section with examples
  - Added branch-specific build examples showing conditional execution
  - Enhanced version variable interpolation examples with new Git variables

## [0.17.0] - 2025-10-05

### Added
- **Slim Container Support**: Complete implementation of slim container functionality using dslim/slim tool
  - Added `SlimImage` method to Engine interface for both Docker and Podman engines
  - Implemented slim container operations with target, tags, network, http_probe, and exec options
  - Created working container-docker-build.yml example demonstrating slim functionality
  - Fixed HTTP probe issues with `--continue-after=exit` flag for non-interactive slim operations

### Fixed
- **Container Command Display**: Fixed container command display in verbose mode level 2 (`-vv`)
  - Build operations now show complete `docker build` commands with all arguments
  - Slim operations now show complete `docker run dslim/slim` commands with all arguments
  - Regular container operations show complete `docker run` commands with all arguments
  - Updated OrderedOutputManager.buildContainerCommand to handle build, slim, and regular operations

### Changed
- **Docker Container Feature Implementation**: Comprehensive Docker and Podman container feature with advanced functionality (100% complete)
  - **Basic Container Execution**: Full support for running containers with Docker and Podman engines, automatic engine detection, and proper error handling
  - **Mount Support**: Comprehensive mount system with bind mounts, read-only options, and automatic workspace mounting for `run_action`/`run_stage` execution
  - **Environment Variables**: Full environment variable support with `env` field and `env_file` loading from mounted workspace directory
  - **Resource Limits**: CPU and memory limit support with simplified CPU configuration (cpu: 2 → --cpus 2.0 --cpuset-cpus "0,1")
  - **Cache Management**: Automatic cache directory mounting to `/tmp/buildfab-cache-{name}` with proper path handling
  - **Buildfab Integration**: Automatic buildfab binary mounting and alias support for easier command usage inside containers
  - **Run Action/Stage Support**: Full `run_action` and `run_stage` execution with proper buildfab binary mounting and configuration copying
  - **Docker Build Support**: Complete `BuildImage` method implementation for both Docker and Podman engines with build args, tags, network, progress, and context support
  - **Slim Image Support**: ContainerSlim configuration with target, tags, network, http_probe, and exec options for creating slim Docker images
  - **Matrix Integration**: Perfect integration with matrix feature for parallel container execution across multiple configurations
  - **Example Configurations**: Created comprehensive example configurations showing Docker build, slim image creation, and matrix container execution
- **Static Binary Requirements**: Updated build rules to require static linking for container compatibility
- **Container Binary Mounting**: Implemented proper binary mounting in containers for run_action and run_stage functionality
- **Matrix-Container Integration**: Matrix variables now properly substitute in container configurations

### Changed
- **Build Process**: All Go binaries must now be built with static linking (`CGO_ENABLED=0` and `-extldflags '-static'`)
- **Container Compatibility**: Static binaries ensure compatibility across Alpine, Ubuntu, and other container images
- **Documentation**: Updated Build.md with static linking requirements and container compatibility notes
- **CPU Configuration**: Simplified CPU feature to use simple integer format (cpu: 2) instead of complex CPU set strings
- **Environment File Loading**: Environment files now load from mounted workspace directory `/tmp/buildfab-workspace/{env_file}` instead of separate mounting

### Fixed
- **Container Binary Execution**: Fixed "executable file not found" errors in containers by using static binaries
- **Matrix Variable Substitution**: Fixed matrix variables not being substituted in container image names and configurations
- **Container Configuration**: Fixed container configuration to properly mount current directory and buildfab binary
- **Container Command Paths**: Simplified container commands to use relative paths since we cd into the workspace directory
- **Environment File Path**: Fixed environment file loading to use correct path from mounted workspace directory
- **CPU Feature Simplification**: Fixed CPU configuration to use simple integer format with automatic CPU set generation

## [v0.16.15] - 2025-10-05

### Fixed
- **Container Streaming Output**: Fixed container engines to provide real-time streaming output using pipes instead of buffered CombinedOutput
- **Matrix-Container Integration**: Fixed matrix expansion to properly interpolate variables in container configurations (image, run commands, environment variables)
- **Shell Compatibility**: Updated container examples to use sh-compatible syntax instead of bash-specific features

### Added
- **Matrix-Container Integration**: Matrix feature now works seamlessly with containers for parallel execution across different images/configurations
- **Container Streaming Callbacks**: Added RunContainerWithCallback method to container engines for real-time output streaming
- **Container Variable Interpolation**: Matrix variables are now properly interpolated in container image names, run commands, and environment variables

### Documentation
- **Container Implementation Status**: Updated documentation with comprehensive analysis of container feature status and matrix integration
- **Matrix-Container Examples**: Added working examples demonstrating matrix execution with containers
- **Shell Compatibility Guidelines**: Added documentation about using sh-compatible syntax in container commands

### Fixed
- **Container Configuration Schema**: Fixed critical schema mismatch where container configuration used `Commands` field instead of required `run` field
  - Updated `pkg/buildfab/container/types.go` to use `Run string` instead of `Commands []string`
  - Updated all container engine implementations (Docker and Podman) to use `run` field consistently
  - Updated all container examples to use `run` field instead of `commands`
  - Container configuration now matches action configuration schema for consistency
  - Basic container execution now works correctly with `run` field

### Documentation
- **Container Implementation Status**: Added comprehensive documentation of current Docker feature implementation status
  - Created `docs/Container-implementation-status.md` with detailed analysis of working and broken features
  - Documented that container feature is ~70% complete with basic execution working
  - Identified critical issues: `run_action`/`run_stage` broken due to missing `prepareContainer()` implementation
  - Listed all working features: basic execution, mounts, environment variables, resource limits
  - Listed missing features: image building, artifact collection, environment file support
  - Provided working examples and next steps for implementation

## [v0.16.14] - 2025-10-03

### Fixed
- **Matrix Streaming Output**: Fixed matrix steps to stream output in real-time instead of buffering
  - Matrix steps now stream output in real-time just like single actions, providing consistent user experience
  - Root cause identified - matrix steps were using OrderedStepCallback buffering system instead of DAG streaming system
  - Solution implemented - modified OrderedOutputManager.OnStepOutput to support streaming for matrix steps without duplication
  - Key changes - added shouldStreamOutput() method to determine streaming eligibility, modified OnStepOutput() to stream immediately for eligible steps, prevented duplication by not buffering already-streamed output
  - Perfect user experience - matrix steps now stream output in real-time with no duplicate output when steps complete
  - Comprehensive testing - verified matrix execution and single action execution have identical streaming behavior

## [v0.16.13] - 2025-10-03

### Fixed
- **Verbosity Flags Parsing**: Fixed verbosity flags parsing issue where `-vv` and `-vvv` flags were not being recognized
  - Updated `handleRegularFlag` function to properly handle multiple consecutive `-v` characters
  - Added specific cases for `-vv` (adds 2 to verbose level) and `-vvv` (adds 3 to verbose level)
  - Maintained backward compatibility with existing `-v`, `--verbose`, `-q`, and `--quiet` flags
  - All verbosity levels now work correctly: `-q` (quiet), `-v` (level 1), `-vv` (level 2), `-vvv` (level 3)

## [v0.16.12] - 2025-10-03

### Added
- **Matrix CLI Flags Feature**: Implemented `--matrix.*` CLI flag functionality to override matrix values defined in YAML configuration
  - Added dynamic matrix flag parsing that allows CLI flags like `--matrix.test_name="custom_test"` and `--matrix.platform="macos"`
  - Extended existing `NewMatrixExpander` function with optional variadic parameter `cliMatrixVars ...map[string]string` for better design and backward compatibility
  - Implemented custom flag parsing to handle `--matrix.*` flags by disabling automatic flag parsing and implementing custom `parseFlags` and `handleRegularFlag` functions
  - Updated both regular Runner and SimpleRunner matrix expansion paths to extract CLI matrix variables and pass them to the matrix expander
  - Matrix value override logic - CLI-provided matrix values now correctly override YAML configuration values, allowing matrix expansion to use CLI values instead of full Cartesian product
  - Comprehensive testing verified functionality works correctly with matrix flags overriding YAML values and creating single matrix jobs instead of full Cartesian product
  - Perfect user experience - users can now use CLI flags like `./bin/buildfab -c ./tests/test_matrix_working.yml working-matrix --matrix.test_name="custom_test" --matrix.platform="macos"` to override matrix values

## [v0.16.10_feat.1_fix.1] - 2025-10-02

### Fixed
- **Matrix Output Doubling**: Fixed critical matrix output doubling issue where first matrix output was being duplicated
  - Root cause identified - output was being displayed twice: once during real-time streaming via OnStepOutput → showStepOutput, and once during buffered output flushing via flushBufferedOutput → showStepOutput
  - Solution implemented - modified OnStepOutput method in OrderedOutputManager to only show output immediately in verbose mode when it's the current step, preventing double display
  - Matrix execution fixed - matrix jobs now show clean, single output without duplication
  - Comprehensive testing - verified fix works correctly with matrix test scenarios showing proper single output display
  - Perfect user experience - users now get clean matrix output without confusing duplicated lines

## [v0.16.10_feat.1] - 2025-10-01

### Added
- **Feature/Matrix Branch Merge**: Successfully merged feature/matrix branch with master branch changes
  - Matrix feature now works perfectly with all master branch changes including verbosity levels, version display, and warning output enhancements
  - Comprehensive testing confirmed matrix expansion, variable interpolation, and parallel execution all work correctly
  - Matrix feature seamlessly integrates with existing buildfab architecture including OrderedOutputManager, step callbacks, and debug logging

## [0.16.5_feat.1_fix.1] - 2025-09-29

### Fixed
- **Matrix Output Suppression**: Fixed race condition in matrix execution where some step output was being lost due to timing issues between step completion and output buffering
  - Added double-flush mechanism in OrderedOutputManager to ensure all buffered output is displayed
  - Matrix steps now consistently show complete output regardless of execution timing
  - Resolved intermittent missing output lines in verbose mode


### Fixed
- **Compilation Errors**: Fixed OnStepComplete method signature calls to include bufferedOutput parameter across all files (buildfab.go, matrix.go)
- **SimpleRunOptions Compatibility**: Updated SimpleRunOptions.Verbose references to use VerboseLevel field for compatibility with verbosity levels feature
- **Version Format**: Updated VERSION file to v0.16.10_feat.1 for feature branch development

## [v0.16.5_feat.1] - 2025-09-26
### Added
- **Matrix Feature Implementation**: Comprehensive matrix builds for parallel execution across multiple configurations
  - Matrix configuration support with configurable parallelism, fail-fast policies, and job management
  - Matrix expansion logic to create multiple jobs from matrix definitions with Cartesian product generation
  - Matrix strategy configuration (max_parallel, fail_fast, continue_on_error, order) for flexible job scheduling
  - Matrix job scheduler with proper queue management, status reporting, and error handling
  - Matrix variable interpolation (${{ matrix.* }}) integration with existing variable system
  - CLI support for matrix-specific commands and overrides with real-time job status reporting
  - Comprehensive test suite for matrix functionality with long-running test scenarios and performance validation
  - Matrix execution patterns integrated with existing DAG execution engine
  - Step callback integration for proper output management and status reporting
  - Matrix job lifecycle management with proper error handling and cleanup
  - Created comprehensive user test examples for different matrix scenarios including basic matrix, parallel execution, error handling, and long-running tests
  - Feature branch created (feature/matrix) with feature version v0.16.5_feat.1 for matrix feature development
  - VERSION 0.16.6 RELEASED with comprehensive matrix feature implementation

## [0.16.10] - 2025-10-01

### Added
- **Version Display Feature**: Added version display functionality to buildfab CLI and library
  - Shows buildfab version and project version when running stages or actions
  - Displays version information in clean format: "buildfab v0.16.10" and "Project buildfab (v0.16.10)"
  - Uses version library to detect project version from VERSION file or git tags
  - Provides consistent version information across all execution contexts (stages and actions)
  - Enhanced user experience with clear version identification before execution
  - Added displayVersionInfo() function to CLI for consistent version display
  - Integrated version display into both runStageDirect() and runActionDirect() functions
  - Refined display format by removing "before push" text for cleaner output

## [0.16.9] - 2025-09-30

### Changed
- **Warning output enhancement**: Enhanced warning output behavior for steps with `onerror: warn` to display the same as errors but with warning icons
  - Modified `showStepCompletion` function to include `StepStatusWarn` alongside `StepStatusError` for buffered output display in quiet mode
  - Updated `enhanceMessage` function to handle both `StepStatusError` and `StepStatusWarn` together for consistent message enhancement
  - Warnings now show buffered output, "execute failure" message, and command reproduction instructions just like errors
  - Warnings use warning icon (`!`) and yellow color instead of error icon (`✗`) and red color
  - Users get comprehensive warning output with all debugging information while maintaining visual distinction from errors

## [0.16.8] - 2025-09-30

### Fixed
- **Default verbose mode**: Fixed CLI and library to default to verbose level 1 (-v) instead of 0
  - Updated `DefaultSimpleRunOptions()` to set `VerboseLevel: 1` by default
  - Updated CLI logic to default to verbose level 1 when no `-v` flags are provided
  - Maintains backward compatibility with `-q` flag for quiet mode (level 0)
  - Ensures consistent default behavior between CLI and library API
  - Users now get detailed output by default without needing to specify `-v` flag

## [0.16.7] - 2025-09-29

### Added
- **Verbosity levels system**: Implemented comprehensive verbosity levels with granular control over output detail and debug options
  - Added `-v`, `-vv`, `-vvv` levels using CountVarP flag for CLI support
  - **Level 0 (quiet, `-q`)**: `<output data>` only on errors, `'to check run'` never
  - **Level 1 (verbose, `-v`)**: `<output data>` always, `'to check run'` never
  - **Level 2 (debug, `-vv`)**: `<output data>` always, `'to check run'` never, adds shell debug options (`-x`)
  - **Level 3 (trace, `-vvv`)**: `<output data>` always, `'to check run'` on errors, adds shell debug options (`-x`)
  - Updated all data structures from `Verbose bool` to `VerboseLevel int` for level-based control
  - Enhanced OrderedOutputManager to support verbosity levels with proper output streaming and buffering
  - Added shell debug options (`-x`) for sh and bash commands on levels 2 and 3
  - Implemented level-based output logic across all components for consistent behavior
  - Comprehensive testing verified all verbosity levels work correctly with proper output behavior
  - Perfect user experience with granular control over output detail and clear level definitions

### Added
- **Quiet mode buffering**: Enhanced quiet mode (-q) to buffer command output and display it when commands fail
  - In quiet mode, stdout and stderr from command execution are captured and stored
  - When a command fails, the buffered output is displayed along with the error message
  - Provides better debugging experience in quiet mode by showing what the command actually output
  - Maintains clean output for successful commands while preserving failure details

## [0.16.6] - 2025-09-29

### Fixed
- **Static build configuration**: Fixed GoReleaser configuration to build static binaries for Linux and Darwin platforms
  - Added `CGO_ENABLED=0` environment variable to disable CGO for static builds
  - Added `-extldflags "-static"` to ldflags to force static linking for Linux and Darwin
  - Verified Linux binaries are now static using ldd command showing "not a dynamic executable"
  - Enhanced cross-platform compatibility with static binaries that don't require external dependencies
  - Fixed build system integration where GoReleaser builds static binaries independently from CMake

## [0.16.5] - 2025-09-25

### Documentation
- **Comprehensive features documentation**: Created comprehensive Features-and-examples.md with detailed examples and usage patterns
  - Added complete documentation of all buildfab features with practical examples
  - Included action variants, conditional execution, include system, and variable interpolation examples
  - Added advanced usage examples for multi-platform builds, environment-specific deployment, and conditional testing
  - Provided comprehensive CLI usage examples and library API examples
  - Added best practices section for configuration organization, error handling, performance, and security
- **YAML syntax reference**: Created comprehensive YAML-syntax-reference.md with complete syntax documentation
  - Added complete reference for all YAML configuration fields, types, and usage examples
  - Documented expression language with all operators, helper functions, and variable types
  - Provided validation rules and error handling documentation
  - Included complete examples for simple projects, cross-platform builds, modular configurations, and complex conditional pipelines
- **README improvements**: Updated README.md to link to comprehensive documentation instead of inline features
  - Simplified features section with key highlights and link to detailed documentation
  - Added proper documentation navigation with links to all comprehensive guides
  - Improved user experience with better organization and discoverability of documentation

## [0.16.4] - 2025-09-25

### Fixed
- **GitHub Actions Windows binary path**: Fixed GitHub Actions Windows platform detection test by correcting binary path logic
  - Simplified binary path handling by removing unnecessary platform-specific logic
  - Go automatically handles .exe extension on Windows, so unified `./bin/buildfab` path works for all platforms
  - Eliminated complex conditional logic in GitHub Actions workflow
  - Fixed Windows test failures on windows-latest runners
  - Enhanced cross-platform testing reliability across all GitHub Actions runners

## [0.16.3] - 2025-09-25

### Updated
- **Version-go library**: Updated from v1.1.1 to v1.2.5 to fix macOS OS version detection
  - Fixed macOS platform detection to return numeric version (e.g., "15.0") instead of "darwin"
  - Improved cross-platform compatibility for platform detection tests
  - Enhanced platform variable accuracy across all supported platforms

## [0.16.2] - 2025-09-25

### Fixed
- **Platform detection test output**: Fixed platform detection test output to return clean values without validation text for better automation and CI/CD integration
  - Updated `tests/cross-platform/unified-platform-validation.yml` to output clean values (e.g., "CPU: 8" instead of "CPU: 8cores(CORRECT->=1)")
  - Removed validation text from success messages for all platform variants (Linux, macOS, Windows)
  - Simplified output parsing for GitHub Actions and other automation systems
  - Verified clean output works correctly with existing validation scripts
- **GitHub Actions platform detection tests**: Fixed failing platform-detection-unit-tests by updating workflow to use unified configuration instead of removed individual platform files
  - Updated `.github/workflows/cross-platform-test.yml` to use `unified-platform-validation.yml` instead of `linux_configuration.yml`, `windows_configuration.yml`, and `macos_configuration.yml`
  - Simplified platform detection test to use single unified configuration with platform-specific variants
  - Updated `tests/cross-platform/README.md` to reflect new unified configuration approach
  - Verified unified configuration works correctly on Linux platform with proper validation

### Future Enhancements
- **Platform-specific tool installation**: Extend pre-check and pre-install tools for other platforms with conditional execution using `when` conditions
  - Add platform-specific variants for tool installation actions (Windows, macOS, Linux) with appropriate `when` conditions using `${{ platform }}` variable
  - Ensure pre-check and pre-install stages work correctly across all supported platforms (linux/amd64, linux/arm64, windows/amd64, windows/arm64, darwin/amd64, darwin/arm64)
  - Use action variants with `when` conditions to execute platform-specific tool installation commands
  - Provide platform-appropriate installation instructions and error messages
  - Verify pre-check and pre-install stages work correctly on all target platforms

## [0.16.0] - 2025-09-25

### Added
- **Self-Building Capability**: Comprehensive self-building capability for buildfab project with automatic tool checking and installation
  - Added pre-check stage with comprehensive tool verification (conan, cmake, goreleaser, go, version utility, pre-push utility)
  - Extended pre-install stage with automatic installation of missing tools
  - Moved tool check actions to include files for better maintainability
  - Updated all build stages to depend on pre-check stage for consistent tool verification
  - Enhanced Conan installation with multiple fallback methods for externally managed Python environments
  - Created comprehensive build system rules and guidelines
  - Updated documentation with detailed self-building instructions and build stages

## [0.15.5] - 2025-09-25
### Added
- **OnError Policy Functionality**: Implemented comprehensive onerror policy functionality for buildfab
  - Fixed `onerror: warn` policy to correctly convert builtin action errors to warnings
  - Updated git actions (git@untracked, git@modified, git@uncommitted) to return errors instead of warnings when issues are detected
  - Enhanced step callback logic to display proper action success messages instead of generic "executed successfully"
  - Created comprehensive test suite (`TestOnErrorPolicyWithTestActions`) with simple test actions (test@success, test@failure, test@warn) for reliable testing
  - Fixed existing test expectations (`TestStepCallbackIntegration`) for onerror policy behavior
  - All tests passing with proper onerror policy verification
  - Perfect user experience: users can now use `onerror: warn` in their project.yml to convert git action errors to warnings, allowing pre-push stages to continue while still showing warning status

### Fixed
- **Version Build Issue**: Fixed CMake cache issue where buildfab binary was showing wrong version (v0.10.2 instead of v0.15.5)
  - Cleaned build directory to force CMake to re-read VERSION file
  - Binary now correctly shows v0.15.5 as expected

## [0.15.4] - 2025-09-25

### Added
- **Dry Run Mode**: Added comprehensive `--dry-run` flag support for both CLI and library usage
  - New `--dry-run` global flag that shows what would be executed without running commands
  - Added `DryRun` field to `RunOptions` and `SimpleRunOptions` structs for library integration
  - Custom actions display interpolated commands that would be executed with perfect formatting
  - Built-in actions show their descriptions indicating what they would do
  - Stage execution shows step-by-step execution plan with command details and summary statistics
  - Perfect command formatting with 💻 icon, ✓ checkmark, and proper multiline indentation
  - Supports quiet mode (`--quiet`) for summary-only output
  - Perfect for testing configurations and understanding execution plans before running
  - Updated README.md with usage examples and comprehensive documentation

## [0.15.3] - 2025-09-25

### Documentation
- **Complete Changes Workflow Rule**: Updated rule-complete-changes.mdc with enhanced workflow and comprehensive push failure recovery process
  - Enhanced workflow with proper buildfab action usage (install-binary)
  - Added version bump type selection (patch/minor/major) based on project changes
  - Implemented proper commit message template with `<change details>` placeholder for future use
  - Added comprehensive push failure recovery process with detailed 5-step recovery workflow
  - Perfect rule coverage now includes proper error handling, push failure recovery, and template-based commit messages
  - Tested complete workflow execution with successful version bump to v0.15.3

## [0.15.2] - 2025-09-25

### Fixed
- **Darwin/macOS Naming Consistency**: Fixed all darwin/macos naming inconsistencies throughout the project
  - Updated GoReleaser configuration to use consistent darwin naming in archive names
  - Renamed installer scripts from macos to darwin naming (buildfab-darwin-amd64-install.sh, buildfab-darwin-arm64-install.sh)
  - Updated Homebrew formula to use darwin archive names (buildfab_darwin_amd64.tar.gz, buildfab_darwin_arm64.tar.gz)
  - Fixed cross-platform test files to use darwin naming (test-platform-darwin.sh, Dockerfile.darwin)
  - Updated all documentation to use consistent darwin naming throughout
  - Perfect consistency - all binary names, archive names, installer scripts, and documentation now use consistent darwin naming

## [0.15.1] - 2025-09-25

### Added
- **Update Checking Feature**: Comprehensive update checking system for buildfab and its utilities
- **Check-updates stage**: Added modular stage to check for updates of buildfab, pre-push utility, and version utility
- **GitHub API integration**: Actions to fetch latest release information from GitHub repositories
- **Pre-install stage**: Added stage to download and install latest versions of utilities into scripts/ directory
- **Platform variable support**: Used buildfab built-in `${{ platform }}` and `${{ arch }}` variables for cross-platform compatibility
- **Modular configuration**: Organized update checking into separate include files for better maintainability
- **Error handling**: Implemented `onerror: warn` policy for update checks to prevent build failures

### Fixed
- **Version comparison**: Fixed version format mismatches by stripping 'v' prefix from GitHub release tags
- **Pre-push utility detection**: Corrected file path from `scripts/prepush` to `scripts/pre-push` for proper detection
- **Repository URLs**: Updated all GitHub API calls and download URLs to use correct repository names

### Changed
- **Installation actions**: Updated to use correct GitHub release URLs with platform-specific install scripts
- **Update checking logic**: Enhanced to properly compare versions and provide clear update notifications

## [0.15.0] - 2025-09-25

### Added
- **Include Feature**: Comprehensive include system for organizing complex configurations into smaller, manageable files
- **Include field**: Added `include` field to Config struct supporting file patterns in YAML configuration
- **File pattern matching**: Support for both exact file paths and glob patterns with comprehensive resolution logic
- **Exact file path support**: Include specific files that must exist or configuration fails with syntax error
- **Glob pattern support**: Include files matching patterns (e.g., `config/*.yml`, `file-*.yml`) where directory must exist but files are optional
- **Circular include detection**: Prevents infinite loops with proper visited file tracking and error reporting
- **File deduplication**: Ensures same file isn't processed multiple times when matching multiple patterns
- **Nested includes**: Included files can also have includes, creating hierarchical configuration structure
- **Comprehensive error handling**: Clear error messages for missing files, invalid patterns, and circular dependencies
- **Configuration merging**: Later includes override earlier ones with proper action and stage merging logic
- **YAML file filtering**: Only `.yml` and `.yaml` files are processed from glob patterns

### Changed
- **Configuration loading**: Enhanced both internal and public configuration loading to process include patterns
- **Documentation**: Updated Project Specification and README with detailed include feature documentation and examples
- **Examples**: Added complete example configuration demonstrating how to split configurations across multiple files

### Documentation
- **Project Specification**: Enhanced `docs/Project-specification.md` with include behavior specification and validation expectations
- **README**: Added comprehensive include feature documentation with behavior explanation and usage examples
- **Example configurations**: Created complete example in `examples/` directory showing split configuration approach
- **Include behavior**: Documented exact file path vs glob pattern behavior with clear error handling rules

### Testing
- **Include resolver tests**: Created extensive test suite in `internal/config/include_test.go` covering all scenarios
- **Configuration loading tests**: Added comprehensive tests in `pkg/buildfab/config_include_test.go` for include processing
- **Error condition testing**: Tests for circular includes, missing files, invalid patterns, and merge behavior
- **Integration testing**: Verified include feature works correctly with configuration validation and stage execution

## [0.14.1] - 2025-09-25

### Added
- **Unified Cross-Platform Configuration**: Created single unified cross-platform configuration using variants for platform validation testing
- **Variant-based approach**: Created `tests/cross-platform/unified-platform-validation.yml` with platform-specific variants using `when` conditions with `${{ }}` expression syntax
- **Comprehensive platform validation**: Each variant has appropriate validation logic for Linux (Ubuntu/Debian), macOS (Darwin), and Windows (PowerShell) platforms
- **Infrastructure updates**: Modified Dockerfiles and test scripts to use unified configuration instead of individual platform-specific files

### Changed
- **Condition evaluation**: Fixed variant conditions to use `platform` variable instead of `os` variable for proper platform detection
- **Test infrastructure**: Updated cross-platform testing to use single unified configuration file

### Removed
- **Individual platform configurations**: Removed `linux_configuration.yml`, `macos_configuration.yml`, and `windows_configuration.yml` files
- **Alternative Windows shell variant**: Removed cmd.exe variant from unified configuration to simplify structure

### Fixed
- **Platform detection**: Corrected variant condition evaluation to properly detect Linux, macOS, and Windows platforms
- **Dockerfile references**: Updated Dockerfiles to reference unified configuration instead of individual platform files

## [0.14.0] - 2025-09-25

### Added
- **Step If Condition Feature**: Implemented conditional execution for stage steps using the existing expression evaluator
- **Step if condition support**: Steps can now use `if` conditions with the same powerful expression language as action variants
- **Conditional step execution**: Added `shouldExecuteStepByCondition()` function that evaluates `if` conditions using existing `evaluateCondition()` and `EvaluateExpression()` functions
- **Enhanced DAG execution**: Updated all three DAG execution loops (`executeDAGWithCallback`, `executeDAGWithOrderedStreaming`, `executeDAGWithParallel`) to check step conditions before execution
- **Proper skip handling**: Steps that fail `if` conditions are marked with `StatusSkipped` and properly displayed in output with "skipped (condition not met)" status
- **Step callback integration**: Skipped steps trigger `OnStepComplete` with `StepStatusSkipped` to ensure they appear in ordered output display
- **Comprehensive testing**: Created unit tests (`pkg/buildfab/step_if_condition_test.go`) and integration tests (`tests/test-step-if-condition*.yml`) for various condition scenarios
- **Documentation updates**: Enhanced `docs/Project-specification.md` and `README.md` to document step `if` condition support with examples
- **Platform variable support**: Steps can use `os`, `arch`, `platform` variables and helper functions in `if` conditions
- **Logical operators**: Full support for `&&`, `||`, `!` operators in step conditions
- **Helper functions**: Support for `contains()`, `startsWith()`, `endsWith()`, `matches()`, `fileExists()`, `semverCompare()` in step conditions

### Documentation
- **Updated Project Specification**: Added `if` condition support for steps in configuration format and validation expectations
- **Updated README**: Added step `if` condition feature to Features section with usage example
- **Enhanced Examples**: Added conditional execution examples showing platform-specific step execution

## [0.13.0] - 2025-09-25

### Added
- **Enhanced When Conditions Expression Language**: Implemented comprehensive expression language for `when` conditions similar to GitHub Actions
- **Expression evaluation engine**: New `pkg/buildfab/expression.go` with full expression parsing and evaluation system supporting variables, operators, and helper functions
- **Variable system**: Support for `os`, `arch`, `env.VAR`, `inputs.NAME`, `matrix.os`, `ci`, `branch` with proper variable resolution and user variable override
- **Logical operators**: Complete support for `&&`, `||`, `!` with proper operator precedence and parentheses handling
- **Comparison operators**: Support for `==`, `!=`, `<`, `<=`, `>`, `>=` with lexicographic string comparison and numeric comparison
- **Helper functions**: `contains()`, `startsWith()`, `endsWith()`, `matches()`, `fileExists()`, `semverCompare()` with comprehensive error handling
- **Enhanced condition evaluation**: Updated `evaluateCondition()` function to use new expression system while maintaining backward compatibility
- **Comprehensive test suite**: Created extensive test coverage with 25+ expression tests, 15+ helper function tests, and 20+ condition evaluation tests

### Fixed
- **Variable resolution**: User variables now properly override platform variables in expression context
- **String-to-boolean conversion**: Fixed `toBool()` function to properly convert non-empty strings to `true` (except "false")
- **Operator precedence**: Fixed logical operator parsing to handle parentheses and nested expressions correctly
- **Type handling**: Enhanced type conversion functions to handle `int` and `int64` types in numeric comparisons

### Changed
- **Expression system integration**: `evaluateCondition()` function now uses new expression evaluation engine
- **Backward compatibility**: Old simple equality checks continue to work while new complex expressions are supported
- **Test coverage**: Updated existing variants tests to work with new expression system

## [0.12.0] - 2025-09-23

### Added
- **Action Variants Feature**: Implemented comprehensive action variants feature allowing conditional execution of different commands within a single action based on `when` conditions
- **ActionVariant struct**: New type supporting `when` conditions with `run`, `uses`, and `shell` fields for conditional execution
- **Enhanced Action struct**: Added `Variants` field to support multiple variants per action with first-matching selection logic
- **Conditional evaluation system**: Created `evaluateCondition()` function supporting both `==` and `=` operators with variable interpolation using `${{ variable }}` syntax
- **Variant selection logic**: `SelectVariant()` method picks first matching variant or returns nil for skipping when no conditions match
- **Comprehensive test suite**: Created extensive test coverage for variant selection, condition evaluation, validation, and end-to-end execution scenarios
- **Example YAML configurations**: Added `test-variants.yml`, `test-variants-simple.yml`, and `test-variants-clean.yml` demonstrating variants usage

### Changed
- **Updated execution flow**: Modified both `runActionInternal()` and `executeActionForDAGWithCallback()` to handle variant selection and skipped actions with proper status reporting
- **Enhanced validation**: Updated `Config.Validate()` to ensure actions with variants don't have direct `run`/`uses` fields and each variant has required fields
- **Improved condition syntax**: Support both `==` and `=` operators for equality comparisons in `when` conditions for better user experience

### Technical Details
- **Action variants**: Actions can now define multiple variants with `when` conditions that are evaluated in order
- **Condition evaluation**: Supports simple equality comparisons with variable interpolation (`${{ os == 'linux' }}`, `${{ platform = 'windows' }}`)
- **Automatic skipping**: Actions with variants that don't match any condition are automatically skipped with clear reason
- **Variable integration**: Variants work seamlessly with existing platform variables (`os`, `platform`, `arch`, `os_version`, `cpu`) and custom variables
- **Backward compatibility**: Existing actions without variants continue to work exactly as before

## [0.11.0] - 2025-09-25

### Added
- **Comprehensive cross-platform testing system**: Implemented validation testing that compares detected platform values against expected values and fails on mismatch
- **Enhanced platform detection validation**: Added comprehensive validation for Linux (Ubuntu/Debian), Windows (Wine), and macOS platforms with proper error handling and clear success/failure messages
- **Cross-platform test configuration**: Added `test` stage to `.project.yml` for local execution of cross-platform tests with `buildfab test`
- **Container runtime detection**: Automatic detection of Podman (preferred) or Docker for cross-platform testing with proper fallback handling
- **Platform-specific validation logic**: PowerShell scripting for Windows validation, bash scripting for Linux/macOS validation with proper syntax for each platform

### Fixed
- **Fixed YAML syntax errors**: Resolved malformed action definitions and dangling action configurations in `.project.yml` test stage
- **Fixed project configuration structure**: Cleaned up test stage configuration to remove syntax errors and malformed YAML
- **Fixed cross-platform test execution**: All tests now passing: Ubuntu, Debian, Windows (Wine), and macOS (with graceful skip on non-macOS hosts)
- **Fixed test validation system**: Platform detection variables are actively validated against expected values with clear error messages and proper exit codes

### Changed
- **Simplified test configuration**: Removed complex Git Bash testing and focused on essential cross-platform functionality with `cmd.exe` and PowerShell for Windows
- **Enhanced test workflow**: Streamlined test workflow with consistent container runtime detection and proper error handling throughout
- **Updated test documentation**: Enhanced cross-platform README with validation testing details and platform-specific configuration information
- **Improved test reliability**: Clean test suite with comprehensive validation system ensuring platform detection accuracy

## [0.10.3] - 2025-09-24

### Added
- **Enhanced YAML validation error reporting with line numbers**: All validation errors now show `config_file:line: error` format for easy editor navigation
- **Configuration validation in all CLI commands**: Every command that works with configuration now validates it before execution
- **Enhanced error location detection**: Added `enhanceValidationError` function to provide precise error location in YAML files
- **Improved user experience**: Users can now directly open files in editors at the exact error location (e.g., `mcedit config.yml:43`)

### Fixed
- **Fixed duplicate error message issue**: Eliminated duplicate error messages in CLI validation output
- **Fixed missing validation in validate command**: `runValidate` now properly calls validation and shows line numbers
- **Fixed error handling in config loading**: `buildfab.LoadConfig` now preserves original validation errors for better error reporting
- **Fixed validation coverage**: All commands now validate configuration: `validate`, `list-actions`, `list-stages`, `list-steps`, `build`, `action`, `run`

### Changed
- **Updated configuration loading**: Both `pkg/buildfab/config.go` and `internal/config/config.go` now call validation during config loading
- **Enhanced error messages**: All validation errors now display with ANSI color codes (red) for better visibility
- **Improved debugging experience**: Configuration errors now provide actionable information with exact file locations
- **Streamlined error handling**: Consistent error handling across all CLI commands with enhanced validation error reporting

## [0.10.2] - 2025-09-24

### Added
- **Single Action Status Display**: Comprehensive status display for single actions with proper SUCCESS, FAILED, and TERMINATED status handling
  - Added final status display for single actions with "🎉 SUCCESS", "💥 FAILED", or "⏹️ TERMINATED" messages
  - Enhanced termination handling with proper TERMINATED status when actions are interrupted with Ctrl+C
  - Comprehensive status logic with proper priority system: TERMINATED > FAILED > SUCCESS

### Fixed
- **Double Error Messages**: Fixed issue where buildfab was showing action errors twice (once from step callback, once from CLI error handling)
- **CLI Error Handling**: Improved CLI error handling to not show duplicate error messages for execution errors
- **Status Display**: Single actions now show proper final status like stages do with appropriate icons and colors
- **Termination Status**: When actions are interrupted with Ctrl+C, they now show "TERMINATED" status instead of incorrectly showing "FAILED"

### Changed
- **Error Output**: CLI now only shows usage hints for "not found" errors, not for execution errors
- **Status Priority**: Implemented proper status priority system with appropriate visual formatting
- **User Experience**: Users now get clean, non-duplicated error messages with accurate status feedback for all execution scenarios

## [0.10.1] - 2025-09-24

### Fixed
- **Error Message Improvements**: Fixed silent error handling and improved error message grammar for non-existent stages and actions
  - Fixed silent error handling - resolved issue where buildfab was not reporting anything when stage, name, or unknown arguments were provided
  - Added proper error output - CLI now displays clear error messages before exiting instead of silent failures
  - Enhanced error messages - improved grammar from "To see list stages" to "To see available stages" for better readability
  - Added helpful guidance - error messages now include suggestions to run `buildfab list-stages` and `buildfab list-actions` to discover available options
  - Comprehensive testing - verified all error scenarios work correctly with proper error messages and helpful guidance
  - Perfect user experience - users now get clear, actionable error messages with helpful suggestions instead of silent failures

## [0.10.0] - 2025-09-24

### Added
- **Platform Detection Variables Feature**: Implemented comprehensive platform detection variables using the latest version-go library (v1.1.1) with new platform detection API
  - Added platform variable system - created `pkg/buildfab/platform.go` with functions to detect platform, architecture, OS, OS version, and CPU count using `version.GetPlatformInfo()` from version-go v1.1.1
  - Implemented variable interpolation - created `pkg/buildfab/variables.go` with `InterpolateVariables()` function to replace `${{ variable }}` placeholders in action commands
  - Updated buildfab library integration - modified `pkg/buildfab/buildfab.go` and `pkg/buildfab/simple.go` to automatically include platform variables in `DefaultRunOptions()` and `DefaultSimpleRunOptions()`
  - Enhanced CLI integration - updated `cmd/buildfab/main.go` to ensure platform variables are passed to runners in both `runActionDirect()` and `runStageDirect()` functions
  - Updated project.yml - modified build actions to use new platform variables with `${{ platform }}`, `${{ arch }}`, `${{ os }}`, `${{ os_version }}`, and `${{ cpu }}` syntax
  - Comprehensive testing - verified platform variables work in all execution contexts: single actions, stages, CLI execution, and API library usage
  - Perfect integration - platform variables are automatically available in all action commands with seamless variable interpolation

### Changed
- **Updated version-go dependency**: Upgraded from v0.8.22 to v1.1.1 to utilize new platform detection API
- **Enhanced variable system**: Platform variables are now automatically included in all execution contexts
- **Improved action execution**: All action commands now support `${{ variable }}` interpolation with platform information

### Documentation
- **Updated memory bank files**: Added platform detection variables feature to activeContext.md and progress.md
- **Enhanced project.yml**: Added platform variable usage examples in build actions

## [0.9.1] - 2025-09-24

### Added
- **Execution Time Display Feature**: Implemented comprehensive execution time measurement and display for both actions and stages
  - Added step execution time formatting with `formatExecutionTime` function that formats durations as requested: fractional seconds for <1s (e.g., '0.002s'), whole seconds for 1-59s (e.g., '20s'), and minutes+seconds for ≥60s (e.g., '1m 20s')
  - Enhanced step completion display - successful actions now show execution time (e.g., "executed successfully - in '0.021s'") while errors and warnings don't show timing
  - Added stage timing - stages now show start message ("▶️ Running stage: stage-name") and completion with timing ("🎉 SUCCESS - stage-name in 3s")
  - Unified formatting - both step and stage execution times use consistent "in" format instead of parentheses for perfect consistency
  - Updated both output systems - modified both SimpleStepCallback and OrderedOutputManager to display execution times consistently
  - Perfect user experience - users get precise execution timing for successful operations with clear, consistent formatting

### Fixed
- **Timing Measurement**: Corrected timing to measure actual step execution duration from start to completion, not including callback overhead
- **Stage Output Format**: Unified stage output format to match action format by removing CLI header and project information display

### Changed
- **Stage Start Display**: Added stage start message display using UI.PrintStageHeader for better user feedback
- **Stage Completion Format**: Updated stage completion messages to use "in" format instead of parentheses for consistency with step execution times

## [0.9.0] - 2025-09-23

### Added
- **Automatic Shell Error Handling**: Implemented automatic shell error handling by adding `-euc` flags to all shell command executions
  - All shell commands now automatically use `sh -euc` which includes `-e` (exit on error), `-u` (exit on undefined variables), and `-c` (execute command)
  - Commands that fail now properly cause actions to fail instead of continuing and reporting success
  - Fixed version-module action - the `ddffd` command that doesn't exist now properly fails the action instead of reporting success
  - Enhanced error message formatting - improved error messages to show "to check run:" with properly aligned commands
  - Updated all shell execution points - modified three key shell command execution methods in buildfab.go to use proper error handling flags
  - Comprehensive testing - verified fix works correctly for single-line commands, multiline scripts, and complex actions
  - Perfect user experience - users now get accurate error reporting when commands fail, with clear reproduction instructions

## [0.8.18] - 2025-09-23

### Fixed
- **Test Race Conditions**: Fixed data race conditions in MockStepCallback test struct that were causing test failures
  - Added thread safety with sync.Mutex to MockStepCallback struct
  - Protected all write operations (OnStepStart, OnStepComplete, OnStepOutput, OnStepError, Reset) with mutex locks
  - Protected all read operations in test methods by copying data while holding the lock
  - Fixed race conditions in parallel step execution tests
  - All tests now pass with `go test ./... -v -race` without any race condition warnings
  - Maintained full test functionality while ensuring thread safety

## [0.8.17] - 2025-09-23

### Fixed
- **Streaming Output Fix**: Fixed OrderedOutputManager to provide true streaming output instead of buffering output until step completion
  - Fixed immediate streaming - output now streams immediately as it's produced for the currently active step, not buffered until completion
  - Fixed parallel step buffering - steps that run in parallel but need to wait their turn now properly buffer their output and flush it when they become the active step
  - Added flushBufferedOutput method - implemented proper buffering and flushing logic for steps that can't stream immediately
  - Enhanced checkAndShowNextStep - now flushes buffered output when a step becomes the current active step
  - Enhanced checkAndShowCompletedSteps - now flushes buffered output when showing completed steps in order
  - Fixed executor integration - added OnStepOutput calls in the executor to properly pass output to the OrderedOutputManager
  - Perfect streaming behavior - both sequential steps (test-streaming) and parallel steps (test-parallel) now work correctly with proper output ordering and immediate streaming
  - Comprehensive testing - verified fix works correctly for both sequential and parallel execution scenarios

### Added
- **Interactive Command Support**: Added stdin connection for interactive commands
  - Connected cmd.Stdin = os.Stdin to allow commands to read from terminal input
  - Interactive prompts are now visible in the output stream in real-time
  - Commands that require user input (like sudo) show their prompts correctly
  - Note: Full interactive input handling has limitations due to subprocess execution constraints

### Technical Details
- **OrderedOutputManager Enhancements**:
  - Modified OnStepOutput to stream output immediately if it's the current active step
  - Added flushBufferedOutput method to flush all buffered output when a step becomes active
  - Updated checkAndShowNextStep to flush buffered output when a step becomes the current step
  - Updated checkAndShowCompletedSteps to flush buffered output when showing completed steps
- **Executor Integration**:
  - Added OnStepOutput calls in executeCommandWithStreaming for both stdout and stderr
  - Added OnStepOutput calls in executeCustomAction for buffered output mode
  - Connected cmd.Stdin = os.Stdin for interactive command support

## [0.8.16] - 2025-09-23

### Fixed
- **Ordered Output Manager Fixes**: Fixed critical issues in the OrderedOutputManager implementation after refactoring
  - Fixed output ordering issues where steps were completing out of order and not being displayed in correct sequential order
  - Fixed duplicate output issue where step output was being shown both in OnStepOutput and showStepCompletion methods
  - Added checkAndShowCompletedSteps method to properly handle completed steps that can now be shown in order
  - Enhanced completion logic to ensure all completed steps are displayed in the correct order
  - Fixed missing step completions - all steps now properly show their completion messages in the correct sequential order
  - Perfect user experience with clean, ordered output and no duplicate display

### Verified
- **Library Refactoring**: Confirmed that the library buildfab is correctly using the new refactoring approach
  - Verified OrderedStepCallback and OrderedOutputManager are properly integrated
  - Confirmed both CLI and library use the same output management system
  - Tested comprehensive output ordering in both verbose and silence modes

## [0.8.15] - 2025-09-23

### Fixed
- **Ctrl+C Termination Message**: Fixed issue where Ctrl+C was working but output didn't show "TERMINATED!" after refactoring
  - Added proper context cancellation detection in both runStageInternal and executeStageWithCallback methods
  - Added printTerminatedSummary method in SimpleRunner that displays "⏹️ TERMINATED" message with yellow color
  - Enhanced RunStage method to check for termination and call appropriate summary method (terminated vs normal)
  - Fixed both execution paths to properly handle context cancellation and display termination messages
  - Perfect user experience with clear "TERMINATED" status and proper summary statistics

## [0.8.14] - 2025-09-23

### Added
- **Queue-Based Output Manager**: Implemented OrderedOutputManager for perfect sequential output display
  - New queue-based system that manages step output in proper sequential order using a queue approach
  - Eliminates mixed output between parallel steps with proper buffering and ordered display logic
  - All steps now show their start messages (○ step-name running...) in correct sequential order
  - Fixed last step issue - goreleaser-dry-run step now properly shows both start and completion messages
  - Implemented OrderedStepCallback - new StepCallback implementation that delegates all output to the OrderedOutputManager
  - Perfect sequential output - steps run in parallel for performance but display output sequentially in declaration order
  - Comprehensive testing - verified fix works correctly for all steps including the last step

### Added
- **Comprehensive Debug Logging**: Added extensive debug output with -d|--debug flag for complex changes
  - Debug output traces queue state and decision-making process in OrderedOutputManager
  - Shows step registration, queue state, and output decisions in real-time
  - Helps developers understand and debug complex output management logic
  - Created debug output rule - documented best practices for using debug output during complex changes
  - Essential for troubleshooting and understanding queue-based output management behavior

### Fixed
- **Missing Step Start Messages**: Fixed issue where only the first step showed its start message
  - All steps now properly show their start messages in correct sequential order
  - Steps wait for previous step to complete before showing their start message
  - Perfect sequential display: ○ step1 running... → ✓ step1 executed successfully → ○ step2 running...
  - Resolves user-reported issue where step start messages were missing for all but the first step

### Fixed
- **Last Step Display Issue**: Fixed goreleaser-dry-run step not showing start message
  - Last step now properly shows both start and completion messages
  - Queue-based logic correctly handles the last step in the execution sequence
  - All steps, including the last one, now display their start and completion messages correctly
  - Perfect user experience with complete visibility into all step execution

### Changed
- **Output Management Architecture**: Replaced StreamingOutputManager with OrderedOutputManager
  - New architecture uses queue-based approach instead of streaming manager logic
  - Executor now delegates all output responsibility to OrderedOutputManager
  - Simplified executor logic by centralizing output management in dedicated component
  - Better separation of concerns between execution and output display

### Documentation
- **Debug Output Rule**: Created comprehensive rule for using debug output during complex changes
  - Documented best practices for implementing debug logging in complex logic
  - Added rule to .cursor/rules/rule-debug-output.mdc for future reference
  - Emphasizes importance of debug output for understanding queue state and decision-making
  - Helps developers implement and debug complex output management systems

## [0.8.13] - 2025-09-23

### Fixed
- **Nil Error Wrapping Bug**: Fixed critical bug in runStageInternal function where fmt.Errorf with %w was being called with nil error
  - Added nil check before error wrapping to prevent formatting errors
  - When result.Status == StatusError but result.Error == nil, now uses %s with result.Message instead of %w
  - Prevents "invalid verb %w for value of type error" formatting errors
  - Added comprehensive test case TestNilErrorWrapping to verify fix
  - Resolves issue where buildfab library would crash with formatting errors in certain error conditions

## [0.8.12] - 2025-09-23

### Fixed
- **Git Actions Status Handling**: Fixed all git actions (git@untracked, git@uncommitted, git@modified) to properly report warning status instead of error status
  - Updated RunAction method to check Result.Status for built-in actions, not just error presence
  - Fixed SimpleStepCallback to use errorOutput (stderr) instead of output (stdout) for step results
  - All git actions now show warning icons (!) with yellow color instead of error icons (✗) with red color
  - Actions now exit with code 0 (success) instead of error codes when issues are detected
  - Perfect user experience - users get helpful warnings instead of confusing error exits

### Changed
- **Git Actions Message Format**: Standardized all git action messages to use consistent formatting
  - All three git actions now use the same message format ending with "to check run:\n    git status"
  - git@untracked: "Untracked files found, to check run:\n    git status"
  - git@uncommitted: "Uncommitted changes found, to check run:\n    git status"
  - git@modified: "There are modified files, to check run:\n    git status"
  - Consistent indentation and formatting across all git actions
  - Users get clear, actionable instructions for checking git status

## [0.8.11] - 2025-09-23

### Fixed
- **GitHub Release**: Fixed release process on GitHub
  - Bumped version to v0.8.11 to resolve release issues
  - Updated packaging files for Windows Scoop and macOS Homebrew
  - Ensured proper version synchronization across all platforms

## [0.8.10] - 2025-09-23

### Added
- **VERSION File Integration**: Modified build process to read version from VERSION file
  - VERSION file is now the primary source of truth for version information
  - CMake build process prioritizes VERSION file over external version utilities
  - Simplified version management with single source of truth
  - Eliminates dependency on external version utility downloads
  - All builds now consistently use VERSION file for version embedding
  - Verified with `buildfab build` command - works perfectly

## [0.8.9] - 2025-09-23

### Fixed
- **Streaming Output Synchronization**: Fixed mixed streaming output issue caused by parallel command execution
  - Implemented `StreamingOutputManager` to coordinate output from parallel commands
  - Steps now run in parallel for performance but display output sequentially in declaration order
  - Only the first step in declaration order streams its output at any given time
  - Eliminated mixed output between parallel steps during execution
  - Resolves issue where Ctrl+C termination fix broke streaming output ordering
  - Both parallel and sequential execution scenarios now work correctly
- **Output Buffering System**: Implemented comprehensive output buffering for steps that cannot stream yet
  - Steps that cannot stream yet now buffer their output instead of discarding it
  - Buffered output is flushed when the step becomes active in declaration order
  - Ensures no output is lost during parallel execution
  - Perfect user experience with complete output visibility in correct order
  - Both verbose and non-verbose modes support output buffering
- **Success Message Ordering**: Fixed success messages appearing in completion order instead of declaration order
  - Success messages now appear in declaration order for both parallel and sequential execution
  - Implemented `ShouldShowStepSuccess` method to control when success messages are displayed
  - Success messages are displayed when steps actually complete, not when they start
  - Eliminated duplicate success messages in sequential execution scenarios
  - Perfect user experience with properly ordered success messages
- **Race Condition Fixes**: Fixed race conditions in test utilities and production code
  - Added mutex synchronization to `StreamingOutputManager` for thread-safe concurrent access
  - Fixed race conditions in `captureUI` test utility with proper mutex protection
  - All UI methods in test utilities now use proper locking for thread safety
  - All tests now pass with `-race` flag enabled for comprehensive race detection
  - Production code is now fully thread-safe for concurrent execution scenarios

### Documentation
- **Memory Bank Updates**: Added Immediate Actions from static analysis as future development tasks
  - Added test coverage improvement targets (80%+ coverage for production readiness)
  - Added performance testing requirements for large dependency graphs
  - Added git action testing setup requirements
  - Updated activeContext.md and progress.md with specific action items and metrics
  - Documented current test coverage by package for targeted improvements

## [0.8.8] - 2025-01-27

### Fixed
- **Version Display Issue**: Fixed version commands returning "unknown" when binary is installed globally or run from bin directory
  - Updated `getVersion()` function to use build-time `appVersion` variable set via ldflags only
  - Removed VERSION file fallback - built application never reads VERSION file at runtime
  - Version commands now work correctly regardless of working directory
  - Fixed both `--version` and `-V` flags to display proper version information
  - Resolved issue where globally installed buildfab showed "unknown" version
  - Updated test to reflect new behavior - version is compiled into binary at build time

## [0.8.7] - 2025-09-23

### Fixed
- **CLI Parser**: Fixed CLI argument parsing to handle stage/action names when no run command is specified
  - When no subcommand is provided, first argument is now treated as stage or action name
  - Stage names have higher priority than action names when both exist with same name
  - Supports both custom actions and built-in actions (version@check, git@untracked, etc.)
  - Resolves issue where `buildfab test-streaming` was treated as unknown command
  - Maintains backward compatibility with explicit `run` and `action` commands

### Changed
- **Rules Enhancement**: Updated versioning and complete changes rules with changelog date requirements
  - Enhanced `rule-versioning.mdc` with changelog date requirements and commands
  - Updated `rule-complete-changes.mdc` to include proper date management
  - Added specific commands for getting dates from git log and terminal
  - Fixed all historical changelog dates using accurate git log information
  - Ensures all future changelog entries use correct dates from git history or current system

## [0.8.6] - 2025-09-23

### Fixed
- **Ctrl+C Signal Handling**: Fixed critical issue where Ctrl+C did not properly terminate the executor
  - Executor now properly handles context cancellation and terminates promptly without hanging
  - Added comprehensive context cancellation checks throughout DAG execution loops
  - Implemented safe channel operations to prevent panics when context is cancelled
  - Added proper command process termination when context is cancelled
  - Resolves issue where buildfab would hang indefinitely when interrupted with Ctrl+C
- **Command Output Display**: Fixed issue where command output was not being displayed during execution
  - Executor now shows real-time command output during execution instead of just command content
  - Fixed UI integration to use `e.ui.PrintCommandOutput()` instead of step callbacks
  - Both stdout and stderr are properly streamed and displayed in real-time
  - Works correctly in both verbose and non-verbose modes
  - Users now see actual command execution results as they happen

### Changed
- **Command Content Suppression**: Suppressed command content from YAML configuration to keep output clean
  - Command content from configuration files is no longer displayed during normal execution
  - Added `PrintStepName()` method to show only step names instead of full command content
  - Command content is still preserved in error messages for debugging and manual reproduction
  - Provides cleaner output while maintaining debugging capabilities when errors occur
- **UI Integration**: Enhanced CLI to use internal executor with proper UI interface
  - CLI now uses internal executor instead of simple runner for better UI integration
  - All output formatting is now handled through the UI interface for consistency
  - Proper integration between executor and UI ensures consistent output formatting

### Added
- **TERMINATED Status Display**: Added proper status display when execution is interrupted
  - Shows "⚠️ TERMINATED" status instead of misleading "SUCCESS" when Ctrl+C is pressed
  - Added `PrintStageTerminated()` method to UI interface for proper termination display
  - Clear indication to users when execution was interrupted rather than completed successfully
  - Maintains proper timing and summary information even when terminated

## [0.8.5] - 2025-09-23

### Changed
- **Verbose Mode Default**: Made verbose mode the default behavior for all buildfab executions
  - `DefaultRunOptions()` now sets `Verbose: true` by default
  - All CLI commands now show detailed command execution and output by default
  - Provides better visibility into what buildfab is doing during execution
  - Maintains backward compatibility with existing configurations

### Added
- **Quiet Mode Option**: Added `-q, --quiet` flag to disable verbose output when needed
  - New `--quiet` and `-q` flags override verbose mode to enable silence mode
  - Silence mode shows only final results and summary without command details
  - Useful for CI/CD environments or when minimal output is preferred
  - Updated CLI help text to clearly indicate verbose as default and quiet as override option

## [0.8.4] - 2025-09-23

### Fixed
- **Streaming Output Ordering**: Fixed step start and completion messages to appear in declaration order during parallel execution
  - Added `StreamingOutputManager` to control which step's output should be streamed
  - Step start messages (`💻 step-name`) now appear only for the first step in declaration order
  - Step completion messages (`✓ step-name`) now appear in the correct order
  - Streaming output is shown only for the currently active step in declaration order
  - Resolves mixed step messages when running parallel stages with verbose output
- **Mixed Output Elimination**: Fixed mixed output between parallel steps during execution
  - Implemented proper buffering and ordered display logic for parallel step execution
  - Steps now run in parallel for performance but display output sequentially in declaration order
  - Eliminated interleaved output between different steps running simultaneously
  - Each step's output is displayed as a complete block when it becomes the active step
- **CLI Argument Parsing**: Fixed CLI argument parsing issues with flag recognition
  - Removed custom argument parsing logic that was interfering with cobra's built-in parsing
  - Fixed issue where flags like `-c` were being treated as commands instead of flags
  - CLI now properly uses cobra library for all argument parsing without custom interference
  - Resolves command line parsing errors when using flags before subcommands
- **Output System Unification**: Unified CLI and library output systems to eliminate duplication
  - CLI now uses library's UI system (`internal/ui/ui.go`) instead of duplicating output logic
  - Removed custom output functions from CLI (`printHeader`, `printStageHeader`, `printSimpleResult`)
  - All output formatting is now centralized in the library's UI system
  - Ensures consistency between CLI and library output formatting
- **Test Organization**: Moved test files to `tests/` directory and added documentation
  - Created `tests/README.md` with comprehensive test documentation
  - Moved `test-streaming.yml` to `tests/` directory for better organization
  - Added usage examples and expected behavior documentation for streaming output tests

### Fixed
- **Streaming Output Ordering**: Fixed step start and completion messages to appear in declaration order during parallel execution
  - Added `StreamingOutputManager` to control which step's output should be streamed
  - Step start messages (`💻 step-name`) now appear only for the first step in declaration order
  - Step completion messages (`✓ step-name`) now appear in the correct order
  - Streaming output is shown only for the currently active step in declaration order
  - Resolves mixed step messages when running parallel stages with verbose output
- **Test Organization**: Moved test files to `tests/` directory and added documentation
  - Created `tests/README.md` with comprehensive test documentation
  - Moved `test-streaming.yml` to `tests/` directory for better organization
  - Added usage examples and expected behavior documentation for streaming output tests
- **GoReleaser Configuration**: Fixed GitHub repository owner references in GoReleaser configuration
  - Updated `.goreleaser.yml` to use `AlexBurnes` instead of `burnes` for both Scoop and Homebrew tap owners
  - Updated `docs/Deploy.md` to reference correct `AlexBurnes/buildfab-scoop-bucket` repository
  - Resolves 404 errors when GoReleaser tries to check default branches for package repositories
- **Binary Name Consistency**: Fixed all buildtools scripts to reference `buildfab` binary instead of `version` binary
  - Updated `buildtools/create-goreleaser-archives.sh` to use `bin/buildfab --version` for version detection
  - Updated `buildtools/create-goreleaser-backup.sh` to use `buildfab-*` binary names and backup file naming
  - Ensured all binary references are consistent across buildtools directory
- **Packaging Documentation**: Updated all packaging README files to reference buildfab instead of version
  - Fixed `packaging/macos/README.md` to use buildfab CLI commands and repository URLs
  - Fixed `packaging/windows/scoop-bucket/README.md` to use buildfab package name
  - Fixed `packaging/linux/README.md` to use buildfab download URLs and installation paths
  - Updated all installation commands and repository references to use buildfab project
- **Documentation URLs**: Ensured all download URLs point to latest releases
  - Updated README.md version badge from v0.7.4 to v0.8.0
  - Updated specific version download examples from v0.7.0 to v0.8.0
  - Updated packaging/linux/README.md manual installation example to use latest URL pattern
  - Verified all "latest" URLs are correctly pointing to current releases

## [0.8.0] - 2025-09-22

### Added
- **Automated Version Bump Script**: Created `scripts/version-bump-with-file` for comprehensive version management
  - Automatically updates VERSION file when bumping version
  - Updates Windows Scoop configuration (`packaging/windows/scoop-bucket/version.json`) with new version and URLs
  - Updates macOS Homebrew formula (`packaging/macos/version.rb`) with new URLs
  - Provides clear next steps for git operations
  - Ensures version consistency across all packaging files

### Changed
- **Version Management Rules**: Updated versioning rules to use automated version bump script
  - `scripts/version-bump-with-file` is now the recommended method for version bumps
  - Updated complete changes shortcut rule to use automated version bump
  - Added packaging file update requirements to versioning rules
  - Enhanced error handling for packaging file updates

## [0.7.5] - 2025-09-22

### Fixed
- **Installer Scripts**: Fixed installer scripts to use correct binary name and repository
  - Fixed Linux installer script (`packaging/linux/install.sh`) to look for `buildfab` binary instead of `version`
  - Fixed installer template (`packaging/linux/installer-template.sh`) to download from `burnes/buildfab` repository
  - Updated Windows Scoop configuration (`packaging/windows/scoop-bucket/version.json`) to use current version v0.7.5
  - Updated macOS Homebrew formula (`packaging/macos/version.rb`) to use correct repository and binary name
  - All installer scripts now correctly download and install the `buildfab` binary from the correct repository

## [0.7.4] - 2025-09-22

### Added
- **Buildfab Migration**: Complete migration from bash scripts to buildfab actions
  - Added tool check actions: `check-conan`, `check-cmake`, `check-goreleaser`
  - Added dependency installation action: `install-conan-deps` with golang package creation
  - Added build actions: `configure-cmake`, `build-binaries`, `build-all-platforms`
  - Added installer creation action: `create-installers`
  - Added GoReleaser actions: `goreleaser-dry-run`, `goreleaser-release`
  - Enhanced build process with proper CMake preset support and fallback configuration
  - Added automatic GoReleaser installation and PATH setup for Go environment

### Changed
- **Project Configuration**: Reorganized `.project.yml` with stages first, actions at end
  - Improved readability with logical grouping of actions by category
  - Enhanced dependency management with proper `require` relationships
  - Updated build and release stages to use new buildfab actions
  - Removed legacy bash script references from project configuration

### Removed
- **Legacy Build Scripts**: Removed old bash scripts that are no longer needed
  - Removed `buildtools/build-conan.sh` (448 lines)
  - Removed `buildtools/build-and-package.sh` (350 lines)
  - Removed `buildtools/build-goreleaser.sh` (232 lines)
  - Removed legacy build actions from project configuration
  - Verified CI/CD pipelines don't use removed scripts to prevent breakage

### Documentation
- **README Updates**: Enhanced development section with buildfab installation instructions
  - Added buildfab installation commands for Linux, macOS, and Windows
  - Updated build section to recommend buildfab-based workflow
  - Added platform-specific installation instructions for both amd64 and arm64
  - Improved developer workflow documentation

## [0.7.3] - 2025-09-22

### Added
- **Silence Mode Enhancement**: Added running step indicators for improved user experience
  - Real-time feedback showing `○ step-name running...` when steps start executing
  - Clean line replacement using carriage return (`\r`) for professional output
  - Running indicators are replaced with final results when steps complete
  - Only active in silence mode - verbose mode maintains existing detailed behavior
  - Perfect balance between clean output and progress visibility
  - Users can now see exactly which step is currently executing instead of wondering if executor is stuck

### Fixed
- **Test Suite Issues**: Fixed all test failures and build issues
  - Fixed examples package build failure by resolving duplicate main function declarations
  - Updated RunAction method to properly check action registry for built-in actions like version@check
  - Modified CLI functions to return errors instead of calling os.Exit(1) in test mode
  - All tests now passing with 100% success rate across all packages
  - Built-in actions now work correctly in both CLI and library usage

### Documentation
- **README Enhancement**: Added comprehensive installation and git hook setup instructions
  - Added detailed installation instructions for Linux, Windows, and macOS using install scripts and Scoop
  - Added git hook setup guide with step-by-step instructions for automated project validation
  - Added version utility installation instructions for development and testing requirements
  - Added project configuration examples showing how to set up `.project.yml` for git hooks
  - Added reference to version-go project for complete version utility documentation
  - Reorganized installation sections to avoid duplication and improve user experience

## [0.7.2] - 2025-09-22

### Fixed
- **Command Alignment Issues**: Fixed multi-line command indentation and duplicate output problems
  - Fixed multi-line command indentation to properly preserve relative indentation structure with 6-space base indentation
  - Commands now maintain their original YAML indentation structure while being properly aligned with "to check run:" prefix
  - Eliminated duplicate "FAILED - stage" messages by removing redundant printSimpleResult calls from CLI
  - Clean single result message per stage execution with proper summary statistics
  - Enhanced error message formatting to use "failed, to check run:" instead of "command failed: exit status 1"
  - Improved skipped step messages to show specific dependency failures (e.g., "skipped (dependency failed: step-name)")

### Changed
- **API Simplification**: Enhanced SimpleRunner for easier consumption without callback complexity
  - Simplified command extraction logic to preserve original YAML indentation structure
  - Updated CLI to use SimpleRunner exclusively, eliminating callback complexity for end users
  - Maintained advanced callback API for internal use while providing simple public interface

## [0.7.1] - 2025-09-22

### Added
- **v0.5.0 Style Output Implementation**: Successfully implemented beautiful v0.5.0 style output formatting
  - Added proper header with project info and version display (🚀 buildfab v0.7.1)
  - Added stage header with clean formatting (▶️ Running stage: pre-push)
  - Added step execution display with proper icons and indentation (💻 for commands, ✓/✗ for results)
  - Added footer summary with statistics and status (💥 FAILED/🎉 SUCCESS with duration and counts)
  - Implemented proper ANSI color codes for green ✓, red ✗, gray →, etc.
  - Added consistent spacing and professional formatting throughout
  - Both normal and verbose modes working perfectly with beautiful output

### Fixed
- **Summary Counting Issue**: Fixed step result collection and summary statistics
  - Fixed summary to show correct count of successful steps (was showing 0 instead of 2)
  - Implemented proper result collection in step callbacks to track actual step results
  - Added deduplication logic to prevent duplicate results in summary
  - Summary now accurately reflects executed steps: ✓ ok 2, ✗ error 1, → skipped 1
- **Duplicate Step Display**: Eliminated duplicate version-module step display issue
  - Added step display deduplication logic to prevent showing same step multiple times
  - Modified OnStepError to not display anything (OnStepComplete handles all display)
  - Prevented duplicate display when both OnStepComplete and OnStepError are called
  - Each step now appears only once in the output with correct status
- **Skipped Steps Visibility**: Implemented proper skipped step display and dependency resolution
  - Added getSkippedSteps() function to analyze stage configuration and executed results
  - Added manual step callback invocation for skipped steps to ensure they appear in output
  - Fixed run-tests to show as → skipped (dependency failed) when version-module fails
  - Added proper dependency analysis to identify steps that should be skipped
  - Skipped steps now appear in both normal and verbose output with correct status
- **CLI Flag Parsing**: Resolved issue where -v pre-push was not working correctly
  - Fixed argument parsing logic to handle flags followed by stage names
  - Added logic to detect when first argument is flag and second argument is stage name
  - Added automatic "run" command insertion for flag + stage name combinations
  - All command variations now work: pre-push, -v pre-push, run pre-push, -v run pre-push
  - Maintained intuitive behavior where stage names can be used directly without explicit run command

### Changed
- **Library API Integration**: Successfully integrated modern buildfab library while maintaining beautiful output
  - Replaced internal package usage with pkg/buildfab library API
  - Implemented CLIStepCallback with v0.5.0 style formatting using library StepCallback interface
  - Added proper result collection and summary generation using library types
  - Maintained all beautiful output formatting while using modern library architecture
  - All functions now use buildfab.LoadConfig(), buildfab.NewRunner(), etc.

## [0.7.0] - 2025-09-22

### Added
- **Step Callback System**: Added comprehensive step-by-step progress reporting for buildfab library
  - Added `StepCallback` interface with `OnStepStart`, `OnStepComplete`, `OnStepOutput`, and `OnStepError` methods
  - Added `StepStatus` types (Pending, Running, OK, Warn, Error, Skipped) for detailed status reporting
  - Added `StepCallback` field to `RunOptions` for optional callback support
  - Integrated step callbacks into all execution methods (`RunStage`, `RunAction`, `RunStageStep`)
  - Added step callback support to both library API and internal executor
  - Added comprehensive test coverage for step callback functionality
  - Added example implementations and usage patterns in `examples/step_callbacks_example.go`
  - Step callbacks provide real-time visibility into individual step execution progress
  - Backward compatible - callbacks are optional and default behavior unchanged
  - Perfect for CLI tools, CI/CD systems, and applications needing step-by-step progress reporting

### Changed
- **Version Check Script**: Updated `scripts/check-version-status` to use `scripts/version check-greatest` functionality
  - Now properly detects when VERSION file version is below the greatest git tag
  - Provides clear error messages and suggestions for version bumping
  - Uses the project's own version utility instead of external dependencies
  - Improved version validation workflow for development and release processes

### Documentation
- **Library API Updates**: Enhanced library documentation with step callback examples
  - Added step callback interface documentation to `docs/Library.md`
  - Added comprehensive usage examples for step callbacks
  - Added examples for different callback patterns (verbose, silent, custom)
  - Updated API reference to include new `StepCallback` field in `RunOptions`

## [0.6.0] - 2025-09-21

### Added
- **Built-in Action Support in Public API**: Added comprehensive built-in action support to the buildfab library
  - Added `ActionRegistry` and `ActionRunner` interfaces for extensible action system
  - Implemented `DefaultActionRegistry` with all built-in actions (git@untracked, git@uncommitted, git@modified, version@check, version@check-greatest)
  - Added `NewRunnerWithRegistry()` function for custom action registry support
  - Added `ListBuiltInActions()` method to list available built-in actions
  - Updated `Runner` to support both `run:` and `uses:` fields in action configuration
  - Added proper error handling and status reporting for built-in actions
  - Added comprehensive test coverage for built-in action functionality
  - Added configuration loading support with `LoadConfig()` and `LoadConfigFromBytes()` functions
  - Built-in actions now work seamlessly in both CLI and library usage

### Documentation
- **README Updates**: Added comprehensive built-in action documentation
  - Added "Built-in Actions" section with complete action reference
  - Added usage examples for both YAML configuration and CLI usage
  - Added library integration examples showing built-in action support
  - Updated feature list to highlight built-in action capabilities

## [0.5.1] - 2025-09-21

### Fixed
- **Library API Implementation**: Fixed `buildfab.Runner.RunStage()`, `RunAction()`, and `RunStageStep()` methods
  - Replaced placeholder "not yet implemented" errors with working implementations
  - Added proper sequential execution for stages with error handling
  - Implemented custom action execution with shell command support
  - Added support for error policies (stop/warn) in stage execution
  - Fixed type issues with `RunOptions.Output` and `RunOptions.ErrorOutput` fields
  - Updated all related unit tests to reflect working implementation
  - Library now fully functional for pre-push integration and other use cases

- **RunCLI Function**: Implemented `buildfab.RunCLI()` function for programmatic CLI execution
  - Added argument parsing for common CLI commands (run, action)
  - Added configuration loading support with config path detection
  - Added proper error handling for invalid arguments and commands
  - Updated tests to reflect new implementation behavior
  - All library methods now fully implemented with no placeholder messages

### Changed
- **RunOptions Type Safety**: Changed `Output` and `ErrorOutput` fields from `interface{}` to `io.Writer`
  - Improves type safety and prevents runtime errors
  - Ensures proper interface compliance for output handling

## [0.5.0] - 2025-09-21

### Added
- **CLI Test Suite**: Added comprehensive test coverage for cmd/buildfab package (68.8% coverage)
  - `cmd/buildfab/main_test.go` - Complete CLI command testing
  - Version detection testing with VERSION file handling
  - Command execution testing for all CLI commands
  - Error handling and output validation testing
  - Flag validation and command structure testing

### Fixed
- **DAG Executor Tests**: Fixed channel panic issues in DAG execution with proper synchronization
- **UI Test Formatting**: Updated test expectations to match current output formatting
- **Test Coverage**: Improved overall project test coverage from 58.6% to 72.5%

### Changed
- **Test Infrastructure**: Expanded from 9 to 10 test files with comprehensive CLI testing
- **Coverage Reporting**: Updated coverage metrics to reflect CLI test improvements

## [0.4.0] - 2025-09-21

### Added
- **Comprehensive Test Suite**: Implemented complete test coverage with 75.3% overall coverage across all packages
- **Test Infrastructure**: Created 9 test files covering unit tests, integration tests, and end-to-end scenarios
  - `pkg/buildfab/types_test.go` - Tests for core types and status enums
  - `pkg/buildfab/errors_test.go` - Tests for custom error types
  - `pkg/buildfab/buildfab_test.go` - Comprehensive tests for main API
  - `internal/config/config_test.go` - YAML parsing and validation tests
  - `internal/actions/registry_test.go` - Built-in action tests
  - `internal/version/version_test.go` - Version detection tests
  - `internal/ui/ui_test.go` - User interface output tests
  - `internal/executor/executor_test.go` - DAG execution tests
  - `integration_test.go` - End-to-end integration tests
- **Coverage Reporting**: Generated detailed coverage reports (coverage.out, coverage.html) with function-level analysis
- **Test Organization**: Clear separation by package with comprehensive error handling and edge case testing
- **Mock Objects**: Custom mock implementations for UI and external dependencies
- **Test Utilities**: Helper functions for common test scenarios and configuration creation

### Fixed
- **Version Validation**: Fixed version format validation to require major.minor.patch format (e.g., v1.2.3)
- **Test Compilation**: Resolved all compilation errors and unused variable issues in test files
- **Integration Test Issues**: Fixed variable naming conflicts and unused variable warnings

### Changed
- **Test Coverage**: Achieved 100% coverage on core API functionality (pkg/buildfab)
- **Error Testing**: Comprehensive error condition coverage across all packages
- **Test Structure**: Organized tests by package with clear separation of concerns

## [v0.3.0] - 2025-09-21

### Added
- **CLI Help Improvements**: Fixed help usage to show `buildfab [flags] [command]` instead of duplicate usage lines
- **Default Run Behavior**: Added default command behavior where first argument is treated as stage name for run command
  - `buildfab pre-push` is now equivalent to `buildfab run pre-push`
- **List Stages Command**: Added `list-stages` command to list defined stages in project configuration
- **Enhanced List Actions Command**: Modified `list-actions` command to show both defined actions in project configuration and built-in actions
- **List Steps Command**: Added `list-steps <stage>` command to list steps for a specific stage defined in project configuration

### Changed
- **CLI Command Structure**: Improved CLI command organization with better help text and usage examples
- **Action Listing**: Enhanced action listing to show both custom and built-in actions with proper descriptions

## [v0.2.0] - 2025-09-21

### Added
- **Complete Changes Shortcut**: Added rule for "complete changes" command that automatically executes full release workflow including version bump, documentation updates, git operations, and push
- **Semantic Commit Formatting**: Extended git commit format to require "and write change description on new line" for better semantic formatting and consistency

## [v0.1.2] - 2025-09-21

### Fixed
- **DAG Executor Streaming**: Fixed critical bug where DAG executor was not properly implementing streaming output
  - Removed wave-based execution with `wg.Wait()` that prevented true streaming
  - Implemented continuous execution where steps start as soon as dependencies are satisfied
  - Changed display logic to show results immediately when they complete, in declaration order
  - Now properly supports true parallel execution with streaming output as specified in pre-push project

## [v0.1.1] - 2025-09-21

### Fixed
- **Dependency Error Messages**: Enhanced dependency failure messages to show specific dependency names instead of generic "dependency failed"
- **Run-tests Execution Order**: Fixed run-tests to execute after version-module step by removing release-only condition
- **Command Error Formatting**: Improved version-module command error messages to place commands on new lines for better readability
- **Version Display**: Fixed duplicate 'v' prefix in version display (was showing 'vv0.1.0', now shows 'v0.1.0')
- **Summary Colors**: Fixed summary color display - counts of 0 now show in gray, counts >0 show in appropriate colors
- **Output Alignment**: Fixed multi-line message alignment in step status display
- **Git-modified Message**: Simplified git-modified action message to show concise message with git status command
- **Multi-line Indentation**: Fixed indentation for subsequent lines in multi-line messages to align properly with message content (improved to use 25 spaces for better emoji alignment)
- **Icon Alignment**: Replaced emoji icons with monospace symbols (✓, !, ✗, →, ○, ?) to ensure consistent alignment across all status indicators
- **Simplified Output Format**: Removed unnecessary alignment between command names and descriptions for cleaner output
- **Colored Icons**: Added color to status icons for better visual distinction and readability
- **Reproduction Instructions Alignment**: Fixed multi-line reproduction instructions to preserve original indentation structure without adding extra indentation
- **Command Error Message Indentation**: Removed extra indentation from custom action error messages to preserve original script indentation structure
- **Summary Number Alignment**: Improved summary formatting with right-aligned numbers and consistent spacing for better readability (removed unnecessary colon)
- **Workflow Rules Update**: Updated version management rules to use simple commit messages and always push with --tags

## [v0.1.0] - 2025-09-21

### Added
- **Release Preparation**: Updated memory bank documents, README, and CHANGELOG for v0.1.0 release
- **Build System Validation**: Successfully tested all build scripts and cross-platform compilation
- **Documentation Updates**: Enhanced README with badges and improved installation instructions

### Fixed
- **Version Library Integration**: Fixed to use official AlexBurnes/version-go v0.8.22 from GitHub
- **Compilation Issues**: Resolved all unused variable errors and compilation warnings
- **Action Command**: Enhanced to support built-in actions without requiring configuration file
- **Go Version**: Updated to Go 1.23.1 for latest features and performance improvements

### Added
- **Core Library Implementation**: Complete library API with Config, Action, Stage, Step, and Result types
- **YAML Configuration System**: Full parsing, validation, and variable interpolation with `${{ }}` syntax
- **DAG Execution Engine**: Parallel execution with dependency management, cycle detection, and streaming output
- **Built-in Actions**: Git checks (untracked, uncommitted, modified) and version validation actions
- **Version Library Integration**: Full integration with AlexBurnes/version-go v0.8.22 providing `${{version.version}}` variables
- **CLI Interface**: Complete cobra-based CLI with run, action, list-actions, and validate commands
- **UI System**: Colorized output with status indicators, progress reporting, and error handling
- **Variable System**: Git and version variable detection with interpolation support
- **Project Structure**: Created complete Go project structure with cmd/, pkg/, internal/ directories
- **Memory Bank System**: Comprehensive memory bank files for project tracking and documentation
- **Documentation Framework**: Complete documentation following naming conventions and standards
- **Build Infrastructure**: CMake/Conan/GoReleaser build system configuration
- **Project Specification**: Comprehensive technical specification document
- **Library Documentation**: Complete API reference with examples and usage patterns
- **Developer Workflow**: Detailed development setup and contribution guidelines
- **Build Documentation**: Build system, packaging, and release process documentation

### Changed
- **Variable Interpolation**: Replaced `${{tag}}` with `${{version.version}}` using version-go library
- **Memory Bank Updates**: Updated activeContext.md and progress.md to reflect implementation completion
- **Module Path**: Updated go.mod to use github.com/AlexBurnes/buildfab module path
- **Version Integration**: Integrated external version-go library v0.8.22 for comprehensive version support
- **Action Execution**: Built-in actions can now be executed directly without configuration file
- **Go Version**: Updated from Go 1.22 to Go 1.23.1 for latest features and performance

### Technical Details
- **Library API**: Complete implementation in pkg/buildfab with Runner, Config, and execution types
- **Configuration Loading**: Full YAML parsing with validation and error reporting in internal/config
- **Version Detection**: Comprehensive version information using AlexBurnes/version-go v0.8.22 library
- **Action Registry**: Extensible system for built-in actions with consistent interface
- **DAG Execution**: Streaming output that respects declaration order while enabling parallel execution
- **CLI Commands**: Complete command set with proper argument parsing and error handling
- **UI Components**: Colorized output with emoji indicators and progress reporting
- **Deploy Documentation**: CI/CD pipeline and deployment automation documentation

### Documentation
- **README.md**: Main project documentation with installation and usage instructions
- **docs/Project-specification.md**: Complete technical specification for buildfab
- **docs/Library.md**: Comprehensive API reference with examples
- **docs/Developer-workflow.md**: Development setup and contribution guidelines
- **docs/Build.md**: Build system and packaging documentation
- **docs/Deploy.md**: CI/CD pipeline and deployment documentation
- **Memory Bank Files**: projectbrief.md, productContext.md, activeContext.md, systemPatterns.md, techContext.md, progress.md

### Changed
- **Project Focus**: Shifted from pre-push to buildfab as the main project
- **Architecture Design**: Library-first approach with CLI as thin wrapper
- **Documentation Structure**: Adopted naming conventions (First-word-second-word.md)
- **Build System**: Reused existing CMake/Conan/GoReleaser infrastructure

## [v0.1.0] - 2025-09-21

### Added
- **Initial Project Setup**: Complete project structure and documentation
- **Go Module**: Initial go.mod with required dependencies
- **Version Management**: VERSION file with initial version v0.1.0
- **Memory Bank Integration**: MCP server integration for project state tracking
- **Comprehensive Documentation**: All required documentation files created
- **Build Configuration**: Updated build scripts and configuration for buildfab project
