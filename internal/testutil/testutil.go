/*
Copyright 2025 The OADP CLI Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package testutil provides shared testing utilities for the OADP CLI
package testutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	// TestTimeout is the default timeout for test operations
	TestTimeout = 30 * time.Second
)

// GetProjectRoot returns the root directory of the project
func GetProjectRoot(t *testing.T) string {
	t.Helper()

	// Get the directory of the current file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get caller information")
	}

	// Navigate up to find the project root (where go.mod is)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

// BuildCLIBinary builds the CLI binary for testing and returns the path
func BuildCLIBinary(t *testing.T) string {
	t.Helper()

	projectRoot := GetProjectRoot(t)

	// Create temp directory for the binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "oadp-test")

	t.Logf("Building CLI binary: %s", binaryPath)
	t.Logf("Project root: %s", projectRoot)

	// Build the binary
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nOutput: %s", err, string(output))
	}

	return binaryPath
}

// RunCommand runs a command with the given binary and arguments
func RunCommand(t *testing.T, binaryPath string, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Args = append(cmd.Args, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Log the command and output for debugging
	t.Logf("Command: %s %s", binaryPath, strings.Join(args, " "))
	if stdout.Len() > 0 {
		t.Logf("Stdout: %s", stdout.String())
	}
	if stderr.Len() > 0 {
		t.Logf("Stderr: %s", stderr.String())
	}

	return stdout.String(), err
}

// TestHelpCommand tests that a command's help output contains expected strings
func TestHelpCommand(t *testing.T, binaryPath string, args []string, expectContains []string) {
	t.Helper()

	output, _ := RunCommand(t, binaryPath, args...)

	// Help commands might exit with non-zero, which is normal
	t.Logf("Command: %s %s", binaryPath, strings.Join(args, " "))
	t.Logf("Output:\n%s", output)

	// Check that all expected strings are present
	for _, expected := range expectContains {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
		}
	}
}

// SetupTempHome creates a temporary home directory for testing client config
func SetupTempHome(t *testing.T) (string, func()) {
	t.Helper()

	tempHome := t.TempDir()
	configDir := filepath.Join(tempHome, ".config", "velero")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Save original HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)

	// Return cleanup function
	cleanup := func() {
		os.Setenv("HOME", originalHome)
	}

	return tempHome, cleanup
}
