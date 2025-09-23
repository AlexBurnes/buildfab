package buildfab

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DefaultActionRegistry provides a default implementation of ActionRegistry
// that includes common built-in actions
type DefaultActionRegistry struct {
	actions map[string]ActionRunner
}

// NewDefaultActionRegistry creates a new default action registry with built-in actions
func NewDefaultActionRegistry() *DefaultActionRegistry {
	registry := &DefaultActionRegistry{
		actions: make(map[string]ActionRunner),
	}
	
	// Register built-in actions
	registry.Register("git@untracked", &GitUntrackedAction{})
	registry.Register("git@uncommitted", &GitUncommittedAction{})
	registry.Register("git@modified", &GitModifiedAction{})
	registry.Register("version@check", &VersionCheckAction{})
	registry.Register("version@check-greatest", &VersionCheckGreatestAction{})
	
	return registry
}

// Register registers a built-in action
func (r *DefaultActionRegistry) Register(name string, runner ActionRunner) {
	r.actions[name] = runner
}

// GetRunner returns the runner for a built-in action
func (r *DefaultActionRegistry) GetRunner(name string) (ActionRunner, bool) {
	runner, exists := r.actions[name]
	return runner, exists
}

// ListActions returns all available built-in actions
func (r *DefaultActionRegistry) ListActions() map[string]string {
	actions := make(map[string]string)
	for name, runner := range r.actions {
		actions[name] = runner.Description()
	}
	return actions
}

// GitUntrackedAction checks for untracked files
type GitUntrackedAction struct{}

func (a *GitUntrackedAction) Run(ctx context.Context) (Result, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "Failed to check git status",
		}, fmt.Errorf("git status failed: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	untracked := []string{}
	
	for _, line := range lines {
		if len(line) >= 2 && line[:2] == "??" {
			untracked = append(untracked, strings.TrimSpace(line[2:]))
		}
	}
	
	if len(untracked) > 0 {
		return Result{
			Status:  StatusWarn,
			Message: "Untracked files found, to check run:\n    git status",
		}, nil // Return nil error for warning status
	}
	
	return Result{
		Status:  StatusOK,
		Message: "No untracked files found",
	}, nil
}

func (a *GitUntrackedAction) Description() string {
	return "Check for untracked files"
}

// GitUncommittedAction checks for uncommitted changes
type GitUncommittedAction struct{}

func (a *GitUncommittedAction) Run(ctx context.Context) (Result, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "Failed to check git status",
		}, fmt.Errorf("git status failed: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	uncommitted := []string{}
	
	for _, line := range lines {
		if len(line) >= 2 && (line[:2] == "M " || line[:2] == "A " || line[:2] == "D " || line[:2] == "R " || line[:2] == "C ") {
			uncommitted = append(uncommitted, strings.TrimSpace(line[2:]))
		}
	}
	
	if len(uncommitted) > 0 {
		return Result{
			Status:  StatusWarn,
			Message: "Uncommitted changes found, to check run:\n    git status",
		}, nil // Return nil error for warning status
	}
	
	return Result{
		Status:  StatusOK,
		Message: "No uncommitted changes found",
	}, nil
}

func (a *GitUncommittedAction) Description() string {
	return "Check for uncommitted changes"
}

// GitModifiedAction checks for modified files
type GitModifiedAction struct{}

func (a *GitModifiedAction) Run(ctx context.Context) (Result, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "Failed to check git diff",
		}, fmt.Errorf("git diff failed: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	modified := []string{}
	
	for _, line := range lines {
		if line != "" {
			modified = append(modified, line)
		}
	}
	
	if len(modified) > 0 {
		return Result{
			Status:  StatusWarn,
			Message: "There are modified files, to check run:\n    git status",
		}, nil // Return nil error for warning status
	}
	
	return Result{
		Status:  StatusOK,
		Message: "No modified files found",
	}, nil
}

func (a *GitModifiedAction) Description() string {
	return "Check for modified files"
}

// VersionCheckAction validates version format
type VersionCheckAction struct{}

func (a *VersionCheckAction) Run(ctx context.Context) (Result, error) {
	// Read VERSION file
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "VERSION file not found",
		}, fmt.Errorf("VERSION file not found: %w", err)
	}
	
	version := strings.TrimSpace(string(data))
	if version == "" {
		return Result{
			Status:  StatusError,
			Message: "VERSION file is empty",
		}, fmt.Errorf("VERSION file is empty")
	}
	
	// Basic version format validation
	if !isValidVersion(version) {
		return Result{
			Status:  StatusError,
			Message: fmt.Sprintf("Invalid version format: %s", version),
		}, fmt.Errorf("invalid version format: %s", version)
	}
	
	return Result{
		Status:  StatusOK,
		Message: fmt.Sprintf("Version format is valid: %s", version),
	}, nil
}

func (a *VersionCheckAction) Description() string {
	return "Validate version format"
}

// VersionCheckGreatestAction checks if current version is the greatest
type VersionCheckGreatestAction struct{}

func (a *VersionCheckGreatestAction) Run(ctx context.Context) (Result, error) {
	// Read current version
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "VERSION file not found",
		}, fmt.Errorf("VERSION file not found: %w", err)
	}
	
	currentVersion := strings.TrimSpace(string(data))
	
	// Get all git tags
	cmd := exec.CommandContext(ctx, "git", "tag", "--sort=-version:refname")
	output, err := cmd.Output()
	if err != nil {
		return Result{
			Status:  StatusError,
			Message: "Failed to get git tags",
		}, fmt.Errorf("git tag failed: %w", err)
	}
	
	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		return Result{
			Status:  StatusOK,
			Message: "No tags found, current version is greatest",
		}, nil
	}
	
	// Find the greatest tag
	greatestTag := tags[0]
	
	if currentVersion == greatestTag {
		return Result{
			Status:  StatusOK,
			Message: fmt.Sprintf("Current version %s is the greatest", currentVersion),
		}, nil
	}
	
	return Result{
		Status:  StatusError,
		Message: fmt.Sprintf("Current version %s is not the greatest. Greatest is %s", currentVersion, greatestTag),
	}, fmt.Errorf("current version is not the greatest")
}

func (a *VersionCheckGreatestAction) Description() string {
	return "Check if current version is the greatest"
}

// isValidVersion performs basic version format validation
func isValidVersion(version string) bool {
	// Basic validation: should start with v and contain numbers
	if !strings.HasPrefix(version, "v") {
		return false
	}
	
	// Remove v prefix
	version = version[1:]
	
	// Should contain at least one dot
	if !strings.Contains(version, ".") {
		return false
	}
	
	// Split by dots and check each part
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}
	
	// Each part should be numeric or contain valid prerelease identifiers
	for _, part := range parts {
		if part == "" {
			return false
		}
		// Allow numeric parts and prerelease identifiers
		if !isValidVersionPart(part) {
			return false
		}
	}
	
	return true
}

// isValidVersionPart validates a version part
func isValidVersionPart(part string) bool {
	// Allow numeric parts
	if isNumeric(part) {
		return true
	}
	
	// Allow prerelease identifiers (alpha, beta, rc, etc.)
	lower := strings.ToLower(part)
	if strings.Contains(lower, "alpha") || strings.Contains(lower, "beta") || 
	   strings.Contains(lower, "rc") || strings.Contains(lower, "dev") {
		return true
	}
	
	return false
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}