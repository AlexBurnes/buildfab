package executor

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// TestDirectCommandCancellation tests that a command is properly cancelled
func TestDirectCommandCancellation(t *testing.T) {
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
				Run:  "echo 'started' > /tmp/test_direct.txt && for i in {1..10}; do echo $i >> /tmp/test_direct.txt; sleep 1; done",
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
		err := executor.RunAction(ctx, "long-running")
		t.Logf("Execution completed with error: %v", err)
		done <- err
	}()

	// Wait a bit for the command to start
	time.Sleep(500 * time.Millisecond)
	
	// Check if the command started
	if _, err := os.Stat("/tmp/test_direct.txt"); err == nil {
		t.Log("Command started successfully")
		// Count lines in the file
		if content, err := os.ReadFile("/tmp/test_direct.txt"); err == nil {
			lines := strings.Split(string(content), "\n")
			t.Logf("File has %d lines: %v", len(lines), lines)
		}
	} else {
		t.Log("Command did not start or file not created")
	}

	// Simulate Ctrl+C by cancelling the context
	t.Log("Cancelling context...")
	cancel()

	// Wait for execution to complete with timeout
	select {
	case err := <-done:
		// Should return context.Canceled or similar
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		t.Logf("Execution terminated with error: %v", err)
		// Check if it's a context cancellation error
		if err != context.Canceled && !strings.Contains(err.Error(), "context canceled") {
			t.Logf("Warning: Expected context cancellation, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Execution did not terminate within 5 seconds after context cancellation")
	}
}