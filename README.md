# buildfab

A Go-based CLI utility and library for executing project automation stages and actions defined in YAML configuration files. buildfab provides a flexible framework for running complex, dependency-aware automation workflows with parallel execution capabilities.

## Features

- **YAML-driven configuration**: Define stages and actions in `project.yml` files
- **DAG-based execution**: Parallel execution with explicit dependencies and cycle detection
- **Built-in action registry**: Extensible system for common automation tasks
- **Custom action support**: Execute shell commands and external tools with variable interpolation
- **Library API**: Embeddable Go library for integration with other tools
- **Cross-platform compatibility**: Linux, Windows, macOS (amd64/arm64)

## Quick Start

### Installation

#### Linux
```bash
# Download and install
curl -sSL https://github.com/burnes/buildfab/releases/latest/download/install.sh | bash
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
- Go 1.22+
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

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

See [Developer Workflow](docs/Developer-workflow.md) for contribution guidelines.