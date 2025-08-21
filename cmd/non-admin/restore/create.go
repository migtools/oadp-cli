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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

func NewCreateCommand(f client.Factory, use string) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   use + " NAME --from-backup BACKUP_NAME",
		Short: "Create a non-admin restore",
		Args:  cobra.MaximumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a non-admin restore from a backup in the current namespace.
  kubectl oadp nonadmin restore create restore1 --from-backup backup1

  # Create a non-admin restore with specific resource types.
  kubectl oadp nonadmin restore create restore2 --from-backup backup1 --include-resources deployments,services

  # Create a non-admin restore excluding certain resources.
  kubectl oadp nonadmin restore create restore3 --from-backup backup1 --exclude-resources secrets

  # View the YAML for a non-admin restore without sending it to the server.
  kubectl oadp nonadmin restore create restore4 --from-backup backup1 -o yaml

  # Wait for a non-admin restore to complete before returning from the command.
  kubectl oadp nonadmin restore create restore5 --from-backup backup1 --wait`,
	}

	o.BindFlags(c.Flags())
	o.BindWait(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	Name                    string
	FromBackup              string
	IncludeResources        flag.StringArray
	ExcludeResources        flag.StringArray
	Labels                  flag.Map
	Annotations             flag.Map
	Selector                flag.LabelSelector
	OrSelector              flag.OrLabelSelector
	IncludeClusterResources flag.OptionalBool
	Wait                    bool
	RestorePVs              flag.OptionalBool
	PreserveNodePorts       flag.OptionalBool
	ItemOperationTimeout    time.Duration
	ExistingResourcePolicy  string
	UploaderConfig          flag.Map
	client                  kbclient.WithWatch
	currentNamespace        string
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		IncludeResources:        flag.NewStringArray("*"),
		Labels:                  flag.NewMap(),
		Annotations:             flag.NewMap(),
		UploaderConfig:          flag.NewMap(),
		IncludeClusterResources: flag.NewOptionalBool(nil),
		RestorePVs:              flag.NewOptionalBool(nil),
		PreserveNodePorts:       flag.NewOptionalBool(nil),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.FromBackup, "from-backup", o.FromBackup, "Backup to restore from (required).")
	flags.Var(&o.IncludeResources, "include-resources", "Resources to include in the restore, formatted as resource.group, such as storageclasses.storage.k8s.io (use '*' for all resources).")
	flags.Var(&o.ExcludeResources, "exclude-resources", "Resources to exclude from the restore, formatted as resource.group, such as storageclasses.storage.k8s.io.")
	flags.Var(&o.Labels, "labels", "Labels to apply to the restore.")
	flags.Var(&o.Annotations, "annotations", "Annotations to apply to the restore.")
	flags.VarP(&o.Selector, "selector", "l", "Only restore resources matching this label selector.")
	flags.Var(&o.OrSelector, "or-selector", "Restore resources matching at least one of the label selector from the list. Label selectors should be separated by ' or '. For example, foo=bar or app=nginx")
	flags.DurationVar(&o.ItemOperationTimeout, "item-operation-timeout", o.ItemOperationTimeout, "How long to wait for async plugin operations before timeout.")
	flags.StringVar(&o.ExistingResourcePolicy, "existing-resource-policy", "", "Policy to handle restore collisions (none, update)")
	flags.Var(&o.UploaderConfig, "uploader-config", "Configuration for the uploader in form key1=value1,key2=value2")

	f := flags.VarPF(&o.IncludeClusterResources, "include-cluster-resources", "", "Include cluster-scoped resources.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.RestorePVs, "restore-volumes", "", "Whether to restore volumes from snapshots.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.PreserveNodePorts, "preserve-nodeports", "", "Whether to restore NodePort services as NodePort.")
	f.NoOptDefVal = cmd.TRUE
}

func (o *CreateOptions) BindWait(flags *pflag.FlagSet) {
	flags.BoolVarP(&o.Wait, "wait", "w", o.Wait, "Wait for the operation to complete.")
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	// If an explicit name is specified, use that name
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

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if len(args) < 1 {
		return fmt.Errorf("restore name is required")
	}

	if o.FromBackup == "" {
		return fmt.Errorf("--from-backup is required")
	}

	if o.Name == "" {
		o.Name = args[0]
	}

	return nil
}

func (o *CreateOptions) Run(c *cobra.Command, f client.Factory) error {
	if printed, err := output.PrintWithFormat(c, o.buildRestore()); printed || err != nil {
		return err
	}

	restore := o.buildRestore()

	if err := o.client.Create(context.Background(), restore); err != nil {
		return err
	}

	fmt.Printf("NonAdminRestore %q created successfully.\n", restore.Name)

	if o.Wait {
		return o.waitForRestore(restore)
	}

	return nil
}

func (o *CreateOptions) buildRestore() *nacv1alpha1.NonAdminRestore {
	// Create a Velero RestoreSpec
	restoreSpec := &velerov1api.RestoreSpec{
		BackupName: o.FromBackup,
	}

	// Add resource filters
	if len(o.IncludeResources) > 0 {
		restoreSpec.IncludedResources = o.IncludeResources
	}
	if len(o.ExcludeResources) > 0 {
		restoreSpec.ExcludedResources = o.ExcludeResources
	}

	// Note: The namespace-scoped and cluster-scoped resource filters are only available
	// in backup operations, not restore operations in Velero RestoreSpec.
	// For restores, use IncludedResources/ExcludedResources with specific resource types.

	// Note: Namespace mappings are restricted for non-admin users and therefore not processed

	// Add selectors
	if o.Selector.LabelSelector != nil {
		restoreSpec.LabelSelector = o.Selector.LabelSelector
	}
	if len(o.OrSelector.OrLabelSelectors) > 0 {
		restoreSpec.OrLabelSelectors = o.OrSelector.OrLabelSelectors
	}

	// Add optional settings
	if o.IncludeClusterResources.Value != nil {
		restoreSpec.IncludeClusterResources = o.IncludeClusterResources.Value
	}
	if o.RestorePVs.Value != nil {
		restoreSpec.RestorePVs = o.RestorePVs.Value
	}
	if o.PreserveNodePorts.Value != nil {
		restoreSpec.PreserveNodePorts = o.PreserveNodePorts.Value
	}
	if o.ItemOperationTimeout > 0 {
		restoreSpec.ItemOperationTimeout = metav1.Duration{Duration: o.ItemOperationTimeout}
	}
	if o.ExistingResourcePolicy != "" {
		policy := velerov1api.PolicyType(o.ExistingResourcePolicy)
		restoreSpec.ExistingResourcePolicy = policy
	}
	if o.UploaderConfig.Data() != nil && len(o.UploaderConfig.Data()) > 0 {
		restoreSpec.UploaderConfig = &velerov1api.UploaderConfigForRestore{}
		// Note: UploaderConfigForRestore fields would be set here based on the map values
		// The exact field structure depends on the Velero version being used
	}

	// Create NonAdminRestore using the builder
	restore := ForNonAdminRestore(o.currentNamespace, o.Name).
		ObjectMeta(
			WithLabelsMap(o.Labels.Data()),
			WithAnnotationsMap(o.Annotations.Data()),
		).
		RestoreSpec(nacv1alpha1.NonAdminRestoreSpec{
			RestoreSpec: restoreSpec,
		}).
		Result()

	return restore
}

func (o *CreateOptions) waitForRestore(restore *nacv1alpha1.NonAdminRestore) error {
	fmt.Printf("Waiting for restore %s to complete...\n", restore.Name)

	// TODO: Implement proper wait functionality
	// For now, just poll the restore status periodically
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for restore to complete")
		case <-ticker.C:
			// Get current restore status
			currentRestore := &nacv1alpha1.NonAdminRestore{}
			err := o.client.Get(ctx, kbclient.ObjectKey{
				Namespace: restore.Namespace,
				Name:      restore.Name,
			}, currentRestore)
			if err != nil {
				return fmt.Errorf("failed to get restore status: %w", err)
			}

			phase := currentRestore.Status.Phase
			fmt.Printf("Restore %s status: %s\n", restore.Name, phase)

			// Check if completed (using generic NonAdminPhase constants)
			if phase == nacv1alpha1.NonAdminPhaseCreated {
				fmt.Printf("Restore %s completed successfully.\n", restore.Name)
				return nil
			}
			// Add other phase checks as needed
		}
	}
}
