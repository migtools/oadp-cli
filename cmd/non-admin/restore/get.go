/*
Copyright The Velero Contributors.

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
package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/migtools/oadp-cli/cmd/non-admin/output"
	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGetCommand(f client.Factory, use string) *cobra.Command {
	c := &cobra.Command{
		Use:   use + " [NAME]",
		Short: "Get non-admin restore(s)",
		Long:  "Get one or more non-admin restores",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the current namespace from kubectl context
			userNamespace, err := shared.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}

			// Create client with full scheme
			kbClient, err := shared.NewClientWithFullScheme(f)
			if err != nil {
				return err
			}

			if len(args) == 1 {
				// Get specific restore
				restoreName := args[0]
				var nar nacv1alpha1.NonAdminRestore
				err := kbClient.Get(context.Background(), kbclient.ObjectKey{
					Namespace: userNamespace,
					Name:      restoreName,
				}, &nar)
				if err != nil {
					return fmt.Errorf("failed to get NonAdminRestore %q: %w", restoreName, err)
				}

				if printed, err := output.PrintWithFormat(cmd, &nar); printed || err != nil {
					return err
				}

				// If no output format specified, print table format for single item
				list := &nacv1alpha1.NonAdminRestoreList{
					Items: []nacv1alpha1.NonAdminRestore{nar},
				}
				return printNonAdminRestoreTable(list)
			} else {
				// List all restores in namespace
				var narList nacv1alpha1.NonAdminRestoreList
				err := kbClient.List(context.Background(), &narList, &kbclient.ListOptions{
					Namespace: userNamespace,
				})
				if err != nil {
					return fmt.Errorf("failed to list NonAdminRestores: %w", err)
				}

				if printed, err := output.PrintWithFormat(cmd, &narList); printed || err != nil {
					return err
				}

				// Print table format
				return printNonAdminRestoreTable(&narList)
			}
		},
		Example: `  # Get all non-admin restores in the current namespace
  oc oadp nonadmin restore get

  # Get a specific non-admin restore
  oc oadp nonadmin restore get my-restore

  # Get restores in YAML format
  oc oadp nonadmin restore get -o yaml

  # Get a specific restore in JSON format
  oc oadp nonadmin restore get my-restore -o json`,
	}

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

func printNonAdminRestoreTable(narList *nacv1alpha1.NonAdminRestoreList) error {
	if len(narList.Items) == 0 {
		fmt.Println("No non-admin restores found.")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %-15s %-15s %-20s %-10s %-10s\n", "NAME", "REQUEST PHASE", "VELERO PHASE", "CREATED", "AGE", "DURATION")

	// Print each restore
	for _, nar := range narList.Items {
		status := getRestoreStatus(&nar)
		veleroPhase := getVeleroRestorePhase(&nar)
		created := nar.CreationTimestamp.Format("2006-01-02 15:04:05")
		age := formatAge(nar.CreationTimestamp.Time)
		duration := getRestoreDuration(&nar)

		fmt.Printf("%-30s %-15s %-15s %-20s %-10s %-10s\n", nar.Name, status, veleroPhase, created, age, duration)
	}

	return nil
}

func getRestoreStatus(nar *nacv1alpha1.NonAdminRestore) string {
	if nar.Status.Phase != "" {
		return string(nar.Status.Phase)
	}
	return "Unknown"
}

func getVeleroRestorePhase(nar *nacv1alpha1.NonAdminRestore) string {
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		if nar.Status.VeleroRestore.Status.Phase != "" {
			return string(nar.Status.VeleroRestore.Status.Phase)
		}
	}
	return "N/A"
}

func getRestoreDuration(nar *nacv1alpha1.NonAdminRestore) string {
	// Check if we have completion timestamp
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		if !nar.Status.VeleroRestore.Status.CompletionTimestamp.IsZero() {
			// Calculate duration from request creation to completion
			duration := nar.Status.VeleroRestore.Status.CompletionTimestamp.Time.Sub(nar.CreationTimestamp.Time)
			return formatDuration(duration)
		}
	}
	return "N/A"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
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
