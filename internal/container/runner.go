package container

import (
	"context"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

// ContainerRunner handles container action execution
type ContainerRunner struct {
	manager *container.Manager
}

// NewContainerRunner creates a new container runner
func NewContainerRunner() (*ContainerRunner, error) {
	manager, err := container.NewManager()
	if err != nil {
		return nil, err
	}
	return &ContainerRunner{manager: manager}, nil
}

// NewContainerRunnerWithEngine creates a new container runner with a specific engine
func NewContainerRunnerWithEngine(engineName string) (*ContainerRunner, error) {
	manager, err := container.NewManagerWithEngine(engineName)
	if err != nil {
		return nil, err
	}
	return &ContainerRunner{manager: manager}, nil
}

// RunAction executes a container action
func (r *ContainerRunner) RunAction(ctx context.Context, config container.ContainerConfig) (*container.ContainerResult, error) {
	// Copy buildfab binary and config to container
	if err := r.prepareContainer(ctx, config); err != nil {
		return nil, err
	}
	
	// Execute container
	result, err := r.manager.ExecuteAction(ctx, config)
	if err != nil {
		return nil, err
	}
	
	// Collect artifacts
	if err := r.collectArtifacts(result, config); err != nil {
		return nil, err
	}
	
	return result, nil
}

// GetManager returns the container manager
func (r *ContainerRunner) GetManager() *container.Manager {
	return r.manager
}

// prepareContainer prepares the container for execution
func (r *ContainerRunner) prepareContainer(ctx context.Context, config container.ContainerConfig) error {
	// Only copy buildfab binary and config if using run_stage or run_action
	if config.RunStage != "" || config.RunAction != "" {
		// Copy buildfab binary to container
		// Copy configuration files
	}
	// Set up environment variables
	return nil
}

// collectArtifacts collects artifacts from the container
func (r *ContainerRunner) collectArtifacts(result *container.ContainerResult, config container.ContainerConfig) error {
	// Collect artifacts from container to host
	for _, artifact := range config.Artifacts.Path {
		// Copy artifact from container to host
		_ = artifact // Placeholder to avoid unused variable error
	}
	return nil
}
