# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is buildfab

buildfab is a **build orchestration tool and Go library** that replaces scattered build scripts with a single declarative `.project.yml` configuration. It executes build stages and actions through a **dependency graph (DAG)**, supports **matrix builds**, runs jobs in **Docker/Podman containers**, and works identically locally, in CI, and in Git hooks. The project is self-hosting — it builds itself using its own `.project.yml`.

## Commands

### Build
```bash
# Quick local build (recommended for development)
go build -o bin/buildfab ./cmd/buildfab

# Static binary (required for container compatibility)
CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -o bin/buildfab cmd/buildfab/main.go

# Build with embedded version
VERSION=$(cat VERSION) && CGO_ENABLED=0 go build -ldflags="-X main.appVersion=${VERSION} -s -w -extldflags '-static'" -o bin/buildfab cmd/buildfab/main.go

# Self-build using buildfab (requires buildfab and CMake/Conan)
buildfab run build
```

### Test
```bash
# Run all tests with race detector (standard)
go test ./... -race

# Run tests with verbose output
go test ./... -v

# Run a specific package
go test ./pkg/buildfab -v

# Run a single test
go test ./pkg/buildfab -run TestName -v

# Run with coverage
go test ./... -cover
```

### Lint / Format
```bash
gofmt -s -w .
goimports -w .
golangci-lint run
```

### Version Management
```bash
# Check version status before making changes
./scripts/check-version-status

# Bump version
scripts/version bump patch   # bug fix
scripts/version bump minor   # new feature
scripts/version bump major   # breaking change
```

### Release
```bash
./buildtools/build-goreleaser.sh dry-run   # validate
./buildtools/build-goreleaser.sh release   # publish
```

## Code Architecture

### Directory Structure
```
cmd/buildfab/          # CLI entrypoint (thin cobra wrapper)
pkg/buildfab/          # Public library API (the importable package)
  buildfab.go          # Core execution engine: action running, shell commands
  simple.go            # SimpleRunner: high-level API with built-in output handling
  config.go            # LoadConfig / LoadConfigFromBytes (YAML -> Config struct)
  types.go             # StepCallback interface, StepResult, StepStatus types
  job_node.go          # JobNode / ExecutableStep: units of work in the DAG
  job_expander.go      # Expands stages into hierarchical JobNode trees
  hierarchical_executor.go  # Executes the hierarchical DAG with wave scheduling
  pool.go              # Pool-based concurrency control for matrix steps
  matrix.go            # Matrix expansion: Cartesian product of dimensions
  expression.go        # Condition evaluator for `if:` and `when:` fields
  variables.go         # ${{ }} interpolation engine
  ordered_output.go    # OrderedOutputManager: queue-based sequential display
  platform.go          # Platform detection (os, arch)
  actions.go           # ActionRegistry interface + action dispatch
  container/           # ContainerConfig types (pkg-level, used in YAML parsing)
internal/
  actions/registry.go  # Built-in action implementations (git@*, version@*)
  config/              # Include file processing (glob expansion, merge, cycle detection)
  container/           # Docker/Podman runner implementations
  ui/                  # Terminal output utilities
  version/             # Version string parsing
```

### Execution Flow

1. **Config loading**: `LoadConfig()` reads `.project.yml`, resolves `include:` globs, validates with strict YAML parsing (unknown fields are errors).

2. **Stage resolution**: A `Stage` is a list of `Step`s. Each step references either an `action:` name or a nested `stage:`. Steps declare dependencies via `require:` / `depends_on:`.

3. **DAG construction**: `job_expander.go` converts a stage's steps into a `HierarchicalDAG` of `JobNode`s. Matrix steps expand into multiple jobs (Cartesian product of matrix dimensions).

4. **Wave execution**: `HierarchicalExecutor` groups jobs into dependency waves. Each wave executes with semaphore-controlled parallelism (`max_parallel`). Jobs within a wave that have no dependencies run concurrently.

5. **Action execution**: Individual actions (`Action.Run` shell commands, `Action.Uses` built-in references, or `Action.Container` for container runs) execute within a `JobNode`'s sequential steps.

6. **Output management**: `OrderedOutputManager` buffers output from parallel steps and displays it in declaration order, preventing interleaved output.

### Key Concepts

- **`Config`** (in `buildfab.go`): Root struct — `Project`, `[]Action`, `map[string]Stage`, `[]Include`.
- **`Action`** can have: `run` (shell), `uses` (built-in like `git@untracked`), `container` (Docker/Podman config), or `variants` (conditional per-platform alternatives with `when:`).
- **`Step`** ties an action into a stage; adds `require`, `if`, `onerror`, `matrix`, `only`, and `variables` overrides.
- **`MatrixConfig`**: Declares dimension keys and their values. Expansion is a Cartesian product. Each combination becomes a `JobNode` with `MatrixVars` injected.
- **`SimpleRunner`**: The recommended library entrypoint — wraps `HierarchicalExecutor` with `OrderedOutputManager` and handles all display logic. Use `buildfab.RunStageSimple()` for one-liners.
- **`ActionRegistry`** interface (in `actions.go`): Implemented by `internal/actions.Registry`. Built-in actions (`git@untracked`, `git@uncommitted`, `git@modified`, `version@check`, `version@check-greatest`) are registered there.
- **`StepStatus`**: Two distinct "skipped" values — `StepStatusSkipped` (dependency failed, blocks dependents) vs `StepStatusSkippedCondition` (`if:` not met, does NOT block dependents). Both display as "skipped" to users.

### Variable Interpolation

`${{ expr }}` syntax throughout YAML. Available variables include `os`, `arch`, `platform`, `matrix.*` (from current matrix job), and any user-defined variables. Expressions support equality operators and boolean logic, evaluated by `expression.go`.

### Static Linking Requirement

All release binaries **must** be statically linked (`CGO_ENABLED=0 -extldflags '-static'`) to work inside Alpine and other musl-based container images. Never use dynamic linking for release artifacts.

### Coding Standards

- 4 spaces for indentation (not tabs — unusual for Go, but this project uses it)
- Max line length: 120 characters
- Conventional commits: `feat:`, `fix:`, `docs:`, `chore:`
- All exported symbols require doc comments
- Context as first parameter for long-running operations
- No panics in normal flow; explicit error returns

### Testing Patterns

- Table-driven tests throughout `pkg/buildfab/`
- Use `-race` flag always — concurrency is central to the codebase
- Integration tests in `*_integration_test.go` files exercise full stage execution
- Golden tests for stable textual outputs

## Workflow Rules

### Before ANY Code Changes

1. Check version status — if VERSION matches the latest git tag, bump it first:
   ```bash
   ./scripts/check-version-status
   scripts/version bump patch   # bug fix
   scripts/version bump minor   # new feature
   scripts/version bump major   # breaking change
   ```
   Prefer `scripts/version-bump-with-file <type>` — it also updates packaging files (Scoop, Homebrew).

2. After code changes, always rebuild the binary before testing:
   ```bash
   VERSION=$(cat VERSION) && CGO_ENABLED=0 go build -ldflags "-X main.appVersion=${VERSION} -s -w -extldflags '-static'" -o bin/buildfab ./cmd/buildfab
   # or via buildfab action:
   ./bin/buildfab action install-binary
   ```

### Commit Format

Conventional Commits with version in the subject:
```
type(scope): vX.Y.Z, brief description

- Bullet details
- More details
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `build`, `ci`.
Scopes: `ui`, `config`, `exec`, `tests`, `docs`, `build`, `ci`.

Examples:
- `fix(exec): v0.32.2, fix matrix step dependency resolution`
- `feat(matrix): v0.33.0, add nested matrix on stage support`

### Release Workflow ("complete changes")

When asked to "complete changes", "complete", "finish changes", or "finish him":

1. `./scripts/check-version-status` — check version
2. `scripts/version-bump-with-file patch|minor|major` — bump version
3. `./bin/buildfab action install-binary` — rebuild binary
4. `date +%Y-%m-%d` — get current date for changelog
5. Update `CHANGELOG.md` (mandatory for any code/doc change)
6. Update `activeContext.md` and `progress.md` memory bank files
7. `git add . && git commit -m "type(scope): vX.Y.Z, description"`
8. `git tag $(cat VERSION)`
9. `git push origin master --tags` — NEVER use `--no-verify`

**If push fails**: fix issues, commit fixes, `git tag -d vX.Y.Z`, retag, push again.
**Never bump version just because push failed** — retag the same version instead.

### Changelog Rules

- **CHANGELOG.md must be updated** for every code or documentation change.
- Section headers: `### Added`, `### Fixed`, `### Changed`, `### Documentation`.
- Historical version dates: `git log -1 --format="%ai" <tag>` — never invent dates.
- Current version date: `date +%Y-%m-%d`.
- Format: `## [X.Y.Z] - YYYY-MM-DD`, newest first.

### Git Tag Rules

- Always push with `git push origin master --tags`.
- If the tag already exists on remote, use the retag process — do NOT bump the version.
- Retag commit message: `chore(retag): retag vX.Y.Z after push failure`.

### Debug Output

For complex logic (parallel execution, queuing, state machines, callbacks), add structured debug output gated on the `--debug` / `-d` flag:

```go
if opts.Debug {
    fmt.Fprintf(opts.ErrorOutput, "[DEBUG] ComponentName: message, state=%v\n", state)
}
```

Format: `[DEBUG] Component: Message`. Always include step/job names in debug lines.

### Documentation File Naming

- Pattern: `First-word-second-word.md` (title-case first word, dash-separated).
- Exceptions: `README.md`, `CHANGELOG.md` stay as-is; abbreviations stay uppercase (`CI-cd.md`).
- All docs except README, CHANGELOG, and memory bank files go in `docs/`.
- Memory bank files (`projectbrief.md`, `productContext.md`, `activeContext.md`, `systemPatterns.md`, `techContext.md`, `progress.md`) must stay in the repo root.
