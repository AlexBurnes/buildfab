# Container Artifact Collection Implementation

## Overview

This document describes the hybrid artifact collection implementation for the buildfab container feature. The implementation preserves full path structure for collected artifacts.

## Implementation Strategy: Hybrid Approach

The artifact collection uses different strategies depending on the container execution scenario:

### Scenario 1: Running Containers (`image.from` + `run`)
**Strategy**: Pre-mounted volume approach

- Mount host output directory as `/buildfab-artifacts` in container
- Add automatic copy commands at end of `run` script
- Artifacts appear directly on host filesystem
- **No `docker cp` overhead** - files are written directly to host

### Scenario 2: Built Images (`image.build` without `run`)
**Strategy**: Docker/Podman `cp` approach

- Create temporary container from built image
- Use `docker cp` / `podman cp` to extract artifacts
- Clean up temporary container
- Works without running the container

## Path Structure Preservation

**All artifact paths preserve their full directory structure:**

```
Artifact Path              Output Directory    Final Location
/app/binary           →    ./dist         →    ./dist/app/binary
/usr/local/bin/myapp  →    ./dist         →    ./dist/usr/local/bin/myapp
/build/out/file.txt   →    ./artifacts    →    ./artifacts/build/out/file.txt
```

## Configuration

```yaml
actions:
  - name: build-with-artifacts
    container:
      image:
        from: golang:1.23
      workdir: /src
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
      run: |
        go build -o /app/myapp ./cmd/myapp
        cp config.yaml /app/config.yaml
      artifacts:
        output: ./dist              # Host directory for artifacts
        path:
          - /app/myapp              # Will be saved as: ./dist/app/myapp
          - /app/config.yaml        # Will be saved as: ./dist/app/config.yaml
```

## Implementation Details

### For Run Commands (Pre-mounted Volume)

```go
// 1. PrepareContainerConfig adds artifact mount
artifactMount := container.ContainerMount{
    Type:   "bind",
    Source: absOutput,           // ./dist
    Target: "/buildfab-artifacts",
    RO:     false,
}

// 2. Add copy commands to run script (preserves full path)
For artifact "/app/binary" or ".rpmbuild/RPMS/*/*.rpm":
  cp --parents -r /app/binary /buildfab-artifacts/
  cp --parents -r .rpmbuild/RPMS/*/*.rpm /buildfab-artifacts/
  
// The --parents flag automatically creates parent directory structure
// This works correctly with wildcard patterns like * and **
```

### For Built Images (Docker CP)

```go
// 1. Create temporary container from image
docker create --name buildfab-artifact-extract-<pid> <image-tag>

// 2. Copy each artifact preserving path structure
For artifact "/app/binary":
  destPath = filepath.Join("./dist", "app/binary")
  docker cp buildfab-artifact-extract-<pid>:/app/binary ./dist/app/binary

// 3. Clean up
docker rm buildfab-artifact-extract-<pid>
```

## Usage Examples

### Example 1: Build and Collect Binary

```yaml
actions:
  - name: build-go-binary
    container:
      image:
        from: golang:1.23-alpine
      workdir: /build
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
```

**Result**: Binary saved as `./dist/usr/local/bin/myapp`

### Example 2: Build from Dockerfile and Extract Artifacts

```yaml
actions:
  - name: build-and-extract
    container:
      image:
        build:
          dockerfile: Dockerfile
          context: .
          tags:
            - myapp:latest
      artifacts:
        output: ./artifacts
        path:
          - /app/myapp
          - /app/config/
          - /app/docs/*.pdf
```

**Result**: 
- `./artifacts/app/myapp`
- `./artifacts/app/config/` (directory with all contents)
- `./artifacts/app/docs/*.pdf` (all PDF files)

### Example 3: Multi-Stage Build with Artifacts

```yaml
actions:
  - name: compile-and-package
    container:
      image:
        from: gcc:latest
      workdir: /src
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
      run: |
        mkdir -p /build/bin /build/lib
        gcc -o /build/bin/myapp src/main.c
        cp lib/*.so /build/lib/
      artifacts:
        output: ./release
        path:
          - /build/bin/myapp
          - /build/lib/
```

**Result**:
- `./release/build/bin/myapp`
- `./release/build/lib/` (all .so files)

## Error Handling

- **Missing artifacts**: Commands use `|| true` to avoid failing on missing artifacts
- **Permission issues**: Artifacts copied with container user permissions
- **Path conflicts**: Full path preservation eliminates naming conflicts

## Benefits

### Full Path Preservation
- ✅ No naming conflicts between files from different directories
- ✅ Clear understanding of original location
- ✅ Easy to organize artifacts by their container paths

### Hybrid Approach
- ✅ Efficient for run commands (no docker cp overhead)
- ✅ Works for build-only images
- ✅ Automatic selection based on configuration

### Cross-Platform
- ✅ Works with both Docker and Podman
- ✅ Same configuration for all platforms
- ✅ Consistent behavior across engines

## Wildcard Pattern Support

Artifact paths support wildcard patterns for flexible file collection:

### Supported Patterns

- **Single wildcard**: `*.rpm`, `*.log`, `*.txt`
- **Directory wildcard**: `dist/*/binary`, `.rpmbuild/RPMS/*/*.rpm`
- **Multi-level wildcard**: `build/**/*.so`

### Example with Wildcards

```yaml
actions:
  - name: collect-rpms
    container:
      image:
        from: fedora:latest
      run: |
        rpmbuild -ba myapp.spec
      artifacts:
        output: ./rpms
        path:
          - .rpmbuild/RPMS/*/*.rpm      # Collects all RPM files
          - .rpmbuild/SRPMS/*.src.rpm   # Collects all SRPM files
```

**Result**:
- `./rpms/.rpmbuild/RPMS/x86_64/myapp-1.0-1.x86_64.rpm`
- `./rpms/.rpmbuild/RPMS/noarch/myapp-docs-1.0-1.noarch.rpm`
- `./rpms/.rpmbuild/SRPMS/myapp-1.0-1.src.rpm`

## Technical Notes

- Output directory is created automatically if it doesn't exist
- Default output directory is `./artifacts` if not specified
- Artifact collection runs after container execution completes
- For run commands, artifacts are collected via pre-mounted volume (no post-execution copy needed)
- For build-only images, artifacts are extracted using docker/podman cp command
- All paths are cleaned and normalized before copying
- Wildcard patterns are fully supported using `cp --parents` for proper path preservation

## Future Enhancements

Potential future improvements:

1. **Compression**: Optional artifact compression before transfer
2. **Streaming**: Stream artifacts during execution instead of after
3. **Filters**: Exclude patterns to avoid collecting unwanted files
4. **Artifact metadata**: Collect file permissions, timestamps, and checksums

