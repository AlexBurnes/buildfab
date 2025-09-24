package buildfab

import (
	"fmt"
	"os"
	"path/filepath"

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