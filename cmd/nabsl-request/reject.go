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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
)

// NewRejectCommand creates the "deny" subcommand under bsl request
func NewRejectCommand(f client.Factory) *cobra.Command {
	o := NewRejectOptions()

	c := &cobra.Command{
		Use:   "reject REQUEST_NAME",
		Short: "Reject a pending backup storage location request",
		Long:  "Reject a pending backup storage location request to deny the user's request for a backup storage location",
		Args:  cobra.ExactArgs(1),
		Example: `  # Deny a request by NABSL name (admin access required)
  kubectl oadp nabsl-request reject user-test-bsl --reason "Invalid configuration"

  # Deny a request by UUID with detailed reason
  kubectl oadp nabsl-request reject nacuser01-user-test-bsl-96dfa8b7-3f6f-4c8d-a168-8527b00fbed8 --reason "Bucket does not exist in specified region"`,
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
	}

	o.BindFlags(c.Flags())

	return c
}

type RejectOptions struct {
	RequestName string
	Reason      string
	client      kbclient.WithWatch
}

func NewRejectOptions() *RejectOptions {
	return &RejectOptions{}
}

func (o *RejectOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Reason, "reason", "", "Reason for denial (recommended)")
}

func (o *RejectOptions) Complete(args []string, f client.Factory) error {
	o.RequestName = args[0]

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

func (o *RejectOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	return nil
}

func (o *RejectOptions) Run(c *cobra.Command, f client.Factory) error {
	// Get the admin namespace (from client config) where requests are stored
	adminNS := f.Namespace()

	// Find the request either by UUID or by looking up NABSL name
	requestName, err := shared.FindNABSLRequestByNameOrUUID(context.Background(), o.client, o.RequestName, adminNS)
	if err != nil {
		return err
	}

	// Get the current request
	var request nacv1alpha1.NonAdminBackupStorageLocationRequest
	err = o.client.Get(context.Background(), kbclient.ObjectKey{
		Name:      requestName,
		Namespace: adminNS,
	}, &request)
	if err != nil {
		return fmt.Errorf("failed to get request %q: %w", requestName, err)
	}

	// Check if already rejected
	if request.Spec.ApprovalDecision == "reject" {
		fmt.Printf("Request %q is already rejected.\n", o.RequestName)
		return nil
	}

	// Update the approval decision
	request.Spec.ApprovalDecision = "reject"
	if o.Reason != "" {
		if request.Annotations == nil {
			request.Annotations = make(map[string]string)
		}
		request.Annotations["openshift.io/oadp-rejection-reason"] = o.Reason
	}

	err = o.client.Update(context.Background(), &request)
	if err != nil {
		return fmt.Errorf("failed to deny request: %w", err)
	}

	// Get the NABSL name for user-friendly output
	nabslName := o.RequestName
	if request.Status.SourceNonAdminBSL != nil {
		nabslName = request.Status.SourceNonAdminBSL.Name
	}

	fmt.Printf("Request for NonAdminBackupStorageLocation %q has been rejected.\n", nabslName)
	if o.Reason != "" {
		fmt.Printf("Reason: %s\n", o.Reason)
	}

	return nil
}
