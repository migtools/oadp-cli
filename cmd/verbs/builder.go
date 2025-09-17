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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// ResourceHandler defines how to handle a specific resource type for a verb
type ResourceHandler struct {
	// GetCommandFunc returns the command for this resource type
	GetCommandFunc func(factory client.Factory) *cobra.Command
	// GetSubCommandFunc returns the specific subcommand (e.g., "get", "create") from the resource command
	GetSubCommandFunc func(resourceCmd *cobra.Command) *cobra.Command
}

// VerbConfig defines the configuration for a verb command
type VerbConfig struct {
	Use     string
	Short   string
	Long    string
	Example string
}

// ResourceRegistry maps resource types to their handlers
type ResourceRegistry map[string]ResourceHandler

// VerbBuilder creates extensible verb commands
type VerbBuilder struct {
	veleroFactory    client.Factory
	nonAdminFactory  client.Factory
	resourceRegistry ResourceRegistry
}

// NewVerbBuilder creates a new verb builder
func NewVerbBuilder(veleroFactory, nonAdminFactory client.Factory) *VerbBuilder {
	return &VerbBuilder{
		veleroFactory:    veleroFactory,
		nonAdminFactory:  nonAdminFactory,
		resourceRegistry: make(ResourceRegistry),
	}
}

// RegisterResource registers a resource type with its handler
func (vb *VerbBuilder) RegisterResource(resourceType string, handler ResourceHandler) {
	vb.resourceRegistry[resourceType] = handler
}

// BuildVerbCommand creates a verb command that delegates to registered resources
func (vb *VerbBuilder) BuildVerbCommand(config VerbConfig) *cobra.Command {
	verbCmd := &cobra.Command{
		Use:   config.Use,
		Short: config.Short,
		Long:  config.Long,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return vb.executeVerbCommand(cmd, args)
		},
		Example: config.Example,
	}

	// Add flags from all registered resources
	vb.addFlagsFromResources(verbCmd)

	return verbCmd
}

// executeVerbCommand handles the execution of a verb command
func (vb *VerbBuilder) executeVerbCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("resource type required")
	}

	resourceType := args[0]
	remainingArgs := args[1:]

	// Get the handler for this resource type
	handler, exists := vb.resourceRegistry[resourceType]
	if !exists {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	// Get the resource command
	resourceCmd := handler.GetCommandFunc(vb.veleroFactory)
	if resourceCmd == nil {
		return fmt.Errorf("failed to get %s command", resourceType)
	}

	// Get the specific subcommand
	subCmd := handler.GetSubCommandFunc(resourceCmd)
	if subCmd == nil {
		return fmt.Errorf("%s %s command not found", resourceType, cmd.Name())
	}

	// Add flags to remaining args so they get passed to the delegated command
	remainingArgs = vb.addFlagsToArgs(cmd, remainingArgs)

	// Create a new command instance to avoid argument inheritance
	newSubCmd := vb.createCommandInstance(subCmd)
	newSubCmd.SetArgs(remainingArgs)

	return newSubCmd.Execute()
}

// addFlagsToArgs adds flags from the verb command to the remaining args
func (vb *VerbBuilder) addFlagsToArgs(cmd *cobra.Command, remainingArgs []string) []string {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			if flag.Value.Type() == "string" {
				remainingArgs = append(remainingArgs, "--"+flag.Name, flag.Value.String())
			} else if flag.Value.Type() == "bool" {
				if flag.Value.String() == "true" {
					remainingArgs = append(remainingArgs, "--"+flag.Name)
				}
			} else if flag.Value.Type() == "stringArray" {
				// Handle string array flags
				remainingArgs = append(remainingArgs, "--"+flag.Name, flag.Value.String())
			}
		}
	})
	return remainingArgs
}

// createCommandInstance creates a new command instance to avoid argument inheritance
func (vb *VerbBuilder) createCommandInstance(originalCmd *cobra.Command) *cobra.Command {
	newCmd := &cobra.Command{
		Use:   originalCmd.Use,
		Short: originalCmd.Short,
		Long:  originalCmd.Long,
		Run:   originalCmd.Run,
		RunE:  originalCmd.RunE,
	}

	// Copy flags from the original command
	originalCmd.Flags().VisitAll(func(flag *pflag.Flag) {
		newCmd.Flags().AddFlag(flag)
	})
	// Also copy persistent flags
	originalCmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		newCmd.PersistentFlags().AddFlag(flag)
	})

	return newCmd
}

// addFlagsFromResources adds flags from all registered resources to the verb command
func (vb *VerbBuilder) addFlagsFromResources(verbCmd *cobra.Command) {
	addedFlags := make(map[string]bool)

	for _, handler := range vb.resourceRegistry {
		resourceCmd := handler.GetCommandFunc(vb.veleroFactory)
		if resourceCmd == nil {
			continue
		}

		subCmd := handler.GetSubCommandFunc(resourceCmd)
		if subCmd == nil {
			continue
		}

		// Add flags from this resource's subcommand
		subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			// Only add flag if it doesn't already exist
			if !addedFlags[flag.Name] {
				verbCmd.Flags().AddFlag(flag)
				addedFlags[flag.Name] = true
			}
		})
	}
}
