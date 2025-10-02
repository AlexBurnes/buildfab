# System Patterns: buildfab

## System Architecture
buildfab follows a layered architecture with clear separation of concerns:

```
┌─────────────────┐
│   CLI Layer     │  ← cmd/buildfab/main.go
├─────────────────┤
│   Library API   │  ← pkg/buildfab/ (public interface)
│   SimpleRunner  │  ← Uses OrderedStepCallback for output management
├─────────────────┤
│  Core Engine    │  ← internal/executor/ (DAG execution)
├─────────────────┤
│  Output Manager │  ← OrderedOutputManager (queue-based output ordering)
├─────────────────┤
│  Configuration  │  ← internal/config/ (YAML parsing)
├─────────────────┤
│  Action System  │  ← internal/actions/ (built-in actions)
├─────────────────┤
│  Variable System│  ← internal/variables/ (interpolation)
└─────────────────┘
```

## Key Technical Decisions
- **Library-first design**: Core functionality in pkg/buildfab for embedding
- **DAG execution engine**: Parallel processing with dependency resolution
- **Queue-based output management**: OrderedOutputManager ensures sequential display of parallel execution
- **YAML configuration**: Human-readable format with validation
- **Built-in action registry**: Extensible system for common operations
- **Variable interpolation**: GitHub-style `${{ }}` syntax for dynamic values
- **Error policy system**: Configurable stop/warn behavior per step
- **Debug logging system**: Comprehensive debug output with -d|--debug flag for complex changes

## Design Patterns in Use
- **Command Pattern**: Actions implement common interface for execution
- **Strategy Pattern**: Different execution strategies for built-in vs custom actions
- **Observer Pattern**: Progress reporting and status updates
- **Builder Pattern**: Configuration and options construction
- **Factory Pattern**: Action instantiation and registration
- **Dependency Injection**: Context and options passed to components
- **Queue Pattern**: OrderedOutputManager manages step output in sequential order
- **Callback Pattern**: OrderedStepCallback delegates output to OrderedOutputManager
- **Debug Pattern**: Comprehensive debug logging for complex change implementation
- **Mock Pattern**: Test doubles for external dependencies and UI components
- **Test Builder Pattern**: Helper functions for creating test configurations

## Component Relationships
- **CLI → Library API**: CLI is thin wrapper around library functions
- **Library API → Executor**: Main API delegates to DAG execution engine
- **Executor → Actions**: Executor manages action execution and dependencies
- **Actions → Variables**: Actions use variable system for interpolation
- **Config → All**: Configuration drives all component behavior
- **UI → All**: UI components provide user feedback and error reporting
- **OrderedOutputManager → UI**: Queue-based output management ensures sequential display
- **OrderedStepCallback → OrderedOutputManager**: Callback delegates all output to queue manager
- **Debug System → All**: Comprehensive debug logging throughout execution pipeline

## Critical Implementation Paths
1. **YAML Parsing**: project.yml → internal model → validation
2. **DAG Construction**: Actions → dependencies → cycle detection → execution plan
3. **Variable Resolution**: `${{ }}` → context → interpolation → action execution
4. **Parallel Execution**: DAG → wave scheduling → concurrent execution → result aggregation
5. **Queue-Based Output**: OrderedOutputManager → step queue → sequential display → UI
6. **Error Handling**: Action failure → policy check → continue/stop → reporting
7. **Debug Logging**: Debug flag → comprehensive logging → queue state → decision tracing
8. **Library Integration**: pre-push → buildfab API → stage execution → result handling

## Data Flow
```
project.yml → Config Parser → DAG Builder → Executor → Actions → Results → UI
     ↓              ↓            ↓           ↓         ↓        ↓
  Validation → Dependency → Wave → Action → Variable → Status
              Resolution  Planning Execution Interpolation Reporting
     ↓              ↓            ↓           ↓         ↓        ↓
  Queue-Based → OrderedOutput → Step → Debug → Sequential
  Output Mgmt    Manager       Queue  Logging  Display
```

## Testing Architecture
```
Test Suite (75.3% Coverage)
├── Unit Tests (pkg/buildfab: 100%)
├── Integration Tests (internal/config: 87.3%)
├── Action Tests (internal/actions: 50.5%)
├── Version Tests (internal/version: 71.6%)
├── UI Tests (internal/ui: 69.4%)
├── Executor Tests (internal/executor: 0% - blocked)
└── End-to-End Tests (integration_test.go)
```

## Output Management Patterns

### OrderedOutputManager Architecture
The OrderedOutputManager implements a queue-based approach to ensure sequential output display from parallel execution:

```
┌─────────────────┐
│   Executor      │  ← Runs steps in parallel for performance
├─────────────────┤
│   StepCallback  │  ← OrderedStepCallback delegates to manager
├─────────────────┤
│   OutputManager │  ← OrderedOutputManager manages display order
├─────────────────┤
│   UI Display    │  ← Shows output in declaration order
└─────────────────┘
```

### Key Output Management Patterns
- **Queue-based ordering**: Steps run in parallel but display output sequentially in declaration order
- **Buffered output**: Output is buffered until the step becomes active for display
- **Completion tracking**: Tracks step completion and ensures proper ordering of success messages
- **Duplicate prevention**: Output is only shown once, either during execution or completion
- **Context awareness**: Handles both verbose and silence modes with appropriate display logic

### Output Flow
1. **Step Start**: `OnStepStart()` → `canShowStepStart()` → `showStepStart()`
2. **Step Output**: `OnStepOutput()` → Buffer output (don't display immediately)
3. **Step Complete**: `OnStepComplete()` → `checkAndShowCompletedSteps()` → `showStepCompletion()`
4. **Next Step**: `checkAndShowNextStep()` → Show next available step

## Test Patterns
- **Table-driven tests**: Comprehensive test cases with expected inputs/outputs
- **Mock objects**: UI and external dependency mocking for isolated testing
- **Error testing**: Comprehensive error condition and edge case coverage
- **Integration testing**: Cross-package functionality validation
- **Coverage analysis**: Function-level coverage reporting and analysis

## Matrix Feature Implementation (COMPLETED)
- **Matrix Feature Implementation (COMPLETED)**: Successfully implemented comprehensive matrix builds for parallel execution across multiple configurations
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
- **Container Feature Implementation (CLARIFIED)**: Docker and Podman support for isolated execution environments with comprehensive requirements clarification
  - **Container Configuration Schema**: Enhanced YAML schema with CPU/memory limits, artifact collection, environment file support, and resource management
  - **Engine Support**: Docker (primary) and Podman (secondary) with automatic detection and fallback, Linux containers initially
  - **Image Management**: Support for existing images and build-from-Dockerfile with automatic pulling and container cleanup (--rm)
  - **Mount Management**: Bind mounts with host directory validation, read-only/read-write permissions, user-controlled configuration
  - **Resource Limits**: CPU limits (--cpus, --cpuset-cpus) and memory limits (human-readable format: 4G, 500M) with user configuration
  - **Artifact Collection**: Manual configuration with pattern support for files and directories, permission handling via uid/gid
  - **Environment Integration**: Explicit environment variable configuration, env_file support, no default host environment pass-through
  - **Buildfab Integration**: Runtime installation of current buildfab binary only for run_stage/run_action options, configuration copying only when needed, run_stage/run_action/run command support
  - **Error Handling**: Standard action error reporting with verbosity levels, full Docker command display with -vv, engine-managed cleanup
  - **Matrix Integration**: Independent features using variables for integration, no special integration code required
  - **Implementation Approach**: 4-phase incremental development (basic execution, mounts/resources, image building, testing/documentation)
  - **Testing Strategy**: Container image building, resource limiting validation, mount testing, artifact collection verification, matrix integration testing, Podman as primary testing engine (no superuser access required)
  - **Matrix Integration**: Matrix testing examples with Alpine, Ubuntu, and Debian containers for platform detection testing
  - **Documentation**: User guides with examples, API reference, troubleshooting with manual container command reproduction, matrix container examples
- **Caching Feature Implementation**: Comprehensive caching system for build optimization
  - Cache configuration support for ccache, Conan, vcpkg, Go modules, npm, and pip with proper isolation
  - Cache mount management with proper isolation, cleanup, and cross-platform compatibility
  - Cache key generation and validation for different cache types with ABI-aware partitioning
  - Cache statistics and monitoring capabilities with hit rate analysis and performance metrics
  - Cache cleanup and maintenance utilities with automatic old entry removal and corruption detection
  - Cache integration with matrix and container features for optimal build performance
  - CLI support for cache management and statistics with comprehensive monitoring and reporting
- **Platform-specific tool installation**: Extend pre-check and pre-install tools for other platforms with conditional execution using `when` conditions
  - Add platform-specific variants for tool installation actions (Windows, macOS, Linux) with appropriate `when` conditions using `${{ platform }}` variable
  - Ensure pre-check and pre-install stages work correctly across all supported platforms (linux/amd64, linux/arm64, windows/amd64, windows/arm64, darwin/amd64, darwin/arm64)
  - Use action variants with `when` conditions to execute platform-specific tool installation commands
  - Provide platform-appropriate installation instructions and error messages
  - Verify pre-check and pre-install stages work correctly on all target platforms