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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

func NewCreateCommand(f client.Factory, use string) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Create a non-admin restore",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a non-admin restore from a backup.
  kubectl oadp nonadmin restore create restore1 --backup-name backup1

  # Create a non-admin restore with namespace mapping.
  kubectl oadp nonadmin restore create restore2 --backup-name backup1 --namespace-mappings old-ns=new-ns

  # Create a non-admin restore with specific resource types.
  kubectl oadp nonadmin restore create restore3 --backup-name backup1 --include-resources deployments,services

  # Create a non-admin restore excluding certain resources.
  kubectl oadp nonadmin restore create restore4 --backup-name backup1 --exclude-resources secrets

  # View the YAML for a non-admin restore without sending it to the server.
  kubectl oadp nonadmin restore create restore5 --backup-name backup1 -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	Name                      string
	BackupName                string
	IncludeNamespaces         flag.StringArray
	ExcludeNamespaces         flag.StringArray
	IncludeResources          flag.StringArray
	ExcludeResources          flag.StringArray
	NamespaceMappings         flag.Map
	Labels                    flag.Map
	Annotations               flag.Map
	Selector                  flag.LabelSelector
	OrSelector                flag.OrLabelSelector
	RestoreVolumes            flag.OptionalBool
	PreserveNodePorts         flag.OptionalBool
	IncludeClusterResources   flag.OptionalBool
	ExistingResourcePolicy    string
	ItemOperationTimeout      time.Duration
	ResourceModifierConfigMap string
	client                    kbclient.WithWatch
	currentNamespace          string
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Labels:           flag.NewMap(),
		Annotations:      flag.NewMap(),
		NamespaceMappings: flag.NewMap(),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.BackupName, "backup-name", "", "The backup to restore from (required).")
	flags.Var(&o.IncludeNamespaces, "include-namespaces", "Namespaces to include in the restore (use '*' for all namespaces).")
	flags.Var(&o.ExcludeNamespaces, "exclude-namespaces", "Namespaces to exclude from the restore.")
	flags.Var(&o.IncludeResources, "include-resources", "Resources to include in the restore, formatted as resource.group, such as storageclasses.storage.k8s.io (use '*' for all resources).")
	flags.Var(&o.ExcludeResources, "exclude-resources", "Resources to exclude from the restore, formatted as resource.group, such as storageclasses.storage.k8s.io.")
	flags.Var(&o.NamespaceMappings, "namespace-mappings", "Namespace mappings from name in the backup to desired restored name in the form src1=dst1,src2=dst2,...")
	flags.Var(&o.Labels, "labels", "Labels to apply to the restore.")
	flags.Var(&o.Annotations, "annotations", "Annotations to apply to the restore.")
	flags.VarP(&o.Selector, "selector", "l", "Only restore resources matching this label selector.")
	flags.Var(&o.OrSelector, "or-selector", "Restore resources matching at least one of the label selector from the list. Label selectors should be separated by ' or '. For example, foo=bar or app=nginx")
	flags.StringVar(&o.ExistingResourcePolicy, "existing-resource-policy", "", "Policy to handle restore of items that already exist in the cluster. Options are 'none' and 'update'.")
	flags.DurationVar(&o.ItemOperationTimeout, "item-operation-timeout", o.ItemOperationTimeout, "How long to wait for async plugin operations before timeout.")
	flags.StringVar(&o.ResourceModifierConfigMap, "resource-modifier-configmap", "", "Reference to the resource modifier configmap that restore should use")

	f := flags.VarPF(&o.RestoreVolumes, "restore-volumes", "", "Whether to restore volumes from snapshots. If the parameter is not set, it is treated as setting to 'true'.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.PreserveNodePorts, "preserve-nodeports", "", "Whether to preserve nodeports when restoring services.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.IncludeClusterResources, "include-cluster-resources", "", "Include cluster-scoped resources in the restore.")
	f.NoOptDefVal = cmd.TRUE
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if err := output.ValidateFlags(c); err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("a restore name is required")
	}

	if o.BackupName == "" {
		return fmt.Errorf("--backup-name is required")
	}

	if o.Selector.LabelSelector != nil && o.OrSelector.OrLabelSelectors != nil {
		return fmt.Errorf("either a 'selector' or an 'or-selector' can be specified, but not both")
	}

	return nil
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	if len(args) > 0 {
		o.Name = args[0]
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

	fmt.Printf("NonAdminRestore request %q submitted successfully.\n", nonAdminRestore.Name)
	fmt.Printf("Run `oc oadp nonadmin restore describe %s` or `oc oadp nonadmin restore logs %s` for more details.\n", nonAdminRestore.Name, nonAdminRestore.Name)
	return nil
}

func (o *CreateOptions) BuildNonAdminRestore(namespace string) (*nacv1alpha1.NonAdminRestore, error) {
	// Use Velero's builder for RestoreSpec
	restoreBuilder := builder.ForRestore(namespace, o.Name).
		Backup(o.BackupName).
		IncludedNamespaces(o.IncludeNamespaces...).
		ExcludedNamespaces(o.ExcludeNamespaces...)

	// Convert namespace mappings from map to alternating key-value pairs
	if len(o.NamespaceMappings.Data()) > 0 {
		mappings := make([]string, 0, len(o.NamespaceMappings.Data())*2)
		for k, v := range o.NamespaceMappings.Data() {
			mappings = append(mappings, k, v)
		}
		restoreBuilder.NamespaceMappings(mappings...)
	}

	restoreBuilder.
		IncludedResources(o.IncludeResources...).
		ExcludedResources(o.ExcludeResources...).
		LabelSelector(o.Selector.LabelSelector).
		OrLabelSelector(o.OrSelector.OrLabelSelectors).
		ItemOperationTimeout(o.ItemOperationTimeout).
		ExistingResourcePolicy(o.ExistingResourcePolicy)

	// Apply optional bools
	if o.RestoreVolumes.Value != nil {
		restoreBuilder.RestorePVs(*o.RestoreVolumes.Value)
	}
	if o.PreserveNodePorts.Value != nil {
		restoreBuilder.PreserveNodePorts(*o.PreserveNodePorts.Value)
	}
	if o.IncludeClusterResources.Value != nil {
		restoreBuilder.IncludeClusterResources(*o.IncludeClusterResources.Value)
	}

	tempRestore := restoreBuilder.Result()

	// Set ResourceModifier manually since there's no builder method
	if o.ResourceModifierConfigMap != "" {
		tempRestore.Spec.ResourceModifier = &corev1.TypedLocalObjectReference{
			Kind: "ConfigMap",
			Name: o.ResourceModifierConfigMap,
		}
	}

	// Wrap in NonAdminRestore
	return ForNonAdminRestore(namespace, o.Name).
		ObjectMeta(
			WithLabelsMap(o.Labels.Data()),
			WithAnnotationsMap(o.Annotations.Data()),
		).
		RestoreSpec(nacv1alpha1.NonAdminRestoreSpec{
			RestoreSpec: &tempRestore.Spec,
		}).
		Result(), nil
}
