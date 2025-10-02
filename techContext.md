# Tech Context: buildfab

## Technologies Used
- **Go 1.22+**: Primary language with modern features and performance
- **YAML v3**: Configuration parsing with gopkg.in/yaml.v3
- **Cobra**: CLI framework for command-line interface (optional)
- **errgroup**: Parallel execution management with golang.org/x/sync/errgroup
- **CMake**: Cross-platform build configuration
- **Conan**: Go toolchain and dependency management
- **GoReleaser**: Automated release and packaging
- **GitHub Actions**: CI/CD pipeline and automation
- **Docker**: Container runtime for isolated execution environments
- **Podman**: Alternative container runtime with Docker compatibility
- **ccache**: C/C++ compiler cache for build optimization
- **vcpkg**: C++ package manager with caching support

## Development Setup
- **Go toolchain**: Managed via Conan with conanfile-golang.py
- **Build system**: CMake + Conan for cross-platform builds
- **Linting**: golangci-lint with comprehensive rule set
- **Testing**: go test with race detection and coverage
- **Formatting**: gofmt and goimports for code style
- **Documentation**: GoDoc comments and markdown documentation

## Technical Constraints
- **CGO disabled**: Static binaries for reproducible builds
- **Cross-platform**: Linux, Windows, macOS (amd64/arm64)
- **Memory efficiency**: Stream processing for large outputs
- **Performance**: Fast startup and efficient parallel execution
- **Security**: Input validation and safe command execution
- **Compatibility**: Maintain existing YAML schema from pre-push

## Dependencies
**Core Dependencies:**
- `gopkg.in/yaml.v3`: YAML configuration parsing
- `golang.org/x/sync/errgroup`: Parallel execution management
- `github.com/spf13/cobra`: CLI framework (optional)

**Development Dependencies:**
- `golangci-lint`: Code linting and quality checks
- `go test`: Testing framework with race detection
- `gofmt`: Code formatting
- `goimports`: Import organization

**Build Dependencies:**
- `conanfile-golang.py`: Go toolchain via Conan
- `CMakeLists.txt`: Cross-platform build configuration
- `.goreleaser.yml`: Release automation
- `buildtools/`: Build and packaging scripts

## Tool Usage Patterns
- **Version management**: VERSION file as single source of truth
- **Changelog**: CHANGELOG.md updated for every change
- **Memory bank**: MCP server integration for project state tracking
- **Documentation**: Comprehensive docs with cross-references
- **Testing**: Unit tests, integration tests, and E2E tests
- **Packaging**: GoReleaser for multi-platform releases
- **CI/CD**: GitHub Actions for automated testing and releases

## Build and Release Process
1. **Development**: Local development with go test and linting
2. **Version bump**: Update VERSION file and CHANGELOG.md
3. **Build**: CMake + Conan for cross-platform compilation
4. **Test**: Automated testing with race detection
5. **Package**: GoReleaser for release artifacts
6. **Deploy**: GitHub Releases with package manager updates

## Platform Support
- **Linux**: tar.gz archives with install.sh script
- **Windows**: Scoop manifest for package manager
- **macOS**: Homebrew formula (future)
- **Cross-platform**: Static binaries for all supported platforms

## Matrix Feature Implementation (COMPLETED)
- **Matrix Feature Implementation**: Successfully implemented comprehensive matrix builds for parallel execution across multiple configurations
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
- **Container Feature Implementation**: Docker and Podman support for isolated execution environments
  - Container configuration in actions with engine selection (docker/podman) and automatic engine detection
  - Image management supporting both existing images and build-from-Dockerfile with automatic tagging
  - Mount management (bind mounts, volume mounts, cache mounts) with proper permissions and isolation
  - Environment variable support, working directory, user context, and network mode configuration
  - Buildfab integration inside containers (run_stage support) with binary availability and configuration mounting
  - Container execution engine with proper lifecycle management, error handling, and cleanup
  - CLI support for container-specific commands and overrides with comprehensive error reporting
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