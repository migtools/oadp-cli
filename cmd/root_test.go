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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/migtools/oadp-cli/internal/testutil"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
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

// TestReplaceVeleroWithOADP_BasicReplacement tests basic Example field replacement
func TestReplaceVeleroWithOADP_BasicReplacement(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "test",
		Example: "velero backup create my-backup",
	}

	replaceVeleroWithOADP(cmd)

	if strings.Contains(cmd.Example, "velero") {
		t.Errorf("Expected 'velero' to be replaced in Example, got: %s", cmd.Example)
	}
	if !strings.Contains(cmd.Example, "oc oadp") {
		t.Errorf("Expected 'oc oadp' in Example, got: %s", cmd.Example)
	}
	expected := "oc oadp backup create my-backup"
	if cmd.Example != expected {
		t.Errorf("Expected Example to be %q, got %q", expected, cmd.Example)
	}
}

// TestReplaceVeleroWithOADP_RecursiveReplacement tests recursive child command replacement
func TestReplaceVeleroWithOADP_RecursiveReplacement(t *testing.T) {
	parent := &cobra.Command{
		Use:     "parent",
		Example: "velero backup get",
	}
	child := &cobra.Command{
		Use:     "child",
		Example: "velero backup create test",
	}
	grandchild := &cobra.Command{
		Use:     "grandchild",
		Example: "velero restore describe my-restore",
	}

	child.AddCommand(grandchild)
	parent.AddCommand(child)

	replaceVeleroWithOADP(parent)

	// Check all levels were replaced
	if strings.Contains(parent.Example, "velero") {
		t.Errorf("Parent Example still contains 'velero': %s", parent.Example)
	}
	if strings.Contains(child.Example, "velero") {
		t.Errorf("Child Example still contains 'velero': %s", child.Example)
	}
	if strings.Contains(grandchild.Example, "velero") {
		t.Errorf("Grandchild Example still contains 'velero': %s", grandchild.Example)
	}

	// Verify replacement happened
	if !strings.Contains(parent.Example, "oc oadp") {
		t.Errorf("Parent Example doesn't contain 'oc oadp': %s", parent.Example)
	}
	if !strings.Contains(child.Example, "oc oadp") {
		t.Errorf("Child Example doesn't contain 'oc oadp': %s", child.Example)
	}
	if !strings.Contains(grandchild.Example, "oc oadp") {
		t.Errorf("Grandchild Example doesn't contain 'oc oadp': %s", grandchild.Example)
	}
}

// TestReplaceVeleroWithOADP_MultipleOccurrences tests replacing multiple occurrences
func TestReplaceVeleroWithOADP_MultipleOccurrences(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		Example: `velero backup create my-backup
velero backup get my-backup
Use velero backup logs to check status`,
	}

	replaceVeleroWithOADP(cmd)

	if strings.Contains(cmd.Example, "velero") {
		t.Errorf("Example still contains 'velero': %s", cmd.Example)
	}

	// Count occurrences of "oc oadp"
	count := strings.Count(cmd.Example, "oc oadp")
	if count != 3 {
		t.Errorf("Expected 3 occurrences of 'oc oadp', got %d\nActual output:\n%s", count, cmd.Example)
	}
}

// TestReplaceVeleroWithOADP_RunFunctionWrapper tests stdout capture and replacement
func TestReplaceVeleroWithOADP_RunFunctionWrapper(t *testing.T) {
	outputCaptured := false
	cmd := &cobra.Command{
		Use: "test",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Run `velero backup describe test` for details.")
			fmt.Println("Or use velero backup logs test")
			outputCaptured = true
		},
	}

	replaceVeleroWithOADP(cmd)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the wrapped command
	cmd.Run(cmd, []string{})

	// Restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Errorf("Error copying output: %v", err)
	}
	output := buf.String()

	if !outputCaptured {
		t.Error("Original Run function was not executed")
	}

	if strings.Contains(output, "velero backup") {
		t.Errorf("Output still contains 'velero backup': %s", output)
	}

	if !strings.Contains(output, "oc oadp") {
		t.Errorf("Output doesn't contain 'oc oadp': %s", output)
	}

	// Verify both lines were replaced
	if !strings.Contains(output, "oc oadp backup describe") {
		t.Errorf("First line not properly replaced: %s", output)
	}
	if !strings.Contains(output, "oc oadp backup logs") {
		t.Errorf("Second line not properly replaced: %s", output)
	}
}

// TestReplaceVeleroWithOADP_EmptyFields tests handling of empty fields
func TestReplaceVeleroWithOADP_EmptyFields(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "test",
		Example: "",
	}

	// Should not panic
	replaceVeleroWithOADP(cmd)

	if cmd.Example != "" {
		t.Errorf("Expected empty Example to remain empty, got: %s", cmd.Example)
	}
}

// TestReplaceVeleroWithOADP_NilRun tests handling of nil Run function
func TestReplaceVeleroWithOADP_NilRun(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "test",
		Example: "velero test",
		Run:     nil,
	}

	// Should not panic
	replaceVeleroWithOADP(cmd)

	if cmd.Run != nil {
		t.Error("Expected Run to remain nil")
	}
}

// TestReplaceVeleroWithOADP_PreservesOtherFields tests that other fields are not affected
func TestReplaceVeleroWithOADP_PreservesOtherFields(t *testing.T) {
	originalShort := "Short description"
	originalLong := "Long description"
	originalUse := "test-command"

	cmd := &cobra.Command{
		Use:     originalUse,
		Short:   originalShort,
		Long:    originalLong,
		Example: "velero backup create",
	}

	replaceVeleroWithOADP(cmd)

	if cmd.Use != originalUse {
		t.Errorf("Use field was modified: expected %q, got %q", originalUse, cmd.Use)
	}
	if cmd.Short != originalShort {
		t.Errorf("Short field was modified: expected %q, got %q", originalShort, cmd.Short)
	}
	if cmd.Long != originalLong {
		t.Errorf("Long field was modified: expected %q, got %q", originalLong, cmd.Long)
	}
}

// TestReplaceVeleroWithOADP_CaseSensitive tests that replacement is case-sensitive
func TestReplaceVeleroWithOADP_CaseSensitive(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "test",
		Example: "Velero backup create\nVELERO backup get\nvelero backup describe",
	}

	replaceVeleroWithOADP(cmd)

	// Only lowercase "velero" should be replaced
	if !strings.Contains(cmd.Example, "Velero") {
		t.Errorf("Expected 'Velero' (capitalized) to remain, got: %s", cmd.Example)
	}
	if !strings.Contains(cmd.Example, "VELERO") {
		t.Errorf("Expected 'VELERO' (uppercase) to remain, got: %s", cmd.Example)
	}
	if strings.Contains(cmd.Example, "velero backup describe") {
		t.Errorf("Expected lowercase 'velero' to be replaced, got: %s", cmd.Example)
	}
	if !strings.Contains(cmd.Example, "oc oadp backup describe") {
		t.Errorf("Expected 'oc oadp backup describe' after replacement, got: %s", cmd.Example)
	}
}

// TestReplaceVeleroWithOADP_PreservesProperNouns tests that "velero" referring to the project is preserved
func TestReplaceVeleroWithOADP_PreservesProperNouns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "velero server reference",
			input:    "This starts the velero server",
			expected: "This starts the velero server",
		},
		{
			name:     "about velero",
			input:    "Learn more about velero at velero.io",
			expected: "Learn more about velero at velero.io",
		},
		{
			name:     "velero project",
			input:    "The velero project provides backup capabilities",
			expected: "The velero project provides backup capabilities",
		},
		{
			name:     "mixed - command and reference",
			input:    "Run velero backup create to use the velero backup feature",
			expected: "Run oc oadp backup create to use the velero backup feature",
		},
		{
			name:     "velero namespace",
			input:    "Resources are in the velero namespace",
			expected: "Resources are in the velero namespace",
		},
		{
			name:     "command at start of line",
			input:    "velero backup get my-backup",
			expected: "oc oadp backup get my-backup",
		},
		{
			name:     "command after backtick",
			input:    "Run `velero backup logs` for details",
			expected: "Run `oc oadp backup logs` for details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:     "test",
				Example: tt.input,
			}

			replaceVeleroWithOADP(cmd)

			if cmd.Example != tt.expected {
				t.Errorf("Expected: %q\nGot:      %q", tt.expected, cmd.Example)
			}
		})
	}
}

// TestReplaceVeleroWithOADP_RunOutputPreservesProperNouns tests Run wrapper preserves "velero" references
func TestReplaceVeleroWithOADP_RunOutputPreservesProperNouns(t *testing.T) {
	tests := []struct {
		name             string
		outputFunc       func()
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "server reference preserved",
			outputFunc: func() {
				fmt.Println("The velero server is running")
			},
			shouldContain:    []string{"velero server"},
			shouldNotContain: []string{"oadp server"},
		},
		{
			name: "command replaced",
			outputFunc: func() {
				fmt.Println("Run `velero backup describe test` for details")
			},
			shouldContain:    []string{"oc oadp backup describe"},
			shouldNotContain: []string{"velero backup describe"},
		},
		{
			name: "mixed content",
			outputFunc: func() {
				fmt.Println("Use velero backup create to backup using the velero backup controller")
			},
			shouldContain:    []string{"oc oadp backup create", "velero backup controller"},
			shouldNotContain: []string{"velero backup create", "oc oadp backup controller"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "test",
				Run: func(c *cobra.Command, args []string) {
					tt.outputFunc()
				},
			}

			replaceVeleroWithOADP(cmd)

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the wrapped command
			cmd.Run(cmd, []string{})

			// Restore stdout
			_ = w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			_, err := io.Copy(&buf, r)
			if err != nil {
				t.Errorf("Error copying output: %v", err)
			}
			output := buf.String()

			for _, should := range tt.shouldContain {
				if !strings.Contains(output, should) {
					t.Errorf("Expected output to contain %q, got: %s", should, output)
				}
			}

			for _, shouldNot := range tt.shouldNotContain {
				if strings.Contains(output, shouldNot) {
					t.Errorf("Expected output NOT to contain %q, got: %s", shouldNot, output)
				}
			}
		})
	}
}

// TestGlobalRequestTimeout tests the thread-safe global timeout get/set functions
func TestGlobalRequestTimeout(t *testing.T) {
	// Reset to zero at start
	setGlobalRequestTimeout(0)

	// Test initial value is zero
	if got := getGlobalRequestTimeout(); got != 0 {
		t.Errorf("Expected initial timeout to be 0, got %v", got)
	}

	// Test setting a value
	expected := 5 * time.Second
	setGlobalRequestTimeout(expected)
	if got := getGlobalRequestTimeout(); got != expected {
		t.Errorf("Expected timeout to be %v, got %v", expected, got)
	}

	// Test setting another value
	expected = 30 * time.Second
	setGlobalRequestTimeout(expected)
	if got := getGlobalRequestTimeout(); got != expected {
		t.Errorf("Expected timeout to be %v, got %v", expected, got)
	}

	// Reset after test
	setGlobalRequestTimeout(0)
}

// TestApplyTimeoutToConfig tests that applyTimeoutToConfig correctly sets timeout on REST config
func TestApplyTimeoutToConfig(t *testing.T) {
	tests := []struct {
		name          string
		globalTimeout time.Duration
		expectTimeout bool
		expectDialer  bool
	}{
		{
			name:          "zero timeout does not modify config",
			globalTimeout: 0,
			expectTimeout: false,
			expectDialer:  false,
		},
		{
			name:          "positive timeout sets config timeout and dialer",
			globalTimeout: 10 * time.Second,
			expectTimeout: true,
			expectDialer:  true,
		},
		{
			name:          "1 second timeout",
			globalTimeout: 1 * time.Second,
			expectTimeout: true,
			expectDialer:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global timeout
			setGlobalRequestTimeout(tt.globalTimeout)
			defer setGlobalRequestTimeout(0)

			// Create a config
			config := &rest.Config{
				Host: "https://test-cluster:6443",
			}

			// Apply timeout
			applyTimeoutToConfig(config)

			// Check timeout
			if tt.expectTimeout {
				if config.Timeout != tt.globalTimeout {
					t.Errorf("Expected config.Timeout to be %v, got %v", tt.globalTimeout, config.Timeout)
				}
			} else {
				if config.Timeout != 0 {
					t.Errorf("Expected config.Timeout to be 0, got %v", config.Timeout)
				}
			}

			// Check dialer
			if tt.expectDialer {
				if config.Dial == nil {
					t.Error("Expected config.Dial to be set, but it was nil")
				}
			} else {
				if config.Dial != nil {
					t.Error("Expected config.Dial to be nil, but it was set")
				}
			}
		})
	}
}

// TestReplaceVeleroWithOADP_OutputWrapperExclusions tests that certain commands are excluded from output wrapping
func TestReplaceVeleroWithOADP_OutputWrapperExclusions(t *testing.T) {
	tests := []struct {
		name       string
		use        string
		shouldWrap bool
	}{
		{
			name:       "logs command",
			use:        "logs",
			shouldWrap: false,
		},
		{
			name:       "logs with args",
			use:        "logs NAME",
			shouldWrap: false,
		},
		{
			name:       "get command",
			use:        "get",
			shouldWrap: true,
		},
		{
			name:       "describe command",
			use:        "describe",
			shouldWrap: false,
		},
		{
			name:       "describe with args",
			use:        "describe NAME",
			shouldWrap: false,
		},
		{
			name:       "create command",
			use:        "create",
			shouldWrap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with Run function
			runCalled := false
			cmd := &cobra.Command{
				Use: tt.use,
				Run: func(c *cobra.Command, args []string) {
					runCalled = true
					fmt.Println("test output with velero backup create")
				},
			}

			// Store reference to original Run function
			originalRun := cmd.Run

			replaceVeleroWithOADP(cmd)

			// If logs command, Run should not be wrapped (same function pointer)
			// If not logs, Run should be wrapped (different function pointer)
			isWrapped := fmt.Sprintf("%p", originalRun) != fmt.Sprintf("%p", cmd.Run)

			if tt.shouldWrap && !isWrapped {
				t.Errorf("Expected command %q to be wrapped, but it wasn't", tt.use)
			}
			if !tt.shouldWrap && isWrapped {
				t.Errorf("Expected command %q NOT to be wrapped, but it was", tt.use)
			}

			// Verify the command still executes
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			cmd.Run(cmd, []string{})
			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("Error copying output: %v", err)
			}

			if !runCalled {
				t.Error("Original Run function was not called")
			}

			output := buf.String()
			if tt.shouldWrap {
				// Wrapped commands should have output replaced
				if strings.Contains(output, "velero backup create") {
					t.Errorf("Wrapped command output should have 'velero' replaced, got: %s", output)
				}
			} else {
				// Excluded commands should NOT have output replaced
				if !strings.Contains(output, "velero backup create") {
					t.Errorf("Excluded command output should NOT be modified, got: %s", output)
				}
			}
		})
	}

	// Test with RunE function
	t.Run("excluded_command_runE_not_wrapped", func(t *testing.T) {
		runECalled := false
		cmd := &cobra.Command{
			Use: "logs",
			RunE: func(c *cobra.Command, args []string) error {
				runECalled = true
				fmt.Println("test output with velero backup logs")
				return nil
			},
		}

		originalRunE := cmd.RunE
		replaceVeleroWithOADP(cmd)

		// Excluded command should not be wrapped
		isWrapped := fmt.Sprintf("%p", originalRunE) != fmt.Sprintf("%p", cmd.RunE)
		if isWrapped {
			t.Error("Expected excluded command RunE NOT to be wrapped, but it was")
		}

		// Verify output is not modified
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		err := cmd.RunE(cmd, []string{})
		_ = w.Close()
		os.Stdout = oldStdout

		if err != nil {
			t.Errorf("RunE returned error: %v", err)
		}

		if !runECalled {
			t.Error("Original RunE function was not called")
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatalf("Error copying output: %v", err)
		}
		output := buf.String()

		// Excluded command output should NOT be modified
		if !strings.Contains(output, "velero backup logs") {
			t.Errorf("Excluded command output should NOT be modified, got: %s", output)
		}
	})
}

// TestApplyTimeoutToConfig_DialerTimeout tests that the custom dialer respects the timeout
func TestApplyTimeoutToConfig_DialerTimeout(t *testing.T) {
	// Set a very short timeout
	timeout := 100 * time.Millisecond
	setGlobalRequestTimeout(timeout)
	defer setGlobalRequestTimeout(0)

	config := &rest.Config{
		Host: "https://test-cluster:6443",
	}

	applyTimeoutToConfig(config)

	if config.Dial == nil {
		t.Fatal("Expected config.Dial to be set")
	}

	// Test that the dialer times out quickly when connecting to a non-routable address
	// 10.255.255.1 is a non-routable IP that should cause a timeout
	ctx := context.Background()
	start := time.Now()
	_, err := config.Dial(ctx, "tcp", "10.255.255.1:6443")
	elapsed := time.Since(start)

	// Should get a timeout error
	if err == nil {
		t.Error("Expected dial to fail with timeout, but it succeeded")
	}

	// Check it's a timeout error
	if netErr, ok := err.(net.Error); ok {
		if !netErr.Timeout() {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	}

	// Should complete within a reasonable time of the timeout
	// Allow some margin for test execution overhead
	maxExpected := timeout + 500*time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("Dial took too long: %v (expected ~%v)", elapsed, timeout)
	}
}
