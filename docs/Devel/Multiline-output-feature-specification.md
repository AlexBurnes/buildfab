# Multiline Output Feature Specification

## Overview

This document specifies the implementation of a multiline output feature for buildfab's quiet mode (`-q` flag), replacing the current single-line running indicator with a comprehensive multiline status display that shows all jobs and their current status in real-time.

## Current Implementation Analysis

### Current Quiet Mode Behavior
- **Single-line indicator**: Shows `◯ step-name (running...)` with cyan circle
- **Line replacement**: Uses carriage return (`\r`) to overwrite the running line with completion status
- **Sequential display**: Only shows one job at a time as it executes
- **Limited visibility**: Users cannot see the overall progress or status of all jobs

### Current Architecture
- **OrderedOutputManager**: Manages step output in proper sequential order using queue-based approach
- **OrderedStepCallback**: Implements StepCallback interface using the ordered output manager
- **DAG Executor**: Runs jobs in parallel but displays output sequentially in declaration order
- **Step Status Tracking**: Tracks step states (Started, Completed, Shown) with proper synchronization

## Proposed Multiline Output System

### Core Concept
Replace the single-line running indicator with a multiline display that shows all jobs and their current status, updating individual job statuses in real-time using ANSI escape codes for cursor control.

### Key Features
1. **All Jobs Visible**: Display all jobs in the stage at once
2. **Real-time Status Updates**: Update individual job statuses as they change
3. **ANSI Cursor Control**: Use escape codes to update specific lines without redrawing entire screen
4. **Status Icons**: Use consistent icons for different job states
5. **Execution Order Preservation**: Maintain declaration order for job display

## Technical Implementation

### ANSI Escape Codes
```go
const (
    hideCursor   = "\x1b[?25l"    // Hide cursor
    showCursor   = "\x1b[?25h"    // Show cursor
    clearLine    = "\x1b[2K"      // Clear current line
    saveCursor   = "\x1b7"        // Save cursor position
    restoreCursor= "\x1b8"        // Restore cursor position
)

// Move cursor to specific row and column (1-based)
func moveTo(row, col int) string {
    return fmt.Sprintf("\x1b[%d;%dH", row, col)
}
```

### Job Status Icons and Colors
```go
const (
    StatusPending  = "○"  // Gray - Job not started
    StatusRunning  = "◯"  // Cyan - Job currently executing
    StatusSuccess  = "✓"  // Green - Job completed successfully
    StatusWarning  = "!"  // Yellow - Job completed with warnings
    StatusError    = "✗"  // Red - Job failed
    StatusSkipped  = "→"  // Gray - Job skipped
)
```

### Multiline Output Manager

#### Core Data Structure
```go
type MultilineOutputManager struct {
    jobs           []JobDisplay      // Jobs in declaration order
    baseRow        int               // Starting row for job display
    currentRow     int               // Current cursor row
    verboseLevel   int               // Verbosity level
    debug          bool              // Debug mode
    errorOutput    io.Writer         // Output writer
    config         *Config           // Configuration
    configPath     string            // Configuration file path
    interpolatedActions map[string]*Action // Interpolated actions
    mu             *sync.Mutex       // Mutex for thread safety
}

type JobDisplay struct {
    Name        string      // Job name
    Status      JobStatus   // Current status
    Message     string      // Status message
    Duration    time.Duration // Execution duration
    Row         int         // Display row number
    Started     bool        // Whether job has started
    Completed   bool        // Whether job has completed
}
```

#### Key Methods
```go
// Initialize multiline display
func (m *MultilineOutputManager) InitializeDisplay() {
    // Hide cursor, reserve screen space, show all jobs
}

// Update job status (called via callback system from DAG executor)
func (m *MultilineOutputManager) UpdateJobStatus(jobName string, status JobStatus, message string, duration time.Duration) {
    // Find job, update status, redraw line using ANSI escape codes
}

// Clean up display
func (m *MultilineOutputManager) Cleanup() {
    // Show cursor, restore terminal state
}
```

### Integration Points

#### 1. OrderedOutputManager Integration
- **Replace quiet mode logic**: Modify `showStepStart()` method to use multiline display
- **Status updates**: Update `OnStepComplete()` to use multiline status updates
- **Backward compatibility**: Maintain verbose mode behavior unchanged

#### 2. DAG Executor Integration
- **Job registration**: Register all jobs before execution starts
- **Status callbacks**: Receive status updates from executor
- **Parallel execution**: Handle status updates from multiple concurrent jobs

#### 3. Step Callback Integration
- **OrderedStepCallback**: Integrate with existing callback system
- **Status propagation**: Ensure status changes propagate to multiline display
- **Error handling**: Handle errors and warnings appropriately

## User Experience

### Display Layout
```
buildfab 0.20.0
Project buildfab (v0.20.1)

► Running stage: pre-push

  ○ version-check     (pending)
  ○ version-greatest  (pending)
  ○ version-module    (pending)
  ○ run-tests         (running...)
```

### Status Updates
```
  ✓ version-check     executed successfully - in '0.010s'
  ✓ version-greatest  executed successfully - in '0.015s'
  ✓ version-module    executed successfully - in '0.013s'
  ◯ run-tests         (running...)
```

### Error Handling
```
  ✓ version-check     executed successfully - in '0.010s'
  ✗ version-greatest  execute failure - in '0.005s'
  → version-module    skipped (dependency failed: version-greatest)
  → run-tests         skipped (dependency failed: version-greatest)
```

## Implementation Plan

### Phase 1: Core Infrastructure (2-3 days)
1. **Create MultilineOutputManager**: Implement core data structures and ANSI control
2. **Job Registration System**: Register all jobs before execution starts
3. **Basic Status Updates**: Implement status update mechanism
4. **ANSI Escape Code Handling**: Implement cursor control and line updates

### Phase 2: Integration (2-3 days)
1. **OrderedOutputManager Integration**: Replace quiet mode logic
2. **Step Callback Integration**: Integrate with existing callback system
3. **DAG Executor Integration**: Handle parallel job status updates
4. **Error Handling**: Implement proper error and warning display

### Phase 3: Testing and Refinement (1-2 days)
1. **Unit Tests**: Test multiline display functionality
2. **Integration Tests**: Test with real buildfab stages
3. **Edge Case Handling**: Handle terminal resizing, interruption, etc.
4. **Performance Testing**: Ensure minimal overhead

### Phase 4: Documentation and Release (1 day)
1. **User Documentation**: Update README and help text
2. **Developer Documentation**: Document implementation details
3. **Release Preparation**: Version bump and changelog updates

## Technical Considerations

### Thread Safety
- **Mutex Protection**: All display operations protected by mutex
- **Atomic Updates**: Status updates are atomic to prevent race conditions
- **Ordered Updates**: Maintain proper ordering of status changes

### Terminal Compatibility
- **TTY Detection**: Detect if output is to a terminal
- **Fallback Mode**: Fall back to simple output for non-TTY environments
- **Terminal Size**: Handle terminal resizing gracefully

### Performance
- **Minimal Overhead**: ANSI operations should be fast
- **Efficient Updates**: Only update changed lines
- **Memory Usage**: Keep job display data minimal

### Error Handling
- **Terminal Errors**: Handle cases where ANSI codes fail
- **Interruption**: Handle Ctrl+C gracefully
- **Partial Updates**: Handle cases where display is partially updated

## Configuration Options

### Display Settings
```go
type MultilineDisplayConfig struct {
    Enabled        bool   // Enable multiline display
    MaxJobs        int    // Maximum number of jobs to display
    ShowDuration   bool   // Show execution duration
    ShowMessages   bool   // Show status messages
}
```

### CLI Options
- `--progress plain`: Disable multiline output, use simple single-line mode
- `--progress multiline`: Enable multiline output (default in quiet mode)
- `-q` or `--quiet`: Enable quiet mode with multiline display

### Environment Variables
- `BUILDFAB_MULTILINE_DISPLAY`: Enable/disable multiline display
- `BUILDFAB_DISPLAY_MAX_JOBS`: Maximum jobs to display

## Testing Strategy

### Unit Tests
- **ANSI Code Generation**: Test escape code generation
- **Status Updates**: Test status update logic
- **Job Registration**: Test job registration system
- **Error Handling**: Test error scenarios

### Integration Tests
- **Matrix Stream Test**: Use `tests/test_matrix_stream.yml` for comprehensive testing
  - **3 Matrix Jobs**: Test with linux, windows, macos matrix jobs running in parallel
  - **Streaming Output**: Test real-time status updates with 5-second streaming commands
  - **Parallel Execution**: Test `max_parallel: 2` with 3 jobs (2 parallel, 1 queued)
  - **Backward Compatibility**: Test both quiet mode (`-q`) and verbose mode (`-v`)
- **Progress Mode Testing**: Test `--progress plain` vs `--progress multiline` options
- **Error Scenarios**: Test error handling and recovery
- **Terminal Compatibility**: Test with different terminal types

### Manual Testing
- **User Experience**: Test actual user workflows
- **Performance**: Test with large numbers of jobs
- **Edge Cases**: Test interruption, terminal resizing, etc.

## Success Criteria

### Functional Requirements
1. **All Jobs Visible**: All jobs in a stage are displayed simultaneously
2. **Real-time Updates**: Job statuses update in real-time as they change
3. **Proper Ordering**: Jobs are displayed in declaration order
4. **Status Accuracy**: Status icons and messages accurately reflect job state
5. **Error Handling**: Errors and warnings are displayed appropriately

### Performance Requirements
1. **Minimal Overhead**: Display updates should not significantly impact execution time
2. **Responsive Updates**: Status changes should be visible within 100ms
3. **Memory Efficient**: Display should not consume excessive memory
4. **Terminal Friendly**: Should work with various terminal types and sizes

### User Experience Requirements
1. **Clear Status**: Job status should be immediately clear from icons and colors
2. **Non-intrusive**: Display should not interfere with job execution
3. **Graceful Degradation**: Should fall back gracefully on unsupported terminals
4. **Consistent**: Should be consistent with existing buildfab output style

## Risks and Mitigation

### Technical Risks
1. **ANSI Compatibility**: Some terminals may not support ANSI escape codes
   - **Mitigation**: Implement TTY detection and fallback mode
2. **Terminal Resizing**: Terminal size changes during execution
   - **Mitigation**: Handle SIGWINCH signal and redraw display
3. **Performance Impact**: ANSI operations may slow down execution
   - **Mitigation**: Implement efficient update batching and profiling

### User Experience Risks
1. **Confusing Display**: Users may find multiline display confusing
   - **Mitigation**: Make it opt-in via configuration, provide clear documentation
     Add option progress: plain, to disable multiline output.
2. **Terminal Interference**: Display may interfere with other terminal operations
   - **Mitigation**: Implement proper cleanup and cursor restoration
3. **Accessibility**: ANSI codes may not work with screen readers
   - **Mitigation**: Provide fallback text-only mode

## Future Enhancements

### Advanced Features
1. **Progress Bars**: Show progress bars for long-running jobs
2. **Job Dependencies**: Visual representation of job dependencies
3. **Timeline View**: Show execution timeline with timestamps
4. **Interactive Mode**: Allow user interaction with running jobs

### Configuration Options
1. **Custom Icons**: Allow users to customize status icons
2. **Color Themes**: Support different color themes
3. **Display Layouts**: Multiple display layout options
4. **Export Options**: Export execution logs in various formats

## Conclusion

The multiline output feature will significantly improve the user experience in quiet mode by providing comprehensive visibility into all job statuses in real-time. The implementation uses proven ANSI escape code techniques while maintaining compatibility with the existing buildfab architecture.

The feature is designed to be robust, performant, and user-friendly, with proper error handling and graceful degradation for unsupported environments. The phased implementation approach ensures thorough testing and refinement before release.
