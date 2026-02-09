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

package mustgather_test

import (
	"strings"
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
)

// TestMustGatherHelp verifies that the help command displays expected content
func TestMustGatherHelp(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	expectContains := []string{
		"Collect diagnostic information",
		"--dest-dir",
		"--request-timeout",
		"--skip-tls",
		"Examples:",
	}

	testutil.TestHelpCommand(t, binaryPath, []string{"must-gather", "--help"}, expectContains)
}

// TestMustGatherHelpFlags verifies that both --help and -h work
func TestMustGatherHelpFlags(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	// Test both --help and -h work
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			output, err := testutil.RunCommand(t, binaryPath, "must-gather", flag)
			if err != nil {
				t.Errorf("Help command with %s failed: %v", flag, err)
			}
			if !strings.Contains(output, "Usage:") {
				t.Errorf("Help output missing Usage section")
			}
			if !strings.Contains(output, "Collect diagnostic information") {
				t.Errorf("Help output missing description")
			}
		})
	}
}

// TestMustGatherHelpContent verifies specific help text content
func TestMustGatherHelpContent(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	output, err := testutil.RunCommand(t, binaryPath, "must-gather", "--help")
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	// Verify all flags are documented
	requiredContent := []string{
		"--dest-dir",
		"--request-timeout",
		"--skip-tls",
		"Directory where must-gather output will be stored",
		"Timeout for the gather script",
		"Skip TLS verification",
	}

	for _, content := range requiredContent {
		if !strings.Contains(output, content) {
			t.Errorf("Help output missing expected content: %q", content)
		}
	}

	// Verify --image flag is hidden (should not appear in help)
	if strings.Contains(output, "--image") {
		t.Errorf("Hidden flag --image should not appear in help output")
	}
}

// TestMustGatherExamples verifies that the help text includes examples
func TestMustGatherExamples(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	output, err := testutil.RunCommand(t, binaryPath, "must-gather", "--help")
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	// Verify examples are present
	examples := []string{
		"oc oadp must-gather",
		"--dest-dir=/tmp/oadp-diagnostics",
		"--request-timeout=30s",
		"--skip-tls",
	}

	for _, example := range examples {
		if !strings.Contains(output, example) {
			t.Errorf("Help output missing example: %q", example)
		}
	}
}
