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

package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNonAdminScheme(t *testing.T) {
	scheme := NonAdminScheme()

	tests := []struct {
		name    string
		gvk     schema.GroupVersionKind
		objType runtime.Object
	}{
		{
			name: "NonAdminBackup is registered",
			gvk: schema.GroupVersionKind{
				Group:   "oadp.openshift.io",
				Version: "v1alpha1",
				Kind:    "NonAdminBackup",
			},
			objType: &nacv1alpha1.NonAdminBackup{},
		},
		{
			name: "NonAdminBackupList is registered",
			gvk: schema.GroupVersionKind{
				Group:   "oadp.openshift.io",
				Version: "v1alpha1",
				Kind:    "NonAdminBackupList",
			},
			objType: &nacv1alpha1.NonAdminBackupList{},
		},
		{
			name: "NonAdminRestore is registered",
			gvk: schema.GroupVersionKind{
				Group:   "oadp.openshift.io",
				Version: "v1alpha1",
				Kind:    "NonAdminRestore",
			},
			objType: &nacv1alpha1.NonAdminRestore{},
		},
		{
			name: "NonAdminRestoreList is registered",
			gvk: schema.GroupVersionKind{
				Group:   "oadp.openshift.io",
				Version: "v1alpha1",
				Kind:    "NonAdminRestoreList",
			},
			objType: &nacv1alpha1.NonAdminRestoreList{},
		},
		{
			name: "NonAdminBackupStorageLocation is registered",
			gvk: schema.GroupVersionKind{
				Group:   "oadp.openshift.io",
				Version: "v1alpha1",
				Kind:    "NonAdminBackupStorageLocation",
			},
			objType: &nacv1alpha1.NonAdminBackupStorageLocation{},
		},
		{
			name: "Velero Backup is registered",
			gvk: schema.GroupVersionKind{
				Group:   "velero.io",
				Version: "v1",
				Kind:    "Backup",
			},
			objType: &velerov1api.Backup{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if the type is recognized by the scheme
			gvks, _, err := scheme.ObjectKinds(tt.objType)
			if err != nil {
				t.Fatalf("Failed to get ObjectKinds: %v", err)
			}

			found := false
			for _, gvk := range gvks {
				if gvk.Group == tt.gvk.Group && gvk.Version == tt.gvk.Version && gvk.Kind == tt.gvk.Kind {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected GVK %v to be registered in scheme, but it was not found", tt.gvk)
			}
		})
	}
}

func TestBindFlags(t *testing.T) {
	cmd := &cobra.Command{}
	BindFlags(cmd.Flags())

	// Check that the output flag is bound
	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("Expected 'output' flag to be bound, but it was not found")
	}

	// Check that the label-columns flag is bound
	labelColumnsFlag := cmd.Flags().Lookup("label-columns")
	if labelColumnsFlag == nil {
		t.Fatal("Expected 'label-columns' flag to be bound, but it was not found")
	}

	// Check that the show-labels flag is bound
	showLabelsFlag := cmd.Flags().Lookup("show-labels")
	if showLabelsFlag == nil {
		t.Fatal("Expected 'show-labels' flag to be bound, but it was not found")
	}
}

func TestClearOutputFlagDefault(t *testing.T) {
	cmd := &cobra.Command{}
	BindFlags(cmd.Flags())

	// Initially, the default should be "table"
	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag.DefValue != "table" {
		t.Errorf("Expected default value to be 'table', got %q", outputFlag.DefValue)
	}

	// Clear the default
	ClearOutputFlagDefault(cmd)

	// After clearing, the default should be empty
	if outputFlag.DefValue != "" {
		t.Errorf("Expected default value to be empty after clearing, got %q", outputFlag.DefValue)
	}
}

func TestPrintWithFormat(t *testing.T) {
	tests := []struct {
		name           string
		outputFormat   string
		obj            runtime.Object
		expectPrinted  bool
		expectError    bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name:          "empty format returns false",
			outputFormat:  "",
			obj:           createTestBackup("test-backup"),
			expectPrinted: false,
			expectError:   false,
		},
		{
			name:          "yaml format",
			outputFormat:  "yaml",
			obj:           createTestBackup("test-backup"),
			expectPrinted: true,
			expectError:   false,
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "apiVersion: oadp.openshift.io/v1alpha1") {
					t.Error("Expected YAML output to contain apiVersion")
				}
				if !strings.Contains(output, "kind: NonAdminBackup") {
					t.Error("Expected YAML output to contain kind")
				}
				if !strings.Contains(output, "name: test-backup") {
					t.Error("Expected YAML output to contain name")
				}
			},
		},
		{
			name:          "json format",
			outputFormat:  "json",
			obj:           createTestBackup("test-backup"),
			expectPrinted: true,
			expectError:   false,
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, `"apiVersion": "oadp.openshift.io/v1alpha1"`) {
					t.Error("Expected JSON output to contain apiVersion")
				}
				if !strings.Contains(output, `"kind": "NonAdminBackup"`) {
					t.Error("Expected JSON output to contain kind")
				}
				if !strings.Contains(output, `"name": "test-backup"`) {
					t.Error("Expected JSON output to contain name")
				}
			},
		},
		{
			name:          "table format returns false",
			outputFormat:  "table",
			obj:           createTestBackup("test-backup"),
			expectPrinted: false,
			expectError:   false,
		},
		{
			name:          "invalid format returns error",
			outputFormat:  "invalid",
			obj:           createTestBackup("test-backup"),
			expectPrinted: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a command with the output flag
			cmd := &cobra.Command{}
			BindFlags(cmd.Flags())
			if tt.outputFormat != "" {
				if err := cmd.Flags().Set("output", tt.outputFormat); err != nil {
					t.Fatalf("Failed to set output flag: %v", err)
				}
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printed, err := PrintWithFormat(cmd, tt.obj)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}
			output := buf.String()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check printed expectation
			if printed != tt.expectPrinted {
				t.Errorf("Expected printed=%v, got %v", tt.expectPrinted, printed)
			}

			// Validate output if provided
			if tt.validateOutput != nil {
				tt.validateOutput(t, output)
			}
		})
	}
}

func TestPrintWithFormatList(t *testing.T) {
	tests := []struct {
		name          string
		outputFormat  string
		obj           runtime.Object
		expectSingle  bool // Should single item list be printed as single object?
		validateCount func(t *testing.T, output string)
	}{
		{
			name:         "single item list printed as single object in yaml",
			outputFormat: "yaml",
			obj: &nacv1alpha1.NonAdminBackupList{
				Items: []nacv1alpha1.NonAdminBackup{
					*createTestBackup("backup-1"),
				},
			},
			expectSingle: true,
			validateCount: func(t *testing.T, output string) {
				// Single object should not have "items:" field
				if strings.Contains(output, "items:") {
					t.Error("Single item from list should not contain 'items:' field")
				}
				if !strings.Contains(output, "name: backup-1") {
					t.Error("Expected output to contain backup name")
				}
			},
		},
		{
			name:         "multiple item list printed as list in yaml",
			outputFormat: "yaml",
			obj: &nacv1alpha1.NonAdminBackupList{
				Items: []nacv1alpha1.NonAdminBackup{
					*createTestBackup("backup-1"),
					*createTestBackup("backup-2"),
				},
			},
			expectSingle: false,
			validateCount: func(t *testing.T, output string) {
				// Multiple objects should have "items:" field
				if !strings.Contains(output, "items:") {
					t.Error("Multiple items should contain 'items:' field")
				}
				if !strings.Contains(output, "name: backup-1") {
					t.Error("Expected output to contain first backup name")
				}
				if !strings.Contains(output, "name: backup-2") {
					t.Error("Expected output to contain second backup name")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			BindFlags(cmd.Flags())
			if err := cmd.Flags().Set("output", tt.outputFormat); err != nil {
				t.Fatalf("Failed to set output flag: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			_, err := PrintWithFormat(cmd, tt.obj)

			// Restore stdout and read captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Fatalf("Failed to read output: %v", copyErr)
			}
			output := buf.String()

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateCount != nil {
				tt.validateCount(t, output)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		obj      runtime.Object
		format   string
		validate func(t *testing.T, data []byte)
	}{
		{
			name:   "encode NonAdminBackup to yaml",
			obj:    createTestBackup("test-backup"),
			format: "yaml",
			validate: func(t *testing.T, data []byte) {
				output := string(data)
				if !strings.Contains(output, "apiVersion: oadp.openshift.io/v1alpha1") {
					t.Error("Expected YAML to contain apiVersion")
				}
				if !strings.Contains(output, "kind: NonAdminBackup") {
					t.Error("Expected YAML to contain kind")
				}
			},
		},
		{
			name:   "encode NonAdminBackup to json",
			obj:    createTestBackup("test-backup"),
			format: "json",
			validate: func(t *testing.T, data []byte) {
				output := string(data)
				if !strings.Contains(output, `"apiVersion"`) {
					t.Error("Expected JSON to contain apiVersion")
				}
				if !strings.Contains(output, `"kind"`) {
					t.Error("Expected JSON to contain kind")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := encode(tt.obj, tt.format)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, data)
			}
		})
	}
}

func TestEncoderFor(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		obj         runtime.Object
		expectError bool
	}{
		{
			name:        "yaml encoder",
			format:      "yaml",
			obj:         createTestBackup("test"),
			expectError: false,
		},
		{
			name:        "json encoder",
			format:      "json",
			obj:         createTestBackup("test"),
			expectError: false,
		},
		{
			name:        "invalid format",
			format:      "xml",
			obj:         createTestBackup("test"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := encoderFor(tt.format, tt.obj)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if encoder == nil {
					t.Error("Expected encoder but got nil")
				}
			}
		})
	}
}

func TestIsListType(t *testing.T) {
	tests := []struct {
		name   string
		obj    runtime.Object
		isList bool
	}{
		{
			name:   "NonAdminBackupList is a list",
			obj:    &nacv1alpha1.NonAdminBackupList{},
			isList: true,
		},
		{
			name:   "NonAdminBackup is not a list",
			obj:    &nacv1alpha1.NonAdminBackup{},
			isList: false,
		},
		{
			name:   "NonAdminRestoreList is a list",
			obj:    &nacv1alpha1.NonAdminRestoreList{},
			isList: true,
		},
		{
			name:   "NonAdminRestore is not a list",
			obj:    &nacv1alpha1.NonAdminRestore{},
			isList: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isList := meta.IsListType(tt.obj)
			if isList != tt.isList {
				t.Errorf("Expected IsListType=%v, got %v", tt.isList, isList)
			}
		})
	}
}

// Helper function to create a test NonAdminBackup
func createTestBackup(name string) *nacv1alpha1.NonAdminBackup {
	return &nacv1alpha1.NonAdminBackup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "oadp.openshift.io/v1alpha1",
			Kind:       "NonAdminBackup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
		},
		Spec: nacv1alpha1.NonAdminBackupSpec{
			BackupSpec: &velerov1api.BackupSpec{
				IncludedNamespaces: []string{"test-namespace"},
			},
		},
	}
}
