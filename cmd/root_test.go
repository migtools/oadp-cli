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
				"nabsl-request",
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
		// Verb-noun order help command tests
		{
			name: "get help",
			args: []string{"get", "--help"},
			expectContains: []string{
				"Get one or more resources",
				"backup",
				"restore",
			},
		},
		{
			name: "create help",
			args: []string{"create", "--help"},
			expectContains: []string{
				"Create a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "delete help",
			args: []string{"delete", "--help"},
			expectContains: []string{
				"Delete a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "describe help",
			args: []string{"describe", "--help"},
			expectContains: []string{
				"Describe a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "logs help",
			args: []string{"logs", "--help"},
			expectContains: []string{
				"Get logs for a resource",
				"backup",
				"restore",
			},
		},
		// Verb-noun order with specific resources
		{
			name: "get backup help",
			args: []string{"get", "backup", "--help"},
			expectContains: []string{
				"Get one or more resources",
				"backup",
				"restore",
			},
		},
		{
			name: "create backup help",
			args: []string{"create", "backup", "--help"},
			expectContains: []string{
				"Create a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "delete backup help",
			args: []string{"delete", "backup", "--help"},
			expectContains: []string{
				"Delete a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "describe backup help",
			args: []string{"describe", "backup", "--help"},
			expectContains: []string{
				"Describe a resource",
				"backup",
				"restore",
			},
		},
		{
			name: "logs backup help",
			args: []string{"logs", "backup", "--help"},
			expectContains: []string{
				"Get logs for a resource",
				"backup",
				"restore",
				"schedule",
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
		// Verb-noun order help flags
		{"get", "--help"},
		{"get", "-h"},
		{"create", "--help"},
		{"create", "-h"},
		{"delete", "--help"},
		{"delete", "-h"},
		{"describe", "--help"},
		{"describe", "-h"},
		{"logs", "--help"},
		{"logs", "-h"},
		{"get", "backup", "--help"},
		{"get", "backup", "-h"},
		{"create", "backup", "--help"},
		{"create", "backup", "-h"},
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
		// Verb-noun order smoke tests
		{"get", "--help"},
		{"create", "--help"},
		{"delete", "--help"},
		{"describe", "--help"},
		{"logs", "--help"},
		{"get", "backup", "--help"},
		{"create", "backup", "--help"},
	}

	for _, cmd := range smokeCommands {
		t.Run("smoke_"+cmd[0], func(t *testing.T) {
			// Just verify commands don't crash
			_, _ = testutil.RunCommand(t, binaryPath, cmd...)
		})
	}
}
