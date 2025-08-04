package tests

import "testing"

// TestCLIHelpCommands tests all help commands - this is the baseline test suite
// These tests verify that all command paths have working help documentation
func TestCLIHelpCommands(t *testing.T) {
	// Build the binary first
	binaryPath := buildCLIBinary(t)
	defer cleanup(t, binaryPath)

	// Define all command paths to test
	testCases := []struct {
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
			name: "nonadmin bsl help",
			args: []string{"nonadmin", "bsl", "--help"},
			expectContains: []string{
				"Create and manage non-admin backup storage locations",
				"create",
				"request",
			},
		},
		{
			name: "nonadmin bsl create help",
			args: []string{"nonadmin", "bsl", "create", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
				"--provider",
				"--bucket",
				"--credential-name",
			},
		},
		{
			name: "nonadmin bsl request help",
			args: []string{"nonadmin", "bsl", "request", "--help"},
			expectContains: []string{
				"View and manage approval requests",
				"approve",
				"deny",
				"describe",
				"get",
			},
		},
		{
			name: "nonadmin bsl request approve help",
			args: []string{"nonadmin", "bsl", "request", "approve", "--help"},
			expectContains: []string{
				"Approve a pending backup storage location request",
				"--reason",
			},
		},
		{
			name: "nonadmin bsl request deny help",
			args: []string{"nonadmin", "bsl", "request", "deny", "--help"},
			expectContains: []string{
				"Deny a pending backup storage location request",
				"--reason",
			},
		},
		{
			name: "nonadmin bsl request get help",
			args: []string{"nonadmin", "bsl", "request", "get", "--help"},
			expectContains: []string{
				"Get non-admin backup storage location requests",
			},
		},
		{
			name: "nonadmin bsl request describe help",
			args: []string{"nonadmin", "bsl", "request", "describe", "--help"},
			expectContains: []string{
				"Describe a non-admin backup storage location request",
			},
		},
	}

	// Run tests for each command path
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testHelpCommand(t, binaryPath, tc.args, tc.expectContains)
		})
	}
}

// TestCLIHelpFlags tests that both --help and -h work consistently
func TestCLIHelpFlags(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer cleanup(t, binaryPath)

	// Test both flags produce similar output
	commands := [][]string{
		{"--help"},
		{"-h"},
		{"nonadmin", "--help"},
		{"nonadmin", "-h"},
		{"backup", "--help"},
		{"backup", "-h"},
		{"nonadmin", "bsl", "--help"},
		{"nonadmin", "bsl", "-h"},
		{"nonadmin", "bsl", "request", "--help"},
		{"nonadmin", "bsl", "request", "-h"},
		{"nonadmin", "bsl", "request", "approve", "--help"},
		{"nonadmin", "bsl", "request", "approve", "-h"},
		{"nonadmin", "bsl", "request", "deny", "--help"},
		{"nonadmin", "bsl", "request", "deny", "-h"},
	}

	for _, cmd := range commands {
		t.Run("help_flags_"+cmd[len(cmd)-1], func(t *testing.T) {
			testHelpCommand(t, binaryPath, cmd, []string{"Usage:"})
		})
	}
}
