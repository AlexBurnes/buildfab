package container

import (
	"context"
	"fmt"
	"os/exec"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

// PodmanEngine implements the Engine interface for Podman
type PodmanEngine struct {
	binary string
}

// NewPodmanEngine creates a new Podman engine instance
func NewPodmanEngine() *PodmanEngine {
	return &PodmanEngine{
		binary: "podman",
	}
}

// DetectEngine checks if Podman is available
func (p *PodmanEngine) DetectEngine() bool {
	cmd := exec.Command(p.binary, "version")
	return cmd.Run() == nil
}

// GetEngineName returns the engine name
func (p *PodmanEngine) GetEngineName() string {
	return "podman"
}

// PullImage pulls a Podman image
func (p *PodmanEngine) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, p.binary, "pull", image)
	return cmd.Run()
}

// PullImageWithCallback pulls a Podman image with streaming output
func (p *PodmanEngine) PullImageWithCallback(ctx context.Context, image string, outputCallback func(string)) error {
	cmd := exec.CommandContext(ctx, p.binary, "pull", image)
	
	// Set up streaming output
	cmd.Stdout = &streamingWriter{callback: outputCallback}
	cmd.Stderr = &streamingWriter{callback: outputCallback}
	
	return cmd.Run()
}

// ImageExists checks if a Podman image exists locally
func (p *PodmanEngine) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, p.binary, "image", "exists", image)
	err := cmd.Run()
	return err == nil, nil
}

// BuildImage builds a Podman image from a Dockerfile
func (p *PodmanEngine) BuildImage(ctx context.Context, config container.ContainerBuild) (string, error) {
	// Implementation for building images
	// Return generated image tag
	return "", fmt.Errorf("not implemented")
}

// RunContainer runs a Podman container
func (p *PodmanEngine) RunContainer(ctx context.Context, config container.ContainerConfig) (*container.ContainerResult, error) {
	// Build podman run command
	args := []string{"run", "--rm"}
	
	// Add working directory if specified
	if config.Workdir != "" {
		args = append(args, "-w", config.Workdir)
	}
	
	// Add environment variables
	for key, value := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	
	// Add mounts
	for _, mount := range config.Mounts {
		if mount.Type == "bind" {
			mountArg := fmt.Sprintf("%s:%s", mount.Source, mount.Target)
			if mount.RO {
				mountArg += ":ro"
			}
			args = append(args, "-v", mountArg)
		}
	}
	
	// Add CPU and memory limits
	if config.CPU > 0 {
		// Simple CPU count: 2 -> --cpus 2.0 --cpuset-cpus "0,1"
		args = append(args, "--cpus", fmt.Sprintf("%d.0", config.CPU))
		
		// Generate CPU set: 2 -> "0,1", 3 -> "0,1,2", etc.
		cpuSet := ""
		for i := 0; i < config.CPU; i++ {
			if i > 0 {
				cpuSet += ","
			}
			cpuSet += fmt.Sprintf("%d", i)
		}
		args = append(args, "--cpuset-cpus", cpuSet)
	}
	
	if config.Memory != "" {
		args = append(args, "-m", config.Memory)
	}
	
	// Add user if specified
	if config.User != "" {
		args = append(args, "-u", config.User)
	}
	
	// Add network if specified
	if config.Network != "" {
		args = append(args, "--network", config.Network)
	}
	
	// Add image
	args = append(args, config.Image.From)
	
	// Add command to run
	if config.Run != "" {
		args = append(args, "sh", "-c", config.Run)
	} else if config.RunAction != "" {
		args = append(args, "buildfab", "action", config.RunAction)
	} else if config.RunStage != "" {
		args = append(args, "buildfab", "run", config.RunStage)
	}
	
	// Execute podman run command
	cmd := exec.CommandContext(ctx, p.binary, args...)
	output, err := cmd.CombinedOutput()
	
	result := &container.ContainerResult{
		ContainerID: "podman-container", // Placeholder
		ExitCode:    0,
		Output:      string(output),
		Error:       "",
		Artifacts:   []string{},
	}
	
	if err != nil {
		result.ExitCode = 1
		result.Error = err.Error()
	}
	
	return result, nil
}

// StopContainer stops a Podman container
func (p *PodmanEngine) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, p.binary, "stop", containerID)
	return cmd.Run()
}

// RemoveContainer removes a Podman container
func (p *PodmanEngine) RemoveContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, p.binary, "rm", containerID)
	return cmd.Run()
}
