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
    // Debug: Print container configuration
    fmt.Printf("[DEBUG] Container config: CPU=%d, Memory=%s, Network=%s, Workdir=%s\n",
        config.CPU, config.Memory, config.Network, config.Workdir)

    // Prepare container configuration for run_action or run_stage
    preparedConfig, err := r.PrepareContainerConfig(config, configFile)
    if err != nil {
        return nil, err
    }

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
    preparedConfig, err := r.PrepareContainerConfig(config, configFile)
    if err != nil {
        return nil, err
    }

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
func (r *ContainerRunner) PrepareContainerConfig(config container.ContainerConfig, configFile string) (container.ContainerConfig, error) {
    // Create a copy of the config
    preparedConfig := config

    // If this is a run_action or run_stage, we need to mount the current directory and buildfab binary
    if config.RunStage != "" || config.RunAction != "" {
        // Get current working directory and buildfab binary path
        wd, err := r.getCurrentWorkingDir()
        if err != nil {
            return config, fmt.Errorf("failed to get current working directory for container mount: %w", err)
        }

        binPath, err := r.getBuildfabBinaryPath()
        if err != nil {
            return config, fmt.Errorf("failed to find buildfab binary for container mount: %w", err)
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

        // Cache mounts are handled by the engine in buildRunArgs method
        // No need to add them here as it would cause duplicate mount destinations

        // Build the execution command using mounted buildfab binary
        // Use absolute path to config file since we're not changing directory
        configFilePath := filepath.Join("/tmp", tempDirName, configFile)

        // Build verbosity flags for the buildfab command inside the container
        // Use minimal verbosity to avoid duplicate output (shell commands + output)
        verbosityFlags := ""
        if r.VerbosityLevel >= 1 {
            verbosityFlags = "-v" // Use -v for basic verbosity
        }
        if r.VerbosityLevel >= 2 {
            verbosityFlags = "-vv" // Keep -v to avoid duplicate output
        }
        if r.VerbosityLevel >= 3 {
            verbosityFlags = "-vvv" // Keep -v to avoid duplicate output
        }

        var command string
        if config.RunAction != "" {
            command = fmt.Sprintf(`buildfab -c %s action %s %s`, configFilePath, config.RunAction, verbosityFlags)
        } else if config.RunStage != "" {
            command = fmt.Sprintf(`buildfab -c %s run %s %s`, configFilePath, config.RunStage, verbosityFlags)
        }

        // Build the complete command in the correct order:
        // 1. alias buildfab
        // 2. source env file (creates directories)
        // 3. cd to workdir (after directories are created)
        // 4. exec buildfab command

        // Start with alias
        finalCommand := fmt.Sprintf("export PATH=${PATH}:/tmp/%s && ", tempBinDirName)

        // Add environment file loading (creates /buildfab directory)
        // Use relative path since we're in the workspace directory
        if config.EnvFile != "" {
            envFileCommand := fmt.Sprintf(". ./%s && ", config.EnvFile)
            finalCommand += envFileCommand
        }

        // Add working directory change (AFTER environment is loaded and directories created)
        // Use the original workdir from config, not the workspace mount
        if config.Workdir != "" {
            workdirCommand := fmt.Sprintf("cd %s && ", config.Workdir)
            finalCommand += workdirCommand
        }

        // Add the final exec command
        finalCommand += command
        finalCommand += ""

        // Set the complete command
        command = finalCommand

        // Set the run command
        preparedConfig.Run = command

        // Clear the original fields
        preparedConfig.RunAction = ""
        preparedConfig.RunStage = ""

        // Clear workdir since we've already handled it in the command
        preparedConfig.Workdir = ""
    }

    return preparedConfig, nil
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
    // First, try to get the current executable's path
    if currentBinaryPath, err := r.getCurrentExecutablePath(); err == nil {
        // If we found the current executable, use it
        return currentBinaryPath, nil
    }

    // If we can't get the current executable path, fall back to searching common locations
    wd, err := r.getCurrentWorkingDir()
    if err != nil {
        return "", err
    }

    // Search paths in order of preference
    searchPaths := []string{
        filepath.Join(wd, "bin", "buildfab"),     // Development: ./bin/buildfab
        filepath.Join(wd, "buildfab"),            // Development: ./buildfab
        "/usr/local/bin/buildfab",                // System installation
        "/usr/bin/buildfab",                      // System installation
    }

    for _, path := range searchPaths {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }

    return "", fmt.Errorf("buildfab binary not found in any of the following locations: %v, working dir: %s", searchPaths, wd)
}

// getCurrentExecutablePath returns the path to the current buildfab executable
func (r *ContainerRunner) getCurrentExecutablePath() (string, error) {
    // Get the path of the current executable
    execPath, err := os.Executable()
    if err != nil {
        return "", fmt.Errorf("failed to get current executable path: %w", err)
    }

    // Resolve any symlinks to get the actual binary path
    resolvedPath, err := filepath.EvalSymlinks(execPath)
    if err != nil {
        // If symlink resolution fails, use the original path
        resolvedPath = execPath
    }

    // Check if the resolved path exists and is executable
    if _, err := os.Stat(resolvedPath); err != nil {
        return "", fmt.Errorf("current executable path does not exist: %s", resolvedPath)
    }

    return resolvedPath, nil
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
