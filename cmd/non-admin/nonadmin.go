package nonadmin

import (
	"github.com/migtools/oadp-cli/cmd/non-admin/backup"
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
	c.AddCommand(backup.NewBackupCommand(f))

	return c
}

// NewNonAdminFactory creates a client factory for NonAdminBackup operations
// that uses the current kubeconfig context namespace instead of hardcoded openshift-adp
func NewNonAdminFactory() client.Factory {
	// Don't set a default namespace, let it use the kubeconfig context
	cfg := client.VeleroConfig{}
	return client.NewFactory("oadp-nonadmin-cli", cfg)
}
