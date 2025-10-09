package container

import (
    "context"
    "fmt"
    "math/rand"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

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
    // Handle this FIRST before artifact collection, since we need to transform run_action/run_stage into run
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

    // Handle artifact collection for ALL run commands (including transformed run_action/run_stage)
    // This must be done AFTER run_action/run_stage transformation
    if preparedConfig.Run != "" && len(preparedConfig.Artifacts.Path) > 0 {
        // Add artifact mount for all run commands
        if err := r.addArtifactMount(&preparedConfig); err != nil {
            return preparedConfig, err
        }
        
        // Add artifact copy commands to the run script
        r.addArtifactCopyCommands(&preparedConfig)
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

// collectArtifacts collects artifacts from the container using the hybrid approach:
// - For run commands: artifacts are collected via pre-mounted volume (already on host, no copy needed)
// - For build-only images: use docker/podman cp from temporary container
func (r *ContainerRunner) collectArtifacts(result *container.ContainerResult, config container.ContainerConfig) error {
    // Only collect artifacts if there are any specified
    if len(config.Artifacts.Path) == 0 {
        return nil
    }

    // Set default output directory
    if config.Artifacts.Output == "" {
        config.Artifacts.Output = "./artifacts"
    }
    
    // Create output directory
    if err := os.MkdirAll(config.Artifacts.Output, 0755); err != nil {
        return fmt.Errorf("failed to create output directory %s: %w", config.Artifacts.Output, err)
    }

    // Check if this is a build-only scenario (no run commands)
    // In this case, we need to extract artifacts from the built image
    if config.Run == "" && config.RunAction == "" && config.RunStage == "" && config.Image.Build != nil {
        // Build-only: create temporary container and copy artifacts
        // Use the image tag from config (manager sets this after build)
        imageTag := config.Image.From
        if imageTag == "" && len(config.Image.Build.Tags) > 0 {
            // Fallback to first build tag
            imageTag = config.Image.Build.Tags[0]
        }
        return r.collectArtifactsFromImage(imageTag, config)
    }
    
    // For run commands with artifacts:
    // Artifacts were already collected via pre-mounted volume during execution
    // The files are already on the host filesystem in the output directory
    // No need to copy - just verify they exist
    if config.Run != "" || config.RunAction != "" || config.RunStage != "" {
        // Artifacts were collected via pre-mounted volume - nothing more to do
        // The addArtifactMount and addArtifactCopyCommands already handled the collection
        
        // Force filesystem sync to ensure all writes from container bind mount are visible
        // This addresses race conditions where directories created through bind mounts
        // may not be immediately visible in directory listings (orphaned inode issue)
        if err := r.syncArtifactDirectory(config.Artifacts.Output); err != nil {
            // Sync is best-effort - log warning but don't fail
            if r.VerbosityLevel >= 2 {
                fmt.Printf("Warning: failed to sync artifact directory: %v\n", err)
            }
        }
        
        return nil
    }
    
    return nil
}

// syncArtifactDirectory forces filesystem sync on the artifact directory
// This ensures directory entries created through container bind mounts are visible
func (r *ContainerRunner) syncArtifactDirectory(artifactPath string) error {
    // Get absolute path
    absPath, err := filepath.Abs(artifactPath)
    if err != nil {
        return fmt.Errorf("failed to get absolute path: %w", err)
    }
    
    // Open and sync the artifact output directory itself
    // This refreshes the directory entry cache for subdirectories created by containers
    dir, err := os.Open(absPath)
    if err != nil {
        return fmt.Errorf("failed to open directory: %w", err)
    }
    defer dir.Close()
    
    // Sync to flush all metadata changes (directory entries)
    if err := dir.Sync(); err != nil {
        return fmt.Errorf("failed to sync directory: %w", err)
    }
    
    // Recursively sync subdirectories that might contain artifacts
    entries, err := os.ReadDir(absPath)
    if err != nil {
        return fmt.Errorf("failed to read directory for recursive sync: %w", err)
    }
    
    for _, entry := range entries {
        if entry.IsDir() {
            subPath := filepath.Join(absPath, entry.Name())
            if subDir, err := os.Open(subPath); err == nil {
                subDir.Sync() // Best effort, ignore errors
                subDir.Close()
            }
        }
    }
    
    return nil
}

// collectArtifactsFromImage extracts artifacts from a built image by creating a temporary container
func (r *ContainerRunner) collectArtifactsFromImage(imageTag string, config container.ContainerConfig) error {
    engineName := r.manager.GetEngineName()
    
    // Create a unique temporary container name using timestamp and random component
    containerName := fmt.Sprintf("buildfab-artifact-extract-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
    
    createCmd := exec.Command(engineName, "create", "--name", containerName, imageTag)
    if err := createCmd.Run(); err != nil {
        return fmt.Errorf("failed to create temporary container for artifact extraction: %w", err)
    }
    
    // Ensure cleanup
    defer func() {
        exec.Command(engineName, "rm", containerName).Run()
    }()
    
    // Copy artifacts from the container
    for _, artifactPath := range config.Artifacts.Path {
        if err := r.copyArtifactFromContainer(containerName, artifactPath, config.Artifacts.Output); err != nil {
            return fmt.Errorf("failed to extract artifact %s from image: %w", artifactPath, err)
        }
    }
    
    return nil
}

// copyArtifactFromContainer copies an artifact from the container to the host
// Preserves full path structure: /app/binary -> ./dist/app/binary
func (r *ContainerRunner) copyArtifactFromContainer(containerID, artifactPath, outputDir string) error {
    if containerID == "" {
        return fmt.Errorf("container ID is empty, cannot copy artifacts")
    }

    // Get the container engine name
    engineName := r.manager.GetEngineName()

    // For absolute paths like /app/binary or /usr/local/bin/myapp
    // We want to preserve the full path structure in the output directory
    // /app/binary -> ./dist/app/binary
    // /usr/local/bin/myapp -> ./dist/usr/local/bin/myapp
    
    // Clean the artifact path to remove any leading slashes for destination
    relPath := filepath.Clean(strings.TrimPrefix(artifactPath, "/"))
    destPath := filepath.Join(outputDir, relPath)
    
    // Create the destination directory structure
    destDir := filepath.Dir(destPath)
    if err := os.MkdirAll(destDir, 0755); err != nil {
        return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
    }

    // Use docker/podman cp to copy the artifact
    // Format: docker cp <container>:<src_path> <dest_path>
    cmd := exec.Command(engineName, "cp", 
        fmt.Sprintf("%s:%s", containerID, artifactPath),
        destPath)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to copy artifact from container: %w, output: %s", err, string(output))
    }

    return nil
}

// addArtifactMount adds a mount point for artifact collection
func (r *ContainerRunner) addArtifactMount(config *container.ContainerConfig) error {
    // Get absolute path for output directory
    absOutput, err := filepath.Abs(config.Artifacts.Output)
    if err != nil {
        return fmt.Errorf("failed to get absolute path for artifacts output: %w", err)
    }
    
    // Create output directory on host
    if err := os.MkdirAll(absOutput, 0755); err != nil {
        return fmt.Errorf("failed to create artifacts output directory: %w", err)
    }
    
    // Add mount for artifacts
    artifactMount := container.ContainerMount{
        Type:   "bind",
        Source: absOutput,
        Target: "/buildfab-artifacts",
        RO:     false,
    }
    config.Mounts = append(config.Mounts, artifactMount)
    
    return nil
}

// addArtifactCopyCommands adds commands to copy artifacts to the mounted volume
// Preserves full path structure: /app/binary -> /buildfab-artifacts/app/binary
func (r *ContainerRunner) addArtifactCopyCommands(config *container.ContainerConfig) {
    if config.Run == "" {
        return
    }
    
    // Build artifact copy commands
    var copyCommands []string
    for _, artifactPath := range config.Artifacts.Path {
        // Use cp --parents to automatically create parent directory structure
        // This works correctly with wildcards unlike mkdir -p with wildcards
        // Example: cp --parents -r .rpmbuild/RPMS/*/*.rpm /buildfab-artifacts/
        // Results in: /buildfab-artifacts/.rpmbuild/RPMS/x86_64/package.rpm
        
        // Use (command || true) to prevent failures from stopping execution
        copyCmd := fmt.Sprintf("(cp --parents -r %s /buildfab-artifacts/ || true)", 
            artifactPath)
        copyCommands = append(copyCommands, copyCmd)
    }
    
    // Append copy commands to run script
    if len(copyCommands) > 0 {
        // Add a separator and then the copy commands
        config.Run = config.Run + "\n" + strings.Join(copyCommands, "\n")
    }
}
