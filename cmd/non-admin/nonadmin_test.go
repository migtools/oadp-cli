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

package nonadmin

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestNonAdminCommands tests the non-admin command functionality
func TestNonAdminCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "nonadmin help",
			args: []string{"nonadmin", "--help"},
			expectContains: []string{
				"Work with non-admin resources",
				"Work with non-admin resources like backups",
				"backup",
				"bsl",
			},
		},
		{
			name: "nonadmin backup help",
			args: []string{"nonadmin", "backup", "--help"},
			expectContains: []string{
				"Work with non-admin backups",
				"create",
			},
		},
		{
			name: "nonadmin backup create help",
			args: []string{"nonadmin", "backup", "create", "--help"},
			expectContains: []string{
				"Create a non-admin backup",
			},
		},
		{
			name: "nonadmin backup create help",
			args: []string{"nonadmin", "create", "backup", "--help"},
			expectContains: []string{
				"Create a non-admin backup",
			},
		},
		// Verb-noun order help command tests
		{
			name: "nonadmin get help",
			args: []string{"nonadmin", "get", "--help"},
			expectContains: []string{
				"Get one or more non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin create help",
			args: []string{"nonadmin", "create", "--help"},
			expectContains: []string{
				"Create non-admin resources",
				"backup",
				"bsl",
			},
		},
		{
			name: "nonadmin delete help",
			args: []string{"nonadmin", "delete", "--help"},
			expectContains: []string{
				"Delete non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin describe help",
			args: []string{"nonadmin", "describe", "--help"},
			expectContains: []string{
				"Describe non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin logs help",
			args: []string{"nonadmin", "logs", "--help"},
			expectContains: []string{
				"Get logs for non-admin resources",
				"backup",
			},
		},
		// Verb-noun order with specific resources
		{
			name: "nonadmin get backup help",
			args: []string{"nonadmin", "get", "backup", "--help"},
			expectContains: []string{
				"Get one or more non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin create backup help",
			args: []string{"nonadmin", "create", "backup", "--help"},
			expectContains: []string{
				"Create non-admin resources",
				"backup",
				"bsl",
			},
		},
		{
			name: "nonadmin delete backup help",
			args: []string{"nonadmin", "delete", "backup", "--help"},
			expectContains: []string{
				"Delete non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin describe backup help",
			args: []string{"nonadmin", "describe", "backup", "--help"},
			expectContains: []string{
				"Describe non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin logs backup help",
			args: []string{"nonadmin", "logs", "backup", "--help"},
			expectContains: []string{
				"Get logs for non-admin resources",
				"backup",
			},
		},
		{
			name: "nonadmin create bsl help",
			args: []string{"nonadmin", "create", "bsl", "--help"},
			expectContains: []string{
				"Create non-admin resources",
				"backup",
				"bsl",
			},
		},
		// Shorthand tests for verb-noun order
		{
			name: "na get help",
			args: []string{"na", "get", "--help"},
			expectContains: []string{
				"Get one or more non-admin resources",
				"backup",
			},
		},
		{
			name: "na create backup help",
			args: []string{"na", "create", "backup", "--help"},
			expectContains: []string{
				"Create non-admin resources",
				"backup",
				"bsl",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestNonAdminHelpFlags tests that both --help and -h work for non-admin commands
func TestNonAdminHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"nonadmin", "--help"},
		{"nonadmin", "-h"},
		{"nonadmin", "backup", "--help"},
		{"nonadmin", "backup", "-h"},
		{"nonadmin", "bsl", "--help"},
		{"nonadmin", "bsl", "-h"},
		// Verb-noun order help flags
		{"nonadmin", "get", "--help"},
		{"nonadmin", "get", "-h"},
		{"nonadmin", "create", "--help"},
		{"nonadmin", "create", "-h"},
		{"nonadmin", "delete", "--help"},
		{"nonadmin", "delete", "-h"},
		{"nonadmin", "describe", "--help"},
		{"nonadmin", "describe", "-h"},
		{"nonadmin", "logs", "--help"},
		{"nonadmin", "logs", "-h"},
		{"nonadmin", "get", "backup", "--help"},
		{"nonadmin", "get", "backup", "-h"},
		{"nonadmin", "create", "backup", "--help"},
		{"nonadmin", "create", "backup", "-h"},
		{"nonadmin", "create", "bsl", "--help"},
		{"nonadmin", "create", "bsl", "-h"},
		// Shorthand verb-noun order help flags
		{"na", "get", "--help"},
		{"na", "get", "-h"},
		{"na", "create", "--help"},
		{"na", "create", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}
