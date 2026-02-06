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

// TestNamespaceFlag tests that the -n/--namespace flag is available for admin commands
func TestNamespaceFlag(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	tests := []struct {
		name           string
		args           []string
		expectContains []string
	}{
		{
			name:           "backup help shows namespace flag",
			args:           []string{"backup", "create", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "restore help shows namespace flag",
			args:           []string{"restore", "create", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "schedule help shows namespace flag",
			args:           []string{"schedule", "create", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "nabsl-request get help shows namespace flag",
			args:           []string{"nabsl-request", "get", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "nabsl-request approve help shows namespace flag",
			args:           []string{"nabsl-request", "approve", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "nabsl-request reject help shows namespace flag",
			args:           []string{"nabsl-request", "reject", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
		{
			name:           "nabsl-request describe help shows namespace flag",
			args:           []string{"nabsl-request", "describe", "--help"},
			expectContains: []string{"-n, --namespace"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
		})
	}
}
