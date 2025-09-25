package buildfab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// If path is empty, try default locations
	if path == "" {
		paths := []string{".project.yml", "project.yml", ".buildfab.yml", "buildfab.yml"}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
		if path == "" {
			return nil, fmt.Errorf("no configuration file found in default locations")
		}
	}
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
	}
	
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}
	
	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}
	
	// Set working directory to the directory containing the config file
	if config.Project.BinDir == "" {
		config.Project.BinDir = filepath.Dir(path)
	}
	
	// Process includes if present
	if len(config.Include) > 0 {
		baseDir := filepath.Dir(path)
		if err := processIncludes(&config, baseDir); err != nil {
			return nil, fmt.Errorf("failed to process includes: %w", err)
		}
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err // Return original validation error to preserve line number information
	}
	
	return &config, nil
}

// LoadConfigFromBytes loads configuration from YAML bytes
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	return &config, nil
}

// processIncludes processes include patterns and merges included configurations
func processIncludes(config *Config, baseDir string) error {
	// Import the internal config package to use IncludeResolver
	// This is a simplified version that doesn't require the full internal package
	visited := make(map[string]bool)
	
	for _, pattern := range config.Include {
		if err := processIncludePattern(config, pattern, baseDir, visited); err != nil {
			return err
		}
	}
	
	return nil
}

// processIncludePattern processes a single include pattern
func processIncludePattern(config *Config, pattern, baseDir string, visited map[string]bool) error {
	// Convert to absolute path if relative
	absPattern := pattern
	if !filepath.IsAbs(pattern) {
		absPattern = filepath.Join(baseDir, pattern)
	}
	
	// Check if pattern contains wildcards
	if strings.Contains(pattern, "*") {
		return processGlobPattern(config, absPattern, visited)
	}
	
	// Exact file path
	return processExactFile(config, absPattern, visited)
}

// processExactFile processes an exact file path
func processExactFile(config *Config, path string, visited map[string]bool) error {
	// Check for circular includes
	if visited[path] {
		return fmt.Errorf("circular include detected: %s", path)
	}
	visited[path] = true
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("included file does not exist: %s", path)
	}
	
	// Load and merge the file
	err := loadAndMergeFile(config, path, visited)
	
	// Remove from visited map when done processing this file
	delete(visited, path)
	
	return err
}

// processGlobPattern processes a glob pattern
func processGlobPattern(config *Config, pattern string, visited map[string]bool) error {
	// Extract directory from pattern
	dir := filepath.Dir(pattern)
	
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory for include pattern does not exist: %s", dir)
	}
	
	// Find matching files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	
	// Filter to only include YAML files and process them
	for _, match := range matches {
		if strings.HasSuffix(strings.ToLower(match), ".yml") || strings.HasSuffix(strings.ToLower(match), ".yaml") {
			if err := processExactFile(config, match, visited); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// loadAndMergeFile loads a file and merges it into the config
func loadAndMergeFile(config *Config, path string, visited map[string]bool) error {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}
	
	// Parse YAML
	var includedConfig Config
	if err := yaml.Unmarshal(content, &includedConfig); err != nil {
		return fmt.Errorf("failed to parse YAML in file %s: %w", path, err)
	}
	
	// Process includes in the included file if any
	if len(includedConfig.Include) > 0 {
		baseDir := filepath.Dir(path)
		
		for _, includePattern := range includedConfig.Include {
			if err := processIncludePattern(config, includePattern, baseDir, visited); err != nil {
				return err
			}
		}
	}
	
	// Merge actions (later actions override earlier ones with same name)
	for _, action := range includedConfig.Actions {
		found := false
		for i, existing := range config.Actions {
			if existing.Name == action.Name {
				config.Actions[i] = action
				found = true
				break
			}
		}
		if !found {
			config.Actions = append(config.Actions, action)
		}
	}
	
	// Merge stages (later stages override earlier ones)
	if config.Stages == nil {
		config.Stages = make(map[string]Stage)
	}
	for name, stage := range includedConfig.Stages {
		config.Stages[name] = stage
	}
	
	return nil
}