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

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	velerobackup "github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

func NewCreateCommand(f client.Factory, use string) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Create a non-admin backup",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a simple backup of all resources in the current namespace.
  kubectl oadp nonadmin backup create backup1

  # Create a backup with specific resource types.
  kubectl oadp nonadmin backup create backup2 --include-resources deployments,services

  # Create a backup with label selector.
  kubectl oadp nonadmin backup create backup3 --selector app=myapp

  # Create a backup with snapshots and TTL.
  kubectl oadp nonadmin backup create backup4 --snapshot-volumes --ttl 720h

  # Create a backup with specific storage location.
  kubectl oadp nonadmin backup create backup5 --storage-location my-nabsl

  # View the YAML for a backup without sending it to the server.
  kubectl oadp nonadmin backup create backup6 -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	*velerobackup.CreateOptions // Embed Velero's CreateOptions

	// NAB-specific fields
	Name             string // The NonAdminBackup resource name (maps to Velero's BackupName)
	client           kbclient.WithWatch
	currentNamespace string
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		CreateOptions: velerobackup.NewCreateOptions(),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	// Resource filtering (MVP)
	flags.Var(&o.IncludeResources, "include-resources", "Resources to include in the backup, formatted as resource.group, such as storageclasses.storage.k8s.io (use '*' for all resources).")
	flags.Var(&o.ExcludeResources, "exclude-resources", "Resources to exclude from the backup, formatted as resource.group, such as storageclasses.storage.k8s.io.")

	// Label selection (MVP)
	flags.VarP(&o.Selector, "selector", "l", "Only back up resources matching this label selector.")
	flags.Var(&o.OrSelector, "or-selector", "Backup resources matching at least one of the label selector from the list. Label selectors should be separated by ' or '. For example, foo=bar or app=nginx")

	// Cluster resources (MVP)
	f := flags.VarPF(&o.IncludeClusterResources, "include-cluster-resources", "", "Include cluster-scoped resources in the backup.")
	f.NoOptDefVal = cmd.TRUE

	// Timing/Storage (MVP)
	flags.DurationVar(&o.TTL, "ttl", o.TTL, "How long before the backup can be garbage collected.")
	flags.StringVar(&o.StorageLocation, "storage-location", "", "Location in which to store the backup.")
	flags.DurationVar(&o.CSISnapshotTimeout, "csi-snapshot-timeout", o.CSISnapshotTimeout, "How long to wait for CSI snapshot creation before timeout.")
	flags.DurationVar(&o.ItemOperationTimeout, "item-operation-timeout", o.ItemOperationTimeout, "How long to wait for async plugin operations before timeout.")

	// Snapshot control (MVP)
	f = flags.VarPF(&o.SnapshotVolumes, "snapshot-volumes", "", "Take snapshots of PersistentVolumes as part of the backup.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.SnapshotMoveData, "snapshot-move-data", "", "Specify whether snapshot data should be moved.")
	f.NoOptDefVal = cmd.TRUE

	f = flags.VarPF(&o.DefaultVolumesToFsBackup, "default-volumes-to-fs-backup", "", "Use pod volume file system backup by default for volumes.")
	f.NoOptDefVal = cmd.TRUE
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if err := output.ValidateFlags(c); err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("a backup name is required")
	}

	if o.Selector.LabelSelector != nil && o.OrSelector.OrLabelSelectors != nil {
		return fmt.Errorf("either a 'selector' or an 'or-selector' can be specified, but not both")
	}

	// Storage location validation
	if o.StorageLocation == "" {
		return fmt.Errorf("--storage-location is required")
	}

	return nil
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	o.Name = args[0]

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
	nonAdminBackup, err := o.BuildNonAdminBackup(o.currentNamespace)
	if err != nil {
		return err
	}

	if printed, err := output.PrintWithFormat(c, nonAdminBackup); printed || err != nil {
		return err
	}

	// Create the backup
	if err := o.client.Create(context.TODO(), nonAdminBackup, &kbclient.CreateOptions{}); err != nil {
		return err
	}

	fmt.Printf("NonAdminBackup request %q submitted successfully.\n", nonAdminBackup.Name)
	fmt.Printf("Run `oc oadp nonadmin backup describe %s` or `oc oadp nonadmin backup logs %s` for more details.\n", nonAdminBackup.Name, nonAdminBackup.Name)
	return nil
}

func (o *CreateOptions) BuildNonAdminBackup(namespace string) (*nacv1alpha1.NonAdminBackup, error) {
	backupSpec, err := o.buildBackupSpecFromOptions(namespace)
	if err != nil {
		return nil, err
	}

	return o.createNonAdminBackup(namespace, backupSpec), nil
}

// buildBackupSpecFromOptions creates a BackupSpec from command line options
func (o *CreateOptions) buildBackupSpecFromOptions(namespace string) (*velerov1api.BackupSpec, error) {
	backupBuilder := builder.ForBackup(namespace, o.Name).
		IncludedNamespaces(namespace). // Automatically include the current namespace
		IncludedResources(o.IncludeResources...).
		ExcludedResources(o.ExcludeResources...).
		LabelSelector(o.Selector.LabelSelector).
		OrLabelSelector(o.OrSelector.OrLabelSelectors).
		TTL(o.TTL).
		StorageLocation(o.StorageLocation).
		CSISnapshotTimeout(o.CSISnapshotTimeout).
		ItemOperationTimeout(o.ItemOperationTimeout)

	if err := o.applyOptionalBackupOptions(backupBuilder); err != nil {
		return nil, err
	}

	tempBackup := backupBuilder.Result()

	return &tempBackup.Spec, nil
}

// applyOptionalBackupOptions applies optional flags to the backup builder
func (o *CreateOptions) applyOptionalBackupOptions(backupBuilder *builder.BackupBuilder) error {
	if o.SnapshotVolumes.Value != nil {
		backupBuilder.SnapshotVolumes(*o.SnapshotVolumes.Value)
	}
	if o.SnapshotMoveData.Value != nil {
		backupBuilder.SnapshotMoveData(*o.SnapshotMoveData.Value)
	}
	if o.IncludeClusterResources.Value != nil {
		backupBuilder.IncludeClusterResources(*o.IncludeClusterResources.Value)
	}
	if o.DefaultVolumesToFsBackup.Value != nil {
		backupBuilder.DefaultVolumesToFsBackup(*o.DefaultVolumesToFsBackup.Value)
	}

	return nil
}

// createNonAdminBackup creates the NonAdminBackup CR from a BackupSpec
func (o *CreateOptions) createNonAdminBackup(namespace string, backupSpec *velerov1api.BackupSpec) *nacv1alpha1.NonAdminBackup {
	return ForNonAdminBackup(namespace, o.Name).
		BackupSpec(nacv1alpha1.NonAdminBackupSpec{
			BackupSpec: backupSpec,
		}).
		Result()
}
