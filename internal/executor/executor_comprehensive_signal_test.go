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

// TestComprehensiveSignalHandling tests various Ctrl+C scenarios
func TestComprehensiveSignalHandling(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		description string
	}{
		{
			name:        "LongRunningCommand",
			command:     "sleep 30",
			description: "Simple long-running command",
		},
		{
			name:        "MultipleCommands",
			command:     "echo 'start' && sleep 30 && echo 'end'",
			description: "Command with multiple parts",
		},
		{
			name:        "BackgroundProcess",
			command:     "sleep 30 &",
			description: "Background process",
		},
		{
			name:        "LoopCommand",
			command:     "for i in {1..30}; do echo $i; sleep 1; done",
			description: "Looping command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test configuration
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
						Name: "test-command",
						Run:  tt.command,
					},
				},
				Stages: map[string]buildfab.Stage{
					"test-stage": {
						Steps: []buildfab.Step{
							{
								Action: "test-command",
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
			t.Logf("Testing %s: %s", tt.name, tt.description)
			t.Log("Cancelling context...")
			cancel()

			// Wait for execution to complete with a reasonable timeout
			select {
			case err := <-done:
				cancelDuration := time.Since(startCancel)
				t.Logf("Execution terminated after %v with error: %v", cancelDuration, err)
				
				// The critical test: it should terminate quickly (within 1 second)
				if cancelDuration > 1*time.Second {
					t.Errorf("Executor took too long to terminate after cancellation: %v", cancelDuration)
				}
				
				// Success - no hang detected
				t.Logf("✅ %s: Executor terminated promptly - no hang detected", tt.name)
				
			case <-time.After(3 * time.Second):
				t.Errorf("❌ %s: CRITICAL - Executor hung and did not terminate within 3 seconds", tt.name)
			}
		})
	}
}

// TestMultipleStepsSignalHandling tests Ctrl+C with multiple parallel steps
func TestMultipleStepsSignalHandling(t *testing.T) {
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
				Run:  "sleep 30",
			},
			{
				Name: "long-running-2",
				Run:  "sleep 30",
			},
			{
				Name: "long-running-3",
				Run:  "sleep 30",
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
	t.Log("Cancelling context with multiple parallel steps...")
	cancel()

	// Wait for execution to complete with a reasonable timeout
	select {
	case err := <-done:
		cancelDuration := time.Since(startCancel)
		t.Logf("Execution terminated after %v with error: %v", cancelDuration, err)
		
		// The critical test: it should terminate quickly (within 2 seconds)
		if cancelDuration > 2*time.Second {
			t.Errorf("Executor took too long to terminate after cancellation: %v", cancelDuration)
		}
		
		// Success - no hang detected
		t.Log("✅ Multiple steps: Executor terminated promptly - no hang detected")
		
	case <-time.After(5 * time.Second):
		t.Error("❌ CRITICAL: Executor hung with multiple parallel steps and did not terminate within 5 seconds")
	}
}