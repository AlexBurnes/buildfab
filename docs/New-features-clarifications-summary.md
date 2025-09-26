# New Features Clarifications Summary

## Overview

This document summarizes the key clarifications provided for the Matrix, Container, and Caching features implementation. These clarifications refine the implementation approach and provide specific constraints and requirements.

## Matrix Feature Clarifications

### Core Constraints
- **Stage Level Only**: Matrix supported at stage level only (not action level)
- **Single Dimension**: Only single matrix dimension supported initially (no nested matrices)
- **Parse Time Expansion**: Matrix expansion happens at parse time, not runtime
- **Empty Values**: Empty matrix values are ignored
- **No Job Limits**: No limit on total number of jobs generated

### Strategy Behavior
- **fail_fast**: Stops **all jobs** on first failure (not just prevents new ones)
- **continue_on_error**: Interaction with fail_fast follows Matrix-shedules.md semantics
- **Order Options**: Additional options (reverse, priority) added to roadmap

### CLI Integration
- **No Separate Command**: Matrix runs as part of normal `buildfab <stage>` execution
- **No Feature Flags**: No CLI flags to disable features
- **Override Support**: `--matrix.test=test_one --max-parallel=1` still supported

## Container Feature Clarifications

### Engine Support
- **Engine Preference**: Docker first, then Podman, then other engines
- **Additional Engines**: containerd, LXC added to roadmap
- **Version Compatibility**: Handled when cases arise (not pre-planned)

### Configuration Approach
- **Action Property**: Container is a property of actions (not separate action type)
- **No Global Config**: No global container configuration for defaults
- **Error Handling**: Container-specific errors shown as job output

### Buildfab Integration
- **Pre-installed**: Buildfab must be pre-installed in container images
- **Version Specific**: Use specific version install scripts (e.g., v0.16.5)
- **System Paths**: Follow LBS standards, install to system bin directories
- **No Custom Paths**: No way to specify custom buildfab binary path

## Caching Feature Clarifications

### Cache Management
- **Directory-Based Only**: No plugin system, only directory-based caches
- **User Configuration**: User decides and configures all cache types
- **Separate Actions**: Cache management via separate action steps
- **No Global Config**: No global cache configuration for defaults

### Cache Operations
- **Cache Keys**: User configures using variables `${{ var.name }}`
- **Cache Cleanup**: Separate action steps, user decides and configures
- **Manual Management**: Separate action steps, user decides and configures
- **Access Control**: Bind mount options (ro, rw) for permissions

### Cache Types
- **Supported Types**: ccache, Conan, vcpkg, Go modules, npm, pip
- **Detection**: Not buildfab's responsibility, handled by external scripts
- **Disabling**: User decides by configuration

## Integration Clarifications

### Feature Integration
- **Matrix + Container**: Matrix jobs run in containers via run_stage or run actions
- **Matrix + Caching**: Isolated caches using `${{ matrix.name }}` variables
- **Container + Caching**: User-configured container mounts
- **All Three**: User decides and configures all integrations

### Configuration Philosophy
- **No Feature Disabling**: Features enabled by configuration only
- **Feature Conflicts**: Result in failure (no graceful degradation)
- **User Control**: User decides and configures all feature integrations
- **No Unified Config**: Features work independently

## Resource Management Clarifications

### Resource Limits
- **Global Limits**: Global resource limits (not per-job limits)
- **CPU-Based**: max_parallel must not exceed CPU count
- **Memory Throttling**: Don't run new jobs if memory exceeds configured limit or 100%
- **No Monitoring**: No performance monitoring required initially

### Resource Management
- **Memory Limits**: Configurable memory limits for jobs
- **CPU Limits**: Configurable CPU limits for jobs
- **Disk Limits**: Configurable disk limits for caches
- **Network Limits**: Configurable network limits for containers

## Security Clarifications

### Container Security
- **User Settings**: User and group settings in action configuration
- **No Scanning**: No security scanning for container images initially
- **Privileged Containers**: Handled via user and group settings
- **Security Policies**: Added to roadmap

### Cache Security
- **Access Control**: Bind mount options (ro, rw) for permissions
- **No Encryption**: No encryption for sensitive cache data initially
- **No Audit Logging**: No audit logging for cache operations initially

## Testing Clarifications

### Test Coverage
- **Comprehensive Testing**: As much test coverage as possible
- **Container Testing**: Test container functionality with Podman
- **No Performance Benchmarks**: No performance benchmarks for caching initially
- **Test Data**: Will be clarified during implementation

### Test Scenarios
- **Matrix Tests**: Various matrix configurations and edge cases
- **Container Tests**: Different base images and configurations
- **Cache Tests**: Different cache types and scenarios
- **Integration Tests**: Feature combination scenarios

## Documentation Clarifications

### User Documentation
- **Separate Docs**: Yes for matrix and containers
- **Integration Docs**: Link to main documentation and in README
- **No Video Tutorials**: No video tutorials or interactive examples

### Developer Documentation
- **Separate Docs**: Yes for each feature
- **Cross-Feature Docs**: In one document with examples for platforms
- **Design Docs**: Optional

## Implementation Impact

### Simplified Approach
- **User-Driven**: All features are user-configured and user-controlled
- **No Magic**: No automatic detection or configuration
- **Explicit Control**: Users must explicitly configure what they want
- **Failure on Conflict**: Clear failure when features conflict

### Development Benefits
- **Clearer Requirements**: Specific constraints reduce ambiguity
- **Focused Implementation**: Single dimension matrix simplifies initial implementation
- **User Responsibility**: Less complex error handling and fallback logic
- **Incremental Features**: Can implement features independently

### User Benefits
- **Predictable Behavior**: Clear, explicit configuration
- **Full Control**: Users control all aspects of feature usage
- **Simple Integration**: Features work together through user configuration
- **Clear Failure Modes**: Obvious when and why things fail

## Next Steps

1. **Begin Phase 1**: Start Matrix feature implementation with single dimension
2. **Follow Constraints**: Implement according to clarified requirements
3. **User Testing**: Get user feedback on configuration approach
4. **Iterate**: Refine based on implementation experience
5. **Documentation**: Create user guides based on actual implementation

## Conclusion

These clarifications provide a clear, user-driven approach to implementing the Matrix, Container, and Caching features. The simplified approach reduces complexity while giving users full control over feature usage and integration. The implementation can now proceed with clear requirements and constraints.
