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
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/schedule"
)

// RegisterAllResources registers all available resource types with the verb builder
func RegisterAllResources(builder *VerbBuilder) {
	// Register backup resource
	builder.RegisterResource("backup", ResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return backup.NewCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, "get")
		},
	})

	// Register restore resource
	builder.RegisterResource("restore", ResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return restore.NewCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, "get")
		},
	})
}

// RegisterBackupResources registers backup resource for a specific verb
func RegisterBackupResources(builder *VerbBuilder, verb string) {
	builder.RegisterResource("backup", ResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return backup.NewCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, verb)
		},
	})
}

// RegisterRestoreResources registers restore resource for a specific verb
func RegisterRestoreResources(builder *VerbBuilder, verb string) {
	builder.RegisterResource("restore", ResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return restore.NewCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, verb)
		},
	})
}

// RegisterScheduleResources registers schedule resource for a specific verb
func RegisterScheduleResources(builder *VerbBuilder, verb string) {
	builder.RegisterResource("schedule", ResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return schedule.NewCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, verb)
		},
	})
}

// getSubCommand finds a subcommand by name
func getSubCommand(parentCmd *cobra.Command, subCommandName string) *cobra.Command {
	for _, subCmd := range parentCmd.Commands() {
		if subCmd.Name() == subCommandName {
			return subCmd
		}
	}
	return nil
}
