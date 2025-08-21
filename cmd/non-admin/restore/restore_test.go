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

package restore

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestNonAdminRestoreCommands tests the non-admin restore command functionality
func TestNonAdminRestoreCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin restore help",
			args: []string{"nonadmin", "restore", "--help"},
			expectContains: []string{
				"Work with non-admin restores",
				"create",
				"describe",
				"delete",
				"get",
				"logs",
			},
		},
		{
			name: "nonadmin restore create help",
			args: []string{"nonadmin", "restore", "create", "--help"},
			expectContains: []string{
				"Create a non-admin restore",
				"--from-backup",
				"--include-resources",
				"--exclude-resources",
				"--wait",
			},
		},
		{
			name: "nonadmin restore describe help",
			args: []string{"nonadmin", "restore", "describe", "--help"},
			expectContains: []string{
				"Describe a non-admin restore",
			},
		},
		{
			name: "nonadmin restore delete help",
			args: []string{"nonadmin", "restore", "delete", "--help"},
			expectContains: []string{
				"Delete one or more non-admin restores",
			},
		},
		{
			name: "nonadmin restore get help",
			args: []string{"nonadmin", "restore", "get", "--help"},
			expectContains: []string{
				"Get non-admin restores in the current namespace",
			},
		},
		{
			name: "nonadmin restore logs help",
			args: []string{"nonadmin", "restore", "logs", "--help"},
			expectContains: []string{
				"Display logs for a specified non-admin restore operation",
			},
		},
		{
			name: "na restore shorthand help",
			args: []string{"na", "restore", "--help"},
			expectContains: []string{
				"Work with non-admin restores",
				"create",
				"describe",
				"delete",
				"get",
				"logs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminRestoreHelpFlags tests that both --help and -h work for restore commands
func TestNonAdminRestoreHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"nonadmin", "restore", "--help"},
		{"nonadmin", "restore", "-h"},
		{"nonadmin", "restore", "create", "--help"},
		{"nonadmin", "restore", "create", "-h"},
		{"nonadmin", "restore", "describe", "--help"},
		{"nonadmin", "restore", "describe", "-h"},
		{"nonadmin", "restore", "delete", "--help"},
		{"nonadmin", "restore", "delete", "-h"},
		{"nonadmin", "restore", "get", "--help"},
		{"nonadmin", "restore", "get", "-h"},
		{"nonadmin", "restore", "logs", "--help"},
		{"nonadmin", "restore", "logs", "-h"},
		{"na", "restore", "--help"},
		{"na", "restore", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}

// TestNonAdminRestoreCreateFlags tests create command specific flags
func TestNonAdminRestoreCreateFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create command has all expected flags", func(t *testing.T) {
		expectedFlags := []string{
			"--from-backup",
			"--include-resources",
			"--exclude-resources",
			"--labels",
			"--annotations",
			"--wait",
			"--selector",
			"--or-selector",
			"--include-cluster-resources",
			"--restore-volumes",
			"--preserve-nodeports",
			"--item-operation-timeout",
			"--existing-resource-policy",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "create", "--help"},
			expectedFlags)
	})
}

// TestNonAdminRestoreExamples tests that help text contains proper examples
func TestNonAdminRestoreExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create examples use correct command format", func(t *testing.T) {
		expectedExamples := []string{
			"kubectl oadp nonadmin restore create",
			"--from-backup",
			"--include-resources",
			"--exclude-resources",
			"--wait",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "create", "--help"},
			expectedExamples)
	})

	t.Run("main restore help shows subcommands", func(t *testing.T) {
		expectedSubcommands := []string{
			"create",
			"delete",
			"describe",
			"get",
			"logs",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "--help"},
			expectedSubcommands)
	})
}

// TestNonAdminRestoreClientConfigIntegration tests that restore commands respect client config
func TestNonAdminRestoreClientConfigIntegration(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	t.Run("restore commands work with client config", func(t *testing.T) {
		// Set a known namespace
		_, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace=user-namespace")
		if err != nil {
			t.Fatalf("Failed to set client config: %v", err)
		}

		// Test that restore commands can be invoked (they should respect the namespace)
		// We test help commands since they don't require actual K8s resources
		commands := [][]string{
			{"nonadmin", "restore", "get", "--help"},
			{"nonadmin", "restore", "create", "--help"},
			{"nonadmin", "restore", "describe", "--help"},
			{"nonadmin", "restore", "delete", "--help"},
			{"nonadmin", "restore", "logs", "--help"},
			{"na", "restore", "get", "--help"},
		}

		for _, cmd := range commands {
			t.Run("config_test_"+cmd[len(cmd)-2], func(t *testing.T) {
				output, err := testutil.RunCommand(t, binaryPath, cmd...)
				if err != nil {
					t.Fatalf("Non-admin restore command should work with client config: %v", err)
				}
				if output == "" {
					t.Errorf("Expected help output for %v", cmd)
				}
			})
		}
	})
}

// TestNonAdminRestoreCommandStructure tests the overall command structure
func TestNonAdminRestoreCommandStructure(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("restore commands available under nonadmin", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "nonadmin", "--help")
		if err != nil {
			t.Fatalf("nonadmin command should exist: %v", err)
		}

		expectedCommands := []string{"restore"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"nonadmin", "--help"}, []string{cmd})
		}
	})

	t.Run("restore commands available under na shorthand", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "na", "--help")
		if err != nil {
			t.Fatalf("na command should exist: %v", err)
		}

		expectedCommands := []string{"restore"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"na", "--help"}, []string{cmd})
		}
	})
}
