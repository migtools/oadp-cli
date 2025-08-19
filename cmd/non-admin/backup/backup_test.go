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

package backup

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestNonAdminBackupCommands tests the non-admin backup command functionality
func TestNonAdminBackupCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin backup help",
			args: []string{"nonadmin", "backup", "--help"},
			expectContains: []string{
				"Work with non-admin backups",
				"create",
				"describe",
				"delete",
				"get",
				"logs",
			},
		},
		{
			name: "nonadmin backup create help",
			args: []string{"nonadmin", "backup", "create", "--help"},
			expectContains: []string{
				"Create a non-admin backup",
				"--storage-location",
				"--include-resources",
				"--exclude-resources",
				"--wait",
				"--force",
				"--assume-yes",
			},
		},
		{
			name: "nonadmin backup describe help",
			args: []string{"nonadmin", "backup", "describe", "--help"},
			expectContains: []string{
				"Describe a non-admin backup",
			},
		},
		{
			name: "nonadmin backup delete help",
			args: []string{"nonadmin", "backup", "delete", "--help"},
			expectContains: []string{
				"Delete one or more non-admin backups",
				"--confirm",
				"--all",
			},
		},
		{
			name: "nonadmin backup get help",
			args: []string{"nonadmin", "backup", "get", "--help"},
			expectContains: []string{
				"Get one or more non-admin backups",
			},
		},
		{
			name: "nonadmin backup logs help",
			args: []string{"nonadmin", "backup", "logs", "--help"},
			expectContains: []string{
				"Show logs for a non-admin backup",
			},
		},
		{
			name: "na backup shorthand help",
			args: []string{"na", "backup", "--help"},
			expectContains: []string{
				"Work with non-admin backups",
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

// TestNonAdminBackupHelpFlags tests that both --help and -h work for backup commands
func TestNonAdminBackupHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"nonadmin", "backup", "--help"},
		{"nonadmin", "backup", "-h"},
		{"nonadmin", "backup", "create", "--help"},
		{"nonadmin", "backup", "create", "-h"},
		{"nonadmin", "backup", "describe", "--help"},
		{"nonadmin", "backup", "describe", "-h"},
		{"nonadmin", "backup", "delete", "--help"},
		{"nonadmin", "backup", "delete", "-h"},
		{"nonadmin", "backup", "get", "--help"},
		{"nonadmin", "backup", "get", "-h"},
		{"nonadmin", "backup", "logs", "--help"},
		{"nonadmin", "backup", "logs", "-h"},
		{"na", "backup", "--help"},
		{"na", "backup", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}

// TestNonAdminBackupCreateFlags tests create command specific flags
func TestNonAdminBackupCreateFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create command has all expected flags", func(t *testing.T) {
		expectedFlags := []string{
			"--storage-location",
			"--include-resources",
			"--exclude-resources",
			"--labels",
			"--annotations",
			"--wait",
			"--force",
			"--assume-yes",
			"--snapshot-volumes",
			"--ttl",
			"--selector",
			"--or-selector",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "create", "--help"},
			expectedFlags)
	})
}

// TestNonAdminBackupExamples tests that help text contains proper examples
func TestNonAdminBackupExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create examples use correct command format", func(t *testing.T) {
		expectedExamples := []string{
			"kubectl oadp nonadmin backup create",
			"--storage-location",
			"--include-resources",
			"--exclude-resources",
			"--wait",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "create", "--help"},
			expectedExamples)
	})

	t.Run("main backup help shows subcommands", func(t *testing.T) {
		expectedSubcommands := []string{
			"create",
			"delete",
			"describe",
			"get",
			"logs",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "--help"},
			expectedSubcommands)
	})
}

// TestNonAdminBackupClientConfigIntegration tests that backup commands respect client config
func TestNonAdminBackupClientConfigIntegration(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	t.Run("backup commands work with client config", func(t *testing.T) {
		// Set a known namespace
		_, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace=user-namespace")
		if err != nil {
			t.Fatalf("Failed to set client config: %v", err)
		}

		// Test that backup commands can be invoked (they should respect the namespace)
		// We test help commands since they don't require actual K8s resources
		commands := [][]string{
			{"nonadmin", "backup", "get", "--help"},
			{"nonadmin", "backup", "create", "--help"},
			{"nonadmin", "backup", "describe", "--help"},
			{"nonadmin", "backup", "delete", "--help"},
			{"nonadmin", "backup", "logs", "--help"},
			{"na", "backup", "get", "--help"},
		}

		for _, cmd := range commands {
			t.Run("config_test_"+cmd[len(cmd)-2], func(t *testing.T) {
				output, err := testutil.RunCommand(t, binaryPath, cmd...)
				if err != nil {
					t.Fatalf("Non-admin backup command should work with client config: %v", err)
				}
				if output == "" {
					t.Errorf("Expected help output for %v", cmd)
				}
			})
		}
	})
}

// TestNonAdminBackupCommandStructure tests the overall command structure
func TestNonAdminBackupCommandStructure(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("backup commands available under nonadmin", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "nonadmin", "--help")
		if err != nil {
			t.Fatalf("nonadmin command should exist: %v", err)
		}

		expectedCommands := []string{"backup"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"nonadmin", "--help"}, []string{cmd})
		}
	})

	t.Run("backup commands available under na shorthand", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "na", "--help")
		if err != nil {
			t.Fatalf("na command should exist: %v", err)
		}

		expectedCommands := []string{"backup"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"na", "--help"}, []string{cmd})
		}
	})
}
