# New Features Implementation Plan: Matrix, Container, and Caching

## Overview

This document outlines the comprehensive implementation plan for three major new features in buildfab:

1. **Matrix Feature** - Parallel execution across multiple configurations
2. **Container Feature** - Docker and Podman support for isolated execution
3. **Caching Feature** - Build optimization through intelligent caching

## Implementation Priority

Based on user requirements and technical dependencies:

1. **Phase 1: Matrix Feature** (First implementation)
   - Focus on testing different parallel strategies with long-running tests
   - Foundation for container and caching features
   - Core matrix expansion and job management

2. **Phase 2: Container Feature** (Second implementation)
   - Docker and Podman engine support
   - Container execution environment
   - Integration with matrix feature

3. **Phase 3: Caching Feature** (Third implementation)
   - Build optimization through caching
   - Integration with matrix and container features
   - Performance improvements

## Phase 1: Matrix Feature Implementation

### Core Requirements

#### Matrix Configuration
```yaml
stages:
  test_matrix:
    - action: simple_test
      matrix:
        test: ["test_one", "test_two"]  # Single dimension only for initial implementation
      strategy:
        max_parallel: 2
        fail_fast: true
        continue_on_error: false
        order: "fifo"
```

**Key Clarifications**:
- Matrix supported at **stage level only** (not action level)
- **Single dimension only** for initial implementation (no nested matrices)
- Matrix expansion happens at **parse time**
- Empty matrix values are **ignored**
- No limit on total number of jobs generated

#### Matrix Expansion Logic
- **Input**: Matrix configuration with multiple dimensions
- **Output**: Cartesian product of all matrix values
- **Example**: `test: [1,2]` × `platform: [linux,win]` = 4 jobs

#### Matrix Strategy Configuration
- **max_parallel**: Maximum concurrent jobs (default: all)
- **fail_fast**: **Stop all jobs** on first failure (default: false)
- **continue_on_error**: Stage succeeds even if some jobs fail (default: false)
- **order**: Job scheduling order - "fifo" or "random" (default: "fifo")

**Key Clarifications**:
- **fail_fast stops all jobs** (not just prevents new ones)
- **continue_on_error** interaction with fail_fast follows Matrix-shedules.md semantics
- Additional order options (reverse, priority) added to roadmap

#### Matrix Variable Interpolation
- Support `${{ matrix.* }}` variables in action commands
- Matrix variables available in all action contexts
- Integration with existing variable system

### Technical Implementation

#### New Data Structures
```go
type MatrixConfig struct {
    Values    map[string][]interface{} `yaml:"values"`
    Strategy  MatrixStrategy           `yaml:"strategy"`
}

type MatrixStrategy struct {
    MaxParallel      int    `yaml:"max_parallel"`
    FailFast         bool   `yaml:"fail_fast"`
    ContinueOnError  bool   `yaml:"continue_on_error"`
    Order           string `yaml:"order"`
}

type MatrixJob struct {
    ID       string
    Matrix   map[string]interface{}
    Action   *Action
    Status   JobStatus
    Result   *Result
}
```

#### Matrix Expansion Engine
- **Cartesian Product Algorithm**: Generate all combinations
- **Job Creation**: Create MatrixJob instances for each combination
- **Variable Context**: Prepare matrix variables for each job

#### Matrix Job Scheduler
- **Queue Management**: FIFO or random job ordering
- **Parallelism Control**: Respect max_parallel limits
- **Status Tracking**: Track job status and results
- **Error Handling**: Implement fail_fast and continue_on_error logic

#### CLI Integration
- **Matrix commands**: `buildfab <stage>` (no separate matrix command)
- **Override support**: `--matrix.test=test_one --max-parallel=1`
- **Status reporting**: Real-time job status and progress

**Key Clarifications**:
- **No separate matrix command** - matrix runs as part of normal stage execution
- **No unified CLI approach** - features integrate naturally with existing CLI
- **No feature disabling via CLI flags** - features enabled by configuration only

### Testing Strategy

#### Unit Tests
- Matrix expansion logic with various configurations
- Job scheduler with different strategies
- Variable interpolation with matrix variables
- Error handling and edge cases

#### Integration Tests
- End-to-end matrix execution
- Long-running test scenarios
- Parallel execution validation
- Error propagation and recovery

#### Performance Tests
- Large matrix configurations (100+ jobs)
- Memory usage with many parallel jobs
- Scheduling performance under load

## Phase 2: Container Feature Implementation

### Core Requirements

#### Container Configuration
```yaml
actions:
  - name: container-build
    container:
      engine: docker  # or "podman"
      image:
        from: ubuntu:22.04
        # or build from Dockerfile
        build:
          dockerfile: ci/Dockerfile.build
          context: .
          args:
            BASE: ubuntu:22.04
      workdir: /src
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
        - type: bind
          source: ./dist
          target: /out
          ro: false
      env:
        ARTIFACTS_DIR: /out
      user: ""  # root or uid:gid
      network: ""  # default bridge or host
      cache:
        ccache: /ccache
      run_stage: build  # or run: [commands]
```

#### Container Engine Support
- **Docker**: Primary container engine (first preference)
- **Podman**: Alternative container engine with Docker compatibility
- **Engine Detection**: Automatic detection and fallback
- **Command Translation**: Convert Docker commands to Podman
- **Future engines**: containerd, LXC added to roadmap

**Key Clarifications**:
- **Engine preference order**: docker, podman, other
- **Version compatibility**: Handled when cases arise
- **Additional engines**: containerd, LXC added to roadmap

#### Image Management
- **From Existing**: Use pre-built images
- **Build from Dockerfile**: Build custom images
- **Image Tagging**: Automatic tagging for built images
- **Image Cleanup**: Remove temporary images

#### Mount Management
- **Bind Mounts**: Host directory to container path
- **Volume Mounts**: Named volumes for persistence
- **Cache Mounts**: Specialized cache directories
- **Read-only/Read-write**: Mount permissions

#### Environment Integration
- **Environment Variables**: Pass variables to container
- **Working Directory**: Set container working directory
- **User Context**: Run as specific user
- **Network Mode**: Bridge or host networking

### Technical Implementation

#### New Data Structures
```go
type ContainerConfig struct {
    Engine    string            `yaml:"engine"`
    Image     ContainerImage    `yaml:"image"`
    Workdir   string            `yaml:"workdir"`
    Mounts    []ContainerMount  `yaml:"mounts"`
    Env       map[string]string `yaml:"env"`
    User      string            `yaml:"user"`
    Network   string            `yaml:"network"`
    Cache     ContainerCache    `yaml:"cache"`
    RunStage  string            `yaml:"run_stage"`
    Run       []string          `yaml:"run"`
}

type ContainerImage struct {
    From  string           `yaml:"from"`
    Build *ContainerBuild  `yaml:"build"`
}

type ContainerBuild struct {
    Dockerfile string            `yaml:"dockerfile"`
    Context    string            `yaml:"context"`
    Args       map[string]string `yaml:"args"`
}

type ContainerMount struct {
    Type   string `yaml:"type"`
    Source string `yaml:"source"`
    Target string `yaml:"target"`
    RO     bool   `yaml:"ro"`
}
```

#### Container Execution Engine
- **Engine Interface**: Abstract interface for Docker/Podman
- **Command Generation**: Generate container run commands
- **Lifecycle Management**: Start, monitor, and cleanup containers
- **Error Handling**: Container-specific error handling

#### Buildfab Integration
- **run_stage Support**: Execute buildfab stages inside containers
- **Binary Availability**: **Pre-installed** buildfab in container images
- **Configuration Mounting**: Mount project.yml and dependencies
- **Artifact Collection**: Collect results from container

**Key Clarifications**:
- **Buildfab pre-installed** in container images (not mounted at runtime)
- **Version compatibility**: Use specific version install scripts (e.g., v0.16.5)
- **Binary path**: Follow LBS standards, install to system bin directories
- **No global container configuration** - per-action configuration only

### Testing Strategy

#### Unit Tests
- Container configuration parsing
- Command generation for different engines
- Mount and environment handling
- Error scenarios and edge cases

#### Integration Tests
- End-to-end container execution
- Docker and Podman engine testing
- Buildfab integration inside containers
- Artifact collection and cleanup

#### Cross-platform Tests
- Linux container execution
- Windows container execution (if supported)
- Different base images and configurations

## Phase 3: Caching Feature Implementation

### Core Requirements

#### Cache Configuration
```yaml
actions:
  - name: cached-build
    container:
      image: { from: ubuntu:22.04 }
      mounts:
        - type: bind
          source: ~/.cache/buildfab/ccache
          target: /ccache
        - type: bind
          source: ~/.conan2
          target: /conan
      env:
        CCACHE_DIR: /ccache
        CCACHE_MAXSIZE: 5G
        CONAN_HOME: /conan
      cache:
        ccache: /ccache
        conan: /conan
        vcpkg: /vcpkg-cache
```

#### Cache Types
- **ccache**: C/C++ compiler cache
- **Conan**: C++ package manager cache
- **vcpkg**: C++ package manager cache
- **Go modules**: Go module cache
- **npm/yarn**: Node.js package cache
- **pip**: Python package cache

**Key Clarifications**:
- **Directory-based caches only** (no plugin system)
- **User-configured** cache types (no automatic detection)
- **Separate action steps** for cache management
- **No global cache configuration** - per-action configuration only

#### Cache Management
- **Cache Keys**: **User-configured** using variables `${{ var.name }}`
- **Cache Isolation**: Separate caches by configuration
- **Cache Cleanup**: **Separate action steps** for cleanup
- **Cache Statistics**: Monitor cache usage and hit rates

**Key Clarifications**:
- **Cache keys**: User decides and configures using variables
- **Cache cleanup**: Separate action steps, user decides and configures
- **Manual cache management**: Separate action steps, user decides and configures
- **Cache access control**: Bind mount options (ro, rw)

#### Cache Integration
- **Matrix Integration**: Separate caches per matrix job
- **Container Integration**: Mount caches in containers
- **Build Integration**: Automatic cache configuration

### Technical Implementation

#### New Data Structures
```go
type CacheConfig struct {
    CCache  string `yaml:"ccache"`
    Conan   string `yaml:"conan"`
    Vcpkg   string `yaml:"vcpkg"`
    GoMod   string `yaml:"go_mod"`
    NPM     string `yaml:"npm"`
    Pip     string `yaml:"pip"`
}

type CacheManager struct {
    Config     CacheConfig
    CacheDir   string
    Statistics CacheStatistics
}

type CacheStatistics struct {
    HitRate    float64
    MissRate   float64
    TotalSize  int64
    EntryCount int
}
```

#### Cache Management System
- **Cache Key Generation**: Create unique keys for cache entries
- **Cache Directory Management**: Organize caches by type and configuration
- **Cache Validation**: Verify cache integrity and compatibility
- **Cache Cleanup**: Remove old and invalid cache entries

#### Cache Integration
- **Environment Variables**: Set cache-related environment variables
- **Mount Configuration**: Configure cache mounts for containers
- **Build Integration**: Integrate caches with build processes

### Testing Strategy

#### Unit Tests
- Cache key generation and validation
- Cache directory management
- Cache statistics and monitoring
- Cache cleanup and maintenance

#### Integration Tests
- End-to-end caching with different build types
- Cache performance and hit rate validation
- Cache integration with matrix and container features
- Cross-platform cache compatibility

#### Performance Tests
- Cache hit rate optimization
- Cache size management
- Build time improvement measurement
- Memory usage with large caches

## Implementation Timeline

### Phase 1: Matrix Feature (Weeks 1-4)
- **Week 1**: Core matrix data structures and expansion logic
- **Week 2**: Matrix job scheduler and parallelism control
- **Week 3**: CLI integration and variable interpolation
- **Week 4**: Testing and documentation

### Phase 2: Container Feature (Weeks 5-8)
- **Week 5**: Container configuration and engine interface
- **Week 6**: Docker and Podman implementation
- **Week 7**: Buildfab integration and artifact collection
- **Week 8**: Testing and documentation

### Phase 3: Caching Feature (Weeks 9-12)
- **Week 9**: Cache configuration and management system
- **Week 10**: Cache integration with matrix and container features
- **Week 11**: Performance optimization and monitoring
- **Week 12**: Testing and documentation

## Feature Integration

### Integration Approach
- **Matrix + Container**: Matrix jobs can run in containers via run_stage or run actions
- **Matrix + Caching**: Each matrix job gets isolated caches using `${{ matrix.name }}` variables
- **Container + Caching**: Containers can mount and use caches via user-configured mounts
- **All three**: Matrix jobs in containers with isolated caches via user configuration

**Key Clarifications**:
- **No feature disabling** - features enabled by configuration only
- **Feature conflicts result in failure** - no graceful degradation
- **User decides and configures** all feature integrations
- **No unified configuration approach** - features work independently

### Resource Management
- **Global resource limits** (not per-job limits)
- **CPU-based parallelism**: max_parallel must not exceed CPU count
- **Memory-based throttling**: Don't run new jobs if memory usage exceeds configured limit or 100% of available memory
- **No performance monitoring** required initially

### Security Considerations
- **Container security**: User and group settings in action configuration
- **Cache security**: Bind mount options (ro, rw) for access control
- **No security scanning** for container images initially
- **No audit logging** for cache operations initially
- **Security policies** added to roadmap

## Success Criteria

### Matrix Feature
- [ ] Support matrix configurations with **single dimension** (stage level only)
- [ ] Implement configurable parallelism and error handling with **fail_fast stopping all jobs**
- [ ] Provide matrix variable interpolation with **parse-time expansion**
- [ ] Achieve 90%+ test coverage
- [ ] Support long-running test scenarios

### Container Feature
- [ ] Support Docker and Podman engines with **Docker as first preference**
- [ ] Implement comprehensive mount and environment management
- [ ] Provide buildfab integration inside containers with **pre-installed buildfab**
- [ ] Achieve 90%+ test coverage
- [ ] Support cross-platform container execution

### Caching Feature
- [ ] Support **directory-based cache types** (ccache, Conan, vcpkg, etc.) with **user configuration**
- [ ] Implement cache isolation and cleanup via **separate action steps**
- [ ] Provide cache statistics and monitoring
- [ ] Achieve 90%+ test coverage
- [ ] Demonstrate significant build time improvements

## Risk Mitigation

### Technical Risks
- **Container Engine Compatibility**: Test with multiple Docker/Podman versions
- **Cross-platform Issues**: Comprehensive testing on all supported platforms
- **Performance Impact**: Monitor memory and CPU usage with large matrices
- **Cache Corruption**: Implement robust cache validation and cleanup

### Implementation Risks
- **Feature Complexity**: Implement in phases with clear milestones
- **Testing Coverage**: Maintain high test coverage throughout development
- **Documentation**: Keep documentation updated with implementation
- **User Adoption**: Provide clear examples and migration guides

## Next Steps

1. **Requirements Clarification**: Review and refine requirements with stakeholders
2. **Architecture Review**: Validate technical approach and data structures
3. **Implementation Start**: Begin Phase 1 (Matrix Feature) implementation
4. **Regular Reviews**: Weekly progress reviews and milestone validation
5. **User Feedback**: Gather feedback during development and testing phases

## Conclusion

This implementation plan provides a comprehensive roadmap for adding Matrix, Container, and Caching features to buildfab. The phased approach ensures manageable development cycles while building toward a powerful, integrated automation platform that can handle complex, multi-environment build and test scenarios.

The features will significantly enhance buildfab's capabilities, making it suitable for enterprise-level CI/CD workflows while maintaining its simplicity and ease of use for smaller projects.
