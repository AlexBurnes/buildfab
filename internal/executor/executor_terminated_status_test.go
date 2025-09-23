package executor

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

// TestTerminatedStatusDisplay tests that TERMINATED status is displayed when Ctrl+C is pressed
func TestTerminatedStatusDisplay(t *testing.T) {
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
				Run:  "sleep 10",
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

	// Create executor with a custom UI that captures output
	opts := &buildfab.RunOptions{
		Verbose: true,
		Output:  os.Stdout,
		ErrorOutput: os.Stderr,
	}
	
	// Create a custom UI that captures the output
	captureUI := &captureUI{}
	executor := New(config, opts, captureUI)

	// Start execution in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.RunStage(ctx, "test-stage")
	}()

	// Wait a bit for the command to start
	time.Sleep(100 * time.Millisecond)

	// Simulate Ctrl+C by cancelling the context
	t.Log("Cancelling context...")
	cancel()

	// Wait for execution to complete
	select {
	case err := <-done:
		t.Logf("Execution completed with error: %v", err)
		
		// Check that TERMINATED status was displayed
		output := captureUI.GetOutput()
		t.Logf("Captured output: %s", output)
		
		if !strings.Contains(output, "TERMINATED") {
			t.Error("Expected TERMINATED status to be displayed, but it was not found in output")
		}
		
		if !strings.Contains(output, "⚠️") {
			t.Error("Expected warning icon (⚠️) to be displayed, but it was not found in output")
		}
		
		t.Log("✅ TERMINATED status displayed correctly")
		
	case <-time.After(5 * time.Second):
		t.Error("Execution did not terminate within 5 seconds after context cancellation")
	}
}

// captureUI captures output for testing
type captureUI struct {
	output strings.Builder
	mu     sync.Mutex
}

func (c *captureUI) PrintCLIHeader(name, version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("CLI Header\n")
}

func (c *captureUI) PrintProjectCheck(projectName, version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Project Check\n")
}

func (c *captureUI) PrintStepStatus(stepName string, status buildfab.Status, message string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Step Status\n")
}

func (c *captureUI) PrintStageHeader(stageName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Stage Header\n")
}

func (c *captureUI) PrintStageResult(stageName string, success bool, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if success {
		c.output.WriteString("SUCCESS\n")
	} else {
		c.output.WriteString("FAILED\n")
	}
}

func (c *captureUI) PrintStageTerminated(stageName string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("⚠️  TERMINATED - " + stageName + "\n")
}

func (c *captureUI) PrintCommand(command string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Command: " + command + "\n")
}

func (c *captureUI) PrintStepName(stepName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Step: " + stepName + "\n")
}

func (c *captureUI) PrintCommandOutput(output string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Output: " + output + "\n")
}

func (c *captureUI) PrintRepro(stepName, repro string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Repro: " + repro + "\n")
}

func (c *captureUI) PrintReproInline(stepName, repro string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Repro Inline: " + repro + "\n")
}

func (c *captureUI) PrintSummary(results []buildfab.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output.WriteString("Summary\n")
}

func (c *captureUI) IsVerbose() bool {
	return true
}

func (c *captureUI) IsDebug() bool {
	return false
}

func (c *captureUI) GetOutput() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.output.String()
}