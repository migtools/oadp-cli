package backup

import (
	"context"
	"fmt"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	corev1 "k8s.io/api/core/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGetCommand(f client.Factory, use string) *cobra.Command {
	c := &cobra.Command{
		Use:   use + " [NAME]",
		Short: "Get non-admin backup(s)",
		Long:  "Get one or more non-admin backups",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the current namespace from kubectl context
			userNamespace, err := getCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}

			// Setup client using factory and add schemes to its existing scheme
			kbClient, err := f.KubebuilderWatchClient()
			if err != nil {
				return fmt.Errorf("failed to create controller-runtime client: %w", err)
			}

			// Add types to the existing client scheme
			if err := nacv1alpha1.AddToScheme(kbClient.Scheme()); err != nil {
				return fmt.Errorf("failed to add OADP non-admin types to scheme: %w", err)
			}
			if err := velerov1.AddToScheme(kbClient.Scheme()); err != nil {
				return fmt.Errorf("failed to add Velero types to scheme: %w", err)
			}
			if err := corev1.AddToScheme(kbClient.Scheme()); err != nil {
				return fmt.Errorf("failed to add Core types to scheme: %w", err)
			}

			if len(args) == 1 {
				// Get specific backup
				backupName := args[0]
				var nab nacv1alpha1.NonAdminBackup
				err := kbClient.Get(context.Background(), kbclient.ObjectKey{
					Namespace: userNamespace,
					Name:      backupName,
				}, &nab)
				if err != nil {
					return fmt.Errorf("failed to get NonAdminBackup %q: %w", backupName, err)
				}

				if printed, err := output.PrintWithFormat(cmd, &nab); printed || err != nil {
					return err
				}

				// If no output format specified, print table format for single item
				list := &nacv1alpha1.NonAdminBackupList{
					Items: []nacv1alpha1.NonAdminBackup{nab},
				}
				return printNonAdminBackupTable(list)
			} else {
				// List all backups in namespace
				var nabList nacv1alpha1.NonAdminBackupList
				err := kbClient.List(context.Background(), &nabList, &kbclient.ListOptions{
					Namespace: userNamespace,
				})
				if err != nil {
					return fmt.Errorf("failed to list NonAdminBackups: %w", err)
				}

				if printed, err := output.PrintWithFormat(cmd, &nabList); printed || err != nil {
					return err
				}

				// Print table format
				return printNonAdminBackupTable(&nabList)
			}
		},
		Example: `  # Get all non-admin backups in the current namespace
  kubectl oadp nonadmin backup get

  # Get a specific non-admin backup
  kubectl oadp nonadmin backup get my-backup

  # Get backups in YAML format
  kubectl oadp nonadmin backup get -o yaml

  # Get a specific backup in JSON format
  kubectl oadp nonadmin backup get my-backup -o json`,
	}

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

func printNonAdminBackupTable(nabList *nacv1alpha1.NonAdminBackupList) error {
	if len(nabList.Items) == 0 {
		fmt.Println("No non-admin backups found.")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %-15s %-20s %-10s\n", "NAME", "STATUS", "CREATED", "AGE")

	// Print each backup
	for _, nab := range nabList.Items {
		status := getBackupStatus(&nab)
		created := nab.CreationTimestamp.Format("2006-01-02 15:04:05")
		age := formatAge(nab.CreationTimestamp.Time)

		fmt.Printf("%-30s %-15s %-20s %-10s\n", nab.Name, status, created, age)
	}

	return nil
}

func getBackupStatus(nab *nacv1alpha1.NonAdminBackup) string {
	if nab.Status.Phase != "" {
		return string(nab.Status.Phase)
	}
	return "Unknown"
}

func formatAge(t time.Time) string {
	duration := time.Since(t)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	} else {
		return "1m"
	}
}
