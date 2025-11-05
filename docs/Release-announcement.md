# buildfab: Universal Build Orchestration System

**Release v0.32.0** | November 2025

## Introduction

We're excited to announce **buildfab** - a universal build orchestration system that **unifies fragmented build scripts** under a single declarative YAML configuration.

### The Problem

Modern projects suffer from **build fragmentation**:

- 📄 Bash scripts for local builds
- 📄 Different YAML for CI pipelines (GitHub Actions, GitLab CI)
- 📄 Custom Dockerfiles and entrypoint scripts for containers
- 📄 Platform-specific setup scripts for different distros

Result: **duplicated logic, inconsistencies, maintenance nightmare**.

### The Solution

buildfab provides **one `.project.yml` file** that works everywhere:

- ✅ **Locally**: Same commands as CI
- ✅ **In CI**: GitHub Actions/GitLab executes buildfab
- ✅ **In containers**: Clean environment testing
- ✅ **In Git hooks**: Automated validation

Built in Go, buildfab combines the simplicity of task runners like Taskfile with the advanced features of GitHub Actions, delivering **CI/CD-grade capabilities** without cloud dependency.

## What's New in v0.32.0

### 🎯 Hierarchical DAG Architecture (Major Refactoring)

The most significant update in v0.32.0 is the complete redesign of the execution engine with a **hierarchical DAG architecture**:

- **Job-Based Execution**: Matrix combinations now form "job nodes" containing sequential steps
- **Improved Parallelism**: Jobs execute in parallel waves, steps within jobs execute sequentially
- **Nested Matrix Support**: Matrix on stage references now works correctly (previously would hang)
- **Better Performance**: Proper dependency waiting with semaphore-based concurrency control
- **Fixed Issues**: Resolved infinite loops and hanging scenarios in complex matrix builds
- **Thread-Safe**: Zero race conditions detected in extensive testing

**Impact**: Complex nested matrix builds that previously failed or hung now work flawlessly.

### 🧹 Major Code Cleanup

Removed 1,697 lines of deprecated flat DAG code:
- 11 deprecated functions eliminated from core executor
- 27% reduction in buildfab.go (4,845 → 3,528 lines)
- Single execution path (no more confusion about which DAG to use)
- Significantly improved code maintainability

### 📚 Complete Documentation Overhaul

**Library API Documentation** - `docs/Library.md` (782 lines):
- 10 core API methods with detailed examples
- 5 practical usage patterns (basic, variables, timeouts, matrix, callbacks)
- Migration guide from v0.29.1 to v0.32.0
- Custom callbacks and advanced integration topics
- Based on real-world usage from pre-push utility

**v0.32.0 Feature Documentation** (~630 new lines):
- **Hierarchical DAG Execution Model** - Complete explanation of job-based architecture
  - Job vs step execution model
  - Execution waves and dependency resolution
  - Nested matrix hierarchical expansion
  - Sliding window dependencies for concurrency control
  - Condition-based skip smart behavior
- **Advanced Matrix Features** - All v0.32.0 enhancements documented
  - Nested matrix support with examples
  - Complex NOT expressions (operator precedence)
  - Matrix variable propagation
  - User dependency inheritance
  - Global max_parallel enforcement
- **Container Support** - Comprehensive container configuration reference
  - Complete syntax reference (image, engine, resources, volumes, environment)
  - Docker/Podman engine auto-detection
  - Resource limits (CPU, memory, network modes)
  - Volume mounting patterns with variable interpolation
  - Advanced examples (multi-platform, caching, container matrix)
  - Troubleshooting guide and best practices

**Documentation Organization** (51 files):
- **docs/User/**: 6 user-facing documents (now 100% complete for v0.32.0)
- **docs/Devel/**: 40 development documents (implementation details, plans, analysis)
- **docs/User.md**: Comprehensive index with quick start guide
- **docs/Release-announcement.md**: Updated to v0.32.0 with all new features

**Total Documentation**: Over 1,400 lines of new content covering all v0.32.0 features

### 🔧 Expression Parsing Improvements

- Fixed logical NOT operator (`!`) parsing in complex expressions
- Improved operator precedence handling (binary before unary)
- Better parentheses handling for nested conditions
- Complex expressions like `!(matrix.images == 'centos7' && matrix.compiler == 'clang')` now work correctly

### ✨ Enhanced Matrix Builds

- Matrix variables correctly propagate in nested scenarios
- Condition-based skips don't block sliding window dependencies
- User dependencies (`require`, `depends_on`) properly inherited by matrix jobs
- Global `max_parallel` setting now properly enforced

## What is buildfab?

buildfab is a declarative automation framework that uses familiar YAML configuration to orchestrate complex workflows. Whether you're running pre-push validation, building multi-platform containers, or executing parallel test matrices, buildfab provides the tools you need without the complexity.

```yaml
project:
  name: "my-project"
  max_parallel: 4

actions:
  - name: test
    run: go test ./...
    
  - name: build
    uses: docker@build
    with:
      image:
        build:
          dockerfile: Dockerfile

stages:
  pre-push:
    steps:
      - action: test
      - action: build
```

## Key Features

### 🚀 CI-Grade Features, Local Execution

- **Matrix Builds**: Run tests across multiple platforms, versions, or configurations in parallel
- **Container Support**: Native Docker/Podman integration with resource management
- **Pool-Based Parallelism**: Global and matrix-specific concurrency control
- **File Caching**: Intelligent caching system for faster builds
- **Artifact Collection**: Automated artifact management with pattern matching

### ⚡ Exceptional Performance

- **Instant Startup**: <10ms cold start, <5ms warm start
- **Minimal Overhead**: ~0.75μs per task (1000x better than spec requirement)
- **High Throughput**: 1.3 million tasks per second
- **Memory Efficient**: <10MB for 1000 concurrent tasks
- **No Daemon Required**: Single binary, no background processes

### 🎯 Developer-Focused

- **Git Integration**: Built-in pre-push hooks and version validation
- **Expression Language**: GitHub Actions-compatible conditional execution
- **Platform Detection**: Automatic OS/architecture detection
- **Cross-Platform**: Linux, Windows, macOS (amd64/arm64)
- **Library API**: Embeddable Go library for custom tooling

### 📦 Modern Architecture

- **DAG Execution**: Intelligent dependency resolution with parallel processing
- **Include System**: Modular configuration organization
- **Action Variants**: Platform-specific execution paths
- **Built-in Actions**: Pre-configured git and version checks
- **Streaming Output**: Real-time progress with ordered display

## Real-World Examples

### Multi-Platform Testing with Matrix

```yaml
stages:
  test:
    - action: test-platform
      matrix:
        values:
          os: ["linux", "windows", "macos"]
          version: ["1.22", "1.23"]
        strategy:
          max_parallel: 3
          fail_fast: true
```

**Result**: Run 6 test combinations (3×2 matrix) with controlled parallelism and immediate failure detection.

### Container-Based Build

```yaml
actions:
  - name: build
    uses: docker@build
    with:
      image:
        build:
          dockerfile: Dockerfile
          context: .
          args:
            GO_VERSION: "1.23"
      resources:
        cpu: 2
        memory: "4G"
      artifacts:
        - source: /app/dist/**
          dest: ./dist/
```

**Result**: Build in isolated container, limit resources, collect artifacts automatically.

### Pre-Push Validation

```yaml
stages:
  pre-push:
    steps:
      - action: git@untracked      # Built-in git check
      - action: version@check       # Built-in version validation
      - action: test
      - action: lint
        if: "os == 'linux'"        # Conditional execution
```

**Setup**: `buildfab install-hook` - one command, automatic validation on every push.

## Technical Highlights

### Parallel Pool System

buildfab implements a sophisticated pool-based execution system that provides fine-grained concurrency control:

- **Global Pool**: Project-wide parallelism limit via `project.max_parallel`
- **Matrix Pools**: Dedicated pools for matrix steps with their own limits
- **Min() Strategy**: Effective limit = `min(global_max, matrix_max)`
- **Worker Queue**: Task queue with context-aware cancellation
- **Statistics Tracking**: Real-time monitoring of active/pending/completed tasks

**Performance Metrics**:
- Submit overhead: ~0.75μs
- Throughput: 1.3M tasks/second
- GetPool lookup: ~57ns
- Memory: <10MB for 1000 tasks
- Zero goroutine leaks

### Container Engine

Full-featured container support with both Docker and Podman:

- **Image Management**: Build, pull, and execute containers
- **Resource Control**: CPU and memory limits
- **Mount System**: Bind mounts with permission control
- **Artifact Collection**: Hybrid approach (pre-mount + docker cp)
- **Environment**: Variables, env files, working directories
- **Matrix Integration**: Multi-platform container builds

### Expression Language

GitHub Actions-compatible expression system:

```yaml
if: "platform == 'linux' && arch == 'amd64' && cpu >= 4"
if: "fileExists('go.mod') && !env.SKIP_TESTS"
if: "semverCompare(version.version, '1.0.0') >= 0"
```

**Supported**:
- Variables: `platform`, `arch`, `os`, `cpu`, `env.*`, `matrix.*`
- Operators: `==`, `!=`, `<`, `<=`, `>`, `>=`, `&&`, `||`, `!`
- Functions: `contains()`, `startsWith()`, `endsWith()`, `matches()`, `fileExists()`, `semverCompare()`

## Comparison with Alternatives

| Feature | buildfab | Taskfile | GitHub Actions | Make |
|---------|----------|----------|----------------|------|
| Matrix Builds | ✅ Full | ❌ No | ✅ Yes | ❌ No |
| Containers | ✅ Native | ❌ No | ⚙️ Services | ❌ No |
| Caching | ✅ Local | ⚙️ Shell | ✅ Cloud | ⚙️ Timestamps |
| Parallelism | ✅ Pools | ⚙️ Basic | ✅ Strategy | ⚙️ -j flag |
| Expressions | ✅ Full | ⚙️ Basic | ✅ Full | ❌ No |
| Git Hooks | ✅ Built-in | ❌ No | ❌ No | ❌ No |
| Library API | ✅ Go | ❌ No | ❌ No | ❌ No |
| Startup | <10ms | <10ms | 30-60s | <5ms |
| Local First | ✅ Yes | ✅ Yes | ❌ No | ✅ Yes |

**See [Detailed Comparison](Comparison-with-others.md) for comprehensive analysis.**

## Practical Applications

buildfab is **self-hosting** and actively used in real production environments:

### 🏗️ Self-Hosting

**buildfab builds itself** using its own `.project.yml` configuration:

- **Locally**: Developers run `buildfab run build` on their machines
- **In GitHub Actions**: CI executes the same stages with same YAML
- **Multi-platform**: Parallel builds for Linux/Windows/macOS (amd64/arm64)

**Proof of concept**: If it can build itself, it can build anything.

### 🔧 Go Projects

Real usage in buildfab project:

- **Cross-platform compilation**: 6 platforms in parallel (Linux/Win/macOS × amd64/arm64)
- **GitHub Actions integration**: Same commands locally and in CI
- **GoReleaser workflow**: Build → Test → Package → Release
- **Pre-push validation**: Automated checks before every push

### 🛠️ C++ Modules

Real production usage in complex C++ projects:

- **GitLab CI integration**: Container-based builds on GitLab
- **Multi-distro support**: Ubuntu, Debian, Alpine, CentOS
- **Matrix builds**: Parallel compilation across OS/compiler combinations
- **CMake + Conan**: Dependency management with reproducible builds
- **Consistent environment**: Same build logic locally and in CI

### 🐳 Container Workflows

Active usage for container-based development:

- **Application builds**: Compile inside containers for clean environments
- **Slim images**: Optimize images (500MB → 15MB, 30x+ reduction)
- **Multi-platform**: Matrix builds for different architectures
- **Artifact collection**: Automatic extraction of binaries and configs
- **Reproducible builds**: Same Dockerfile + same YAML = same results

## Use Cases

### Perfect For

1. **Local Development**: Replace bash scripts with maintainable YAML
2. **Pre-Push Validation**: Automated checks before committing
3. **Cross-Platform Builds**: Matrix testing across OS/architectures
4. **Container Workflows**: Docker/Podman-based development
5. **Custom Tooling**: Library API for embedded automation
6. **DevOps Teams**: Consistent workflows across team members

### Success Stories

**Self-Hosting**: buildfab successfully builds itself using its own configuration, proving the robustness of the approach. Developers and CI use identical workflows.

**Go Projects**: Replace complex Makefiles with declarative YAML, add matrix testing across Go versions, integrate pre-push hooks for version validation.

**C++ Projects**: Real production usage on GitLab CI with multi-distro matrix builds, CMake/Conan integration, and container-based compilation for reproducibility.

**Container Projects**: Build multi-platform images, run tests in isolated environments, create slim optimized images (30x smaller), collect artifacts automatically, manage resource limits.

**CI/CD Pipelines**: Test locally before pushing, use same configuration for local and CI, eliminate "works on my machine" issues.

## Getting Started

### Installation

**Linux/macOS**:
```bash
curl -sSL https://github.com/AlexBurnes/buildfab/releases/latest/download/install.sh | bash
```

**Windows (Scoop)**:
```powershell
scoop bucket add buildfab https://github.com/AlexBurnes/buildfab-scoop-bucket
scoop install buildfab
```

### Quick Start

1. Create `.project.yml`:
```yaml
project:
  name: "my-project"

actions:
  - name: test
    run: go test ./...

stages:
  pre-push:
    steps:
      - action: test
```

2. Run:
```bash
buildfab run pre-push
```

3. Install git hook:
```bash
buildfab install-hook
```

### Library Usage

```go
import "github.com/AlexBurnes/buildfab/pkg/buildfab"

func main() {
    ctx := context.Background()
    err := buildfab.RunStageSimple(ctx, ".project.yml", "pre-push", true)
}
```

## Legacy Features (v0.20.0 and earlier)

### Parallel Pool Feature (Major)

- ✅ **Pool-Based Execution**: Sophisticated worker pool system with task queues
- ✅ **Global Concurrency Control**: `project.max_parallel` for project-wide limits
- ✅ **Matrix-Specific Pools**: Dedicated pools for matrix steps
- ✅ **Min() Strategy**: Intelligent limit resolution
- ✅ **Context-Aware**: Proper cancellation and cleanup
- ✅ **High Performance**: 1.3M tasks/sec, ~0.75μs overhead

### Comprehensive Testing

- ✅ 20 unit tests covering all pool functionality
- ✅ 6 integration tests with timing validation
- ✅ 7 performance benchmarks confirming specifications
- ✅ 100% pass rate with race detector

### Documentation

- ✅ Updated Matrix-feature.md with pool execution details
- ✅ Enhanced Features-and-examples.md with pool configuration
- ✅ Added YAML-syntax-reference.md pool documentation
- ✅ Updated README with parallelism notes

## Roadmap

### Near Term (Q4 2025)

- Multi-dimensional matrices
- Enhanced caching strategies
- Additional built-in actions
- Plugin system for custom actions
- Visual DAG viewer

### Medium Term (Q1-Q2 2026)

- Remote executors (optional)
- Secrets management integration
- Webhook support
- Performance profiling tools
- Enhanced container features

### Long Term (2026+)

- Distributed execution
- Cloud integration (optional)
- GUI/web interface
- Marketplace for actions
- Advanced monitoring

## Community and Support

### Documentation

- **Quick Start**: [README.md](../README.md)
- **Features**: [Features-and-examples.md](Features-and-examples.md)
- **YAML Reference**: [YAML-syntax-reference.md](YAML-syntax-reference.md)
- **API Documentation**: [Library.md](Library.md)
- **Comparison**: [Comparison-with-others.md](Comparison-with-others.md)

### Get Involved

- **GitHub**: [AlexBurnes/buildfab](https://github.com/AlexBurnes/buildfab)
- **Issues**: Report bugs, request features
- **Discussions**: Ask questions, share ideas
- **Contributions**: Pull requests welcome!

### License

Apache License 2.0 - see [LICENSE](../LICENSE)

## Technical Details

### Architecture

```
┌─────────────────┐
│   CLI Layer     │  ← cmd/buildfab (cobra-based CLI)
├─────────────────┤
│   Library API   │  ← pkg/buildfab (public Go API)
├─────────────────┤
│  Core Engine    │  ← DAG execution + pools
├─────────────────┤
│  Output Manager │  ← Ordered output display
├─────────────────┤
│  Configuration  │  ← YAML parsing + validation
├─────────────────┤
│  Action System  │  ← Built-in + custom actions
├─────────────────┤
│  Container Mgr  │  ← Docker/Podman integration
└─────────────────┘
```

### Implementation Stats

- **Language**: Go 1.23+
- **Lines of Code**: ~15,000 (excluding tests)
- **Test Coverage**: 75%+ (100% on core API)
- **Dependencies**: Minimal (stdlib + cobra + yaml)
- **Platforms**: Linux, Windows, macOS
- **Architectures**: amd64, arm64

### Build System

buildfab can build itself using its own configuration:

```bash
buildfab run pre-check   # Verify tools
buildfab run build       # Build project
buildfab run test        # Run tests
buildfab run release     # Create release
```

**Tools**: Go, CMake, Conan, GoReleaser (auto-installed)

## Why buildfab?

### The Problem

Modern development requires complex automation:

- Multiple platforms and architectures
- Container-based workflows
- Pre-commit/pre-push validation
- CI/CD consistency
- Fast iteration cycles

Existing solutions fall short:

- **Task runners**: Too simple (no matrices, containers, caching)
- **GitHub Actions**: Cloud-only (slow, network-dependent)
- **Make**: Complex syntax, limited features
- **Docker/Earthly**: Container-only, daemon required

### The Solution

buildfab provides:

- ✅ **CI-grade features** without cloud dependency
- ✅ **Blazing performance** with instant startup
- ✅ **Modern capabilities** (matrices, containers, pools)
- ✅ **Simple configuration** (familiar YAML)
- ✅ **Developer-friendly** (git hooks, library API)
- ✅ **Cross-platform** (works everywhere)

### The Result

- **Faster development**: Instant local testing
- **Better quality**: Automated pre-push validation
- **Consistent workflows**: Same config locally and in CI
- **Improved productivity**: Less time debugging, more time coding
- **Team alignment**: Shared automation tooling

## Conclusion

buildfab represents a new category of development tools: **Local CI/CD Platforms**. It combines the best features of task runners, CI/CD systems, and build tools while eliminating their limitations.

Whether you're a solo developer looking for better automation, a team standardizing workflows, or an organization building custom DevOps tools, buildfab provides the foundation you need.

**Get started today**: [Installation Guide](../README.md#installation-and-git-hook-setup)

---

**buildfab** - Universal Local Automation Platform  
Version 0.32.0 | November 2025 | Apache License 2.0

**Project**: https://github.com/AlexBurnes/buildfab  
**Documentation**: https://github.com/AlexBurnes/buildfab/tree/master/docs  
**Issues**: https://github.com/AlexBurnes/buildfab/issues

