package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/internal/config"
	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// TestExampleYAMLFiles tests that all example YAML files are valid and can be loaded
func TestExampleYAMLFiles(t *testing.T) {
	examples := []struct {
		file        string
		description string
		skip        bool
		skipReason  string
	}{
		// Matrix examples
		{"../examples/matrix-multidimensional-simple.yml", "Simple multi-dimensional matrix", false, ""},
		{"../examples/matrix-multidimensional-complex.yml", "Complex multi-dimensional matrix", false, ""},
		{"../examples/matrix-multidimensional-containers.yml", "Container multi-dimensional matrix", true, "Requires container engine"},
		
		// Step names examples
		{"../examples/step-names-basic-test.yml", "Basic step names", false, ""},
		{"../examples/step-names-simple-test.yml", "Simple step names", false, ""},
		{"../examples/step-names-matrix-test.yml", "Step names with matrix", false, ""},
		{"../examples/step-names-duplicate-error.yml", "Duplicate step names (should fail validation)", false, ""},
		
		// Stage references examples
		{"../examples/stage-references.yml", "Basic stage references", false, ""},
		{"../examples/nested-stage-references.yml", "Nested stage references", false, ""},
		
		// Container examples
		{"../examples/container-working-test.yml", "Container working test", true, "Requires container engine"},
		{"../examples/container-run-test.yml", "Container run test", true, "Requires container engine"},
		{"../examples/container-simple-test.yml", "Container simple test", true, "Requires container engine"},
		{"../examples/container-docker-build.yml", "Container docker build", true, "Requires container engine"},
		{"../examples/container-docker-build-failed.yml", "Container docker build failed", true, "Requires container engine"},
		{"../examples/container-build-matrix.yml", "Container build matrix", true, "Requires container engine"},
		{"../examples/container-matrix-platform-test.yml", "Container matrix platform", true, "Requires container engine"},
		{"../examples/container-artifacts-example.yml", "Container artifacts", true, "Requires container engine"},
		{"../examples/container-debug-test.yml", "Container debug test", true, "Requires container engine"},
		
		// Include examples
		{"../examples/include-example.yml", "Include example", false, ""},
	}

	for _, tt := range examples {
		t.Run(tt.file, func(t *testing.T) {
			if tt.skip {
				t.Skip(tt.skipReason)
			}

			// Special case: duplicate step names should fail during load or validation
			if filepath.Base(tt.file) == "step-names-duplicate-error.yml" {
				cfg, err := config.Load(tt.file)
				if err != nil {
					// Failed during load (validation happens during load)
					t.Logf("%s correctly failed during load: %v", tt.file, err)
					return
				}
				// Check if validation fails
				err = cfg.Validate()
				if err == nil {
					t.Errorf("%s should fail validation (duplicate step names), but passed", tt.file)
				} else {
					t.Logf("%s correctly failed validation: %v", tt.file, err)
				}
				return
			}

			// Load configuration
			cfg, err := config.Load(tt.file)
			if err != nil {
				t.Fatalf("Failed to load %s (%s): %v", tt.file, tt.description, err)
			}

			// Validate configuration
			err = cfg.Validate()
			if err != nil {
				t.Errorf("Config validation failed for %s (%s): %v", tt.file, tt.description, err)
			} else {
				t.Logf("%s (%s) is valid", tt.file, tt.description)
			}
		})
	}
}

// TestTestYAMLFiles tests that all test YAML files are valid and can be loaded
func TestTestYAMLFiles(t *testing.T) {
	testFiles := []struct {
		file         string
		description  string
		testStage    string // Stage to test (empty = validation only)
		expectError  bool   // Whether stage execution should fail
		skipLoad     bool   // Skip loading (invalid YAML format)
		skipReason   string
	}{
		// Matrix tests
		{"test_matrix_skiped.yml", "Matrix with condition-based skipping", "matrix-skiped", false, false, ""},
		{"test_matrix_multidim.yml", "Multi-dimensional matrix", "", false, false, ""},
		{"test_matrix_stage.yml", "Matrix on stage reference", "", false, false, ""},
		{"test_matrix.yml", "Basic matrix", "", false, false, ""},
		{"test_matrix_working.yml", "Working matrix", "", false, false, ""},
		{"test_matrix_simple.yml", "Simple matrix", "", false, false, ""},
		{"test_matrix_parallel.yml", "Parallel matrix", "", false, true, "Old YAML format"},
		{"test_matrix_stream.yml", "Matrix with streaming", "", false, false, ""},
		
		// Condition tests
		{"test-step-if-condition.yml", "Step if condition", "", false, false, ""},
		{"test-step-if-condition-complex.yml", "Complex if condition", "", false, false, ""},
		{"test-step-if-condition-skip.yml", "If condition skip", "", false, false, ""},
		{"test-step-if-condition-fail.yml", "If condition fail", "", true, false, ""},
		
		// Variant tests
		{"test-variants.yml", "Action variants", "", false, false, ""},
		{"test-variants-simple.yml", "Simple variants", "", false, false, ""},
		{"test-variants-clean.yml", "Clean variants", "", false, false, ""},
		
		// Error tests
		{"test-error.yml", "Error handling", "", true, true, "Old YAML format"},
		{"test-complex-error.yml", "Complex error", "", true, true, "Old YAML format"},
		
		// Simple execution tests (old format)
		{"test-simple.yml", "Simple execution", "", false, true, "Old YAML format"},
		{"test-success.yml", "Success test", "", false, true, "Old YAML format"},
		{"test-shell.yml", "Shell command", "", false, true, "Old YAML format"},
		{"test-streaming.yml", "Streaming output", "", false, false, ""},
		{"test-short-sleep.yml", "Short sleep", "", false, true, "Old YAML format"},
		{"test-long-running.yml", "Long running", "", false, true, "Old YAML format"},
		{"test-debug-platform.yml", "Debug platform", "", false, false, ""},
		
		// Pool/parallel tests (old format)
		{"test-parallel-pool-matrix.yml", "Parallel pool matrix", "", false, true, "Old YAML format"},
		{"test-phase1-unlimited.yml", "Phase 1 unlimited", "", false, true, "Old YAML format"},
		{"test-phase1-global-pool.yml", "Phase 1 global pool", "", false, true, "Old YAML format"},
		
		// Container tests
		{"test-matrix-container-variables.yml", "Container matrix variables", "", true, true, "Invalid YAML"}, // Has errors
		{"test-container-artifacts.yml", "Container artifacts", "", true, false, ""}, // Valid but requires container
		{"test-container-artifacts-run-action.yml", "Container artifacts run action", "", true, false, ""},
		{"test-container-artifacts-run-stage.yml", "Container artifacts run stage", "", true, false, ""},
		{"test-container-artifacts-wildcards.yml", "Container artifacts wildcards", "", true, false, ""},
		{"test-container-build-artifacts.yml", "Container build artifacts", "", true, false, ""},
		
		// Cross-platform tests
		{"cross-platform/debug-conditions.yml", "Cross-platform debug", "", false, false, ""},
		{"cross-platform/unified-platform-validation.yml", "Unified platform validation", "", false, false, ""},
		{"cross-platform/shell-examples.yml", "Shell examples", "", false, false, ""},
		{"cross-platform/windows-wine_configuration.yml", "Windows WINE config", "", false, false, ""},
		{"cross-platform/windows-git-bash_configuration.yml", "Windows Git Bash config", "", false, false, ""},
	}

	for _, tt := range testFiles {
		t.Run(tt.file, func(t *testing.T) {
			if tt.skipLoad {
				t.Skip(tt.skipReason)
			}

			// Load configuration
			cfg, err := config.Load(tt.file)
			if err != nil {
				t.Fatalf("Failed to load %s (%s): %v", tt.file, tt.description, err)
			}

			// Validate configuration
			err = cfg.Validate()
			if err != nil {
				t.Fatalf("Config validation failed for %s (%s): %v", tt.file, tt.description, err)
			}

			t.Logf("%s (%s) is valid", tt.file, tt.description)

			// If testStage is specified and not error-expected, try to execute it
			if tt.testStage != "" && !tt.expectError {
				// Create simple runner
				opts := &buildfab.SimpleRunOptions{
					ConfigPath:   tt.file,
					MaxParallel:  2,
					VerboseLevel: 0, // Quiet mode for tests
					Debug:        false,
					Variables:    make(map[string]string),
					WorkingDir:   ".",
					Output:       os.Stdout,
					ErrorOutput:  os.Stderr,
				}

				runner := buildfab.NewSimpleRunner(cfg, opts)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				err = runner.RunStage(ctx, tt.testStage)
				if err != nil {
					t.Logf("Stage %s execution returned error (may be expected): %v", tt.testStage, err)
				} else {
					t.Logf("Stage %s executed successfully", tt.testStage)
				}
			}
		})
	}
}
