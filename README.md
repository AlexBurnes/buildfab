# buildfab

A Go-based CLI utility and library for executing project automation stages and actions defined in YAML configuration files. buildfab provides a flexible framework for running complex, dependency-aware automation workflows with parallel execution capabilities.

[![Go Version](https://img.shields.io/badge/go-1.23.1-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.7.1-orange.svg)](https://github.com/burnes/buildfab/releases)

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

#### Linux
```bash
# Download and install
curl -sSL https://github.com/burnes/buildfab/releases/latest/download/install.sh | bash

# Or download specific version
curl -sSL https://github.com/burnes/buildfab/releases/download/v0.1.0/install.sh | bash
```

#### Windows (Scoop)
```powershell
scoop bucket add buildfab https://github.com/burnes/buildfab-scoop-bucket
scoop install buildfab
```

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

```go
package main

import (
    "context"
    "github.com/burnes/buildfab"
)

func main() {
    ctx := context.Background()
    
    opts := &buildfab.RunOptions{
        ConfigPath: ".project.yml",
        Verbose:    true,
    }
    
    err := buildfab.RunStage(ctx, "pre-push", opts)
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

### Build
```bash
# Build with CMake/Conan
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