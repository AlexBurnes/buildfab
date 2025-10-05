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

// ExecuteActionWithCallback executes a container action with streaming callback
func (m *Manager) ExecuteActionWithCallback(ctx context.Context, config ContainerConfig, outputCallback func(string)) (*ContainerResult, error) {
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
	
	// Run container with callback
	return m.engine.RunContainerWithCallback(ctx, config, outputCallback)
}

// validateConfig validates the container configuration
func (m *Manager) validateConfig(config ContainerConfig) error {
	// Validate required fields
	if config.Image.From == "" && config.Image.Build == nil {
		return fmt.Errorf("image.from or image.build must be specified")
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
