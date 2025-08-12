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

package nabsl

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestNABSLCommands tests the NABSL command functionality
func TestNABSLCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nabsl-request help",
			args: []string{"nabsl-request", "--help"},
			expectContains: []string{
				"Manage approval requests for non-admin backup storage locations",
				"approve",
				"reject",
				"describe",
				"get",
			},
		},
		{
			name: "nabsl-request approve help",
			args: []string{"nabsl-request", "approve", "--help"},
			expectContains: []string{
				"Approve a pending backup storage location request",
				"--reason",
			},
		},
		{
			name: "nabsl-request reject help",
			args: []string{"nabsl-request", "reject", "--help"},
			expectContains: []string{
				"Reject a pending backup storage location request",
				"--reason",
			},
		},
		{
			name: "nabsl-request get help",
			args: []string{"nabsl-request", "get", "--help"},
			expectContains: []string{
				"Get non-admin backup storage location requests",
			},
		},
		{
			name: "nabsl-request describe help",
			args: []string{"nabsl-request", "describe", "--help"},
			expectContains: []string{
				"Describe a non-admin backup storage location request",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNABSLHelpFlags tests that both --help and -h work for nabsl-request commands
func TestNABSLHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"nabsl-request", "--help"},
		{"nabsl-request", "-h"},
		{"nabsl-request", "approve", "--help"},
		{"nabsl-request", "approve", "-h"},
		{"nabsl-request", "reject", "--help"},
		{"nabsl-request", "reject", "-h"},
		{"nabsl-request", "get", "--help"},
		{"nabsl-request", "get", "-h"},
		{"nabsl-request", "describe", "--help"},
		{"nabsl-request", "describe", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}

// TestNABSLClientConfigIntegration tests that NABSL request commands respect client config
func TestNABSLClientConfigIntegration(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	t.Run("nabsl-request commands work with client config", func(t *testing.T) {
		// Set a known namespace
		_, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace=admin-namespace")
		if err != nil {
			t.Fatalf("Failed to set client config: %v", err)
		}

		// Test that nabsl-request commands can be invoked (they should respect the namespace)
		// We test help commands since they don't require actual K8s resources
		commands := [][]string{
			{"nabsl-request", "get", "--help"},
			{"nabsl-request", "approve", "--help"},
			{"nabsl-request", "reject", "--help"},
			{"nabsl-request", "describe", "--help"},
		}

		for _, cmd := range commands {
			t.Run("config_test_"+cmd[1], func(t *testing.T) {
				output, err := testutil.RunCommand(t, binaryPath, cmd...)
				if err != nil {
					t.Fatalf("NABSL request command should work with client config: %v", err)
				}
				if output == "" {
					t.Errorf("Expected help output for %v", cmd)
				}
			})
		}
	})
}
