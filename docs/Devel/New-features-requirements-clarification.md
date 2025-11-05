# New Features Requirements Clarification

## Overview

This document clarifies the requirements and implementation priorities for the three new features: Matrix, Container, and Caching. Based on the provided documentation and user requirements, this document addresses key questions and provides detailed specifications.

## Matrix Feature Requirements

### Core Functionality

#### 1. Matrix Configuration Syntax
**Question**: What is the exact YAML syntax for matrix configuration?

**Answer**: Based on the provided example, the syntax should be:
```yaml
stages:
  test_matrix:
    - action: simple_test
      matrix:
        test: ["test_one", "test_two"]
        platform: ["linux", "windows", "macos"]
      strategy:
        max_parallel: 2
        fail_fast: true
        continue_on_error: false
        order: "fifo"
```

**Clarification Needed**:
- Should matrix be supported at the stage level or action level?
  **Answer**: Right now on stage level. At stage level.
- Can matrix be nested (matrix within matrix)?
  **Answer**: Will be to complicated, may be later, but not right now. Not.
- What is the maximum number of matrix dimensions supported?
  **Answer**: For current plan is only 1.

#### 2. Matrix Expansion Logic
**Question**: How should matrix values be expanded?

**Answer**: Cartesian product of all matrix dimensions:
- `test: [1,2]` × `platform: [linux,win]` = 4 jobs
- Each job gets unique combination of matrix values
- Matrix variables available as `${{ matrix.test }}`, `${{ matrix.platform }}`

**Clarification Needed**:
- Should matrix expansion happen at parse time or runtime?
  **Answer**: Parse time
- How to handle empty matrix values or invalid combinations?
  **Answer**: Ignore
- Should there be a limit on total number of jobs generated?
  **Answer**: Not

#### 3. Matrix Strategy Configuration
**Question**: What are the exact semantics of matrix strategy options?

**Answer**:
- **max_parallel**: Maximum concurrent jobs (default: all)
- **fail_fast**: Stop scheduling new jobs on first failure (default: false)
- **continue_on_error**: Stage succeeds even if some jobs fail (default: false)
- **order**: Job scheduling order - "fifo" or "random" (default: "fifo")

**Clarification Needed**:
- Should fail_fast stop in-flight jobs or just prevent new ones?
  **Answer**: stop all jobs
- How should continue_on_error interact with fail_fast?
  **Answer**: See Matrix-shedules.md paragpraph Semantics
- Should order support other options like "reverse" or "priority"?
  **Answer**: could be, add to roadmap of this feature

### Implementation Priorities

#### Phase 1: Basic Matrix Support
1. **Matrix expansion**: Cartesian product generation
2. **Basic job scheduling**: Sequential execution with max_parallel
3. **Matrix variable interpolation**: `${{ matrix.* }}` support
4. **Simple CLI support**: `buildfab matrix <stage>`

#### Phase 2: Advanced Matrix Features
1. **Strategy configuration**: fail_fast, continue_on_error, order
2. **Job queue management**: FIFO/random ordering
3. **Status reporting**: Real-time job status and progress
4. **Error handling**: Comprehensive error propagation

#### Phase 3: Matrix Integration
1. **Container integration**: Matrix jobs in containers
2. **Caching integration**: Matrix-specific cache isolation
3. **Performance optimization**: Large matrix handling
4. **Advanced CLI**: Override support, filtering, etc.

## Container Feature Requirements

### Core Functionality

#### 1. Container Engine Support
**Question**: Which container engines should be supported?

**Answer**: 
- **Docker**: Primary container engine
- **Podman**: Alternative container engine with Docker compatibility
- **Auto-detection**: Automatically detect available engine
- **Fallback**: Graceful fallback between engines

**Clarification Needed**:
- Should we support other engines like containerd or LXC?
  **Answer**: Yes, but add to road map.
- How to handle version compatibility between engines?
  **Answer**: Not yet unknow, leave it now, when case would happen, than clarify it.
- Should there be a preference order for engine selection?
  **Answer**: It could be defined as engine: docker, podmand, other - so docker is a first

#### 2. Container Configuration Syntax
**Question**: What is the exact YAML syntax for container configuration?

**Answer**: Based on the provided example:
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

**Clarification Needed**:
- Should container be a separate action type or a property of existing actions?
  **Answer**: as you can see example it defined in action, so this action defined as container. So answer is property of an action.
- How to handle container-specific error messages and debugging?
  **Answer**: as output of a job 
- Should there be a global container configuration for defaults?
  **Answer**: nope.

#### 3. Buildfab Integration
**Question**: How should buildfab integrate with containers?

**Answer**:
- **run_stage**: Execute buildfab stages inside containers
- **Binary availability**: Ensure buildfab is available in container
- **Configuration mounting**: Mount project.yml and dependencies
- **Artifact collection**: Collect results from container

**Clarification Needed**:
- Should buildfab be pre-installed in container images or mounted at runtime?
  **Answer**: pre installed.
- How to handle buildfab version compatibility between host and container?
  **Answer**: it is possible, buildfav -V and install scripts from version release, instead of latest. For example v0.16.5 wget -O - https://github.com/AlexBurnes/buildfab/releases/download/v0.16.5/buildfab-linux-amd64-install.sh | sh
- Should there be a way to specify buildfab binary path in container?
  **Answer**: Do not required it follow LBS and install into system bin directories

### Implementation Priorities

#### Phase 1: Basic Container Support
1. **Container configuration**: Basic YAML parsing
2. **Docker engine**: Basic Docker integration
3. **Image management**: From existing images
4. **Basic mounts**: Bind mounts for source and output

#### Phase 2: Advanced Container Features
1. **Podman support**: Podman engine integration
2. **Image building**: Dockerfile support
3. **Advanced mounts**: Volume mounts, cache mounts
4. **Environment management**: Variables, working directory, user

#### Phase 3: Buildfab Integration
1. **run_stage support**: Execute stages inside containers
2. **Binary management**: Buildfab availability in containers
3. **Artifact collection**: Results collection and cleanup
4. **Error handling**: Container-specific error reporting

## Caching Feature Requirements

### Core Functionality

#### 1. Cache Types Support
**Question**: Which cache types should be supported?

**Answer**: Based on the provided documentation:
- **ccache**: C/C++ compiler cache
- **Conan**: C++ package manager cache
- **vcpkg**: C++ package manager cache
- **Go modules**: Go module cache
- **npm/yarn**: Node.js package cache
- **pip**: Python package cache

**Clarification Needed**:
- Should there be a plugin system for additional cache types?
  **Answer**: No, at current time we will use directory base caches
- How to handle cache type detection and validation?
  **Answer**: It not a problem of buildfab, outside it, by scripts or steps in actions clean_conan for example with condition action with if: .
- Should there be a way to disable specific cache types?
  **Answer**: User will deside by configuration.

#### 2. Cache Configuration Syntax
**Question**: What is the exact YAML syntax for cache configuration?

**Answer**: Based on the provided example:
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

**Clarification Needed**:
- Should cache be a separate action type or integrated with container actions?
  **Answer**: separate action step, user decide and configure. 
- How to handle cache configuration validation and error reporting?
  **Answer**:  action step, user decide and configure.
- Should there be a global cache configuration for defaults?
  **Answer**: Nope.

#### 3. Cache Management
**Question**: How should caches be managed and maintained?

**Answer**:
- **Cache keys**: Generate unique keys for cache entries
- **Cache isolation**: Separate caches by configuration
- **Cache cleanup**: Automatic cleanup of old entries
- **Cache statistics**: Monitor cache usage and hit rates

**Clarification Needed**:
- How to generate cache keys for different cache types?
  **Answer**: user decide and configure using variables ${{ var.name }}/${{ var.name }}
- What is the cleanup policy for old cache entries?
  **Answer**: separate action step, user decide and configure.
- Should there be a way to manually manage caches?
  **Answer**: separate action step, user decide and configure.

### Implementation Priorities

#### Phase 1: Basic Cache Support
1. **Cache configuration**: Basic YAML parsing
2. **ccache support**: C/C++ compiler cache
3. **Basic mount management**: Cache directory mounting
4. **Environment variables**: Cache-related environment setup

#### Phase 2: Advanced Cache Features
1. **Multiple cache types**: Conan, vcpkg, Go modules, etc.
2. **Cache isolation**: Configuration-based cache separation
3. **Cache statistics**: Hit rate monitoring and reporting
4. **Cache cleanup**: Automatic maintenance and cleanup

#### Phase 3: Cache Integration
1. **Matrix integration**: Matrix-specific cache isolation
2. **Container integration**: Cache mounting in containers
3. **Performance optimization**: Cache hit rate optimization
4. **Advanced management**: Cache validation and repair

## Integration Requirements

### Feature Integration
**Question**: How should the three features integrate with each other?

**Answer**:
- **Matrix + Container**: Matrix jobs can run in containers
  **Answer**: yes run_stage, or run will be done for action in container, matrix just run action, but action could be run steps localy or in a container.
- **Matrix + Caching**: Each matrix job gets isolated caches
  **Answer**: user decide and configure using matrix.name variable.
- **Container + Caching**: Containers can mount and use caches
  **Answer**: user decide and configure container mounts.
- **All three**: Matrix jobs in containers with isolated caches
  **Answer**: user decide and configure container mounts.

**Clarification Needed**:
- Should there be a way to disable feature integration?
  **Answer**: Nope, no configuration no feature.
- How to handle feature conflicts or incompatibilities?
  **Answer**: failure.
- Should there be a unified configuration approach?
  **Answer**: Could not be clarified yet, show me examples.

### CLI Integration
**Question**: How should the features integrate with the CLI?

**Answer**:
- **Matrix commands**: `buildfab <stage>` 
- **Override support**: `--matrix.test=test_one --max-parallel=1`

**Clarification Needed**:
- Should there be a unified CLI approach for all features?
  **Answer**: Nope
- How to handle feature-specific help and documentation?
  **Answer**: Documentation
- Should there be a way to disable features via CLI flags?
  **Answer**: Nope

## Testing Requirements

### Test Coverage
**Question**: What level of test coverage is required?

**Answer**:
- **Unit tests**: 90%+ coverage for all new code
- **Integration tests**: End-to-end feature testing
- **Performance tests**: Large matrix and cache scenarios
- **Cross-platform tests**: All supported platforms

**Clarification Needed**:
- Should there be specific test scenarios for each feature?
  **Answer**: Yes, as much as possible.
- How to test container functionality without Docker/Podman?
  **Answer**: Test containers functionality with podmand
- Should there be performance benchmarks for caching?
  **Answer**: Nor right now.

### Test Data
**Question**: What test data and scenarios are needed?

**Answer**:
- **Matrix tests**: Various matrix configurations and edge cases
- **Container tests**: Different base images and configurations
- **Cache tests**: Different cache types and scenarios
- **Integration tests**: Feature combination scenarios

**Clarification Needed**:
- Should there be test fixtures for common scenarios?
- How to handle test data cleanup and isolation?
- Should there be automated test data generation?

**Answer**: Will be clarified on implementaion.

## Documentation Requirements

### User Documentation
**Question**: What documentation is needed for users?

**Answer**:
- **Feature guides**: Comprehensive usage guides for each feature
- **Examples**: Real-world examples and use cases
- **Migration guides**: How to migrate from existing solutions
- **Troubleshooting**: Common issues and solutions

**Clarification Needed**:
- Should there be separate documentation for each feature?
  **Answer**: yes for matrix and containers
- How to handle feature integration documentation?
  **Answer**: link to main documetation and in a readme
- Should there be video tutorials or interactive examples?
  **Answer**: Nope.

### Developer Documentation
**Question**: What documentation is needed for developers?

**Answer**:
- **API documentation**: Complete API reference
- **Architecture docs**: System design and implementation details
- **Contributing guides**: How to contribute to the features
- **Testing guides**: How to test and validate features

**Clarification Needed**:
- Should there be separate documentation for each feature?
  **Answer**: yes
- How to handle cross-feature documentation?
  **Answer**: in one document with examples for platforms
- Should there be design documents for each feature?
  **Answer**: Optional.

## Performance Requirements

### Scalability
**Question**: What are the scalability requirements?

**Answer**:
- **Matrix jobs**: Support 100+ parallel jobs
- **Container execution**: Handle multiple concurrent containers
- **Cache size**: Support multi-gigabyte caches
- **Memory usage**: Efficient memory usage for large scenarios

**Clarification Needed**:
- What are the specific performance targets for each feature?
- How to handle resource limits and throttling?
  **Answer**: Optional. By memory and cpu resources, i.e parallel strategy must not exceed number of cpu, limitin memory usage, do not run new jobs in parallel if memory usage more then set value in configuration or 100% of available memory.
- Should there be performance monitoring and alerting?
  **Answer**: not yet require

### Resource Management
**Question**: How should resources be managed?

**Answer**:
- **Memory limits**: Configurable memory limits for jobs
- **CPU limits**: Configurable CPU limits for jobs
- **Disk limits**: Configurable disk limits for caches
- **Network limits**: Configurable network limits for containers

**Clarification Needed**:
- Should there be global resource limits or per-job limits?
  **Answer**: global
- How to handle resource contention and prioritization?
- Should there be resource monitoring and reporting?
  **Answer**: not

## Security Requirements

### Container Security
**Question**: What security considerations are needed for containers?

**Answer**:
- **User isolation**: Run containers as non-root when possible
- **Network isolation**: Isolated network configurations
- **Mount security**: Secure mount configurations
- **Image security**: Validate and scan container images

**Clarification Needed**:
- Should there be security scanning for container images?
  **Answer**: Not
- How to handle privileged container requirements?
  **Answer**: setting in action user and group
- Should there be security policies and enforcement?
  **Answer**: Not right now. Add to road map.

### Cache Security
**Question**: What security considerations are needed for caches?

**Answer**:
- **Cache isolation**: Prevent cache leakage between jobs
- **Access control**: Proper permissions for cache directories
- **Cache validation**: Validate cache integrity and authenticity
- **Cleanup security**: Secure cleanup of sensitive cache data

**Clarification Needed**:
- Should there be encryption for sensitive cache data?
  **Answer**: Not yet.
- How to handle cache access control and permissions?
  **Answer**: bind option ro, rw
- Should there be audit logging for cache operations?
  **Answer**: Nope.

## Conclusion

This requirements clarification document addresses the key questions and provides detailed specifications for implementing the Matrix, Container, and Caching features in buildfab. The phased implementation approach ensures manageable development cycles while building toward a comprehensive automation platform.

The next steps are to:
1. **Review and approve** these requirements with stakeholders
2. **Begin Phase 1 implementation** starting with the Matrix feature
3. **Establish regular review cycles** to validate progress and adjust requirements
4. **Create detailed technical specifications** for each feature component
5. **Set up testing infrastructure** for comprehensive feature validation

This approach ensures that buildfab evolves into a powerful, enterprise-ready automation platform while maintaining its simplicity and ease of use for smaller projects.
