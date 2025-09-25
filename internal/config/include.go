// Package config provides configuration loading and validation functionality.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IncludeResolver handles resolving include patterns and loading included files
type IncludeResolver struct {
	baseDir string
	visited map[string]bool // Track visited files to prevent circular includes
}

// NewIncludeResolver creates a new include resolver
func NewIncludeResolver(baseDir string) *IncludeResolver {
	return &IncludeResolver{
		baseDir: baseDir,
		visited: make(map[string]bool),
	}
}

// ResolveIncludes processes include patterns and returns resolved file paths
func (ir *IncludeResolver) ResolveIncludes(patterns []string) ([]string, error) {
	var resolvedFiles []string
	seen := make(map[string]bool)
	
	for _, pattern := range patterns {
		files, err := ir.resolvePattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve include pattern %q: %w", pattern, err)
		}
		
		// Add files, avoiding duplicates
		for _, file := range files {
			if !seen[file] {
				resolvedFiles = append(resolvedFiles, file)
				seen[file] = true
			}
		}
	}
	
	return resolvedFiles, nil
}

// resolvePattern resolves a single include pattern
func (ir *IncludeResolver) resolvePattern(pattern string) ([]string, error) {
	// Convert to absolute path if relative
	absPattern := pattern
	if !filepath.IsAbs(pattern) {
		absPattern = filepath.Join(ir.baseDir, pattern)
	}
	
	// Check if pattern contains wildcards
	if strings.Contains(pattern, "*") {
		return ir.resolveGlobPattern(absPattern)
	}
	
	// Exact file path
	return ir.resolveExactPath(absPattern)
}

// resolveExactPath resolves an exact file path
func (ir *IncludeResolver) resolveExactPath(path string) ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("included file does not exist: %s", path)
	}
	
	return []string{path}, nil
}

// resolveGlobPattern resolves a glob pattern
func (ir *IncludeResolver) resolveGlobPattern(pattern string) ([]string, error) {
	// Extract directory from pattern
	dir := filepath.Dir(pattern)
	
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory for include pattern does not exist: %s", dir)
	}
	
	// Find matching files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	
	// Filter to only include YAML files
	var yamlFiles []string
	for _, match := range matches {
		if strings.HasSuffix(strings.ToLower(match), ".yml") || strings.HasSuffix(strings.ToLower(match), ".yaml") {
			yamlFiles = append(yamlFiles, match)
		}
	}
	
	// Return empty list if no matches - this is allowed for glob patterns
	return yamlFiles, nil
}

// LoadIncludedConfigs loads and parses included configuration files
func (ir *IncludeResolver) LoadIncludedConfigs(filePaths []string) (*PartialConfig, error) {
	var mergedConfig PartialConfig
	
	for _, filePath := range filePaths {
		// Check for circular includes
		if ir.visited[filePath] {
			return nil, fmt.Errorf("circular include detected: %s", filePath)
		}
		ir.visited[filePath] = true
		
		// Load the included file
		config, err := ir.loadIncludedFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load included file %s: %w", filePath, err)
		}
		
		// Merge with accumulated config
		ir.mergeConfig(&mergedConfig, config)
	}
	
	return &mergedConfig, nil
}

// PartialConfig represents a partial configuration that can be merged
type PartialConfig struct {
	Actions []Action `yaml:"actions,omitempty"`
	Stages  map[string]Stage `yaml:"stages,omitempty"`
}

// Action represents a single action (duplicated from buildfab package for include processing)
type Action struct {
	Name     string          `yaml:"name"`
	Run      string          `yaml:"run,omitempty"`
	Uses     string          `yaml:"uses,omitempty"`
	Shell    string          `yaml:"shell,omitempty"`
	Variants []ActionVariant `yaml:"variants,omitempty"`
}

// ActionVariant represents a conditional variant of an action
type ActionVariant struct {
	When  string `yaml:"when"`
	Run   string `yaml:"run,omitempty"`
	Uses  string `yaml:"uses,omitempty"`
	Shell string `yaml:"shell,omitempty"`
}

// Stage represents a collection of steps (duplicated from buildfab package for include processing)
type Stage struct {
	Steps []Step `yaml:"steps"`
}

// Step represents a single step in a stage
type Step struct {
	Action  string   `yaml:"action"`
	Require []string `yaml:"require,omitempty"`
	OnError string   `yaml:"onerror,omitempty"`
	If      string   `yaml:"if,omitempty"`
	Only    []string `yaml:"only,omitempty"`
}

// loadIncludedFile loads and parses a single included file
func (ir *IncludeResolver) loadIncludedFile(filePath string) (*PartialConfig, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Parse YAML
	var config PartialConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	return &config, nil
}

// mergeConfig merges an included configuration into the accumulated config
func (ir *IncludeResolver) mergeConfig(merged *PartialConfig, included *PartialConfig) {
	// Merge actions (later actions override earlier ones with same name)
	for _, action := range included.Actions {
		// Check if action already exists
		found := false
		for i, existing := range merged.Actions {
			if existing.Name == action.Name {
				merged.Actions[i] = action
				found = true
				break
			}
		}
		if !found {
			merged.Actions = append(merged.Actions, action)
		}
	}
	
	// Merge stages (later stages override earlier ones)
	if merged.Stages == nil {
		merged.Stages = make(map[string]Stage)
	}
	for name, stage := range included.Stages {
		merged.Stages[name] = stage
	}
}
