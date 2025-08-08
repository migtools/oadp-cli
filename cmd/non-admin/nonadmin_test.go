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
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}
