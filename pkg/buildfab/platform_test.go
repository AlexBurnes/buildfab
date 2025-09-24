package buildfab

import (
	"testing"
)

func TestGetPlatformVariables(t *testing.T) {
	vars := GetPlatformVariables()
	
	if vars.Platform == "" {
		t.Error("Platform should not be empty")
	}
	
	if vars.Arch == "" {
		t.Error("Arch should not be empty")
	}
	
	if vars.OS == "" {
		t.Error("OS should not be empty")
	}
	
	if vars.OSVersion == "" {
		t.Error("OSVersion should not be empty")
	}
	
	if vars.CPU <= 0 {
		t.Error("CPU count should be greater than 0")
	}
	
	t.Logf("Platform: %s", vars.Platform)
	t.Logf("Arch: %s", vars.Arch)
	t.Logf("OS: %s", vars.OS)
	t.Logf("OSVersion: %s", vars.OSVersion)
	t.Logf("CPU: %d", vars.CPU)
}

func TestGetPlatformVariablesMap(t *testing.T) {
	vars := GetPlatformVariablesMap()
	
	expectedKeys := []string{"platform", "arch", "os", "os_version", "cpu"}
	for _, key := range expectedKeys {
		if _, exists := vars[key]; !exists {
			t.Errorf("Expected key %s not found in variables map", key)
		}
	}
	
	if vars["platform"] == "" {
		t.Error("platform variable should not be empty")
	}
	
	if vars["arch"] == "" {
		t.Error("arch variable should not be empty")
	}
	
	if vars["os"] == "" {
		t.Error("os variable should not be empty")
	}
	
	if vars["os_version"] == "" {
		t.Error("os_version variable should not be empty")
	}
	
	if vars["cpu"] == "" {
		t.Error("cpu variable should not be empty")
	}
	
	t.Logf("Variables map: %+v", vars)
}

func TestAddPlatformVariables(t *testing.T) {
	// Test with nil map
	vars := AddPlatformVariables(nil)
	if len(vars) != 5 {
		t.Errorf("Expected 5 variables, got %d", len(vars))
	}
	
	// Test with existing map
	existing := map[string]string{"custom": "value"}
	vars = AddPlatformVariables(existing)
	if len(vars) != 6 {
		t.Errorf("Expected 6 variables, got %d", len(vars))
	}
	
	if vars["custom"] != "value" {
		t.Error("Existing variables should be preserved")
	}
	
	// Test with empty map
	empty := make(map[string]string)
	vars = AddPlatformVariables(empty)
	if len(vars) != 5 {
		t.Errorf("Expected 5 variables, got %d", len(vars))
	}
}
