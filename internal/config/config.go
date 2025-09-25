// Package config provides configuration loading and validation functionality.
package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// Loader handles loading and parsing configuration files
type Loader struct {
	configPath string
}

// New creates a new configuration loader
func New(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// Load loads configuration from the specified file
func Load(configPath string) (*buildfab.Config, error) {
	loader := New(configPath)
	return loader.Load()
}

// Load loads and parses the configuration file
func (l *Loader) Load() (*buildfab.Config, error) {
	// Check if file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", l.configPath)
	}

	// Open file
	file, err := os.Open(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var config buildfab.Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Process includes if present
	if len(config.Include) > 0 {
		baseDir := filepath.Dir(l.configPath)
		resolver := NewIncludeResolver(baseDir)
		
		// Resolve include patterns
		includedFiles, err := resolver.ResolveIncludes(config.Include)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve includes: %w", err)
		}
		
		// Load included configurations
		includedConfig, err := resolver.LoadIncludedConfigs(includedFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to load included configurations: %w", err)
		}
		
		// Merge included configurations
		mergeConfig(&config, includedConfig)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err // Return original validation error to preserve line number information
	}

	return &config, nil
}

// LoadFromDir searches for configuration files in the specified directory
func LoadFromDir(dir string) (*buildfab.Config, error) {
	// Common configuration file names
	configFiles := []string{
		".project.yml",
		".project.yaml",
		"project.yml",
		"project.yaml",
		"buildfab.yml",
		"buildfab.yaml",
	}

	for _, filename := range configFiles {
		configPath := filepath.Join(dir, filename)
		if _, err := os.Stat(configPath); err == nil {
			return Load(configPath)
		}
	}

	return nil, fmt.Errorf("no configuration file found in directory: %s", dir)
}

// ResolveVariables resolves variable interpolation in configuration
func ResolveVariables(config *buildfab.Config, variables map[string]string) error {
	// Resolve variables in action run commands
	for i := range config.Actions {
		if config.Actions[i].Run != "" {
			resolved, err := resolveString(config.Actions[i].Run, variables)
			if err != nil {
				return fmt.Errorf("failed to resolve variables in action %s: %w", config.Actions[i].Name, err)
			}
			config.Actions[i].Run = resolved
		}
	}

	return nil
}

// resolveString resolves variable interpolation in a string
func resolveString(s string, variables map[string]string) (string, error) {
	result := s
	
	// Find all variable references in the format ${{ variable }}
	for {
		start := strings.Index(result, "${{")
		if start == -1 {
			break
		}
		
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			return "", fmt.Errorf("unclosed variable reference: %s", result[start:])
		}
		
		end += start + 2 // Adjust for the start position
		
		variableName := strings.TrimSpace(result[start+3 : end-2])
		
		value, exists := variables[variableName]
		if !exists {
			return "", fmt.Errorf("undefined variable: %s", variableName)
		}
		
		result = result[:start] + value + result[end:]
	}
	
	return result, nil
}

// mergeConfig merges included configuration into the main config
func mergeConfig(config *buildfab.Config, included *PartialConfig) {
	// Merge actions (later actions override earlier ones with same name)
	for _, action := range included.Actions {
		// Convert to buildfab.Action type
		buildfabAction := buildfab.Action{
			Name:     action.Name,
			Run:      action.Run,
			Uses:     action.Uses,
			Shell:    action.Shell,
			Variants: make([]buildfab.ActionVariant, len(action.Variants)),
		}
		
		// Copy variants
		for i, variant := range action.Variants {
			buildfabAction.Variants[i] = buildfab.ActionVariant{
				When:  variant.When,
				Run:   variant.Run,
				Uses:  variant.Uses,
				Shell: variant.Shell,
			}
		}
		
		// Check if action already exists
		found := false
		for i, existing := range config.Actions {
			if existing.Name == buildfabAction.Name {
				config.Actions[i] = buildfabAction
				found = true
				break
			}
		}
		if !found {
			config.Actions = append(config.Actions, buildfabAction)
		}
	}
	
	// Merge stages (later stages override earlier ones)
	if config.Stages == nil {
		config.Stages = make(map[string]buildfab.Stage)
	}
	for name, stage := range included.Stages {
		// Convert to buildfab.Stage type
		buildfabStage := buildfab.Stage{
			Steps: make([]buildfab.Step, len(stage.Steps)),
		}
		
		// Copy steps
		for i, step := range stage.Steps {
			buildfabStage.Steps[i] = buildfab.Step{
				Action:  step.Action,
				Require: step.Require,
				OnError: step.OnError,
				If:      step.If,
				Only:    step.Only,
			}
		}
		
		config.Stages[name] = buildfabStage
	}
}

// GetDefaultVariables returns default variables available for interpolation
func GetDefaultVariables() map[string]string {
	return map[string]string{
		"tag":    "", // Will be set by version detection
		"branch": "", // Will be set by git detection
	}
}

// DetectGitVariables detects Git-related variables from the current repository
func DetectGitVariables(ctx context.Context) (map[string]string, error) {
	variables := make(map[string]string)
	
	// Detect current tag
	if tag, err := detectGitTag(ctx); err == nil {
		variables["tag"] = tag
	}
	
	// Detect current branch
	if branch, err := detectGitBranch(ctx); err == nil {
		variables["branch"] = branch
	}
	
	return variables, nil
}

// detectGitTag detects the current Git tag
func detectGitTag(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// detectGitBranch detects the current Git branch
func detectGitBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}