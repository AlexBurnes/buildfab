package container

import (
	"context"
	"testing"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

func TestSimpleContainerExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping simple container test")
	}
	
	// Test with a simple echo command
	config := container.ContainerConfig{
		Image: container.ContainerImage{
			From: "alpine:latest",
		},
		Commands: []string{"echo", "Hello from container"},
	}
	
	manager, err := container.NewManager()
	if err != nil {
		t.Skip("No container engine available")
	}
	
	result, err := manager.ExecuteAction(context.Background(), config)
	if err != nil {
		t.Errorf("Container execution failed: %v", err)
		return
	}
	
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Output: %s, Error: %s", result.ExitCode, result.Output, result.Error)
	}
	
	t.Logf("Container output: %s", result.Output)
}
