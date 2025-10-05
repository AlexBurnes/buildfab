package container

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// NewDockerEngine creates a new Docker engine
func NewDockerEngine() Engine {
	return &dockerEngineImpl{binary: "docker"}
}

// NewPodmanEngine creates a new Podman engine
func NewPodmanEngine() Engine {
	return &podmanEngineImpl{binary: "podman"}
}

// Internal engine implementations
type dockerEngineImpl struct {
	binary string
}

func (d *dockerEngineImpl) DetectEngine() bool {
	cmd := exec.Command(d.binary, "version")
	return cmd.Run() == nil
}

func (d *dockerEngineImpl) GetEngineName() string {
	return "docker"
}

func (d *dockerEngineImpl) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, d.binary, "pull", image)
	return cmd.Run()
}

func (d *dockerEngineImpl) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, d.binary, "image", "inspect", image)
	err := cmd.Run()
	return err == nil, nil
}

func (d *dockerEngineImpl) BuildImage(ctx context.Context, config ContainerBuild) (string, error) {
	// Implementation for building images
	// Return generated image tag
	return "", fmt.Errorf("not implemented")
}

func (d *dockerEngineImpl) RunContainer(ctx context.Context, config ContainerConfig) (*ContainerResult, error) {
	// Build docker run command
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
	if len(config.CPUs) > 0 {
		cpuSet := ""
		for i, cpu := range config.CPUs {
			if i > 0 {
				cpuSet += ","
			}
			cpuSet += fmt.Sprintf("%d", cpu)
		}
		args = append(args, "--cpuset-cpus", cpuSet)
		args = append(args, "--cpus", fmt.Sprintf("%d", len(config.CPUs)))
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
	
	// Execute docker run command
	cmd := exec.CommandContext(ctx, d.binary, args...)
	output, err := cmd.CombinedOutput()
	
	result := &ContainerResult{
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

// RunContainerWithCallback runs a Docker container with streaming output callback
func (d *dockerEngineImpl) RunContainerWithCallback(ctx context.Context, config ContainerConfig, outputCallback func(string)) (*ContainerResult, error) {
	// Build docker run command (same as RunContainer)
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
	if len(config.CPUs) > 0 {
		cpuSet := ""
		for i, cpu := range config.CPUs {
			if i > 0 {
				cpuSet += ","
			}
			cpuSet += fmt.Sprintf("%d", cpu)
		}
		args = append(args, "--cpuset-cpus", cpuSet)
		args = append(args, "--cpus", fmt.Sprintf("%d", len(config.CPUs)))
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
	
	// Execute docker run command with streaming
	cmd := exec.CommandContext(ctx, d.binary, args...)
	
	// Set up streaming pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	
	// Stream output in real-time
	var output strings.Builder
	var errorOutput strings.Builder
	
	// Channel to signal when streaming is complete
	done := make(chan struct{})
	
	// Stream stdout
	go func() {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			// Send to callback for real-time display
			if outputCallback != nil {
				outputCallback(line)
			}
		}
	}()
	
	// Stream stderr
	go func() {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			errorOutput.WriteString(line + "\n")
			// Send to callback for real-time display
			if outputCallback != nil {
				outputCallback(line)
			}
		}
	}()
	
	// Wait for command to complete
	err = cmd.Wait()
	
	// Wait for both goroutines to finish
	<-done
	<-done
	
	result := &ContainerResult{
		ContainerID: "docker-container", // Placeholder
		ExitCode:    0,
		Output:      output.String(),
		Error:       errorOutput.String(),
		Artifacts:   []string{},
	}
	
	if err != nil {
		result.ExitCode = 1
		if result.Error == "" {
			result.Error = err.Error()
		}
	}
	
	return result, nil
}

func (d *dockerEngineImpl) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, d.binary, "stop", containerID)
	return cmd.Run()
}

func (d *dockerEngineImpl) RemoveContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, d.binary, "rm", containerID)
	return cmd.Run()
}

type podmanEngineImpl struct {
	binary string
}

func (p *podmanEngineImpl) DetectEngine() bool {
	cmd := exec.Command(p.binary, "version")
	return cmd.Run() == nil
}

func (p *podmanEngineImpl) GetEngineName() string {
	return "podman"
}

func (p *podmanEngineImpl) PullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, p.binary, "pull", image)
	return cmd.Run()
}

func (p *podmanEngineImpl) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, p.binary, "image", "exists", image)
	err := cmd.Run()
	return err == nil, nil
}

func (p *podmanEngineImpl) BuildImage(ctx context.Context, config ContainerBuild) (string, error) {
	// Implementation for building images
	// Return generated image tag
	return "", fmt.Errorf("not implemented")
}

func (p *podmanEngineImpl) RunContainer(ctx context.Context, config ContainerConfig) (*ContainerResult, error) {
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
	if len(config.CPUs) > 0 {
		cpuSet := ""
		for i, cpu := range config.CPUs {
			if i > 0 {
				cpuSet += ","
			}
			cpuSet += fmt.Sprintf("%d", cpu)
		}
		args = append(args, "--cpuset-cpus", cpuSet)
		args = append(args, "--cpus", fmt.Sprintf("%d", len(config.CPUs)))
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
	
	result := &ContainerResult{
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

// RunContainerWithCallback runs a Podman container with streaming output callback
func (p *podmanEngineImpl) RunContainerWithCallback(ctx context.Context, config ContainerConfig, outputCallback func(string)) (*ContainerResult, error) {
	// Build podman run command (same as RunContainer)
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
	if len(config.CPUs) > 0 {
		cpuSet := ""
		for i, cpu := range config.CPUs {
			if i > 0 {
				cpuSet += ","
			}
			cpuSet += fmt.Sprintf("%d", cpu)
		}
		args = append(args, "--cpuset-cpus", cpuSet)
		args = append(args, "--cpus", fmt.Sprintf("%d", len(config.CPUs)))
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
	
	// Execute podman run command with streaming
	cmd := exec.CommandContext(ctx, p.binary, args...)
	
	// Set up streaming pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	
	// Stream output in real-time
	var output strings.Builder
	var errorOutput strings.Builder
	
	// Channel to signal when streaming is complete
	done := make(chan struct{})
	
	// Stream stdout
	go func() {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			// Send to callback for real-time display
			if outputCallback != nil {
				outputCallback(line)
			}
		}
	}()
	
	// Stream stderr
	go func() {
		defer func() { done <- struct{}{} }()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			errorOutput.WriteString(line + "\n")
			// Send to callback for real-time display
			if outputCallback != nil {
				outputCallback(line)
			}
		}
	}()
	
	// Wait for command to complete
	err = cmd.Wait()
	
	// Wait for both goroutines to finish
	<-done
	<-done
	
	result := &ContainerResult{
		ContainerID: "podman-container", // Placeholder
		ExitCode:    0,
		Output:      output.String(),
		Error:       errorOutput.String(),
		Artifacts:   []string{},
	}
	
	if err != nil {
		result.ExitCode = 1
		if result.Error == "" {
			result.Error = err.Error()
		}
	}
	
	return result, nil
}

func (p *podmanEngineImpl) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, p.binary, "stop", containerID)
	return cmd.Run()
}

func (p *podmanEngineImpl) RemoveContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, p.binary, "rm", containerID)
	return cmd.Run()
}
