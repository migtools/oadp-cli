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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCreateCommand(f client.Factory, use string) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Create a non-admin backup storage location",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a non-admin backup storage location
  kubectl oadp nonadmin bsl create my-bsl --backup-storage-location default

  # Create a non-admin backup storage location with specific namespace
  kubectl oadp nonadmin bsl create my-bsl --backup-storage-location aws-bsl --namespace my-namespace

  # Create with custom BSL namespace (if OADP operator is not in openshift-adp)
  kubectl oadp nonadmin bsl create my-bsl --backup-storage-location default --bsl-namespace velero

  # View the YAML for a non-admin backup storage location without sending it to the server
  kubectl oadp nonadmin bsl create my-bsl --backup-storage-location default -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	Name                  string
	BackupStorageLocation string
	NonAdminNamespace     string
	BSLNamespace          string
	client                kbclient.WithWatch
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		BSLNamespace: "openshift-adp", // Default OADP operator namespace
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.BackupStorageLocation, "backup-storage-location", "", "Name of the BackupStorageLocation to reference.")
	flags.StringVar(&o.NonAdminNamespace, "namespace", "", "Namespace for the NonAdminBackupStorageLocation (defaults to current context namespace).")
	flags.StringVar(&o.BSLNamespace, "bsl-namespace", "openshift-adp", "Namespace where the BackupStorageLocation exists.")
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	o.Name = args[0]

	client, err := f.KubebuilderWatchClient()
	if err != nil {
		return err
	}

	// Add Velero types to the scheme so we can fetch BackupStorageLocation objects
	err = velerov1.AddToScheme(client.Scheme())
	if err != nil {
		return fmt.Errorf("failed to add Velero types to scheme: %w", err)
	}

	o.client = client

	if o.NonAdminNamespace == "" {
		namespace := f.Namespace()
		o.NonAdminNamespace = namespace
	}

	return nil
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if o.BackupStorageLocation == "" {
		return fmt.Errorf("--backup-storage-location is required")
	}

	return nil
}

func (o *CreateOptions) Run(c *cobra.Command, f client.Factory) error {
	// If we have a BackupStorageLocation name, we need to fetch its spec
	var bslSpec *velerov1.BackupStorageLocationSpec
	if o.BackupStorageLocation != "" {
		// Get the existing BackupStorageLocation to copy its spec
		existingBSL := &velerov1.BackupStorageLocation{}
		err := o.client.Get(context.Background(), kbclient.ObjectKey{
			Name:      o.BackupStorageLocation,
			Namespace: o.BSLNamespace, // Use the BSLNamespace flag
		}, existingBSL)
		if err != nil {
			return fmt.Errorf("failed to get BackupStorageLocation %q: %w", o.BackupStorageLocation, err)
		}
		bslSpec = &existingBSL.Spec
	}

	bsl := &nacv1alpha1.NonAdminBackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.NonAdminNamespace,
		},
		Spec: nacv1alpha1.NonAdminBackupStorageLocationSpec{
			BackupStorageLocationSpec: bslSpec,
		},
	}

	if printed, err := output.PrintWithFormat(c, bsl); printed || err != nil {
		return err
	}

	err := o.client.Create(context.Background(), bsl)
	if err != nil {
		return err
	}

	fmt.Printf("NonAdminBackupStorageLocation %q created successfully.\n", bsl.Name)
	return nil
}
