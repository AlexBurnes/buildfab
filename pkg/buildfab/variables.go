package buildfab

import (
	"fmt"
	"regexp"
	"strings"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

// InterpolateVariables replaces ${{ variable }} syntax with actual values
func InterpolateVariables(text string, variables map[string]string) (string, error) {
	if variables == nil {
		return text, nil
	}
	
	// Pattern to match ${{ variable }} syntax
	pattern := regexp.MustCompile(`\$\{\{\s*([^}]+)\s*\}\}`)
	
	// First pass: collect all variable references to validate they exist
	var missingVars []string
	allMatches := pattern.FindAllStringSubmatch(text, -1)
	
	for _, match := range allMatches {
		if len(match) >= 2 {
			varName := strings.TrimSpace(match[1])
			if _, exists := variables[varName]; !exists {
				missingVars = append(missingVars, varName)
			}
		}
	}
	
	// If any variables are missing, return a comprehensive error
	if len(missingVars) > 0 {
		var availableVars []string
		for varName := range variables {
			availableVars = append(availableVars, varName)
		}
		
		errorMsg := fmt.Sprintf("undefined variables: %s", strings.Join(missingVars, ", "))
		if len(availableVars) > 0 {
			errorMsg += fmt.Sprintf("\navailable variables: %s", strings.Join(availableVars, ", "))
		}
		return "", fmt.Errorf(errorMsg)
	}
	
	// Second pass: perform actual interpolation
	result := pattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name from ${{ variable }}
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Return original if no match
		}
		
		varName := strings.TrimSpace(submatches[1])
		
		// Variable should exist at this point since we validated above
		if value, exists := variables[varName]; exists {
			return value
		}
		
		// This should not happen due to validation above, but just in case
		return match
	})
	
	return result, nil
}

// InterpolateAction interpolates variables in an action's run command
func InterpolateAction(action Action, variables map[string]string) (Action, error) {
	if action.Run == "" {
		return action, nil
	}
	
	interpolated, err := InterpolateVariables(action.Run, variables)
	if err != nil {
		return action, fmt.Errorf("failed to interpolate variables in action %s: %w", action.Name, err)
	}
	
	action.Run = interpolated
	return action, nil
}

// InterpolateStep interpolates variables in a step's configuration
func InterpolateStep(step Step, variables map[string]string) (Step, error) {
	// Interpolate the action name if it's a variable
	interpolatedAction, err := InterpolateVariables(step.Action, variables)
	if err != nil {
		return step, fmt.Errorf("failed to interpolate variables in step action: %w", err)
	}
	step.Action = interpolatedAction
	
	// Interpolate require dependencies
	for i, req := range step.Require {
		interpolated, err := InterpolateVariables(req, variables)
		if err != nil {
			return step, fmt.Errorf("failed to interpolate variables in step require %d: %w", err)
		}
		step.Require[i] = interpolated
	}
	
	// Interpolate if condition
	if step.If != "" {
		interpolated, err := InterpolateVariables(step.If, variables)
		if err != nil {
			return step, fmt.Errorf("failed to interpolate variables in step if condition: %w", err)
		}
		step.If = interpolated
	}
	
	return step, nil
}

// InterpolateContainerConfig interpolates variables in a container configuration
func InterpolateContainerConfig(config *container.ContainerConfig, variables map[string]string) (*container.ContainerConfig, error) {
	if config == nil {
		return config, nil
	}
	
	// Create a copy to avoid modifying the original
	interpolated := *config
	
	// Interpolate engine
	if interpolated.Engine != "" {
		engine, err := InterpolateVariables(interpolated.Engine, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container engine: %w", err)
		}
		interpolated.Engine = engine
	}
	
	// Interpolate workdir
	if interpolated.Workdir != "" {
		workdir, err := InterpolateVariables(interpolated.Workdir, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container workdir: %w", err)
		}
		interpolated.Workdir = workdir
	}
	
	// Interpolate memory
	if interpolated.Memory != "" {
		memory, err := InterpolateVariables(interpolated.Memory, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container memory: %w", err)
		}
		interpolated.Memory = memory
	}
	
	// Interpolate user
	if interpolated.User != "" {
		user, err := InterpolateVariables(interpolated.User, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container user: %w", err)
		}
		interpolated.User = user
	}
	
	// Interpolate network
	if interpolated.Network != "" {
		network, err := InterpolateVariables(interpolated.Network, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container network: %w", err)
		}
		interpolated.Network = network
	}
	
	// Interpolate env_file
	if interpolated.EnvFile != "" {
		envFile, err := InterpolateVariables(interpolated.EnvFile, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container env_file: %w", err)
		}
		interpolated.EnvFile = envFile
	}
	
	// Interpolate run_stage
	if interpolated.RunStage != "" {
		runStage, err := InterpolateVariables(interpolated.RunStage, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container run_stage: %w", err)
		}
		interpolated.RunStage = runStage
	}
	
	// Interpolate run_action
	if interpolated.RunAction != "" {
		runAction, err := InterpolateVariables(interpolated.RunAction, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container run_action: %w", err)
		}
		interpolated.RunAction = runAction
	}
	
	// Interpolate run
	if interpolated.Run != "" {
		run, err := InterpolateVariables(interpolated.Run, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate container run: %w", err)
		}
		interpolated.Run = run
	}
	
	// Interpolate environment variables
	if interpolated.Env != nil {
		for key, value := range interpolated.Env {
			interpolatedValue, err := InterpolateVariables(value, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate container env %s: %w", key, err)
			}
			interpolated.Env[key] = interpolatedValue
		}
	}
	
	// Interpolate cache
	if interpolated.Cache != nil {
		for key, value := range interpolated.Cache {
			interpolatedValue, err := InterpolateVariables(value, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate container cache %s: %w", key, err)
			}
			interpolated.Cache[key] = interpolatedValue
		}
	}
	
	// Interpolate mounts
	for i, mount := range interpolated.Mounts {
		if mount.Type != "" {
			mountType, err := InterpolateVariables(mount.Type, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate container mount %d type: %w", i, err)
			}
			interpolated.Mounts[i].Type = mountType
		}
		
		if mount.Source != "" {
			mountSource, err := InterpolateVariables(mount.Source, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate container mount %d source: %w", i, err)
			}
			interpolated.Mounts[i].Source = mountSource
		}
		
		if mount.Target != "" {
			mountTarget, err := InterpolateVariables(mount.Target, variables)
			if err != nil {
				return nil, fmt.Errorf("failed to interpolate container mount %d target: %w", i, err)
			}
			interpolated.Mounts[i].Target = mountTarget
		}
	}
	
	// Interpolate image configuration
	if err := interpolateContainerImage(&interpolated.Image, variables); err != nil {
		return nil, fmt.Errorf("failed to interpolate container image: %w", err)
	}
	
	// Interpolate artifacts
	if err := interpolateContainerArtifacts(&interpolated.Artifacts, variables); err != nil {
		return nil, fmt.Errorf("failed to interpolate container artifacts: %w", err)
	}
	
	return &interpolated, nil
}

// interpolateContainerImage interpolates variables in container image configuration
func interpolateContainerImage(image *container.ContainerImage, variables map[string]string) error {
	if image == nil {
		return nil
	}
	
	// Interpolate from
	if image.From != "" {
		from, err := InterpolateVariables(image.From, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate image from: %w", err)
		}
		image.From = from
	}
	
	// Interpolate build configuration
	if image.Build != nil {
		if err := interpolateContainerBuild(image.Build, variables); err != nil {
			return fmt.Errorf("failed to interpolate image build: %w", err)
		}
	}
	
	// Interpolate slim configuration
	if image.Slim != nil {
		if err := interpolateContainerSlim(image.Slim, variables); err != nil {
			return fmt.Errorf("failed to interpolate image slim: %w", err)
		}
	}
	
	return nil
}

// interpolateContainerBuild interpolates variables in container build configuration
func interpolateContainerBuild(build *container.ContainerBuild, variables map[string]string) error {
	if build == nil {
		return nil
	}
	
	// Interpolate dockerfile
	if build.Dockerfile != "" {
		dockerfile, err := InterpolateVariables(build.Dockerfile, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate build dockerfile: %w", err)
		}
		build.Dockerfile = dockerfile
	}
	
	// Interpolate context
	if build.Context != "" {
		context, err := InterpolateVariables(build.Context, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate build context: %w", err)
		}
		build.Context = context
	}
	
	// Interpolate network
	if build.Network != "" {
		network, err := InterpolateVariables(build.Network, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate build network: %w", err)
		}
		build.Network = network
	}
	
	// Interpolate progress
	if build.Progress != "" {
		progress, err := InterpolateVariables(build.Progress, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate build progress: %w", err)
		}
		build.Progress = progress
	}
	
	// Interpolate args
	if build.Args != nil {
		for key, value := range build.Args {
			interpolatedValue, err := InterpolateVariables(value, variables)
			if err != nil {
				return fmt.Errorf("failed to interpolate build arg %s: %w", key, err)
			}
			build.Args[key] = interpolatedValue
		}
	}
	
	// Interpolate tags
	for i, tag := range build.Tags {
		interpolatedTag, err := InterpolateVariables(tag, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate build tag %d: %w", i, err)
		}
		build.Tags[i] = interpolatedTag
	}
	
	return nil
}

// interpolateContainerSlim interpolates variables in container slim configuration
func interpolateContainerSlim(slim *container.ContainerSlim, variables map[string]string) error {
	if slim == nil {
		return nil
	}
	
	// Interpolate target
	if slim.Target != "" {
		target, err := InterpolateVariables(slim.Target, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate slim target: %w", err)
		}
		slim.Target = target
	}
	
	// Interpolate network
	if slim.Network != "" {
		network, err := InterpolateVariables(slim.Network, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate slim network: %w", err)
		}
		slim.Network = network
	}
	
	// Interpolate exec
	if slim.Exec != "" {
		exec, err := InterpolateVariables(slim.Exec, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate slim exec: %w", err)
		}
		slim.Exec = exec
	}
	
	// Interpolate tags
	for i, tag := range slim.Tags {
		interpolatedTag, err := InterpolateVariables(tag, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate slim tag %d: %w", i, err)
		}
		slim.Tags[i] = interpolatedTag
	}
	
	return nil
}

// interpolateContainerArtifacts interpolates variables in container artifacts configuration
func interpolateContainerArtifacts(artifacts *container.ContainerArtifacts, variables map[string]string) error {
	if artifacts == nil {
		return nil
	}
	
	// Interpolate output
	if artifacts.Output != "" {
		output, err := InterpolateVariables(artifacts.Output, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate artifacts output: %w", err)
		}
		artifacts.Output = output
	}
	
	// Interpolate paths
	for i, path := range artifacts.Path {
		interpolatedPath, err := InterpolateVariables(path, variables)
		if err != nil {
			return fmt.Errorf("failed to interpolate artifacts path %d: %w", i, err)
		}
		artifacts.Path[i] = interpolatedPath
	}
	
	return nil
}
