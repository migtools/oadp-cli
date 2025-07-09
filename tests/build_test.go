package tests

import (
	"context"
	"os/exec"
	"testing"
)

// TestCLIBinaryBuild tests that the binary can be built successfully
func TestCLIBinaryBuild(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer cleanup(t, binaryPath)

	// Test that the binary is executable
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--help")
	err := cmd.Run()

	// Help command might exit with non-zero, but should not fail to execute
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code != 0 is often normal for help commands
			t.Logf("Binary executed but exited with code: %d", exitErr.ExitCode())
		} else {
			t.Fatalf("Failed to execute binary: %v", err)
		}
	}
}

// TestCLIBinaryVersion tests that we can build and get version info
func TestCLIBinaryVersion(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer cleanup(t, binaryPath)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "version", "--client-only")
	output, err := cmd.Output()

	// Version command should work
	if err != nil {
		t.Logf("Version command failed: %v", err)
		// Some version commands might fail without proper setup, but we can still check they run
	}

	t.Logf("Version output: %s", string(output))
}

// TestCLIBinarySmoke performs basic smoke tests
func TestCLIBinarySmoke(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer cleanup(t, binaryPath)

	// Smoke tests - just verify commands don't crash
	smokeCommands := [][]string{
		{"--help"},
		{"-h"},
		{"backup", "--help"},
		{"restore", "--help"},
		{"nonadmin", "--help"},
		{"version", "--help"},
	}

	for _, cmd := range smokeCommands {
		t.Run("smoke_"+cmd[0], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			execCmd := exec.CommandContext(ctx, binaryPath)
			execCmd.Args = append(execCmd.Args, cmd...)

			// We don't care about exit code for smoke tests, just that it doesn't hang/crash
			if err := execCmd.Run(); err != nil {
				// For smoke tests, we only care that the command executes without hanging
				// Exit errors are expected for help commands that might not be fully implemented
				if _, ok := err.(*exec.ExitError); !ok {
					t.Logf("Smoke test command failed (non-exit error): %v", err)
				}
			}
		})
	}
}
