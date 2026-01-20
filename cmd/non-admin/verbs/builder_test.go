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

package verbs

import (
	"strings"
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestVerbNounOrderFlagPassing tests that flags are properly passed through
// when using verb-noun order (e.g., "create bsl" instead of "bsl create")
func TestVerbNounOrderFlagPassing(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name              string
		args              []string
		expectContains    []string
		expectNotContains []string
		shouldError       bool
		errorContains     []string
	}{
		{
			name: "create bsl help shows all flags",
			args: []string{"nonadmin", "create", "bsl", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
				"--provider",
				"--bucket",
				"--credential",
				"--region",
				"--prefix",
			},
			shouldError: false,
		},
		{
			name: "create bsl with missing required flags shows validation errors",
			args: []string{"nonadmin", "create", "bsl", "test-bsl"},
			expectContains: []string{
				"--provider is required",
			},
			shouldError: true,
		},
		{
			name: "create bsl with provider but missing bucket shows validation error",
			args: []string{"nonadmin", "create", "bsl", "test-bsl", "--provider", "aws"},
			expectContains: []string{
				"--bucket is required",
			},
			shouldError: true,
		},
		{
			name: "create bsl with provider and bucket but missing credential shows validation error",
			args: []string{"nonadmin", "create", "bsl", "test-bsl", "--provider", "aws", "--bucket", "test-bucket"},
			expectContains: []string{
				"--credential is required",
			},
			shouldError: true,
		},
		{
			name: "create bsl with all required flags passes validation (will fail on actual creation without cluster)",
			args: []string{"nonadmin", "create", "bsl", "test-bsl", "--provider", "aws", "--bucket", "test-bucket", "--credential", "secret=key", "--region", "us-east-1"},
			expectNotContains: []string{
				"--provider is required",
				"--bucket is required",
				"--credential is required",
			},
			// This will fail because we don't have a real cluster, but it should fail
			// with a different error (like connection error), not validation errors
			shouldError:   true,
			errorContains: []string{
				// Should NOT contain validation errors
			},
		},
		{
			name: "create bsl with prefix flag is recognized",
			args: []string{"nonadmin", "create", "bsl", "test-bsl", "--provider", "aws", "--bucket", "test-bucket", "--credential", "secret=key", "--region", "us-east-1", "--prefix", "velero"},
			expectNotContains: []string{
				"--provider is required",
				"--bucket is required",
				"--credential is required",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testutil.RunCommand(t, binaryPath, tt.args...)

			// Check expected content
			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
				}
			}

			// Check that certain content is NOT present (if specified)
			if len(tt.expectNotContains) > 0 {
				for _, notExpected := range tt.expectNotContains {
					if strings.Contains(output, notExpected) {
						t.Errorf("Expected output to NOT contain %q, but it did.\nFull output:\n%s", notExpected, output)
					}
				}
			}

			// Check error expectations
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected command to error, but it succeeded.\nOutput:\n%s", output)
				} else {
					// If specific error content is expected, verify it
					if len(tt.errorContains) > 0 {
						for _, expectedError := range tt.errorContains {
							if !strings.Contains(output, expectedError) {
								t.Errorf("Expected error output to contain %q, but it didn't.\nFull output:\n%s", expectedError, output)
							}
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected command to succeed, but it errored: %v\nOutput:\n%s", err, output)
				}
			}
		})
	}
}

// TestNounVerbOrderStillWorks tests that noun-verb order (bsl create) still works
// This ensures we didn't break the existing functionality
func TestNounVerbOrderStillWorks(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
		shouldError    bool
	}{
		{
			name: "bsl create help shows all flags",
			args: []string{"nonadmin", "bsl", "create", "--help"},
			expectContains: []string{
				"Create a non-admin backup storage location",
				"--provider",
				"--bucket",
				"--credential",
				"--region",
			},
			shouldError: false,
		},
		{
			name: "bsl create with missing required flags shows validation errors",
			args: []string{"nonadmin", "bsl", "create", "test-bsl"},
			expectContains: []string{
				"--provider is required",
			},
			shouldError: true,
		},
		{
			name:           "bsl create with all required flags passes validation",
			args:           []string{"nonadmin", "bsl", "create", "test-bsl", "--provider", "aws", "--bucket", "test-bucket", "--credential", "secret=key", "--region", "us-east-1"},
			expectContains: []string{
				// Should NOT contain validation errors - check that validation passes
			},
			shouldError: true, // Will fail on actual creation without cluster
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testutil.RunCommand(t, binaryPath, tt.args...)

			// Check expected content
			for _, expected := range tt.expectContains {
				if expected != "" && !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nFull output:\n%s", expected, output)
				}
			}

			// For the "all flags provided" test, verify validation errors are NOT present
			if tt.name == "bsl create with all required flags passes validation" {
				validationErrors := []string{
					"--provider is required",
					"--bucket is required",
					"--credential is required",
				}
				for _, validationError := range validationErrors {
					if strings.Contains(output, validationError) {
						t.Errorf("Expected validation to pass, but got error: %q\nFull output:\n%s", validationError, output)
					}
				}
			}

			// Check error expectations
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected command to error, but it succeeded.\nOutput:\n%s", output)
				}
			} else {
				if err != nil {
					t.Errorf("Expected command to succeed, but it errored: %v\nOutput:\n%s", err, output)
				}
			}
		})
	}
}

// TestVerbNounOrderAllFlagsRecognized tests that all flag types are properly recognized
// in verb-noun order, including map types like --credential
func TestVerbNounOrderAllFlagsRecognized(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create bsl recognizes credential flag as map type", func(t *testing.T) {
		// This test verifies that the credential flag (which is a map type)
		// is properly recognized and passed through in verb-noun order
		args := []string{"nonadmin", "create", "bsl", "test-bsl", "--provider", "aws", "--bucket", "test-bucket", "--credential", "cloud-credentials=cloud", "--region", "us-east-1"}

		output, err := testutil.RunCommand(t, binaryPath, args...)

		// Should NOT fail with "--credential is required" error
		// (it will fail for other reasons like missing cluster, but not validation)
		if err != nil {
			if strings.Contains(output, "--credential is required") {
				t.Errorf("Credential flag was not recognized in verb-noun order.\nOutput:\n%s", output)
			}
		} else {
			t.Logf("Command succeeded (unexpected, but good): %s", output)
		}
	})

	t.Run("create bsl recognizes all flag types", func(t *testing.T) {
		// Test with all flag types: string (provider, bucket, region), map (credential), string (prefix)
		args := []string{"nonadmin", "create", "bsl", "test-bsl",
			"--provider", "aws",
			"--bucket", "test-bucket",
			"--credential", "cloud-credentials=cloud",
			"--region", "us-east-1",
			"--prefix", "velero",
		}

		output, err := testutil.RunCommand(t, binaryPath, args...)

		// Should NOT fail with any validation errors
		validationErrors := []string{
			"--provider is required",
			"--bucket is required",
			"--credential is required",
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
}

// TestVerbNounOrderHelpShowsFlags tests that help output for verb-noun order
// shows all the flags that are available
func TestVerbNounOrderHelpShowsFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("create bsl help shows all BSL flags", func(t *testing.T) {
		args := []string{"nonadmin", "create", "bsl", "--help"}
		output, err := testutil.RunCommand(t, binaryPath, args...)

		if err != nil {
			t.Fatalf("Help command should not error: %v\nOutput:\n%s", err, output)
		}

		expectedFlags := []string{
			"--provider",
			"--bucket",
			"--credential",
			"--region",
			"--prefix",
			"--config",
		}

		for _, flag := range expectedFlags {
			if !strings.Contains(output, flag) {
				t.Errorf("Expected help output to contain flag %q, but it didn't.\nOutput:\n%s", flag, output)
			}
		}
	})
}
