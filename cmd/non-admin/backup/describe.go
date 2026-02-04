package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	var (
		requestTimeout time.Duration
		details        bool
	)

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupName := args[0]

			// Get effective timeout (flag takes precedence over env var)
			effectiveTimeout := shared.GetHTTPTimeoutWithOverride(requestTimeout)

			// Create context with the effective timeout
			ctx, cancel := context.WithTimeout(context.Background(), effectiveTimeout)
			defer cancel()

			// Get the current namespace from kubectl context
			userNamespace, err := shared.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}

			// Create client with required scheme types and timeout
			kbClient, err := shared.NewClientWithScheme(f, shared.ClientOptions{
				IncludeNonAdminTypes: true,
				IncludeVeleroTypes:   true,
				IncludeCoreTypes:     true,
				Timeout:              effectiveTimeout,
			})
			if err != nil {
				return err
			}

			// Get the specific backup
			var nab nacv1alpha1.NonAdminBackup
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: userNamespace,
				Name:      backupName,
			}, &nab); err != nil {
				// Check for context cancellation
				if ctx.Err() == context.DeadlineExceeded {
					return fmt.Errorf("timed out after %v getting NonAdminBackup %q", effectiveTimeout, backupName)
				}
				if ctx.Err() == context.Canceled {
					return fmt.Errorf("operation cancelled: %w", ctx.Err())
				}
				return fmt.Errorf("NonAdminBackup %q not found in namespace %q: %w", backupName, userNamespace, err)
			}

			// Print in Velero-style format
			printNonAdminBackupDetails(cmd, &nab, kbClient, backupName, userNamespace, effectiveTimeout)

			// Add detailed output if --details flag is set
			if details {
				if err := printDetailedBackupInfo(cmd, kbClient, backupName, userNamespace, effectiveTimeout); err != nil {
					return fmt.Errorf("failed to fetch detailed backup information: %w", err)
				}
			}

			return nil
		},
		Example: `  kubectl oadp nonadmin backup describe my-backup
  kubectl oadp nonadmin backup describe my-backup --details
  kubectl oadp nonadmin backup describe my-backup --details --request-timeout=30m`,
	}

	c.Flags().DurationVar(&requestTimeout, "request-timeout", 0, fmt.Sprintf("The length of time to wait before giving up on a single server request (e.g., 30s, 5m, 1h). Overrides %s env var. Default: %v", shared.TimeoutEnvVar, shared.DefaultHTTPTimeout))
	c.Flags().BoolVar(&details, "details", false, "Display additional backup details including volume snapshots, resource lists, and item operations")

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// printNonAdminBackupDetails prints backup details in Velero admin describe format
func printNonAdminBackupDetails(cmd *cobra.Command, nab *nacv1alpha1.NonAdminBackup, kbClient kbclient.Client, backupName string, userNamespace string, timeout time.Duration) {
	out := cmd.OutOrStdout()

	// Get Velero backup reference if available
	var vb *nacv1alpha1.VeleroBackup
	if nab.Status.VeleroBackup != nil {
		vb = nab.Status.VeleroBackup
	}

	// Name and Namespace
	fmt.Fprintf(out, "Name:         %s\n", nab.Name)
	fmt.Fprintf(out, "Namespace:    %s\n", nab.Namespace)

	// Labels
	fmt.Fprintf(out, "Labels:       ")
	if len(nab.Labels) == 0 {
		fmt.Fprintf(out, "<none>\n")
	} else {
		labelKeys := make([]string, 0, len(nab.Labels))
		for k := range nab.Labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)
		for i, k := range labelKeys {
			if i == 0 {
				fmt.Fprintf(out, "%s=%s\n", k, nab.Labels[k])
			} else {
				fmt.Fprintf(out, "              %s=%s\n", k, nab.Labels[k])
			}
		}
	}

	// Annotations
	fmt.Fprintf(out, "Annotations:  ")
	if len(nab.Annotations) == 0 {
		fmt.Fprintf(out, "<none>\n")
	} else {
		annotationKeys := make([]string, 0, len(nab.Annotations))
		for k := range nab.Annotations {
			annotationKeys = append(annotationKeys, k)
		}
		sort.Strings(annotationKeys)
		for i, k := range annotationKeys {
			if i == 0 {
				fmt.Fprintf(out, "%s=%s\n", k, nab.Annotations[k])
			} else {
				fmt.Fprintf(out, "              %s=%s\n", k, nab.Annotations[k])
			}
		}
	}

	fmt.Fprintf(out, "\n")

	// Phase (with color)
	phase := string(nab.Status.Phase)
	if vb != nil && vb.Status != nil && vb.Status.Phase != "" {
		phase = string(vb.Status.Phase)
	}
	fmt.Fprintf(out, "Phase:  %s\n", colorizePhase(phase))

	fmt.Fprintf(out, "\n")

	// Backup Spec details
	if nab.Spec.BackupSpec != nil {
		spec := nab.Spec.BackupSpec

		// Namespaces
		fmt.Fprintf(out, "Namespaces:\n")
		if len(spec.IncludedNamespaces) == 0 {
			fmt.Fprintf(out, "  Included:  *\n")
		} else {
			fmt.Fprintf(out, "  Included:  %s\n", strings.Join(spec.IncludedNamespaces, ", "))
		}
		if len(spec.ExcludedNamespaces) == 0 {
			fmt.Fprintf(out, "  Excluded:  <none>\n")
		} else {
			fmt.Fprintf(out, "  Excluded:  %s\n", strings.Join(spec.ExcludedNamespaces, ", "))
		}

		fmt.Fprintf(out, "\n")

		// Resources
		fmt.Fprintf(out, "Resources:\n")
		if len(spec.IncludedResources) == 0 {
			fmt.Fprintf(out, "  Included:        *\n")
		} else {
			fmt.Fprintf(out, "  Included:        %s\n", strings.Join(spec.IncludedResources, ", "))
		}
		if len(spec.ExcludedResources) == 0 {
			fmt.Fprintf(out, "  Excluded:        <none>\n")
		} else {
			fmt.Fprintf(out, "  Excluded:        %s\n", strings.Join(spec.ExcludedResources, ", "))
		}
		if spec.IncludeClusterResources != nil {
			if *spec.IncludeClusterResources {
				fmt.Fprintf(out, "  Cluster-scoped:  included\n")
			} else {
				fmt.Fprintf(out, "  Cluster-scoped:  excluded\n")
			}
		} else {
			fmt.Fprintf(out, "  Cluster-scoped:  auto\n")
		}

		fmt.Fprintf(out, "\n")

		// Label selector
		if spec.LabelSelector != nil && len(spec.LabelSelector.MatchLabels) > 0 {
			var selectorParts []string
			for k, v := range spec.LabelSelector.MatchLabels {
				selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
			}
			fmt.Fprintf(out, "Label selector:  %s\n", strings.Join(selectorParts, ","))
		} else {
			fmt.Fprintf(out, "Label selector:  <none>\n")
		}

		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "Or label selector:  <none>\n")
		fmt.Fprintf(out, "\n")

		// Storage Location
		if spec.StorageLocation != "" {
			fmt.Fprintf(out, "Storage Location:  %s\n", spec.StorageLocation)
		} else {
			fmt.Fprintf(out, "Storage Location:  <none>\n")
		}

		fmt.Fprintf(out, "\n")

		// Snapshot settings
		if spec.SnapshotVolumes != nil {
			if *spec.SnapshotVolumes {
				fmt.Fprintf(out, "Velero-Native Snapshot PVs:  true\n")
			} else {
				fmt.Fprintf(out, "Velero-Native Snapshot PVs:  false\n")
			}
		} else {
			fmt.Fprintf(out, "Velero-Native Snapshot PVs:  auto\n")
		}

		if spec.SnapshotMoveData != nil && *spec.SnapshotMoveData {
			fmt.Fprintf(out, "Snapshot Move Data:          true\n")
		} else {
			fmt.Fprintf(out, "Snapshot Move Data:          false\n")
		}

		if spec.DataMover != "" {
			fmt.Fprintf(out, "Data Mover:                  %s\n", spec.DataMover)
		} else {
			fmt.Fprintf(out, "Data Mover:                  velero\n")
		}

		fmt.Fprintf(out, "\n")

		// TTL
		if spec.TTL.Duration > 0 {
			fmt.Fprintf(out, "TTL:  %s\n", spec.TTL.Duration)
		} else {
			fmt.Fprintf(out, "TTL:  720h0m0s\n") // default
		}

		fmt.Fprintf(out, "\n")

		// Timeouts
		if spec.CSISnapshotTimeout.Duration > 0 {
			fmt.Fprintf(out, "CSISnapshotTimeout:    %s\n", spec.CSISnapshotTimeout.Duration)
		} else {
			fmt.Fprintf(out, "CSISnapshotTimeout:    10m0s\n")
		}

		if spec.ItemOperationTimeout.Duration > 0 {
			fmt.Fprintf(out, "ItemOperationTimeout:  %s\n", spec.ItemOperationTimeout.Duration)
		} else {
			fmt.Fprintf(out, "ItemOperationTimeout:  4h0m0s\n")
		}

		fmt.Fprintf(out, "\n")

		// Hooks
		if len(spec.Hooks.Resources) > 0 {
			fmt.Fprintf(out, "Hooks:  %d resources with hooks\n", len(spec.Hooks.Resources))
		} else {
			fmt.Fprintf(out, "Hooks:  <none>\n")
		}

		fmt.Fprintf(out, "\n")
	}

	// Velero backup status information
	if vb != nil && vb.Status != nil {
		status := vb.Status

		// Backup Format Version
		if status.FormatVersion != "" {
			fmt.Fprintf(out, "Backup Format Version:  %s\n", status.FormatVersion)
		}

		fmt.Fprintf(out, "\n")

		// Started and Completed times
		if !status.StartTimestamp.IsZero() {
			fmt.Fprintf(out, "Started:    %s\n", status.StartTimestamp.Time.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if !status.CompletionTimestamp.IsZero() {
			fmt.Fprintf(out, "Completed:  %s\n", status.CompletionTimestamp.Time.Format("2006-01-02 15:04:05 -0700 MST"))
		}

		fmt.Fprintf(out, "\n")

		// Expiration
		if status.Expiration != nil {
			fmt.Fprintf(out, "Expiration:  %s\n", status.Expiration.Format("2006-01-02 15:04:05 -0700 MST"))
		}

		fmt.Fprintf(out, "\n")

		// Progress
		if status.Progress != nil {
			fmt.Fprintf(out, "Total items to be backed up:  %d\n", status.Progress.TotalItems)
			fmt.Fprintf(out, "Items backed up:              %d\n", status.Progress.ItemsBackedUp)
		}

		fmt.Fprintf(out, "\n")

		// Fetch and display Resource List
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resourceList, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
			BackupName:  backupName,
			DataType:    "BackupResourceList",
			Namespace:   userNamespace,
			HTTPTimeout: timeout,
		})

		if err == nil && resourceList != "" {
			if formattedList := formatResourceList(resourceList); formattedList != "" {
				fmt.Fprintf(out, "Resource List:\n")
				fmt.Fprintf(out, "%s\n", formattedList)
				fmt.Fprintf(out, "\n")
			}
		}

		// Backup Volumes
		fmt.Fprintf(out, "Backup Volumes:\n")

		hasVeleroSnapshots := status.VolumeSnapshotsAttempted > 0
		if hasVeleroSnapshots {
			fmt.Fprintf(out, "  Velero-Native Snapshots:  %d of %d snapshots completed successfully (specify --details for more information)\n",
				status.VolumeSnapshotsCompleted, status.VolumeSnapshotsAttempted)
		} else {
			fmt.Fprintf(out, "  Velero-Native Snapshots: <none included>\n")
		}

		fmt.Fprintf(out, "\n")

		hasCSISnapshots := status.CSIVolumeSnapshotsAttempted > 0
		if hasCSISnapshots {
			fmt.Fprintf(out, "  CSI Snapshots:  %d of %d snapshots completed successfully\n",
				status.CSIVolumeSnapshotsCompleted, status.CSIVolumeSnapshotsAttempted)
		} else {
			fmt.Fprintf(out, "  CSI Snapshots: <none included>\n")
		}

		fmt.Fprintf(out, "\n")

		// Pod Volume Backups
		fmt.Fprintf(out, "  Pod Volume Backups: <none included>\n")

		fmt.Fprintf(out, "\n")

		// Hooks
		fmt.Fprintf(out, "HooksAttempted:  %d\n", status.HookStatus.HooksAttempted)
		fmt.Fprintf(out, "HooksFailed:     %d\n", status.HookStatus.HooksFailed)
	} else {
		// Velero backup not available yet
		fmt.Fprintf(out, "Velero backup information not yet available.\n")
		fmt.Fprintf(out, "Request Phase: %s\n", nab.Status.Phase)
	}
}

// printDetailedBackupInfo fetches and displays additional backup details when --details flag is used.
// It uses NonAdminDownloadRequest to fetch:
// - BackupVolumeInfos (snapshot details)
// - BackupResults (errors, warnings)
// - BackupItemOperations (plugin operations)
func printDetailedBackupInfo(cmd *cobra.Command, kbClient kbclient.Client, backupName string, userNamespace string, timeout time.Duration) error {
	out := cmd.OutOrStdout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	hasOutput := false

	// 1. Fetch BackupVolumeInfos
	volumeInfo, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  backupName,
		DataType:    "BackupVolumeInfos",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && volumeInfo != "" {
		if formattedInfo := formatVolumeInfo(volumeInfo); formattedInfo != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
				hasOutput = true
			}
			fmt.Fprintf(out, "Volume Snapshot Details:\n")
			fmt.Fprintf(out, "%s\n", formattedInfo)
			fmt.Fprintf(out, "\n")
		}
	}

	// 2. Fetch BackupResults
	results, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  backupName,
		DataType:    "BackupResults",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && results != "" {
		if formattedResults := formatBackupResults(results); formattedResults != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
				hasOutput = true
			}
			fmt.Fprintf(out, "Backup Results:\n")
			fmt.Fprintf(out, "%s\n", formattedResults)
			fmt.Fprintf(out, "\n")
		}
	}

	// 3. Fetch BackupItemOperations
	itemOps, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  backupName,
		DataType:    "BackupItemOperations",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && itemOps != "" {
		if formattedOps := formatItemOperations(itemOps); formattedOps != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
			}
			fmt.Fprintf(out, "Backup Item Operations:\n")
			fmt.Fprintf(out, "%s\n", formattedOps)
			fmt.Fprintf(out, "\n")
		}
	}

	return nil
}

// formatVolumeInfo formats volume snapshot information for display
func formatVolumeInfo(volumeInfo string) string {
	if strings.TrimSpace(volumeInfo) == "" {
		return ""
	}

	// Try to parse as JSON array
	var snapshots []interface{}
	if err := json.Unmarshal([]byte(volumeInfo), &snapshots); err != nil {
		// If parsing fails, fall back to indented output
		return indent(volumeInfo, "  ")
	}

	// If empty array, return empty string (will show "<none>")
	if len(snapshots) == 0 {
		return ""
	}

	// Format as indented JSON for readability
	formatted, err := json.MarshalIndent(snapshots, "  ", "  ")
	if err != nil {
		return indent(volumeInfo, "  ")
	}
	return indent(string(formatted), "  ")
}

// formatResourceList formats the resource list for display
func formatResourceList(resourceList string) string {
	if strings.TrimSpace(resourceList) == "" {
		return ""
	}

	// Try to parse as JSON map
	var resources map[string][]string
	if err := json.Unmarshal([]byte(resourceList), &resources); err != nil {
		// If parsing fails, fall back to indented output
		return indent(resourceList, "  ")
	}

	// Sort the keys (GroupVersionKind)
	keys := make([]string, 0, len(resources))
	for k := range resources {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build formatted output
	var output strings.Builder
	for _, gvk := range keys {
		items := resources[gvk]
		output.WriteString(fmt.Sprintf("  %s:\n", gvk))
		for _, item := range items {
			output.WriteString(fmt.Sprintf("    - %s\n", item))
		}
	}

	return strings.TrimSuffix(output.String(), "\n")
}

// formatBackupResults formats backup results (errors/warnings) for display
func formatBackupResults(results string) string {
	if strings.TrimSpace(results) == "" {
		return ""
	}

	// Try to parse as JSON object with errors and warnings
	var resultsObj struct {
		Errors   map[string]interface{} `json:"errors"`
		Warnings map[string]interface{} `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(results), &resultsObj); err != nil {
		// If parsing fails, fall back to indented output
		return indent(results, "  ")
	}

	// If both are empty, return empty string so section won't be printed
	if len(resultsObj.Errors) == 0 && len(resultsObj.Warnings) == 0 {
		return ""
	}

	// Format nicely
	var output strings.Builder

	// Show errors
	output.WriteString("  Errors:\n")
	if len(resultsObj.Errors) > 0 {
		formatted, _ := json.MarshalIndent(resultsObj.Errors, "    ", "  ")
		output.WriteString(indent(string(formatted), "    "))
	} else {
		output.WriteString("    <none>")
	}
	output.WriteString("\n\n")

	// Show warnings
	output.WriteString("  Warnings:\n")
	if len(resultsObj.Warnings) > 0 {
		formatted, _ := json.MarshalIndent(resultsObj.Warnings, "    ", "  ")
		output.WriteString(indent(string(formatted), "    "))
	} else {
		output.WriteString("    <none>")
	}

	return strings.TrimSuffix(output.String(), "\n")
}

// formatItemOperations formats backup item operations for display
func formatItemOperations(itemOps string) string {
	if strings.TrimSpace(itemOps) == "" {
		return ""
	}

	// Try to parse as JSON array
	var operations []interface{}
	if err := json.Unmarshal([]byte(itemOps), &operations); err != nil {
		// If parsing fails, fall back to indented output
		return indent(itemOps, "  ")
	}

	// If empty array, return empty string (will show "<none>")
	if len(operations) == 0 {
		return ""
	}

	// Format as indented JSON for readability
	formatted, err := json.MarshalIndent(operations, "  ", "  ")
	if err != nil {
		return indent(itemOps, "  ")
	}
	return indent(string(formatted), "  ")
}

// colorizePhase returns the phase string with ANSI color codes
func colorizePhase(phase string) string {
	const (
		colorGreen  = "\033[32m"
		colorYellow = "\033[33m"
		colorRed    = "\033[31m"
		colorReset  = "\033[0m"
	)

	switch phase {
	case "Completed":
		return colorGreen + phase + colorReset
	case "InProgress", "New":
		return colorYellow + phase + colorReset
	case "Failed", "FailedValidation", "PartiallyFailed":
		return colorRed + phase + colorReset
	default:
		return phase
	}
}

// NonAdminDescribeBackup mirrors Velero's output.DescribeBackup functionality
// but works within non-admin RBAC boundaries using NonAdminDownloadRequest.
// The timeout parameter controls how long to wait for download requests to complete.
// If timeout is 0, DefaultOperationTimeout is used.
func NonAdminDescribeBackup(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, userNamespace string, timeout time.Duration) error {
	// Use provided timeout or fall back to default
	if timeout == 0 {
		timeout = shared.DefaultOperationTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Print basic backup information
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", nab.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", nab.Namespace)

	// Print labels
	fmt.Fprintf(cmd.OutOrStdout(), "Labels:\n")
	if len(nab.Labels) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  <none>\n")
	} else {
		labelKeys := make([]string, 0, len(nab.Labels))
		for k := range nab.Labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)
		for _, k := range labelKeys {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s=%s\n", k, nab.Labels[k])
		}
	}

	// Print annotations
	fmt.Fprintf(cmd.OutOrStdout(), "Annotations:\n")
	if len(nab.Annotations) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  <none>\n")
	} else {
		annotationKeys := make([]string, 0, len(nab.Annotations))
		for k := range nab.Annotations {
			annotationKeys = append(annotationKeys, k)
		}
		sort.Strings(annotationKeys)
		for _, k := range annotationKeys {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s=%s\n", k, nab.Annotations[k])
		}
	}

	// Print timestamps and status from NonAdminBackup
	fmt.Fprintf(cmd.OutOrStdout(), "Creation Timestamp:  %s\n", nab.CreationTimestamp.Format(time.RFC3339))
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:               %s\n", nab.Status.Phase)

	// If there's a referenced Velero backup, get more details
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Name != "" {
		veleroBackupName := nab.Status.VeleroBackup.Name

		// Try to get additional backup details, but don't block if they're not available
		fmt.Fprintf(cmd.OutOrStdout(), "\nFetching additional backup details...")

		// Get backup results using NonAdminDownloadRequest (most important data)
		if results, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
			BackupName:  veleroBackupName,
			DataType:    "BackupResults",
			Namespace:   userNamespace,
			HTTPTimeout: timeout,
		}); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Results:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(results, "  "))
		}

		// Get backup details using NonAdminDownloadRequest for BackupResourceList
		if resourceList, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
			BackupName:  veleroBackupName,
			DataType:    "BackupResourceList",
			Namespace:   userNamespace,
			HTTPTimeout: timeout,
		}); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Resource List:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(resourceList, "  "))
		}

		// Get backup volume info using NonAdminDownloadRequest
		if volumeInfo, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
			BackupName:  veleroBackupName,
			DataType:    "BackupVolumeInfos",
			Namespace:   userNamespace,
			HTTPTimeout: timeout,
		}); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Volume Info:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(volumeInfo, "  "))
		}

		// Get backup item operations using NonAdminDownloadRequest
		if itemOps, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
			BackupName:  veleroBackupName,
			DataType:    "BackupItemOperations",
			Namespace:   userNamespace,
			HTTPTimeout: timeout,
		}); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Item Operations:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(itemOps, "  "))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\nDone fetching additional details.")
	}

	// Print NonAdminBackup Spec (excluding sensitive information)
	if nab.Spec.BackupSpec != nil {
		specYaml, err := yaml.Marshal(nab.Spec.BackupSpec)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nSpec: <error marshaling spec: %v>\n", err)
		} else {
			filteredSpec := filterIncludedNamespaces(string(specYaml))
			fmt.Fprintf(cmd.OutOrStdout(), "\nSpec:\n%s", indent(filteredSpec, "  "))
		}
	}

	// Print NonAdminBackup Status (excluding sensitive information)
	statusYaml, err := yaml.Marshal(nab.Status)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "\nStatus: <error marshaling status: %v>\n", err)
	} else {
		// Filter out includednamespaces from status output as well
		filteredStatus := filterIncludedNamespaces(string(statusYaml))
		fmt.Fprintf(cmd.OutOrStdout(), "\nStatus:\n%s", indent(filteredStatus, "  "))
	}

	// Print Events for NonAdminBackup
	fmt.Fprintf(cmd.OutOrStdout(), "\nEvents:\n")
	var eventList corev1.EventList
	if err := kbClient.List(ctx, &eventList, kbclient.InNamespace(userNamespace)); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "  <error fetching events: %v>\n", err)
	} else {
		// Filter events related to this NonAdminBackup
		var relatedEvents []corev1.Event
		for _, event := range eventList.Items {
			if event.InvolvedObject.Kind == "NonAdminBackup" && event.InvolvedObject.Name == nab.Name {
				relatedEvents = append(relatedEvents, event)
			}
		}

		if len(relatedEvents) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  <none>\n")
		} else {
			for _, e := range relatedEvents {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", e.Reason, e.Message)
			}
		}
	}

	return nil
}

// Helper to filter out includednamespaces from YAML output
func filterIncludedNamespaces(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	var filtered []string
	skip := false
	var skipIndentLevel int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Calculate indentation level
		indentLevel := len(line) - len(strings.TrimLeft(line, " \t"))

		// Check if this line starts the includednamespaces field
		if !skip && (trimmed == "includednamespaces:" || trimmed == "includedNamespaces:" ||
			strings.HasPrefix(trimmed, "includednamespaces: ") || strings.HasPrefix(trimmed, "includedNamespaces: ")) {
			skip = true
			skipIndentLevel = indentLevel
			continue
		}

		if skip {
			// Stop skipping if we found a line at the same or lesser indentation level
			// and it's not an empty line and it's not a list item belonging to the skipped field
			if trimmed != "" && indentLevel <= skipIndentLevel && !strings.HasPrefix(trimmed, "- ") {
				skip = false
				// Process this line since we're no longer skipping
				filtered = append(filtered, line)
			}
			// If we're still skipping, don't add the line
			continue
		}

		// Add the line if we're not skipping
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

// Helper to indent YAML blocks
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
