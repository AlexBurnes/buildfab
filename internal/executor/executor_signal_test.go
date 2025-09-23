package executor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// TestSignalHandling tests that Ctrl+C properly terminates the executor
func TestSignalHandling(t *testing.T) {
	// Create a test configuration with a long-running command
	config := &buildfab.Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []buildfab.Action{
			{
				Name: "long-running",
				Run:  "echo 'started' > /tmp/test_signal.txt && for i in {1..10}; do echo $i >> /tmp/test_signal.txt; sleep 1; done", // This will run for 10 seconds
			},
		},
		Stages: map[string]buildfab.Stage{
			"test-stage": {
				Steps: []buildfab.Step{
					{
						Action: "long-running",
					},
				},
			},
		},
	}

	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create executor
	opts := &buildfab.RunOptions{
		Verbose: true,
		Output:  os.Stdout,
		ErrorOutput: os.Stderr,
	}
	executor := New(config, opts, &testUI{})

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		t.Log("Starting execution...")
		err := executor.RunStage(ctx, "test-stage")
		t.Logf("Execution completed with error: %v", err)
		done <- err
	}()

	// Wait a bit for the command to start
	time.Sleep(500 * time.Millisecond)
	
	// Check if the command started
	if _, err := os.Stat("/tmp/test_signal.txt"); err == nil {
		t.Log("Command started successfully")
		// Count lines in the file
		if content, err := os.ReadFile("/tmp/test_signal.txt"); err == nil {
			lines := strings.Split(string(content), "\n")
			t.Logf("File has %d lines: %v", len(lines), lines)
		}
	} else {
		t.Log("Command did not start or file not created")
	}

	// Simulate Ctrl+C by cancelling the context
	startCancel := time.Now()
	t.Log("Cancelling context...")
	cancel()

	// Wait for execution to complete with timeout
	select {
	case err := <-done:
		// The critical test: execution should terminate quickly (within 1 second)
		terminationTime := time.Since(startCancel)
		t.Logf("Execution terminated after %v with error: %v", terminationTime, err)
		
		if terminationTime > 1*time.Second {
			t.Errorf("Executor took too long to terminate after cancellation: %v", terminationTime)
		}
		
		// Success - no hang detected (error can be nil or context.Canceled)
		t.Log("✅ Executor terminated promptly - no hang detected")
		
	case <-time.After(5 * time.Second):
		t.Error("❌ CRITICAL: Executor hung and did not terminate within 5 seconds after context cancellation")
	}
}

// TestSignalHandlingWithMultipleSteps tests Ctrl+C with multiple parallel steps
func TestSignalHandlingWithMultipleSteps(t *testing.T) {
	// Create a test configuration with multiple long-running commands
	config := &buildfab.Config{
		Project: struct {
			Name    string   `yaml:"name"`
			Modules []string `yaml:"modules"`
			BinDir  string   `yaml:"bin,omitempty"`
		}{
			Name: "test-project",
		},
		Actions: []buildfab.Action{
			{
				Name: "long-running-1",
				Run:  "sleep 5",
			},
			{
				Name: "long-running-2",
				Run:  "sleep 5",
			},
			{
				Name: "long-running-3",
				Run:  "sleep 5",
			},
		},
		Stages: map[string]buildfab.Stage{
			"test-stage": {
				Steps: []buildfab.Step{
					{
						Action: "long-running-1",
					},
					{
						Action: "long-running-2",
					},
					{
						Action: "long-running-3",
					},
				},
			},
		},
	}

	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create executor
	opts := &buildfab.RunOptions{
		Verbose: true,
		Output:  os.Stdout,
		ErrorOutput: os.Stderr,
	}
	executor := New(config, opts, &testUI{})

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.RunStage(ctx, "test-stage")
	}()

	// Wait a bit for the commands to start
	time.Sleep(100 * time.Millisecond)

	// Simulate Ctrl+C by cancelling the context
	startCancel := time.Now()
	t.Log("Cancelling context...")
	cancel()

	// Wait for execution to complete with timeout
	select {
	case err := <-done:
		// The critical test: execution should terminate quickly (within 1 second)
		terminationTime := time.Since(startCancel)
		t.Logf("Execution terminated after %v with error: %v", terminationTime, err)
		
		if terminationTime > 1*time.Second {
			t.Errorf("Executor took too long to terminate after cancellation: %v", terminationTime)
		}
		
		// Success - no hang detected (error can be nil or context.Canceled)
		t.Log("✅ Executor terminated promptly - no hang detected")
		
	case <-time.After(5 * time.Second):
		t.Error("❌ CRITICAL: Executor hung and did not terminate within 5 seconds after context cancellation")
	}
}

// testUI implements the UI interface for testing
type testUI struct{}

func (ui *testUI) PrintCLIHeader(name, version string) {
	fmt.Printf("🚀 %s %s\n", name, version)
}

func (ui *testUI) PrintProjectCheck(projectName, version string) {
	fmt.Printf("📋 Project: %s\n", projectName)
}

func (ui *testUI) PrintStepStatus(stepName string, status buildfab.Status, message string) {
	fmt.Printf("Step %s: %s - %s\n", stepName, status.String(), message)
}

func (ui *testUI) PrintStageHeader(stageName string) {
	fmt.Printf("▶️  Running stage: %s\n", stageName)
}

func (ui *testUI) PrintStageResult(stageName string, success bool, duration time.Duration) {
	if success {
		fmt.Printf("🎉 SUCCESS - %s\n", stageName)
	} else {
		fmt.Printf("💥 FAILED - %s\n", stageName)
	}
}

func (ui *testUI) PrintStageTerminated(stageName string, duration time.Duration) {
	fmt.Printf("⚠️  TERMINATED - %s\n", stageName)
}

func (ui *testUI) PrintCommand(command string) {
	fmt.Printf("💻 %s\n", command)
}

func (ui *testUI) PrintStepName(stepName string) {
	fmt.Printf("💻 %s\n", stepName)
}

func (ui *testUI) PrintCommandOutput(output string) {
	fmt.Printf("    %s\n", output)
}

func (ui *testUI) PrintRepro(stepName, repro string) {
	fmt.Printf("To reproduce %s:\n%s\n", stepName, repro)
}

func (ui *testUI) PrintReproInline(stepName, repro string) {
	fmt.Printf("To reproduce %s: %s\n", stepName, repro)
}

func (ui *testUI) PrintSummary(results []buildfab.Result) {
	fmt.Printf("Summary: %d results\n", len(results))
}

func (ui *testUI) IsVerbose() bool {
	return true
}

func (ui *testUI) IsDebug() bool {
	return false
}