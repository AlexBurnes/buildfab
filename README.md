# buildfab

A Go-based CLI utility and library for executing project automation stages and actions defined in YAML configuration files. buildfab provides a flexible framework for running complex, dependency-aware automation workflows with parallel execution capabilities.

[![Go Version](https://img.shields.io/badge/go-1.23.1-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.10.3-orange.svg)](https://github.com/AlexBurnes/buildfab/releases)

## Features

- **YAML-driven configuration**: Define stages and actions in `project.yml` files
- **DAG-based execution**: Parallel execution with explicit dependencies and cycle detection
- **Built-in action registry**: Extensible system for common automation tasks (git checks, version validation)
- **Custom action support**: Execute shell commands and external tools with variable interpolation
- **Library API**: Embeddable Go library for integration with other tools
- **Version integration**: Full integration with AlexBurnes/version-go for comprehensive version support
- **Variable interpolation**: GitHub-style `${{ }}` syntax for Git and version variables
- **Cross-platform compatibility**: Linux, Windows, macOS (amd64/arm64)

## Quick Start

### Installation

See the [Installation and Git Hook Setup](#installation-and-git-hook-setup) section below for detailed installation instructions.

### Basic Usage

1. Create a `project.yml` file:
```yaml
project:
  name: "my-project"

actions:
  - name: test
    run: |
      go test ./...
  
  - name: lint
    run: |
      golangci-lint run

stages:
  pre-push:
    steps:
      - action: test
      - action: lint
```

2. Run the pre-push stage:
```bash
buildfab run pre-push
```

3. Run individual actions:
```bash
# Check version format
buildfab action version@check

# Check for untracked files
buildfab action git@untracked

# List all available actions
buildfab list-actions
```

## CLI Usage

### Command Line Options

buildfab provides several command-line options to control execution behavior:

#### Output Control
- **`-v, --verbose`**: Enable verbose output (default) - shows detailed command execution and output
- **`-q, --quiet`**: Disable verbose output (silence mode) - shows only final results and summary
- **`-d, --debug`**: Enable debug output for troubleshooting

#### Execution Control
- **`-c, --config`**: Path to configuration file (default: `.project.yml`)
- **`-w, --working-dir`**: Working directory for execution (default: current directory)
- **`--max-parallel`**: Maximum parallel execution (default: CPU count)
- **`--only`**: Only run steps matching these labels
- **`--with-requires`**: Include required dependencies when running single step

#### Environment
- **`--env`**: Export environment variables to actions

### Examples

```bash
# Run with verbose output (default)
buildfab run pre-push

# Run in quiet mode
buildfab run pre-push --quiet
buildfab run pre-push -q

# Run with debug output
buildfab run pre-push --debug

# Run with custom configuration
buildfab run pre-push --config my-project.yml

# Run only specific steps
buildfab run pre-push --only test,lint

# Run with environment variables
buildfab run pre-push --env GO_VERSION=1.23.1 --env BUILD_TARGET=linux
```

## Installation and Git Hook Setup

### Installing buildfab

#### Linux
```bash
# Download and install using the install script
curl -sSL https://github.com/AlexBurnes/buildfab/releases/latest/download/install.sh | bash

# Or download specific version
curl -sSL https://github.com/AlexBurnes/buildfab/releases/download/v0.8.0/install.sh | bash
```

#### Windows (Scoop)
```powershell
# Add the bucket (if not already added)
scoop bucket add buildfab https://github.com/AlexBurnes/buildfab-scoop-bucket

# Install buildfab
scoop install buildfab

# Update buildfab
scoop update buildfab
```

#### macOS
```bash
# Download and install using the install script
curl -sSL https://github.com/AlexBurnes/buildfab/releases/latest/download/install.sh | bash

# Or download specific version
curl -sSL https://github.com/AlexBurnes/buildfab/releases/download/v0.8.0/install.sh | bash
```

### Setting up Git Hooks

Once installed, you can set up buildfab as a git hook for automated project validation:

#### 1. Install as Git Hook
```bash
# Run once to install buildfab as a pre-push hook
buildfab install-hook

# Or manually create the hook
echo '#!/bin/bash
buildfab run pre-push' > .git/hooks/pre-push
chmod +x .git/hooks/pre-push
```

#### 2. Configure Your Project
Create a `.project.yml` file in your project root (see example from this project):

```yaml
project:
  name: "your-project-name"

actions:
  - name: test
    run: |
      go test ./...
  
  - name: lint
    run: |
      golangci-lint run

  - name: version-check
    uses: version@check

  - name: git-untracked
    uses: git@untracked

stages:
  pre-push:
    steps:
      - action: test
      - action: lint
      - action: version-check
      - action: git-untracked
```

#### 3. Test the Setup
```bash
# Test the pre-push stage manually
buildfab run pre-push

# Test individual actions
buildfab action version@check
buildfab action git@untracked
```

### Version Utility for Testing

This project uses the `version` CLI utility for testing and validation. Installation instructions for the `version` utility can be found in the [Build section](#installing-version-utility) above. The `version` utility provides:

- Version format validation
- Version comparison and sorting
- Git tag integration
- CMake build type detection

For complete documentation and usage examples, see the [version-go project README](https://github.com/AlexBurnes/version-go).

## Built-in Actions

buildfab includes a comprehensive set of built-in actions for common automation tasks:

### Git Actions
- **`git@untracked`**: Check for untracked files in the repository
- **`git@uncommitted`**: Check for uncommitted changes
- **`git@modified`**: Check for modified files (warning only)

### Version Actions
- **`version@check`**: Validate version format in VERSION file
- **`version@check-greatest`**: Check if current version is the greatest tag

### Using Built-in Actions

Built-in actions can be used in two ways:

1. **In YAML configuration**:
```yaml
actions:
  - name: git-untracked
    uses: git@untracked

  - name: version-check
    uses: version@check

stages:
  pre-push:
    steps:
      - action: git-untracked
      - action: version-check
```

2. **Directly via CLI**:
```bash
# Run built-in actions directly
buildfab action git@untracked
buildfab action version@check

# List all available built-in actions
buildfab list-actions
```

### Library Integration

Built-in actions are automatically available when using the buildfab library:

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    config, _ := buildfab.LoadConfig(".project.yml")
    runner := buildfab.NewRunner(config, nil)
    
    // Built-in actions work automatically
    err := runner.RunAction(context.Background(), "git-untracked")
    if err != nil {
        // Handle error
    }
}
```

## Configuration

See [Project Specification](docs/Project-specification.md) for complete configuration reference.

## Library Usage

### SimpleRunner (Recommended)

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Load configuration
    cfg, err := buildfab.LoadConfig(".project.yml")
    if err != nil {
        // Handle error
        return
    }
    
    // Create simple run options
    opts := &buildfab.SimpleRunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
    }
    
    // Create simple runner
    runner := buildfab.NewSimpleRunner(cfg, opts)
    
    // Run a stage - all output is handled automatically!
    err = runner.RunStage(ctx, "pre-push")
    if err != nil {
        // Handle error
    }
}
```

### One-liner Usage

```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Simple one-liner
    err := buildfab.RunStageSimple(ctx, ".project.yml", "pre-push", true)
    if err != nil {
        // Handle error
    }
}
```

## Development

### Prerequisites
- Go 1.23.1+
- CMake
- Conan
- buildfab (latest version from GitHub)
- Version utility (for testing and build requirements)

### Installing buildfab

To build this project, you need to install the latest version of buildfab from GitHub:

#### Linux/macOS
```bash
# Download and install to ./scripts/ directory
# For x86_64/amd64 systems:
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-linux-amd64-install.sh | INSTALL_DIR=./scripts sh

# For ARM64 systems:
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-linux-arm64-install.sh | INSTALL_DIR=./scripts sh
```

#### Windows
```powershell
# Download and install to ./scripts/ directory
# For x86_64/amd64 systems:
Invoke-WebRequest -Uri "https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-windows-amd64.zip" -OutFile "buildfab.zip"
Expand-Archive -Path "buildfab.zip" -DestinationPath "./scripts/"
Remove-Item "buildfab.zip"

# For ARM64 systems:
Invoke-WebRequest -Uri "https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-windows-arm64.zip" -OutFile "buildfab.zip"
Expand-Archive -Path "buildfab.zip" -DestinationPath "./scripts/"
Remove-Item "buildfab.zip"
```

### Installing Version Utility

buildfab requires the `version` utility from the [version-go project](https://github.com/AlexBurnes/version-go) for testing and build requirements. Install it into the `./scripts/` directory:

#### Linux/macOS
```bash
# Download and install to ./scripts/ directory
# For x86_64/amd64 systems:
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-amd64-install.sh | INSTALL_DIR=./scripts sh

# For ARM64 systems:
wget -O - https://github.com/AlexBurnes/version-go/releases/latest/download/version-linux-arm64-install.sh | INSTALL_DIR=./scripts sh
```

#### Windows
```powershell
# Download and install to ./scripts/ directory
# For x86_64/amd64 systems:
Invoke-WebRequest -Uri "https://github.com/AlexBurnes/version-go/releases/latest/download/version-windows-amd64.zip" -OutFile "version.zip"
Expand-Archive -Path "version.zip" -DestinationPath "./scripts/"
Remove-Item "version.zip"

# For ARM64 systems:
Invoke-WebRequest -Uri "https://github.com/AlexBurnes/version-go/releases/latest/download/version-windows-arm64.zip" -OutFile "version.zip"
Expand-Archive -Path "version.zip" -DestinationPath "./scripts/"
Remove-Item "version.zip"
```

### Build

This project uses buildfab for its build process. Make sure you have installed buildfab and the version utility as described above.

```bash
# Build using buildfab (recommended)
./scripts/buildfab run build

# Or build with CMake/Conan directly
mkdir build && cd build
cmake ..
cmake --build .

# Or build with Go directly
go build -o buildfab cmd/buildfab/main.go
```

### Test
```bash
go test ./... -race
```

## Documentation

- [Project Specification](docs/Project-specification.md) - Complete technical specification
- [API Reference](docs/Library.md) - Library API documentation
- [Developer Workflow](docs/Developer-workflow.md) - Development setup and workflow
- [Build System](docs/Build.md) - Build and packaging documentation

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Contributing

See [Developer Workflow](docs/Developer-workflow.md) for contribution guidelines.