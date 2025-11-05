# Container Feature Requirements - Final Specification

## Document Purpose

This document defines the final, clarified requirements for the Docker and Podman container feature implementation in buildfab, based on comprehensive clarification and analysis.

## Executive Summary

The container feature will enable buildfab to execute actions and stages inside Docker or Podman containers, providing isolated execution environments. The feature will be implemented incrementally with focus on Linux containers initially, using a simple, user-driven configuration approach.

## 1. Core Requirements

### 1.1 Container Engine Support
- **Primary Engine**: Docker (first preference)
- **Secondary Engine**: Podman (Docker compatibility)
- **Platform Focus**: Linux containers initially
- **Engine Detection**: Automatic detection with fallback
- **Command Translation**: Convert Docker commands to Podman when needed

### 1.2 Container Configuration Schema

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
      cpus: [0,1]  # CPU limits: --cpus 2.0 --cpuset-cpus="0,1"
      memory: 4G   # Memory limits: -m 4294967296 (human readable: 500M, 2G)
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
      artifacts: 
        output: host_directory  # relative or absolute host directory
        path:  # patterns to collect artifacts in container
          - /usr/local/bin/binary  # binary file in directory
          - distr  # directory in workdir
          - some_dir/*  # pattern for files in some dir
      env:
        ARTIFACTS_DIR: /out
      env_file: .env  # environment variable file
      user: ""  # root or uid:gid
      network: ""  # default bridge or host
      cache:
        ccache: /ccache
      run_stage: build  
      # or run_action: name
      # or run: [commands]
```

## 2. Detailed Requirements

### 2.1 Configuration Validation
- **Mount Directories**: Validate that mount source directories exist on host
- **Image Validation**: Check that specified images exist locally, pull if needed
- **Configuration Conflicts**: Fail on invalid or conflicting configuration
- **Required Fields**: Validate required fields are present

### 2.2 Image Management
- **Automatic Pull**: Pull images if they don't exist locally
- **Container Cleanup**: Use `--rm` flag for automatic container removal
- **Image Retention**: Keep built images, remove containers automatically
- **Private Registries**: Not required initially (roadmap item)

### 2.3 Mount Configuration
- **Host Validation**: Verify mount source directories exist on host
- **Permission Handling**: Use mount options for permissions (ro, rw)
- **Volume Drivers**: Not required initially (roadmap item)
- **User Control**: Users configure all mount points explicitly

### 2.4 Environment Variables
- **No Default Pass-through**: Don't pass all host environment variables by default
- **Explicit Configuration**: Users must explicitly configure environment variables
- **Environment Files**: Support `env_file: .env` option
- **Sensitive Variables**: Need examples for clarification (future enhancement)

### 2.5 Buildfab Integration
- **Installation Strategy**: 
  - If using `build` option: User installs buildfab in Dockerfile
  - If using `from` option with `run_stage` or `run_action`: buildfab installs current copy at runtime
  - If using `from` option with `run` commands: No buildfab installation required
- **Version Management**: Use current buildfab binary, not GitHub releases
- **Configuration Copying**: Copy current configuration into container only when using `run_stage` or `run_action`
- **No Custom Paths**: No way to specify custom buildfab binary path

### 2.6 Artifact Collection
- **Manual Configuration**: Users must configure artifact collection explicitly
- **Pattern Support**: Support file patterns and directory collection
- **Permission Handling**: Use buildfab's uid/gid or user-specified uid:gid
- **Output Directory**: Specify host directory for artifact collection

### 2.7 Error Handling
- **Standard Reporting**: Use standard action error reporting with verbosity levels
- **Debug Information**: Show full Docker command with `-vv` verbosity
- **Engine Cleanup**: Let Docker/Podman handle container cleanup
- **Reproduction**: Provide Docker commands for manual reproduction

### 2.8 Resource Management
- **CPU Limits**: Support CPU count and CPU set specification
- **Memory Limits**: Support human-readable memory limits (500M, 2G)
- **User Configuration**: Users configure all resource limits
- **No Defaults**: No automatic resource limit defaults

## 3. Integration Requirements

### 3.1 Matrix Feature Integration
- **Independent Features**: Matrix and container features are independent
- **Variable Support**: Use variables for integration between features
- **No Special Integration**: No special integration code required

### 3.2 Cross-Platform Support
- **Linux Focus**: Focus on Linux containers initially
- **Windows Support**: Not required initially (roadmap item)
- **Platform Detection**: Use existing platform detection system
- **Testing Engine**: Use Podman for testing and examples (no superuser access required)

## 4. Implementation Approach

### 4.1 Incremental Development
1. **Phase 1**: Run container on Alpine/Debian and execute action/stage/run
2. **Phase 2**: Add mount support for directories and files
3. **Phase 3**: Add CPU and memory limiting capabilities
4. **Phase 4**: Add container image building and execution

### 4.2 Code Organization
- **Container Package**: Create `pkg/buildfab/container/` package
- **Engine Interface**: Abstract interface for Docker/Podman
- **Configuration Integration**: Integrate with existing configuration system
- **No External Dependencies**: Use standard Go libraries only

### 4.3 Testing Strategy
- **Image Building**: Build test images for container testing
- **Resource Testing**: Test CPU limiting and memory constraints
- **Mount Testing**: Test directory mounting with ro/rw options
- **Artifact Testing**: Test artifact collection and output
- **Integration Testing**: Test with real containers and buildfab execution

## 5. Documentation Requirements

### 5.1 User Documentation
- **Option Descriptions**: Document all container configuration options
- **Examples**: Provide common container use case examples
- **Troubleshooting**: Allow users to run Docker manually for debugging
- **Command Reproduction**: Provide Docker commands for error reproduction

### 5.2 CLI Integration
- **No Special Commands**: No special container management commands initially
- **Standard Help**: Use standard help and examples
- **Configuration Validation**: Validate container configuration via CLI

## 6. Success Criteria

### 6.1 Functional Requirements
- [ ] Execute actions inside Docker containers
- [ ] Execute stages inside Docker containers
- [ ] Execute custom commands inside containers
- [ ] Support Docker and Podman engines
- [ ] Mount host directories into containers
- [ ] Collect artifacts from containers
- [ ] Support CPU and memory limits
- [ ] Build container images from Dockerfiles

### 6.2 Quality Requirements
- [ ] 90%+ test coverage for container features
- [ ] Comprehensive error handling and reporting
- [ ] Clear documentation with examples
- [ ] Integration with existing buildfab architecture
- [ ] No external dependencies beyond Docker/Podman

### 6.3 Performance Requirements
- [ ] Efficient container startup and execution
- [ ] Proper resource management
- [ ] Clean container cleanup
- [ ] Minimal overhead compared to local execution

## 7. Risk Mitigation

### 7.1 Technical Risks
- **Container Engine Compatibility**: Test with multiple Docker/Podman versions
- **Platform Issues**: Focus on Linux initially, add other platforms later
- **Resource Management**: Implement proper resource limits and cleanup
- **Error Handling**: Provide clear error messages and reproduction steps

### 7.2 Implementation Risks
- **Complexity**: Implement incrementally with clear milestones
- **Testing**: Maintain high test coverage throughout development
- **Documentation**: Keep documentation updated with implementation
- **User Adoption**: Provide clear examples and migration guides

## 8. Future Enhancements (Roadmap)

### 8.1 Phase 2 Enhancements
- **Windows Container Support**: Add Windows container execution
- **Private Registry Support**: Add authentication for private registries
- **Volume Driver Support**: Add custom volume driver support
- **Advanced Security**: Add security scanning and policies

### 8.2 Phase 3 Enhancements
- **Container Orchestration**: Add support for Kubernetes, Docker Swarm
- **Container Networking**: Add advanced networking options
- **Container Monitoring**: Add container health and performance monitoring
- **Container Caching**: Add container layer caching and optimization

## 9. Implementation Timeline

### 9.1 Phase 1: Basic Container Execution (Weeks 1-2)
- **Week 1**: Basic container execution with Alpine/Debian images
- **Week 2**: Action and stage execution inside containers

### 9.2 Phase 2: Mount and Resource Support (Weeks 3-4)
- **Week 3**: Mount support for directories and files
- **Week 4**: CPU and memory limiting capabilities

### 9.3 Phase 3: Image Building and Advanced Features (Weeks 5-6)
- **Week 5**: Container image building from Dockerfiles
- **Week 6**: Artifact collection and advanced configuration

### 9.4 Phase 4: Testing and Documentation (Weeks 7-8)
- **Week 7**: Comprehensive testing and integration
- **Week 8**: Documentation and user guides

## 10. Conclusion

The container feature will significantly enhance buildfab's capabilities by providing isolated execution environments for build and test processes. The incremental implementation approach ensures manageable development cycles while building toward a powerful, integrated automation platform.

The clarified requirements provide a clear roadmap for implementation, with specific technical details, success criteria, and risk mitigation strategies. The feature will integrate seamlessly with existing buildfab functionality while providing users with powerful container-based automation capabilities.

---

**Document Status**: Final Requirements Specification
**Version**: 1.0
**Date**: 2025-01-27
**Next Review**: After implementation completion
