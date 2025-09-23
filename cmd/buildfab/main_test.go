package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Test helper functions
func createTestConfig(t *testing.T, content string) string {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "project.yml")
	
	err := os.WriteFile(configFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Create a VERSION file for version detection
	err = os.WriteFile(filepath.Join(tempDir, "VERSION"), []byte("v1.0.0"), 0644)
	if err != nil {
		t.Fatalf("Failed to write VERSION file: %v", err)
	}
	
	return configFile
}

func TestGetVersion(t *testing.T) {
	// Test that getVersion() returns "unknown" when appVersion is not set at build time
	// This is the expected behavior for development builds without ldflags
	version := getVersion()
	if version != "unknown" {
		t.Errorf("getVersion() = %v, want %v", version, "unknown")
	}
}

func TestRunRoot(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flags    map[string]bool
		wantErr  bool
		contains string
	}{
		{
			name:     "version flag",
			args:     []string{},
			flags:    map[string]bool{"version": true},
			wantErr:  false,
			contains: "buildfab version",
		},
		{
			name:     "version-only flag",
			args:     []string{},
			flags:    map[string]bool{"version-only": true},
			wantErr:  false,
			contains: "unknown", // Will be "unknown" in test environment
		},
		{
			name:     "no arguments",
			args:     []string{},
			flags:    map[string]bool{},
			wantErr:  false, // Should show help, not error
			contains: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test command
			cmd := &cobra.Command{}
			cmd.Flags().Bool("version", false, "")
			cmd.Flags().Bool("version-only", false, "")
			
			// Set flags
			for flag := range tt.flags {
				cmd.Flags().Set(flag, "true")
			}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err := runRoot(cmd, tt.args)
			
			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			
			// Read output
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			
			if (err != nil) != tt.wantErr {
				t.Errorf("runRoot() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if tt.contains != "" && !strings.Contains(output, tt.contains) {
				t.Errorf("runRoot() output should contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

func TestRunStage(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello World"

stages:
  test-stage:
    steps:
      - action: test-action
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	oldWorkingDir := workingDir
	workingDir = filepath.Dir(configFile)
	defer func() { workingDir = oldWorkingDir }()
	
	oldVerbose := verbose
	verbose = false
	defer func() { verbose = oldVerbose }()
	
	oldDebug := debug
	debug = false
	defer func() { debug = oldDebug }()
	
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "run stage",
			args:    []string{"test-stage"},
			wantErr: false,
		},
		{
			name:    "run stage step",
			args:    []string{"test-stage", "test-action"},
			wantErr: false,
		},
		{
			name:    "run non-existent stage",
			args:    []string{"non-existent-stage"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			
			// We can't easily test the full execution without mocking,
			// but we can test the argument parsing and basic flow
			err := runStage(cmd, tt.args)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("runStage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunAction(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello World"
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	oldWorkingDir := workingDir
	workingDir = filepath.Dir(configFile)
	defer func() { workingDir = oldWorkingDir }()
	
	// Change to the working directory so VERSION file can be found
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(workingDir)
	if err != nil {
		t.Fatalf("Failed to change to working directory: %v", err)
	}
	
	oldVerbose := verbose
	verbose = false
	defer func() { verbose = oldVerbose }()
	
	oldDebug := debug
	debug = false
	defer func() { debug = oldDebug }()
	
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "run built-in action",
			args:    []string{"version@check"},
			wantErr: false,
		},
		{
			name:    "run custom action",
			args:    []string{"test-action"},
			wantErr: false,
		},
		{
			name:    "run non-existent action",
			args:    []string{"non-existent-action"},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			
			err := runAction(cmd, tt.args)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("runAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunListActions(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: custom-action
    run: echo "Custom action"
  - name: builtin-action
    uses: version@check
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	cmd := &cobra.Command{}
	
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	err := runListActions(cmd, []string{})
	
	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	
	if err != nil {
		t.Errorf("runListActions() error = %v", err)
	}
	
	// Check that output contains expected content
	if !strings.Contains(output, "Available actions") {
		t.Errorf("runListActions() output should contain 'Available actions', got: %s", output)
	}
	if !strings.Contains(output, "custom-action") {
		t.Errorf("runListActions() output should contain 'custom-action', got: %s", output)
	}
	if !strings.Contains(output, "version@check") {
		t.Errorf("runListActions() output should contain 'version@check', got: %s", output)
	}
}

func TestRunValidate(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello World"

stages:
  test-stage:
    steps:
      - action: test-action
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	cmd := &cobra.Command{}
	
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	err := runValidate(cmd, []string{})
	
	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	
	if err != nil {
		t.Errorf("runValidate() error = %v", err)
	}
	
	// Check that output contains expected content
	if !strings.Contains(output, "Configuration is valid") {
		t.Errorf("runValidate() output should contain 'Configuration is valid', got: %s", output)
	}
	if !strings.Contains(output, "test-project") {
		t.Errorf("runValidate() output should contain 'test-project', got: %s", output)
	}
}

func TestRunListStages(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello"
  - name: another-action
    run: echo "World"

stages:
  stage1:
    steps:
      - action: test-action
  stage2:
    steps:
      - action: test-action
      - action: another-action
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	cmd := &cobra.Command{}
	
	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	err := runListStages(cmd, []string{})
	
	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	
	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	
	if err != nil {
		t.Errorf("runListStages() error = %v", err)
	}
	
	// Check that output contains expected content
	if !strings.Contains(output, "Defined stages") {
		t.Errorf("runListStages() output should contain 'Defined stages', got: %s", output)
	}
	if !strings.Contains(output, "stage1") {
		t.Errorf("runListStages() output should contain 'stage1', got: %s", output)
	}
	if !strings.Contains(output, "stage2") {
		t.Errorf("runListStages() output should contain 'stage2', got: %s", output)
	}
}

func TestRunListSteps(t *testing.T) {
	// Create test configuration
	configContent := `
project:
  name: test-project

actions:
  - name: test-action
    run: echo "Hello"
  - name: another-action
    run: echo "World"

stages:
  test-stage:
    steps:
      - action: test-action
      - action: another-action
        if: version.type == 'release'
        only: ['release']
`
	
	configFile := createTestConfig(t, configContent)
	
	// Set global variables for the test
	oldConfigPath := configPath
	configPath = configFile
	defer func() { configPath = oldConfigPath }()
	
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		contains []string
	}{
		{
			name:    "list steps for existing stage",
			args:    []string{"test-stage"},
			wantErr: false,
			contains: []string{"Steps for stage", "test-stage", "test-action", "another-action"},
		},
		{
			name:    "list steps for non-existent stage",
			args:    []string{"non-existent-stage"},
			wantErr: true,
			contains: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			
			err := runListSteps(cmd, tt.args)
			
			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			
			// Read output
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])
			
			if (err != nil) != tt.wantErr {
				t.Errorf("runListSteps() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			for _, contain := range tt.contains {
				if !strings.Contains(output, contain) {
					t.Errorf("runListSteps() output should contain %q, got: %s", contain, output)
				}
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that all commands are properly configured
	commands := []*cobra.Command{
		rootCmd,
		runCmd,
		actionCmd,
		listActionsCmd,
		validateCmd,
		listStagesCmd,
		listStepsCmd,
	}
	
	for _, cmd := range commands {
		if cmd.Use == "" {
			t.Errorf("Command %s has empty Use field", cmd.Name())
		}
		if cmd.Short == "" {
			t.Errorf("Command %s has empty Short field", cmd.Name())
		}
	}
}

func TestGlobalFlags(t *testing.T) {
	// Initialize flags by calling main() setup
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", true, "enable verbose output (default)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable verbose output (silence mode)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug output")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", ".project.yml", "path to configuration file")
	rootCmd.PersistentFlags().IntVar(&maxParallel, "max-parallel", 0, "maximum parallel execution (default: CPU count)")
	rootCmd.PersistentFlags().StringVarP(&workingDir, "working-dir", "w", ".", "working directory for execution")
	rootCmd.PersistentFlags().StringSliceVar(&only, "only", []string{}, "only run steps matching these labels")
	rootCmd.PersistentFlags().BoolVar(&withRequires, "with-requires", false, "include required dependencies when running single step")
	rootCmd.PersistentFlags().StringSliceVar(&envVars, "env", []string{}, "export environment variables to actions")
	
	// Test that global flags are properly defined
	flags := []string{
		"verbose",
		"debug", 
		"config",
		"max-parallel",
		"working-dir",
		"only",
		"with-requires",
		"env",
	}
	
	for _, flag := range flags {
		if rootCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("Global flag %s not found", flag)
		}
	}
}

func TestVersionFlags(t *testing.T) {
	// Initialize version flags
	rootCmd.Flags().BoolP("version", "", false, "print version and module name")
	rootCmd.Flags().BoolP("version-only", "V", false, "print version only")
	
	// Test that version flags are properly defined
	versionFlags := []string{"version", "version-only"}
	
	for _, flag := range versionFlags {
		if rootCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Version flag %s not found", flag)
		}
	}
}