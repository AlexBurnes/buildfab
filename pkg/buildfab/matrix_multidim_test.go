package buildfab

import (
	"testing"
)

// TestMultiDimensionalMatrix_BasicExpansion tests basic multi-dimensional matrix expansion
func TestMultiDimensionalMatrix_BasicExpansion(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	// images:
	//   - centos7:
	//       compiler: "gcc"
	//   - centos8:
	//       compiler: ["gcc", "clang"]
	// builds: ["Release", "Debug"]
	matrixValues := map[string][]interface{}{
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
		},
		"builds": {"Release", "Debug"},
	}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: 3 image+compiler combos * 2 builds = 6 combinations
	expectedCount := 6
	if len(combinations) != expectedCount {
		t.Errorf("Expected %d combinations, got %d", expectedCount, len(combinations))
		for i, combo := range combinations {
			t.Logf("Combination %d: %v", i+1, combo)
		}
	}

	// Verify all combinations have required variables
	for i, combo := range combinations {
		if _, ok := combo["images"]; !ok {
			t.Errorf("Combination %d missing 'images' variable: %v", i+1, combo)
		}
		if _, ok := combo["compiler"]; !ok {
			t.Errorf("Combination %d missing 'compiler' variable: %v", i+1, combo)
		}
		if _, ok := combo["builds"]; !ok {
			t.Errorf("Combination %d missing 'builds' variable: %v", i+1, combo)
		}
	}
}

// TestMultiDimensionalMatrix_ThreeLevels tests three-level nested matrix
func TestMultiDimensionalMatrix_ThreeLevels(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	// Full example from user specification:
	// images:
	//   - centos7:
	//       compiler: "gcc"
	//   - centos8:
	//       compiler: ["gcc", "clang"]
	//   - centos9:
	//       compiler: ["gcc", "clang", "icc"]
	// builds: ["Release", "Debug"]
	matrixValues := map[string][]interface{}{
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
	}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: (1 + 2 + 3) * 2 = 12 combinations
	expectedCount := 12
	if len(combinations) != expectedCount {
		t.Errorf("Expected %d combinations, got %d", expectedCount, len(combinations))
		for i, combo := range combinations {
			t.Logf("Combination %d: builds=%v images=%v compiler=%v",
				i+1, combo["builds"], combo["images"], combo["compiler"])
		}
		return
	}

	// Verify expected combinations exist
	expectedCombos := []struct {
		builds   string
		images   string
		compiler string
	}{
		{"Release", "centos7", "gcc"},
		{"Debug", "centos7", "gcc"},
		{"Release", "centos8", "gcc"},
		{"Release", "centos8", "clang"},
		{"Debug", "centos8", "gcc"},
		{"Debug", "centos8", "clang"},
		{"Release", "centos9", "gcc"},
		{"Release", "centos9", "clang"},
		{"Release", "centos9", "icc"},
		{"Debug", "centos9", "gcc"},
		{"Debug", "centos9", "clang"},
		{"Debug", "centos9", "icc"},
	}

	// Create a map for easy lookup
	comboMap := make(map[string]bool)
	for _, combo := range combinations {
		key := combo["builds"].(string) + "|" + combo["images"].(string) + "|" + combo["compiler"].(string)
		comboMap[key] = true
	}

	// Verify all expected combinations exist
	for _, expected := range expectedCombos {
		key := expected.builds + "|" + expected.images + "|" + expected.compiler
		if !comboMap[key] {
			t.Errorf("Missing expected combination: builds=%s images=%s compiler=%s",
				expected.builds, expected.images, expected.compiler)
		}
	}
}

// TestMultiDimensionalMatrix_FlatVariableNaming tests that variables are flattened
func TestMultiDimensionalMatrix_FlatVariableNaming(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	matrixValues := map[string][]interface{}{
		"os": {
			map[string]interface{}{
				"ubuntu": map[string]interface{}{
					"version": []interface{}{"20.04", "22.04"},
				},
			},
		},
	}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: 2 combinations (ubuntu 20.04, ubuntu 22.04)
	if len(combinations) != 2 {
		t.Errorf("Expected 2 combinations, got %d", len(combinations))
	}

	// Verify flat variable naming: "version" not "os.version"
	for i, combo := range combinations {
		if _, ok := combo["os"]; !ok {
			t.Errorf("Combination %d missing 'os' variable", i+1)
		}
		if _, ok := combo["version"]; !ok {
			t.Errorf("Combination %d missing 'version' variable (should be flat, not nested)", i+1)
		}
		// Verify no nested naming
		if _, ok := combo["os.version"]; ok {
			t.Errorf("Combination %d has nested 'os.version' variable (should be flat 'version')", i+1)
		}
	}
}

// TestMultiDimensionalMatrix_MixedSimpleAndComplex tests mixing simple and complex dimensions
func TestMultiDimensionalMatrix_MixedSimpleAndComplex(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	// Mix of simple and complex dimensions:
	// platforms: ["linux", "windows"]  # Simple dimension
	// images:                           # Complex dimension
	//   - alpine:
	//       tag: ["3.18", "3.19"]
	matrixValues := map[string][]interface{}{
		"platforms": {"linux", "windows"},
		"images": {
			map[string]interface{}{
				"alpine": map[string]interface{}{
					"tag": []interface{}{"3.18", "3.19"},
				},
			},
		},
	}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: 2 platforms * 2 tags = 4 combinations
	expectedCount := 4
	if len(combinations) != expectedCount {
		t.Errorf("Expected %d combinations, got %d", expectedCount, len(combinations))
		for i, combo := range combinations {
			t.Logf("Combination %d: %v", i+1, combo)
		}
	}

	// Verify all have required variables
	for i, combo := range combinations {
		if _, ok := combo["platforms"]; !ok {
			t.Errorf("Combination %d missing 'platforms' variable", i+1)
		}
		if _, ok := combo["images"]; !ok {
			t.Errorf("Combination %d missing 'images' variable", i+1)
		}
		if _, ok := combo["tag"]; !ok {
			t.Errorf("Combination %d missing 'tag' variable", i+1)
		}
	}
}

// TestMultiDimensionalMatrix_StepNameGeneration tests step name generation with all dimensions
func TestMultiDimensionalMatrix_StepNameGeneration(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "build",
				Run:  "echo build",
			},
		},
	}

	expander := NewMatrixExpander(config)

	matrixValues := map[string][]interface{}{
		"os": {
			map[string]interface{}{
				"ubuntu": map[string]interface{}{
					"version": "20.04",
				},
			},
		},
		"arch": {"amd64"},
	}

	combinations := expander.generateCombinations(matrixValues)

	if len(combinations) != 1 {
		t.Fatalf("Expected 1 combination, got %d", len(combinations))
	}

	// Test step name generation
	originalValues := map[string][]interface{}{
		"os":   matrixValues["os"],
		"arch": {"amd64"},
	}

	stepName := expander.generateStepName("build", combinations[0], originalValues)

	// Step name should include all dimensions in alphabetical order
	// Expected: build.amd64.ubuntu.20.04
	// The exact order depends on sorted keys
	if stepName == "" {
		t.Error("Step name should not be empty")
	}

	// Verify step name contains all components
	expectedParts := []string{"build", "amd64", "ubuntu", "20.04"}
	for _, part := range expectedParts {
		if !containsSubstring(stepName, part) {
			t.Errorf("Step name '%s' should contain '%s'", stepName, part)
		}
	}
}

// TestMultiDimensionalMatrix_SingleValue tests that single values work correctly
func TestMultiDimensionalMatrix_SingleValue(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	// Single value in nested dimension
	matrixValues := map[string][]interface{}{
		"os": {
			map[string]interface{}{
				"centos": map[string]interface{}{
					"version": "7",
				},
			},
		},
	}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: 1 combination
	if len(combinations) != 1 {
		t.Errorf("Expected 1 combination, got %d", len(combinations))
	}

	if len(combinations) > 0 {
		combo := combinations[0]
		if combo["os"] != "centos" {
			t.Errorf("Expected os='centos', got '%v'", combo["os"])
		}
		if combo["version"] != "7" {
			t.Errorf("Expected version='7', got '%v'", combo["version"])
		}
	}
}

// TestMultiDimensionalMatrix_EmptyMatrix tests empty matrix handling
func TestMultiDimensionalMatrix_EmptyMatrix(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	expander := NewMatrixExpander(config)

	matrixValues := map[string][]interface{}{}

	combinations := expander.generateCombinations(matrixValues)

	// Expected: 0 combinations
	if len(combinations) != 0 {
		t.Errorf("Expected 0 combinations for empty matrix, got %d", len(combinations))
	}
}

// TestMultiDimensionalMatrix_DeterministicOrdering tests that combinations are generated in deterministic order
func TestMultiDimensionalMatrix_DeterministicOrdering(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test-action",
				Run:  "echo test",
			},
		},
	}

	matrixValues := map[string][]interface{}{
		"builds": {"Release", "Debug"},
		"arch":   {"amd64", "arm64"},
	}

	// Generate combinations multiple times
	var previousOrder []string
	for i := 0; i < 5; i++ {
		expander := NewMatrixExpander(config)
		combinations := expander.generateCombinations(matrixValues)

		var currentOrder []string
		for _, combo := range combinations {
			key := combo["builds"].(string) + "-" + combo["arch"].(string)
			currentOrder = append(currentOrder, key)
		}

		if i > 0 {
			// Verify order is the same as previous run
			if len(currentOrder) != len(previousOrder) {
				t.Fatalf("Run %d: Different number of combinations", i)
			}
			for j := range currentOrder {
				if currentOrder[j] != previousOrder[j] {
					t.Errorf("Run %d: Order differs at index %d: got %s, want %s",
						i, j, currentOrder[j], previousOrder[j])
				}
			}
		}
		previousOrder = currentOrder
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestExpandMatrixToSteps tests full expansion to steps
func TestExpandMatrixToSteps_MultiDimensional(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "build",
				Run:  "echo 'Building for ${{ matrix.os }} on ${{ matrix.arch }}'",
			},
		},
	}

	expander := NewMatrixExpander(config)

	step := &Step{
		Action: "build",
		Matrix: &MatrixConfig{
			Values: map[string][]interface{}{
				"os": {
					map[string]interface{}{
						"ubuntu": map[string]interface{}{
							"arch": []interface{}{"amd64", "arm64"},
						},
					},
				},
			},
		},
	}

	action := &config.Actions[0]
	steps, err := expander.ExpandMatrixToSteps(step, action)
	if err != nil {
		t.Fatalf("ExpandMatrixToSteps failed: %v", err)
	}

	// Expected: 2 steps (ubuntu-amd64, ubuntu-arm64)
	expectedCount := 2
	if len(steps) != expectedCount {
		t.Errorf("Expected %d steps, got %d", expectedCount, len(steps))
	}

	// Verify step names are unique
	stepNames := make(map[string]bool)
	for _, step := range steps {
		if stepNames[step.Action] {
			t.Errorf("Duplicate step name: %s", step.Action)
		}
		stepNames[step.Action] = true
	}

	// Verify all steps have descriptions
	for i, step := range steps {
		if step.Description == "" {
			t.Errorf("Step %d has empty description", i)
		}
	}
}

// TestExpandMatrixToStepsWithActions tests full expansion with action interpolation
func TestExpandMatrixToStepsWithActions_MultiDimensional(t *testing.T) {
	config := &Config{
		Actions: []Action{
			{
				Name: "test",
				Run:  "echo 'Testing ${{ matrix.suite }} on ${{ matrix.platform }}'",
			},
		},
	}

	globalVars := map[string]string{
		"version.branch": "main",
		"os":             "linux",
	}

	expander := NewMatrixExpander(config, map[string]string{}, globalVars)

	step := &Step{
		Action: "test",
		Matrix: &MatrixConfig{
			Values: map[string][]interface{}{
				"platform": {
					map[string]interface{}{
						"docker": map[string]interface{}{
							"suite": []interface{}{"unit", "integration"},
						},
					},
				},
			},
		},
	}

	action := &config.Actions[0]
	steps, actions, err := expander.ExpandMatrixToStepsWithActions(step, action)
	if err != nil {
		t.Fatalf("ExpandMatrixToStepsWithActions failed: %v", err)
	}

	// Expected: 2 steps
	expectedCount := 2
	if len(steps) != expectedCount {
		t.Errorf("Expected %d steps, got %d", expectedCount, len(steps))
	}

	if len(actions) != expectedCount {
		t.Errorf("Expected %d actions, got %d", expectedCount, len(actions))
	}

	// Verify interpolated actions
	for stepName, action := range actions {
		if action == nil {
			t.Errorf("Action for step %s is nil", stepName)
			continue
		}
		if action.Run == "" {
			t.Errorf("Action for step %s has empty Run command", stepName)
		}
		// Run command should have interpolated matrix variables
		// (actual interpolation tested elsewhere)
	}
}

// Benchmark multi-dimensional matrix expansion
func BenchmarkMultiDimensionalMatrix_Expansion(b *testing.B) {
	config := &Config{
		Actions: []Action{{Name: "test", Run: "echo test"}},
	}

	matrixValues := map[string][]interface{}{
		"images": {
			map[string]interface{}{"centos7": map[string]interface{}{"compiler": "gcc"}},
			map[string]interface{}{"centos8": map[string]interface{}{"compiler": []interface{}{"gcc", "clang"}}},
			map[string]interface{}{"centos9": map[string]interface{}{"compiler": []interface{}{"gcc", "clang", "icc"}}},
		},
		"builds": {"Release", "Debug"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expander := NewMatrixExpander(config)
		_ = expander.generateCombinations(matrixValues)
	}
}

