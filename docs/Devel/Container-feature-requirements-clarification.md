# Container Feature Requirements Clarification

## Document Purpose

This document provides a comprehensive clarification of the Docker and Podman container feature requirements for buildfab. It serves as a reference for implementation planning and decision-making.

## Current Project Status

- **Matrix Feature**: ✅ **COMPLETED** (v0.16.10_feat.1_fix.1)
- **Container Feature**: ❌ **NOT IMPLEMENTED** (Phase 2 - Next Priority)
- **Caching Feature**: ❌ **NOT IMPLEMENTED** (Phase 3)

## Container Feature Overview

The container feature will enable buildfab to execute actions and stages inside Docker or Podman containers, providing isolated execution environments for build and test processes.

## 1. Implementation Priority and Timeline

### Questions for Clarification

**Q1.1: Implementation Priority**
- Should container feature implementation begin immediately after matrix feature completion?
- Are there any external dependencies or blockers that would delay implementation?
- What is the target completion timeline for the container feature?

**Q1.2: Phase Integration**
- Should container feature be implemented as a standalone feature or integrated with existing matrix functionality?
- How should container feature interact with the completed matrix feature?

**Your Answers:**
```
[Please provide your answers here]
```

## 2. Container Engine Support

### Current Requirements
- **Primary Engine**: Docker (first preference)
- **Secondary Engine**: Podman (Docker compatibility)
- **Future Engines**: containerd, LXC (roadmap)
- **Engine Detection**: Automatic detection and fallback
- **Command Translation**: Convert Docker commands to Podman

### Questions for Clarification

**Q2.1: Engine Priority and Detection**
- Should buildfab automatically detect available engines or require explicit configuration?
- What should happen if neither Docker nor Podman is available?
- Should there be a fallback to local execution or should it fail with clear error messages?

**Q2.2: Engine Version Compatibility**
- What are the minimum supported versions for Docker and Podman?
- Should buildfab check engine versions and warn about compatibility issues?
- How should version compatibility be handled for future engine additions?

**Q2.3: Engine Configuration**
- Should users be able to specify custom Docker/Podman binary paths?
- Should there be support for Docker contexts or Podman connections?
- How should engine-specific configuration be handled?

**Your Answers:**
```
[Please provide your answers here]
```

## 3. Container Configuration Structure

### Current YAML Schema
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
      cpus: [0,1] # number of cpu limts - require set --cpus 2.0 (count of cpus) --cpuset-cpus="0,1"  
      memory: 4G # limiting memory set -m 4294967296 (set in bytes, allow user set in human readable 500M[b] 2G[b])
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
      artifacts: 
        output: host_directory # relative or aboslute host directory
        path: # patterns to collect artifacts in container
          - /usr/local/bin/binary # binary file in directory
          - distr # directory in workdir
          - some_dir/* # pattern for files in some dir
      env:
        ARTIFACTS_DIR: /out
      user: ""  # root or uid:gid
      network: ""  # default bridge or host
      cache:
        ccache: /ccache
      run_stage: build  
      # or run_action: name
      # or run: [commands]
```

### Questions for Clarification

**Q3.1: Configuration Validation**
- What validation rules should be applied to container configuration?
  **Answer** - validate mount directories and image exists
- Should buildfab validate that specified images exist before execution?
  **Answer** yes
- How should invalid or conflicting configuration be handled?
  **Answer** failure

**Q3.2: Image Management**
- Should buildfab automatically pull images if they don't exist locally?
  **Answer** yes, require to pull
- How should image tagging and cleanup be handled for built images?
  **Answer** images could be kept, but containers automaticaly removed, start with --rm
- Should there be support for private registries and authentication?
  **Answer** do not required right now.

**Q3.3: Mount Configuration**
- What validation should be performed on mount paths?
  **Answer** directory exists on a host
- How should mount permissions and ownership be handled?
  **Answer** by options in mounts, if it possible
- Should there be support for volume drivers or custom mount types?
  **Answer** do not required right now, add to road map

**Q3.4: Environment Variables**
- Should all host environment variables be passed to containers by default?
  **Answer** nope
- How should sensitive environment variables be handled?
  **Answer** need examples for clarify
- Should there be support for environment variable files?
  **Answer** yes, view option env_file: .env



## 4. Buildfab Integration Inside Containers

### Current Requirements
- **Pre-installed buildfab**: Must be pre-installed in container images
- **Version-specific**: Use specific version install scripts (e.g., v0.16.5)
- **System paths**: Follow LBS standards, install to system bin directories
- **No custom paths**: No way to specify custom buildfab binary path
- **run_stage support**: Execute buildfab stages inside containers

### Questions for Clarification

**Q4.1: Buildfab Installation**
- Should buildfab be installed during image build or at runtime?
  **Answer** if image will build using Dockerfile that will be defined by user, if option build is ommited, i.e image will be run then buildfab require to be installed
- What happens if the required buildfab version is not available in the container?
  **Answer** bildfab will installed the copy of current buildfab on platform, not from git releases.
- Should there be support for installing buildfab from different sources (GitHub releases, local binary, etc.)?
  **Answer** own copy only

**Q4.2: Configuration Mounting**
- Should the entire project directory be mounted or only specific files?
  **Answer** as user mounts, if not mount a direcotry not mount anything
- How should buildfab configuration files be handled inside containers?
  **Answer** should be copied current configuration into container with buildfab in case of run container
- Should there be support for different configuration files per container?
  **Answer** nope

**Q4.3: Artifact Collection**
- How should artifacts be collected from containers after execution?
  **Answer** must be an option artifacts with list of directories and files 
- Should there be automatic artifact collection or manual configuration?
  **Answer** manual
- How should artifact permissions and ownership be handled?
  **Answer** uid and guid of running buildfab or if user define uid:gid in option 'user'

**Q4.4: Error Handling**
- How should container execution errors be reported?
  **Answer** as usal for actions repsect verbosity 
- Should buildfab provide container-specific debugging information?
  **Answer** for -vv should write full command for docker to run 
- How should container cleanup be handled on errors?
  **Answer** leave it to docker or podman engine, as I say containers will run with --rm, docker build use its own rules


## 5. Integration with Matrix Feature

### Current Integration Plan
- Matrix jobs can run in containers via `run_stage` or `run` actions
- Container configuration per matrix job vs shared configuration
- Matrix variables should work inside containers

### Questions for Clarification

**Q5.1: Matrix-Container Integration**
- Should each matrix job get its own container or share containers?
- How should matrix variables be passed to container environments?
- Should container configuration be matrix-aware (different images per matrix job)?

**Q5.2: Resource Management**
- How should container resources be managed across matrix jobs?
- Should there be global resource limits for all containers?
- How should container cleanup be handled for matrix jobs?

**Q5.3: Parallel Execution**
- Should matrix jobs with containers respect the same parallelism limits?
- How should container startup time affect matrix job scheduling?
- Should there be container-specific parallelism controls?

**Your Answers:**
```
Matrix and container will be indepotent features, so does not require some integration all will be done using variables.
```

## 6. Cross-Platform Considerations

### Current Requirements
- Linux container execution (primary)
- Windows container execution (if supported)
- Different base images and configurations

### Questions for Clarification

**Q6.1: Platform Support**
- Which platforms should be supported for container execution?
- Should there be platform-specific container configurations?
- How should platform detection work for container features?

**Q6.2: Base Image Compatibility**
- What base images should be officially supported?
- Should there be platform-specific base image recommendations?
- How should base image compatibility be validated?

**Q6.3: Cross-Platform Testing**
- What testing strategy should be used for cross-platform container execution?
- Should there be CI/CD integration for container testing?
- How should platform-specific issues be handled?

**Your Answers:**
```
At first we will focus on linux containers. 
```

## 7. Security and Resource Management

### Current Requirements
- Container security: User and group settings
- Resource limits: CPU, memory, disk, network
- No security scanning initially (roadmap item)

### Questions for Clarification

**Q7.1: Security Configuration**
- What security defaults should be applied to containers?
- Should there be support for privileged containers?
- How should user and group permissions be handled?

**Q7.2: Resource Limits**
- What are the default resource limits for containers?
- Should users be able to configure resource limits per action?
- How should resource limit violations be handled?

**Q7.3: Network Security**
- What network policies should be applied by default?
- Should there be support for custom network configurations?
- How should network access be controlled?

**Your Answers:**
```
Resource limits if user define cpu and memory.
```

## 8. Error Handling and Debugging

### Questions for Clarification

**Q8.1: Error Reporting**
- How should container-specific errors be reported to users?
- Should buildfab provide container logs and debugging information?
- How should container startup failures be handled?

**Q8.2: Debugging Support**
- Should there be support for interactive container debugging?
- How should container state be inspected during execution?
- Should there be verbose logging for container operations?

**Q8.3: Cleanup and Recovery**
- How should failed containers be cleaned up?
- Should there be automatic retry mechanisms for container failures?
- How should partial container execution be handled?

**Your Answers:**
```
Nothing special. As usal docker output all info on failed output as for action, if not respect verbose level.
Cleanup leave docker
```

## 9. Performance and Optimization

### Questions for Clarification

**Q9.1: Container Startup**
- How should container startup time be optimized?
- Should there be support for container image caching?
- How should container reuse be handled for similar actions?

**Q9.2: Resource Usage**
- How should container resource usage be monitored?
- Should there be automatic resource optimization?
- How should resource conflicts be resolved?

**Q9.3: Parallel Execution**
- How should multiple containers be managed efficiently?
- Should there be container pooling or reuse mechanisms?
- How should container resource allocation be optimized?

**Your Answers:**
```
No limits and caching used by docker build and docker run. Nothing special
```

## 10. Documentation and User Experience

### Questions for Clarification

**Q10.1: User Documentation**
- What level of documentation detail is needed for container features?
  **Answer** describe options, examples.
- Should there be examples for common container use cases?
- How should container troubleshooting be documented?
  **Answer** allow user to run docker manualy, require only provide docker command on errors to user to reproduce the trouble and fix it by user.

**Q10.2: CLI Integration**
- What CLI commands should be added for container management?
  **Answer** nothing special, may be later in real cases
- Should there be container-specific help and examples?
- How should container configuration be validated via CLI?

**Q10.3: Error Messages**
- What level of detail should container error messages provide?
- Should there be suggestions for resolving container issues?
- How should container-specific help be integrated?


## 11. Testing Strategy

### Questions for Clarification

**Q11.1: Test Coverage**
- What level of test coverage is required for container features?
- Should there be integration tests with real containers?
- How should container testing be automated in CI/CD?

**Q11.2: Test Scenarios**
- What container scenarios should be tested?
- Should there be tests for different base images and configurations?
- How should container failure scenarios be tested?

**Q11.3: Performance Testing**
- Should there be performance tests for container execution?
- How should container resource usage be tested?
- What benchmarks should be established for container features?

**Your Answers:**
```
For testing feature build image, then run container based on that image. use buildfab detection for cpu limiting is work. Check mounted directories and ro, rw options. Check artifacts generation and output. 
```

## 12. Implementation Approach

### Questions for Clarification

**Q12.1: Development Strategy**
- Should container feature be developed incrementally or as a complete implementation?
- What are the key milestones for container feature development?
- How should container feature be integrated with existing codebase?

**Q12.2: Code Organization**
- How should container-related code be organized in the project?
- Should there be separate packages for different container engines?
- How should container configuration be integrated with existing configuration system?

**Q12.3: Dependencies**
- What external dependencies are acceptable for container features?
- Should buildfab use existing container libraries or implement custom solutions?
- How should container engine dependencies be managed?

**Your Answers:**
```
Implement incrementaly.
1. run container on some image alpine or debian and run action|stage|run
2. run container with mounts 
3. run container with limiting cpus and memory
4. build countainer image and run it
```

## Next Steps

After completing this clarification document:

1. **Review Answers**: Analyze all provided answers for consistency and completeness
2. **Update Requirements**: Refine container feature requirements based on clarifications
3. **Create Implementation Plan**: Develop detailed implementation plan with specific tasks
4. **Update Documentation**: Update project documentation with clarified requirements
5. **Begin Implementation**: Start container feature development based on clarified requirements

## Document Status

- **Created**: [Current Date]
- **Status**: Awaiting Clarification
- **Next Review**: After answers are provided
- **Owner**: [To be assigned]

---

**Note**: Please provide detailed answers to all questions to ensure comprehensive understanding of container feature requirements. If any questions are not applicable or need further clarification, please indicate this in your response.
