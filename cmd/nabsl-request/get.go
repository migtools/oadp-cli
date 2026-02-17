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
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGetCommand(f client.Factory) *cobra.Command {
	o := NewGetOptions()

	c := &cobra.Command{
		Use:   "get [NAME]",
		Short: "Get non-admin backup storage location requests",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Get all backup storage location requests (admin access required)
  kubectl oadp nabsl-request get

  # Get a specific request by NABSL name
  kubectl oadp nabsl-request get my-bsl-request

  # Get a specific request by UUID
  kubectl oadp nabsl-request get nacuser01-my-bsl-96dfa8b7-3f6f-4c8d-a168-8527b00fbed8

  # Get output in YAML format
  kubectl oadp nabsl-request get my-bsl-request -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type GetOptions struct {
	Name          string
	AllNamespaces bool
	client        kbclient.WithWatch
}

func NewGetOptions() *GetOptions {
	return &GetOptions{}
}

func (o *GetOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.AllNamespaces, "all-namespaces", false, "If present, list requests across all namespaces")
}

func (o *GetOptions) Complete(args []string, f client.Factory) error {
	if len(args) > 0 {
		o.Name = args[0]
	}

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

func (o *GetOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	return nil
}

func (o *GetOptions) Run(c *cobra.Command, f client.Factory) error {
	// Get the admin namespace (from client config) where requests are stored
	adminNS := f.Namespace()

	if o.Name != "" {
		// Get specific request by name (UUID)
		var request nacv1alpha1.NonAdminBackupStorageLocationRequest
		err := o.client.Get(context.Background(), kbclient.ObjectKey{
			Name:      o.Name,
			Namespace: adminNS,
		}, &request)
		if err != nil {
			return fmt.Errorf("failed to get request %q: %w", o.Name, err)
		}

		if printed, err := output.PrintWithFormat(c, &request); printed || err != nil {
			return err
		}

		list := &nacv1alpha1.NonAdminBackupStorageLocationRequestList{
			Items: []nacv1alpha1.NonAdminBackupStorageLocationRequest{request},
		}
		return printRequestTable(list)
	}

	// List all requests in admin namespace
	var requestList nacv1alpha1.NonAdminBackupStorageLocationRequestList
	var err error
	if o.AllNamespaces {
		err = o.client.List(context.Background(), &requestList)
	} else {
		err = o.client.List(context.Background(), &requestList, kbclient.InNamespace(adminNS))
	}
	if err != nil {
		return fmt.Errorf("failed to list requests: %w", err)
	}

	if printed, err := output.PrintWithFormat(c, &requestList); printed || err != nil {
		return err
	}

	return printRequestTable(&requestList)
}

func printRequestTable(requestList *nacv1alpha1.NonAdminBackupStorageLocationRequestList) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "NAME\tNAMESPACE\tPHASE\tREQUESTED-NABSL\tREQUESTED-NAMESPACE\tAGE")

	for _, request := range requestList.Items {
		age := metav1.Now().Sub(request.CreationTimestamp.Time)

		requestedNABSL := ""
		requestedNamespace := ""
		if request.Status.SourceNonAdminBSL != nil {
			requestedNABSL = request.Status.SourceNonAdminBSL.Name
			requestedNamespace = request.Status.SourceNonAdminBSL.Namespace
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			request.Name,
			request.Namespace,
			request.Status.Phase,
			requestedNABSL,
			requestedNamespace,
			age.Round(1e9).String(),
		)
	}

	return nil
}
