package container

import (
	"context"
	"fmt"
	"os"
)

// Manager manages container operations
type Manager struct {
	engine Engine
}

// NewManager creates a new container manager with Podman as default
func NewManager() (*Manager, error) {
	// Default to Podman, fallback to Docker if Podman is not available
	if podman := NewPodmanEngine(); podman.DetectEngine() {
		return &Manager{engine: podman}, nil
	}
	if docker := NewDockerEngine(); docker.DetectEngine() {
		return &Manager{engine: docker}, nil
	}
	return nil, fmt.Errorf("no container engine available (tried podman and docker)")
}

// NewManagerWithEngine creates a new container manager with a specific engine
func NewManagerWithEngine(engineName string) (*Manager, error) {
	switch engineName {
	case "docker":
		if docker := NewDockerEngine(); docker.DetectEngine() {
			return &Manager{engine: docker}, nil
		}
		return nil, fmt.Errorf("docker engine not available")
	case "podman":
		if podman := NewPodmanEngine(); podman.DetectEngine() {
			return &Manager{engine: podman}, nil
		}
		return nil, fmt.Errorf("podman engine not available")
	default:
		return nil, fmt.Errorf("unsupported engine: %s (supported: docker, podman)", engineName)
	}
}

// ExecuteAction executes a container action
func (m *Manager) ExecuteAction(ctx context.Context, config ContainerConfig) (*ContainerResult, error) {
	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return nil, err
	}
	
	// Handle slim image operation
	if config.Image.Slim != nil {
		image, err := m.engine.SlimImage(ctx, *config.Image.Slim)
		if err != nil {
			return nil, err
		}
		// For slim operations, we don't run a container, just return success
		return &ContainerResult{
			ContainerID: "slim-operation",
			ExitCode:    0,
			Output:      fmt.Sprintf("Slim image created: %s", image),
			Error:       "",
			Artifacts:   []string{},
		}, nil
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
	
	// Only run container if run commands are specified
	if config.Run == "" && config.RunAction == "" && config.RunStage == "" {
		// No run commands specified, just return success after build/pull
		return &ContainerResult{
			ContainerID: "build-only",
			ExitCode:    0,
			Output:      fmt.Sprintf("Image built/pulled successfully: %s", config.Image.From),
			Error:       "",
			Artifacts:   []string{},
		}, nil
	}
	
	// Run container
	return m.engine.RunContainer(ctx, config)
}

// ExecuteActionWithCallback executes a container action with streaming callback
func (m *Manager) ExecuteActionWithCallback(ctx context.Context, config ContainerConfig, outputCallback func(string)) (*ContainerResult, error) {
	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return nil, err
	}
	
	// Handle slim image operation
	if config.Image.Slim != nil {
		image, err := m.engine.SlimImageWithCallback(ctx, *config.Image.Slim, outputCallback)
		if err != nil {
			return nil, err
		}
		// For slim operations, we don't run a container, just return success
		result := &ContainerResult{
			ContainerID: "slim-operation",
			ExitCode:    0,
			Output:      fmt.Sprintf("Slim image created: %s", image),
			Error:       "",
			Artifacts:   []string{},
		}
		return result, nil
	}
	
	// Pull or build image
	if config.Image.Build != nil {
		image, err := m.engine.BuildImageWithCallback(ctx, *config.Image.Build, outputCallback)
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
	
	// Only run container if run commands are specified
	if config.Run == "" && config.RunAction == "" && config.RunStage == "" {
		// No run commands specified, just return success after build/pull
		result := &ContainerResult{
			ContainerID: "build-only",
			ExitCode:    0,
			Output:      fmt.Sprintf("Image built/pulled successfully: %s", config.Image.From),
			Error:       "",
			Artifacts:   []string{},
		}
		return result, nil
	}
	
	// Run container with callback
	return m.engine.RunContainerWithCallback(ctx, config, outputCallback)
}

// GetEngineName returns the name of the current engine
func (m *Manager) GetEngineName() string {
	return m.engine.GetEngineName()
}

// validateConfig validates the container configuration
func (m *Manager) validateConfig(config ContainerConfig) error {
	// Validate required fields
	if config.Image.From == "" && config.Image.Build == nil && config.Image.Slim == nil {
		return fmt.Errorf("image.from, image.build, or image.slim must be specified")
	}
	
	// Validate slim configuration
	if config.Image.Slim != nil {
		if config.Image.Slim.Target == "" {
			return fmt.Errorf("image.slim.target must be specified for slim operations")
		}
	}
	
	// Validate mounts
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
