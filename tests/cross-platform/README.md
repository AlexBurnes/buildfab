# Cross-Platform Buildfab Detection Tests

This directory contains comprehensive cross-platform testing for the buildfab utility's platform detection functionality.

## Overview

The platform detection feature provides variables that can be used in buildfab actions to detect:
- `${{ platform }}` - Current platform (GOOS value)
- `${{ arch }}` - Current architecture (GOARCH value) 
- `${{ os }}` - Current operating system (distribution name)
- `${{ os_version }}` - Current OS version
- `${{ cpu }}` - Number of logical CPUs

## Validation Testing

The cross-platform tests now include **proper validation** that compares detected values against expected values and fails if they don't match:

### Linux Tests
- **Platform**: Must be `linux`
- **Architecture**: Must be `amd64`
- **OS**: Must be `ubuntu` or `debian`
- **OS Version**: Must be numeric (e.g., `24.04`, `12`)
- **CPU**: Must be >= 1

### Windows Tests (via Wine)
- **Platform**: Must be `windows`
- **Architecture**: Must be `amd64`
- **OS**: Must be `windows`
- **OS Version**: Must contain `windows`
- **CPU**: Must be >= 1

### Windows Shell Configuration Tests
- **cmd.exe Validation**: Tests Windows cmd.exe shell execution under Wine
- **Shell Configuration**: Demonstrates buildfab's shell configuration feature
- **Variable Interpolation**: Validates platform variables work with Windows cmd syntax

### macOS Tests
- **Platform**: Must be `darwin`
- **Architecture**: Must be `amd64` or `arm64`
- **OS**: Must be `darwin`
- **OS Version**: Must be numeric (e.g., `14.0`, `15.0`)
- **CPU**: Must be >= 1

## Shell Configuration

You can specify different shells for your actions using the `shell` field:

```yaml
actions:
  - name: bash-script
    shell: bash          # Use bash (auto-adds .exe on Windows)
    run: |
      echo "Using bash"
      
  - name: powershell-script
    shell: powershell    # Use PowerShell (auto-adds .exe on Windows)
    run: |
      Write-Host "Using PowerShell"
      
  - name: cmd-script
    shell: cmd           # Use Windows cmd (auto-adds .exe on Windows)
    run: |
      echo "Using cmd"
      
  - name: default-shell
    # No shell specified - uses platform default
    run: |
      echo "Using platform default shell"
```

### Platform Defaults:
- **Linux/macOS**: `sh`
- **Windows**: `bash` (if Git Bash available), otherwise `cmd`

### Shell Arguments:
- **bash/sh/zsh/fish**: `-euc` (exit on error, unset variables, compact output)
- **powershell**: `-NoProfile -Command`
- **cmd**: `/C`

### Error Handling:
If a specified shell is not found, buildfab will show a clear error message:
```
shell 'zsh' not found in PATH. Please install it or use a different shell
```

## Test Strategies

### 1. Container-based Testing (Recommended)

Uses container runtimes (Podman or Docker) to test on different Linux distributions and simulate Windows using Wine.

#### Prerequisites
- Container runtime installed (Podman preferred, Docker supported)
- All platform binaries built (`buildfab-linux-amd64`, `buildfab-windows-amd64.exe`, `buildfab-darwin-amd64`)

**Container Runtime Options:**
- **Podman (Recommended)**: Rootless, no sudo required, more secure
- **Docker**: May require sudo, traditional container runtime

#### Running Tests

```bash
# Build all platform binaries first
go build -o bin/buildfab-linux-amd64 ./cmd/buildfab
GOOS=windows GOARCH=amd64 go build -o bin/buildfab-windows-amd64.exe ./cmd/buildfab
GOOS=darwin GOARCH=amd64 go build -o bin/buildfab-darwin-amd64 ./cmd/buildfab

# Run tests with buildfab (auto-detects Podman/Docker)
buildfab test

# Or run individual platform tests manually
# Tests will auto-detect container runtime (Podman preferred)

# Ubuntu test
docker build -f tests/cross-platform/Dockerfile.linux-ubuntu -t buildfab-test-ubuntu .
docker run --rm buildfab-test-ubuntu

# Debian test  
docker build -f tests/cross-platform/Dockerfile.linux-debian -t buildfab-test-debian .
docker run --rm buildfab-test-debian

# Windows test
docker build -f tests/cross-platform/Dockerfile.windows -t buildfab-test-windows .
docker run --rm buildfab-test-windows

# macOS testing requires macOS host - containers not supported
echo "macOS testing requires actual macOS host or VM"
```

#### Supported Platforms
- **Ubuntu 24.04** - Tests Linux distribution detection via containers (Podman/Docker)
- **Debian 12** - Tests Linux distribution detection via containers (Podman/Docker)
- **Windows** - Tests Windows platform detection using Wine via containers (Podman/Docker)
- **macOS** - Tests macOS platform detection directly (requires macOS host)

### 2. Direct Platform Testing

Test buildfab platform detection directly on each platform:

```bash
# Test on current platform
./bin/buildfab -c tests/cross-platform/linux_configuration.yml check-detection
./bin/buildfab -c tests/cross-platform/windows_configuration.yml check-detection
./bin/buildfab -c tests/cross-platform/macos_configuration.yml check-detection
```

### 3. GitHub Actions Testing

Automated testing on GitHub Actions runners:
- Ubuntu latest
- Windows latest  
- macOS latest

## Expected Results

### Linux (Ubuntu/Debian)
```
=== Buildfab Platform Detection Validation Test ===
Testing on: Ubuntu 24.04.3 LTS

=== Validation Results ===
✅ Platform: linux (CORRECT)
✅ Architecture: amd64 (CORRECT)
✅ OS: ubuntu (CORRECT - valid Linux distribution)
✅ OS Version: 24.04 (CORRECT - valid version format)
✅ CPU: 8 cores (CORRECT - >= 1)

=== All Platform Detection Validations Passed! ===
✅ Platform detection test completed successfully
```

### Windows
```
=== Buildfab Platform Detection Test ===
Testing on Windows platform

Platform: windows
Architecture: amd64
OS: windows
OS Version: windows
CPU: 8

=== Platform Detection Variables Test ===
Testing variable interpolation with platform variables...
Current platform is windows with architecture amd64
Running on windows version windows with 8 CPU cores

✅ Platform detection test completed successfully
```

### macOS
```
=== Buildfab Platform Detection Test ===
Testing on macOS platform

Platform: darwin
Architecture: amd64 (or arm64)
OS: darwin
OS Version: darwin
CPU: 8

=== Platform Detection Variables Test ===
Testing variable interpolation with platform variables...
Current platform is darwin with architecture amd64
Running on darwin version darwin with 8 CPU cores

✅ Platform detection test completed successfully
```

## Test Files

### Dockerfiles
- `Dockerfile.linux-ubuntu` - Ubuntu 24.04 test environment with buildfab
- `Dockerfile.linux-debian` - Debian 12 test environment with buildfab
- `Dockerfile.windows` - Windows test environment using Wine with buildfab
- `Dockerfile.windows-with-git-bash` - Windows shell configuration test environment
- `Dockerfile.macos` - macOS test environment (NOT SUPPORTED - requires macOS host)

### Platform Configuration Files
- `linux_configuration.yml` - Linux platform detection test configuration
- `windows_configuration.yml` - Windows platform detection test configuration
- `windows-wine_configuration.yml` - Windows testing under Wine emulation
- `windows-git-bash_configuration.yml` - Windows shell configuration testing
- `macos_configuration.yml` - macOS platform detection test configuration

### Test Scripts
- `test-platform.sh` - Linux platform detection tests using buildfab
- `test-platform-windows.sh` - Windows platform detection tests using buildfab
- `test-platform-windows-git-bash.sh` - Windows shell configuration tests
- `test-platform-macos.sh` - macOS platform detection tests using buildfab

## Building Test Binaries

Before running tests, ensure all platform binaries are built:

```bash
# Build all platform binaries
go build -o bin/buildfab-linux-amd64 ./cmd/buildfab
GOOS=windows GOARCH=amd64 go build -o bin/buildfab-windows-amd64.exe ./cmd/buildfab
GOOS=darwin GOARCH=amd64 go build -o bin/buildfab-darwin-amd64 ./cmd/buildfab
```

## Manual Testing

For manual testing on different platforms:

```bash
# Test platform detection with buildfab
./bin/buildfab -c tests/cross-platform/linux_configuration.yml check-detection
./bin/buildfab -c tests/cross-platform/windows_configuration.yml check-detection
./bin/buildfab -c tests/cross-platform/macos_configuration.yml check-detection

# Test comprehensive platform info
./bin/buildfab --help | grep -E "(platform|arch|os|cpu)"
```

## Troubleshooting

### Container Tests Fail
- **Podman Issues**: Ensure Podman is installed and working (no sudo required)
- **Docker Issues**: Ensure Docker is running and accessible (may require sudo)
- **Container Runtime**: Tests auto-detect Podman (preferred) or Docker
- **Resources**: Verify container runtime has sufficient resources
- **Binaries**: Check that all required binaries exist in `bin/` directory
- **Config Files**: Ensure platform configuration files are present

### Buildfab Tests Fail
- Ensure buildfab binary is built for the target platform
- Check that platform configuration files exist and are valid YAML
- Verify buildfab can access platform detection variables
- Test platform detection variables manually

### Platform Detection Incorrect
- Check `/etc/os-release` file on Linux systems
- Verify runtime.GOOS and runtime.GOARCH values
- Test on actual target platform if possible
- Check that platform variables are properly interpolated in actions

### Container Runtime Issues
- **Install Podman**: `sudo dnf install podman` (Fedora/RHEL) or `sudo apt install podman` (Ubuntu/Debian)
- **Install Docker**: `sudo dnf install docker` (Fedora/RHEL) or `sudo apt install docker.io` (Ubuntu/Debian)
- **Podman Advantages**: No sudo required, rootless, more secure
- **Docker Fallback**: Traditional option, may require sudo for some operations

## CI/CD Integration

The cross-platform tests are integrated into GitHub Actions:

```yaml
# Runs on every push/PR
name: Cross-Platform Buildfab Detection Tests
```

Tests run on:
- Ubuntu latest (with Docker containers for Ubuntu 24.04, Debian 12, Windows Wine)
- Windows latest
- macOS latest

## Contributing

When adding new platform detection features:

1. Add tests to appropriate platform configuration files
2. Update Dockerfiles if new dependencies needed
3. Update expected results in this README
4. Ensure GitHub Actions tests pass
5. Test platform variable interpolation in buildfab actions

## Limitations

- **macOS Testing**: Cannot be tested via containers - requires actual macOS host or VM
- **Windows Testing**: Uses Wine simulation, may not catch all Windows-specific issues
- **ARM Testing**: Limited ARM platform availability in CI/CD
- **Container Dependencies**: Requires Podman or Docker for Linux/Windows tests
- **Root Requirements**: Docker may require sudo, Podman runs rootless

For production deployment, test on actual target platforms when possible.
