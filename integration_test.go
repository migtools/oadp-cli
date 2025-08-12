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

package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestBinaryBuild tests that the binary can be built successfully
func TestBinaryBuild(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	// Test that the binary is executable
	output, err := testutil.RunCommand(t, binaryPath, "--help")

	// Help command might exit with non-zero, but should produce output
	if output == "" {
		t.Errorf("Expected help output, but got empty string. Error: %v", err)
	}
}

// TestMakefileInstallation tests the Makefile installation functionality
func TestMakefileInstallation(t *testing.T) {
	// Change to project root for make commands
	projectRoot := testutil.GetProjectRoot(t)
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	err = os.Chdir(projectRoot)
	if err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	t.Run("makefile help shows installation options", func(t *testing.T) {
		cmd := exec.Command("make", "help")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to run make help: %v", err)
		}

		outputStr := string(output)
		expectedOptions := []string{
			"make install",
			"ASSUME_DEFAULT=true",
			"VELERO_NAMESPACE=velero",
		}

		for _, option := range expectedOptions {
			if !strings.Contains(outputStr, option) {
				t.Errorf("Expected make help to contain %q, but it didn't.\nFull output:\n%s", option, outputStr)
			}
		}
	})

	t.Run("make build works", func(t *testing.T) {
		// Clean first
		if err := exec.Command("make", "clean").Run(); err != nil {
			t.Logf("Failed to clean (non-fatal): %v", err)
		}

		cmd := exec.Command("make", "build")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to run make build: %v\nOutput: %s", err, string(output))
		}

		// Check binary was created
		binaryName := "kubectl-oadp"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		if _, err := os.Stat(binaryName); os.IsNotExist(err) {
			t.Errorf("Binary %s was not created", binaryName)
		}

		// Cleanup
		os.Remove(binaryName)
	})
}

// TestClientConfigIntegration tests end-to-end client configuration
func TestClientConfigIntegration(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	t.Run("client config set and get", func(t *testing.T) {
		// Set namespace
		output, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace=test-namespace")
		if err != nil {
			t.Fatalf("Failed to set client config: %v\nOutput: %s", err, output)
		}

		// Get namespace
		output, err = testutil.RunCommand(t, binaryPath, "client", "config", "get")
		if err != nil {
			t.Fatalf("Failed to get client config: %v", err)
		}

		if !strings.Contains(output, "test-namespace") {
			t.Errorf("Expected client config to contain 'test-namespace', got: %s", output)
		}
	})
}

// TestCommandArchitecture tests the overall command structure
func TestCommandArchitecture(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("all major commands exist", func(t *testing.T) {
		majorCommands := []string{"backup", "restore", "nabsl-request", "nonadmin", "client", "version"}

		output, _ := testutil.RunCommand(t, binaryPath, "--help")

		for _, cmd := range majorCommands {
			if !strings.Contains(output, cmd) {
				t.Errorf("Expected root help to contain %q command", cmd)
			}
		}
	})

	t.Run("nabsl-request command has correct subcommands", func(t *testing.T) {
		expectedSubcommands := []string{"approve", "reject", "describe", "get"}

		output, _ := testutil.RunCommand(t, binaryPath, "nabsl-request", "--help")

		for _, subcmd := range expectedSubcommands {
			if !strings.Contains(output, subcmd) {
				t.Errorf("Expected nabsl-request help to contain %q subcommand", subcmd)
			}
		}
	})
}
