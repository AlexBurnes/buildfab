package container

import (
	"context"
	"fmt"
	"os/exec"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

// DockerEngine implements the Engine interface for Docker
type DockerEngine struct {
	binary string
}

// NewDockerEngine creates a new Docker engine instance
func NewDockerEngine() *DockerEngine {
	return &DockerEngine{
		binary: "docker",
	}
}

// DetectEngine checks if Docker is available
func (d *DockerEngine) DetectEngine() bool {
	cmd := exec.Command(d.binary, "version")
	return cmd.Run() == nil
}

// GetEngineName returns the engine name
func (d *DockerEngine) GetEngineName() string {
	return "docker"
}

// PullImage pulls a Docker image
func (d *DockerEngine) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, d.binary, "pull", image)
	return cmd.Run()
}

// PullImageWithCallback pulls a Docker image with streaming output
func (d *DockerEngine) PullImageWithCallback(ctx context.Context, image string, outputCallback func(string)) error {
	cmd := exec.CommandContext(ctx, d.binary, "pull", image)
	
	// Set up streaming output
	cmd.Stdout = &streamingWriter{callback: outputCallback}
	cmd.Stderr = &streamingWriter{callback: outputCallback}
	
	return cmd.Run()
}

// ImageExists checks if a Docker image exists locally
func (d *DockerEngine) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, d.binary, "image", "inspect", image)
	err := cmd.Run()
	return err == nil, nil
}

// BuildImage builds a Docker image from a Dockerfile
func (d *DockerEngine) BuildImage(ctx context.Context, config container.ContainerBuild) (string, error) {
	// Implementation for building images
	// Return generated image tag
	return "", fmt.Errorf("not implemented")
}

// RunContainer runs a Docker container
func (d *DockerEngine) RunContainer(ctx context.Context, config container.ContainerConfig) (*container.ContainerResult, error) {
	// Build docker run command
	args := []string{"run", "--rm"}
	
	// Check if there's a bind mount with target /tmp/buildfab-workspace and add -w accordingly
	hasWorkspaceMount := false
	for _, mount := range config.Mounts {
		if mount.Type == "bind" && mount.Target == "/tmp/buildfab-workspace" {
			hasWorkspaceMount = true
			break
		}
	}
	
	// Add working directory only if workspace mount exists
	if hasWorkspaceMount {
		args = append(args, "-w", "/tmp/buildfab-workspace")
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
		runCommand := config.Run
		if config.Workdir != "" {
			runCommand = fmt.Sprintf("cd %s && %s", config.Workdir, config.Run)
		}
		args = append(args, "sh", "-c", runCommand)
	} else if config.RunAction != "" {
		args = append(args, "sh", "-c", config.RunAction)
	} else if config.RunStage != "" {
		args = append(args, "sh", "-c", config.RunStage)
	}
	
	// Execute docker run command
	cmd := exec.CommandContext(ctx, d.binary, args...)
	output, err := cmd.CombinedOutput()
	
	result := &container.ContainerResult{
		ContainerID: "docker-container", // Placeholder
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

// StopContainer stops a Docker container
func (d *DockerEngine) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, d.binary, "stop", containerID)
	return cmd.Run()
}

// RemoveContainer removes a Docker container
func (d *DockerEngine) RemoveContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, d.binary, "rm", containerID)
	return cmd.Run()
}

// streamingWriter is a helper that streams output to a callback function
type streamingWriter struct {
	callback func(string)
}

func (w *streamingWriter) Write(p []byte) (n int, err error) {
	if w.callback != nil {
		w.callback(string(p))
	}
	return len(p), nil
}
