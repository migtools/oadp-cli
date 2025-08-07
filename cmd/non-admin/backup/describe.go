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

	// Header in Velero style
	output.WriteString(fmt.Sprintf("Name:         %s\n", nab.Name))
	output.WriteString(fmt.Sprintf("Namespace:    %s\n", nab.Namespace))

	// Labels (Velero-style format) - Admin Enforceable: Yes
	if len(nab.Labels) > 0 {
		output.WriteString("Labels:       ")
		labelPairs := make([]string, 0, len(nab.Labels))
		for k, v := range nab.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(labelPairs)
		output.WriteString(fmt.Sprintf("%s\n", strings.Join(labelPairs, ",")))
	} else {
		output.WriteString("Labels:       <none>\n")
	}

	// Annotations (Velero-style format) - Admin Enforceable: Yes
	if len(nab.Annotations) > 0 {
		output.WriteString("Annotations:  ")
		annotationPairs := make([]string, 0, len(nab.Annotations))
		for k, v := range nab.Annotations {
			annotationPairs = append(annotationPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(annotationPairs)
		output.WriteString(fmt.Sprintf("%s\n", strings.Join(annotationPairs, ",")))
	} else {
		output.WriteString("Annotations:  <none>\n")
	}

	output.WriteString("\n")

	// Phase/Status information - Admin Enforceable: Yes
	output.WriteString(fmt.Sprintf("Phase:  %s\n", nab.Status.Phase))

	// Add error/warning information if available - Admin Enforceable: Yes
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status
		if vStatus.Errors > 0 {
			output.WriteString(fmt.Sprintf("Errors:    %d\n", vStatus.Errors))
		}
		if vStatus.Warnings > 0 {
			output.WriteString(fmt.Sprintf("Warnings:  %d\n", vStatus.Warnings))
		}
	}

	output.WriteString("\n")

	// === ADMIN ENFORCEABLE FIELDS (only show if configured) ===

	// CSISnapshotTimeout - Admin Enforceable: Yes
	if veleroBackup.Spec.CSISnapshotTimeout.Duration > 0 {
		output.WriteString(fmt.Sprintf("CSI Snapshot Timeout:      %s\n", veleroBackup.Spec.CSISnapshotTimeout.Duration.String()))
	}

	// ItemOperationTimeout - Admin Enforceable: Yes
	if veleroBackup.Spec.ItemOperationTimeout.Duration > 0 {
		output.WriteString(fmt.Sprintf("Item Operation Timeout:    %s\n", veleroBackup.Spec.ItemOperationTimeout.Duration.String()))
	}

	// ResourcePolicy - Admin Enforceable: Yes (special case - admins can enforce config-map in OADP Operator NS)
	if veleroBackup.Spec.ResourcePolicy != nil {
		output.WriteString(fmt.Sprintf("Resource Policy:           %s\n", veleroBackup.Spec.ResourcePolicy.Name))
	}

	// ExcludedNamespaces - Admin Enforceable: Yes, Restricted: Yes (special case - restricted but admin enforceable)
	if len(veleroBackup.Spec.ExcludedNamespaces) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Namespaces:       %s\n", strings.Join(veleroBackup.Spec.ExcludedNamespaces, ", ")))
	}

	// IncludedResources - Admin Enforceable: Yes
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.IncludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Resources:        %s\n", strings.Join(nab.Spec.BackupSpec.IncludedResources, ", ")))
	} else {
		output.WriteString("Included Resources:        * (all)\n")
	}

	// ExcludedResources - Admin Enforceable: Yes
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.ExcludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Resources:        %s\n", strings.Join(nab.Spec.BackupSpec.ExcludedResources, ", ")))
	}

	// OrderedResources - Admin Enforceable: Yes
	if len(veleroBackup.Spec.OrderedResources) > 0 {
		output.WriteString("Ordered Resources:         configured\n")
	}

	// IncludeClusterResources - Admin Enforceable: Yes (special case - non-admin users can only set to false)
	if veleroBackup.Spec.IncludeClusterResources != nil {
		output.WriteString(fmt.Sprintf("Include Cluster Resources: %v\n", getBoolPointerValue(veleroBackup.Spec.IncludeClusterResources)))
	} else {
		output.WriteString("Include Cluster Resources: auto\n")
	}

	// ExcludedClusterScopedResources - Admin Enforceable: Yes
	if len(veleroBackup.Spec.ExcludedClusterScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Cluster Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.ExcludedClusterScopedResources, ", ")))
	}

	// IncludedClusterScopedResources - Admin Enforceable: Yes (special case - only empty list acceptable)
	if len(veleroBackup.Spec.IncludedClusterScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Cluster Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.IncludedClusterScopedResources, ", ")))
	}

	// ExcludedNamespaceScopedResources - Admin Enforceable: Yes
	if len(veleroBackup.Spec.ExcludedNamespaceScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Namespace Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.ExcludedNamespaceScopedResources, ", ")))
	}

	// IncludedNamespaceScopedResources - Admin Enforceable: Yes
	if len(veleroBackup.Spec.IncludedNamespaceScopedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Namespace Scoped Resources: %s\n", strings.Join(veleroBackup.Spec.IncludedNamespaceScopedResources, ", ")))
	}

	// LabelSelector - Admin Enforceable: Yes
	if nab.Spec.BackupSpec != nil && nab.Spec.BackupSpec.LabelSelector != nil {
		output.WriteString(fmt.Sprintf("Label Selector:            %v\n", nab.Spec.BackupSpec.LabelSelector))
	}

	// OrLabelSelectors - Admin Enforceable: Yes
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.OrLabelSelectors) > 0 {
		output.WriteString(fmt.Sprintf("Or Label Selectors:        %v\n", nab.Spec.BackupSpec.OrLabelSelectors))
	}

	// SnapshotVolumes - Admin Enforceable: Yes
	output.WriteString(fmt.Sprintf("Snapshot Volumes:          %v\n", getSnapshotVolumesValue(veleroBackup.Spec.SnapshotVolumes)))

	// StorageLocation - Admin Enforceable: (blank/unspecified) - should point to existing NABSL
	if veleroBackup.Spec.StorageLocation != "" {
		output.WriteString(fmt.Sprintf("Storage Location:          %s\n", veleroBackup.Spec.StorageLocation))
	}

	// VolumeSnapshotLocations - Admin Enforceable: (blank/unspecified) - not supported for non-admin users
	if len(veleroBackup.Spec.VolumeSnapshotLocations) > 0 {
		output.WriteString(fmt.Sprintf("Volume Snapshot Locations: %s\n", strings.Join(veleroBackup.Spec.VolumeSnapshotLocations, ", ")))
	} else {
		output.WriteString("Volume Snapshot Locations: default\n")
	}

	// TTL - Admin Enforceable: Yes
	if veleroBackup.Spec.TTL.Duration > 0 {
		output.WriteString(fmt.Sprintf("TTL:                       %s\n", veleroBackup.Spec.TTL.Duration.String()))
	}

	// DefaultVolumesToFsBackup - Admin Enforceable: Yes
	if veleroBackup.Spec.DefaultVolumesToFsBackup != nil {
		output.WriteString(fmt.Sprintf("Default Volumes to FS Backup: %v\n", getBoolPointerValue(veleroBackup.Spec.DefaultVolumesToFsBackup)))
	}

	// SnapshotMoveData - Admin Enforceable: Yes
	if veleroBackup.Spec.SnapshotMoveData != nil {
		output.WriteString(fmt.Sprintf("Snapshot Move Data:        %v\n", getBoolPointerValue(veleroBackup.Spec.SnapshotMoveData)))
	}

	// DataMover - Admin Enforceable: Yes
	if veleroBackup.Spec.DataMover != "" {
		output.WriteString(fmt.Sprintf("Data Mover:                %s\n", veleroBackup.Spec.DataMover))
	}

	// UploaderConfig.ParallelFilesUpload - Admin Enforceable: Yes
	if veleroBackup.Spec.UploaderConfig != nil {
		output.WriteString("Uploader Config:           configured\n")
		if veleroBackup.Spec.UploaderConfig.ParallelFilesUpload > 0 {
			output.WriteString(fmt.Sprintf("  Parallel Files Upload:   %d\n", veleroBackup.Spec.UploaderConfig.ParallelFilesUpload))
		}
	}

	// Hooks - Admin Enforceable: Yes
	if len(veleroBackup.Spec.Hooks.Resources) > 0 {
		output.WriteString(fmt.Sprintf("Hooks:                     %d hook(s) configured\n", len(veleroBackup.Spec.Hooks.Resources)))
	}

	output.WriteString("\n")
	output.WriteString("=== RESTRICTED FIELDS (Not shown for non-admin users) ===\n")
	output.WriteString("Included Namespaces:       [RESTRICTED]\n")

	output.WriteString("\n")

	// Backup format and timing - Status information (always shown when available)
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status

		output.WriteString("=== STATUS INFORMATION ===\n")

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

		// Progress information
		if vStatus.Progress != nil {
			output.WriteString(fmt.Sprintf("Total items to be backed up: %d\n", vStatus.Progress.TotalItems))
			output.WriteString(fmt.Sprintf("Items backed up:           %d\n", vStatus.Progress.ItemsBackedUp))
		}

		output.WriteString("\n")

		// Resource List
		output.WriteString("Resource List:\n")
		if vStatus.Progress != nil {
			output.WriteString(fmt.Sprintf("  Total items backed up:   %d\n", vStatus.Progress.ItemsBackedUp))
		} else {
			output.WriteString("  <detailed resource breakdown not available>\n")
		}

		output.WriteString("\n")

		// Volume information
		output.WriteString("Backup Volumes:\n")
		if vStatus.VolumeSnapshotsCompleted > 0 {
			output.WriteString(fmt.Sprintf("  Velero-Native Snapshots: <%d included>\n", vStatus.VolumeSnapshotsCompleted))
		} else {
			output.WriteString("  Velero-Native Snapshots: <none included>\n")
		}
		if vStatus.CSIVolumeSnapshotsCompleted > 0 {
			output.WriteString(fmt.Sprintf("  CSI Snapshots:           <%d included>\n", vStatus.CSIVolumeSnapshotsCompleted))
		} else {
			output.WriteString("  CSI Snapshots:           <none included>\n")
		}
		if nab.Status.FileSystemPodVolumeBackups != nil && nab.Status.FileSystemPodVolumeBackups.Completed > 0 {
			output.WriteString(fmt.Sprintf("  Pod Volume Backups:      <%d included>\n", nab.Status.FileSystemPodVolumeBackups.Completed))
		} else {
			output.WriteString("  Pod Volume Backups:      <none included>\n")
		}

		output.WriteString("\n")

		// Hook status information
		output.WriteString(fmt.Sprintf("Hooks Attempted:           %d\n", vStatus.HookStatus.HooksAttempted))
		output.WriteString(fmt.Sprintf("Hooks Failed:              %d\n", vStatus.HookStatus.HooksFailed))
	}

	return output.String()
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
