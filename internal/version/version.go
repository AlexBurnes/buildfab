// Package version provides version detection and variable integration
package version

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlexBurnes/version-go/pkg/version"
)

// Detector handles version detection and provides version variables
type Detector struct{}

// New creates a new version detector
func New() *Detector {
	return &Detector{}
}

// VersionInfo represents version information
type VersionInfo struct {
	Version string
	Project string
	Commit  string
	Date    string
	Type    string // release, prerelease, patch, minor, major
	Major   int
	Minor   int
	Patch   int
}

// DetectCurrentVersion detects the current version from various sources
func (d *Detector) DetectCurrentVersion(ctx context.Context) (string, error) {
	// Try to read from VERSION file first
	if version, err := d.readVersionFile(); err == nil {
		return version, nil
	}
	
	// Fallback to git tag detection
	if version, err := d.detectGitTag(ctx); err == nil {
		return version, nil
	}
	
	// Final fallback
	return "unknown", nil
}

// DetectVersionInfo detects comprehensive version information
func (d *Detector) DetectVersionInfo(ctx context.Context) (*VersionInfo, error) {
	info := &VersionInfo{}
	
	// Detect version
	versionStr, err := d.DetectCurrentVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect version: %w", err)
	}
	info.Version = versionStr
	
	// Parse version using version-go library
	parsedVersion, err := version.Parse(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	
	info.Major = parsedVersion.Major
	info.Minor = parsedVersion.Minor
	info.Patch = parsedVersion.Patch
	info.Type = parsedVersion.Type.String()
	
	// Detect project name (from go.mod or directory)
	project, err := d.detectProjectName(ctx)
	if err == nil {
		info.Project = project
	}
	
	// Detect commit hash
	commit, err := d.detectGitCommit(ctx)
	if err == nil {
		info.Commit = commit
	}
	
	// Detect build date
	info.Date = d.detectBuildDate()
	
	return info, nil
}

// GetVersionVariables returns version variables for interpolation
func (d *Detector) GetVersionVariables(ctx context.Context) (map[string]string, error) {
	info, err := d.DetectVersionInfo(ctx)
	if err != nil {
		return nil, err
	}
	
	variables := map[string]string{
		"version.version": info.Version,
		"version.project": info.Project,
		"version.commit":  info.Commit,
		"version.date":    info.Date,
		"version.type":    info.Type,
		"version.major":   fmt.Sprintf("%d", info.Major),
		"version.minor":   fmt.Sprintf("%d", info.Minor),
		"version.patch":   fmt.Sprintf("%d", info.Patch),
	}
	
	return variables, nil
}

// readVersionFile reads the version from the VERSION file
func (d *Detector) readVersionFile() (string, error) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "", err
	}
	
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}
	
	return version, nil
}

// detectGitTag detects the current Git tag
func (d *Detector) detectGitTag(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// detectProjectName detects the project name
func (d *Detector) detectProjectName(ctx context.Context) (string, error) {
	// Try to read from go.mod
	cmd := exec.CommandContext(ctx, "go", "list", "-m")
	output, err := cmd.Output()
	if err == nil {
		module := strings.TrimSpace(string(output))
		// Extract project name from module path
		parts := strings.Split(module, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}
	
	// Fallback to current directory name
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	parts := strings.Split(wd, string(os.PathSeparator))
	if len(parts) > 0 {
		return parts[len(parts)-1], nil
	}
	
	return "unknown", nil
}

// detectGitCommit detects the current Git commit hash
func (d *Detector) detectGitCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// detectBuildDate detects the build date
func (d *Detector) detectBuildDate() string {
	// For now, return current time
	// In a real implementation, this would be set at build time
	return "unknown"
}

// determineVersionType determines the type of version using version-go library
func (d *Detector) determineVersionType(versionStr string) string {
	parsedVersion, err := version.Parse(versionStr)
	if err != nil {
		return "invalid"
	}
	return parsedVersion.Type.String()
}