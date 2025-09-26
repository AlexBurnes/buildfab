# Matrix Feature Test Examples

This directory contains comprehensive test examples for the buildfab Matrix feature. These tests demonstrate various matrix configurations, parallel execution patterns, and error handling scenarios.

## Test Files

### 1. `test_matrix.yml` - Comprehensive Matrix Tests
**Purpose**: Complete matrix feature testing with various configurations
**Usage**: `./bin/buildfab -c tests/test_matrix.yml <stage_name>`

**Available Stages**:
- `basic-matrix`: Simple 3x1 matrix (OS testing)
- `multi-dimension-matrix`: 2x2x2 matrix (OS, version, architecture)
- `long-running-tests`: Different duration tests with parallel execution
- `parallel-workload`: 5 parallel workloads with random ordering
- `fail-fast-test`: Demonstrates fail-fast behavior
- `continue-on-error-test`: Demonstrates continue-on-error behavior
- `platform-tests`: Platform-specific command execution
- `large-matrix`: 10-job matrix for stress testing
- `random-order-test`: Random job ordering demonstration

### 2. `test_matrix_simple.yml` - Quick Tests
**Purpose**: Simple, fast tests for quick validation
**Usage**: `./bin/buildfab -c tests/test_matrix_simple.yml <stage_name>`

**Available Stages**:
- `simple-matrix`: 2x2 matrix (test_name x platform)
- `single-dimension`: 3-job single dimension matrix
- `random-test`: 4-job random order test

### 3. `test_matrix_parallel.yml` - Parallel Execution Tests
**Purpose**: Test parallel execution with different workload types
**Usage**: `./bin/buildfab -c tests/test_matrix_parallel.yml <stage_name>`

**Available Stages**:
- `cpu-parallel`: CPU-intensive parallel tasks
- `io-parallel`: IO-intensive parallel tasks
- `network-parallel`: Network simulation parallel tasks
- `memory-parallel`: Memory-intensive parallel tasks
- `mixed-workload`: Mixed CPU and IO workloads
- `stress-test`: 8-job stress test with random ordering

### 4. `test_matrix_errors.yml` - Error Handling Tests
**Purpose**: Test error handling and edge cases
**Usage**: `./bin/buildfab -c tests/test_matrix_errors.yml <stage_name>`

**Available Stages**:
- `fail-fast-test`: Demonstrates fail_fast=true behavior
- `continue-on-error-test`: Demonstrates continue_on_error=true behavior
- `all-fail-test`: All jobs fail scenario
- `mixed-timing-failures`: Mixed timing with failures
- `fail-fast-slow`: Fail-fast with slow-failing jobs
- `random-failures`: Random order with failures
- `single-failure`: Single job failure test
- `all-pass-test`: All jobs pass scenario

## Quick Start Examples

### Basic Matrix Test
```bash
# Test basic matrix functionality
./bin/buildfab -c tests/test_matrix_simple.yml simple-matrix

# Test with verbose output
./bin/buildfab -c tests/test_matrix_simple.yml simple-matrix --verbose
```

### Parallel Execution Test
```bash
# Test parallel CPU workloads
./bin/buildfab -c tests/test_matrix_parallel.yml cpu-parallel

# Test with different parallelism levels
./bin/buildfab -c tests/test_matrix_parallel.yml stress-test
```

### Error Handling Test
```bash
# Test fail-fast behavior
./bin/buildfab -c tests/test_matrix_errors.yml fail-fast-test

# Test continue-on-error behavior
./bin/buildfab -c tests/test_matrix_errors.yml continue-on-error-test
```

### Long-Running Tests
```bash
# Test long-running matrix jobs
./bin/buildfab -c tests/test_matrix.yml long-running-tests

# Test with different parallel levels
./bin/buildfab -c tests/test_matrix.yml parallel-workload
```

## Matrix Configuration Examples

### Basic Matrix Configuration
```yaml
matrix:
  values:
    os: ["linux", "windows", "macos"]
  strategy:
    max_parallel: 2
    fail_fast: false
    continue_on_error: false
    order: "fifo"
```

### Multi-Dimensional Matrix
```yaml
matrix:
  values:
    os: ["linux", "windows"]
    version: ["1.0", "2.0"]
    arch: ["amd64", "arm64"]
  strategy:
    max_parallel: 4
    fail_fast: false
    continue_on_error: false
    order: "fifo"
```

### Parallel Execution with Random Ordering
```yaml
matrix:
  values:
    workload_id: ["A", "B", "C", "D", "E"]
  strategy:
    max_parallel: 3
    fail_fast: false
    continue_on_error: false
    order: "random"
```

### Fail-Fast Configuration
```yaml
matrix:
  values:
    test_case: ["pass", "fail", "pass", "pass"]
  strategy:
    max_parallel: 2
    fail_fast: true
    continue_on_error: false
    order: "fifo"
```

## Matrix Variables

Matrix jobs have access to the following variables:

- `${{ matrix.* }}`: Matrix values for the current job
- `${{ platform }}`: Current platform (linux, windows, darwin)
- `${{ arch }}`: Current architecture (amd64, arm64)
- `${{ matrix.job_id }}`: Unique job identifier

### Example Variable Usage
```bash
echo "Testing ${{ matrix.test_name }} on ${{ matrix.platform }}"
echo "Matrix values: ${{ matrix.test_name }}, ${{ matrix.platform }}"
echo "Job ID: ${{ matrix.job_id }}"
```

## Strategy Options

### max_parallel
- **Default**: All jobs run in parallel
- **Purpose**: Limit concurrent job execution
- **Example**: `max_parallel: 3` limits to 3 concurrent jobs

### fail_fast
- **Default**: `false`
- **Purpose**: Stop all jobs when first job fails
- **Example**: `fail_fast: true` stops all jobs on first failure

### continue_on_error
- **Default**: `false`
- **Purpose**: Continue execution even if some jobs fail
- **Example**: `continue_on_error: true` allows stage to succeed with some failures

### order
- **Default**: `"fifo"`
- **Options**: `"fifo"` (first-in-first-out) or `"random"`
- **Purpose**: Control job scheduling order
- **Example**: `order: "random"` randomizes job execution order

## Performance Testing

### CPU-Intensive Tests
```bash
# Test CPU-intensive parallel workloads
./bin/buildfab -c tests/test_matrix_parallel.yml cpu-parallel
```

### IO-Intensive Tests
```bash
# Test IO-intensive parallel workloads
./bin/buildfab -c tests/test_matrix_parallel.yml io-parallel
```

### Memory-Intensive Tests
```bash
# Test memory-intensive parallel workloads
./bin/buildfab -c tests/test_matrix_parallel.yml memory-parallel
```

### Stress Tests
```bash
# Test with many parallel jobs
./bin/buildfab -c tests/test_matrix_parallel.yml stress-test
```

## Troubleshooting

### Common Issues

1. **Jobs not running in parallel**: Check `max_parallel` setting
2. **Jobs stopping on first failure**: Check `fail_fast` setting
3. **Stage failing with job failures**: Check `continue_on_error` setting
4. **Jobs running in wrong order**: Check `order` setting

### Debug Mode
```bash
# Run with debug output
./bin/buildfab -c tests/test_matrix.yml basic-matrix --debug

# Run with verbose output
./bin/buildfab -c tests/test_matrix.yml basic-matrix --verbose
```

### Dry Run Mode
```bash
# Test configuration without execution
./bin/buildfab -c tests/test_matrix.yml basic-matrix --dry-run
```

## Expected Output

### Successful Matrix Execution
```
[matrix-tests] basic-matrix:echo-matrix:linux: Starting
[matrix-tests] basic-matrix:echo-matrix:windows: Starting
[matrix-tests] basic-matrix:echo-matrix:macos: Starting
[matrix-tests] basic-matrix:echo-matrix:linux: Matrix values: os=linux, version=1.0, arch=amd64
[matrix-tests] basic-matrix:echo-matrix:windows: Matrix values: os=windows, version=1.0, arch=amd64
[matrix-tests] basic-matrix:echo-matrix:macos: Matrix values: os=macos, version=1.0, arch=amd64
[matrix-tests] basic-matrix:echo-matrix:linux: Completed
[matrix-tests] basic-matrix:echo-matrix:windows: Completed
[matrix-tests] basic-matrix:echo-matrix:macos: Completed
[matrix-tests] basic-matrix: Completed (3 jobs)
```

### Failed Matrix Execution
```
[matrix-tests] fail-fast-test:sometimes-fail:pass: Starting
[matrix-tests] fail-fast-test:sometimes-fail:fail: Starting
[matrix-tests] fail-fast-test:sometimes-fail:pass: Completed
[matrix-tests] fail-fast-test:sometimes-fail:fail: Failed: exit status 1
[matrix-tests] fail-fast-test: Stopped due to fail_fast=true
```

## Contributing

When adding new test cases:

1. Follow the existing naming conventions
2. Include clear descriptions in comments
3. Test both success and failure scenarios
4. Update this README with new test descriptions
5. Ensure tests are deterministic and reproducible
