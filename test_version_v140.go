package main

import (
	"fmt"
	"github.com/AlexBurnes/version-go/pkg/version"
)

func main() {
	// Test new functions
	fmt.Println("=== Testing new version-go v1.4.0 functions ===")
	
	// Test GetBuildType
	if buildType, err := version.GetBuildType("v1.2.3"); err == nil {
		fmt.Printf("Build Type: %s\n", buildType)
	} else {
		fmt.Printf("Build Type Error: %v\n", err)
	}
	
	// Test GetVersionType
	if versionType, err := version.GetVersionType("v1.2.3-alpha.1"); err == nil {
		fmt.Printf("Version Type: %s\n", versionType)
	} else {
		fmt.Printf("Version Type Error: %v\n", err)
	}
	
	// Test GetVersionType with empty string (from git)
	if versionType, err := version.GetVersionType(""); err == nil {
		fmt.Printf("Version Type (from git): %s\n", versionType)
	} else {
		fmt.Printf("Version Type (from git) Error: %v\n", err)
	}
	
	// Test GetProjectConfigFromFile
	if config, err := version.GetProjectConfigFromFile(".project.yml"); err == nil {
		fmt.Printf("Project Config: %+v\n", config)
	} else {
		fmt.Printf("Project Config Error: %v\n", err)
	}
	
	// Test existing platform info
	fmt.Println("\n=== Existing Platform Info ===")
	info := version.GetPlatformInfo()
	fmt.Printf("Platform: %s\n", info.Platform)
	fmt.Printf("Arch: %s\n", info.Arch)
	fmt.Printf("OS: %s\n", info.OS)
	fmt.Printf("OS Version: %s\n", info.OSVersion)
	fmt.Printf("CPU: %d\n", info.NumCPU)
}
