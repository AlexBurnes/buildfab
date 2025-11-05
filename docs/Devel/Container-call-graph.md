# Container Feature Call Graph

This document provides a detailed call graph analysis of the current container feature implementation in buildfab, showing how container actions are executed through the system.

## Overview

The container feature allows buildfab to execute actions inside Docker or Podman containers, providing isolated execution environments for build tasks.

## Call Graph for Stage Execution

### 1. CLI Entry Point (`buildfab run test-container`)

```
cmd/buildfab/main.go
├── runRoot()
│   └── runStageDirect()
│       ├── buildfab.LoadConfig() - Load YAML configuration
│       ├── displayVersionInfo() - Show version info
│       ├── buildfab.NewSimpleRunner() - Create simple runner
│       └── runner.RunStage(ctx, "test-container")
```

### 2. SimpleRunner.RunStage() Flow

```
pkg/buildfab/simple.go
└── SimpleRunner.RunStage(ctx, stageName)
    ├── r.config.GetStage(stageName) - Get stage configuration
    ├── r.expandMatrix() - Expand matrix configurations
    ├── r.validateStage() - Validate stage configuration
    └── For each matrix combination:
        └── r.RunStageInternal(ctx, stage, matrixVars)
            ├── r.expandActions() - Expand action references
            ├── r.buildDAG() - Build dependency graph
            └── r.executeDAG(ctx, dag) - Execute actions in dependency order
```

### 3. DAG Execution Flow

```
pkg/buildfab/simple.go
└── SimpleRunner.executeDAG(ctx, dag)
    ├── For each action in topological order:
    │   └── r.RunAction(ctx, action)
    │       └── SimpleRunner.RunAction(ctx, action)
    │           ├── r.validateAction() - Validate action
    │           ├── r.expandVariables() - Expand variables
    │           └── r.RunActionInternal(ctx, action)
    │               └── r.runCustomActionForDAG(ctx, action)
```

### 4. Container Action Execution

```
pkg/buildfab/buildfab.go
└── Runner.runCustomActionForDAG(ctx, action)
    ├── Check if action.Container != nil
    │   └── YES: r.runContainerAction(ctx, action)
    │       ├── container.NewRunner(action.Container.Engine) - Create container runner
    │       ├── runner.Prepare(ctx, *action.Container) - Prepare container
    │       │   ├── Copy buildfab binary to container
    │       │   ├── Copy configuration files
    │       │   └── Set up working directory
    │       ├── runner.RunAction(ctx, *action.Container) - Execute action
    │       │   ├── Execute command in container
    │       │   ├── Stream output
    │       │   └── Return result
    │       ├── Display container output (if verbose)
    │       ├── Display container errors
    │       └── Return Result with status
    └── NO: Execute regular shell command
        └── r.runShellAction(ctx, action)
```

## Container Components Architecture

### 1. Container Types and Interfaces

```
pkg/buildfab/container/
├── types.go - Container configuration structures
│   ├── Container struct - Main container configuration
│   ├── Engine enum - Docker/Podman selection
│   ├── Mount struct - Volume mount configuration
│   └── Result struct - Container execution result
├── engine.go - Container engine interface
│   └── Engine interface - Abstract container operations
└── engines.go - Engine implementations
    ├── DockerEngine - Docker implementation
    └── PodmanEngine - Podman implementation
```

### 2. Container Manager

```
pkg/buildfab/container/manager.go
└── Manager struct
    ├── engines map[Engine]Engine - Available engines
    ├── NewRunner(engine Engine) - Create container runner
    └── GetEngine(engine Engine) - Get engine instance
```

### 3. Container Runner

```
internal/container/runner.go
└── Runner struct
    ├── Prepare(ctx, container Container) - Prepare container environment
    ├── RunAction(ctx, container Container) - Execute action in container
    └── Cleanup(ctx, container Container) - Clean up container resources
```

## Example: container-simple-test.yml Execution

### Configuration Analysis

```yaml
stages:
  test-container:
    actions:
      - name: hello-world
        container:
          image: alpine:latest
          engine: docker
          mount:
            source: .
            target: /workspace
          working_dir: /workspace
        run: echo "Hello from container!"
```

### Execution Flow

1. **CLI Parsing**: `buildfab run test-container`
2. **Config Loading**: Load `examples/container-simple-test.yml`
3. **Stage Resolution**: Find `test-container` stage
4. **Action Expansion**: Process `hello-world` action
5. **Container Detection**: `action.Container != nil` → container execution
6. **Engine Selection**: `docker` engine selected
7. **Container Preparation**:
   - Pull `alpine:latest` image
   - Mount current directory to `/workspace`
   - Set working directory to `/workspace`
8. **Action Execution**: Run `echo "Hello from container!"` in container
9. **Output Processing**: Stream output and return result
10. **Cleanup**: Remove temporary container

## Current Implementation Status

### ✅ Implemented Components

1. **Container Data Structures** - Complete in `pkg/buildfab/container/types.go`
2. **Container Engine Interface** - Complete in `pkg/buildfab/container/engine.go`
3. **Docker Engine** - Complete in `pkg/buildfab/container/engines.go`
4. **Podman Engine** - Complete in `pkg/buildfab/container/engines.go`
5. **Container Manager** - Complete in `pkg/buildfab/container/manager.go`
6. **Container Runner** - Basic structure in `internal/container/runner.go`
7. **Container Integration** - Integrated into `runCustomActionForDAG`
8. **Basic Tests** - Complete in `tests/container/basic_test.go`

### 🔧 Current Issues

1. **Method Signature Mismatch**: `runContainerAction` returns `error` but should return `(Result, error)`
2. **Missing Container Preparation**: `prepareContainer` method not fully implemented
3. **Missing Artifact Collection**: `collectArtifacts` method not implemented
4. **Incomplete Error Handling**: Container errors not properly propagated

### 🚧 Next Implementation Steps

1. **Fix Method Signatures**: Update `runContainerAction` to return `(Result, error)`
2. **Implement Container Preparation**: Complete `prepareContainer` method
3. **Implement Artifact Collection**: Complete `collectArtifacts` method
4. **Enhance Error Handling**: Improve error propagation and user feedback
5. **Add Integration Tests**: Test container actions end-to-end
6. **Documentation Updates**: Update user documentation with container examples

## Testing Strategy

### Unit Tests
- Container engine implementations
- Container runner methods
- Configuration validation

### Integration Tests
- End-to-end container execution
- Multi-stage container workflows
- Error handling scenarios

### Example Tests
- `examples/container-simple-test.yml`
- `examples/container-debug-test.yml`
- `examples/container-matrix-platform-test.yml`

## Error Handling Flow

```
Container Execution Error Handling:
├── Container Engine Errors
│   ├── Image pull failures
│   ├── Container creation failures
│   └── Engine not available
├── Container Runtime Errors
│   ├── Command execution failures
│   ├── Permission errors
│   └── Resource limit exceeded
├── Configuration Errors
│   ├── Invalid container configuration
│   ├── Missing required fields
│   └── Invalid mount paths
└── System Errors
    ├── Docker/Podman not installed
    ├── Insufficient permissions
    └── Network connectivity issues
```

## Performance Considerations

1. **Container Startup Time**: Minimize container creation overhead
2. **Image Caching**: Leverage Docker/Podman image caching
3. **Resource Usage**: Monitor container resource consumption
4. **Parallel Execution**: Support parallel container execution where possible

## Security Considerations

1. **Container Isolation**: Ensure proper container isolation
2. **Volume Mounts**: Validate mount paths for security
3. **User Permissions**: Run containers with appropriate user permissions
4. **Network Access**: Control container network access

## Future Enhancements

1. **Container Orchestration**: Support for multi-container workflows
2. **Advanced Mounting**: Support for complex volume mount configurations
3. **Container Templates**: Predefined container configurations
4. **Performance Monitoring**: Container execution metrics and monitoring
5. **Custom Container Images**: Support for custom buildfab container images
