package executor

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// TestExecutorHangPrevention tests that the executor does not hang on Ctrl+C
func TestExecutorHangPrevention(t *testing.T) {
	// Create a test configuration with a very long-running command
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
				Name: "very-long-running",
				Run:  "sleep 60", // This will run for 60 seconds
			},
		},
		Stages: map[string]buildfab.Stage{
			"test-stage": {
				Steps: []buildfab.Step{
					{
						Action: "very-long-running",
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

	// Wait a bit for the command to start
	time.Sleep(100 * time.Millisecond)

	// Simulate Ctrl+C by cancelling the context
	startCancel := time.Now()
	t.Log("Cancelling context...")
	cancel()

	// Wait for execution to complete with a reasonable timeout
	// The key test is that it should terminate within a few seconds, not hang
	select {
	case err := <-done:
		cancelDuration := time.Since(startCancel)
		t.Logf("Execution terminated after %v with error: %v", cancelDuration, err)
		
		// The critical test: it should terminate quickly (within 2 seconds)
		if cancelDuration > 2*time.Second {
			t.Errorf("Executor took too long to terminate after cancellation: %v", cancelDuration)
		}
		
		// It's OK if it returns nil error - the important thing is it terminates quickly
		t.Log("✅ Executor terminated promptly - no hang detected")
		
	case <-time.After(5 * time.Second):
		t.Error("❌ CRITICAL: Executor hung and did not terminate within 5 seconds after context cancellation")
	}
}