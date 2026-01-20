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
	"strings"
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestBSLCommands tests the BSL command functionality
func TestBSLCommands(t *testing.T) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}

// TestBSLNoLongerHasRequestCommands verifies that request commands were moved to nabsl-request
func TestBSLNoLongerHasRequestCommands(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("nonadmin bsl no longer has request", func(t *testing.T) {
		output, err := testutil.RunCommand(t, binaryPath, "nonadmin", "bsl", "--help")
		if err != nil {
			t.Fatalf("nonadmin bsl command should exist: %v", err)
		}

		// Should have create but not request
		if !strings.Contains(output, "create") {
			t.Errorf("Expected nonadmin bsl to still have create command")
		}

		// Check that "request" doesn't appear as a subcommand in Available Commands section
		lines := strings.Split(output, "\n")
		inAvailableCommands := false
		for _, line := range lines {
			if strings.Contains(line, "Available Commands:") {
				inAvailableCommands = true
				continue
			}
			if inAvailableCommands && strings.Contains(line, "Flags:") {
				break
			}
			if inAvailableCommands && strings.Contains(strings.TrimSpace(line), "request") {
				t.Errorf("Expected nonadmin bsl to NOT have request subcommand anymore, but found: %s", strings.TrimSpace(line))
			}
		}
	})
}

// TestBSLCreateUsesNewCredentialFlag verifies the credential flag format
func TestBSLCreateUsesNewCredentialFlag(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("nonadmin bsl create uses new credential flag", func(t *testing.T) {
		output, err := testutil.RunCommand(t, binaryPath, "nonadmin", "bsl", "create", "--help")
		if err != nil {
			t.Fatalf("nonadmin bsl create command should exist: %v", err)
		}

		// Should use --credential not --credential-name
		if !strings.Contains(output, "--credential") {
			t.Errorf("Expected nonadmin bsl create to have --credential flag")
		}
		if strings.Contains(output, "--credential-name") {
			t.Errorf("Expected nonadmin bsl create to NOT have --credential-name flag anymore")
		}
	})
}

// TestBSLVerbNounOrderFlagPassing tests that flags are properly passed through
// when using verb-noun order (e.g., "create bsl" instead of "bsl create")
func TestBSLVerbNounOrderFlagPassing(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create bsl recognizes credential flag", func(t *testing.T) {
		// This test verifies that the credential flag is properly recognized
		// in verb-noun order. The command should fail with validation errors
		// if flags are missing, but should NOT fail with "--credential is required"
		// when the credential flag is provided.
		args := []string{"nonadmin", "create", "bsl", "test-bsl",
			"--provider", "aws",
			"--bucket", "test-bucket",
			"--credential", "cloud-credentials=cloud",
			"--region", "us-east-1",
		}

		output, err := testutil.RunCommand(t, binaryPath, args...)

		// Should NOT fail with "--credential is required" error
		// (it will fail for other reasons like missing cluster, but not validation)
		if strings.Contains(output, "--credential is required") {
			t.Errorf("Credential flag was not recognized in verb-noun order.\nOutput:\n%s", output)
		}

		// Should also NOT fail with other validation errors since all required flags are provided
		validationErrors := []string{
			"--provider is required",
			"--bucket is required",
		}
		for _, validationError := range validationErrors {
			if strings.Contains(output, validationError) {
				t.Errorf("Flag validation failed, flag was not recognized: %s\nOutput:\n%s", validationError, output)
			}
		}

		// Command will error (no cluster), but should not be a validation error
		if err != nil {
			t.Logf("Command errored as expected (no cluster): %s", output)
		}
	})

	t.Run("create bsl with missing credential shows validation error", func(t *testing.T) {
		// Verify that validation still works - if credential is missing, it should error
		args := []string{"nonadmin", "create", "bsl", "test-bsl",
			"--provider", "aws",
			"--bucket", "test-bucket",
			"--region", "us-east-1",
		}

		output, err := testutil.RunCommand(t, binaryPath, args...)

		if err == nil {
			t.Errorf("Expected command to error when credential is missing, but it succeeded.\nOutput:\n%s", output)
		}

		if !strings.Contains(output, "--credential is required") {
			t.Errorf("Expected validation error for missing credential flag, but didn't get it.\nOutput:\n%s", output)
		}
	})
}
