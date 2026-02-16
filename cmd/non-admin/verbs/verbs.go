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

	"github.com/migtools/oadp-cli/cmd/non-admin/backup"
	"github.com/migtools/oadp-cli/cmd/non-admin/bsl"
	"github.com/migtools/oadp-cli/cmd/non-admin/restore"
)

// NewGetCommand creates the "get" verb command that delegates to noun commands
func NewGetCommand(factory client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "get",
		Short: "Get one or more non-admin resources",
		Long:  "Get one or more non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get all non-admin backups
  kubectl oadp nonadmin get backup

  # Get a specific non-admin backup
  kubectl oadp nonadmin get backup my-backup

  # Get all non-admin restores
  kubectl oadp nonadmin get restore

  # Get a specific non-admin restore
  kubectl oadp nonadmin get restore my-restore

  # Get all non-admin backup storage locations
  kubectl oadp nonadmin get bsl

  # Get a specific backup storage location
  kubectl oadp nonadmin get bsl my-storage`,
	}

	c.AddCommand(
		backup.NewGetCommand(factory, "backup"),
		restore.NewGetCommand(factory, "restore"),
		bsl.NewGetCommand(factory, "bsl"),
	)

	return c
}

// NewCreateCommand creates the "create" verb command that delegates to noun commands
func NewCreateCommand(factory client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create non-admin resources",
		Long:  "Create non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Create a non-admin backup
  kubectl oadp nonadmin create backup my-backup

  # Create a non-admin restore
  kubectl oadp nonadmin create restore my-restore --backup-name my-backup

  # Create a backup storage location
  kubectl oadp nonadmin create bsl my-bsl`,
	}

	c.AddCommand(
		backup.NewCreateCommand(factory, "backup"),
		restore.NewCreateCommand(factory, "restore"),
		bsl.NewCreateCommand(factory, "bsl"),
	)

	return c
}

// NewDeleteCommand creates the "delete" verb command that delegates to noun commands
func NewDeleteCommand(factory client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete non-admin resources",
		Long:  "Delete non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Delete a non-admin backup
  kubectl oadp nonadmin delete backup my-backup

  # Delete a non-admin restore
  kubectl oadp nonadmin delete restore my-restore`,
	}

	c.AddCommand(
		backup.NewDeleteCommand(factory, "backup"),
		restore.NewDeleteCommand(factory, "restore"),
	)

	return c
}

// NewDescribeCommand creates the "describe" verb command that delegates to noun commands
func NewDescribeCommand(factory client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "describe",
		Short: "Describe non-admin resources",
		Long:  "Describe non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Describe a non-admin backup
  kubectl oadp nonadmin describe backup my-backup

  # Describe a non-admin restore
  kubectl oadp nonadmin describe restore my-restore`,
	}

	c.AddCommand(
		backup.NewDescribeCommand(factory, "backup"),
		restore.NewDescribeCommand(factory, "restore"),
	)

	return c
}

// NewLogsCommand creates the "logs" verb command that delegates to noun commands
func NewLogsCommand(factory client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "logs",
		Short: "Get logs for non-admin resources",
		Long:  "Get logs for non-admin resources. This is a verb-based command that delegates to the appropriate noun command.",
		Example: `  # Get logs for a non-admin backup
  kubectl oadp nonadmin logs backup my-backup

  # Get logs for a non-admin restore
  kubectl oadp nonadmin logs restore my-restore`,
	}

	c.AddCommand(
		backup.NewLogsCommand(factory, "backup"),
		restore.NewLogsCommand(factory, "restore"),
	)

	return c
}
