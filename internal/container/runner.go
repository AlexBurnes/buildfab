package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

// ContainerRunner handles container action execution
type ContainerRunner struct {
	manager        *container.Manager
	VerbosityLevel int
}

// NewContainerRunner creates a new container runner
func NewContainerRunner() (*ContainerRunner, error) {
	manager, err := container.NewManager()
	if err != nil {
		return nil, err
	}
	return &ContainerRunner{manager: manager, VerbosityLevel: 1}, nil
}

// NewContainerRunnerWithVerbosity creates a new container runner with specified verbosity
func NewContainerRunnerWithVerbosity(verbosityLevel int) (*ContainerRunner, error) {
	manager, err := container.NewManager()
	if err != nil {
		return nil, err
	}
	return &ContainerRunner{manager: manager, VerbosityLevel: verbosityLevel}, nil
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
func (r *ContainerRunner) RunAction(ctx context.Context, config container.ContainerConfig, configFile string) (*container.ContainerResult, error) {
	// Prepare container configuration for run_action or run_stage
	preparedConfig := r.PrepareContainerConfig(config, configFile)
	
	// Execute container with callback (for streaming output)
	result, err := r.manager.ExecuteActionWithCallback(ctx, preparedConfig, nil)
	if err != nil {
		return nil, err
	}
	
	// Collect artifacts
	if err := r.collectArtifacts(result, preparedConfig); err != nil {
		return nil, err
	}
	
	return result, nil
}

// RunActionWithCallback executes a container action with streaming output callback
func (r *ContainerRunner) RunActionWithCallback(ctx context.Context, config container.ContainerConfig, configFile string, outputCallback func(string)) (*container.ContainerResult, error) {
	// Prepare container configuration for run_action or run_stage
	preparedConfig := r.PrepareContainerConfig(config, configFile)
	
	// Execute container with callback
	result, err := r.manager.ExecuteActionWithCallback(ctx, preparedConfig, outputCallback)
	if err != nil {
		return nil, err
	}
	
	// Collect artifacts
	if err := r.collectArtifacts(result, preparedConfig); err != nil {
		return nil, err
	}
	
	return result, nil
}

// GetManager returns the container manager
func (r *ContainerRunner) GetManager() *container.Manager {
	return r.manager
}

// PrepareContainerConfig prepares the container configuration for run_action or run_stage
func (r *ContainerRunner) PrepareContainerConfig(config container.ContainerConfig, configFile string) container.ContainerConfig {
	// Create a copy of the config
	preparedConfig := config
	
	// If this is a run_action or run_stage, we need to mount the current directory and buildfab binary
	if config.RunStage != "" || config.RunAction != "" {
		// Get current working directory and buildfab binary path
		wd, err := r.getCurrentWorkingDir()
		if err != nil {
			// If we can't get the working directory, return the original config
			return config
		}
		
		binPath, err := r.getBuildfabBinaryPath()
		if err != nil {
			// If we can't find the buildfab binary, return the original config
			// This means the container won't have the buildfab binary mounted
			return config
		}
		
		// Create temporary directory names for mounting
		tempDirName := "buildfab-workspace"
		tempBinDirName := "buildfab-bin"
		
		// Mount current directory as /tmp/buildfab-workspace (read-only for config)
		workspaceMount := container.ContainerMount{
			Type:   "bind",
			Source: wd,
			Target: fmt.Sprintf("/tmp/%s", tempDirName),
			RO:     true,
		}
		
		// Mount buildfab binary directory as /tmp/buildfab-bin (read-only for binary)
		binMount := container.ContainerMount{
			Type:   "bind",
			Source: filepath.Dir(binPath),
			Target: fmt.Sprintf("/tmp/%s", tempBinDirName),
			RO:     true,
		}
		
		// Add the mounts to the configuration
		preparedConfig.Mounts = append(preparedConfig.Mounts, workspaceMount, binMount)
		
		// Handle environment file if specified
		if config.EnvFile != "" {
			// Environment file is already available in the mounted workspace
			// No need to mount it separately since ./ is mounted as /tmp/buildfab-workspace
			// The env_file will be loaded from /tmp/buildfab-workspace/{env_file}
		}
		
		// Handle cache mounts if specified
		for cacheName, cachePath := range config.Cache {
			// Mount cache to standard buildfab cache location
			targetPath := fmt.Sprintf("/tmp/buildfab-cache-%s", cacheName)
			cacheMount := container.ContainerMount{
				Type:   "bind",
				Source: filepath.Join(wd, cachePath),
				Target: targetPath,
				RO:     false,
			}
			preparedConfig.Mounts = append(preparedConfig.Mounts, cacheMount)
		}
		
		// Build the execution command using mounted buildfab binary
		// Since we cd into the workspace directory, we can use the relative config file path
		configFilePath := configFile
		
		// Build verbosity flags for the buildfab command inside the container
		// Use minimal verbosity to avoid duplicate output (shell commands + output)
		verbosityFlags := ""
		if r.VerbosityLevel >= 1 {
			verbosityFlags = "-v" // Use -v for basic verbosity
		}
		if r.VerbosityLevel >= 2 {
			verbosityFlags = "-v" // Keep -v to avoid duplicate output
		}
		if r.VerbosityLevel >= 3 {
			verbosityFlags = "-v" // Keep -v to avoid duplicate output
		}
		
		var command string
		if config.RunAction != "" {
			command = fmt.Sprintf(`cd /tmp/%s && exec /tmp/%s/buildfab -c %s action %s %s`, tempDirName, tempBinDirName, configFilePath, config.RunAction, verbosityFlags)
		} else if config.RunStage != "" {
			command = fmt.Sprintf(`cd /tmp/%s && exec /tmp/%s/buildfab -c %s run %s %s`, tempDirName, tempBinDirName, configFilePath, config.RunStage, verbosityFlags)
		}
		
		// Handle environment file if specified
		if config.EnvFile != "" {
			// Load environment file from mounted workspace directory
			envFileCommand := fmt.Sprintf("source /tmp/%s/%s && ", tempDirName, config.EnvFile)
			command = envFileCommand + command
		}
		
		// Add buildfab alias for easier usage
		aliasCommand := fmt.Sprintf("alias buildfab='/tmp/%s/buildfab' && ", tempBinDirName)
		command = aliasCommand + command
		
		// Set the run command
		preparedConfig.Run = command
		
		// Clear the original fields
		preparedConfig.RunAction = ""
		preparedConfig.RunStage = ""
	}
	
	return preparedConfig
}

// getCurrentWorkingDir returns the current working directory
func (r *ContainerRunner) getCurrentWorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return wd, nil
}

// getBuildfabBinaryPath returns the path to the buildfab binary
func (r *ContainerRunner) getBuildfabBinaryPath() (string, error) {
	wd, err := r.getCurrentWorkingDir()
	if err != nil {
		return "", err
	}
	
	// Check for buildfab binary in bin/ directory
	binPath := filepath.Join(wd, "bin", "buildfab")
	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}
	
	// Check for buildfab binary in current directory
	currentPath := filepath.Join(wd, "buildfab")
	if _, err := os.Stat(currentPath); err == nil {
		return currentPath, nil
	}
	
	return "", fmt.Errorf("buildfab binary not found in bin/ (%s) or current directory (%s), working dir: %s", binPath, currentPath, wd)
}

// collectArtifacts collects artifacts from the container
func (r *ContainerRunner) collectArtifacts(result *container.ContainerResult, config container.ContainerConfig) error {
	// Only collect artifacts if there are any specified
	if len(config.Artifacts.Path) == 0 {
		return nil
	}
	
	// Create output directory if it doesn't exist
	if config.Artifacts.Output != "" {
		if err := os.MkdirAll(config.Artifacts.Output, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", config.Artifacts.Output, err)
		}
	}
	
	// Collect artifacts from container to host
	for _, artifact := range config.Artifacts.Path {
		if err := r.copyArtifactFromContainer(result.ContainerID, artifact, config.Artifacts.Output); err != nil {
			return fmt.Errorf("failed to copy artifact %s: %w", artifact, err)
		}
	}
	
	return nil
}

// copyArtifactFromContainer copies an artifact from the container to the host
func (r *ContainerRunner) copyArtifactFromContainer(containerID, artifactPath, outputDir string) error {
	if containerID == "" {
		return fmt.Errorf("container ID is empty, cannot copy artifacts")
	}
	
	// Get the container engine name
	engineName := r.manager.GetEngineName()
	
	// Build the copy command
	var cmd *exec.Cmd
	if outputDir != "" {
		cmd = exec.Command(engineName, "cp", fmt.Sprintf("%s:%s", containerID, artifactPath), outputDir)
	} else {
		cmd = exec.Command(engineName, "cp", fmt.Sprintf("%s:%s", containerID, artifactPath), ".")
	}
	
	// Execute the copy command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s cp command: %w", engineName, err)
	}
	
	return nil
}
