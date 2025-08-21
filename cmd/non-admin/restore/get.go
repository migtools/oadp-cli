package restore

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

func NewGetCommand(f client.Factory, use string) *cobra.Command {
	o := NewGetOptions()

	c := &cobra.Command{
		Use:   use,
		Short: "Get non-admin restores",
		Long:  "Get non-admin restores in the current namespace",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # List all non-admin restores in the current namespace
  kubectl oadp nonadmin restore get

  # List restores in table format with extra columns
  kubectl oadp nonadmin restore get --show-labels`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())

	return c
}

type GetOptions struct {
	client           kbclient.WithWatch
	currentNamespace string
}

func NewGetOptions() *GetOptions {
	return &GetOptions{}
}

func (o *GetOptions) BindFlags(_flags *pflag.FlagSet) {
	// Add any get-specific flags here if needed
}

func (o *GetOptions) Complete(args []string, f client.Factory) error {
	// Create client with NonAdmin scheme
	client, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeNonAdminTypes: true,
	})
	if err != nil {
		return err
	}

	// Get the current namespace from kubeconfig
	currentNS, err := shared.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}

	o.client = client
	o.currentNamespace = currentNS
	return nil
}

func (o *GetOptions) Validate(_c *cobra.Command, _args []string, _f client.Factory) error {
	return nil
}

func (o *GetOptions) Run(c *cobra.Command, _f client.Factory) error {
	// List NonAdminRestore resources
	restoreList := &nacv1alpha1.NonAdminRestoreList{}

	err := o.client.List(context.Background(), restoreList, &kbclient.ListOptions{
		Namespace: o.currentNamespace,
	})
	if err != nil {
		return fmt.Errorf("failed to list non-admin restores: %w", err)
	}

	if len(restoreList.Items) == 0 {
		fmt.Printf("No non-admin restores found in namespace %s.\n", o.currentNamespace)
		return nil
	}

	// Print results in table format
	o.printTable(c, restoreList.Items)

	return nil
}

func (o *GetOptions) printTable(_ *cobra.Command, restores []nacv1alpha1.NonAdminRestore) {
	// Print header (backupName is not admin enforceable and therefore not displayed)
	fmt.Printf("%-20s %-15s %-20s\n", "NAME", "PHASE", "CREATED")
	fmt.Printf("%-20s %-15s %-20s\n", "----", "-----", "-------")

	// Print each restore
	for _, restore := range restores {
		name := restore.Name
		phase := string(restore.Status.Phase)
		if phase == "" {
			phase = "Unknown"
		}

		// Note: backupName is not admin enforceable and therefore not displayed

		created := restore.CreationTimestamp.Format("2006-01-02T15:04:05Z")

		fmt.Printf("%-20s %-15s %-20s\n",
			truncateString(name, 20),
			truncateString(phase, 15),
			created)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
