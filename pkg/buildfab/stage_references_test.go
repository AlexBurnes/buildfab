package buildfab

import (
    "testing"
)

// TestStageReferences tests that steps can reference stages
func TestStageReferences(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
            {Name: "action-b", Run: "echo b"},
            {Name: "action-c", Run: "echo c"},
        },
        Stages: map[string]Stage{
            "sub-stage": {
                Steps: []Step{
                    {Action: "action-a"},
                    {Action: "action-b"},
                },
            },
            "main-stage": {
                Steps: []Step{
                    {Stage: "sub-stage"},
                    {Action: "action-c"},
                },
            },
        },
    }

    // Validation should pass
    err := config.Validate()
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }

    // Test stage expansion
    opts := DefaultRunOptions()
    opts.DryRun = true
    runner := NewRunner(config, opts)

    mainStage, _ := config.GetStage("main-stage")
    expandedSteps, err := runner.expandStageReferences(mainStage.Steps)
    if err != nil {
        t.Fatalf("Failed to expand stage references: %v", err)
    }

    // Should have 3 steps: action-a, action-b, action-c
    if len(expandedSteps) != 3 {
        t.Errorf("Expected 3 expanded steps, got %d", len(expandedSteps))
    }

    expectedActions := []string{"action-a", "action-b", "action-c"}
    for i, step := range expandedSteps {
        if step.Action != expectedActions[i] {
            t.Errorf("Step %d: expected action %s, got %s", i, expectedActions[i], step.Action)
        }
    }
}

// TestNestedStageReferences tests nested stage references
func TestNestedStageReferences(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
            {Name: "action-b", Run: "echo b"},
            {Name: "action-c", Run: "echo c"},
        },
        Stages: map[string]Stage{
            "level-3": {
                Steps: []Step{
                    {Action: "action-a"},
                },
            },
            "level-2": {
                Steps: []Step{
                    {Stage: "level-3"},
                    {Action: "action-b"},
                },
            },
            "level-1": {
                Steps: []Step{
                    {Stage: "level-2"},
                    {Action: "action-c"},
                },
            },
        },
    }

    // Validation should pass
    err := config.Validate()
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }

    // Test stage expansion
    opts := DefaultRunOptions()
    opts.DryRun = true
    runner := NewRunner(config, opts)

    level1Stage, _ := config.GetStage("level-1")
    expandedSteps, err := runner.expandStageReferences(level1Stage.Steps)
    if err != nil {
        t.Fatalf("Failed to expand nested stage references: %v", err)
    }

    // Should have 3 steps: action-a (from level-3), action-b (from level-2), action-c (from level-1)
    if len(expandedSteps) != 3 {
        t.Errorf("Expected 3 expanded steps, got %d", len(expandedSteps))
    }

    expectedActions := []string{"action-a", "action-b", "action-c"}
    for i, step := range expandedSteps {
        if step.Action != expectedActions[i] {
            t.Errorf("Step %d: expected action %s, got %s", i, expectedActions[i], step.Action)
        }
    }
}

// TestStageReferenceSelfReference tests that self-referencing stages are rejected
func TestStageReferenceSelfReference(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
        },
        Stages: map[string]Stage{
            "recursive-stage": {
                Steps: []Step{
                    {Stage: "recursive-stage"}, // Self-reference
                    {Action: "action-a"},
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for self-referencing stage")
    }

    if !contains(err.Error(), "cannot reference itself") {
        t.Errorf("Expected error about self-reference, got: %v", err)
    }
}

// TestStageReferenceUnknownStage tests that referencing unknown stages is rejected
func TestStageReferenceUnknownStage(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
        },
        Stages: map[string]Stage{
            "main-stage": {
                Steps: []Step{
                    {Stage: "unknown-stage"}, // Unknown stage
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for unknown stage reference")
    }

    if !contains(err.Error(), "unknown stage") {
        t.Errorf("Expected error about unknown stage, got: %v", err)
    }
}

// TestStepWithBothActionAndStage tests that steps with both action and stage are rejected
func TestStepWithBothActionAndStage(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
        },
        Stages: map[string]Stage{
            "sub-stage": {
                Steps: []Step{
                    {Action: "action-a"},
                },
            },
            "main-stage": {
                Steps: []Step{
                    {Action: "action-a", Stage: "sub-stage"}, // Both action and stage
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for step with both action and stage")
    }

    if !contains(err.Error(), "cannot have both 'action' and 'stage'") {
        t.Errorf("Expected error about both action and stage, got: %v", err)
    }
}

// TestStepWithNeitherActionNorStage tests that steps without action or stage are rejected
func TestStepWithNeitherActionNorStage(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
        },
        Stages: map[string]Stage{
            "main-stage": {
                Steps: []Step{
                    {Description: "empty step"}, // Neither action nor stage
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for step without action or stage")
    }

    if !contains(err.Error(), "must have either 'action' or 'stage'") {
        t.Errorf("Expected error about missing action or stage, got: %v", err)
    }
}

// TestStageReferenceWithVariables tests variable inheritance from stage references
func TestStageReferenceWithVariables(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
            {Name: "action-b", Run: "echo b"},
        },
        Stages: map[string]Stage{
            "sub-stage": {
                Steps: []Step{
                    {Action: "action-a", Variables: map[string]string{"var1": "child"}},
                    {Action: "action-b"},
                },
            },
            "main-stage": {
                Steps: []Step{
                    {
                        Stage: "sub-stage",
                        Variables: map[string]string{
                            "var1": "parent",
                            "var2": "parent-only",
                        },
                    },
                },
            },
        },
    }

    // Validation should pass
    err := config.Validate()
    if err != nil {
        t.Fatalf("Validation failed: %v", err)
    }

    // Test stage expansion
    opts := DefaultRunOptions()
    opts.DryRun = true
    runner := NewRunner(config, opts)

    mainStage, _ := config.GetStage("main-stage")
    expandedSteps, err := runner.expandStageReferences(mainStage.Steps)
    if err != nil {
        t.Fatalf("Failed to expand stage references: %v", err)
    }

    // Should have 2 steps
    if len(expandedSteps) != 2 {
        t.Errorf("Expected 2 expanded steps, got %d", len(expandedSteps))
    }

    // First step (action-a) should have child's var1 (not overwritten by parent)
    if expandedSteps[0].Variables["var1"] != "child" {
        t.Errorf("Expected var1='child' in step 0, got %s", expandedSteps[0].Variables["var1"])
    }

    // First step should inherit parent's var2
    if expandedSteps[0].Variables["var2"] != "parent-only" {
        t.Errorf("Expected var2='parent-only' in step 0, got %s", expandedSteps[0].Variables["var2"])
    }

    // Second step (action-b) should have both parent variables
    if expandedSteps[1].Variables["var1"] != "parent" {
        t.Errorf("Expected var1='parent' in step 1, got %s", expandedSteps[1].Variables["var1"])
    }
    if expandedSteps[1].Variables["var2"] != "parent-only" {
        t.Errorf("Expected var2='parent-only' in step 1, got %s", expandedSteps[1].Variables["var2"])
    }
}

// TestCycleDetectionInDependencies tests that circular dependencies are detected
func TestCycleDetectionInDependencies(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
            {Name: "action-b", Run: "echo b"},
            {Name: "action-c", Run: "echo c"},
        },
        Stages: map[string]Stage{
            "cycle-stage": {
                Steps: []Step{
                    {Action: "action-a", Require: []string{"action-b"}},
                    {Action: "action-b", Require: []string{"action-c"}},
                    {Action: "action-c", Require: []string{"action-a"}}, // Creates cycle
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for circular dependencies")
    }

    depErr, ok := err.(*DependencyError)
    if !ok {
        t.Errorf("Expected DependencyError, got %T: %v", err, err)
    } else {
        if len(depErr.Cycle) == 0 {
            t.Error("Expected cycle path in error, got empty")
        }
    }
}

// TestCycleDetectionInStageReferences tests that circular stage references are detected
func TestCycleDetectionInStageReferences(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
        },
        Stages: map[string]Stage{
            "stage-a": {
                Steps: []Step{
                    {Stage: "stage-b"},
                },
            },
            "stage-b": {
                Steps: []Step{
                    {Stage: "stage-c"},
                },
            },
            "stage-c": {
                Steps: []Step{
                    {Stage: "stage-a"}, // Creates cycle
                },
            },
        },
    }

    // Validation should fail
    err := config.Validate()
    if err == nil {
        t.Fatal("Expected validation to fail for circular stage references")
    }

    depErr, ok := err.(*DependencyError)
    if !ok {
        t.Errorf("Expected DependencyError, got %T: %v", err, err)
    } else {
        if len(depErr.Cycle) == 0 {
            t.Error("Expected cycle path in error, got empty")
        }
    }
}

// TestValidDependencyGraph tests that valid dependency graphs pass validation
func TestValidDependencyGraph(t *testing.T) {
    config := &Config{
        Project: Project{
            Name: "test-project",
        },
        Actions: []Action{
            {Name: "action-a", Run: "echo a"},
            {Name: "action-b", Run: "echo b"},
            {Name: "action-c", Run: "echo c"},
            {Name: "action-d", Run: "echo d"},
        },
        Stages: map[string]Stage{
            "valid-stage": {
                Steps: []Step{
                    {Action: "action-a"},
                    {Action: "action-b", Require: []string{"action-a"}},
                    {Action: "action-c", Require: []string{"action-a"}},
                    {Action: "action-d", Require: []string{"action-b", "action-c"}},
                },
            },
        },
    }

    // Validation should pass
    err := config.Validate()
    if err != nil {
        t.Fatalf("Validation failed for valid dependency graph: %v", err)
    }
}

// Note: contains() helper function is defined in config_test.go

