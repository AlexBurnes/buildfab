package version

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	detector := New()
	if detector == nil {
		t.Error("New() should return a non-nil detector")
	}
}

func TestDetector_DetectCurrentVersion(t *testing.T) {
	detector := New()
	
	// Create a temporary directory for test
	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with no VERSION file and no git
	version, err := detector.DetectCurrentVersion(ctx)
	if err != nil {
		t.Errorf("DetectCurrentVersion() unexpected error: %v", err)
	}
	if version != "unknown" {
		t.Errorf("DetectCurrentVersion() = %v, want %v", version, "unknown")
	}

	// Test with VERSION file
	err = os.WriteFile("VERSION", []byte("v1.2.3"), 0644)
	if err != nil {
		t.Fatalf("Failed to write VERSION file: %v", err)
	}

	version, err = detector.DetectCurrentVersion(ctx)
	if err != nil {
		t.Errorf("DetectCurrentVersion() unexpected error: %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("DetectCurrentVersion() = %v, want %v", version, "v1.2.3")
	}
}

func TestDetector_DetectVersionInfo(t *testing.T) {
	detector := New()
	
	// Create a temporary directory for test
	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with VERSION file
	err = os.WriteFile("VERSION", []byte("v1.2.3"), 0644)
	if err != nil {
		t.Fatalf("Failed to write VERSION file: %v", err)
	}

	versionInfo, err := detector.DetectVersionInfo(ctx)
	if err != nil {
		t.Errorf("DetectVersionInfo() unexpected error: %v", err)
	}
	if versionInfo.Version != "v1.2.3" {
		t.Errorf("Version = %v, want %v", versionInfo.Version, "v1.2.3")
	}
}

func TestDetector_GetVersionVariables(t *testing.T) {
	detector := New()
	
	// Create a temporary directory for test
	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with VERSION file
	err = os.WriteFile("VERSION", []byte("v1.2.3"), 0644)
	if err != nil {
		t.Fatalf("Failed to write VERSION file: %v", err)
	}

	variables, err := detector.GetVersionVariables(ctx)
	if err != nil {
		t.Errorf("GetVersionVariables() unexpected error: %v", err)
	}

	// Check that version variables are present
	expectedKeys := []string{
		"version.version",
		"version.project",
		"version.commit",
		"version.date",
		"version.type",
		"version.major",
		"version.minor",
		"version.patch",
	}

	for _, key := range expectedKeys {
		if _, exists := variables[key]; !exists {
			t.Errorf("Version variables should contain key: %s", key)
		}
	}

	// Check specific values
	if variables["version.version"] != "v1.2.3" {
		t.Errorf("version.version = %v, want %v", variables["version.version"], "v1.2.3")
	}
	if variables["version.major"] != "1" {
		t.Errorf("version.major = %v, want %v", variables["version.major"], "1")
	}
	if variables["version.minor"] != "2" {
		t.Errorf("version.minor = %v, want %v", variables["version.minor"], "2")
	}
	if variables["version.patch"] != "3" {
		t.Errorf("version.patch = %v, want %v", variables["version.patch"], "3")
	}
}

func TestDetector_Integration(t *testing.T) {
	// This test requires a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository, skipping integration test")
	}

	detector := New()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test version detection in real git repository
	version, err := detector.DetectCurrentVersion(ctx)
	if err != nil {
		t.Errorf("DetectCurrentVersion() unexpected error: %v", err)
	}
	t.Logf("Detected version: %s", version)

	// Test version info detection
	versionInfo, err := detector.DetectVersionInfo(ctx)
	if err != nil {
		t.Errorf("DetectVersionInfo() unexpected error: %v", err)
	}
	t.Logf("Version info: %+v", versionInfo)

	// Test version variables
	variables, err := detector.GetVersionVariables(ctx)
	if err != nil {
		t.Errorf("GetVersionVariables() unexpected error: %v", err)
	}
	t.Logf("Version variables: %+v", variables)
}