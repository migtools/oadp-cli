package bsl

import (
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewBSLCommand creates the "bsl" subcommand under nonadmin
func NewBSLCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "bsl",
		Short: "Work with non-admin backup storage locations",
		Long:  "Work with non-admin backup storage locations",
	}

	c.AddCommand(
		NewCreateCommand(f, "create"),
	)

	return c
}
