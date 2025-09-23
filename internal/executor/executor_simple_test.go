package executor

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestSimpleCommandCancellation tests that a simple command is properly cancelled
func TestSimpleCommandCancellation(t *testing.T) {
	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create a long-running command
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo 'started' > /tmp/test_simple.txt && for i in {1..10}; do echo $i >> /tmp/test_simple.txt; sleep 1; done")

	// Start the command
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Wait a bit for the command to start
	time.Sleep(500 * time.Millisecond)

	// Check if the command started
	if _, err := os.Stat("/tmp/test_simple.txt"); err == nil {
		t.Log("Command started successfully")
	} else {
		t.Log("Command did not start or file not created")
	}

	// Simulate Ctrl+C by cancelling the context
	t.Log("Cancelling context...")
	cancel()

	// Wait for command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err == nil {
			t.Error("Expected command to be cancelled, but it completed successfully")
		} else {
			t.Logf("Command terminated with error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Command did not terminate within 5 seconds after context cancellation")
	}
}