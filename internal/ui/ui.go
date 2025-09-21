// Package ui provides user interface functionality for buildfab
package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// UI provides user interface operations
type UI struct {
	verbose bool
	debug   bool
}

// New creates a new UI instance
func New(verbose, debug bool) *UI {
	return &UI{
		verbose: verbose,
		debug:   debug,
	}
}

// PrintCLIHeader prints the CLI header
func (u *UI) PrintCLIHeader(name, version string) {
	// Handle version that already has 'v' prefix
	versionDisplay := version
	if !strings.HasPrefix(version, "v") {
		versionDisplay = "v" + version
	}
	fmt.Fprintf(os.Stderr, "🚀 %s %s\n", name, versionDisplay)
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// PrintProjectCheck prints project information
func (u *UI) PrintProjectCheck(projectName, version string) {
	fmt.Fprintf(os.Stderr, "📦 Project: %s\n", projectName)
	fmt.Fprintf(os.Stderr, "🏷️  Version: %s\n", version)
	fmt.Fprintf(os.Stderr, "\n")
}

// PrintStageHeader prints stage header
func (u *UI) PrintStageHeader(stageName string) {
	fmt.Fprintf(os.Stderr, "▶️  Running stage: %s\n", stageName)
	fmt.Fprintf(os.Stderr, "\n")
}

// PrintStepStatus prints step status
func (u *UI) PrintStepStatus(stepName string, status buildfab.Status, message string) {
	var icon string
	var color string
	
	switch status {
	case buildfab.StatusOK:
		icon = "✓"
		color = "\033[32m" // Green
	case buildfab.StatusWarn:
		icon = "!"
		color = "\033[33m" // Yellow
	case buildfab.StatusError:
		icon = "✗"
		color = "\033[31m" // Red
	case buildfab.StatusSkipped:
		icon = "→"
		color = "\033[90m" // Gray
	case buildfab.StatusRunning:
		icon = "○"
		color = "\033[36m" // Cyan
	default:
		icon = "?"
		color = "\033[37m" // White
	}
	
	reset := "\033[0m"
	
	// Handle multi-line messages properly
	lines := strings.Split(message, "\n")
	if len(lines) == 1 {
		// Single line message
		fmt.Fprintf(os.Stderr, "  %s%s%s %s %s\n", color, icon, reset, stepName, message)
	} else {
		// Multi-line message - first line with step name, subsequent lines indented
		fmt.Fprintf(os.Stderr, "  %s%s%s %s %s\n", color, icon, reset, stepName, lines[0])
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) != "" {
				// Indent subsequent lines with simple spacing
				fmt.Fprintf(os.Stderr, "    %s\n", line)
			}
		}
	}
}

// PrintStageResult prints stage result
func (u *UI) PrintStageResult(stageName string, success bool, duration time.Duration) {
	fmt.Fprintf(os.Stderr, "\n")
	
	var icon string
	var color string
	var status string
	
	if success {
		icon = "🎉"
		color = "\033[32m" // Green
		status = "SUCCESS"
	} else {
		icon = "💥"
		color = "\033[31m" // Red
		status = "FAILED"
	}
	
	reset := "\033[0m"
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(os.Stderr, "%s %s%s%s - %s (%.2fs)\n", icon, color, status, reset, stageName, duration.Seconds())
}

// PrintCommand prints a command being executed
func (u *UI) PrintCommand(command string) {
	if u.verbose {
		fmt.Fprintf(os.Stderr, "  💻 %s\n", command)
	}
}

// PrintCommandOutput prints command output
func (u *UI) PrintCommandOutput(output string) {
	if u.verbose && output != "" {
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		for _, line := range lines {
			fmt.Fprintf(os.Stderr, "    %s\n", line)
		}
	}
}

// PrintRepro prints reproduction instructions
func (u *UI) PrintRepro(stepName, repro string) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "🔧 To reproduce %s:\n", stepName)
	
	// Handle multi-line reproduction instructions
	lines := strings.Split(strings.TrimRight(repro, "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			// Preserve the original indentation structure
			fmt.Fprintf(os.Stderr, "%s\n", line)
		}
	}
}

// PrintReproInline prints inline reproduction instructions
func (u *UI) PrintReproInline(stepName, repro string) {
	fmt.Fprintf(os.Stderr, "   💡 %s\n", repro)
}

// PrintSummary prints execution summary
func (u *UI) PrintSummary(results []buildfab.Result) {
	if len(results) == 0 {
		return
	}
	
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "📊 Summary:\n")
	
	statusCounts := make(map[buildfab.Status]int)
	for _, result := range results {
		statusCounts[result.Status]++
	}
	
	// Define status order for consistent display
	statusOrder := []buildfab.Status{
		buildfab.StatusError,
		buildfab.StatusWarn,
		buildfab.StatusOK,
		buildfab.StatusSkipped,
	}
	
	for _, status := range statusOrder {
		count := statusCounts[status]
		var icon string
		var color string
		
		switch status {
		case buildfab.StatusOK:
			icon = "✓"
			if count > 0 {
				color = "\033[32m" // Green
			} else {
				color = "\033[90m" // Gray
			}
		case buildfab.StatusWarn:
			icon = "!"
			if count > 0 {
				color = "\033[33m" // Yellow
			} else {
				color = "\033[90m" // Gray
			}
		case buildfab.StatusError:
			icon = "✗"
			if count > 0 {
				color = "\033[31m" // Red
			} else {
				color = "\033[90m" // Gray
			}
		case buildfab.StatusSkipped:
			icon = "→"
			if count > 0 {
				color = "\033[90m" // Gray
			} else {
				color = "\033[90m" // Gray
			}
		default:
			icon = "?"
			if count > 0 {
				color = "\033[37m" // White
			} else {
				color = "\033[90m" // Gray
			}
		}
		
		reset := "\033[0m"
		fmt.Fprintf(os.Stderr, "   %s%s%s %s%-8s %3d%s\n", color, icon, reset, color, status.String(), count, reset)
	}
}

// IsVerbose returns true if verbose mode is enabled
func (u *UI) IsVerbose() bool {
	return u.verbose
}

// IsDebug returns true if debug mode is enabled
func (u *UI) IsDebug() bool {
	return u.debug
}