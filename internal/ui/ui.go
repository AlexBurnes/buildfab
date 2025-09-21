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
	fmt.Fprintf(os.Stderr, "🚀 %s v%s\n", name, version)
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
		icon = "✅"
		color = "\033[32m" // Green
	case buildfab.StatusWarn:
		icon = "⚠️"
		color = "\033[33m" // Yellow
	case buildfab.StatusError:
		icon = "❌"
		color = "\033[31m" // Red
	case buildfab.StatusSkipped:
		icon = "⏭️"
		color = "\033[90m" // Gray
	case buildfab.StatusRunning:
		icon = "🔄"
		color = "\033[36m" // Cyan
	default:
		icon = "❓"
		color = "\033[37m" // White
	}
	
	reset := "\033[0m"
	fmt.Fprintf(os.Stderr, "  %s %s%-20s%s %s\n", icon, color, stepName, reset, message)
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
	fmt.Fprintf(os.Stderr, "   %s\n", repro)
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
	
	for status, count := range statusCounts {
		if count > 0 {
			var icon string
			switch status {
			case buildfab.StatusOK:
				icon = "✅"
			case buildfab.StatusWarn:
				icon = "⚠️"
			case buildfab.StatusError:
				icon = "❌"
			case buildfab.StatusSkipped:
				icon = "⏭️"
			default:
				icon = "❓"
			}
			fmt.Fprintf(os.Stderr, "   %s %s: %d\n", icon, status.String(), count)
		}
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