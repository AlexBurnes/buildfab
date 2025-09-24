package buildfab

import (
	"fmt"
	
	"github.com/AlexBurnes/version-go/pkg/version"
)

// PlatformVariables provides platform detection variables for buildfab
type PlatformVariables struct {
	Platform  string // e.g., "linux", "windows", "darwin"
	Arch      string // e.g., "amd64", "arm64", "386"
	OS        string // e.g., "ubuntu", "windows", "darwin"
	OSVersion string // e.g., "24.04", "windows", "darwin"
	CPU       int    // number of logical CPUs
}

// GetPlatformVariables returns the current platform variables
func GetPlatformVariables() *PlatformVariables {
	info := version.GetPlatformInfo()
	return &PlatformVariables{
		Platform:  info.Platform,
		Arch:      info.Arch,
		OS:        info.OS,
		OSVersion: info.OSVersion,
		CPU:       info.NumCPU,
	}
}

// GetPlatformVariablesMap returns platform variables as a map for variable interpolation
func GetPlatformVariablesMap() map[string]string {
	info := version.GetPlatformInfo()
	return map[string]string{
		"platform":   info.Platform,
		"arch":       info.Arch,
		"os":         info.OS,
		"os_version": info.OSVersion,
		"cpu":        fmt.Sprintf("%d", info.NumCPU),
	}
}

// AddPlatformVariables adds platform variables to the existing variables map
func AddPlatformVariables(variables map[string]string) map[string]string {
	if variables == nil {
		variables = make(map[string]string)
	}
	
	platformVars := GetPlatformVariablesMap()
	for k, v := range platformVars {
		variables[k] = v
	}
	
	return variables
}
