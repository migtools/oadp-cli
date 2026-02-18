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

package bsl

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestNonAdminBSLCommands tests the non-admin BSL command functionality
func TestNonAdminBSLCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin bsl help",
			args: []string{"nonadmin", "bsl", "--help"},
			expectContains: []string{
				"Create and manage non-admin backup storage locations",
				"create",
				"get",
			},
		},
		{
			name: "nonadmin bsl create help",
			args: []string{"nonadmin", "bsl", "create", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
				"--provider",
				"--bucket",
				"--credential",
				"--region",
				"--prefix",
			},
		},
		{
			name: "nonadmin bsl get help",
			args: []string{"nonadmin", "bsl", "get", "--help"},
			expectContains: []string{
				"Get one or more non-admin backup storage locations",
			},
		},
		{
			name: "na bsl shorthand help",
			args: []string{"na", "bsl", "--help"},
			expectContains: []string{
				"Create and manage non-admin backup storage locations",
				"create",
				"get",
			},
		},
		// Verb-noun order help command tests
		{
			name: "nonadmin get bsl help",
			args: []string{"nonadmin", "get", "bsl", "--help"},
			expectContains: []string{
				"Get one or more non-admin backup storage locations",
			},
		},
		{
			name: "nonadmin create bsl help",
			args: []string{"nonadmin", "create", "bsl", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
			},
		},
		// Shorthand verb-noun order tests
		{
			name: "na get bsl help",
			args: []string{"na", "get", "bsl", "--help"},
			expectContains: []string{
				"Get one or more non-admin backup storage locations",
			},
		},
		{
			name: "na create bsl help",
			args: []string{"na", "create", "bsl", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminBSLHelpFlags tests that both --help and -h work for BSL commands
func TestNonAdminBSLHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"nonadmin", "bsl", "--help"},
		{"nonadmin", "bsl", "-h"},
		{"nonadmin", "bsl", "create", "--help"},
		{"nonadmin", "bsl", "create", "-h"},
		{"nonadmin", "bsl", "get", "--help"},
		{"nonadmin", "bsl", "get", "-h"},
		{"na", "bsl", "--help"},
		{"na", "bsl", "-h"},
		// Verb-noun order help flags
		{"nonadmin", "get", "bsl", "--help"},
		{"nonadmin", "get", "bsl", "-h"},
		{"nonadmin", "create", "bsl", "--help"},
		{"nonadmin", "create", "bsl", "-h"},
		// Shorthand verb-noun order help flags
		{"na", "get", "bsl", "--help"},
		{"na", "get", "bsl", "-h"},
		{"na", "create", "bsl", "--help"},
		{"na", "create", "bsl", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}

// TestNonAdminBSLCreateFlags tests create command specific flags
func TestNonAdminBSLCreateFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create command has all expected flags", func(t *testing.T) {
		expectedFlags := []string{
			"--provider",
			"--bucket",
			"--credential",
			"--region",
			"--prefix",
			"--config",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "create", "--help"},
			expectedFlags)
	})
}

// TestNonAdminBSLExamples tests that help text contains proper examples
func TestNonAdminBSLExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create examples use correct command format", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin bsl create",
			"--provider",
			"--bucket",
			"--credential",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "create", "--help"},
			expectedExamples)
	})

	t.Run("get examples use correct command format", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin bsl get",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "get", "--help"},
			expectedExamples)
	})

	t.Run("main bsl help shows subcommands", func(t *testing.T) {
		expectedSubcommands := []string{
			"create",
			"get",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "--help"},
			expectedSubcommands)
	})
}

// TestNonAdminBSLClientConfigIntegration tests that BSL commands respect client config
func TestNonAdminBSLClientConfigIntegration(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)
	_, cleanup := testutil.SetupTempHome(t)
	defer cleanup()

	t.Run("bsl commands work with client config", func(t *testing.T) {
		// Set a known namespace
		_, err := testutil.RunCommand(t, binaryPath, "client", "config", "set", "namespace=user-namespace")
		if err != nil {
			t.Fatalf("Failed to set client config: %v", err)
		}

		// Test that BSL commands can be invoked (they should respect the namespace)
		// We test help commands since they don't require actual K8s resources
		commands := [][]string{
			{"nonadmin", "bsl", "get", "--help"},
			{"nonadmin", "bsl", "create", "--help"},
			{"na", "bsl", "get", "--help"},
			// Verb-noun order commands
			{"nonadmin", "get", "bsl", "--help"},
			{"nonadmin", "create", "bsl", "--help"},
			{"na", "get", "bsl", "--help"},
			{"na", "create", "bsl", "--help"},
		}

		for _, cmd := range commands {
			t.Run("config_test_"+cmd[len(cmd)-2], func(t *testing.T) {
				output, err := testutil.RunCommand(t, binaryPath, cmd...)
				if err != nil {
					t.Fatalf("Non-admin BSL command should work with client config: %v", err)
				}
				if output == "" {
					t.Errorf("Expected help output for %v", cmd)
				}
			})
		}
	})
}

// TestNonAdminBSLCommandStructure tests the overall command structure
func TestNonAdminBSLCommandStructure(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("bsl commands available under nonadmin", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "nonadmin", "--help")
		if err != nil {
			t.Fatalf("nonadmin command should exist: %v", err)
		}

		expectedCommands := []string{"bsl"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"nonadmin", "--help"}, []string{cmd})
		}
	})

	t.Run("bsl commands available under na shorthand", func(t *testing.T) {
		_, err := testutil.RunCommand(t, binaryPath, "na", "--help")
		if err != nil {
			t.Fatalf("na command should exist: %v", err)
		}

		expectedCommands := []string{"bsl"}
		for _, cmd := range expectedCommands {
			testutil.TestHelpCommand(t, binaryPath, []string{"na", "--help"}, []string{cmd})
		}
	})
}

// TestVerbNounOrderBSLExamples tests that verb-noun order commands show proper BSL examples
func TestVerbNounOrderBSLExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("get verb command shows bsl examples", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin get bsl",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "get", "--help"},
			expectedExamples)
	})

	t.Run("create verb command shows bsl examples", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin create bsl",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "create", "--help"},
			expectedExamples)
	})

	t.Run("get bsl with specific resource shows proper examples", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin bsl get", // Shows noun-first format from underlying command
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "get", "bsl", "--help"},
			expectedExamples)
	})

	t.Run("create bsl with specific resource shows proper examples", func(t *testing.T) {
		expectedExamples := []string{
			"oc oadp nonadmin bsl create", // Shows noun-first format from underlying command
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "create", "bsl", "--help"},
			expectedExamples)
	})
}

// TestNonAdminBSLOutputFormat tests that help text uses correct command format
func TestNonAdminBSLOutputFormat(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("usage shows oc oadp prefix", func(t *testing.T) {
		expectedStrings := []string{
			"oc oadp nonadmin bsl",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "--help"},
			expectedStrings)
	})

	t.Run("create usage shows oc oadp prefix", func(t *testing.T) {
		expectedStrings := []string{
			"oc oadp nonadmin bsl create",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "create", "--help"},
			expectedStrings)
	})

	t.Run("get usage shows oc oadp prefix", func(t *testing.T) {
		expectedStrings := []string{
			"oc oadp nonadmin bsl get",
		}

		testutil.TestHelpCommand(t, binaryPath,
			[]string{"nonadmin", "bsl", "get", "--help"},
			expectedStrings)
	})
}
