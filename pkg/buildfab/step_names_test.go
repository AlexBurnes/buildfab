package buildfab

import (
    "testing"
)

func TestStepGetStepName(t *testing.T) {
    tests := []struct {
        name     string
        step     Step
        expected string
    }{
        {
            name: "custom name takes priority",
            step: Step{
                Name:   "custom-name",
                Action: "action-name",
            },
            expected: "custom-name",
        },
        {
            name: "action name when no custom name",
            step: Step{
                Action: "action-name",
            },
            expected: "action-name",
        },
        {
            name: "stage name when no action",
            step: Step{
                Stage: "stage-name",
            },
            expected: "stage-name",
        },
        {
            name: "custom name with stage reference",
            step: Step{
                Name:  "custom-name",
                Stage: "stage-name",
            },
            expected: "custom-name",
        },
        {
            name: "empty step",
            step: Step{},
            expected: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.step.GetStepName()
            if result != tt.expected {
                t.Errorf("GetStepName() = %q, want %q", result, tt.expected)
            }
        })
    }
}

func TestValidateDuplicateStepNames(t *testing.T) {
    tests := []struct {
        name        string
        config      Config
        expectError bool
        errorMsg    string
    }{
        {
            name: "unique step names with custom names",
            config: Config{
                Project: Project{Name: "test"},
                Actions: []Action{
                    {Name: "sleep", Run: "sleep 1"},
                },
                Stages: map[string]Stage{
                    "test": {
                        Steps: []Step{
                            {Name: "sleep-1", Action: "sleep"},
                            {Name: "sleep-2", Action: "sleep"},
                        },
                    },
                },
            },
            expectError: false,
        },
        {
            name: "duplicate step names without custom names",
            config: Config{
                Project: Project{Name: "test"},
                Actions: []Action{
                    {Name: "sleep", Run: "sleep 1"},
                },
                Stages: map[string]Stage{
                    "test": {
                        Steps: []Step{
                            {Action: "sleep"},
                            {Action: "sleep"}, // Duplicate!
                        },
                    },
                },
            },
            expectError: true,
            errorMsg:    "duplicate step name 'sleep'",
        },
        {
            name: "no duplicates with different actions",
            config: Config{
                Project: Project{Name: "test"},
                Actions: []Action{
                    {Name: "action1", Run: "echo 1"},
                    {Name: "action2", Run: "echo 2"},
                },
                Stages: map[string]Stage{
                    "test": {
                        Steps: []Step{
                            {Action: "action1"},
                            {Action: "action2"},
                        },
                    },
                },
            },
            expectError: false,
        },
        {
            name: "duplicate with same custom name",
            config: Config{
                Project: Project{Name: "test"},
                Actions: []Action{
                    {Name: "action1", Run: "echo 1"},
                    {Name: "action2", Run: "echo 2"},
                },
                Stages: map[string]Stage{
                    "test": {
                        Steps: []Step{
                            {Name: "same-name", Action: "action1"},
                            {Name: "same-name", Action: "action2"}, // Duplicate custom name!
                        },
                    },
                },
            },
            expectError: true,
            errorMsg:    "duplicate step name 'same-name'",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            
            if tt.expectError {
                if err == nil {
                    t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
                } else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
                    t.Errorf("Validate() error = %q, expected to contain %q", err.Error(), tt.errorMsg)
                }
            } else {
                if err != nil {
                    t.Errorf("Validate() unexpected error: %v", err)
                }
            }
        })
    }
}

func TestStepNameDependencies(t *testing.T) {
    config := Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "sleep", Run: "sleep 1"},
            {Name: "echo", Run: "echo test"},
        },
        Stages: map[string]Stage{
            "test": {
                Steps: []Step{
                    {Name: "step1", Action: "sleep"},
                    {Name: "step2", Action: "echo", Require: []string{"step1"}},
                    {Name: "step3", Action: "sleep", Require: []string{"step2"}},
                },
            },
        },
    }

    err := config.Validate()
    if err != nil {
        t.Errorf("Validate() unexpected error: %v", err)
    }
}

func TestStepNameInvalidDependency(t *testing.T) {
    config := Config{
        Project: Project{Name: "test"},
        Actions: []Action{
            {Name: "sleep", Run: "sleep 1"},
        },
        Stages: map[string]Stage{
            "test": {
                Steps: []Step{
                    {Name: "step1", Action: "sleep"},
                    {Name: "step2", Action: "sleep", Require: []string{"nonexistent"}},
                },
            },
        },
    }

    err := config.Validate()
    if err == nil {
        t.Error("Validate() expected error for invalid dependency, got nil")
    } else if !contains(err.Error(), "invalid dependency") && !contains(err.Error(), "not found") {
        t.Errorf("Validate() error = %q, expected to contain 'invalid dependency' or 'not found'", err.Error())
    }
}

// Note: contains() helper function is defined in config_test.go

