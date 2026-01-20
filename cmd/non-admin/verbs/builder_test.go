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
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// TestNonAdminVerbBuilder_RegisterResource tests resource registration
func TestNonAdminVerbBuilder_RegisterResource(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	handler := NonAdminResourceHandler{
		GetCommandFunc: func(f client.Factory) *cobra.Command {
			return &cobra.Command{Use: "test"}
		},
		GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command {
			return &cobra.Command{Use: "get"}
		},
	}

	builder.RegisterResource("test", handler)

	if _, exists := builder.resourceRegistry["test"]; !exists {
		t.Error("Expected resource 'test' to be registered")
	}
}

// TestNonAdminVerbBuilder_BuildVerbCommand tests basic verb command creation
func TestNonAdminVerbBuilder_BuildVerbCommand(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	config := NonAdminVerbConfig{
		Use:     "create",
		Short:   "Create resources",
		Long:    "Create non-admin resources",
		Example: "kubectl oadp nonadmin create backup",
	}

	cmd := builder.BuildVerbCommand(config)

	if cmd.Use != "create" {
		t.Errorf("Expected Use to be 'create', got %s", cmd.Use)
	}
	if cmd.Short != "Create resources" {
		t.Errorf("Expected Short to be 'Create resources', got %s", cmd.Short)
	}
	if cmd.Long != "Create non-admin resources" {
		t.Errorf("Expected Long description, got %s", cmd.Long)
	}
	if cmd.Example != "kubectl oadp nonadmin create backup" {
		t.Errorf("Expected Example, got %s", cmd.Example)
	}
}

// TestNonAdminVerbBuilder_FlagPassing tests that flags are passed correctly when using verb-first order
func TestNonAdminVerbBuilder_FlagPassing(t *testing.T) {
	tests := []struct {
		name          string
		flagName      string
		flagValue     string
		flagType      string
		expectedInArg string
	}{
		{
			name:          "string flag passed",
			flagName:      "storage-location",
			flagValue:     "aws-backup",
			flagType:      "string",
			expectedInArg: "--storage-location",
		},
		{
			name:          "label flag passed",
			flagName:      "labels",
			flagValue:     "app=test",
			flagType:      "map",
			expectedInArg: "--labels",
		},
		{
			name:          "include-resources flag passed",
			flagName:      "include-resources",
			flagValue:     "deployments,services",
			flagType:      "stringArray",
			expectedInArg: "--include-resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track what flag value was received in the delegated command
			var flagReceived bool
			var receivedValue string

			// Create a mock subcommand that captures the flag value
			mockSubCmd := &cobra.Command{
				Use:   "create NAME",
				Short: "Create a backup",
				RunE: func(cmd *cobra.Command, args []string) error {
					flag := cmd.Flags().Lookup(tt.flagName)
					if flag != nil && flag.Changed {
						flagReceived = true
						receivedValue = flag.Value.String()
					}
					return nil
				},
			}

			// Add the flag to the mock subcommand
			switch tt.flagType {
			case "string":
				mockSubCmd.Flags().String(tt.flagName, "", "test flag")
			case "map":
				mockSubCmd.Flags().StringToString(tt.flagName, nil, "test flag")
			case "stringArray":
				mockSubCmd.Flags().StringArray(tt.flagName, nil, "test flag")
			}

			// Create mock resource handler
			mockResourceCmd := &cobra.Command{
				Use:   "backup",
				Short: "Work with backups",
			}
			mockResourceCmd.AddCommand(mockSubCmd)

			handler := NonAdminResourceHandler{
				GetCommandFunc: func(f client.Factory) *cobra.Command {
					return mockResourceCmd
				},
				GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command {
					return mockSubCmd
				},
			}

			// Build the verb command
			builder := NewNonAdminVerbBuilder(nil)
			builder.RegisterResource("backup", handler)

			verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{
				Use:   "create",
				Short: "Create resources",
			})

			// Set the flag value on the verb command and execute
			verbCmd.SetArgs([]string{"backup", "test-backup", "--" + tt.flagName + "=" + tt.flagValue})

			// Execute the command
			err := verbCmd.Execute()
			if err != nil {
				t.Logf("Command execution error (expected if no cluster): %v", err)
			}

			// Verify the flag was passed to the delegated command
			if !flagReceived {
				t.Errorf("Expected flag %s to be passed to delegated command, but it wasn't received",
					tt.flagName)
			} else {
				t.Logf("Flag %s successfully passed with value: %s", tt.flagName, receivedValue)
			}
		})
	}
}

// TestNonAdminVerbBuilder_BoolFlagPassing tests boolean flag handling
func TestNonAdminVerbBuilder_BoolFlagPassing(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		flagValue    bool
		shouldAppear bool
	}{
		{
			name:         "bool flag true",
			flagName:     "force",
			flagValue:    true,
			shouldAppear: true,
		},
		{
			name:         "bool flag false",
			flagName:     "force",
			flagValue:    false,
			shouldAppear: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flagReceived bool

			mockSubCmd := &cobra.Command{
				Use:   "create NAME",
				Short: "Create a backup",
				RunE: func(cmd *cobra.Command, args []string) error {
					flag := cmd.Flags().Lookup(tt.flagName)
					if flag != nil && flag.Changed {
						flagReceived = true
					}
					return nil
				},
			}
			mockSubCmd.Flags().Bool(tt.flagName, false, "test flag")

			mockResourceCmd := &cobra.Command{Use: "backup"}
			mockResourceCmd.AddCommand(mockSubCmd)

			handler := NonAdminResourceHandler{
				GetCommandFunc:    func(f client.Factory) *cobra.Command { return mockResourceCmd },
				GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command { return mockSubCmd },
			}

			builder := NewNonAdminVerbBuilder(nil)
			builder.RegisterResource("backup", handler)

			verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{Use: "create"})

			args := []string{"backup", "test-backup"}
			if tt.flagValue {
				args = append(args, "--"+tt.flagName)
			}
			verbCmd.SetArgs(args)

			err := verbCmd.Execute()
			if err != nil {
				t.Logf("Command execution error (expected if no cluster): %v", err)
			}

			if tt.shouldAppear && !flagReceived {
				t.Errorf("Expected flag --%s to be passed when value is %v", tt.flagName, tt.flagValue)
			}
			if !tt.shouldAppear && flagReceived {
				t.Errorf("Expected flag --%s NOT to be passed when value is %v", tt.flagName, tt.flagValue)
			}
		})
	}
}

// TestNonAdminVerbBuilder_UnknownResourceType tests error handling for unknown resources
func TestNonAdminVerbBuilder_UnknownResourceType(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "get",
		Short: "Get resources",
	})

	// Try to use an unregistered resource type
	verbCmd.SetArgs([]string{"unknown-resource", "test-name"})

	err := verbCmd.Execute()
	if err == nil {
		t.Error("Expected error for unknown resource type, got nil")
	}

	if !strings.Contains(err.Error(), "unknown resource type") {
		t.Errorf("Expected error message containing 'unknown resource type', got: %v", err)
	}

	t.Logf("Got expected error: %v", err)
}

// TestNonAdminVerbBuilder_MissingResourceType tests error when no resource type is provided
func TestNonAdminVerbBuilder_MissingResourceType(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "get",
		Short: "Get resources",
	})

	// Execute without any arguments
	verbCmd.SetArgs([]string{})

	var stderr bytes.Buffer
	verbCmd.SetErr(&stderr)

	err := verbCmd.Execute()
	if err == nil {
		t.Error("Expected error when no resource type provided, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

// TestNonAdminVerbBuilder_AddFlagsFromResources tests that flags from registered resources are added to verb command
func TestNonAdminVerbBuilder_AddFlagsFromResources(t *testing.T) {
	// Create a mock subcommand with specific flags
	mockSubCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a backup",
	}
	mockSubCmd.Flags().String("storage-location", "", "storage location")
	mockSubCmd.Flags().StringArray("include-resources", nil, "resources to include")
	mockSubCmd.Flags().Bool("force", false, "force creation")

	mockResourceCmd := &cobra.Command{Use: "backup"}
	mockResourceCmd.AddCommand(mockSubCmd)

	handler := NonAdminResourceHandler{
		GetCommandFunc:    func(f client.Factory) *cobra.Command { return mockResourceCmd },
		GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command { return mockSubCmd },
	}

	builder := NewNonAdminVerbBuilder(nil)
	builder.RegisterResource("backup", handler)

	verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "create",
		Short: "Create resources",
	})

	// Verify that flags from the backup create command were added to the verb command
	expectedFlags := []string{"storage-location", "include-resources", "force"}
	for _, flagName := range expectedFlags {
		flag := verbCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' to be added to verb command, but it wasn't found", flagName)
		} else {
			t.Logf("Flag '%s' successfully added to verb command", flagName)
		}
	}
}

// TestNonAdminVerbBuilder_MultipleResourceTypes tests verb command with multiple resource types
func TestNonAdminVerbBuilder_MultipleResourceTypes(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	// Register backup resource
	mockBackupSubCmd := &cobra.Command{Use: "create NAME", Short: "Create a backup"}
	mockBackupCmd := &cobra.Command{Use: "backup"}
	mockBackupCmd.AddCommand(mockBackupSubCmd)

	builder.RegisterResource("backup", NonAdminResourceHandler{
		GetCommandFunc:    func(f client.Factory) *cobra.Command { return mockBackupCmd },
		GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command { return mockBackupSubCmd },
	})

	// Register bsl resource
	mockBSLSubCmd := &cobra.Command{Use: "create NAME", Short: "Create a BSL"}
	mockBSLCmd := &cobra.Command{Use: "bsl"}
	mockBSLCmd.AddCommand(mockBSLSubCmd)

	builder.RegisterResource("bsl", NonAdminResourceHandler{
		GetCommandFunc:    func(f client.Factory) *cobra.Command { return mockBSLCmd },
		GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command { return mockBSLSubCmd },
	})

	// Verify both resource types are registered
	if len(builder.resourceRegistry) != 2 {
		t.Errorf("Expected 2 registered resources, got %d", len(builder.resourceRegistry))
	}

	// Test that both resource types work with the verb command
	resourceTypes := []string{"backup", "bsl"}
	for _, resourceType := range resourceTypes {
		t.Run("resource_"+resourceType, func(t *testing.T) {
			testVerbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{
				Use:   "create",
				Short: "Create resources",
			})

			testVerbCmd.SetArgs([]string{resourceType, "test-name"})
			err := testVerbCmd.Execute()
			// Error is expected (no cluster), but it should not be "unknown resource type"
			if err != nil && strings.Contains(err.Error(), "unknown resource type") {
				t.Errorf("Resource type '%s' should be recognized, got error: %v", resourceType, err)
			}
		})
	}
}

// TestNonAdminVerbBuilder_CreateCommandInstance tests command instance creation
func TestNonAdminVerbBuilder_CreateCommandInstance(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	originalCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a resource",
		Long:  "Create a non-admin resource",
	}
	originalCmd.Flags().String("test-flag", "default", "test flag")
	originalCmd.PersistentFlags().String("persistent-flag", "default", "persistent flag")

	newCmd := builder.createCommandInstance(originalCmd)

	// Verify basic fields are copied
	if newCmd.Use != originalCmd.Use {
		t.Errorf("Expected Use to be %s, got %s", originalCmd.Use, newCmd.Use)
	}
	if newCmd.Short != originalCmd.Short {
		t.Errorf("Expected Short to be %s, got %s", originalCmd.Short, newCmd.Short)
	}
	if newCmd.Long != originalCmd.Long {
		t.Errorf("Expected Long to be %s, got %s", originalCmd.Long, newCmd.Long)
	}

	// Verify flags are copied
	if newCmd.Flags().Lookup("test-flag") == nil {
		t.Error("Expected test-flag to be copied to new command")
	}
	if newCmd.PersistentFlags().Lookup("persistent-flag") == nil {
		t.Error("Expected persistent-flag to be copied to new command")
	}
}

// TestNonAdminVerbBuilder_NilHandler tests handling of nil handlers
func TestNonAdminVerbBuilder_NilHandler(t *testing.T) {
	builder := NewNonAdminVerbBuilder(nil)

	// Register a handler that returns nil for GetCommandFunc
	builder.RegisterResource("nil-resource", NonAdminResourceHandler{
		GetCommandFunc:    func(f client.Factory) *cobra.Command { return nil },
		GetSubCommandFunc: func(cmd *cobra.Command) *cobra.Command { return nil },
	})

	verbCmd := builder.BuildVerbCommand(NonAdminVerbConfig{Use: "get"})
	verbCmd.SetArgs([]string{"nil-resource", "test-name"})

	err := verbCmd.Execute()
	if err == nil {
		t.Error("Expected error when handler returns nil command, got nil")
	}

	if !strings.Contains(err.Error(), "command not found") {
		t.Logf("Got error (as expected): %v", err)
	}
}
