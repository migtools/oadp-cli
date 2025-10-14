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
)

// NewGetCommand creates the "get" verb command that delegates to noun commands
func NewGetCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
	builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
	RegisterBackupResources(builder, "get")
	RegisterRestoreResources(builder, "get")
	RegisterScheduleResources(builder, "get")

	return builder.BuildVerbCommand(VerbConfig{
		Use:   "get",
		Short: "Get one or more resources",
		Long:  "Get one or more resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get all backups
  kubectl oadp get backup

  # Get a specific backup
  kubectl oadp get backup my-backup

  # Get all restores
  kubectl oadp get restore`,
	})
}

// NewCreateCommand creates the "create" verb command that delegates to noun commands
func NewCreateCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
	builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
	RegisterBackupResources(builder, "create")
	RegisterRestoreResources(builder, "create")
	RegisterScheduleResources(builder, "create")

	return builder.BuildVerbCommand(VerbConfig{
		Use:   "create",
		Short: "Create a resource",
		Long:  "Create a resource. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Create a backup
  kubectl oadp create backup my-backup

  # Create a restore
  kubectl oadp create restore my-restore`,
	})
}

// NewDeleteCommand creates the "delete" verb command that delegates to noun commands
func NewDeleteCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
	builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
	RegisterBackupResources(builder, "delete")
	RegisterRestoreResources(builder, "delete")
	RegisterScheduleResources(builder, "delete")

	return builder.BuildVerbCommand(VerbConfig{
		Use:   "delete",
		Short: "Delete a resource",
		Long:  "Delete a resource. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Delete a backup
  kubectl oadp delete backup my-backup

  # Delete a restore
  kubectl oadp delete restore my-restore`,
	})
}

// NewDescribeCommand creates the "describe" verb command that delegates to noun commands
func NewDescribeCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
	builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
	RegisterBackupResources(builder, "describe")
	RegisterRestoreResources(builder, "describe")
	RegisterScheduleResources(builder, "describe")

	return builder.BuildVerbCommand(VerbConfig{
		Use:   "describe",
		Short: "Describe a resource",
		Long:  "Describe a resource. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Describe a backup
  kubectl oadp describe backup my-backup

  # Describe a restore
  kubectl oadp describe restore my-restore`,
	})
}

// NewLogsCommand creates the "logs" verb command that delegates to noun commands
// This is an example of how easy it is to add new verbs!
func NewLogsCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
	builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
	RegisterBackupResources(builder, "logs")
	RegisterRestoreResources(builder, "logs")
	RegisterScheduleResources(builder, "logs")

	return builder.BuildVerbCommand(VerbConfig{
		Use:   "logs",
		Short: "Get logs for a resource",
		Long:  "Get logs for a resource. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get logs for a backup
  kubectl oadp logs backup my-backup

  # Get logs for a restore
  kubectl oadp logs restore my-restore

  # Get logs for a schedule
  kubectl oadp logs schedule my-schedule`,
	})
}
