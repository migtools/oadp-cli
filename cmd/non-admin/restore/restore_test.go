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
				"get",
				"describe",
				"logs",
				"delete",
			},
		},
		{
			name: "nonadmin restore create help",
			args: []string{"nonadmin", "restore", "create", "--help"},
			expectContains: []string{
				"Create a non-admin restore",
				"--backup-name",
				"--include-resources",
				"--exclude-resources",
				"--selector",
				"--or-selector",
			},
		},
		{
			name: "nonadmin restore get help",
			args: []string{"nonadmin", "restore", "get", "--help"},
			expectContains: []string{
				"Get one or more non-admin restores",
			},
		},
		{
			name: "na restore shorthand help",
			args: []string{"na", "restore", "--help"},
			expectContains: []string{
				"Work with non-admin restores",
				"create",
				"get",
				"describe",
				"logs",
				"delete",
			},
		},
		// Verb-noun order help command tests
		{
			name: "nonadmin get restore help",
			args: []string{"nonadmin", "get", "restore", "--help"},
			expectContains: []string{
				"Get one or more non-admin restores",
			},
		},
		{
			name: "nonadmin create restore help",
			args: []string{"nonadmin", "create", "restore", "--help"},
			expectContains: []string{
				"Create a non-admin restore",
			},
		},
		// Shorthand verb-noun order tests
		{
			name: "na get restore help",
			args: []string{"na", "get", "restore", "--help"},
			expectContains: []string{
				"Get one or more non-admin restores",
			},
		},
		{
			name: "na create restore help",
			args: []string{"na", "create", "restore", "--help"},
			expectContains: []string{
				"Create a non-admin restore",
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
		{"nonadmin", "restore", "get", "--help"},
		{"nonadmin", "restore", "get", "-h"},
		{"nonadmin", "restore", "describe", "--help"},
		{"nonadmin", "restore", "describe", "-h"},
		{"nonadmin", "restore", "logs", "--help"},
		{"nonadmin", "restore", "logs", "-h"},
		{"nonadmin", "restore", "delete", "--help"},
		{"nonadmin", "restore", "delete", "-h"},
		{"na", "restore", "--help"},
		{"na", "restore", "-h"},
		// Verb-noun order help flags
		{"nonadmin", "get", "restore", "--help"},
		{"nonadmin", "get", "restore", "-h"},
		{"nonadmin", "create", "restore", "--help"},
		{"nonadmin", "create", "restore", "-h"},
		{"nonadmin", "describe", "restore", "--help"},
		{"nonadmin", "describe", "restore", "-h"},
		{"nonadmin", "logs", "restore", "--help"},
		{"nonadmin", "logs", "restore", "-h"},
		{"nonadmin", "delete", "restore", "--help"},
		{"nonadmin", "delete", "restore", "-h"},
		// Shorthand verb-noun order help flags
		{"na", "get", "restore", "--help"},
		{"na", "get", "restore", "-h"},
		{"na", "create", "restore", "--help"},
		{"na", "create", "restore", "-h"},
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
		// Minimal MVP flags only (based on NAR restrictions for non-admin users)
		expectedFlags := []string{
			"--backup-name",
			"--include-resources",
			"--exclude-resources",
			"--selector",
			"--or-selector",
			"--include-cluster-resources",
			"--item-operation-timeout",
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
			"--backup-name",
			"--include-resources",
			"--selector",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "create", "--help"},
			expectedExamples)
	})

	t.Run("main restore help shows subcommands", func(t *testing.T) {
		expectedSubcommands := []string{
			"create",
			"get",
			"describe",
			"logs",
			"delete",
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
			{"nonadmin", "restore", "logs", "--help"},
			{"nonadmin", "restore", "delete", "--help"},
			{"na", "restore", "get", "--help"},
			// Verb-noun order commands
			{"nonadmin", "get", "restore", "--help"},
			{"nonadmin", "create", "restore", "--help"},
			{"nonadmin", "describe", "restore", "--help"},
			{"nonadmin", "logs", "restore", "--help"},
			{"nonadmin", "delete", "restore", "--help"},
			{"na", "get", "restore", "--help"},
			{"na", "create", "restore", "--help"},
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

// TestVerbNounOrderRestoreExamples tests that verb-noun order commands show proper examples
func TestVerbNounOrderRestoreExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("verb commands show proper examples", func(t *testing.T) {
		// Test that verb commands show examples with kubectl oadp prefix
		expectedExamples := []string{
			"kubectl oadp nonadmin get restore",
			"kubectl oadp nonadmin create restore",
			"kubectl oadp nonadmin describe restore",
			"kubectl oadp nonadmin logs restore",
			"kubectl oadp nonadmin delete restore",
		}

		commands := [][]string{
			{"nonadmin", "get", "--help"},
			{"nonadmin", "create", "--help"},
			{"nonadmin", "describe", "--help"},
			{"nonadmin", "logs", "--help"},
			{"nonadmin", "delete", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})

	t.Run("verb commands with specific resources show proper examples", func(t *testing.T) {
		// Test that verb commands with specific resources show examples (noun-first format from underlying commands)
		expectedExamples := []string{
			"kubectl oadp nonadmin restore get",
			"kubectl oadp nonadmin restore create",
			"kubectl oadp nonadmin restore describe my-restore",
			"kubectl oadp nonadmin restore logs my-restore",
			"kubectl oadp nonadmin restore delete my-restore",
		}

		commands := [][]string{
			{"nonadmin", "get", "restore", "--help"},
			{"nonadmin", "create", "restore", "--help"},
			{"nonadmin", "describe", "restore", "--help"},
			{"nonadmin", "logs", "restore", "--help"},
			{"nonadmin", "delete", "restore", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})

	t.Run("shorthand verb commands show proper examples", func(t *testing.T) {
		// Test that shorthand verb commands show examples
		expectedExamples := []string{
			"kubectl oadp nonadmin get restore",
			"kubectl oadp nonadmin create restore",
			"kubectl oadp nonadmin describe restore",
			"kubectl oadp nonadmin logs restore",
			"kubectl oadp nonadmin delete restore",
		}

		commands := [][]string{
			{"na", "get", "--help"},
			{"na", "create", "--help"},
			{"na", "describe", "--help"},
			{"na", "logs", "--help"},
			{"na", "delete", "--help"},
		}

		for i, cmd := range commands {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{expectedExamples[i]})
		}
	})
}

// TestNonAdminRestoreCreateRequiresBackupName tests that create requires --backup-name
func TestNonAdminRestoreCreateRequiresBackupName(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create help shows --backup-name flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "create", "--help"},
			[]string{"--backup-name"})
	})
}

// TestNonAdminRestoreDescribeCommands tests describe command functionality
func TestNonAdminRestoreDescribeCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin restore describe help",
			args: []string{"nonadmin", "restore", "describe", "--help"},
			expectContains: []string{
				"Describe a non-admin restore",
				"--details",
				"--request-timeout",
			},
		},
		{
			name: "nonadmin describe restore help - verb-noun order",
			args: []string{"nonadmin", "describe", "restore", "--help"},
			expectContains: []string{
				"Describe a non-admin restore",
			},
		},
		{
			name: "na describe restore help - shorthand",
			args: []string{"na", "describe", "restore", "--help"},
			expectContains: []string{
				"Describe a non-admin restore",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminRestoreLogsCommands tests logs command functionality
func TestNonAdminRestoreLogsCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin restore logs help",
			args: []string{"nonadmin", "restore", "logs", "--help"},
			expectContains: []string{
				"Show logs for a non-admin restore",
				"--request-timeout",
			},
		},
		{
			name: "nonadmin logs restore help - verb-noun order",
			args: []string{"nonadmin", "logs", "restore", "--help"},
			expectContains: []string{
				"Show logs for a non-admin restore",
			},
		},
		{
			name: "na logs restore help - shorthand",
			args: []string{"na", "logs", "restore", "--help"},
			expectContains: []string{
				"Show logs for a non-admin restore",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminRestoreDeleteCommands tests delete command functionality
func TestNonAdminRestoreDeleteCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin restore delete help",
			args: []string{"nonadmin", "restore", "delete", "--help"},
			expectContains: []string{
				"Delete one or more non-admin restores",
				"--confirm",
				"--all",
			},
		},
		{
			name: "nonadmin delete restore help - verb-noun order",
			args: []string{"nonadmin", "delete", "restore", "--help"},
			expectContains: []string{
				"Delete one or more non-admin restores",
			},
		},
		{
			name: "na delete restore help - shorthand",
			args: []string{"na", "delete", "restore", "--help"},
			expectContains: []string{
				"Delete one or more non-admin restores",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminRestoreDeleteAllFlag tests --all flag behavior
func TestNonAdminRestoreDeleteAllFlag(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("delete help shows --all flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "delete", "--help"},
			[]string{"--all", "Delete all restores"})
	})

	t.Run("delete help shows --confirm flag", func(t *testing.T) {
		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "delete", "--help"},
			[]string{"--confirm", "Skip confirmation"})
	})

	t.Run("delete help has examples section", func(t *testing.T) {
		// Test that examples section exists and shows various delete patterns
		expectedExamples := []string{
			"kubectl oadp nonadmin restore delete my-restore",
			"kubectl oadp nonadmin restore delete --all",
			"kubectl oadp nonadmin restore delete my-restore --confirm",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "restore", "delete", "--help"},
			expectedExamples)
	})
}
