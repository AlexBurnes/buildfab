package container

import (
	"context"
	"io"
)

// Engine represents a container engine interface
type Engine interface {
	// Image operations
	PullImage(ctx context.Context, image string) error
	PullImageWithCallback(ctx context.Context, image string, outputCallback func(string)) error
	ImageExists(ctx context.Context, image string) (bool, error)
	BuildImage(ctx context.Context, config ContainerBuild) (string, error)
	BuildImageWithCallback(ctx context.Context, config ContainerBuild, outputCallback func(string)) (string, error)
	SlimImage(ctx context.Context, config ContainerSlim) (string, error)
	SlimImageWithCallback(ctx context.Context, config ContainerSlim, outputCallback func(string)) (string, error)
	
	// Container operations
	RunContainer(ctx context.Context, config ContainerConfig) (*ContainerResult, error)
	RunContainerWithCallback(ctx context.Context, config ContainerConfig, outputCallback func(string), stdin io.Reader) (*ContainerResult, error)
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	
	// Utility operations
	DetectEngine() bool
	GetEngineName() string
}
