# Deployment and CI/CD Documentation

This document describes the deployment process, CI/CD pipeline, and release automation for buildfab.

## CI/CD Pipeline Overview

buildfab uses GitHub Actions for continuous integration and deployment:

1. **CI Pipeline**: Runs on every push and pull request
2. **Release Pipeline**: Runs on version tags
3. **Package Updates**: Automatic package manager updates

## GitHub Actions Workflows

### CI Workflow (.github/workflows/ci.yml)

**Triggers:**
- Push to any branch
- Pull request events

**Jobs:**
1. **Test**: Run tests with race detection
2. **Lint**: Code quality checks
3. **Build**: Cross-platform compilation
4. **Security**: Vulnerability scanning

```yaml
name: CI

on:
  push:
    branches: [ master, develop ]
  pull_request:
    branches: [ master ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: go test ./... -race
      - run: golangci-lint run
```

### Release Workflow (.github/workflows/release.yml)

**Triggers:**
- Push of version tags (v*)

**Jobs:**
1. **Build**: Multi-platform compilation
2. **Package**: Create release archives
3. **Deploy**: Publish to GitHub Releases
4. **Update Packages**: Update package managers

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: ./buildtools/build-goreleaser.sh release
```

## Release Process

### 1. Automated Release

When a version tag is pushed:

1. **GitHub Actions triggers** release workflow
2. **GoReleaser builds** for all platforms
3. **Archives created** for each platform
4. **GitHub Release** published automatically
5. **Package managers** updated

### 2. Manual Release

```bash
# 1. Check version status
./scripts/check-version-status

# 2. Increment version
./scripts/version-bump minor

# 3. Update changelog
# (Edit CHANGELOG.md)

# 4. Commit and tag
git add .
git commit -m "chore: release v$(cat VERSION)"
git tag $(cat VERSION)
git push origin master
git push origin $(cat VERSION)

# 5. GitHub Actions handles the rest
```

## Package Manager Integration

### Scoop (Windows)

#### Automatic Updates
- GitHub Action updates `packaging/windows/scoop-bucket/version.json`
- Scoop users get updates via `scoop update buildfab`

#### Manual Update Process
```bash
# Update version in scoop manifest
cd packaging/windows/scoop-bucket
# Edit version.json with new version and hash
git add version.json
git commit -m "chore: update scoop manifest to v1.2.3"
git push origin master
```

### Homebrew (macOS)

#### Future Support
- Homebrew formula planned for future releases
- Automatic formula updates via GitHub Action
- Easy installation via `brew install buildfab`

### Linux Package Managers

#### tar.gz Archives
- Available on GitHub Releases
- Include install.sh script for easy installation
- Support for multiple architectures

## Deployment Environments

### GitHub Releases

**Primary distribution method:**
- Multi-platform binaries
- Source code archives
- Release notes and changelog
- Checksums for verification

**Release artifacts:**
- `buildfab-linux-amd64.tar.gz`
- `buildfab-linux-arm64.tar.gz`
- `buildfab-windows-amd64.zip`
- `buildfab-windows-arm64.zip`
- `buildfab-darwin-amd64.tar.gz`
- `buildfab-darwin-arm64.tar.gz`

### Package Managers

#### Scoop (Windows)
- **Repository**: `burnes/buildfab-scoop-bucket`
- **Manifest**: `packaging/windows/scoop-bucket/version.json`
- **Installation**: `scoop install buildfab`

#### Homebrew (macOS) - Future
- **Formula**: `buildfab.rb`
- **Installation**: `brew install buildfab`

## Security and Verification

### Release Signing

**Git tag signing:**
```bash
# Sign tags with GPG
git tag -s v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

**Binary verification:**
- SHA256 checksums provided
- GPG signatures for releases
- Reproducible builds

### Security Scanning

**Automated security checks:**
- CodeQL analysis
- Dependency vulnerability scanning
- License compliance checking
- Secret detection

## Monitoring and Observability

### Release Monitoring

**GitHub Actions status:**
- Build status badges
- Release status tracking
- Failure notifications

**Package manager status:**
- Scoop manifest updates
- Homebrew formula updates
- Download statistics

### Error Handling

**Build failures:**
- Automatic rollback on failure
- Notification to maintainers
- Retry mechanisms

**Deployment failures:**
- Manual intervention required
- Rollback procedures
- Incident response

## Rollback Procedures

### Release Rollback

1. **Delete release tag:**
   ```bash
   git tag -d v1.2.3
   git push origin :refs/tags/v1.2.3
   ```

2. **Delete GitHub release:**
   - Go to GitHub Releases
   - Delete the release
   - Remove associated artifacts

3. **Revert package manager updates:**
   - Update Scoop manifest to previous version
   - Revert Homebrew formula (if applicable)

### Emergency Procedures

**Critical issues:**
1. **Immediate response**: Delete release and tag
2. **Communication**: Notify users via GitHub issues
3. **Fix**: Address the issue
4. **Re-release**: Create new version with fix

## Performance Optimization

### Build Performance

**Parallel builds:**
- Multi-platform builds in parallel
- Go module caching
- Conan package caching

**Release performance:**
- Incremental builds where possible
- Artifact caching
- Parallel uploads

### Deployment Performance

**CDN integration:**
- GitHub Releases as CDN
- Package manager mirrors
- Geographic distribution

## Troubleshooting

### Common Issues

1. **Build failures:**
   - Check Go version compatibility
   - Verify Conan configuration
   - Review GitHub Actions logs

2. **Release failures:**
   - Check GoReleaser configuration
   - Verify GitHub token permissions
   - Review release workflow logs

3. **Package manager issues:**
   - Verify manifest format
   - Check hash calculations
   - Review update automation

### Debug Commands

```bash
# Test GoReleaser configuration
./buildtools/build-goreleaser.sh dry-run

# Check package manager manifests
cat packaging/windows/scoop-bucket/version.json

# Verify release artifacts
sha256sum buildfab-*
```

## Future Improvements

### Planned Enhancements

1. **Multi-architecture support**: ARM64 for all platforms
2. **Container images**: Docker images for CI/CD
3. **Package manager expansion**: APT, YUM, Chocolatey
4. **Automated testing**: Integration tests in CI
5. **Performance monitoring**: Build and deployment metrics

### Automation Improvements

1. **Dependency updates**: Automated dependency updates
2. **Security scanning**: Enhanced security checks
3. **Release notes**: Automated changelog generation
4. **Rollback automation**: Automated rollback procedures