# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.1.1] - 2025-01-21

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

## [v0.1.0] - 2025-01-21

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

## [v0.1.0] - 2025-01-27

### Added
- **Initial Project Setup**: Complete project structure and documentation
- **Go Module**: Initial go.mod with required dependencies
- **Version Management**: VERSION file with initial version v0.1.0
- **Memory Bank Integration**: MCP server integration for project state tracking
- **Comprehensive Documentation**: All required documentation files created
- **Build Configuration**: Updated build scripts and configuration for buildfab project