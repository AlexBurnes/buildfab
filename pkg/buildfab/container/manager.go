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

// NewManager creates a new container manager, automatically detecting available engines
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
