package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AlexBurnes/buildfab/pkg/buildfab"
)

func TestNew(t *testing.T) {
	ui := New(true, false)
	if ui.verbose != true {
		t.Errorf("verbose = %v, want %v", ui.verbose, true)
	}
	if ui.debug != false {
		t.Errorf("debug = %v, want %v", ui.debug, false)
	}
}

func TestUI_IsVerbose(t *testing.T) {
	ui := New(true, false)
	if !ui.IsVerbose() {
		t.Error("IsVerbose() should return true")
	}

	ui2 := New(false, false)
	if ui2.IsVerbose() {
		t.Error("IsVerbose() should return false")
	}
}

func TestUI_IsDebug(t *testing.T) {
	ui := New(false, true)
	if !ui.IsDebug() {
		t.Error("IsDebug() should return true")
	}

	ui2 := New(false, false)
	if ui2.IsDebug() {
		t.Error("IsDebug() should return false")
	}
}

func TestUI_PrintCLIHeader(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintCLIHeader("buildfab", "1.0.0")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "🚀 buildfab v1.0.0") {
		t.Errorf("PrintCLIHeader() output should contain header, got: %s", output)
	}
	if !strings.Contains(output, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") {
		t.Errorf("PrintCLIHeader() output should contain separator line, got: %s", output)
	}
}

func TestUI_PrintProjectCheck(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintProjectCheck("test-project", "v1.0.0")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "📦 Project: test-project") {
		t.Errorf("PrintProjectCheck() output should contain project info, got: %s", output)
	}
	if !strings.Contains(output, "🏷️  Version: v1.0.0") {
		t.Errorf("PrintProjectCheck() output should contain version info, got: %s", output)
	}
}

func TestUI_PrintStageHeader(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintStageHeader("test-stage")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "▶️  Running stage: test-stage") {
		t.Errorf("PrintStageHeader() output should contain stage info, got: %s", output)
	}
}

func TestUI_PrintStepStatus(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintStepStatus("test-step", buildfab.StatusOK, "Success message")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "test-step") {
		t.Errorf("PrintStepStatus() output should contain step name, got: %s", output)
	}
	if !strings.Contains(output, "Success message") {
		t.Errorf("PrintStepStatus() output should contain message, got: %s", output)
	}
}

func TestUI_PrintStageResult(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintStageResult("test-stage", true, 5*time.Second)

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "test-stage") {
		t.Errorf("PrintStageResult() output should contain stage name, got: %s", output)
	}
	if !strings.Contains(output, "5.00s") {
		t.Errorf("PrintStageResult() output should contain duration, got: %s", output)
	}
}

func TestUI_PrintCommand(t *testing.T) {
	ui := New(true, false) // Enable verbose mode
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintCommand("echo hello")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "echo hello") {
		t.Errorf("PrintCommand() output should contain command, got: %s", output)
	}
}

func TestUI_PrintCommandOutput(t *testing.T) {
	ui := New(true, false) // Enable verbose mode
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintCommandOutput("Hello World")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "Hello World") {
		t.Errorf("PrintCommandOutput() output should contain output, got: %s", output)
	}
}

func TestUI_PrintRepro(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintRepro("test-step", "To reproduce: run command")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "test-step") {
		t.Errorf("PrintRepro() output should contain step name, got: %s", output)
	}
	if !strings.Contains(output, "To reproduce: run command") {
		t.Errorf("PrintRepro() output should contain repro message, got: %s", output)
	}
}

func TestUI_PrintReproInline(t *testing.T) {
	ui := New(false, false)
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintReproInline("test-step", "To reproduce: run command")

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "To reproduce: run command") {
		t.Errorf("PrintReproInline() output should contain repro message, got: %s", output)
	}
}

func TestUI_PrintSummary(t *testing.T) {
	ui := New(false, false)
	
	results := []buildfab.Result{
		{
			Name:    "step1",
			Status:  buildfab.StatusOK,
			Message: "Success",
			Duration: 1 * time.Second,
		},
		{
			Name:    "step2",
			Status:  buildfab.StatusError,
			Message: "Failed",
			Duration: 2 * time.Second,
		},
	}
	
	// Capture stderr output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ui.PrintSummary(results)

	// Close write end and read output
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Check that output contains expected content
	if !strings.Contains(output, "Summary") {
		t.Errorf("PrintSummary() output should contain summary header, got: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("PrintSummary() output should contain ERROR status, got: %s", output)
	}
	if !strings.Contains(output, "OK") {
		t.Errorf("PrintSummary() output should contain OK status, got: %s", output)
	}
}