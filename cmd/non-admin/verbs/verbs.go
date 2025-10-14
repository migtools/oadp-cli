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
func NewGetCommand(factory client.Factory) *cobra.Command {
	builder := NewNonAdminVerbBuilder(factory)
	RegisterBackupResources(builder, "get")
	RegisterBSLResources(builder, "get")

	return builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "get",
		Short: "Get one or more non-admin resources",
		Long:  "Get one or more non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get all non-admin backups
  kubectl oadp nonadmin get backup

  # Get a specific non-admin backup
  kubectl oadp nonadmin get backup my-backup`,
	})
}

// NewCreateCommand creates the "create" verb command that delegates to noun commands
func NewCreateCommand(factory client.Factory) *cobra.Command {
	builder := NewNonAdminVerbBuilder(factory)
	RegisterBackupResources(builder, "create")
	RegisterBSLResources(builder, "create")

	return builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "create",
		Short: "Create non-admin resources",
		Long:  "Create non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Create a non-admin backup
  kubectl oadp nonadmin create backup my-backup

  # Create a backup storage location
  kubectl oadp nonadmin create bsl my-bsl`,
	})
}

// NewDeleteCommand creates the "delete" verb command that delegates to noun commands
func NewDeleteCommand(factory client.Factory) *cobra.Command {
	builder := NewNonAdminVerbBuilder(factory)
	RegisterBackupResources(builder, "delete")
	RegisterBSLResources(builder, "delete")

	return builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "delete",
		Short: "Delete non-admin resources",
		Long:  "Delete non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Delete a non-admin backup
  kubectl oadp nonadmin delete backup my-backup`,
	})
}

// NewDescribeCommand creates the "describe" verb command that delegates to noun commands
func NewDescribeCommand(factory client.Factory) *cobra.Command {
	builder := NewNonAdminVerbBuilder(factory)
	RegisterBackupResources(builder, "describe")
	RegisterBSLResources(builder, "describe")

	return builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "describe",
		Short: "Describe non-admin resources",
		Long:  "Describe non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Describe a non-admin backup
  kubectl oadp nonadmin describe backup my-backup`,
	})
}

// NewLogsCommand creates the "logs" verb command that delegates to noun commands
func NewLogsCommand(factory client.Factory) *cobra.Command {
	builder := NewNonAdminVerbBuilder(factory)
	RegisterBackupResources(builder, "logs")
	RegisterBSLResources(builder, "logs")

	return builder.BuildVerbCommand(NonAdminVerbConfig{
		Use:   "logs",
		Short: "Get logs for non-admin resources",
		Long:  "Get logs for non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get logs for a non-admin backup
  kubectl oadp nonadmin logs backup my-backup`,
	})
}
