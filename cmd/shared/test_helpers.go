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

package shared

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestClientConfigIntegrationPattern provides a reusable pattern for testing that commands
// respect client configuration (like namespace settings). This avoids duplicating the
// setup/teardown and common testing pattern across multiple test files.
//
// testNamespace: The namespace to configure for testing
// commands: List of command sequences to test (each should work with the configured namespace)
// validateOutput: Optional function to perform additional validation on command output
func TestClientConfigIntegrationPattern(t *testing.T, testNamespace string, commands [][]string, validateOutput func(t *testing.T, cmd []string, output string)) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	// Set the test namespace via client config
	_, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace="+testNamespace)
	if err != nil {
		t.Fatalf("Failed to set client config namespace: %v", err)
	}

	// Test each command sequence
	for _, cmd := range commands {
		t.Run("config_test_"+cmd[len(cmd)-2], func(t *testing.T) {
			output, err := testutil.RunCommand(t, binaryPath, cmd...)
			if err != nil {
				t.Fatalf("Command should work with client config: %v\nCommand: %v", err, cmd)
			}
			if output == "" {
				t.Errorf("Expected help output for command: %v", cmd)
			}

			// Run additional validation if provided
			if validateOutput != nil {
				validateOutput(t, cmd, output)
			}
		})
	}
}

// ClientConfigTestCommands is a helper type for organizing command test data
type ClientConfigTestCommands struct {
	Name         string
	Commands     [][]string
	TestSetup    func(t *testing.T, binaryPath string) // Optional additional setup
	Namespace    string                               // Namespace to configure
	ValidateFunc func(t *testing.T, cmd []string, output string) // Optional output validation
}

// RunClientConfigTests runs client config integration tests for multiple command groups
func RunClientConfigTests(t *testing.T, testGroups []ClientConfigTestCommands) {
	for _, group := range testGroups {
		t.Run(group.Name, func(t *testing.T) {
			namespace := group.Namespace
			if namespace == "" {
				namespace = "test-namespace" // Default test namespace
			}

			TestClientConfigIntegrationPattern(t, namespace, group.Commands, group.ValidateFunc)
		})
	}
}