package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIncludeResolver_ResolveExactPath(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.yml")
	if err := os.WriteFile(testFile, []byte("actions: []"), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver(tempDir)

	tests := []struct {
		name        string
		pattern     string
		expectError bool
		expectFiles int
	}{
		{
			name:        "existing file",
			pattern:     "test.yml",
			expectError: false,
			expectFiles: 1,
		},
		{
			name:        "non-existing file",
			pattern:     "nonexistent.yml",
			expectError: true,
			expectFiles: 0,
		},
		{
			name:        "absolute path to existing file",
			pattern:     testFile,
			expectError: false,
			expectFiles: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := resolver.resolvePattern(tt.pattern)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(files) != tt.expectFiles {
				t.Errorf("expected %d files, got %d", tt.expectFiles, len(files))
			}
		})
	}
}

func TestIncludeResolver_ResolveGlobPattern(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := []string{
		"test1.yml",
		"test2.yaml",
		"test3.txt", // Should be ignored
		"config.yml",
	}
	
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte("actions: []"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create subdirectory with files
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	subFiles := []string{"sub1.yml", "sub2.yaml"}
	for _, file := range subFiles {
		if err := os.WriteFile(filepath.Join(subDir, file), []byte("actions: []"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	resolver := NewIncludeResolver(tempDir)

	tests := []struct {
		name        string
		pattern     string
		expectError bool
		minFiles    int // Minimum expected files (glob patterns can vary)
	}{
		{
			name:        "glob pattern matching yaml files",
			pattern:     "test*.yml",
			expectError: false,
			minFiles:    1,
		},
		{
			name:        "glob pattern matching yaml files with different extension",
			pattern:     "test*.yaml",
			expectError: false,
			minFiles:    1,
		},
		{
			name:        "glob pattern in subdirectory",
			pattern:     "subdir/*.yml",
			expectError: false,
			minFiles:    1,
		},
		{
			name:        "glob pattern with no matches",
			pattern:     "nonexistent*.yml",
			expectError: false,
			minFiles:    0,
		},
		{
			name:        "glob pattern with non-existing directory",
			pattern:     "nonexistent/*.yml",
			expectError: true,
			minFiles:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := resolver.resolvePattern(tt.pattern)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(files) < tt.minFiles {
				t.Errorf("expected at least %d files, got %d", tt.minFiles, len(files))
			}
			
			// Verify all returned files are YAML files
			for _, file := range files {
				ext := filepath.Ext(file)
				if ext != ".yml" && ext != ".yaml" {
					t.Errorf("expected YAML file, got %s", file)
				}
			}
		})
	}
}

func TestIncludeResolver_LoadIncludedConfigs(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration files
	config1 := filepath.Join(tempDir, "config1.yml")
	config1Content := `
actions:
  - name: action1
    run: echo "action1"
stages:
  stage1:
    steps:
      - action: action1
`

	config2 := filepath.Join(tempDir, "config2.yml")
	config2Content := `
actions:
  - name: action2
    run: echo "action2"
  - name: action1
    run: echo "action1-overridden"
stages:
  stage2:
    steps:
      - action: action2
`

	if err := os.WriteFile(config1, []byte(config1Content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config2, []byte(config2Content), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver(tempDir)
	
	// Test loading included configs
	config, err := resolver.LoadIncludedConfigs([]string{config1, config2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify actions
	if len(config.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(config.Actions))
	}

	// Verify action1 was overridden
	action1Found := false
	for _, action := range config.Actions {
		if action.Name == "action1" {
			if action.Run != "echo \"action1-overridden\"" {
				t.Errorf("expected action1 to be overridden, got: %s", action.Run)
			}
			action1Found = true
			break
		}
	}
	if !action1Found {
		t.Error("action1 not found in merged config")
	}

	// Verify stages
	if len(config.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(config.Stages))
	}

	if _, exists := config.Stages["stage1"]; !exists {
		t.Error("stage1 not found in merged config")
	}
	if _, exists := config.Stages["stage2"]; !exists {
		t.Error("stage2 not found in merged config")
	}
}

func TestIncludeResolver_CircularInclude(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that includes itself
	configFile := filepath.Join(tempDir, "config.yml")
	configContent := `
include:
  - config.yml
actions: []
`
	
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewIncludeResolver(tempDir)
	
	// Test circular include detection
	_, err = resolver.LoadIncludedConfigs([]string{configFile, configFile})
	if err == nil {
		t.Error("expected error for circular include, but got none")
	}
	
	if err != nil && !contains(err.Error(), "circular include detected") {
		t.Errorf("expected circular include error, got: %v", err)
	}
}

func TestIncludeResolver_ResolveIncludes(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := []string{"file1.yml", "file2.yml", "file3.txt"}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte("actions: []"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	resolver := NewIncludeResolver(tempDir)

	tests := []struct {
		name        string
		patterns    []string
		expectError bool
		expectFiles int
	}{
		{
			name:        "exact file patterns",
			patterns:    []string{"file1.yml", "file2.yml"},
			expectError: false,
			expectFiles: 2,
		},
		{
			name:        "mixed exact and glob patterns",
			patterns:    []string{"file1.yml", "file*.yml"},
			expectError: false,
			expectFiles: 2, // file1.yml (exact, but also matches glob) + file2.yml (from glob) - deduplicated to 2
		},
		{
			name:        "non-existing exact file",
			patterns:    []string{"nonexistent.yml"},
			expectError: true,
			expectFiles: 0,
		},
		{
			name:        "glob pattern with no matches",
			patterns:    []string{"nonexistent*.yml"},
			expectError: false,
			expectFiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := resolver.ResolveIncludes(tt.patterns)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if len(files) != tt.expectFiles {
				t.Errorf("expected %d files, got %d", tt.expectFiles, len(files))
			}
		})
	}
}

// Helper function to check if error message contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
				s[len(s)-len(substr):] == substr || 
				containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
