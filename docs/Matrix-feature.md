# Matrix Feature Documentation

## Overview

The Matrix feature in buildfab enables parallel execution across multiple configurations, allowing you to run the same action with different parameter combinations. This is particularly useful for testing across different platforms, versions, or configurations.

## Key Features

- **Single-dimension matrix support**: Define matrix values for one dimension at a time
- **Parse-time expansion**: Matrix values are expanded when the configuration is parsed
- **Configurable parallelism**: Control how many matrix jobs run concurrently
- **Fail-fast and continue-on-error policies**: Flexible error handling strategies
- **Matrix variable interpolation**: Use `${{ matrix.* }}` variables in action commands
- **Job ordering**: FIFO or random job scheduling
- **Real-time status reporting**: See job progress and results as they execute

## Configuration Syntax

### Basic Matrix Configuration

```yaml
stages:
  test-matrix:
    - action: test-action
      matrix:
        values:
          os: ["linux", "windows", "macos"]
        strategy:
          max_parallel: 2
          fail_fast: true
          continue_on_error: false
          order: "fifo"
```

### Matrix Values

Matrix values are defined in the `values` section and can contain any YAML-compatible data types:

```yaml
matrix:
  values:
    os: ["linux", "windows", "macos"]
    version: ["1.0", "2.0", "3.0"]
    arch: ["amd64", "arm64"]
```

**Note**: Currently, only single-dimension matrices are supported. Multi-dimensional matrices will be supported in future versions.

### Matrix Strategy

The `strategy` section controls how matrix jobs are executed:

```yaml
strategy:
  max_parallel: 2        # Maximum concurrent jobs (default: all)
  fail_fast: true        # Stop all jobs on first failure (default: false)
  continue_on_error: false # Stage succeeds even if some jobs fail (default: false)
  order: "fifo"          # Job scheduling order: "fifo" or "random" (default: "fifo")
```

#### Strategy Options

- **`max_parallel`**: Maximum number of jobs to run concurrently. Defaults to all available jobs if not specified.
- **`fail_fast`**: When `true`, stops all jobs (including running ones) when any job fails. When `false`, allows all jobs to complete.
- **`continue_on_error`**: When `true`, the stage succeeds even if some matrix jobs fail. When `false`, the stage fails if any job fails.
- **`order`**: Job scheduling order:
  - `"fifo"`: First In, First Out (default)
  - `"random"`: Random order

## Matrix Variable Interpolation

Matrix values are available as variables in action commands using the `${{ matrix.* }}` syntax:

```yaml
actions:
  - name: test-action
    run: echo "Testing on ${{ matrix.os }} version ${{ matrix.version }}"
```

### Available Matrix Variables

- `${{ matrix.os }}` - Operating system value
- `${{ matrix.version }}` - Version value
- `${{ matrix.arch }}` - Architecture value
- `${{ matrix.* }}` - Any matrix dimension value

## Examples

### Cross-Platform Testing

```yaml
stages:
  cross-platform-test:
    - action: run-tests
      matrix:
        values:
          os: ["linux", "windows", "macos"]
        strategy:
          max_parallel: 3
          fail_fast: false
          continue_on_error: false
          order: "fifo"

actions:
  - name: run-tests
    run: |
      echo "Running tests on ${{ matrix.os }}"
      # Your test commands here
```

### Version Matrix Testing

```yaml
stages:
  version-test:
    - action: test-version
      matrix:
        values:
          version: ["1.0", "1.1", "2.0", "2.1"]
        strategy:
          max_parallel: 2
          fail_fast: true
          continue_on_error: false
          order: "fifo"

actions:
  - name: test-version
    run: |
      echo "Testing version ${{ matrix.version }}"
      # Install and test specific version
```

### Single Test Matrix

```yaml
stages:
  test-suite:
    - action: run-test
      matrix:
        values:
          test: ["unit", "integration", "e2e", "performance"]
        strategy:
          max_parallel: 4
          fail_fast: false
          continue_on_error: true
          order: "random"

actions:
  - name: run-test
    run: |
      echo "Running ${{ matrix.test }} tests"
      # Run specific test type
```

## Error Handling

### Fail Fast Strategy

When `fail_fast: true`, the matrix execution stops immediately when any job fails:

```yaml
strategy:
  fail_fast: true
```

**Behavior**:
- Stops scheduling new jobs when a job fails
- Cancels all running jobs
- Stage fails immediately

### Continue on Error Strategy

When `continue_on_error: true`, the stage succeeds even if some jobs fail:

```yaml
strategy:
  continue_on_error: true
```

**Behavior**:
- All jobs continue to completion
- Stage succeeds even with failed jobs
- Useful for exploratory testing

### Combined Strategies

You can combine both strategies:

```yaml
strategy:
  fail_fast: true
  continue_on_error: true
```

**Behavior**:
- `fail_fast` stops scheduling new jobs on first failure
- `continue_on_error` determines final stage status
- Running jobs are cancelled when a job fails

## Job Status and Monitoring

### Real-time Status

Matrix jobs show real-time status during execution:

```
[matrix] ✓ (1/3): test-action-1 - os=linux, version=1.0
[matrix] ✓ (2/3): test-action-2 - os=windows, version=1.0
[matrix] ✗ (3/3): test-action-3 - os=macos, version=1.0
```

### Status Icons

- `✓` - Job completed successfully
- `✗` - Job failed
- `○` - Job running (if verbose mode enabled)

### Final Summary

After all jobs complete, a summary is shown:

```
Matrix execution completed: 3 jobs
✓ 2 successful, ✗ 1 failed
```

## Integration with Other Features

### Matrix with Action Variants

Matrix jobs can use action variants for conditional execution:

```yaml
actions:
  - name: platform-specific-test
    variants:
      - when: "${{ matrix.os == 'linux' }}"
        run: "echo 'Linux-specific test'"
      - when: "${{ matrix.os == 'windows' }}"
        run: "echo 'Windows-specific test'"
      - when: "${{ matrix.os == 'macos' }}"
        run: "echo 'macOS-specific test'"
```

### Matrix with Step Conditions

Matrix steps can use step conditions:

```yaml
stages:
  conditional-matrix:
    - action: test-action
      if: "${{ matrix.os != 'windows' }}"  # Skip Windows
      matrix:
        values:
          os: ["linux", "windows", "macos"]
```

## Best Practices

### 1. Use Appropriate Parallelism

Set `max_parallel` based on your system resources:

```yaml
strategy:
  max_parallel: 4  # Adjust based on CPU cores and memory
```

### 2. Choose the Right Error Strategy

- Use `fail_fast: true` for critical tests that must all pass
- Use `continue_on_error: true` for exploratory testing
- Use `fail_fast: false` with `continue_on_error: false` for comprehensive testing

### 3. Organize Matrix Values Logically

Group related values together:

```yaml
# Good: Logical grouping
matrix:
  values:
    os: ["linux", "windows", "macos"]
    version: ["1.0", "2.0"]

# Avoid: Too many combinations
matrix:
  values:
    test: ["unit", "integration", "e2e", "performance", "load", "stress"]
```

### 4. Use Descriptive Matrix Values

Use meaningful values that are easy to understand:

```yaml
# Good: Clear and descriptive
matrix:
  values:
    environment: ["dev", "staging", "prod"]
    database: ["postgresql", "mysql", "sqlite"]

# Avoid: Cryptic values
matrix:
  values:
    env: ["d", "s", "p"]
    db: ["pg", "my", "sq"]
```

## Limitations

### Current Limitations

1. **Single-dimension only**: Multi-dimensional matrices are not yet supported
2. **Stage-level only**: Matrix configuration is only supported at the stage level
3. **No job dependencies**: Matrix jobs cannot depend on other matrix jobs
4. **No job filtering**: Cannot filter matrix jobs at runtime

### Future Enhancements

1. **Multi-dimensional matrices**: Support for multiple matrix dimensions
2. **Job dependencies**: Allow matrix jobs to depend on other jobs
3. **Runtime filtering**: Filter matrix jobs based on conditions
4. **Matrix includes/excludes**: Include or exclude specific combinations
5. **Dynamic matrix values**: Generate matrix values at runtime

## Troubleshooting

### Common Issues

#### 1. Matrix Jobs Not Running

**Problem**: Matrix jobs are not executing.

**Solution**: Check that:
- Matrix values are not empty
- Action exists and is properly configured
- No syntax errors in matrix configuration

#### 2. Variable Interpolation Not Working

**Problem**: `${{ matrix.* }}` variables are not being replaced.

**Solution**: Ensure:
- Variables are properly formatted with `${{ }}` syntax
- Matrix values are defined in the `values` section
- Action commands use the correct variable names

#### 3. Jobs Running Sequentially Instead of Parallel

**Problem**: Matrix jobs are running one at a time.

**Solution**: Check:
- `max_parallel` is set to a value greater than 1
- System has sufficient resources
- No resource constraints in the action configuration

#### 4. Unexpected Job Failures

**Problem**: Matrix jobs are failing unexpectedly.

**Solution**: Verify:
- Action commands are correct for each matrix value
- Dependencies are available for all matrix combinations
- Error handling is appropriate for the use case

### Debug Mode

Enable debug mode to see detailed matrix execution information:

```bash
buildfab run test-matrix --debug
```

This will show:
- Matrix expansion details
- Job scheduling information
- Variable interpolation results
- Execution timing

## Migration Guide

### From Sequential Testing

If you're migrating from sequential testing to matrix testing:

**Before**:
```yaml
stages:
  test-all:
    - action: test-linux
    - action: test-windows
    - action: test-macos
```

**After**:
```yaml
stages:
  test-all:
    - action: test-platform
      matrix:
        values:
          os: ["linux", "windows", "macos"]
```

### From External Matrix Tools

If you're migrating from external matrix tools (like GitHub Actions matrix):

**GitHub Actions**:
```yaml
strategy:
  matrix:
    os: [linux, windows, macos]
    version: [1.0, 2.0]
```

**buildfab**:
```yaml
matrix:
  values:
    os: ["linux", "windows", "macos"]
    version: ["1.0", "2.0"]
```

## Conclusion

The Matrix feature provides powerful parallel execution capabilities for buildfab, enabling efficient testing and validation across multiple configurations. By following the best practices and understanding the configuration options, you can create robust, scalable automation workflows that take full advantage of parallel execution.

For more information about buildfab features, see the [Features and Examples](Features-and-examples.md) documentation.
