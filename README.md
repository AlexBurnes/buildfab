# buildfab

> **Universal build orchestration tool** that unifies local development, CI/CD, and container workflows under a single declarative YAML configuration.

[![Go Version](https://img.shields.io/badge/go-1.23.1-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.29.0-orange.svg)](https://github.com/AlexBurnes/buildfab/releases)

## Why buildfab?

Modern projects suffer from **fragmented build logic**: separate bash scripts for local builds, different YAML for CI pipelines, custom Dockerfiles for containers, and platform-specific setup scripts. This fragmentation leads to:

- ❌ **Duplicated logic** across different environments
- ❌ **"Works on my machine"** syndrome
- ❌ **Inconsistent** local vs CI behavior
- ❌ **Hard to maintain** scattered scripts

**buildfab solves this** by providing a **single source of truth** for all your build automation:

```yaml
# .project.yml - one file for everything
project:
  name: "my-project"

stages:
  pre-push:
    steps:
      - action: test

  build:
    steps:
      - action: test
      - action: build

actions:
  - name: test
    run: go test ./...

  - name: build
    run: go build  -o bin/app ./cmd/app

```

This same configuration works:

- ✅ **Locally** - developers run the same commands as CI
- ✅ **In CI** - GitHub Actions/GitLab CI executes buildfab stages
- ✅ **In containers** - test builds in clean environments
- ✅ **In Git hooks** - automated validation before push

## About

buildfab is a **universal build automation utility and library** that replaces complex project-specific scripts with a single declarative YAML configuration — `.project.yml`.

It defines how a project is **built, tested, and packaged** locally, in CI/CD pipelines, and inside containers — **all from the same configuration file**.

buildfab executes build stages and actions through a **dependency graph (DAG)**, supports **matrix builds and variants**, runs jobs in **Docker/Podman containers**, and includes **caching and artifact storage** for reproducible builds.

The same configuration is used by [pre-push](https://github.com/AlexBurnes/pre-push) and CI systems, ensuring **consistent results everywhere**.

### Self-Hosting

The tool is **self-hosting** — buildfab builds itself using its own `.project.yml` configuration, demonstrating the power and flexibility of the approach.

### Real-World Usage

buildfab is actively used in production:

- ✅ **Go projects**: Building buildfab itself locally and in GitHub Actions
- ✅ **C++ modules**: Compiling complex C++ projects on GitLab CI with multi-distro support
- ✅ **Container workflows**: Building applications and creating slim optimized images (30x+ smaller)

## 🌍 Project Vision

**buildfab** was born from a simple frustration — every project had its own fragile build scripts, slightly different for each language, OS, or CI.
Switching from Go to C++ or between Linux and macOS meant remembering new commands, reinstalling tools, and rewriting scripts.
For years, every project maintained its own fragile scripts — for local builds, CI pipelines, and container environments — all drifting apart and hard to support.
buildfab unifies them into a single declarative system where one YAML file defines everything: how to build, test, package, and verify a project — locally, in CI, or inside containers.

As a developer, I didn't want to *think about building* — I wanted to *build by thinking*.

So **buildfab** became not a build system, but a **build orchestrator** — a universal runner that executes whatever your project requires:
Conan for dependencies, CMake for compilation, GoReleaser for publishing, Docker for container builds, or any custom script you still want to keep.
Each project, language, and platform had its own fragile scripts: installing dependencies with Conan, compiling via CMake, packaging with GoReleaser, publishing for Scoop or Homebrew.
Developers had to remember dozens of commands and flags — just to build the same project in a different place.
I wanted the opposite:

- to type `buildfab build` instead of a page-long command;
- to type `make` when I changed code;
- to type `install` when I was ready to run integration tests.

All logic — dependencies, build steps, variants, and environments — should be written **once** in `.project.yml`.
After that, the developer only chooses *what* to do, not *how* to do it.

---

## 🚀 Philosophy

buildfab replaces the chaos of scattered build scripts with a single, declarative view of your project's lifecycle — but it doesn't forbid scripts or tools; isn't about replacing your build tools — it's about giving them order and visibility.
You can still call `bash`, `conan`, `cmake`, or `docker` directly — buildfab simply coordinates them, caches results, and makes every build reproducible across local, CI, and container environments.
It's the **conductor** of your build orchestra: every instrument keeps playing, but now under a single, clear score.
**buildfab** transforms traditional, fragmented build workflows into a clear, reproducible orchestration model.

---

## 💬 In essence

> **buildfab** is a lightweight, self-hosted CI runner and build orchestrator, is not a build system — it's a **build orchestrator**.
> that lets developers focus on code, not commands.
> One YAML file defines how every system builds your project — and even how buildfab builds itself.
> It turns your project's build process into a structured graph of actions that can run locally, in CI, or inside containers — all from the same `.project.yml`.

---

## 🧩 Architecture Overview

**From scripts to orchestration — from chaos to structure**

---

### 🎼 High-level View

```
┌───────────────────────────┐
│      Developer / CI       │
│  (runs "buildfab build")  │
└────────────┬──────────────┘
             │
             ▼
     ┌──────────────┐
     │   buildfab   │
     │ (orchestrator│
     │   & runner)  │
     └──────┬───────┘
            │
 ┌──────────┼───────────┬─────────────────────────┐
 │          │           │                         │
 ▼          ▼           ▼                         ▼
Conan   →  CMake   →  GoReleaser   →  Docker/Podman
deps       compile     package/release   container builds
```

🟢 buildfab provides **structure, visibility, and reproducibility**
for what used to be a jungle of shell scripts and ad-hoc CI commands.

---

### 🧠 Detailed Architecture

```
                            ┌────────────────────────────────────────────┐
                            │               Developer / CI               │
                            │    git push, pre-push, build, test...      │
                            └────────────────────┬───────────────────────┘
                                                 │
                                                 ▼
                                      ┌───────────────────────────┐
                                      │        buildfab CLI       │
                                      │  YAML orchestrator engine │
                                      └─────────────┬─────────────┘
                                                    │
                 ┌──────────────────────────────────┴─────────────────────────────────┐
                 │                               │                                   │
                 ▼                               ▼                                   ▼
        ┌────────────────┐             ┌──────────────────┐               ┌────────────────┐
        │   Stage graph  │             │  Action executor │               │  Matrix runner │
        │  (DAG planner) │             │ (run / uses /    │               │ (variants &    │
        │                │             │  container)      │               │ parallel exec) │
        └────────────────┘             └──────────────────┘               └────────────────┘
                 │                               │                                   │
                 └──────────────┬────────────────┴──────────────┬────────────────────┘
                                │                               │
                                ▼                               ▼
               ┌────────────────────────────┐     ┌────────────────────────────┐
               │     Container engine       │     │     Local environment       │
               │ (Docker / Podman runtime)  │     │ (bash, cmake, conan, etc.) │
               └────────────┬───────────────┘     └────────────┬───────────────┘
                            │                                 │
                            ▼                                 ▼
                ┌────────────────────┐             ┌────────────────────┐
                │ build scripts,     │             │ native commands,   │
                │ cmake, conan, etc. │             │ tests, linters     │
                └────────────────────┘             └────────────────────┘
```

---

### 💬 Core Idea

> buildfab brings **structure to chaos** — a single YAML file defines how your project is built, tested, and released everywhere.
> It's transparent, self-hosted, and human-readable — even for complex containerized builds.
> What used to be hidden inside `set -x` debug output is now clear, reproducible, and beautifully organized.

Where a `bash set -x` dump once looked like chaos,
buildfab now prints clear, colored, and reproducible logs showing exactly what runs — even inside containers.
Developers (and newcomers) can finally see *what happens* during the build, step by step.

---

### 🐳 Example — transparent container build

```bash
./bin/buildfab --config examples/container-build-matrix.yml images-build \
  --matrix.image='alpine:latest' -vv
```

produces:

```
buildfab v0.20.0
▶️  Running stage: images-build
  💻 image-build.alpine:latest
  🐳 Running container: podman run --rm ... alpine:latest sh -c '... buildfab run container-build ...'
  ...
  ✓ test-check executed successfully - in '10s'
🎉 SUCCESS - images-build in 1m 15s
```

and the corresponding YAML:

```yaml
actions:
  - name: image-build
    container:
      engine: podman
      image: { from: ${{ matrix.image }} }
      cpu: 2
      memory: 4G
      network: host
      cache: { conan: ./.cache }
      run_stage: container-build

  - name: test-check
    run: |
      sh examples/test.sh
```

Every executed command, including full container invocations, is shown exactly — so the developer can reproduce any step manually.

---

## 👥 Team

- 🧠 **Alex Burnes** - Team Lead & Architect. Designed the core idea, architecture, and ecosystem — from the first version-go library to buildfab and pre-push. Defines strategy, oversees design, and ensures clarity of the overall engineering vision.
- 🐝 **AI Assistant "Buzzy"** — Architectural support & prompt engineering. Responsible for structuring, refining, and documenting the architecture; assists with system design decisions, documentation, and clarity. Provides the buzz behind the structure.
- 🤖 **AI Agent "Coddy"** — Coder & tester. Implements and validates ideas at lightning speed — remembers every detail, calculates instantly, and ensures the logic works in practice. A quiet genius of precision and memory.

---

## Features

buildfab provides a comprehensive automation framework with powerful features for project automation:

- **YAML-driven configuration** with intuitive syntax and modular organization
- **DAG-based execution** with parallel processing and dependency management
- **Matrix feature** for parallel execution across multiple configurations
- **Matrix on stages** for cross-compiler builds and multi-platform testing
- **Pool-based concurrency control** with global and matrix-specific limits
- **Built-in actions** for common tasks (git checks, version validation)
- **Action variants** for platform-specific execution
- **Conditional execution** with powerful expression language
- **Variable interpolation** with GitHub-style `${{ }}` syntax
- **Include system** for organizing complex configurations
- **Library API** for embedding in other tools
- **Cross-platform support** (Linux, Windows, macOS)

📖 **See [Features and Examples](docs/Features-and-examples.md) for comprehensive documentation with detailed examples and usage patterns.**

## Quick Start

### Installation

See the [Installation](#installation) section below for detailed installation instructions.

### Basic Usage

1. Create a `project.yml` file:
```yaml
project:
  name: "my-project"
  max_parallel: 4  # Global concurrency limit (optional)

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
        if: "os == 'linux'"  # Only run lint on Linux
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
- **`--max-parallel`**: Override global parallel execution limit (default: from config or CPU count)
- **`--only`**: Only run steps matching these labels
- **`--with-requires`**: Include required dependencies when running single step
- **`--dry-run`**: Show what would be executed without running commands

**Note on Parallelism**: The `--max-parallel` CLI flag overrides the `project.max_parallel` configuration. Matrix steps with their own `max_parallel` setting create dedicated pools with effective limit = `min(global, matrix)`.

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

# Preview what would be executed without running commands
buildfab run pre-push --dry-run

# Dry run with quiet mode (shows only summary)
buildfab run pre-push --dry-run --quiet

# Dry run a single action
buildfab action version@check --dry-run
```

## Installation

### Installing buildfab

#### Linux
```bash
# Download and install using the install script
wget -O - https://github.com/AlexBurnes/buildfab/releases/latest/download/buildfab-linux-amd64-install.sh | sh
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

#### macOS (Homebrew)
```bash
# Add the tap (if not already added)
brew tap AlexBurnes/buildfab https://github.com/AlexBurnes/homebrew-buildfab

# Install buildfab
brew install buildfab

# Update buildfab
brew upgrade buildfab
```

### Git Hook Setup

buildfab can execute `pre-push` stages, which are used by the [pre-push utility](https://github.com/AlexBurnes/pre-push) — a Git hook automation tool based on the buildfab library.

**Note**: buildfab itself does **not** install Git hooks. To set up automated pre-push validation, use the [pre-push utility](https://github.com/AlexBurnes/pre-push):

#### 1. Install pre-push utility
```bash
# Linux/macOS
curl -sSL https://github.com/AlexBurnes/pre-push/releases/latest/download/install.sh | bash

# Windows (Scoop)
scoop bucket add pre-push https://github.com/AlexBurnes/pre-push-scoop-bucket
scoop install pre-push
```

#### 2. Install Git Hook
```bash
# Run once to install the pre-push hook
pre-push install

# This creates .git/hooks/pre-push that runs: buildfab run pre-push
```

#### 3. Configure Your Project
Create a `.project.yml` file in your project root with a `pre-push` stage:

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

#### 4. Test the Setup
```bash
# Test the pre-push stage manually with buildfab
buildfab run pre-push

# Or test using pre-push utility
pre-push test

# Test individual actions
buildfab action version@check
buildfab action git@untracked
```

### Version Utility

This project uses the `version` CLI utility for **semantic version validation** and **platform/git detection variables**. Installation instructions for the `version` utility can be found in the [Build section](#installing-version-utility) below. The `version` utility provides:

- Semantic version format validation
- Version comparison and sorting
- Git tag integration
- Platform and Git detection variables

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

### Include Feature

buildfab supports including external YAML files to organize complex configurations into smaller, manageable files:

```yaml
project:
  name: "my-project"

# Include other configuration files
include:
  - actions.yml          # Exact file path (must exist)
  - config/*.yml         # Glob pattern (directory must exist, files optional)
  - stages/common.yml    # Subdirectory file

# Main configuration
actions:
  - name: main-action
    run: echo "main action"
```

#### Include Behavior

- **Exact file paths** (`actions.yml`, `config/file.yml`): Must exist or configuration fails
- **Glob patterns** (`config/*.yml`, `file-*.yml`): Directory must exist, but no files required
- **Merge order**: Later includes override earlier ones
- **Circular includes**: Detected and prevented
- **YAML files only**: Only `.yml` and `.yaml` files are processed

#### Example: Split Configuration

**Main file (`project.yml`)**:
```yaml
project:
  name: "my-project"

include:
  - actions/test.yml
  - actions/build.yml
  - stages/ci.yml

stages:
  pre-push:
    steps:
      - action: test
      - action: build
```

**Actions file (`actions/test.yml`)**:
```yaml
actions:
  - name: test
    run: go test ./...
  - name: test-coverage
    run: go test -cover ./...
```

**Actions file (`actions/build.yml`)**:
```yaml
actions:
  - name: build
    run: go build ./...
  - name: build-static
    run: go build -ldflags="-s -w" ./...
```

**Stages file (`stages/ci.yml`)**:
```yaml
stages:
  ci:
    steps:
      - action: test
      - action: build
      - action: test-coverage
```

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

This project uses buildfab for its build process and can build itself using its own configuration. The buildfab project includes comprehensive self-building capabilities with automatic tool checking and installation.

#### Self-Building with buildfab (Recommended)

The buildfab project can build itself using its own configuration:

```bash
# Check if all required tools are installed
buildfab run pre-check

# Install missing tools if needed
buildfab run pre-install

# Build the project using buildfab
buildfab run build

# Run tests
buildfab run test

# Create release artifacts
buildfab run release
```

#### Build Stages

The buildfab project includes several build stages:

- **`pre-check`**: Check if all required tools (conan, cmake, goreleaser, go, version utility, pre-push utility) are installed
- **`pre-install`**: Install missing tools automatically
- **`build`**: Build the project with all dependencies
- **`test`**: Run cross-platform tests
- **`release`**: Create release artifacts and packages

#### Manual Build (Alternative)

If you prefer to build manually without using buildfab:

```bash
# Build with CMake/Conan directly
mkdir build && cd build
cmake ..
cmake --build .

# Or build with Go directly
go build -o buildfab cmd/buildfab/main.go
```

#### Required Tools

The buildfab project requires the following tools for building:

- **Go 1.22+**: Programming language
- **Conan**: Package manager for Go toolchain
- **CMake**: Build system (optional, Conan can provide it)
- **GoReleaser**: Release automation (installed automatically if Go is available)
- **Version utility**: For version management (installed via pre-install stage)
- **Pre-push utility**: For git hooks (installed via pre-install stage)

### Test
```bash
go test ./... -race
```

## Documentation

- [Features and Examples](docs/Features-and-examples.md) - Comprehensive features documentation with detailed examples
- [YAML Syntax Reference](docs/YAML-syntax-reference.md) - Complete YAML configuration syntax reference
- [Comparison with Others](docs/Comparison-with-others.md) - Detailed comparison with Taskfile, GitHub Actions, Make, and other tools
- [Release Announcement](docs/Release-announcement.md) - Latest release highlights and feature overview
- [Project Specification](docs/Project-specification.md) - Complete technical specification
- [API Reference](docs/Library.md) - Library API documentation
- [Developer Workflow](docs/Developer-workflow.md) - Development setup and workflow
- [Build System](docs/Build.md) - Build and packaging documentation

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Contributing

See [Developer Workflow](docs/Developer-workflow.md) for contribution guidelines.
