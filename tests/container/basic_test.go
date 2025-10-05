package container

import (
	"testing"
	
	"github.com/AlexBurnes/buildfab/pkg/buildfab/container"
)

func TestPodmanEngineDetection(t *testing.T) {
	engine := container.NewPodmanEngine()
	if !engine.DetectEngine() {
		t.Skip("Podman not available")
	}
	
	if engine.GetEngineName() != "podman" {
		t.Errorf("Expected engine name 'podman', got '%s'", engine.GetEngineName())
	}
}

func TestDockerEngineDetection(t *testing.T) {
	engine := container.NewDockerEngine()
	if !engine.DetectEngine() {
		t.Skip("Docker not available")
	}
	
	if engine.GetEngineName() != "docker" {
		t.Errorf("Expected engine name 'docker', got '%s'", engine.GetEngineName())
	}
}

func TestManagerCreation(t *testing.T) {
	manager, err := container.NewManager()
	if err != nil {
		t.Skip("No container engine available")
	}
	
	if manager == nil {
		t.Error("Expected manager, got nil")
	}
}
