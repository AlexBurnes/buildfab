# Buildfab User Documentation

## Overview

This directory contains user-facing documentation for buildfab - a Go-based CLI and library for executing DAG-based automation workflows with matrix builds, parallel execution, and container support.

**Current Version**: v0.32.0

## Quick Links

- [Installation & Getting Started](../README.md)
- [Project Specification](User/Project-specification.md) - Complete feature reference
- [YAML Syntax Reference](User/YAML-syntax-reference.md) - Configuration syntax
- [Examples](../examples/) - Working configuration examples

## Documentation Index

### Core Documentation

#### [Project Specification](User/Project-specification.md)
Complete specification of buildfab features, syntax, and capabilities.

**Contents**:
- Project configuration
- Actions and stages
- Matrix builds
- Dependencies and execution order
- Conditional execution
- Error handling
- Container support
- Variable interpolation
- Built-in actions

#### [YAML Syntax Reference](User/YAML-syntax-reference.md)
Detailed syntax reference for `.project.yml` configuration files.

**Contents**:
- Project section
- Actions section  
- Stages section
- Matrix configuration
- Dependency syntax
- Conditional expressions
- Variable syntax

### Feature Guides

#### [Features and Examples](User/Features-and-examples.md)
Practical examples of buildfab features.

**Contents**:
- Basic workflow examples
- Matrix build examples
- Container action examples
- Conditional execution examples
- Dependency examples
- Error handling examples

#### [Practical Applications](User/Practical-applications.md)
Real-world use cases and patterns.

**Contents**:
- CI/CD workflows
- Build automation
- Testing workflows
- Deployment automation
- Cross-platform builds

#### [Comparison with Others](User/Comparison-with-others.md)
How buildfab compares to other workflow tools.

**Contents**:
- Comparison with GitHub Actions
- Comparison with GitLab CI
- Comparison with CircleCI
- Comparison with Make
- Feature matrix
- When to use buildfab

#### [Caching](User/Caching.md)
Caching feature documentation (if available).

**Contents**:
- Cache configuration
- Cache strategies
- Cache invalidation
- Performance benefits

## Additional Resources

### Command Line Interface

```bash
# Get help
buildfab --help

# List available stages
buildfab list-stages

# List available actions
buildfab list-actions

# Run a stage
buildfab run <stage-name>

# Run with verbose output
buildfab run <stage-name> -v

# Dry run (show what would execute)
buildfab run <stage-name> --dry-run
```

### Library API

For using buildfab as a Go library in your applications:

- [Library API Documentation](Library.md) - Complete Go API reference

### Build and Deployment

- [Build Documentation](Build.md) - Building buildfab from source
- [Deploy Documentation](Deploy.md) - CI/CD and release process

### Development

For contributing to buildfab:

- [Developer Workflow](Developer-workflow.md) - Development setup and workflow
- [Development Documentation](Devel/) - Implementation details and plans

## Quick Start

### 1. Installation

```bash
# Linux/macOS
curl -sSL https://github.com/AlexBurnes/buildfab/releases/latest/download/install.sh | bash

# Windows (Scoop)
scoop bucket add buildfab https://github.com/AlexBurnes/buildfab
scoop install buildfab
```

### 2. Create Configuration

Create `.project.yml`:

```yaml
project:
  name: my-project

actions:
  - name: build
    run: go build ./...
  
  - name: test
    run: go test ./...

stages:
  ci:
    steps:
      - action: build
      - action: test
        require: [build]
```

### 3. Execute

```bash
buildfab run ci
```

## Common Tasks

### Running Tests

```bash
buildfab run test
```

### Building Release

```bash
buildfab run build --only release
```

### Matrix Builds

```yaml
stages:
  cross-platform:
    steps:
      - action: build
        matrix:
          values:
            platform: [linux, darwin, windows]
            arch: [amd64, arm64]
```

```bash
buildfab run cross-platform
```

### Container Actions

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
```

```bash
buildfab run container-test
```

## Support and Community

- **GitHub**: https://github.com/AlexBurnes/buildfab
- **Issues**: https://github.com/AlexBurnes/buildfab/issues
- **Examples**: [examples/](../examples/)
- **Tests**: [tests/](../tests/)

## See Also

- [Developer Workflow](Developer-workflow.md) - For contributors
- [Library API](Library.md) - For Go developers embedding buildfab
- [Build Documentation](Build.md) - Building from source
- [Development Documentation](Devel/) - Implementation details

## Version Information

Current version: **v0.32.0**

For version history and changelog, see [CHANGELOG.md](../CHANGELOG.md)

