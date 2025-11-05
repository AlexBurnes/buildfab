# Container Feature Implementation Steps

## Document Purpose

This document provides detailed, step-by-step implementation instructions for the Docker and Podman container feature in buildfab, based on the clarified requirements.

## Implementation Overview

The container feature will be implemented in 4 phases over 8 weeks, with incremental development and testing at each phase.

## Phase 1: Basic Container Execution (Weeks 1-2)

### Week 1: Basic Container Execution

#### Step 1.1: Create Container Package Structure
```bash
mkdir -p pkg/buildfab/container
mkdir -p internal/container
mkdir -p tests/container
```

#### Step 1.2: Define Container Data Structures
Create `pkg/buildfab/container/types.go`:
```go
package container

import "time"

// ContainerConfig represents container configuration
type ContainerConfig struct {
    Engine     string            `yaml:"engine"`
    Image      ContainerImage    `yaml:"image"`
    Workdir    string            `yaml:"workdir"`
    CPUs       []int             `yaml:"cpus"`
    Memory     string            `yaml:"memory"`
    Mounts     []ContainerMount  `yaml:"mounts"`
    Artifacts  ContainerArtifacts `yaml:"artifacts"`
    Env        map[string]string `yaml:"env"`
    EnvFile    string            `yaml:"env_file"`
    User       string            `yaml:"user"`
    Network    string            `yaml:"network"`
    Cache      map[string]string `yaml:"cache"`
    RunStage   string            `yaml:"run_stage"`
    RunAction  string            `yaml:"run_action"`
    Run        []string          `yaml:"run"`
}

type ContainerImage struct {
    From  string           `yaml:"from"`
    Build *ContainerBuild  `yaml:"build"`
}

type ContainerBuild struct {
    Dockerfile string            `yaml:"dockerfile"`
    Context    string            `yaml:"context"`
    Args       map[string]string `yaml:"args"`
}

type ContainerMount struct {
    Type   string `yaml:"type"`
    Source string `yaml:"source"`
    Target string `yaml:"target"`
    RO     bool   `yaml:"ro"`
}

type ContainerArtifacts struct {
    Output string   `yaml:"output"`
    Path   []string `yaml:"path"`
}
```

#### Step 1.3: Create Container Engine Interface
Create `pkg/buildfab/container/engine.go`:
```go
package container

import (
    "context"
    "io"
)

// Engine represents a container engine interface
type Engine interface {
    // Image operations
    PullImage(ctx context.Context, image string) error
    ImageExists(ctx context.Context, image string) (bool, error)
    BuildImage(ctx context.Context, config ContainerBuild) (string, error)
    
    // Container operations
    RunContainer(ctx context.Context, config ContainerConfig) (*ContainerResult, error)
    StopContainer(ctx context.Context, containerID string) error
    RemoveContainer(ctx context.Context, containerID string) error
    
    // Utility operations
    DetectEngine() bool
    GetEngineName() string
}

// ContainerResult represents the result of container execution
type ContainerResult struct {
    ContainerID string
    ExitCode    int
    Output      string
    Error       string
    Artifacts   []string
}
```

#### Step 1.4: Implement Docker Engine
Create `internal/container/docker.go`:
```go
package container

import (
    "context"
    "fmt"
    "os/exec"
    "strings"
)

type DockerEngine struct {
    binary string
}

func NewDockerEngine() *DockerEngine {
    return &DockerEngine{
        binary: "docker",
    }
}

func (d *DockerEngine) DetectEngine() bool {
    cmd := exec.Command(d.binary, "version")
    return cmd.Run() == nil
}

func (d *DockerEngine) GetEngineName() string {
    return "docker"
}

func (d *DockerEngine) PullImage(ctx context.Context, image string) error {
    cmd := exec.CommandContext(ctx, d.binary, "pull", image)
    return cmd.Run()
}

func (d *DockerEngine) ImageExists(ctx context.Context, image string) (bool, error) {
    cmd := exec.CommandContext(ctx, d.binary, "image", "inspect", image)
    err := cmd.Run()
    return err == nil, nil
}

func (d *DockerEngine) BuildImage(ctx context.Context, config ContainerBuild) (string, error) {
    // Implementation for building images
    // Return generated image tag
    return "", fmt.Errorf("not implemented")
}

func (d *DockerEngine) RunContainer(ctx context.Context, config ContainerConfig) (*ContainerResult, error) {
    // Implementation for running containers
    return nil, fmt.Errorf("not implemented")
}

func (d *DockerEngine) StopContainer(ctx context.Context, containerID string) error {
    cmd := exec.CommandContext(ctx, d.binary, "stop", containerID)
    return cmd.Run()
}

func (d *DockerEngine) RemoveContainer(ctx context.Context, containerID string) error {
    cmd := exec.CommandContext(ctx, d.binary, "rm", containerID)
    return cmd.Run()
}
```

#### Step 1.5: Implement Podman Engine
Create `internal/container/podman.go`:
```go
package container

import (
    "context"
    "fmt"
    "os/exec"
)

type PodmanEngine struct {
    binary string
}

func NewPodmanEngine() *PodmanEngine {
    return &PodmanEngine{
        binary: "podman",
    }
}

func (p *PodmanEngine) DetectEngine() bool {
    cmd := exec.Command(p.binary, "version")
    return cmd.Run() == nil
}

func (p *PodmanEngine) GetEngineName() string {
    return "podman"
}

// Implement same interface methods as DockerEngine
// with podman-specific command translations
```

#### Step 1.6: Create Container Manager
Create `pkg/buildfab/container/manager.go`:
```go
package container

import (
    "context"
    "fmt"
)

type Manager struct {
    engine Engine
}

func NewManager() (*Manager, error) {
    // Try Docker first, then Podman
    if docker := NewDockerEngine(); docker.DetectEngine() {
        return &Manager{engine: docker}, nil
    }
    if podman := NewPodmanEngine(); podman.DetectEngine() {
        return &Manager{engine: podman}, nil
    }
    return nil, fmt.Errorf("no container engine available")
}

func (m *Manager) ExecuteAction(ctx context.Context, config ContainerConfig) (*ContainerResult, error) {
    // Validate configuration
    if err := m.validateConfig(config); err != nil {
        return nil, err
    }
    
    // Pull or build image
    if config.Image.Build != nil {
        image, err := m.engine.BuildImage(ctx, *config.Image.Build)
        if err != nil {
            return nil, err
        }
        config.Image.From = image
    } else {
        if exists, _ := m.engine.ImageExists(ctx, config.Image.From); !exists {
            if err := m.engine.PullImage(ctx, config.Image.From); err != nil {
                return nil, err
            }
        }
    }
    
    // Run container
    return m.engine.RunContainer(ctx, config)
}

func (m *Manager) validateConfig(config ContainerConfig) error {
    // Validate required fields
    if config.Image.From == "" && config.Image.Build == nil {
        return fmt.Errorf("image.from or image.build must be specified")
    }
    
    // Validate mounts
    for _, mount := range config.Mounts {
        if mount.Type == "bind" {
            // Check if source directory exists
            // Implementation here
        }
    }
    
    return nil
}
```

#### Step 1.7: Create Basic Tests
Create `tests/container/basic_test.go`:
```go
package container

import (
    "context"
    "testing"
)

func TestPodmanEngineDetection(t *testing.T) {
    engine := NewPodmanEngine()
    if !engine.DetectEngine() {
        t.Skip("Podman not available")
    }
    
    if engine.GetEngineName() != "podman" {
        t.Errorf("Expected engine name 'podman', got '%s'", engine.GetEngineName())
    }
}

func TestDockerEngineDetection(t *testing.T) {
    engine := NewDockerEngine()
    if !engine.DetectEngine() {
        t.Skip("Docker not available")
    }
    
    if engine.GetEngineName() != "docker" {
        t.Errorf("Expected engine name 'docker', got '%s'", engine.GetEngineName())
    }
}

func TestManagerCreation(t *testing.T) {
    manager, err := NewManager()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if manager == nil {
        t.Error("Expected manager, got nil")
    }
}
```

### Week 2: Action and Stage Execution

#### Step 2.1: Integrate with Action System
Update `pkg/buildfab/buildfab.go` to support container actions:
```go
// Add container field to Action struct
type Action struct {
    Name        string           `yaml:"name"`
    Description string           `yaml:"description"`
    Run         string           `yaml:"run"`
    Uses        string           `yaml:"uses"`
    Container   *ContainerConfig `yaml:"container"`
    // ... existing fields
}
```

#### Step 2.2: Create Container Action Runner
Create `internal/container/runner.go`:
```go
package container

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
)

type ContainerRunner struct {
    manager *Manager
}

func NewContainerRunner() (*ContainerRunner, error) {
    manager, err := NewManager()
    if err != nil {
        return nil, err
    }
    return &ContainerRunner{manager: manager}, nil
}

func (r *ContainerRunner) RunAction(ctx context.Context, config ContainerConfig) error {
    // Copy buildfab binary and config to container
    if err := r.prepareContainer(ctx, config); err != nil {
        return err
    }
    
    // Execute container
    result, err := r.manager.ExecuteAction(ctx, config)
    if err != nil {
        return err
    }
    
    // Collect artifacts
    if err := r.collectArtifacts(result, config); err != nil {
        return err
    }
    
    return nil
}

func (r *ContainerRunner) prepareContainer(ctx context.Context, config ContainerConfig) error {
    // Only copy buildfab binary and config if using run_stage or run_action
    if config.RunStage != "" || config.RunAction != "" {
        // Copy buildfab binary to container
        // Copy configuration files
    }
    // Set up environment variables
    return nil
}

func (r *ContainerRunner) collectArtifacts(result *ContainerResult, config ContainerConfig) error {
    // Collect artifacts from container to host
    for _, artifact := range config.Artifacts.Path {
        // Copy artifact from container to host
    }
    return nil
}
```

#### Step 2.3: Update Action Execution
Update `pkg/buildfab/buildfab.go` to handle container actions:
```go
func (r *Runner) runActionInternal(ctx context.Context, action *Action, opts *RunOptions) (*Result, error) {
    // Check if action has container configuration
    if action.Container != nil {
        return r.runContainerAction(ctx, action, opts)
    }
    
    // ... existing action execution logic
}

func (r *Runner) runContainerAction(ctx context.Context, action *Action, opts *RunOptions) (*Result, error) {
    runner, err := NewContainerRunner()
    if err != nil {
        return nil, fmt.Errorf("failed to create container runner: %w", err)
    }
    
    if err := runner.RunAction(ctx, *action.Container); err != nil {
        return &Result{
            Status:  StatusError,
            Message: err.Error(),
        }, nil
    }
    
    return &Result{
        Status:  StatusOK,
        Message: "Container action completed successfully",
    }, nil
}
```

#### Step 2.4: Create Integration Tests
Create `tests/container/integration_test.go`:
```go
package container

import (
    "context"
    "testing"
)

func TestContainerActionExecution(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container integration test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        Run: []string{"echo", "Hello from container"},
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("Container action failed: %v", err)
    }
}

func TestContainerRunStageExecution(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container run_stage test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        RunStage: "test-stage",
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("Container run_stage failed: %v", err)
    }
}
```

## Phase 2: Mount and Resource Support (Weeks 3-4)

### Week 3: Mount Support

#### Step 3.1: Implement Mount Validation
Update `pkg/buildfab/container/manager.go`:
```go
func (m *Manager) validateMounts(config ContainerConfig) error {
    for _, mount := range config.Mounts {
        if mount.Type == "bind" {
            // Check if source directory exists
            if _, err := os.Stat(mount.Source); os.IsNotExist(err) {
                return fmt.Errorf("mount source directory does not exist: %s", mount.Source)
            }
        }
    }
    return nil
}
```

#### Step 3.2: Implement Mount Command Generation
Update `internal/container/docker.go`:
```go
func (d *DockerEngine) generateMountArgs(mounts []ContainerMount) []string {
    var args []string
    for _, mount := range mounts {
        if mount.Type == "bind" {
            mountArg := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
            if mount.RO {
                mountArg += ":ro"
            }
            args = append(args, "-v", mountArg)
        }
    }
    return args
}
```

#### Step 3.3: Create Mount Tests
Create `tests/container/mount_test.go`:
```go
package container

import (
    "os"
    "testing"
)

func TestMountValidation(t *testing.T) {
    // Create temporary directory for testing
    tempDir := t.TempDir()
    
    config := ContainerConfig{
        Mounts: []ContainerMount{
            {
                Type:   "bind",
                Source: tempDir,
                Target: "/test",
                RO:     false,
            },
        },
    }
    
    manager, err := NewManager()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := manager.validateMounts(config); err != nil {
        t.Errorf("Mount validation failed: %v", err)
    }
}

func TestMountValidationFailure(t *testing.T) {
    config := ContainerConfig{
        Mounts: []ContainerMount{
            {
                Type:   "bind",
                Source: "/nonexistent/directory",
                Target: "/test",
                RO:     false,
            },
        },
    }
    
    manager, err := NewManager()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := manager.validateMounts(config); err == nil {
        t.Error("Expected mount validation to fail for nonexistent directory")
    }
}
```

### Week 4: CPU and Memory Limiting

#### Step 4.1: Implement Resource Limit Parsing
Create `pkg/buildfab/container/resources.go`:
```go
package container

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
)

// ParseMemoryLimit parses human-readable memory limits
func ParseMemoryLimit(memory string) (int64, error) {
    // Remove spaces and convert to lowercase
    memory = strings.TrimSpace(strings.ToLower(memory))
    
    // Regular expression to match memory format (e.g., "4G", "500M", "1024")
    re := regexp.MustCompile(`^(\d+)([kmg]?)$`)
    matches := re.FindStringSubmatch(memory)
    
    if len(matches) != 3 {
        return 0, fmt.Errorf("invalid memory format: %s", memory)
    }
    
    value, err := strconv.ParseInt(matches[1], 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid memory value: %s", matches[1])
    }
    
    unit := matches[2]
    switch unit {
    case "k":
        return value * 1024, nil
    case "m":
        return value * 1024 * 1024, nil
    case "g":
        return value * 1024 * 1024 * 1024, nil
    case "":
        return value, nil
    default:
        return 0, fmt.Errorf("invalid memory unit: %s", unit)
    }
}

// ParseCPULimit parses CPU limits
func ParseCPULimit(cpus []int) (float64, string, error) {
    if len(cpus) == 0 {
        return 0, "", nil
    }
    
    // Calculate CPU count
    cpuCount := float64(len(cpus))
    
    // Generate CPU set string
    cpuSet := make([]string, len(cpus))
    for i, cpu := range cpus {
        cpuSet[i] = strconv.Itoa(cpu)
    }
    
    return cpuCount, strings.Join(cpuSet, ","), nil
}
```

#### Step 4.2: Implement Resource Limit Command Generation
Update `internal/container/docker.go`:
```go
func (d *DockerEngine) generateResourceArgs(config ContainerConfig) ([]string, error) {
    var args []string
    
    // CPU limits
    if len(config.CPUs) > 0 {
        cpuCount, cpuSet, err := ParseCPULimit(config.CPUs)
        if err != nil {
            return nil, err
        }
        
        args = append(args, "--cpus", fmt.Sprintf("%.1f", cpuCount))
        args = append(args, "--cpuset-cpus", cpuSet)
    }
    
    // Memory limits
    if config.Memory != "" {
        memoryBytes, err := ParseMemoryLimit(config.Memory)
        if err != nil {
            return nil, err
        }
        
        args = append(args, "-m", strconv.FormatInt(memoryBytes, 10))
    }
    
    return args, nil
}
```

#### Step 4.3: Create Resource Tests
Create `tests/container/resources_test.go`:
```go
package container

import (
    "testing"
)

func TestParseMemoryLimit(t *testing.T) {
    tests := []struct {
        input    string
        expected int64
        hasError bool
    }{
        {"4G", 4 * 1024 * 1024 * 1024, false},
        {"500M", 500 * 1024 * 1024, false},
        {"1024", 1024, false},
        {"invalid", 0, true},
        {"", 0, true},
    }
    
    for _, test := range tests {
        result, err := ParseMemoryLimit(test.input)
        if test.hasError {
            if err == nil {
                t.Errorf("Expected error for input '%s', got none", test.input)
            }
        } else {
            if err != nil {
                t.Errorf("Unexpected error for input '%s': %v", test.input, err)
            }
            if result != test.expected {
                t.Errorf("Expected %d for input '%s', got %d", test.expected, test.input, result)
            }
        }
    }
}

func TestParseCPULimit(t *testing.T) {
    tests := []struct {
        input        []int
        expectedCPU  float64
        expectedSet  string
        hasError     bool
    }{
        {[]int{0, 1}, 2.0, "0,1", false},
        {[]int{0}, 1.0, "0", false},
        {[]int{}, 0, "", false},
    }
    
    for _, test := range tests {
        cpuCount, cpuSet, err := ParseCPULimit(test.input)
        if test.hasError {
            if err == nil {
                t.Errorf("Expected error for input %v, got none", test.input)
            }
        } else {
            if err != nil {
                t.Errorf("Unexpected error for input %v: %v", test.input, err)
            }
            if cpuCount != test.expectedCPU {
                t.Errorf("Expected CPU count %f for input %v, got %f", test.expectedCPU, test.input, cpuCount)
            }
            if cpuSet != test.expectedSet {
                t.Errorf("Expected CPU set '%s' for input %v, got '%s'", test.expectedSet, test.input, cpuSet)
            }
        }
    }
}
```

## Phase 3: Image Building and Advanced Features (Weeks 5-6)

### Week 5: Container Image Building

#### Step 5.1: Implement Image Building
Update `internal/container/docker.go`:
```go
func (d *DockerEngine) BuildImage(ctx context.Context, config ContainerBuild) (string, error) {
    // Generate unique tag
    tag := fmt.Sprintf("buildfab-%s-%d", config.Context, time.Now().Unix())
    
    // Build command
    cmd := exec.CommandContext(ctx, d.binary, "build")
    cmd.Args = append(cmd.Args, "-t", tag)
    
    // Add build args
    for key, value := range config.Args {
        cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
    }
    
    // Add dockerfile and context
    cmd.Args = append(cmd.Args, "-f", config.Dockerfile, config.Context)
    
    // Run build
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("failed to build image: %w", err)
    }
    
    return tag, nil
}
```

#### Step 5.2: Create Image Building Tests
Create `tests/container/build_test.go`:
```go
package container

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestImageBuilding(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container build test")
    }
    
    // Create temporary Dockerfile
    tempDir := t.TempDir()
    dockerfile := filepath.Join(tempDir, "Dockerfile")
    
    content := `FROM alpine:latest
RUN echo "Hello from container"
`
    
    if err := os.WriteFile(dockerfile, []byte(content), 0644); err != nil {
        t.Fatalf("Failed to create Dockerfile: %v", err)
    }
    
    config := ContainerBuild{
        Dockerfile: dockerfile,
        Context:    tempDir,
        Args: map[string]string{
            "BASE": "alpine:latest",
        },
    }
    
    // Try Podman first, then Docker
    engine := NewPodmanEngine()
    if !engine.DetectEngine() {
        engine = NewDockerEngine()
        if !engine.DetectEngine() {
            t.Skip("No container engine available")
        }
    }
    
    tag, err := engine.BuildImage(context.Background(), config)
    if err != nil {
        t.Errorf("Image building failed: %v", err)
    }
    
    if tag == "" {
        t.Error("Expected non-empty tag")
    }
}
```

### Week 6: Artifact Collection

#### Step 6.1: Implement Artifact Collection
Update `internal/container/runner.go`:
```go
func (r *ContainerRunner) collectArtifacts(containerID string, config ContainerConfig) error {
    if len(config.Artifacts.Path) == 0 {
        return nil
    }
    
    // Create output directory
    if err := os.MkdirAll(config.Artifacts.Output, 0755); err != nil {
        return fmt.Errorf("failed to create output directory: %w", err)
    }
    
    // Copy artifacts from container
    for _, artifact := range config.Artifacts.Path {
        if err := r.copyArtifact(containerID, artifact, config.Artifacts.Output); err != nil {
            return fmt.Errorf("failed to copy artifact %s: %w", artifact, err)
        }
    }
    
    return nil
}

func (r *ContainerRunner) copyArtifact(containerID, artifact, outputDir string) error {
    // Use docker cp to copy artifact from container
    cmd := exec.Command("docker", "cp", fmt.Sprintf("%s:%s", containerID, artifact), outputDir)
    return cmd.Run()
}
```

#### Step 6.2: Create Artifact Collection Tests
Create `tests/container/artifacts_test.go`:
```go
package container

import (
    "os"
    "path/filepath"
    "testing"
)

func TestArtifactCollection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping artifact collection test")
    }
    
    // Create temporary output directory
    outputDir := t.TempDir()
    
    config := ContainerConfig{
        Artifacts: ContainerArtifacts{
            Output: outputDir,
            Path: []string{
                "/tmp/test.txt",
                "/tmp/test_dir/",
            },
        },
    }
    
    // Test artifact collection logic
    // (This would require a running container with test artifacts)
    // Implementation depends on container execution setup
}
```

## Phase 4: Testing and Documentation (Weeks 7-8)

### Week 7: Comprehensive Testing

#### Step 7.1: Create End-to-End Tests
Create `tests/container/e2e_test.go`:
```go
package container

import (
    "context"
    "testing"
)

func TestEndToEndContainerExecution(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping end-to-end container test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        Workdir: "/test",
        Mounts: []ContainerMount{
            {
                Type:   "bind",
                Source: t.TempDir(),
                Target: "/test",
                RO:     false,
            },
        },
        CPUs:   []int{0, 1},
        Memory: "512M",
        Run:    []string{"echo", "Hello from container"},
        Artifacts: ContainerArtifacts{
            Output: t.TempDir(),
            Path:   []string{"/test/output.txt"},
        },
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("End-to-end container execution failed: %v", err)
    }
}

func TestEndToEndContainerRunStage(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping end-to-end container run_stage test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        Workdir: "/test",
        Mounts: []ContainerMount{
            {
                Type:   "bind",
                Source: t.TempDir(),
                Target: "/test",
                RO:     false,
            },
        },
        RunStage: "test-stage",
        Artifacts: ContainerArtifacts{
            Output: t.TempDir(),
            Path:   []string{"/test/output.txt"},
        },
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("End-to-end container run_stage failed: %v", err)
    }
}

func TestContainerMatrixPlatformDetection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container matrix platform detection test")
    }
    
    // Test platform detection across different container images
    // This test uses run_action which copies buildfab binary and config into container
    images := []string{"alpine:latest", "ubuntu:22.04", "debian:12"}
    
    for _, image := range images {
        t.Run(image, func(t *testing.T) {
            config := ContainerConfig{
                Image: ContainerImage{
                    From: image,
                },
                Workdir: "/test",
                Mounts: []ContainerMount{
                    {
                        Type:   "bind",
                        Source: t.TempDir(),
                        Target: "/test",
                        RO:     false,
                    },
                },
                RunAction: "platform-view",
            }
            
            runner, err := NewContainerRunner()
            if err != nil {
                t.Skip("No container engine available")
            }
            
            if err := runner.RunAction(context.Background(), config); err != nil {
                t.Errorf("Container matrix platform detection failed for %s: %v", image, err)
            }
        })
    }
}
```

#### Step 7.1.1: Create Matrix Integration Tests
Create `tests/container/matrix_test.go`:
```go
package container

import (
    "context"
    "testing"
)

func TestContainerMatrixIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container matrix integration test")
    }
    
    // Test matrix integration with container feature
    // This test uses run_action which copies buildfab binary and config into container
    // Tests the integration between matrix and container features using platform detection
    
    matrixImages := []string{"alpine:latest", "ubuntu:22.04", "debian:12"}
    
    for _, image := range matrixImages {
        t.Run("matrix-"+image, func(t *testing.T) {
            config := ContainerConfig{
                Image: ContainerImage{
                    From: image,
                },
                Workdir: "/test",
                Mounts: []ContainerMount{
                    {
                        Type:   "bind",
                        Source: t.TempDir(),
                        Target: "/test",
                        RO:     false,
                    },
                },
                RunAction: "platform-view",
                Artifacts: ContainerArtifacts{
                    Output: t.TempDir(),
                    Path:   []string{"/test/matrix-platform-output.txt"},
                },
            }
            
            runner, err := NewContainerRunner()
            if err != nil {
                t.Skip("No container engine available")
            }
            
            if err := runner.RunAction(context.Background(), config); err != nil {
                t.Errorf("Container matrix integration failed for %s: %v", image, err)
            }
        })
    }
}
```

#### Step 7.2: Create Performance Tests
Create `tests/container/performance_test.go`:
```go
package container

import (
    "context"
    "testing"
    "time"
)

func TestContainerPerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container performance test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        Run: []string{"sleep", "1"},
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    start := time.Now()
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("Container performance test failed: %v", err)
    }
    duration := time.Since(start)
    
    // Container should complete within reasonable time
    if duration > 10*time.Second {
        t.Errorf("Container execution took too long: %v", duration)
    }
}

func TestContainerRunStagePerformance(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping container run_stage performance test")
    }
    
    config := ContainerConfig{
        Image: ContainerImage{
            From: "alpine:latest",
        },
        RunStage: "test-stage",
    }
    
    runner, err := NewContainerRunner()
    if err != nil {
        t.Skip("No container engine available")
    }
    
    start := time.Now()
    if err := runner.RunAction(context.Background(), config); err != nil {
        t.Errorf("Container run_stage performance test failed: %v", err)
    }
    duration := time.Since(start)
    
    // Container should complete within reasonable time
    if duration > 15*time.Second {
        t.Errorf("Container run_stage execution took too long: %v", duration)
    }
}
```

### Week 8: Documentation and User Guides

#### Step 8.1: Create User Documentation
Create `docs/Container-feature-user-guide.md`:
```markdown
# Container Feature User Guide

## Overview

The container feature allows you to execute buildfab actions and stages inside Docker or Podman containers, providing isolated execution environments for your build and test processes.

## Basic Usage

### Simple Container Action

```yaml
actions:
  - name: test-in-container
    container:
      image:
        from: alpine:latest
      run:
        - echo "Hello from container"
```

### Container with Mounts

```yaml
actions:
  - name: build-in-container
    container:
      image:
        from: ubuntu:22.04
      workdir: /src
      mounts:
        - type: bind
          source: .
          target: /src
          ro: true
      run:
        - make build
```

### Container with Resource Limits

```yaml
actions:
  - name: resource-limited-build
    container:
      image:
        from: ubuntu:22.04
      cpus: [0, 1]  # Use CPUs 0 and 1
      memory: 2G    # Limit memory to 2GB
      run:
        - make build
```

### Container Matrix with Platform Detection

```yaml
stages:
  test-platforms:
    - action: platform-test
      matrix:
        image: ["alpine:latest", "ubuntu:22.04", "debian:12"]
      strategy:
        max_parallel: 3
        fail_fast: false

actions:
  - name: platform-view
    description: Display platform information
    run:
      - echo "=== Platform Detection Test ==="
      - echo "Container Image: ${{ matrix.image }}"
      - echo "Platform: ${{platform}}"
      - echo "Architecture: '${{arch}}'"
      - echo "OS: '${{os}}'"
      - echo "OS Version: '${{os_version}}'"
      - echo "CPU Count: ${{cpu}}"
      - echo "================================"

  - name: platform-test
    description: Test platform detection in ${{ matrix.image }}
    container:
      image:
        from: ${{ matrix.image }}
      workdir: /test
      mounts:
        - type: bind
          source: .
          target: /test
          ro: false
      run_action: platform-view  # Uses run_action to copy buildfab binary and config
      artifacts:
        output: ./platform-results
        path:
          - /test/platform-output.txt
```

## Advanced Configuration

### Container Execution Modes

#### Using `run` (Shell Commands)
```yaml
actions:
  - name: shell-command
    container:
      image:
        from: alpine:latest
      run:
        - echo "Hello from container"
        - ls -la
        - cat /etc/os-release
```
- **No buildfab binary copied** into container
- **No configuration copied** into container
- **Direct shell command execution**
- **Use for**: Simple shell commands, system utilities, package installations

#### Using `run_action` (Buildfab Actions)
```yaml
actions:
  - name: buildfab-action
    container:
      image:
        from: alpine:latest
      run_action: platform-view  # Executes: buildfab -c config.yml action platform-view
```
- **Buildfab binary copied** into container
- **Configuration copied** into container
- **Executes buildfab action** inside container
- **Use for**: Buildfab actions, complex workflows, matrix integration

#### Using `run_stage` (Buildfab Stages)
```yaml
actions:
  - name: buildfab-stage
    container:
      image:
        from: ubuntu:22.04
      run_stage: test-stage  # Executes: buildfab -c config.yml run test-stage
```
- **Buildfab binary copied** into container
- **Configuration copied** into container
- **Executes buildfab stage** inside container
- **Use for**: Complete build stages, complex workflows

### Building Custom Images

```yaml
actions:
  - name: custom-build
    container:
      image:
        build:
          dockerfile: ci/Dockerfile.build
          context: .
          args:
            BASE: ubuntu:22.04
      run:
        - make build
```

### Artifact Collection

```yaml
actions:
  - name: build-with-artifacts
    container:
      image:
        from: ubuntu:22.04
      artifacts:
        output: ./dist
        path:
          - /usr/local/bin/myapp
          - /build/dist/
      run:
        - make build
```

## Troubleshooting

### Debug Container Execution

Use `-vv` verbosity to see the full container command (Docker or Podman):

```bash
buildfab run my-action -vv
```

### Manual Container Testing

If container execution fails, you can run the container manually using the provided container command from the error output. The system will show the exact command to reproduce the issue.

## Best Practices

1. **Use specific image tags** instead of `latest` for reproducible builds
2. **Mount source code read-only** to prevent accidental modifications
3. **Set appropriate resource limits** to prevent resource exhaustion
4. **Use artifact collection** to gather build outputs
5. **Test containers locally** before using in CI/CD
6. **Use Podman for testing** (no superuser access required)
7. **Only install buildfab in containers when using run_stage or run_action**

## Examples

See the `examples/container/` directory for complete working examples.
```

#### Step 8.2: Create API Documentation
Create `docs/Container-feature-api-reference.md`:
```markdown
# Container Feature API Reference

## Data Structures

### ContainerConfig

```go
type ContainerConfig struct {
    Engine     string            `yaml:"engine"`
    Image      ContainerImage    `yaml:"image"`
    Workdir    string            `yaml:"workdir"`
    CPUs       []int             `yaml:"cpus"`
    Memory     string            `yaml:"memory"`
    Mounts     []ContainerMount  `yaml:"mounts"`
    Artifacts  ContainerArtifacts `yaml:"artifacts"`
    Env        map[string]string `yaml:"env"`
    EnvFile    string            `yaml:"env_file"`
    User       string            `yaml:"user"`
    Network    string            `yaml:"network"`
    Cache      map[string]string `yaml:"cache"`
    RunStage   string            `yaml:"run_stage"`
    RunAction  string            `yaml:"run_action"`
    Run        []string          `yaml:"run"`
}
```

## Functions

### NewManager

```go
func NewManager() (*Manager, error)
```

Creates a new container manager, automatically detecting available container engines.

### ExecuteAction

```go
func (m *Manager) ExecuteAction(ctx context.Context, config ContainerConfig) (*ContainerResult, error)
```

Executes a container action with the given configuration.

## Error Handling

Container execution errors are returned as standard Go errors and can be handled using the existing buildfab error handling mechanisms.

## Examples

### Basic Container Execution

```go
config := ContainerConfig{
    Image: ContainerImage{
        From: "alpine:latest",
    },
    Run: []string{"echo", "Hello from container"},
}

manager, err := NewManager()
if err != nil {
    log.Fatal(err)
}

result, err := manager.ExecuteAction(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Container exited with code: %d\n", result.ExitCode)
```
```

## Implementation Checklist

### Phase 1: Basic Container Execution
- [ ] Create container package structure
- [ ] Define container data structures
- [ ] Create container engine interface
- [ ] Implement Docker engine
- [ ] Implement Podman engine
- [ ] Create container manager
- [ ] Create basic tests
- [ ] Integrate with action system
- [ ] Create container action runner
- [ ] Update action execution
- [ ] Create integration tests

### Phase 2: Mount and Resource Support
- [ ] Implement mount validation
- [ ] Implement mount command generation
- [ ] Create mount tests
- [ ] Implement resource limit parsing
- [ ] Implement resource limit command generation
- [ ] Create resource tests

### Phase 3: Image Building and Advanced Features
- [ ] Implement image building
- [ ] Create image building tests
- [ ] Implement artifact collection
- [ ] Create artifact collection tests

### Phase 4: Testing and Documentation
- [ ] Create end-to-end tests
- [ ] Create matrix integration tests
- [ ] Create performance tests
- [ ] Create user documentation
- [ ] Create API documentation
- [ ] Create examples
- [ ] Create matrix container example

## Success Criteria

- [ ] All tests pass
- [ ] Documentation is complete
- [ ] Examples work correctly
- [ ] Integration with existing buildfab functionality
- [ ] No external dependencies beyond Docker/Podman
- [ ] Clear error messages and debugging information

---

**Document Status**: Implementation Steps
**Version**: 1.0
**Date**: 2025-01-27
**Next Review**: After implementation completion
