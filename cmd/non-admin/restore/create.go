package restore

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

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	velerorestore "github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

func NewCreateCommand(f client.Factory, use string) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   use + " [NAME]",
		Short: "Create a non-admin restore",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a non-admin restore from a backup (auto-generated name).
  oc oadp nonadmin restore create --backup-name backup1

  # Create a non-admin restore with a specific name.
  oc oadp nonadmin restore create restore1 --backup-name backup1

  # Create a non-admin restore with specific resource types.
  oc oadp nonadmin restore create restore2 --backup-name backup1 --include-resources deployments,services

  # Create a non-admin restore excluding certain resources.
  oc oadp nonadmin restore create restore3 --backup-name backup1 --exclude-resources secrets

  # Create a non-admin restore with label selector.
  oc oadp nonadmin restore create restore4 --backup-name backup1 --selector app=myapp

  # View the YAML for a non-admin restore without sending it to the server.
  oc oadp nonadmin restore create restore5 --backup-name backup1 -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	*velerorestore.CreateOptions

	// NAR-specific fields
	Name             string // The NonAdminRestore resource name (maps to Velero's RestoreName)
	client           kbclient.WithWatch
	currentNamespace string
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		CreateOptions: velerorestore.NewCreateOptions(),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {

	flags.StringVar(&o.BackupName, "backup-name", "", "The backup to restore from.")

	// Label selection
	flags.VarP(&o.Selector, "selector", "l", "Only restore resources matching this label selector.")
	flags.Var(&o.OrSelector, "or-selector", "Restore resources matching at least one of the label selector from the list. Label selectors should be separated by ' or '. For example, foo=bar or app=nginx")

	flags.DurationVar(&o.ItemOperationTimeout, "item-operation-timeout", o.ItemOperationTimeout, "How long to wait for async plugin operations before timeout.")

	flags.Var(&o.IncludeResources, "include-resources", "Resources to include in the restore, formatted as resource.group, such as storageclasses.storage.k8s.io (use '*' for all resources).")
	flags.Var(&o.ExcludeResources, "exclude-resources", "Resources to exclude from the restore, formatted as resource.group, such as storageclasses.storage.k8s.io.")

	f := flags.VarPF(&o.IncludeClusterResources, "include-cluster-resources", "", "Include cluster-scoped resources in the restore.")
	f.NoOptDefVal = cmd.TRUE
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if err := output.ValidateFlags(c); err != nil {
		return err
	}

	// Must specify backup-name
	if o.BackupName == "" {
		return fmt.Errorf("--backup-name is required")
	}

	if o.Selector.LabelSelector != nil && o.OrSelector.OrLabelSelectors != nil {
		return fmt.Errorf("either a 'selector' or an 'or-selector' can be specified, but not both")
	}

	return nil
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	// Name is optional - if not provided, will use GenerateName in the builder
	if len(args) > 0 {
		o.Name = args[0]
	} else {
		o.Name = ""
	}

	// Create client with NonAdmin scheme
	client, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeNonAdminTypes: true,
	})
	if err != nil {
		return err
	}

	// Get the current namespace from kubeconfig instead of using factory namespace
	currentNS, err := shared.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}

	o.client = client
	o.currentNamespace = currentNS
	return nil
}

func (o *CreateOptions) Run(c *cobra.Command, f client.Factory) error {
	nonAdminRestore, err := o.BuildNonAdminRestore(o.currentNamespace)
	if err != nil {
		return err
	}

	if printed, err := output.PrintWithFormat(c, nonAdminRestore); printed || err != nil {
		return err
	}

	// Create the restore
	if err := o.client.Create(context.TODO(), nonAdminRestore, &kbclient.CreateOptions{}); err != nil {
		return err
	}

	// Use the actual name (either provided or auto-generated by the API server)
	actualName := nonAdminRestore.Name
	fmt.Printf("NonAdminRestore request %q submitted successfully.\n", actualName)
	fmt.Printf("Run `oc oadp nonadmin restore describe %s` or `oc oadp nonadmin restore logs %s` for more details.\n", actualName, actualName)
	return nil
}

func (o *CreateOptions) BuildNonAdminRestore(namespace string) (*nacv1alpha1.NonAdminRestore, error) {
	// Use Velero's builder for RestoreSpec
	restoreBuilder := builder.ForRestore(namespace, o.Name).
		Backup(o.BackupName).
		IncludedResources(o.IncludeResources...).
		ExcludedResources(o.ExcludeResources...).
		LabelSelector(o.Selector.LabelSelector).
		OrLabelSelector(o.OrSelector.OrLabelSelectors).
		ItemOperationTimeout(o.ItemOperationTimeout)

	// Apply optional include-cluster-resources flag
	if o.IncludeClusterResources.Value != nil {
		restoreBuilder.IncludeClusterResources(*o.IncludeClusterResources.Value)
	}

	tempRestore := restoreBuilder.Result()

	// Wrap in NonAdminRestore
	return ForNonAdminRestore(namespace, o.Name).
		RestoreSpec(nacv1alpha1.NonAdminRestoreSpec{
			RestoreSpec: &tempRestore.Spec,
		}).
		Result(), nil
}
