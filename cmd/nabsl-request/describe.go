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

package nabsl

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
)

func NewDescribeCommand(f client.Factory) *cobra.Command {
	o := NewDescribeOptions()

	c := &cobra.Command{
		Use:   "describe NAME",
		Short: "Describe a non-admin backup storage location request",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Describe a request by NABSL name
  kubectl oadp nabsl-request describe my-bsl-request

  # Describe a request by UUID
  kubectl oadp nabsl-request describe nacuser01-my-bsl-96dfa8b7-3f6f-4c8d-a168-8527b00fbed8`,
	}

	return c
}

type DescribeOptions struct {
	Name   string
	client kbclient.WithWatch
}

func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{}
}

func (o *DescribeOptions) Complete(args []string, f client.Factory) error {
	o.Name = args[0]

	client, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeVeleroTypes:   true,
		IncludeNonAdminTypes: true,
	})
	if err != nil {
		return err
	}

	o.client = client
	return nil
}

func (o *DescribeOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	return nil
}

func (o *DescribeOptions) Run(c *cobra.Command, f client.Factory) error {
	// Get the admin namespace (from client config) where requests are stored
	adminNS := f.Namespace()

	// Get the current namespace to find user's NABSLs
	currentNS, err := shared.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}

	// First get all NABSLs in user's namespace to find related requests
	var nabslList nacv1alpha1.NonAdminBackupStorageLocationList
	err = o.client.List(context.Background(), &nabslList, kbclient.InNamespace(currentNS))
	if err != nil {
		return fmt.Errorf("failed to list NABSLs: %w", err)
	}

	// Find the target UUID for the request
	var targetUUID string
	for _, nabsl := range nabslList.Items {
		if nabsl.Status.VeleroBackupStorageLocation != nil && nabsl.Status.VeleroBackupStorageLocation.NACUUID != "" {
			uuid := nabsl.Status.VeleroBackupStorageLocation.NACUUID
			// Check if o.Name matches the UUID or NABSL name
			if uuid == o.Name || nabsl.Name == o.Name {
				targetUUID = uuid
				break
			}
		}
	}

	if targetUUID == "" {
		return fmt.Errorf("request %q not found for NABSLs in namespace %s", o.Name, currentNS)
	}

	// Get the request from openshift-adp namespace using the UUID
	var request nacv1alpha1.NonAdminBackupStorageLocationRequest
	err = o.client.Get(context.Background(), kbclient.ObjectKey{
		Name:      targetUUID,
		Namespace: adminNS,
	}, &request)
	if err != nil {
		return fmt.Errorf("failed to get request for %q: %w", o.Name, err)
	}

	return describeRequest(&request)
}

func describeRequest(request *nacv1alpha1.NonAdminBackupStorageLocationRequest) error {
	fmt.Printf("Name:\t%s\n", request.Name)
	fmt.Printf("Namespace:\t%s\n", request.Namespace)

	fmt.Printf("Labels:\t%s\n", formatLabels(request.Labels))
	fmt.Printf("Annotations:\t%s\n", formatLabels(request.Annotations))

	fmt.Printf("Phase:\t%s\n", request.Status.Phase)

	if request.Spec.ApprovalDecision != "" {
		fmt.Printf("Approval Decision:\t%s\n", request.Spec.ApprovalDecision)
	}

	if request.Status.SourceNonAdminBSL != nil {
		source := request.Status.SourceNonAdminBSL
		fmt.Printf("Requested NonAdminBackupStorageLocation:\n")
		fmt.Printf("  Name:\t%s\n", source.Name)
		fmt.Printf("  Namespace:\t%s\n", source.Namespace)

		if source.NACUUID != "" {
			fmt.Printf("  NACUUID:\t%s\n", source.NACUUID)
		}

		if source.RequestedSpec != nil {
			spec := source.RequestedSpec
			fmt.Printf("Requested BackupStorageLocation Spec:\n")
			fmt.Printf("  Provider:\t%s\n", spec.Provider)
			fmt.Printf("  Object Storage Bucket:\t%s\n", spec.ObjectStorage.Bucket)

			if spec.ObjectStorage.Prefix != "" {
				fmt.Printf("  Prefix:\t%s\n", spec.ObjectStorage.Prefix)
			}

			if len(spec.Config) > 0 {
				fmt.Printf("  Config:\t%s\n", formatLabels(spec.Config))
			}

			if spec.AccessMode != "" {
				fmt.Printf("  Access Mode:\t%s\n", spec.AccessMode)
			}

			if spec.BackupSyncPeriod != nil {
				fmt.Printf("  Backup Sync Period:\t%s\n", spec.BackupSyncPeriod.String())
			}

			if spec.ValidationFrequency != nil {
				fmt.Printf("  Validation Frequency:\t%s\n", spec.ValidationFrequency.String())
			}
		}
	}

	fmt.Printf("Creation Timestamp:\t%s\n", request.CreationTimestamp.String())

	return nil
}

// formatLabels formats a map of labels into a string
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}

	var pairs []string
	for key, value := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}
