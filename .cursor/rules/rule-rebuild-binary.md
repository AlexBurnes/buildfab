# Rebuild Binary After Code Changes Rule

## Rule Description
Always rebuild the binary after making any code changes before testing with examples or running tests.

## When to Apply
- After making ANY Go code changes
- Before running examples or tests
- Before demonstrating functionality to user
- After any modification to source code

## Required Steps
1. **Build the binary**:
   ```bash
   go build -o bin/buildfab ./cmd/buildfab
   ```

2. **Verify build success**:
   - Check exit code is 0
   - Ensure no compilation errors

3. **Test the changes**:
   - Run examples: `./bin/buildfab run test-container --config examples/container-working-test.yml --verbose`
   - Run tests: `./bin/buildfab run-tests`

## Common Mistakes to Avoid
- Forgetting to rebuild after code changes
- Testing with old binary that doesn't reflect changes
- Assuming changes are active without rebuilding
- Not verifying build success before testing

## Integration with Workflow
This rule should be applied:
- After every code modification
- Before any testing or demonstration
- As part of the development cycle
- When user reports issues with functionality

## Examples

### Correct Workflow
```bash
# 1. Make code changes
# 2. Build binary
go build -o bin/buildfab ./cmd/buildfab

# 3. Test changes
./bin/buildfab run test-container --config examples/container-working-test.yml --verbose
```

### Incorrect Workflow
```bash
# 1. Make code changes
# 2. Test with old binary (WRONG!)
./bin/buildfab run test-container --config examples/container-working-test.yml --verbose
```

## Enforcement
- Always apply this rule when making code changes
- Verify binary is rebuilt before any testing
- Include rebuild step in all development workflows
- Document rebuild requirement in commit messages when applicable
