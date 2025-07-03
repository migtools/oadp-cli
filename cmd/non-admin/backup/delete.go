package backup

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

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

// NewDeleteCommand creates a cobra command for deleting non-admin backups
func NewDeleteCommand(f client.Factory, use string) *cobra.Command {
	o := NewDeleteOptions()

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Delete a non-admin backup",
		Long:  "Delete a non-admin backup by setting the deletebackup field to true",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate())
			cmd.CheckError(o.Run())
		},
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// DeleteOptions holds the options for the delete command
type DeleteOptions struct {
	Name      string
	Namespace string // Internal field - automatically determined from kubectl context
	client    kbclient.Client
}

// NewDeleteOptions creates a new DeleteOptions instance
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{}
}

// BindFlags binds the command line flags to the options
func (o *DeleteOptions) BindFlags(flags *pflag.FlagSet) {
	// No user-facing flags - namespace is determined automatically from kubectl context
}

// Complete completes the options by setting up the client and determining the namespace
func (o *DeleteOptions) Complete(args []string, f client.Factory) error {
	o.Name = args[0]

	// Get the Kubernetes client
	kbClient, err := f.KubebuilderWatchClient()
	if err != nil {
		return err
	}

	// Add NonAdminBackup types to the scheme
	err = nacv1alpha1.AddToScheme(kbClient.Scheme())
	if err != nil {
		return fmt.Errorf("failed to add NonAdminBackup types to scheme: %w", err)
	}

	o.client = kbClient

	// Always use the current namespace from kubectl context
	currentNS, err := getCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}
	o.Namespace = currentNS

	return nil
}

// Validate validates the options
func (o *DeleteOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("backup name is required")
	}
	if o.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	return nil
}

// Run executes the delete command
func (o *DeleteOptions) Run() error {
	// Get the NonAdminBackup resource
	nab := &nacv1alpha1.NonAdminBackup{}
	err := o.client.Get(context.TODO(), kbclient.ObjectKey{
		Name:      o.Name,
		Namespace: o.Namespace,
	}, nab)
	if err != nil {
		return fmt.Errorf("failed to get NonAdminBackup %s/%s: %w", o.Namespace, o.Name, err)
	}

	// Set the deletebackup field to true
	nab.Spec.DeleteBackup = true

	// Update the resource
	err = o.client.Update(context.TODO(), nab)
	if err != nil {
		return fmt.Errorf("failed to update NonAdminBackup %s/%s: %w", o.Namespace, o.Name, err)
	}

	fmt.Printf("NonAdminBackup %s/%s marked for deletion\n", o.Namespace, o.Name)
	return nil
}
