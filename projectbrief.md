# Project Brief: buildfab

## Overview
buildfab is a Go-based CLI utility and library for executing project automation stages and actions defined in YAML configuration files. It provides a flexible framework for running complex, dependency-aware automation workflows with parallel execution capabilities.

## Core Requirements
- **YAML-driven configuration**: Define stages and actions in `project.yml` files
- **DAG-based execution**: Parallel execution with explicit dependencies and cycle detection
- **Built-in action registry**: Extensible system for common automation tasks (git checks, version validation)
- **Custom action support**: Execute shell commands and external tools with variable interpolation
- **Library API**: Embeddable Go library for integration with other tools like pre-push
- **Cross-platform compatibility**: Linux, Windows, macOS (amd64/arm64)

## Goals
- **Replace bash scripts**: Provide a robust, maintainable alternative to complex bash automation scripts
- **Enable pre-push integration**: Serve as the execution engine for the pre-push utility
- **Improve developer experience**: Clear error messages, reproduction hints, and colorized output
- **Support complex workflows**: Handle dependency graphs, conditional execution, and error policies
- **Maintain compatibility**: Preserve existing YAML schema and semantics from pre-push utility

## Project Scope
**In Scope:**
- CLI interface for running stages and actions
- Go library API for embedding in other tools
- YAML configuration parsing and validation
- DAG execution engine with parallel processing
- Built-in actions for git and version operations
- Variable interpolation system
- Error handling and reporting
- Cross-platform build and packaging

**Out of Scope (v1):**
- Matrix builds or reusable workflows
- Remote executors or distributed execution
- Complex secrets management
- Webhook integrations
- Plugin system for external actions