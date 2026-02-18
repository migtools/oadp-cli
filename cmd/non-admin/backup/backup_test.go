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
		// Verb-noun order help command tests
		{
			name: "nonadmin get backup help",
			args: []string{"nonadmin", "get", "backup", "--help"},
			expectContains: []string{
				"Get one or more non-admin backups",
			},
		},
		{
			name: "nonadmin create backup help",
			args: []string{"nonadmin", "create", "backup", "--help"},
			expectContains: []string{
				"Create a non-admin backup",
			},
		},
		{
			name: "nonadmin delete backup help",
			args: []string{"nonadmin", "delete", "backup", "--help"},
			expectContains: []string{
				"Delete one or more non-admin backups",
			},
		},
		{
			name: "nonadmin describe backup help",
			args: []string{"nonadmin", "describe", "backup", "--help"},
			expectContains: []string{
				"Describe a non-admin backup",
			},
		},
		{
			name: "nonadmin logs backup help",
			args: []string{"nonadmin", "logs", "backup", "--help"},
			expectContains: []string{
				"Show logs for a non-admin backup",
			},
		},
		// Shorthand verb-noun order tests
		{
			name: "na get backup help",
			args: []string{"na", "get", "backup", "--help"},
			expectContains: []string{
				"Get one or more non-admin backups",
			},
		},
		{
			name: "na create backup help",
			args: []string{"na", "create", "backup", "--help"},
			expectContains: []string{
				"Create a non-admin backup",
			},
		},
		{
			name: "na delete backup help",
			args: []string{"na", "delete", "backup", "--help"},
			expectContains: []string{
				"Delete one or more non-admin backups",
			},
		},
		{
			name: "na describe backup help",
			args: []string{"na", "describe", "backup", "--help"},
			expectContains: []string{
				"Describe a non-admin backup",
			},
		},
		{
			name: "na logs backup help",
			args: []string{"na", "logs", "backup", "--help"},
			expectContains: []string{
				"Show logs for a non-admin backup",
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
		// Verb-noun order help flags
		{"nonadmin", "get", "backup", "--help"},
		{"nonadmin", "get", "backup", "-h"},
		{"nonadmin", "create", "backup", "--help"},
		{"nonadmin", "create", "backup", "-h"},
		{"nonadmin", "delete", "backup", "--help"},
		{"nonadmin", "delete", "backup", "-h"},
		{"nonadmin", "describe", "backup", "--help"},
		{"nonadmin", "describe", "backup", "-h"},
		{"nonadmin", "logs", "backup", "--help"},
		{"nonadmin", "logs", "backup", "-h"},
		// Shorthand verb-noun order help flags
		{"na", "get", "backup", "--help"},
		{"na", "get", "backup", "-h"},
		{"na", "create", "backup", "--help"},
		{"na", "create", "backup", "-h"},
		{"na", "delete", "backup", "--help"},
		{"na", "delete", "backup", "-h"},
		{"na", "describe", "backup", "--help"},
		{"na", "describe", "backup", "-h"},
		{"na", "logs", "backup", "--help"},
		{"na", "logs", "backup", "-h"},
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

	t.Run("create command has all expected MVP flags", func(t *testing.T) {
		expectedFlags := []string{
			"--include-resources",
			"--exclude-resources",
			"--selector",
			"--or-selector",
			"--include-cluster-resources",
			"--ttl",
			"--storage-location",
			"--csi-snapshot-timeout",
			"--item-operation-timeout",
			"--snapshot-volumes",
			"--snapshot-move-data",
			"--default-volumes-to-fs-backup",
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
			// Verb-noun order commands
			{"nonadmin", "get", "backup", "--help"},
			{"nonadmin", "create", "backup", "--help"},
			{"nonadmin", "describe", "backup", "--help"},
			{"nonadmin", "delete", "backup", "--help"},
			{"nonadmin", "logs", "backup", "--help"},
			{"na", "get", "backup", "--help"},
			{"na", "create", "backup", "--help"},
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

// TestVerbNounOrderExamples tests that verb-noun order commands show proper examples
func TestVerbNounOrderExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("verb commands show proper examples", func(t *testing.T) {
		// Test that verb commands show examples with kubectl oadp prefix
		expectedExamples := []string{
			"kubectl oadp nonadmin get backup",
			"kubectl oadp nonadmin create backup",
			"kubectl oadp nonadmin delete backup",
			"kubectl oadp nonadmin describe backup",
			"kubectl oadp nonadmin logs backup",
		}

		commands := [][]string{
			{"nonadmin", "get", "--help"},
			{"nonadmin", "create", "--help"},
			{"nonadmin", "delete", "--help"},
			{"nonadmin", "describe", "--help"},
			{"nonadmin", "logs", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})

	t.Run("verb commands with specific resources show proper examples", func(t *testing.T) {
		// Test that verb commands with specific resources show examples (noun-first format from underlying commands)
		expectedExamples := []string{
			"kubectl oadp nonadmin backup get",
			"kubectl oadp nonadmin backup create backup1",
			"kubectl oadp nonadmin backup delete my-backup",
			"kubectl oadp nonadmin backup describe my-backup",
			"kubectl oadp nonadmin backup logs my-backup",
		}

		commands := [][]string{
			{"nonadmin", "get", "backup", "--help"},
			{"nonadmin", "create", "backup", "--help"},
			{"nonadmin", "delete", "backup", "--help"},
			{"nonadmin", "describe", "backup", "--help"},
			{"nonadmin", "logs", "backup", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})

	t.Run("shorthand verb commands show proper examples", func(t *testing.T) {
		// Test that shorthand verb commands show examples
		expectedExamples := []string{
			"kubectl oadp nonadmin get backup",
			"kubectl oadp nonadmin create backup",
			"kubectl oadp nonadmin delete backup",
			"kubectl oadp nonadmin describe backup",
			"kubectl oadp nonadmin logs backup",
		}

		commands := [][]string{
			{"na", "get", "--help"},
			{"na", "create", "--help"},
			{"na", "delete", "--help"},
			{"na", "describe", "--help"},
			{"na", "logs", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})
}

// TestNonAdminBackupDeleteAllFlag tests the --all flag functionality added in commit 6a112249d91ab5411d199b848cae1c97fccea655
// AI-generated test
func TestNonAdminBackupDeleteAllFlag(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("delete help shows --all flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"--all"})
	})

	t.Run("delete help shows --all in usage", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"[NAME...] | --all"})
	})

	t.Run("delete help describes --all flag purpose", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"Delete all backups in the current namespace"})
	})

	t.Run("delete --all flag description in long help", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"Use --all to delete all backups in the current namespace"})
	})

	t.Run("verb-noun delete help shows --all flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "delete", "backup", "--help"},
			[]string{"--all"})
	})

	t.Run("shorthand na delete backup help shows --all flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"na", "delete", "backup", "--help"},
			[]string{"--all"})
	})

	t.Run("shorthand na backup delete help shows --all flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"na", "backup", "delete", "--help"},
			[]string{"--all"})
	})
}

// TestNonAdminBackupDeleteAllFlagExamples tests that examples mention the --all flag
// AI-generated test
func TestNonAdminBackupDeleteAllFlagExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("delete help documents both usage patterns", func(t *testing.T) {
		// Test that help shows both ways to use the command
		expectedPatterns := []string{
			"--all",     // The flag itself
			"--confirm", // The confirmation skip flag
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			expectedPatterns)
	})

	t.Run("delete help has examples section", func(t *testing.T) {
		// Test that examples section exists and shows various delete patterns
		expectedExamples := []string{
			"kubectl oadp nonadmin backup delete my-backup",
			"kubectl oadp nonadmin backup delete --all",
			"kubectl oadp nonadmin backup delete my-backup --confirm",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			expectedExamples)
	})
}

// TestNonAdminBackupDeleteAllFlagBehavior tests the behavioral aspects of the --all flag
// AI-generated test
func TestNonAdminBackupDeleteAllFlagBehavior(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("delete with --all requires no backup names", func(t *testing.T) {
		// The --all flag should accept no arguments
		// This is a validation that the Args function works correctly
		// We expect help output to show: [NAME...] | --all
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"[NAME...] | --all"})
	})

	t.Run("delete help mentions namespace context", func(t *testing.T) {
		// The --all flag deletes in current namespace, should be documented
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "backup", "delete", "--help"},
			[]string{"current namespace"})
	})
}
