# Docker-based Cross-Platform Testing

This document explains how to run comprehensive cross-platform tests using Docker containers.

## Prerequisites

1. **Docker installed and running**
   ```bash
   # Check Docker installation
   docker --version
   docker run hello-world
   ```

2. **Built platform binaries**
   ```bash
   cd /path/to/version/project
   ./buildtools/build-and-package.sh
   
   # Or build manually:
   go build -o bin/version-linux-amd64 cmd/version/*.go
   GOOS=windows GOARCH=amd64 go build -o bin/version-windows-amd64.exe cmd/version/*.go
   GOOS=darwin GOARCH=amd64 go build -o bin/version-darwin-amd64 cmd/version/*.go
   ```

## Running Docker Tests

### Option 1: Run All Tests
```bash
cd test/cross-platform
buildfab test
```

### Option 2: Run Individual Platform Tests
```bash
cd test/cross-platform

# Test Ubuntu 24.04
buildfab test-linux-ubuntu

# Test Debian 12  
buildfab test-linux-debian

# Test Windows (using Wine)
buildfab test-windows
```

### Option 3: Manual Docker Commands
```bash
cd test/cross-platform

# Build and test Ubuntu
docker build -f Dockerfile.linux-ubuntu -t version-test-ubuntu .
docker run --rm version-test-ubuntu

# Build and test Debian
docker build -f Dockerfile.linux-debian -t version-test-debian .
docker run --rm version-test-debian

# Build and test Windows (Wine)
docker build -f Dockerfile.windows -t version-test-windows .
docker run --rm version-test-windows
```

## Expected Test Results

### Ubuntu 24.04
```
=== Cross-Platform Platform Detection Test ===
Testing on: Ubuntu 24.04 LTS

Testing platform detection commands...
1. Testing 'platform' command:
   Result: linux
   ✅ PASS

2. Testing 'arch' command:
   Result: amd64
   ✅ PASS

3. Testing 'os' command:
   Result: ubuntu
   ✅ PASS

4. Testing 'os_version' command:
   Result: 24.04
   ✅ PASS

5. Testing 'cpu' command:
   Result: 8
   ✅ PASS

=== All Platform Detection Tests Passed! ===
Platform: linux
Architecture: amd64
OS: ubuntu
OS Version: 24.04
CPU Cores: 8
```

### Debian 12
```
=== Cross-Platform Platform Detection Test ===
Testing on: Debian GNU/Linux 12 (bookworm)

Testing platform detection commands...
1. Testing 'platform' command:
   Result: linux
   ✅ PASS

2. Testing 'arch' command:
   Result: amd64
   ✅ PASS

3. Testing 'os' command:
   Result: debian
   ✅ PASS

4. Testing 'os_version' command:
   Result: 12
   ✅ PASS

5. Testing 'cpu' command:
   Result: 8
   ✅ PASS

=== All Platform Detection Tests Passed! ===
Platform: linux
Architecture: amd64
OS: debian
OS Version: 12
CPU Cores: 8
```

### Windows (Wine)
```
=== Cross-Platform Platform Detection Test (Windows) ===
Testing Windows platform detection using Wine

Testing platform detection commands...
1. Testing 'platform' command:
   Result: windows
   ✅ PASS

2. Testing 'arch' command:
   Result: amd64
   ✅ PASS

3. Testing 'os' command:
   Result: windows
   ✅ PASS

4. Testing 'os_version' command:
   Result: windows
   ✅ PASS

5. Testing 'cpu' command:
   Result: 8
   ✅ PASS

=== All Windows Platform Detection Tests Passed! ===
Platform: windows
Architecture: amd64
OS: windows
OS Version: windows
CPU Cores: 8
```

## Troubleshooting

### Docker Permission Denied
```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Log out and back in, or run:
newgrp docker
```

### Missing Binaries Error
```bash
❌ Missing binaries:
   - version-linux-amd64
   - version-windows-amd64.exe
   - version-darwin-amd64

Please build all platform binaries first:
   cd /path/to/version/project
   ./buildtools/build-and-package.sh
```

### Wine Issues on Windows Testing
```bash
# Wine may need additional setup
sudo apt-get update
sudo apt-get install wine-stable

# Or use specific Wine version
sudo apt-get install winehq-stable
```

### Docker Build Fails
```bash
# Clean up Docker
docker system prune -a

# Rebuild from scratch
docker build --no-cache -f Dockerfile.linux-ubuntu -t version-test-ubuntu .
```

## Test Customization

### Adding New Linux Distributions
1. Create new Dockerfile (e.g., `Dockerfile.linux-centos`)
2. Add test case to `run-cross-platform-tests.sh`
3. Update expected results in test scripts

### Modifying Test Scripts
Edit the test scripts in this directory:
- `test-platform.sh` - Linux test logic
- `test-platform-windows.sh` - Windows test logic
- `test-platform-macos.sh` - macOS test logic

### Adding New Platform Tests
1. Add new command to test scripts
2. Update expected results
3. Add validation logic

## Performance Notes

- **Build Time**: ~2-3 minutes per platform
- **Test Time**: ~30 seconds per platform
- **Total Time**: ~10-15 minutes for all platforms
- **Resource Usage**: ~1GB RAM, ~2GB disk per container

## Integration with CI/CD

The Docker tests are integrated into GitHub Actions:

```yaml
# .github/workflows/cross-platform-test.yml
name: Cross-Platform Platform Detection Tests
```

Runs automatically on:
- Push to master
- Pull requests
- Manual trigger (workflow_dispatch)

## Limitations

1. **macOS Testing**: Cannot run macOS Docker containers on Linux
2. **ARM Architecture**: Limited ARM testing (requires ARM hosts)
3. **Wine Accuracy**: Windows testing via Wine may not catch all edge cases
4. **Network Access**: Some tests may require internet access for package downloads

For production deployments, test on actual target platforms when possible.
