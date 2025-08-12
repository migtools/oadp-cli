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
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewNABSLRequestCommand creates the "nabsl-request" command for managing non-admin backup storage location requests
func NewNABSLRequestCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "nabsl-request",
		Short: "Manage non-admin backup storage location approval requests",
		Long: `Manage approval requests for non-admin backup storage locations.

Non-admin backup storage locations (NABSL) require admin approval before they can be used.
When users create NABSLs, approval requests are automatically generated for admin review.

Use these commands to view, approve, or reject pending NABSL requests.`,
		Example: `  # List all pending NABSL approval requests
  kubectl oadp nabsl-request get

  # Describe a specific NABSL approval request
  kubectl oadp nabsl-request describe my-storage-request

  # Approve a NABSL approval request
  kubectl oadp nabsl-request approve my-storage-request

  # Reject a NABSL approval request  
  kubectl oadp nabsl-request reject my-storage-request`,
	}

	c.AddCommand(
		NewGetCommand(f),
		NewDescribeCommand(f),
		NewApproveCommand(f),
		NewRejectCommand(f),
	)

	return c
}
