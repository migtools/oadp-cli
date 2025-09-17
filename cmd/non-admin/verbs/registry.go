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
	"github.com/migtools/oadp-cli/cmd/non-admin/backup"
	"github.com/migtools/oadp-cli/cmd/non-admin/bsl"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// RegisterBackupResources registers backup resource for a specific verb
func RegisterBackupResources(builder *NonAdminVerbBuilder, verb string) {
	builder.RegisterResource("backup", NonAdminResourceHandler{
		GetCommandFunc: func(factory client.Factory) *cobra.Command {
			return backup.NewBackupCommand(factory)
		},
		GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
			return getSubCommand(resourceCmd, verb)
		},
	})
}

// RegisterBSLResources registers bsl resource for a specific verb
// Note: BSL only supports create command, so we only register for create
func RegisterBSLResources(builder *NonAdminVerbBuilder, verb string) {
	// Only register BSL for create command since it doesn't have get, delete, describe, or logs
	if verb == "create" {
		builder.RegisterResource("bsl", NonAdminResourceHandler{
			GetCommandFunc: func(factory client.Factory) *cobra.Command {
				return bsl.NewBSLCommand(factory)
			},
			GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
				return getSubCommand(resourceCmd, verb)
			},
		})
	}
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
