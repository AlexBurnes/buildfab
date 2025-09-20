# Developer Workflow

This document describes the development setup, workflow, and contribution guidelines for the buildfab project.

## Development Setup

### Prerequisites

- **Go 1.22+**: Primary development language
- **CMake**: Cross-platform build system
- **Conan**: Dependency management
- **Git**: Version control
- **golangci-lint**: Code linting (optional but recommended)

### Environment Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/burnes/buildfab.git
   cd buildfab
   ```

2. **Install Go toolchain via Conan**:
   ```bash
   mkdir build && cd build
   cmake ..
   cmake --build .
   ```

3. **Or install Go directly**:
   ```bash
   # Install Go 1.22+ from https://golang.org/dl/
   go version
   ```

4. **Install development dependencies**:
   ```bash
   go mod tidy
   ```

### IDE Setup

#### VS Code
- Install Go extension
- Install Go tools: `Ctrl+Shift+P` → "Go: Install/Update Tools"
- Configure formatting: `gofmt` and `goimports`

#### GoLand/IntelliJ
- Enable Go plugin
- Configure Go SDK to use project's Go version
- Enable format on save with `gofmt`

## Development Workflow

### 1. Version Management

**Before making ANY changes**, check version status:
```bash
./scripts/check-version-status
```

If version matches current tag, increment version:
```bash
./scripts/version-bump patch    # for bug fixes
./scripts/version-bump minor    # for new features
./scripts/version-bump major    # for breaking changes
```

### 2. Making Changes

1. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes** following Go coding standards:
   - Use 4 spaces for indentation
   - Maximum line length of 120 characters
   - Follow golang naming conventions
   - Add docstrings for all exported functions
   - Use descriptive variable and function names

3. **Test your changes**:
   ```bash
   go test ./... -race
   go test ./... -v
   ```

4. **Format and lint**:
   ```bash
   gofmt -s -w .
   goimports -w .
   golangci-lint run
   ```

5. **Update documentation**:
   - Update relevant documentation files
   - Add entries to CHANGELOG.md
   - Update memory bank files if needed

### 3. Build and Test

#### Local Build
```bash
# Build with Go directly
go build -o buildfab cmd/buildfab/main.go

# Or build with CMake/Conan
mkdir build && cd build
cmake ..
cmake --build .
```

#### Testing
```bash
# Run all tests
go test ./... -race

# Run tests with coverage
go test ./... -cover

# Run specific package tests
go test ./pkg/buildfab -v

# Run integration tests
go test ./internal/... -v
```

#### Dry Run Release
```bash
./buildtools/build-goreleaser.sh dry-run
```

### 4. Commit and Push

1. **Stage changes**:
   ```bash
   git add .
   ```

2. **Commit with conventional format**:
   ```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: resolve bug in execution engine"
   git commit -m "docs: update API documentation"
   ```

3. **Push changes**:
   ```bash
   git push origin feature/your-feature-name
   ```

### 5. Create Pull Request

1. **Create PR** with:
   - Clear description of changes
   - Reference to related issues
   - Links to memory bank entries if applicable
   - Updated documentation

2. **Wait for review** and address feedback

3. **Merge** after approval

### 6. Release Process

1. **Increment version** (if not done already):
   ```bash
   ./scripts/version-bump minor
   ```

2. **Update CHANGELOG.md** with new version

3. **Create and push tag**:
   ```bash
   git tag $(cat VERSION)
   git push origin master
   git push origin $(cat VERSION)
   ```

4. **Trigger release**:
   ```bash
   ./buildtools/build-goreleaser.sh release
   ```

## Code Standards

### Go Coding Standards

- **Indentation**: 4 spaces (not tabs)
- **Line length**: Maximum 120 characters
- **Naming**: Follow Go conventions (camelCase for private, PascalCase for public)
- **Documentation**: All exported symbols must have doc comments
- **Error handling**: Explicit error handling, no panics in normal flow
- **Context**: Pass context.Context as first parameter for long-running operations

### File Organization

```
buildfab/
├── cmd/buildfab/           # CLI application
├── pkg/buildfab/           # Public library API
├── internal/               # Private implementation
│   ├── config/             # YAML parsing and validation
│   ├── executor/           # DAG execution engine
│   ├── actions/            # Built-in action implementations
│   ├── variables/          # Variable interpolation
│   └── ui/                 # Output formatting
├── docs/                   # Documentation
├── examples/               # Usage examples
└── testdata/               # Test fixtures
```

### Testing Standards

- **Unit tests**: Table-driven tests for parsing and logic
- **Integration tests**: End-to-end stage execution
- **Race detection**: Use `-race` flag in CI
- **Coverage**: Aim for ≥70% coverage in core packages
- **Golden tests**: For stable textual outputs

## Memory Bank Integration

### Updating Memory Bank

After significant changes, update memory bank files:
- `projectbrief.md`: Core requirements and goals
- `productContext.md`: User experience and success metrics
- `activeContext.md`: Current work focus and recent changes
- `systemPatterns.md`: Architecture and technical decisions
- `techContext.md`: Technologies and development setup
- `progress.md`: What works and what's left to build

### Memory Bank Commands

```bash
# Update memory bank after changes
# (This will be handled by the AI assistant)
```

## Troubleshooting

### Common Issues

1. **Version check fails**:
   ```bash
   ./scripts/check-version-status
   # If needed: ./scripts/version-bump patch
   ```

2. **Build fails**:
   ```bash
   go mod tidy
   go clean -cache
   ```

3. **Tests fail**:
   ```bash
   go test ./... -v -race
   ```

4. **Linting errors**:
   ```bash
   golangci-lint run --fix
   ```

### Getting Help

- Check existing issues on GitHub
- Review documentation in `docs/` directory
- Check memory bank files for project context
- Ask questions in project discussions

## Contributing Guidelines

### Before Contributing

1. **Read the documentation**: Understand the project goals and architecture
2. **Check existing issues**: Look for related work or discussions
3. **Follow coding standards**: Ensure your code matches project style
4. **Write tests**: Include tests for new functionality
5. **Update documentation**: Keep docs current with your changes

### Pull Request Guidelines

- **Small, focused changes**: One feature or fix per PR
- **Clear description**: Explain what and why
- **Tests included**: New code must have tests
- **Documentation updated**: Update relevant docs
- **CHANGELOG.md updated**: Document your changes
- **Memory bank updated**: If applicable

### Code Review Process

1. **Automated checks**: CI runs tests and linting
2. **Manual review**: Maintainers review code quality and design
3. **Feedback**: Address review comments
4. **Approval**: Merge after approval

## Release Management

### Version Numbering

- **Semantic versioning**: `vX.Y.Z` format
- **Patch (Z)**: Bug fixes and minor improvements
- **Minor (Y)**: New features, backward compatible
- **Major (X)**: Breaking changes

### Release Checklist

- [ ] Version incremented in VERSION file
- [ ] CHANGELOG.md updated
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Memory bank updated
- [ ] GoReleaser dry-run successful
- [ ] Tag created and pushed
- [ ] Release published