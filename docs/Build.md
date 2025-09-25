# Build System Documentation

This document describes the build system, packaging, and release process for buildfab.

## Build System Overview

buildfab uses a multi-layered build system:

1. **CMake**: Cross-platform build configuration
2. **Conan**: Go toolchain and dependency management
3. **GoReleaser**: Automated release and packaging
4. **GitHub Actions**: CI/CD pipeline

## Prerequisites

### Required Tools
- **CMake 3.16+**: Cross-platform build system
- **Conan 2.0+**: C/C++ package manager
- **Go 1.22+**: Go compiler and toolchain
- **Git**: Version control

### Optional Tools
- **golangci-lint**: Code linting and quality checks
- **goimports**: Import organization
- **gofmt**: Code formatting

## Build Process

### 1. Self-Building with buildfab (Recommended)

The buildfab project can build itself using its own configuration with automatic tool checking and installation:

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

- **`pre-check`**: Verify all required tools (conan, cmake, goreleaser, go, version utility, pre-push utility) are installed
- **`pre-install`**: Install missing tools automatically
- **`build`**: Build the project with all dependencies using CMake/Conan
- **`test`**: Run cross-platform tests
- **`release`**: Create release artifacts and packages

### 2. Manual CMake + Conan Build

```bash
# Create build directory
mkdir build && cd build

# Configure with CMake
cmake ..

# Build the project
cmake --build .

# Run tests
cmake --build . --target test
```

### 3. Direct Go Build

```bash
# Build CLI application
go build -o buildfab cmd/buildfab/main.go

# Build with specific flags
go build -ldflags="-s -w" -o buildfab cmd/buildfab/main.go

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o buildfab-linux-amd64 cmd/buildfab/main.go
```

### 3. Cross-Platform Builds

```bash
# Build for all supported platforms
./buildtools/build-and-package.sh

# Build specific platform
GOOS=windows GOARCH=amd64 go build -o buildfab.exe cmd/buildfab/main.go
GOOS=darwin GOARCH=arm64 go build -o buildfab-darwin-arm64 cmd/buildfab/main.go
```

## Build Configuration

### CMakeLists.txt

The main CMake configuration includes:
- Go toolchain detection
- Conan integration
- Cross-platform support
- Test configuration
- Installation rules

### Conan Configuration

#### conanfile.py
- Go toolchain requirements
- Build dependencies
- Cross-platform support

#### conanfile-golang.py
- Go-specific Conan profile
- Toolchain configuration
- Platform-specific settings

### GoReleaser Configuration

#### .goreleaser.yml
- Multi-platform builds
- Archive creation
- Package manager integration
- Release automation

## Packaging

### Linux Packaging

#### tar.gz Archives
- Created by GoReleaser
- Include install.sh script
- Support for multiple architectures

#### install.sh Script
```bash
# Install buildfab
curl -sSL https://github.com/AlexBurnes/buildfab/releases/latest/download/install.sh | bash
```

### Windows Packaging

#### Scoop Manifest
- Located in `packaging/windows/scoop-bucket/`
- Automatic updates via GitHub Actions
- Easy installation via Scoop

```powershell
# Install via Scoop
scoop bucket add buildfab https://github.com/AlexBurnes/buildfab-scoop-bucket
scoop install buildfab
```

### macOS Packaging

#### Homebrew Formula
- Future support planned
- Automatic formula updates
- Easy installation via Homebrew

```bash
# Future installation via Homebrew
brew install buildfab
```

## Release Process

### 1. Pre-Release Checklist

- [ ] Version incremented in VERSION file
- [ ] CHANGELOG.md updated
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Memory bank updated
- [ ] GoReleaser dry-run successful

### 2. Release Steps

```bash
# 1. Check version status
./scripts/check-version-status

# 2. Increment version (if needed)
./scripts/version-bump minor

# 3. Update changelog
# (Edit CHANGELOG.md manually)

# 4. Commit changes
git add .
git commit -m "chore: release v$(cat VERSION)"

# 5. Create and push tag
git tag $(cat VERSION)
git push origin master
git push origin $(cat VERSION)

# 6. Trigger release
./buildtools/build-goreleaser.sh release
```

### 3. Automated Release

GitHub Actions automatically:
- Builds for all platforms
- Creates release archives
- Updates package managers
- Publishes to GitHub Releases

## Build Scripts

### buildtools/build-and-package.sh
- Main build script
- Cross-platform compilation
- Package creation

### buildtools/build-goreleaser.sh
- GoReleaser wrapper
- Dry-run and release modes
- Configuration validation

### buildtools/build-conan.sh
- Conan-specific build
- Dependency management
- Toolchain setup

## Development Builds

### Local Development
```bash
# Quick build for testing
go build -o buildfab cmd/buildfab/main.go

# Build with debug info
go build -gcflags="all=-N -l" -o buildfab cmd/buildfab/main.go
```

### CI/CD Builds
- Automated testing on every push
- Cross-platform builds on tags
- Release automation
- Package manager updates

## Troubleshooting

### Common Build Issues

1. **Conan not found**:
   ```bash
   pip install conan
   conan --version
   ```

2. **Go version mismatch**:
   ```bash
   go version
   # Update to Go 1.22+ if needed
   ```

3. **CMake configuration fails**:
   ```bash
   rm -rf build
   mkdir build && cd build
   cmake ..
   ```

4. **GoReleaser fails**:
   ```bash
   ./buildtools/build-goreleaser.sh dry-run
   ```

### Build Optimization

- **Static binaries**: Use `-ldflags="-s -w"` for smaller binaries
- **Cross-compilation**: Set GOOS and GOARCH environment variables
- **Parallel builds**: Use `-j` flag with cmake --build
- **Caching**: Use Conan cache for faster builds

## Performance Considerations

### Build Time Optimization
- Use Conan cache for dependencies
- Parallel compilation where possible
- Incremental builds with CMake
- Go module caching

### Binary Size Optimization
- Strip debug symbols: `-ldflags="-s -w"`
- Use UPX for compression (optional)
- Remove unused dependencies
- Optimize for target platform

## Security Considerations

### Build Security
- Use official Go toolchain
- Verify Conan package integrity
- Sign release artifacts
- Use secure build environments

### Distribution Security
- Provide checksums for downloads
- Use HTTPS for all downloads
- Sign release tags
- Verify package manager integrity