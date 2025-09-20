# System Patterns: buildfab

## System Architecture
buildfab follows a layered architecture with clear separation of concerns:

```
┌─────────────────┐
│   CLI Layer     │  ← cmd/buildfab/main.go
├─────────────────┤
│   Library API   │  ← pkg/buildfab/ (public interface)
├─────────────────┤
│  Core Engine    │  ← internal/executor/ (DAG execution)
├─────────────────┤
│  Configuration  │  ← internal/config/ (YAML parsing)
├─────────────────┤
│  Action System  │  ← internal/actions/ (built-in actions)
├─────────────────┤
│  Variable System│  ← internal/variables/ (interpolation)
└─────────────────┘
```

## Key Technical Decisions
- **Library-first design**: Core functionality in pkg/buildfab for embedding
- **DAG execution engine**: Parallel processing with dependency resolution
- **YAML configuration**: Human-readable format with validation
- **Built-in action registry**: Extensible system for common operations
- **Variable interpolation**: GitHub-style `${{ }}` syntax for dynamic values
- **Error policy system**: Configurable stop/warn behavior per step

## Design Patterns in Use
- **Command Pattern**: Actions implement common interface for execution
- **Strategy Pattern**: Different execution strategies for built-in vs custom actions
- **Observer Pattern**: Progress reporting and status updates
- **Builder Pattern**: Configuration and options construction
- **Factory Pattern**: Action instantiation and registration
- **Dependency Injection**: Context and options passed to components

## Component Relationships
- **CLI → Library API**: CLI is thin wrapper around library functions
- **Library API → Executor**: Main API delegates to DAG execution engine
- **Executor → Actions**: Executor manages action execution and dependencies
- **Actions → Variables**: Actions use variable system for interpolation
- **Config → All**: Configuration drives all component behavior
- **UI → All**: UI components provide user feedback and error reporting

## Critical Implementation Paths
1. **YAML Parsing**: project.yml → internal model → validation
2. **DAG Construction**: Actions → dependencies → cycle detection → execution plan
3. **Variable Resolution**: `${{ }}` → context → interpolation → action execution
4. **Parallel Execution**: DAG → wave scheduling → concurrent execution → result aggregation
5. **Error Handling**: Action failure → policy check → continue/stop → reporting
6. **Library Integration**: pre-push → buildfab API → stage execution → result handling

## Data Flow
```
project.yml → Config Parser → DAG Builder → Executor → Actions → Results → UI
     ↓              ↓            ↓           ↓         ↓        ↓
  Validation → Dependency → Wave → Action → Variable → Status
              Resolution  Planning Execution Interpolation Reporting
```