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

// NonAdminResourceHandler defines functions to get the main command and its subcommand for a resource type.
type NonAdminResourceHandler struct {
	GetCommandFunc    func(client.Factory) *cobra.Command
	GetSubCommandFunc func(*cobra.Command) *cobra.Command
}

// NonAdminVerbBuilder helps construct verb-based commands dynamically for non-admin resources.
type NonAdminVerbBuilder struct {
	factory          client.Factory
	resourceRegistry map[string]NonAdminResourceHandler
}

// NewNonAdminVerbBuilder creates a new NonAdminVerbBuilder instance.
func NewNonAdminVerbBuilder(factory client.Factory) *NonAdminVerbBuilder {
	return &NonAdminVerbBuilder{
		factory:          factory,
		resourceRegistry: make(map[string]NonAdminResourceHandler),
	}
}

// RegisterResource registers a resource type with its handler functions.
func (vb *NonAdminVerbBuilder) RegisterResource(resourceType string, handler NonAdminResourceHandler) {
	vb.resourceRegistry[resourceType] = handler
}

// NonAdminVerbConfig holds configuration for a verb command.
type NonAdminVerbConfig struct {
	Use     string
	Short   string
	Long    string
	Example string
}

// BuildVerbCommand constructs a cobra.Command for a verb, delegating to registered noun commands.
func (vb *NonAdminVerbBuilder) BuildVerbCommand(config NonAdminVerbConfig) *cobra.Command {
	verbCmd := &cobra.Command{
		Use:     config.Use,
		Short:   config.Short,
		Long:    config.Long,
		Args:    cobra.MinimumNArgs(1),
		RunE:    vb.runEFunc(config.Use),
		Example: config.Example,
	}

	vb.addFlagsFromResources(verbCmd, config.Use)

	return verbCmd
}

func (vb *NonAdminVerbBuilder) runEFunc(verb string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("resource type required")
		}

		resourceType := args[0]
		remainingArgs := args[1:]

		handler, ok := vb.resourceRegistry[resourceType]
		if !ok {
			return fmt.Errorf("unknown resource type: %s", resourceType)
		}

		// Get the main command for the resource (e.g., "backup" command)
		resourceCmd := handler.GetCommandFunc(vb.factory)
		if resourceCmd == nil {
			return fmt.Errorf("%s command not found for resource type %s", verb, resourceType)
		}

		// Get the specific subcommand for the verb (e.g., "backup get" command)
		subCmd := handler.GetSubCommandFunc(resourceCmd)
		if subCmd == nil {
			return fmt.Errorf("%s %s command not found", resourceType, verb)
		}

		// Add flags to remaining args so they get passed to the delegated command
		remainingArgs = vb.addFlagsToArgs(cmd, remainingArgs)

		// Create a new command instance to avoid argument inheritance
		newSubCmd := vb.createCommandInstance(subCmd)
		newSubCmd.SetArgs(remainingArgs)

		return newSubCmd.Execute()
	}
}

// addFlagsToArgs adds flags from the verb command to the remaining args
func (vb *NonAdminVerbBuilder) addFlagsToArgs(cmd *cobra.Command, remainingArgs []string) []string {
	// Use Visit instead of VisitAll to only process flags that were actually set
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		flagValue := flag.Value.String()
		flagType := flag.Value.Type()

		switch flagType {
		case "string", "map":
			remainingArgs = append(remainingArgs, "--"+flag.Name, flagValue)
		case "bool":
			if flagValue == "true" {
				remainingArgs = append(remainingArgs, "--"+flag.Name)
			}
		case "stringArray", "stringSlice":
			// Handle string array/slice flags
			remainingArgs = append(remainingArgs, "--"+flag.Name, flagValue)
		default:
			// For any other flag types, try to add them as string values
			// This handles custom types that implement pflag.Value
			if flagValue != "" {
				remainingArgs = append(remainingArgs, "--"+flag.Name, flagValue)
			}
		}
	})
	return remainingArgs
}

// createCommandInstance creates a new cobra.Command instance from an existing one to avoid argument/flag inheritance issues.
func (vb *NonAdminVerbBuilder) createCommandInstance(originalCmd *cobra.Command) *cobra.Command {
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
func (vb *NonAdminVerbBuilder) addFlagsFromResources(verbCmd *cobra.Command, verb string) {
	addedFlags := make(map[string]bool)

	for _, handler := range vb.resourceRegistry {
		resourceCmd := handler.GetCommandFunc(vb.factory)
		if resourceCmd == nil {
			continue
		}

		// Add flags from the specific verb subcommand (e.g., "backup create" flags to "create" command)
		// This ensures flags are recognized at the verb level
		subCmd := handler.GetSubCommandFunc(resourceCmd)
		if subCmd != nil {
			subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if !addedFlags[flag.Name] {
					verbCmd.Flags().AddFlag(flag)
					addedFlags[flag.Name] = true
				}
			})
		}
	}
}
