package container

import (
	"context"
	"testing"
	
	containerRunner "github.com/AlexBurnes/buildfab/internal/container"
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

func TestContainerActionExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping container integration test")
	}
	
	config := container.ContainerConfig{
		Image: container.ContainerImage{
			From: "alpine:latest",
		},
		Run: "echo \"Hello from container\"",
	}
	
	runner, err := containerRunner.NewContainerRunner()
	if err != nil {
		t.Skip("No container engine available")
	}
	
	result, err := runner.RunAction(context.Background(), config, "")
	if err != nil {
		t.Errorf("Container action failed: %v", err)
	}
	if result == nil {
		t.Error("Expected container result, got nil")
	}
}

func TestContainerRunStageExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping container run_stage test")
	}
	
	config := container.ContainerConfig{
		Image: container.ContainerImage{
			From: "alpine:latest",
		},
		RunStage: "test-stage",
	}
	
	runner, err := containerRunner.NewContainerRunner()
	if err != nil {
		t.Skip("No container engine available")
	}
	
	result, err := runner.RunAction(context.Background(), config, "")
	if err != nil {
		t.Errorf("Container run_stage failed: %v", err)
	}
	if result == nil {
		t.Error("Expected container result, got nil")
	}
}
