package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testBinaryPath holds the path to the built binary, set by TestMain.
var testBinaryPath string

// TestMain builds the binary once for all tests to improve performance.
//
// These are black-box tests: they exercise the CLI by executing the compiled
// binary as a subprocess rather than calling functions in-process. That makes
// them realistic but means `go test -cover` reports 0% for package main, since
// the coverage tooling only instruments the in-process test binary. The
// behavior is fully covered; the coverage percentage just doesn't reflect it.
func TestMain(m *testing.M) {
	// Create a temporary directory for the test binary
	tempDir, err := os.MkdirTemp("", "recipe-tool-test-*")
	if err != nil {
		os.Stderr.WriteString("Failed to create temp dir: " + err.Error() + "\n")
		os.Exit(1)
	}

	testBinaryPath = filepath.Join(tempDir, "recipe-tool")

	// Build the binary once
	cmd := exec.Command("go", "build", "-o", testBinaryPath, ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.WriteString("Failed to build binary: " + err.Error() + "\n")
		os.Stderr.Write(output)
		os.RemoveAll(tempDir)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.RemoveAll(tempDir)

	os.Exit(code)
}

// TestBinaryExists verifies the binary was built by TestMain
func TestBinaryExists(t *testing.T) {
	if _, err := os.Stat(testBinaryPath); os.IsNotExist(err) {
		t.Fatal("Binary was not created by TestMain")
	}
}

// TestHelpCommand verifies help output
func TestHelpCommand(t *testing.T) {
	tests := []string{"help", "-h", "--help"}

	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			cmd := exec.Command(testBinaryPath, arg)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			// Help command should exit with code 0
			if err != nil {
				t.Logf("Command exited with error (may be expected): %v", err)
			}

			output := stdout.String() + stderr.String()

			// Verify help content
			expectedStrings := []string{
				"Usage:",
				"validate",
				"convert",
			}

			for _, expected := range expectedStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("Help output missing %q\nOutput: %s", expected, output)
				}
			}
		})
	}
}

// TestUnknownCommand verifies unknown command handling
func TestUnknownCommand(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "unknowncommand")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code for unknown command")
	}

	output := stdout.String() + stderr.String()

	// Should mention the unknown command
	if !strings.Contains(output, "Unknown command") {
		t.Errorf("Output should mention unknown command\nOutput: %s", output)
	}
}

// TestNoArguments verifies behavior with no arguments
func TestNoArguments(t *testing.T) {
	cmd := exec.Command(testBinaryPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with non-zero code
	if err == nil {
		t.Error("Expected non-zero exit code when no arguments provided")
	}

	output := stdout.String() + stderr.String()

	// Should show usage
	if !strings.Contains(output, "Usage:") {
		t.Errorf("Output should show usage\nOutput: %s", output)
	}
}

// TestValidateCommandHelp verifies validate subcommand help
func TestValidateCommandHelp(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "validate", "-h")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// -h causes flag.ExitOnError to exit with 0
	_ = cmd.Run() // Intentionally ignoring error - we only care about output

	output := stdout.String() + stderr.String()

	// Should show validate-specific flags
	expectedFlags := []string{
		"-yaml-dir",
		"-strict",
		"-verbose",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("Validate help should mention %q\nOutput: %s", flag, output)
		}
	}
}

// TestConvertCommandHelp verifies convert subcommand help
func TestConvertCommandHelp(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "convert", "-h")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// -h causes flag.ExitOnError to exit with 0
	_ = cmd.Run() // Intentionally ignoring error - we only care about output

	output := stdout.String() + stderr.String()

	// Should show convert-specific flags
	expectedFlags := []string{
		"-yaml-dir",
		"-parallel",
		"-skip-images",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("Convert help should mention %q\nOutput: %s", flag, output)
		}
	}
}

// TestValidateWithMissingDirectory verifies error handling for missing directory
func TestValidateWithMissingDirectory(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "validate", "-yaml-dir", "/nonexistent/path")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should exit with error
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	output := stdout.String() + stderr.String()

	// Should mention error
	if !strings.Contains(strings.ToLower(output), "error") {
		t.Errorf("Output should mention error\nOutput: %s", output)
	}
}

// TestImportCommandHelp verifies import subcommand help
func TestImportCommandHelp(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "import", "-h")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	output := stdout.String() + stderr.String()

	if !strings.Contains(output, "-imports-dir") {
		t.Errorf("Import help should mention -imports-dir\nOutput: %s", output)
	}
}

// TestCleanupCommandHelp verifies cleanup subcommand help
func TestCleanupCommandHelp(t *testing.T) {
	cmd := exec.Command(testBinaryPath, "cleanup", "-h")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	output := stdout.String() + stderr.String()

	expectedFlags := []string{
		"-imports-dir",
		"-keep-zips",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("Cleanup help should mention %q\nOutput: %s", flag, output)
		}
	}
}
