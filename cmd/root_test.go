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

package cmd

import (
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestRootCommand tests the root command functionality
func TestRootCommand(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name: "root help",
			args: []string{"--help"},
			expectContains: []string{
				"OADP CLI commands",
				"Available Commands:",
				"version",
				"backup",
				"restore",
				"nabsl",
				"nonadmin",
			},
		},
		{
			name: "root help short",
			args: []string{"-h"},
			expectContains: []string{
				"OADP CLI commands",
				"Available Commands:",
			},
		},
		{
			name: "version help",
			args: []string{"version", "--help"},
			expectContains: []string{
				"Print the velero version and associated image",
			},
		},
		{
			name: "backup help",
			args: []string{"backup", "--help"},
			expectContains: []string{
				"Work with backups",
			},
		},
		{
			name: "restore help",
			args: []string{"restore", "--help"},
			expectContains: []string{
				"Work with restores",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestRootCommandHelpFlags tests that both --help and -h work consistently
func TestRootCommandHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	commands := [][]string{
		{"--help"},
		{"-h"},
		{"backup", "--help"},
		{"backup", "-h"},
		{"restore", "--help"},
		{"restore", "-h"},
		{"version", "--help"},
		{"version", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}

// TestRootCommandSmoke performs basic smoke tests
func TestRootCommandSmoke(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	smokeCommands := [][]string{
		{"--help"},
		{"-h"},
		{"backup", "--help"},
		{"restore", "--help"},
		{"version", "--help"},
	}

	for _, cmd := range smokeCommands {
		t.Run("smoke_"+cmd[0], func(t *testing.T) {
			// Just verify commands don't crash
			_, _ = testutil.RunCommand(t, binaryPath, cmd...)
		})
	}
}
