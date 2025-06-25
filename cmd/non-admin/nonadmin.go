package nonadmin

import (
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewNonAdminCommand creates the top-level "nonadmin" subcommand
func NewNonAdminCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "nonadmin",
		Short: "Work with non-admin resources",
		Long:  "Work with non-admin resources like backups",
	}

	// Add backup subcommand
	c.AddCommand(NewBackupCommand(f))

	return c
}
