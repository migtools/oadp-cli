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

package bsl

import (
	"context"
	"fmt"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGetCommand(f client.Factory, use string) *cobra.Command {
	c := &cobra.Command{
		Use:   use + " [NAME]",
		Short: "Get non-admin backup storage location(s)",
		Long:  "Get one or more non-admin backup storage locations",
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
				// Get specific BSL
				bslName := args[0]
				var nabsl nacv1alpha1.NonAdminBackupStorageLocation
				err := kbClient.Get(context.Background(), kbclient.ObjectKey{
					Namespace: userNamespace,
					Name:      bslName,
				}, &nabsl)
				if err != nil {
					return fmt.Errorf("failed to get NonAdminBackupStorageLocation %q: %w", bslName, err)
				}

				if printed, err := output.PrintWithFormat(cmd, &nabsl); printed || err != nil {
					return err
				}

				// If no output format specified, print table format for single item
				list := &nacv1alpha1.NonAdminBackupStorageLocationList{
					Items: []nacv1alpha1.NonAdminBackupStorageLocation{nabsl},
				}
				return printNonAdminBSLTable(list)
			} else {
				// List all BSLs in namespace
				var nabslList nacv1alpha1.NonAdminBackupStorageLocationList
				err := kbClient.List(context.Background(), &nabslList, &kbclient.ListOptions{
					Namespace: userNamespace,
				})
				if err != nil {
					return fmt.Errorf("failed to list NonAdminBackupStorageLocations: %w", err)
				}

				if printed, err := output.PrintWithFormat(cmd, &nabslList); printed || err != nil {
					return err
				}

				// Print table format
				return printNonAdminBSLTable(&nabslList)
			}
		},
		Example: `  # Get all non-admin backup storage locations in the current namespace
  kubectl oadp nonadmin bsl get

  # Get a specific non-admin backup storage location
  kubectl oadp nonadmin bsl get my-storage

  # Get backup storage locations in YAML format
  kubectl oadp nonadmin bsl get -o yaml

  # Get a specific backup storage location in JSON format
  kubectl oadp nonadmin bsl get my-storage -o json`,
	}

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

func printNonAdminBSLTable(nabslList *nacv1alpha1.NonAdminBackupStorageLocationList) error {
	if len(nabslList.Items) == 0 {
		fmt.Println("No non-admin backup storage locations found.")
		return nil
	}

	// Print header
	fmt.Printf("%-30s %-15s %-15s %-20s %-10s\n", "NAME", "REQUEST PHASE", "PROVIDER", "BUCKET/PREFIX", "AGE")

	// Print each BSL
	for _, nabsl := range nabslList.Items {
		status := getBSLStatus(&nabsl)
		provider := getProvider(&nabsl)
		bucketPrefix := getBucketPrefix(&nabsl)
		age := formatAge(nabsl.CreationTimestamp.Time)

		fmt.Printf("%-30s %-15s %-15s %-20s %-10s\n", nabsl.Name, status, provider, bucketPrefix, age)
	}

	return nil
}

func getBSLStatus(nabsl *nacv1alpha1.NonAdminBackupStorageLocation) string {
	if nabsl.Status.Phase != "" {
		return string(nabsl.Status.Phase)
	}
	return "Unknown"
}

func getProvider(nabsl *nacv1alpha1.NonAdminBackupStorageLocation) string {
	if nabsl.Spec.BackupStorageLocationSpec != nil && nabsl.Spec.BackupStorageLocationSpec.Provider != "" {
		return nabsl.Spec.BackupStorageLocationSpec.Provider
	}
	return "N/A"
}

func getBucketPrefix(nabsl *nacv1alpha1.NonAdminBackupStorageLocation) string {
	if nabsl.Spec.BackupStorageLocationSpec != nil && nabsl.Spec.BackupStorageLocationSpec.ObjectStorage != nil {
		bucket := nabsl.Spec.BackupStorageLocationSpec.ObjectStorage.Bucket
		prefix := nabsl.Spec.BackupStorageLocationSpec.ObjectStorage.Prefix
		if prefix != "" {
			return fmt.Sprintf("%s/%s", bucket, prefix)
		}
		return bucket
	}
	return "N/A"
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
