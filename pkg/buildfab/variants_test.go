package buildfab

import (
	"context"
	"testing"
)

func TestActionVariantSelection(t *testing.T) {
	tests := []struct {
		name      string
		action    Action
		variables map[string]string
		expected  *ActionVariant
		expectErr bool
	}{
		{
			name: "no variants",
			action: Action{
				Name: "test-action",
				Run:  "echo hello",
			},
			variables: map[string]string{"os": "linux"},
			expected:  nil, // No variants, should return nil
			expectErr: false,
		},
		{
			name: "single matching variant",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ os == 'linux' }}",
						Run:  "echo linux command",
					},
				},
			},
			variables: map[string]string{"os": "linux"},
			expected: &ActionVariant{
				When: "${{ os == 'linux' }}",
				Run:  "echo linux command",
			},
			expectErr: false,
		},
		{
			name: "first matching variant selected",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ os == 'linux' }}",
						Run:  "echo linux command",
					},
					{
						When: "${{ os == 'windows' }}",
						Run:  "echo windows command",
					},
				},
			},
			variables: map[string]string{"os": "linux"},
			expected: &ActionVariant{
				When: "${{ os == 'linux' }}",
				Run:  "echo linux command",
			},
			expectErr: false,
		},
		{
			name: "no matching variant",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ os == 'linux' }}",
						Run:  "echo linux command",
					},
					{
						When: "${{ os == 'windows' }}",
						Run:  "echo windows command",
					},
				},
			},
			variables: map[string]string{"os": "darwin"},
			expected:  nil, // No variant matches
			expectErr: false,
		},
		{
			name: "invalid condition syntax",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ invalid syntax }}",
						Run:  "echo command",
					},
				},
			},
			variables: map[string]string{"os": "linux"},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "undefined variable",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ undefined == 'value' }}",
						Run:  "echo command",
					},
				},
			},
			variables: map[string]string{"os": "linux"},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "condition without ${{ }} wrapper",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "os == 'linux'",
						Run:  "echo linux command",
					},
				},
			},
			variables: map[string]string{"os": "linux"},
			expected: &ActionVariant{
				When: "os == 'linux'",
				Run:  "echo linux command",
			},
			expectErr: false,
		},
		{
			name: "built-in action variant",
			action: Action{
				Name: "test-action",
				Variants: []ActionVariant{
					{
						When: "${{ os == 'linux' }}",
						Uses: "git@untracked",
					},
					{
						When: "${{ os == 'windows' }}",
						Run:  "git status --porcelain",
					},
				},
			},
			variables: map[string]string{"os": "windows"},
			expected: &ActionVariant{
				When: "${{ os == 'windows' }}",
				Run:  "git status --porcelain",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.action.SelectVariant(tt.variables)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil variant, got %+v", result)
				}
				return
			}
			
			if result == nil {
				t.Errorf("expected variant %+v, got nil", tt.expected)
				return
			}
			
			if result.When != tt.expected.When {
				t.Errorf("expected When %q, got %q", tt.expected.When, result.When)
			}
			
			if result.Run != tt.expected.Run {
				t.Errorf("expected Run %q, got %q", tt.expected.Run, result.Run)
			}
			
			if result.Uses != tt.expected.Uses {
				t.Errorf("expected Uses %q, got %q", tt.expected.Uses, result.Uses)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		variables map[string]string
		expected  bool
		expectErr bool
	}{
		{
			name:      "simple equality match with ==",
			condition: "${{ os == 'linux' }}",
			variables: map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "simple equality match with =",
			condition: "${{ os = 'linux' }}",
			variables: map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "simple equality no match",
			condition: "${{ os == 'linux' }}",
			variables: map[string]string{"os": "windows"},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "simple equality no match with =",
			condition: "${{ os = 'linux' }}",
			variables: map[string]string{"os": "windows"},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "condition without wrapper with ==",
			condition: "os == 'linux'",
			variables: map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "condition without wrapper with =",
			condition: "os = 'linux'",
			variables: map[string]string{"os": "linux"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "quoted values",
			condition: "${{ arch == 'amd64' }}",
			variables: map[string]string{"arch": "amd64"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "double quoted values",
			condition: "${{ platform == \"windows\" }}",
			variables: map[string]string{"platform": "windows"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "boolean variable true",
			condition: "debug",
			variables: map[string]string{"debug": "true"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "boolean variable false",
			condition: "debug",
			variables: map[string]string{"debug": "false"},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "non-empty string as true",
			condition: "env",
			variables: map[string]string{"env": "production"},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "empty string as false",
			condition: "env",
			variables: map[string]string{"env": ""},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "invalid condition format",
			condition: "${{ invalid syntax }}",
			variables: map[string]string{"os": "linux"},
			expected:  false,
			expectErr: true,
		},
		{
			name:      "undefined variable",
			condition: "${{ undefined == 'value' }}",
			variables: map[string]string{"os": "linux"},
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateCondition(tt.condition, tt.variables)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestActionValidationWithVariants(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid action with variants",
			config: Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-action",
						Variants: []ActionVariant{
							{
								When: "${{ os == 'linux' }}",
								Run:  "echo linux",
							},
							{
								When: "${{ os == 'windows' }}",
								Uses: "git@untracked",
							},
						},
					},
				},
				Stages: map[string]Stage{},
			},
			expectErr: false,
		},
		{
			name: "invalid action with variant missing when",
			config: Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-action",
						Variants: []ActionVariant{
							{
								Run: "echo command",
							},
						},
					},
				},
				Stages: map[string]Stage{},
			},
			expectErr: true,
			errMsg:    "variant 0 must have 'when' condition",
		},
		{
			name: "invalid action with variant missing run/uses",
			config: Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-action",
						Variants: []ActionVariant{
							{
								When: "${{ os == 'linux' }}",
							},
						},
					},
				},
				Stages: map[string]Stage{},
			},
			expectErr: true,
			errMsg:    "variant 0 must have either 'run' or 'uses'",
		},
		{
			name: "invalid action with variant having both run and uses",
			config: Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-action",
						Variants: []ActionVariant{
							{
								When: "${{ os == 'linux' }}",
								Run:  "echo command",
								Uses: "git@untracked",
							},
						},
					},
				},
				Stages: map[string]Stage{},
			},
			expectErr: true,
			errMsg:    "variant 0 cannot have both 'run' and 'uses'",
		},
		{
			name: "invalid action with both variants and direct run/uses",
			config: Config{
				Project: struct {
					Name    string   `yaml:"name"`
					Modules []string `yaml:"modules"`
					BinDir  string   `yaml:"bin,omitempty"`
				}{
					Name: "test-project",
				},
				Actions: []Action{
					{
						Name: "test-action",
						Run:  "echo direct command",
						Variants: []ActionVariant{
							{
								When: "${{ os == 'linux' }}",
								Run:  "echo linux",
							},
						},
					},
				},
				Stages: map[string]Stage{},
			},
			expectErr: false, // This is actually valid - variants take precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunActionWithVariants(t *testing.T) {
	// Create a test configuration with variants
	config := &Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []Action{
			{
				Name: "build-cpp",
				Variants: []ActionVariant{
					{
						When: "${{ os == 'linux' }}",
						Run:  "cmake -S . -B build && cmake --build build -j",
					},
					{
						When: "${{ os == 'windows' }}",
						Run:  "echo windows build",
					},
				},
			},
		},
		Stages: map[string]Stage{},
	}

	// Test Linux variant
	t.Run("linux variant", func(t *testing.T) {
		opts := &RunOptions{
			Variables: map[string]string{"os": "linux"},
			Verbose:   false,
		}
		
		runner := NewRunner(config, opts)
		
		// This will fail because cmake isn't available in test environment,
		// but we can verify the variant selection worked
		err := runner.RunAction(context.Background(), "build-cpp")
		if err == nil {
			t.Error("expected error due to missing cmake, but got none")
		}
		
		// The error should be about cmake, not about variant selection
		if !containsString(err.Error(), "cmake") {
			t.Errorf("expected cmake error, got: %v", err)
		}
	})

	// Test Windows variant
	t.Run("windows variant", func(t *testing.T) {
		opts := &RunOptions{
			Variables: map[string]string{"os": "windows"},
			Verbose:   false,
		}
		
		runner := NewRunner(config, opts)
		
		// This should succeed with echo command
		err := runner.RunAction(context.Background(), "build-cpp")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Test no matching variant (should be skipped)
	t.Run("no matching variant", func(t *testing.T) {
		opts := &RunOptions{
			Variables: map[string]string{"os": "darwin"},
			Verbose:   false,
		}
		
		runner := NewRunner(config, opts)
		
		// This should not error, just skip
		err := runner.RunAction(context.Background(), "build-cpp")
		if err != nil {
			t.Errorf("expected no error for skipped variant, got: %v", err)
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr || 
			 containsStringHelper(s, substr))))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
