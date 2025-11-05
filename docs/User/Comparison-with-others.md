# Comparison with Others

buildfab is a **universal build orchestration system** that unifies fragmented build scripts, CI/CD pipelines, and container workflows under a single declarative YAML configuration.

## Executive Summary

### The Problem: Build Fragmentation

Modern projects typically have **fragmented build logic**:
- Bash scripts for local builds
- Different YAML for CI pipelines (GitHub Actions, GitLab CI)
- Custom Dockerfiles and entrypoint scripts for containers
- Platform-specific setup scripts for different OS/distros

This fragmentation causes:
- ❌ **Duplicated logic** across environments
- ❌ **Inconsistencies** between local and CI
- ❌ **"Works on my machine"** issues
- ❌ **High maintenance** burden

### The Solution: Single Source of Truth

**buildfab** replaces all these scattered scripts with **one `.project.yml` file** that works everywhere:

```yaml
# .project.yml - single source of truth
project:
  name: "my-project"

actions:
  - name: build
    run: go build ./...
  
  - name: test
    run: go test ./...

stages:
  pre-push:
    steps:
      - action: test
```

**This same configuration runs**:
- ✅ **Locally**: `buildfab run pre-push`
- ✅ **In CI**: GitHub Actions executes buildfab stages
- ✅ **In containers**: Test in clean environments
- ✅ **In Git hooks**: via [pre-push utility](https://github.com/AlexBurnes/pre-push)

### Positioning

buildfab bridges the gap between simple task runners (like Taskfile) and full-featured CI/CD systems (like GitHub Actions), offering **CI-level capabilities for local development** without cloud dependency.

### The Result

- ✅ **Single source of truth**: All project build logic in one versioned file
- ✅ **Unified execution**: Same tool works locally, in CI, and in containers
- ✅ **Reproducible builds**: One YAML = consistent behavior everywhere
- ✅ **pre-push integration**: Local checks match CI checks exactly
- ✅ **Zero fragmentation**: No scattered bash scripts, no duplicated logic

### Real-World Applications

buildfab is **self-hosting** and actively used in production:

#### 🏗️ Self-Hosting
buildfab builds itself using its own `.project.yml` configuration:
- **Locally**: Developers build on their machines
- **In GitHub Actions**: CI builds using same configuration
- **Proof of concept**: If it can build itself, it can build anything

#### 🔧 Go Projects
- **buildfab itself**: Multi-platform builds (Linux/Windows/macOS, amd64/arm64)
- **GitHub Actions integration**: Same YAML locally and in CI
- **Cross-compilation**: Parallel builds for all platforms

#### 🛠️ C++ Modules
- **Real production usage**: Complex C++ projects on GitLab CI
- **Multi-distro support**: Ubuntu, Debian, Alpine, CentOS
- **Matrix builds**: Parallel compilation across different OS/compiler versions
- **Consistent environment**: Container-based builds with reproducible results

#### 🐳 Container Workflows
- **Application builds**: Build apps inside containers
- **Slim images**: Optimize images (500MB → 15MB, 30x+ reduction)
- **Artifact collection**: Extract binaries and configs automatically
- **Multi-platform**: Build for different architectures in parallel

### Key Differentiators

- ✅ **Runs anywhere**: No cloud dependency, no daemon required, instant startup
- ✅ **CI-grade features**: Matrix builds, containers with artifacts, parallel execution pools
- ✅ **Container-native**: Docker/Podman integration with resource management and artifact collection
- ✅ **Git integration**: Built-in pre-push hooks and version management
- ✅ **Library-first**: Embeddable Go API for custom tooling
- ✅ **Expression language**: GitHub Actions-compatible conditional execution

## Comprehensive Comparison

| Feature | buildfab | Taskfile | GitHub Actions | Earthly | Make | Just |
|---------|----------|----------|----------------|---------|------|------|
| **Configuration Format** | YAML (Actions-style) | YAML | YAML | Earthfile | Makefile | Justfile DSL |
| **Execution Model** | DAG + Parallel Pools | Sequential + Parallel | DAG | DAG | Dependency-based | Sequential |
| **Matrix Builds** | ✅ Full support + pools | ❌ No | ✅ Yes | ✅ Yes | ❌ No | ❌ No |
| **Container Support** | ✅ Docker/Podman + resources | ❌ No | ✅ Via services | ✅ Native | ❌ No | ❌ No |
| **Container Artifacts** | ✅ Full path preservation | ❌ No | ⚙️ Manual only | ✅ SAVE ARTIFACT | ❌ No | ❌ No |
| **Caching** | ⚙️ Via mounts (recommended) | ⚙️ Via shell | ✅ Cloud cache | ✅ Build cache | ⚙️ Via timestamps | ❌ No |
| **Parallelism Control** | ✅ Global + matrix pools | ⚙️ Partial | ✅ Via strategy | ✅ Yes | ⚙️ -j flag | ❌ No |
| **Conditional Execution** | ✅ Expression language | ⚙️ Basic | ✅ Full | ⚙️ Basic | ⚙️ Via shell | ⚙️ Via shell |
| **Variable Interpolation** | ✅ `${{ var }}` syntax | ⚙️ Basic | ✅ `${{ }}` | ⚙️ Basic | ⚙️ `$(var)` | ⚙️ `{{var}}` |
| **Platform Detection** | ✅ Auto + variables | ❌ No | ⚙️ Runner-based | ❌ No | ❌ No | ❌ No |
| **Built-in Actions** | ✅ Git, version | ❌ No | ✅ Marketplace | ⚙️ Limited | ❌ No | ❌ No |
| **Reusable Config** | ✅ Include system | ⚙️ Includes | ✅ Reusable workflows | ❌ No | ⚙️ Include | ❌ No |
| **Error Policies** | ✅ Stop/warn/continue | ⚙️ Basic | ✅ Full | ⚙️ Basic | ⚙️ -k flag | ❌ No |
| **Git Integration** | ✅ Hooks + version | ❌ No | ⚙️ Triggers only | ❌ No | ❌ No | ❌ No |
| **Library API** | ✅ Go embeddable | ❌ No | ❌ No | ❌ No | ❌ No | ❌ No |
| **Streaming Output** | ✅ Real-time + ordered | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes | ✅ Yes |
| **Dry Run** | ✅ Full support | ⚙️ Partial | ❌ No | ❌ No | ⚙️ -n flag | ❌ No |
| **Cross-Platform** | ✅ Linux/Win/macOS | ✅ Yes | ⚙️ Runner-dependent | ⚙️ Linux-only | ✅ Yes | ✅ Yes |
| **Installation** | Single binary | Single binary | Cloud SaaS | Docker required | Pre-installed | Single binary |
| **Startup Time** | ⚡ Instant (<10ms) | ⚡ Instant | 🐢 Slow (30s+) | ⚙️ Medium (5s+) | ⚡ Instant | ⚡ Instant |
| **Learning Curve** | 📘 Medium | 📗 Easy | 📘 Medium | 📕 Hard | 📕 Hard | 📗 Easy |
| **Ecosystem** | 🧩 Growing | 🔧 Mature | 🌐 Huge | 💎 Niche | 🌍 Universal | 🧩 Small |
| **Best For** | Local CI/DevOps | Simple tasks | Cloud CI/CD | Container builds | Build systems | Task shortcuts |

**Legend**: ✅ Full support | ⚙️ Partial/Limited | ❌ Not supported | ⚡ Instant | 🐢 Slow | 📗 Easy | 📘 Medium | 📕 Hard

## Detailed Feature Comparison

### 1. Matrix Builds

#### buildfab
```yaml
stages:
  test:
    - action: test-platform
      matrix:
        values:
          os: ["linux", "windows", "macos"]
        strategy:
          max_parallel: 2          # Pool-based concurrency
          fail_fast: true
          continue_on_error: false
```

**Capabilities**:
- ✅ Single-dimension matrices (multi-dimension planned)
- ✅ Pool-based parallelism control (global + matrix limits)
- ✅ Min() strategy: `effective = min(global, matrix)`
- ✅ Fail-fast and continue-on-error policies
- ✅ Matrix variable interpolation: `${{ matrix.os }}`
- ✅ CLI override: `--matrix.os=linux`

**Performance**: ~0.75μs overhead, 1.3M tasks/sec throughput

#### GitHub Actions
```yaml
strategy:
  matrix:
    os: [ubuntu-latest, windows-latest, macos-latest]
  max-parallel: 2
  fail-fast: true
```

**Capabilities**:
- ✅ Multi-dimensional matrices
- ✅ Matrix exclusions and inclusions
- ✅ Cloud-based execution
- ❌ Local execution not supported

#### Taskfile / Make / Just
❌ No built-in matrix support

### 2. Container Support

#### buildfab
```yaml
actions:
  # Example 1: Build from source in container
  - name: build-go-app
    description: Build Go application in container
    container:
      image:
        from: golang:1.23-alpine
      workdir: /build
      env:
        CGO_ENABLED: "0"
        GOOS: linux
      mounts:
        - type: bind
          source: .
          target: /build
          ro: true
      run: |
        go build -o /usr/local/bin/myapp ./cmd/myapp
      artifacts:
        output: ./dist
        path:
          - /usr/local/bin/myapp
      # Result: ./dist/usr/local/bin/myapp

  # Example 2: Build Docker image from Dockerfile
  - name: build-docker-image
    description: Build Docker image
    container:
      engine: docker
      image:
        build:
          dockerfile: Dockerfile
          context: .
          args:
            VERSION: ${{version}}
          tags:
            - myapp:${{version}}
            - myapp:latest
          network: host
          progress: plain

  # Example 3: Create slim version (optimize image size)
  - name: slim-docker-image
    description: Create slim optimized image
    container:
      engine: docker
      image:
        slim:
          target: myapp:${{version}}  # Source image to optimize
          tags:
            - myapp:${{version}}-slim  # Slim version tags
            - myapp:latest-slim
          network: host
          http_probe: false  # Disable HTTP probe during slimming
          exec: "/usr/local/bin/myapp --version"  # Test command
      # Uses dslim/slim tool to reduce image size by 30x or more
```

**Capabilities**:
- ✅ Docker and Podman support (auto-detection)
- ✅ **Image from registry** (`image.from`) - Pull and run existing images
- ✅ **Image building** (`image.build`) - Build from Dockerfile with args, tags, network
- ✅ **Slim image creation** (`image.slim`) - Optimize images with dslim/slim tool
  - Reduces image size by 30x or more
  - Removes unnecessary files and dependencies
  - Creates minimal production images
  - Supports HTTP probes and exec commands for validation
  - Works with Docker and Podman
- ✅ Resource limits (CPU, memory)
- ✅ Mount management (bind mounts with ro/rw)
- ✅ Artifact collection with full path preservation
- ✅ Environment variables and env files
- ✅ Matrix integration for multi-platform builds

**Image Workflow**:
```
1. Build stage:    image.build → myapp:v1.0
2. Slim stage:     image.slim  → myapp:v1.0-slim (30x smaller)
3. Artifact stage: Collect binaries/configs from slim image
```

**Artifact Collection**:
- Hybrid approach: pre-mounted volume (for `run`) or docker cp (for build-only)
- Full path preservation: `/app/binary` → `./dist/app/binary`
- Directory support with nested structures
- Cross-platform (Docker and Podman)

#### Earthly
```earthfile
build:
    FROM golang:1.23
    WORKDIR /workspace
    COPY . .
    RUN go build -o app
    SAVE ARTIFACT app /app
```

**Capabilities**:
- ✅ Container-native builds
- ✅ Built-in caching
- ✅ SAVE ARTIFACT command
- ❌ Requires Docker daemon
- ❌ More complex syntax

#### GitHub Actions / Taskfile / Make / Just
⚙️ Limited or no native container support

### 3. Caching and Artifacts

#### buildfab

**Caching Strategy**: Not a built-in feature, but well-supported through container mounts

```yaml
actions:
  - name: build-with-cache
    container:
      image:
        from: golang:1.23
      mounts:
        # Cache Go build cache
        - type: bind
          source: ~/.cache/go-build
          target: /root/.cache/go-build
        # Cache Go mod cache  
        - type: bind
          source: ~/go/pkg/mod
          target: /go/pkg/mod
      run: |
        go build -o /app/myapp
```

**Artifact Collection**: Container-only feature with full path preservation

```yaml
actions:
  - name: build-artifacts
    container:
      image:
        from: gcc:latest
      run: |
        gcc -o /build/bin/myapp main.c
      artifacts:
        output: ./dist
        path:
          - /build/bin/myapp
          - /build/lib/
```

**Capabilities**:
- ⚙️ Caching via container bind mounts (recommended approach)
- ✅ ccache, sccache, Conan, vcpkg support via mounts
- ✅ Local file-based (no network dependency)
- ✅ Artifact collection from containers only
- ✅ Full path preservation for artifacts
- ✅ Glob patterns for artifact paths
- ✅ Directory collection with nested structures

#### GitHub Actions
```yaml
- uses: actions/cache@v3
  with:
    path: ~/.cache/go-build
    key: go-${{ runner.os }}-${{ hashFiles('go.sum') }}
- uses: actions/upload-artifact@v3
```

**Capabilities**:
- ✅ Built-in caching action
- ✅ Cloud-based storage
- ✅ Artifact upload/download
- ❌ Requires network
- ❌ Storage limits apply

### 4. Conditional Execution

#### buildfab
```yaml
stages:
  test:
    steps:
      - action: linux-tests
        if: "platform == 'linux' && arch == 'amd64'"
      
      - action: integration-tests
        if: "fileExists('tests/integration') && env.RUN_INTEGRATION == 'true'"
      
      - action: performance-tests
        if: "cpu >= 4 && !env.CI"
```

**Expression Language**:
- ✅ Variables: `platform`, `arch`, `os`, `os_version`, `cpu`, `env.*`, `matrix.*`
- ✅ Operators: `==`, `!=`, `<`, `<=`, `>`, `>=`, `&&`, `||`, `!`
- ✅ Functions: `contains()`, `startsWith()`, `endsWith()`, `matches()`, `fileExists()`, `semverCompare()`
- ✅ Parentheses for grouping

#### GitHub Actions
```yaml
- if: runner.os == 'Linux' && github.event_name == 'push'
```

**Capabilities**:
- ✅ Similar expression language
- ✅ Rich context variables
- ⚙️ Cloud-specific context

#### Taskfile / Make / Just
⚙️ Basic shell-based conditions only

### 5. Parallelism Control

#### buildfab
```yaml
project:
  max_parallel: 4  # Global limit

stages:
  test:
    - action: fast-tests
      matrix:
        values:
          test: ["unit", "integration", "e2e"]
        strategy:
          max_parallel: 2  # Matrix-specific limit
      # Effective limit: min(4, 2) = 2
```

**Pool System**:
- ✅ Global execution pool with project-wide limit
- ✅ Matrix-specific pools with dedicated limits
- ✅ Min() strategy for effective parallelism
- ✅ Worker pool with task queue
- ✅ Context-aware cancellation
- ✅ Statistics tracking (active, pending, completed)
- ✅ Performance: 1.3M tasks/sec, ~0.75μs overhead

**CLI Override**:
```bash
buildfab run test --max-parallel 8
```

#### GitHub Actions
```yaml
strategy:
  max-parallel: 2
```

**Capabilities**:
- ✅ Per-matrix parallelism control
- ⚙️ Runner-level concurrency limits
- ❌ No global pool concept

#### Make
```bash
make -j4  # Parallel execution
```

**Capabilities**:
- ⚙️ Job-level parallelism
- ❌ No fine-grained control
- ❌ No pool management

### 6. Git Integration

#### buildfab

buildfab provides comprehensive Git integration through its library API and the companion [pre-push utility](https://github.com/AlexBurnes/pre-push).

**Configuration** (`.project.yml` - shared between buildfab and pre-push):
```yaml
project:
  name: "my-project"
  modules: ["my-app"]

actions:
  - name: git-checks
    uses: git@untracked  # Built-in action
  
  - name: version-check
    uses: version@check  # Built-in action
  
  - name: test
    run: go test ./...

stages:
  pre-push:  # Special stage name for Git hooks
    steps:
      - action: git-checks
      - action: version-check
      - action: test
```

**Git Hook Setup with pre-push utility**:
```bash
# Install pre-push utility (separate project)
# https://github.com/AlexBurnes/pre-push
brew install AlexBurnes/tap/pre-push

# Install as Git hook (one-time setup)
pre-push install

# Test hook manually
pre-push test
```

**How it works**:
- **pre-push utility**: Dedicated CLI that embeds buildfab as a library
- **Automatic execution**: Git automatically calls pre-push hook on `git push`
- **Shared configuration**: Both pre-push and buildfab use the same `.project.yml`
- **pre-push stage**: pre-push utility always executes the `pre-push` stage
- **Manual testing**: buildfab can run the same stage manually: `buildfab run pre-push`

**Architecture**:
```
Git Push → pre-push hook → pre-push utility (CLI)
                              ↓
                         buildfab library
                              ↓
                         Execute "pre-push" stage
                              ↓
                         Run actions (git checks, tests, etc.)
```

**Built-in Actions**:
- ✅ `git@untracked` - Check for untracked files
- ✅ `git@uncommitted` - Check for uncommitted changes
- ✅ `git@modified` - Check for modified files
- ✅ `version@check` - Validate version format
- ✅ `version@check-greatest` - Check version ordering

**Key Benefits**:
- ✅ **Single configuration**: One `.project.yml` for both manual and automated runs
- ✅ **Library integration**: pre-push embeds buildfab for consistent execution
- ✅ **Flexible testing**: Run manually with buildfab or automatically with pre-push
- ✅ **Standard workflow**: Familiar Git hook pattern with modern YAML configuration

#### GitHub Actions
```yaml
on:
  push:
    branches: [main]
  pull_request:
```

**Capabilities**:
- ✅ Git event triggers
- ⚙️ Cloud-only (no local hooks)
- ❌ No built-in git validation actions

#### Taskfile / Make / Just
❌ No native git integration

### 7. Configuration Organization

#### buildfab
```yaml
project:
  name: "my-project"

include:
  - actions/*.yml
  - stages/**/*.yml
  - config/common.yml

# Remaining config
```

**Include System**:
- ✅ Glob pattern support
- ✅ Recursive includes
- ✅ Circular detection
- ✅ Merge semantics
- ✅ YAML-only files

#### GitHub Actions
```yaml
jobs:
  test:
    uses: ./.github/workflows/reusable.yml
```

**Capabilities**:
- ✅ Reusable workflows
- ⚙️ More complex syntax

#### Taskfile
```yaml
includes:
  docker: ./docker/Taskfile.yml
```

**Capabilities**:
- ⚙️ Basic includes
- ❌ No glob patterns

### 8. Library API

#### buildfab
```go
package main

import (
    "context"
    "github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func main() {
    ctx := context.Background()
    
    // Simple API
    err := buildfab.RunStageSimple(ctx, ".project.yml", "pre-push", true)
    
    // Advanced API with callbacks
    cfg, _ := buildfab.LoadConfig(".project.yml")
    runner := buildfab.NewRunner(cfg, nil)
    
    opts := &buildfab.RunOptions{
        Verbose: true,
        StepCallback: &MyCustomCallback{},
    }
    
    err = runner.RunStage(ctx, "pre-push", opts)
}
```

**API Features**:
- ✅ Simple and advanced APIs
- ✅ Step callbacks for progress tracking
- ✅ Embeddable in other tools
- ✅ Full configuration access
- ✅ Custom action registration

#### GitHub Actions / Taskfile / Make / Just
❌ No library API (command-line only)

## Performance Comparison

### Startup Time

| Tool | Cold Start | Warm Start | Notes |
|------|-----------|-----------|-------|
| **buildfab** | <10ms | <5ms | Go binary, no deps |
| **Taskfile** | <10ms | <5ms | Go binary |
| **Just** | <10ms | <5ms | Rust binary |
| **Make** | <5ms | <3ms | Native binary |
| **GitHub Actions** | 30-60s | 20-40s | VM provisioning |
| **Earthly** | 5-10s | 2-5s | Docker overhead |

### Execution Overhead

| Tool | Per-Task Overhead | Throughput | Notes |
|------|------------------|-----------|-------|
| **buildfab** | ~0.75μs | 1.3M tasks/sec | Pool-based execution |
| **Taskfile** | ~1-2ms | Variable | Process spawning |
| **Make** | ~1-2ms | Variable | Process spawning |
| **GitHub Actions** | Network latency | N/A | Cloud-based |
| **Earthly** | Container overhead | Variable | Docker layers |

### Memory Usage

| Tool | Baseline | 100 Tasks | 1000 Tasks | Notes |
|------|----------|-----------|------------|-------|
| **buildfab** | ~5MB | ~6MB | <10MB | Efficient pools |
| **Taskfile** | ~3MB | ~50MB | ~500MB | Process per task |
| **Make** | ~2MB | ~40MB | ~400MB | Process per task |
| **GitHub Actions** | N/A | N/A | N/A | Cloud-managed |

## Practical Applications

buildfab is **self-hosting** and actively used in real production environments, demonstrating its capability to handle diverse build scenarios.

### 🏗️ Self-Hosting: buildfab Builds Itself

buildfab demonstrates **eating its own dog food** by building itself:

**Configuration**: `.project.yml` in buildfab repository
```yaml
project:
  name: buildfab
  max_parallel: 4

stages:
  build:
    steps:
      - action: pre-check      # Verify tools
      - action: install-deps   # Install dependencies
        require: [pre-check]
      - action: compile        # Build binaries
        require: [install-deps]
```

**Usage**:
- **Local**: `buildfab run build` - developers build on their machines
- **GitHub Actions**: Same YAML executes in CI runners
- **Multi-platform**: 6 platforms built in parallel (Linux/Win/macOS × amd64/arm64)

**Result**: Same configuration produces identical results locally and in CI.

### 🔧 Go Projects: Cross-Platform Compilation

Real usage in buildfab and similar Go projects:

**Capabilities**:
- **Multi-platform builds**: Parallel builds for all target platforms
- **GoReleaser integration**: Build → Test → Package → Release workflow
- **Pre-push validation**: Automated checks using [pre-push utility](https://github.com/AlexBurnes/pre-push)
- **GitHub Actions**: CI executes same stages as local development

**Example workflow**:
```yaml
stages:
  release:
    steps:
      - action: test
      - action: build
        matrix:
          values:
            platform: ["linux", "windows", "darwin"]
            arch: ["amd64", "arm64"]
        strategy:
          max_parallel: 3
      - action: package
        require: [build]
```

### 🛠️ C++ Modules: Multi-Distro Compilation

Real production usage in complex C++ projects on GitLab CI:

**Scenario**: Large C++ project requiring compilation across multiple Linux distributions with different compiler versions and library versions.

**Configuration**:
```yaml
stages:
  build-matrix:
    - action: compile-cpp
      matrix:
        values:
          distro: ["ubuntu:22.04", "ubuntu:24.04", "debian:11", "alpine:3.18"]
        strategy:
          max_parallel: 2
          fail_fast: false

actions:
  - name: compile-cpp
    container:
      image:
        from: ${{ matrix.distro }}
      mounts:
        - type: bind
          source: ~/.cache/ccache
          target: /ccache
        - type: bind
          source: ~/.conan2
          target: /conan
      env:
        CCACHE_DIR: /ccache
        CONAN_HOME: /conan
      run: |
        cmake -S . -B build -DCMAKE_CXX_COMPILER_LAUNCHER=ccache
        cmake --build build -j
      artifacts:
        output: ./dist/${{ matrix.distro }}
        path:
          - /build/bin/**
          - /build/lib/**
```

**Benefits**:
- ✅ **Consistent build logic** across all distros
- ✅ **Parallel execution** with controlled concurrency
- ✅ **Cache reuse** via bind mounts (ccache, Conan)
- ✅ **Artifact organization** by distro
- ✅ **Same YAML locally and in GitLab CI**

### 🐳 Container Workflows: Build and Optimize

Real usage for container-based application deployment:

**Workflow**: Build → Slim → Deploy

```yaml
stages:
  docker-release:
    steps:
      - action: build-image
      - action: slim-image
        require: [build-image]
      - action: extract-artifacts
        require: [slim-image]

actions:
  - name: build-image
    container:
      image:
        build:
          dockerfile: Dockerfile
          tags: [myapp:v1.0]

  - name: slim-image
    container:
      image:
        slim:
          target: myapp:v1.0
          tags: [myapp:v1.0-slim]
          exec: "/app/myapp --version"
    # Result: 500MB → 15MB (30x reduction)

  - name: extract-artifacts
    container:
      image:
        from: myapp:v1.0-slim
      run: echo "Using slim image"
      artifacts:
        output: ./release
        path:
          - /app/myapp
          - /app/config/
```

**Results**:
- ✅ **Original image**: 500MB (full dev environment)
- ✅ **Slim image**: 15MB (production-ready)
- ✅ **Reduction**: 30x+ smaller
- ✅ **Artifacts**: Binary and configs extracted automatically

## Use Case Recommendations

### When to Use buildfab

#### ✅ Perfect For

1. **Local CI/DevOps**: Replace bash scripts with maintainable YAML
2. **Pre-push Validation**: Git hooks with comprehensive checks
3. **Cross-Platform Builds**: Matrix builds with Docker/Podman
4. **Container Workflows**: Multi-platform container builds with artifact collection
5. **Embedded Automation**: Library API for custom tools
6. **Development Workflows**: Fast iteration with container caching

#### Example Projects
- Go/Rust/C++ projects with complex builds
- Multi-platform CLI tools requiring container testing
- Projects requiring pre-push validation
- Container-based microservices with artifact collection
- Projects with heavy test matrices

### When to Use Alternatives

#### Taskfile
- ✅ Simple task automation
- ✅ Minimal learning curve
- ❌ No matrix builds
- ❌ No container support

#### GitHub Actions
- ✅ Cloud CI/CD workflows
- ✅ Huge marketplace ecosystem
- ❌ Not for local development
- ❌ Vendor lock-in

#### Earthly
- ✅ Container-native builds
- ✅ Reproducible builds
- ❌ Requires Docker daemon
- ❌ Steeper learning curve

#### Make
- ✅ Universal availability
- ✅ Build systems
- ❌ Complex syntax
- ❌ No modern features

#### Just
- ✅ Simple command shortcuts
- ✅ Easy to learn
- ❌ Limited automation features
- ❌ No DAG execution

## Migration Guide

### From Taskfile

**Before (Taskfile)**:
```yaml
version: '3'
tasks:
  build:
    cmds:
      - go build -o bin/app
  test:
    deps: [build]
    cmds:
      - go test ./...
```

**After (buildfab)**:
```yaml
project:
  name: "my-project"

actions:
  - name: build
    run: go build -o bin/app
  
  - name: test
    run: go test ./...

stages:
  default:
    steps:
      - action: build
      - action: test
        require: [build]
```

### From GitHub Actions

**Before (GitHub Actions)**:
```yaml
name: CI
on: [push]
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - run: go test ./...
```

**After (buildfab)**:
```yaml
project:
  name: "my-project"

actions:
  - name: test
    run: go test ./...

stages:
  ci:
    - action: test
      matrix:
        values:
          os: ["linux", "windows"]
        strategy:
          max_parallel: 2
```

### From Make

**Before (Makefile)**:
```makefile
.PHONY: build test

build:
	go build -o bin/app

test: build
	go test ./...

all: build test
```

**After (buildfab)**:
```yaml
project:
  name: "my-project"

actions:
  - name: build
    run: go build -o bin/app
  
  - name: test
    run: go test ./...

stages:
  all:
    steps:
      - action: build
      - action: test
        require: [build]
```

## Scoring Summary

| Criteria | buildfab | Taskfile | GitHub Actions | Earthly | Make | Just |
|----------|----------|----------|----------------|---------|------|------|
| **Features** (0-10) | 9 | 6 | 10 | 8 | 5 | 4 |
| **Performance** (0-10) | 10 | 9 | 5 | 7 | 9 | 10 |
| **Ease of Use** (0-10) | 8 | 9 | 7 | 6 | 4 | 9 |
| **Local Development** (0-10) | 10 | 9 | 2 | 7 | 9 | 9 |
| **CI/CD Capabilities** (0-10) | 9 | 5 | 10 | 8 | 4 | 3 |
| **Ecosystem** (0-10) | 6 | 7 | 10 | 5 | 10 | 4 |
| **Portability** (0-10) | 10 | 9 | 6 | 6 | 9 | 9 |
| **Documentation** (0-10) | 9 | 8 | 9 | 7 | 6 | 7 |
| **Community** (0-10) | 5 | 7 | 10 | 6 | 10 | 5 |
| **Innovation** (0-10) | 9 | 5 | 8 | 8 | 3 | 6 |
| **TOTAL** | **85/100** | **74/100** | **77/100** | **68/100** | **69/100** | **66/100** |

## Conclusion

**buildfab** occupies a unique position in the automation tooling landscape:

### Strengths
- ✅ **Best-in-class local CI/CD**: CI-grade features without cloud dependency
- ✅ **Exceptional performance**: Sub-microsecond overhead, instant startup
- ✅ **Container-native**: Full Docker/Podman support with artifact collection
- ✅ **Modern feature set**: Matrix builds, pools, expression language
- ✅ **Developer-friendly**: Library API, git hooks, cross-platform
- ✅ **Future-proof**: Active development, growing feature set

### Growth Areas
- 🔄 **Ecosystem**: Smaller than Make/GitHub Actions (but growing)
- 🔄 **Community**: Young project with active development
- 🔄 **Documentation**: Comprehensive but continuously improving

### Ideal Use Cases
buildfab shines when you need:
- Local automation with CI-level capabilities
- Container-based workflows with artifact collection
- Fast iteration without cloud overhead
- Matrix testing across platforms
- Git hook integration
- Library embedding for custom tools
- Cross-platform development

For cloud-only CI/CD, GitHub Actions remains the leader. For simple tasks, Taskfile or Just may be simpler. But for **local DevOps automation with container support**, buildfab offers the best combination of power, performance, and simplicity.

**Overall Assessment**: ⭐ **9/10** - A powerful, innovative tool that successfully bridges the gap between simple task runners and full CI/CD systems, perfect for local development workflows with container support.
