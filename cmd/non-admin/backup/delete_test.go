/*
Copyright The Velero Contributors.

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

package backup

import (
	"testing"
)

func TestDeleteOptionsValidate(t *testing.T) {
	tests := []struct {
		name        string
		options     *DeleteOptions
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid with backup names",
			options: &DeleteOptions{
				Names:     []string{"backup1", "backup2"},
				Namespace: "test-namespace",
				All:       false,
			},
			args:        []string{"backup1", "backup2"},
			expectError: false,
		},
		{
			name: "valid with --all flag",
			options: &DeleteOptions{
				Names:     []string{"backup1"}, // Simulating found backups after listing
				Namespace: "test-namespace",
				All:       true,
			},
			args:        []string{}, // No args when using --all
			expectError: false,
		},
		{
			name: "invalid - both names and --all",
			options: &DeleteOptions{
				Names:     []string{},
				Namespace: "test-namespace",
				All:       true,
			},
			args:        []string{"backup1"}, // Args provided with --all
			expectError: true,
			errorMsg:    "cannot specify both backup names and --all flag",
		},
		{
			name: "invalid - neither names nor --all",
			options: &DeleteOptions{
				Names:     []string{},
				Namespace: "test-namespace",
				All:       false,
			},
			args:        []string{}, // No args and no --all
			expectError: true,
			errorMsg:    "at least one backup name is required, or use --all to delete all backups",
		},
		{
			name: "invalid - missing namespace",
			options: &DeleteOptions{
				Names:     []string{"backup1"},
				Namespace: "",
				All:       false,
			},
			args:        []string{"backup1"},
			expectError: true,
			errorMsg:    "namespace is required",
		},
		{
			name: "invalid - --all flag but no backups found",
			options: &DeleteOptions{
				Names:     []string{}, // No backups found
				Namespace: "test-namespace",
				All:       true,
			},
			args:        []string{}, // No args
			expectError: true,
			errorMsg:    "no backups found in namespace 'test-namespace'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDeleteOptionsValidateLogic(t *testing.T) {
	// Test the special validation logic for --all with args provided
	t.Run("all flag should reject explicit backup names as args", func(t *testing.T) {
		// This simulates what would happen if someone runs:
		// oadp nonadmin backup delete backup1 backup2 --all
		// The args would be captured as Names, but All=true should cause validation error
		options := &DeleteOptions{
			Names:     []string{}, // Names not set yet (before Complete)
			Namespace: "test-namespace",
			All:       true, // But --all flag is also specified
		}

		args := []string{"backup1", "backup2"} // From command line args

		err := options.Validate(args)
		if err == nil {
			t.Errorf("expected error when both backup names and --all flag are provided")
		}

		expectedMsg := "cannot specify both backup names and --all flag"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}
