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

package nonadmin

import (
	"github.com/migtools/oadp-cli/cmd/non-admin/backup"
	"github.com/migtools/oadp-cli/cmd/non-admin/bsl"
	"github.com/migtools/oadp-cli/cmd/non-admin/restore"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewNonAdminCommand creates the top-level "nonadmin" subcommand
func NewNonAdminCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:     "nonadmin",
		Short:   "Work with non-admin resources",
		Long:    "Work with non-admin resources like backups, restores, and backup storage locations",
		Aliases: []string{"na"},
	}

	// Add backup subcommand
	c.AddCommand(backup.NewBackupCommand(f))

	// Add restore subcommand
	c.AddCommand(restore.NewRestoreCommand(f))
	// Add backup storage location subcommand
	c.AddCommand(bsl.NewBSLCommand(f))

	return c
}
