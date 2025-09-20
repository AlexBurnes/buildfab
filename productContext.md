# Product Context: buildfab

## Problem Statement
Current project automation relies on complex bash scripts that are difficult to maintain, debug, and extend. The pre-push utility and similar tools need a robust, dependency-aware execution engine that can handle complex workflows with parallel execution, error policies, and clear reporting.

## User Experience Goals
- **Simple configuration**: Define automation workflows in clear, readable YAML files
- **Fast execution**: Parallel processing where possible, with clear progress indication
- **Clear feedback**: Colorized output with status indicators (✔, ⚠, ✖, ○)
- **Easy debugging**: Detailed error messages with reproduction hints
- **Flexible execution**: Run entire stages, individual steps, or standalone actions
- **Consistent behavior**: Predictable execution across different platforms and environments

## Success Metrics
- **Adoption**: Successful integration with pre-push utility and other automation tools
- **Performance**: Faster execution compared to sequential bash scripts
- **Reliability**: Consistent execution results across different environments
- **Maintainability**: Reduced complexity in automation configuration and debugging
- **Developer satisfaction**: Positive feedback on error messages and debugging experience

## Target Users
- **Primary**: Developers using pre-push and similar automation tools
- **Secondary**: DevOps engineers creating project automation workflows
- **Tertiary**: Open source maintainers needing flexible build/validation systems

## Value Proposition
- **Replaces complex bash**: Eliminates error-prone shell scripting for automation
- **Enables complex workflows**: Supports dependency graphs and parallel execution
- **Improves reliability**: Better error handling and cross-platform compatibility
- **Reduces maintenance**: Clear configuration format and built-in actions
- **Enhances debugging**: Detailed error reporting and reproduction hints