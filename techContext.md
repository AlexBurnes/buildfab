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

## Future Platform Enhancements
- **Platform-specific tool installation**: Extend pre-check and pre-install tools for other platforms with conditional execution using `when` conditions
  - Add platform-specific variants for tool installation actions (Windows, macOS, Linux) with appropriate `when` conditions using `${{ platform }}` variable
  - Ensure pre-check and pre-install stages work correctly across all supported platforms (linux/amd64, linux/arm64, windows/amd64, windows/arm64, darwin/amd64, darwin/arm64)
  - Use action variants with `when` conditions to execute platform-specific tool installation commands
  - Provide platform-appropriate installation instructions and error messages
  - Verify pre-check and pre-install stages work correctly on all target platforms