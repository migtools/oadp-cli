package backup

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DescribeOptions holds options for the describe command
type DescribeOptions struct {
	Details bool
}

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	options := &DescribeOptions{}

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupName := args[0]

			// Get the current namespace from kubectl context
			userNamespace, err := shared.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}

			// Create client with required scheme types
			kbClient, err := shared.NewClientWithScheme(f, shared.ClientOptions{
				IncludeNonAdminTypes: true,
				IncludeVeleroTypes:   true,
				IncludeCoreTypes:     true,
			})
			if err != nil {
				return err
			}

			// Find the NonAdminBackup
			var nabList nacv1alpha1.NonAdminBackupList
			if err := kbClient.List(context.Background(), &nabList, &kbclient.ListOptions{
				Namespace: userNamespace,
			}); err != nil {
				return fmt.Errorf("failed to list NonAdminBackup: %w", err)
			}

			var foundNAB *nacv1alpha1.NonAdminBackup
			for i := range nabList.Items {
				if nabList.Items[i].Name == backupName {
					foundNAB = &nabList.Items[i]
					break
				}
			}

			if foundNAB == nil {
				return fmt.Errorf("NonAdminBackup %q not found in namespace %q", backupName, userNamespace)
			}

			return NonAdminDescribeBackup(cmd, kbClient, foundNAB, userNamespace, options)
		},
		Example: `  # Describe a non-admin backup (concise summary)
  kubectl oadp nonadmin backup describe my-backup
  
  # Describe with complete detailed output (same as Velero)
  kubectl oadp nonadmin backup describe my-backup --details`,
	}

	// Add the --details flag
	c.Flags().BoolVar(&options.Details, "details", false, "Show complete detailed output (same as Velero backup describe --details)")

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// NonAdminDescribeBackup provides a Velero-style detailed output format
// but works within non-admin RBAC boundaries using NonAdminDownloadRequest
func NonAdminDescribeBackup(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, userNamespace string, options *DescribeOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if options.Details {
		// Show the full Velero-style detailed output using filtering approach
		return NonAdminDescribeBackupDetailed(cmd, kbClient, nab, userNamespace, ctx)
	} else {
		// Show a concise summary
		return NonAdminDescribeBackupSummary(cmd, kbClient, nab, userNamespace, ctx)
	}
}

// NonAdminDescribeBackupSummary provides a concise backup summary
func NonAdminDescribeBackupSummary(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, _ string, ctx context.Context) error {
	_ = ctx // Context not currently used but kept for future use

	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", nab.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", nab.Namespace)
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:        %s\n", nab.Status.Phase)

	// Basic timing if available
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status
		if vStatus.StartTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Started:      %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.CompletionTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Completed:    %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.Progress != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Items backed up: %d\n", vStatus.Progress.ItemsBackedUp)
		}
		if vStatus.Errors > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Errors:       %d\n", vStatus.Errors)
		}
		if vStatus.Warnings > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Warnings:     %d\n", vStatus.Warnings)
		}
	}

	return nil
}

// NonAdminDescribeBackupDetailed leverages Velero's native describe logic with filtering for non-admin users
func NonAdminDescribeBackupDetailed(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, _ string, ctx context.Context) error {
	// Step 1: Get the underlying Velero Backup
	veleroBackup, err := getVeleroBackupFromNAB(nab, kbClient, ctx)
	if err != nil {
		return fmt.Errorf("failed to get Velero backup: %w", err)
	}

	// Step 2: Create our own detailed output by filtering Velero-style information
	filteredOutput := createFilteredVeleroOutput(veleroBackup, nab)

	// Step 3: Present the refined output
	fmt.Fprint(cmd.OutOrStdout(), filteredOutput)
	return nil
}

// getVeleroBackupFromNAB retrieves the underlying Velero Backup from NonAdminBackup
func getVeleroBackupFromNAB(nab *nacv1alpha1.NonAdminBackup, kbClient kbclient.Client, ctx context.Context) (*velerov1.Backup, error) {
	if nab.Status.VeleroBackup == nil || nab.Status.VeleroBackup.Name == "" {
		return nil, fmt.Errorf("no Velero backup associated with NonAdminBackup %s", nab.Name)
	}

	veleroBackupName := nab.Status.VeleroBackup.Name
	veleroNamespace := nab.Status.VeleroBackup.Namespace
	if veleroNamespace == "" {
		veleroNamespace = "openshift-adp" // Default OADP namespace
	}

	var veleroBackup velerov1.Backup
	err := kbClient.Get(ctx, kbclient.ObjectKey{
		Namespace: veleroNamespace,
		Name:      veleroBackupName,
	}, &veleroBackup)

	if err != nil {
		return nil, fmt.Errorf("failed to get Velero backup %s/%s: %w", veleroNamespace, veleroBackupName, err)
	}

	return &veleroBackup, nil
}

// createFilteredVeleroOutput creates a Velero-style output with non-admin field restrictions applied
func createFilteredVeleroOutput(veleroBackup *velerov1.Backup, nab *nacv1alpha1.NonAdminBackup) string {
	var output strings.Builder

	// Build output sections
	writeBasicInfo(&output, nab)
	writeMetadata(&output, nab)
	writePhaseAndErrors(&output, nab)
	writeAdminEnforceableFields(&output, veleroBackup, nab)
	writeStatusInformation(&output, nab)

	return output.String()
}

// writeBasicInfo writes the basic name and namespace information
func writeBasicInfo(output *strings.Builder, nab *nacv1alpha1.NonAdminBackup) {
	output.WriteString(fmt.Sprintf("Name:         %s\n", nab.Name))
	output.WriteString(fmt.Sprintf("Namespace:    %s\n", nab.Namespace))
}

// writeMetadata writes labels and annotations in Velero-style format
func writeMetadata(output *strings.Builder, nab *nacv1alpha1.NonAdminBackup) {
	writeKeyValuePairs(output, "Labels", nab.Labels)
	writeKeyValuePairs(output, "Annotations", nab.Annotations)
	output.WriteString("\n")
}

// writeKeyValuePairs formats and writes key-value pairs (labels or annotations)
func writeKeyValuePairs(output *strings.Builder, fieldName string, pairs map[string]string) {
	if len(pairs) > 0 {
		output.WriteString(fmt.Sprintf("%-13s ", fieldName+":"))

		pairStrings := make([]string, 0, len(pairs))
		for k, v := range pairs {
			pairStrings = append(pairStrings, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(pairStrings)
		output.WriteString(fmt.Sprintf("%s\n", strings.Join(pairStrings, ",")))
	} else {
		output.WriteString(fmt.Sprintf("%-13s <none>\n", fieldName+":"))
	}
}

// writePhaseAndErrors writes phase and error/warning information
func writePhaseAndErrors(output *strings.Builder, nab *nacv1alpha1.NonAdminBackup) {
	output.WriteString(fmt.Sprintf("Phase:  %s\n", nab.Status.Phase))

	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status
		if vStatus.Errors > 0 {
			output.WriteString(fmt.Sprintf("Errors:    %d\n", vStatus.Errors))
		}
		if vStatus.Warnings > 0 {
			output.WriteString(fmt.Sprintf("Warnings:  %d\n", vStatus.Warnings))
		}
	}

}

// writeAdminEnforceableFields writes all admin enforceable fields
func writeAdminEnforceableFields(output *strings.Builder, veleroBackup *velerov1.Backup, nab *nacv1alpha1.NonAdminBackup) {
	writeTimeoutFields(output, veleroBackup)
	writeResourcePolicyFields(output, veleroBackup)
	writeNamespaceFields(output, veleroBackup)
	writeResourceFields(output, veleroBackup, nab)
	writeClusterResourceFields(output, veleroBackup)
	writeSelectorFields(output, nab)
	writeVolumeFields(output, veleroBackup)
	writeStorageFields(output, veleroBackup)
	writeBackupPolicyFields(output, veleroBackup)
	writeUploaderConfigFields(output, veleroBackup)
	writeHookFields(output, veleroBackup)
}

// writeTimeoutFields writes timeout-related fields
func writeTimeoutFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if veleroBackup.Spec.CSISnapshotTimeout.Duration > 0 {
		output.WriteString(fmt.Sprintf("CSI Snapshot Timeout:      %s\n", veleroBackup.Spec.CSISnapshotTimeout.Duration.String()))
	}
	if veleroBackup.Spec.ItemOperationTimeout.Duration > 0 {
		output.WriteString(fmt.Sprintf("Item Operation Timeout:    %s\n", veleroBackup.Spec.ItemOperationTimeout.Duration.String()))
	}
}

// writeResourcePolicyFields writes resource policy fields
func writeResourcePolicyFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if veleroBackup.Spec.ResourcePolicy != nil {
		output.WriteString(fmt.Sprintf("Resource Policy:           %s\n", veleroBackup.Spec.ResourcePolicy.Name))
	}
}

// writeNamespaceFields writes namespace-related fields
func writeNamespaceFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if len(veleroBackup.Spec.ExcludedNamespaces) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Namespaces:       %s\n", strings.Join(veleroBackup.Spec.ExcludedNamespaces, ", ")))
	}
}

// writeResourceFields writes resource inclusion/exclusion fields
func writeResourceFields(output *strings.Builder, veleroBackup *velerov1.Backup, nab *nacv1alpha1.NonAdminBackup) {
	// Included Resources
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.IncludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Resources:        %s\n", strings.Join(nab.Spec.BackupSpec.IncludedResources, ", ")))
	} else {
		output.WriteString("Included Resources:        * (all)\n")
	}

	// Excluded Resources
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.ExcludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Resources:        %s\n", strings.Join(nab.Spec.BackupSpec.ExcludedResources, ", ")))
	}

	// Ordered Resources
	if len(veleroBackup.Spec.OrderedResources) > 0 {
		output.WriteString("Ordered Resources:         configured\n")
	}
}

// writeClusterResourceFields writes cluster resource fields
func writeClusterResourceFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	// Include Cluster Resources
	if veleroBackup.Spec.IncludeClusterResources != nil {
		output.WriteString(fmt.Sprintf("Include Cluster Resources: %v\n", getBoolPointerValue(veleroBackup.Spec.IncludeClusterResources)))
	} else {
		output.WriteString("Include Cluster Resources: auto\n")
	}

	// Excluded Cluster Scoped Resources
	if len(veleroBackup.Spec.ExcludedClusterScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Cluster Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.ExcludedClusterScopedResources, ", ")))
	}

	// Included Cluster Scoped Resources
	if len(veleroBackup.Spec.IncludedClusterScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Cluster Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.IncludedClusterScopedResources, ", ")))
	}

	// Excluded Namespace Scoped Resources
	if len(veleroBackup.Spec.ExcludedNamespaceScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Namespace Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.ExcludedNamespaceScopedResources, ", ")))
	}

	// Included Namespace Scoped Resources
	if len(veleroBackup.Spec.IncludedNamespaceScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Namespace Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.IncludedNamespaceScopedResources, ", ")))
	}
}

// writeSelectorFields writes label selector fields
func writeSelectorFields(output *strings.Builder, nab *nacv1alpha1.NonAdminBackup) {
	if nab.Spec.BackupSpec != nil && nab.Spec.BackupSpec.LabelSelector != nil {
		output.WriteString(fmt.Sprintf("Label Selector:            %v\n", nab.Spec.BackupSpec.LabelSelector))
	}
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.OrLabelSelectors) > 0 {
		output.WriteString(fmt.Sprintf("Or Label Selectors:        %v\n", nab.Spec.BackupSpec.OrLabelSelectors))
	}
}

// writeVolumeFields writes volume-related fields
func writeVolumeFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	output.WriteString(fmt.Sprintf("Snapshot Volumes:          %v\n", getSnapshotVolumesValue(veleroBackup.Spec.SnapshotVolumes)))
}

// writeStorageFields writes storage-related fields
func writeStorageFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if veleroBackup.Spec.StorageLocation != "" {
		output.WriteString(fmt.Sprintf("Storage Location:          %s\n", veleroBackup.Spec.StorageLocation))
	}

	if len(veleroBackup.Spec.VolumeSnapshotLocations) > 0 {
		output.WriteString(fmt.Sprintf("Volume Snapshot Locations: %s\n", strings.Join(veleroBackup.Spec.VolumeSnapshotLocations, ", ")))
	} else {
		output.WriteString("Volume Snapshot Locations: default\n")
	}
}

// writeBackupPolicyFields writes backup policy fields
func writeBackupPolicyFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if veleroBackup.Spec.TTL.Duration > 0 {
		output.WriteString(fmt.Sprintf("TTL:                       %s\n", veleroBackup.Spec.TTL.Duration.String()))
	}

	if veleroBackup.Spec.DefaultVolumesToFsBackup != nil {
		output.WriteString(fmt.Sprintf("Default Volumes to FS Backup: %v\n", getBoolPointerValue(veleroBackup.Spec.DefaultVolumesToFsBackup)))
	}

	if veleroBackup.Spec.SnapshotMoveData != nil {
		output.WriteString(fmt.Sprintf("Snapshot Move Data:        %v\n", getBoolPointerValue(veleroBackup.Spec.SnapshotMoveData)))
	}

	if veleroBackup.Spec.DataMover != "" {
		output.WriteString(fmt.Sprintf("Data Mover:                %s\n", veleroBackup.Spec.DataMover))
	}
}

// writeUploaderConfigFields writes uploader configuration fields
func writeUploaderConfigFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if veleroBackup.Spec.UploaderConfig != nil {
		output.WriteString("Uploader Config:           configured\n")
		if veleroBackup.Spec.UploaderConfig.ParallelFilesUpload > 0 {
			output.WriteString(fmt.Sprintf("  Parallel Files Upload:   %d\n", veleroBackup.Spec.UploaderConfig.ParallelFilesUpload))
		}
	}
}

// writeHookFields writes hook-related fields
func writeHookFields(output *strings.Builder, veleroBackup *velerov1.Backup) {
	if len(veleroBackup.Spec.Hooks.Resources) > 0 {
		output.WriteString(fmt.Sprintf("Hooks:                     %d hook(s) configured\n", len(veleroBackup.Spec.Hooks.Resources)))
	}
}

// writeStatusInformation writes the status information section
func writeStatusInformation(output *strings.Builder, nab *nacv1alpha1.NonAdminBackup) {
	if nab.Status.VeleroBackup == nil || nab.Status.VeleroBackup.Status == nil {
		return
	}

	vStatus := nab.Status.VeleroBackup.Status
	output.WriteString("=== STATUS INFORMATION ===\n")

	writeFormatAndTimestamps(output, vStatus)
	writeProgressInformation(output, vStatus)
	writeResourceList(output, vStatus)
	writeVolumeInformation(output, vStatus, nab)
	writeHookStatus(output, vStatus)
}

// writeFormatAndTimestamps writes backup format version and timestamps
func writeFormatAndTimestamps(output *strings.Builder, vStatus *velerov1.BackupStatus) {
	if vStatus.FormatVersion != "" {
		output.WriteString(fmt.Sprintf("Backup Format Version:     %s\n", vStatus.FormatVersion))
	}

	if vStatus.StartTimestamp != nil {
		output.WriteString(fmt.Sprintf("Started:                   %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST")))
	}
	if vStatus.CompletionTimestamp != nil {
		output.WriteString(fmt.Sprintf("Completed:                 %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST")))
	}
	if vStatus.Expiration != nil {
		output.WriteString(fmt.Sprintf("Expiration:                %s\n", vStatus.Expiration.Format("2006-01-02 15:04:05 -0700 MST")))
	}

	output.WriteString("\n")
}

// writeProgressInformation writes backup progress information
func writeProgressInformation(output *strings.Builder, vStatus *velerov1.BackupStatus) {
	if vStatus.Progress != nil {
		output.WriteString(fmt.Sprintf("Total items to be backed up: %d\n", vStatus.Progress.TotalItems))
		output.WriteString(fmt.Sprintf("Items backed up:           %d\n", vStatus.Progress.ItemsBackedUp))
	}
	output.WriteString("\n")
}

// writeResourceList writes the resource list information
func writeResourceList(output *strings.Builder, vStatus *velerov1.BackupStatus) {
	output.WriteString("Resource List:\n")
	if vStatus.Progress != nil {
		output.WriteString(fmt.Sprintf("  Total items backed up:   %d\n", vStatus.Progress.ItemsBackedUp))
	} else {
		output.WriteString("  <detailed resource breakdown not available>\n")
	}
	output.WriteString("\n")
}

// writeVolumeInformation writes backup volume information
func writeVolumeInformation(output *strings.Builder, vStatus *velerov1.BackupStatus, nab *nacv1alpha1.NonAdminBackup) {
	output.WriteString("Backup Volumes:\n")

	// Velero-Native Snapshots
	if vStatus.VolumeSnapshotsCompleted > 0 {
		output.WriteString(fmt.Sprintf("  Velero-Native Snapshots: <%d included>\n", vStatus.VolumeSnapshotsCompleted))
	} else {
		output.WriteString("  Velero-Native Snapshots: <none included>\n")
	}

	// CSI Snapshots
	if vStatus.CSIVolumeSnapshotsCompleted > 0 {
		output.WriteString(fmt.Sprintf("  CSI Snapshots:           <%d included>\n", vStatus.CSIVolumeSnapshotsCompleted))
	} else {
		output.WriteString("  CSI Snapshots:           <none included>\n")
	}

	// Pod Volume Backups
	if nab.Status.FileSystemPodVolumeBackups != nil && nab.Status.FileSystemPodVolumeBackups.Completed > 0 {
		output.WriteString(fmt.Sprintf("  Pod Volume Backups:      <%d included>\n", nab.Status.FileSystemPodVolumeBackups.Completed))
	} else {
		output.WriteString("  Pod Volume Backups:      <none included>\n")
	}

	output.WriteString("\n")
}

// writeHookStatus writes hook status information
func writeHookStatus(output *strings.Builder, vStatus *velerov1.BackupStatus) {
	output.WriteString(fmt.Sprintf("Hooks Attempted:           %d\n", vStatus.HookStatus.HooksAttempted))
	output.WriteString(fmt.Sprintf("Hooks Failed:              %d\n", vStatus.HookStatus.HooksFailed))
}

// Helper functions for the detailed output
func getSnapshotVolumesValue(snapshots *bool) string {
	if snapshots == nil {
		return "auto"
	}
	if *snapshots {
		return "true"
	}
	return "false"
}

func getBoolPointerValue(b *bool) string {
	if b == nil {
		return "auto"
	}
	if *b {
		return "true"
	}
	return "false"
}
