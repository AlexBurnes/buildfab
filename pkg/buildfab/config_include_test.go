package buildfab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_WithIncludes(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - actions.yml
  - stages.yml
actions:
  - name: main-action
    run: echo "main action"
stages:
  main:
    steps:
      - action: main-action
`

	// Create included actions file
	actionsFile := filepath.Join(tempDir, "actions.yml")
	actionsContent := `
actions:
  - name: included-action
    run: echo "included action"
  - name: main-action
    run: echo "main action overridden"
`

	// Create included stages file
	stagesFile := filepath.Join(tempDir, "stages.yml")
	stagesContent := `
stages:
  included:
    steps:
      - action: included-action
  main:
    steps:
      - action: included-action
`

	// Write files
	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(actionsFile, []byte(actionsContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stagesFile, []byte(stagesContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration
	config, err := LoadConfig(mainConfig)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	// Verify project name
	if config.Project.Name != "test-project" {
		t.Errorf("expected project name 'test-project', got '%s'", config.Project.Name)
	}

	// Verify actions were merged
	if len(config.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(config.Actions))
	}

	// Verify main-action was overridden
	mainActionFound := false
	for _, action := range config.Actions {
		if action.Name == "main-action" {
			if action.Run != "echo \"main action overridden\"" {
				t.Errorf("expected main-action to be overridden, got: %s", action.Run)
			}
			mainActionFound = true
			break
		}
	}
	if !mainActionFound {
		t.Error("main-action not found in merged config")
	}

	// Verify included-action was added
	includedActionFound := false
	for _, action := range config.Actions {
		if action.Name == "included-action" {
			includedActionFound = true
			break
		}
	}
	if !includedActionFound {
		t.Error("included-action not found in merged config")
	}

	// Verify stages were merged
	if len(config.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(config.Stages))
	}

	// Verify included stage exists
	if _, exists := config.Stages["included"]; !exists {
		t.Error("included stage not found in merged config")
	}

	// Verify main stage was overridden
	if mainStage, exists := config.Stages["main"]; exists {
		if len(mainStage.Steps) != 1 || mainStage.Steps[0].Action != "included-action" {
			t.Errorf("expected main stage to be overridden with included-action step")
		}
	} else {
		t.Error("main stage not found in merged config")
	}
}

func TestLoadConfig_IncludeWithGlobPattern(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - config-*.yml
actions:
  - name: main-action
    run: echo "main action"
`

	// Create multiple config files matching the pattern
	configFiles := []string{"config-actions.yml", "config-stages.yml", "config-other.yml"}
	for i, file := range configFiles {
		content := `
actions:
  - name: action` + string(rune('0'+i+1)) + `
    run: echo "action ` + string(rune('0'+i+1)) + `"
`
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a file that doesn't match the pattern (should be ignored)
	if err := os.WriteFile(filepath.Join(tempDir, "other.txt"), []byte("not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write main config
	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration
	config, err := LoadConfig(mainConfig)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	// Verify all actions were loaded (main + 3 from included files)
	expectedActions := 4
	if len(config.Actions) != expectedActions {
		t.Errorf("expected %d actions, got %d", expectedActions, len(config.Actions))
	}

	// Verify specific actions exist
	actionNames := make(map[string]bool)
	for _, action := range config.Actions {
		actionNames[action.Name] = true
	}

	expectedActionNames := []string{"main-action", "action1", "action2", "action3"}
	for _, name := range expectedActionNames {
		if !actionNames[name] {
			t.Errorf("expected action '%s' not found", name)
		}
	}
}

func TestLoadConfig_IncludeNonExistentFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file with non-existent include
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - nonexistent.yml
actions:
  - name: main-action
    run: echo "main action"
`

	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration - should fail
	_, err = LoadConfig(mainConfig)
	if err == nil {
		t.Error("expected error for non-existent include file, but got none")
	}

	if err != nil && !contains(err.Error(), "included file does not exist") {
		t.Errorf("expected 'included file does not exist' error, got: %v", err)
	}
}

func TestLoadConfig_IncludeGlobWithNonExistentDirectory(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file with glob pattern for non-existent directory
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - nonexistent/*.yml
actions:
  - name: main-action
    run: echo "main action"
`

	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration - should fail
	_, err = LoadConfig(mainConfig)
	if err == nil {
		t.Error("expected error for non-existent directory in glob pattern, but got none")
	}

	if err != nil && !contains(err.Error(), "directory for include pattern does not exist") {
		t.Errorf("expected 'directory for include pattern does not exist' error, got: %v", err)
	}
}

func TestLoadConfig_IncludeGlobWithNoMatches(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file with glob pattern that has no matches
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - nonexistent*.yml
actions:
  - name: main-action
    run: echo "main action"
`

	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration - should succeed (no matches is allowed for glob patterns)
	config, err := LoadConfig(mainConfig)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	// Should only have the main action
	if len(config.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(config.Actions))
	}

	if config.Actions[0].Name != "main-action" {
		t.Errorf("expected main-action, got %s", config.Actions[0].Name)
	}
}

func TestLoadConfig_CircularInclude(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file that includes itself
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - project.yml
actions:
  - name: main-action
    run: echo "main action"
`

	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration - should fail with circular include error
	_, err = LoadConfig(mainConfig)
	if err == nil {
		t.Error("expected error for circular include, but got none")
	}

	if err != nil && !contains(err.Error(), "circular include detected") {
		t.Errorf("expected 'circular include detected' error, got: %v", err)
	}
}

func TestLoadConfig_IncludeInvalidYAML(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buildfab-config-include-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create main configuration file
	mainConfig := filepath.Join(tempDir, "project.yml")
	mainConfigContent := `
project:
  name: test-project
include:
  - invalid.yml
actions:
  - name: main-action
    run: echo "main action"
`

	// Create invalid YAML file
	invalidFile := filepath.Join(tempDir, "invalid.yml")
	invalidContent := `
actions:
  - name: invalid-action
    run: echo "invalid action"
  invalid: yaml: structure
`

	if err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidFile, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load configuration - should fail with YAML parsing error
	_, err = LoadConfig(mainConfig)
	if err == nil {
		t.Error("expected error for invalid YAML in included file, but got none")
	}

	if err != nil && !contains(err.Error(), "failed to parse YAML") {
		t.Errorf("expected YAML parsing error, got: %v", err)
	}
}

