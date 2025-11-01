package buildfab

import (
	"context"
	"os"
	"testing"
)

// TestIntegration_MultiDimensionalMatrix tests full integration of multi-dimensional matrices
func TestIntegration_MultiDimensionalMatrix(t *testing.T) {
	config := &Config{
		Project: Project{
			Name:        "test-multidim",
			MaxParallel: 4,
		},
		Actions: []Action{
			{
				Name: "test-build",
				Run:  "echo \"Platform: ${{ matrix.platform }}, Compiler: ${{ matrix.compiler }}, Config: ${{ matrix.config }}\"",
			},
		},
		Stages: map[string]Stage{
			"build-matrix": {
				Steps: []Step{
					{
						Action: "test-build",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"platform": {
									map[string]interface{}{
										"linux": map[string]interface{}{
											"compiler": []interface{}{"gcc", "clang"},
										},
									},
									map[string]interface{}{
										"windows": map[string]interface{}{
											"compiler": "msvc",
										},
									},
								},
								"config": {"Release", "Debug"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 4,
							},
						},
					},
				},
			},
		},
	}

	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunStage(ctx, "build-matrix")
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}

	// Expected: (2 + 1) * 2 = 6 combinations
	// Success if no error
	t.Logf("Multi-dimensional matrix test passed with 6 expected combinations")
}

// TestIntegration_MultiDimensionalMatrix_ThreeLevels tests three-level nested matrix
func TestIntegration_MultiDimensionalMatrix_ThreeLevels(t *testing.T) {
	config := &Config{
		Project: Project{
			Name: "test-three-levels",
		},
		Actions: []Action{
			{
				Name: "build",
				Run:  "echo \"Image: ${{ matrix.images }}, Compiler: ${{ matrix.compiler }}, Build: ${{ matrix.builds }}\"",
			},
		},
		Stages: map[string]Stage{
			"full-matrix": {
				Steps: []Step{
					{
						Action: "build",
						Matrix: &MatrixConfig{
							Values: map[string][]interface{}{
								"images": {
									map[string]interface{}{
										"centos7": map[string]interface{}{
											"compiler": "gcc",
										},
									},
									map[string]interface{}{
										"centos8": map[string]interface{}{
											"compiler": []interface{}{"gcc", "clang"},
										},
									},
									map[string]interface{}{
										"centos9": map[string]interface{}{
											"compiler": []interface{}{"gcc", "clang", "icc"},
										},
									},
								},
								"builds": {"Release", "Debug"},
							},
							Strategy: MatrixStrategy{
								MaxParallel: 4,
							},
						},
					},
				},
			},
		},
	}

	opts := &RunOptions{
		VerboseLevel: 0,
		Debug:        false,
		Variables:    make(map[string]string),
		Output:       os.Stdout,
		ErrorOutput:  os.Stderr,
	}

	runner := NewRunner(config, opts)

	ctx := context.Background()
	err := runner.RunStage(ctx, "full-matrix")
	if err != nil {
		t.Fatalf("Stage execution failed: %v", err)
	}

	// Expected: 12 combinations
	// centos7: 1 compiler * 2 builds = 2
	// centos8: 2 compilers * 2 builds = 4
	// centos9: 3 compilers * 2 builds = 6
	// Total: 12
	t.Logf("Three-level multi-dimensional matrix test passed with 12 expected combinations")
}

