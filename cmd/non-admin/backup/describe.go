package backup

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		// Show the full Velero-style detailed output
		return NonAdminDescribeBackupDetailed(cmd, kbClient, nab, userNamespace, ctx)
	} else {
		// Show a concise summary
		return NonAdminDescribeBackupSummary(cmd, kbClient, nab, userNamespace, ctx)
	}

	return nil
}

// NonAdminDescribeBackupSummary provides a concise backup summary
func NonAdminDescribeBackupSummary(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, userNamespace string, ctx context.Context) error {
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

// NonAdminDescribeBackupDetailed provides a Velero-style detailed output format
func NonAdminDescribeBackupDetailed(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, userNamespace string, ctx context.Context) error {
	// Header in Velero style
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", nab.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", nab.Namespace)

	// Labels (Velero-style format)
	if len(nab.Labels) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Labels:       ")
		labelPairs := make([]string, 0, len(nab.Labels))
		for k, v := range nab.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(labelPairs)
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(labelPairs, ","))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Labels:       <none>\n")
	}

	// Annotations (Velero-style format)
	if len(nab.Annotations) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Annotations:  ")
		annotationPairs := make([]string, 0, len(nab.Annotations))
		for k, v := range nab.Annotations {
			annotationPairs = append(annotationPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(annotationPairs)
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(annotationPairs, ","))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Annotations:  <none>\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Phase/Status information
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:  %s\n", nab.Status.Phase)

	// Add error/warning information if available
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status
		if vStatus.Errors > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Errors:    %d\n", vStatus.Errors)
		}
		if vStatus.Warnings > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Warnings:  %d\n", vStatus.Warnings)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Namespaces section (only show excluded since they're admin-controlled, skip included for security)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespaces:\n")
	// NOTE: We skip includedNamespaces entirely for security - non-admin users should not see which namespaces are included

	// excludedNamespaces is safe to show since it's admin-controlled (non-admin users cannot set it)
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Spec != nil && len(nab.Status.VeleroBackup.Spec.ExcludedNamespaces) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:  %s\n", strings.Join(nab.Status.VeleroBackup.Spec.ExcludedNamespaces, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:  <none>\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Resources section (Velero-style)
	fmt.Fprintf(cmd.OutOrStdout(), "Resources:\n")
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.IncludedResources) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Included:        %s\n", strings.Join(nab.Spec.BackupSpec.IncludedResources, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Included:        *\n")
	}
	if nab.Spec.BackupSpec != nil && len(nab.Spec.BackupSpec.ExcludedResources) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:        %s\n", strings.Join(nab.Spec.BackupSpec.ExcludedResources, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:        <none>\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Cluster-scoped:  auto\n")

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Label selector (Velero-style)
	if nab.Spec.BackupSpec != nil && nab.Spec.BackupSpec.LabelSelector != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Label selector:  %v\n", nab.Spec.BackupSpec.LabelSelector)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Label selector:  <none>\n")
	}

	// Or label selector (Velero-style)
	if nab.Spec.BackupSpec != nil && nab.Spec.BackupSpec.OrLabelSelectors != nil && len(nab.Spec.BackupSpec.OrLabelSelectors) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Or label selector:  %v\n", nab.Spec.BackupSpec.OrLabelSelectors)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Or label selector:  <none>\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Storage location and backup details
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Spec != nil {
		vSpec := nab.Status.VeleroBackup.Spec
		fmt.Fprintf(cmd.OutOrStdout(), "Storage Location:  %s\n", vSpec.StorageLocation)

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Snapshot settings
		fmt.Fprintf(cmd.OutOrStdout(), "Velero-Native Snapshot PVs:  %v\n", getSnapshotVolumesValue(vSpec.SnapshotVolumes))
		fmt.Fprintf(cmd.OutOrStdout(), "Snapshot Move Data:          %v\n", getBoolPointerValue(vSpec.SnapshotMoveData))
		fmt.Fprintf(cmd.OutOrStdout(), "Data Mover:                  %s\n", getStringValue(vSpec.DataMover))

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// TTL and timeouts
		if vSpec.TTL.Duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "TTL:  %s\n", vSpec.TTL.Duration.String())
		}
		if vSpec.CSISnapshotTimeout.Duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "CSISnapshotTimeout:    %s\n", vSpec.CSISnapshotTimeout.Duration.String())
		}
		if vSpec.ItemOperationTimeout.Duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "ItemOperationTimeout:  %s\n", vSpec.ItemOperationTimeout.Duration.String())
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Hooks
		if len(vSpec.Hooks.Resources) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Hooks:  %d hook(s) configured\n", len(vSpec.Hooks.Resources))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Hooks:  <none>\n")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")
	}

	// Backup format and timing
	if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Status != nil {
		vStatus := nab.Status.VeleroBackup.Status

		if vStatus.FormatVersion != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Backup Format Version:  %s\n", vStatus.FormatVersion)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		if vStatus.StartTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Started:    %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.CompletionTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Completed:  %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.Expiration != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Expiration: %s\n", vStatus.Expiration.Format("2006-01-02 15:04:05 -0700 MST"))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Progress information
		if vStatus.Progress != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Total items to be backed up:  %d\n", vStatus.Progress.TotalItems)
			fmt.Fprintf(cmd.OutOrStdout(), "Items backed up:              %d\n", vStatus.Progress.ItemsBackedUp)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Add Resource List section FIRST (like regular Velero backup describe)
		if nab.Status.VeleroBackup != nil && nab.Status.VeleroBackup.Name != "" {
			nabName := nab.Name

			fmt.Fprintf(cmd.OutOrStdout(), "Resource List:\n")

			// Try to get resource information using BackupResourceList download target
			resourceListFound := false

			// Try BackupResourceList which contains the structured resource data
			if resourceData, err := downloadBackupData(ctx, kbClient, userNamespace, nabName, "BackupResourceList"); err == nil && resourceData != "" {
				// Decompress the data since it's gzipped
				if decompressed, err := decompressData(resourceData); err == nil {
					if formattedList := formatBackupResourceList(decompressed); formattedList != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s", formattedList)
						resourceListFound = true
					}
				}
			}

			// If BackupResourceList didn't work, show basic info
			if !resourceListFound {
				fmt.Fprintf(cmd.OutOrStdout(), "  <detailed resource breakdown not available via BackupResourceList>\n")
				if vStatus != nil && vStatus.Progress != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  Total items backed up: %d\n", vStatus.Progress.ItemsBackedUp)
				}
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Volume information (after Resource List)
		fmt.Fprintf(cmd.OutOrStdout(), "Backup Volumes:\n")
		if vStatus.VolumeSnapshotsCompleted > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  Velero-Native Snapshots: <%d included>\n", vStatus.VolumeSnapshotsCompleted)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Velero-Native Snapshots: <none included>\n")
		}
		if vStatus.CSIVolumeSnapshotsCompleted > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  CSI Snapshots: <%d included>\n", vStatus.CSIVolumeSnapshotsCompleted)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  CSI Snapshots: <none included>\n")
		}
		if nab.Status.FileSystemPodVolumeBackups != nil && nab.Status.FileSystemPodVolumeBackups.Completed > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  Pod Volume Backups: <%d included>\n", nab.Status.FileSystemPodVolumeBackups.Completed)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Pod Volume Backups: <none included>\n")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Hooks information (at the very end)
		fmt.Fprintf(cmd.OutOrStdout(), "HooksAttempted:  %d\n", vStatus.HookStatus.HooksAttempted)
		fmt.Fprintf(cmd.OutOrStdout(), "HooksFailed:     %d\n", vStatus.HookStatus.HooksFailed)
	}

	return nil
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

func getStringValue(s string) string {
	if s == "" {
		return "velero"
	}
	return s
}

// downloadBackupData uses NonAdminDownloadRequest to fetch detailed backup information
// This replaces direct access to Velero backups with RBAC-compliant requests
func downloadBackupData(ctx context.Context, kbClient kbclient.Client, userNamespace, backupName, dataType string) (string, error) {
	// Create NonAdminDownloadRequest for the specified data type
	req := &nacv1alpha1.NonAdminDownloadRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupName + "-" + strings.ToLower(dataType) + "-",
			Namespace:    userNamespace,
		},
		Spec: nacv1alpha1.NonAdminDownloadRequestSpec{
			Target: velerov1.DownloadTarget{
				Kind: velerov1.DownloadTargetKind(dataType),
				Name: backupName,
			},
		},
	}

	if err := kbClient.Create(ctx, req); err != nil {
		return "", fmt.Errorf("failed to create NonAdminDownloadRequest for %s: %w", dataType, err)
	}

	// Clean up the download request when done
	defer func() {
		deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDelete()
		_ = kbClient.Delete(deleteCtx, req)
	}()

	// Wait for the download request to be processed
	timeout := time.After(10 * time.Second) // Reduced timeout since most failures are quick
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timed out waiting for %s download request to be processed", dataType)
		case <-tick:
			var updated nacv1alpha1.NonAdminDownloadRequest
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: req.Namespace,
				Name:      req.Name,
			}, &updated); err != nil {
				return "", fmt.Errorf("failed to get NonAdminDownloadRequest: %w", err)
			}

			// Check if the download request was processed successfully
			for _, condition := range updated.Status.Conditions {
				if condition.Type == "Processed" && condition.Status == "True" {
					if updated.Status.VeleroDownloadRequest.Status.DownloadURL != "" {
						// Download and return the content
						return downloadContent(updated.Status.VeleroDownloadRequest.Status.DownloadURL)
					}
				}
			}

			// Check for failure conditions
			for _, condition := range updated.Status.Conditions {
				if condition.Status == "True" && condition.Reason == "Error" {
					return "", fmt.Errorf("NonAdminDownloadRequest failed for %s: %s - %s", dataType, condition.Type, condition.Message)
				}
			}
		}
	}
}

// downloadContent fetches content from a signed URL and returns it as a string
func downloadContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download content from URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to download content: status %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Try to decompress if it's gzipped
	var reader io.Reader = resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		reader = gzr
	}

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	return string(content), nil
}

// decompressData attempts to decompress gzipped data.
func decompressData(data string) (string, error) {
	reader := strings.NewReader(data)
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader for decompression: %w", err)
	}
	defer gzr.Close()

	decompressed, err := io.ReadAll(gzr)
	if err != nil {
		return "", fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return string(decompressed), nil
}

// formatBackupResourceList formats the raw BackupResourceList data into a readable string
func formatBackupResourceList(data string) string {
	if data == "" {
		return "  <none>\n"
	}

	// Parse the JSON data (the BackupResourceList is JSON format)
	var resourceList map[string][]string
	if err := json.Unmarshal([]byte(data), &resourceList); err != nil {
		// If JSON parsing fails, just return the raw data indented
		return indent(data, "  ")
	}

	// Sort keys for consistent output
	var sortedKeys []string
	for k := range resourceList {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	var result strings.Builder
	totalResources := 0

	for _, resourceType := range sortedKeys {
		resources := resourceList[resourceType]
		if len(resources) == 0 {
			continue
		}

		totalResources += len(resources)
		result.WriteString(fmt.Sprintf("  %s (%d):\n", resourceType, len(resources)))

		// Sort resource names for consistent output
		sort.Strings(resources)

		// Show first few resources, summarize if too many
		displayLimit := 5
		for i, resource := range resources {
			if i >= displayLimit {
				remaining := len(resources) - displayLimit
				result.WriteString(fmt.Sprintf("    ... and %d more\n", remaining))
				break
			}
			result.WriteString(fmt.Sprintf("    - %s\n", resource))
		}
	}

	if totalResources > 0 {
		result.WriteString(fmt.Sprintf("\nTotal resources: %d\n", totalResources))
	}

	return result.String()
}

// Helper to filter out includednamespaces from YAML output
func filterIncludedNamespaces(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	var filteredLines []string
	skipNext := false

	for _, line := range lines {
		// Skip lines containing includednamespaces and the values that follow
		if strings.Contains(line, "includednamespaces") || strings.Contains(line, "includedNamespaces") {
			skipNext = true
			continue
		}

		// If we're skipping and this line starts with whitespace (indicating it's part of the array/list)
		if skipNext {
			trimmed := strings.TrimSpace(line)
			// If it's an array item (starts with -) or seems to be a namespace value, skip it
			if strings.HasPrefix(trimmed, "-") || (trimmed != "" && !strings.Contains(line, ":")) {
				continue
			}
			// If we hit a new field (contains :), stop skipping
			if strings.Contains(line, ":") {
				skipNext = false
			}
		}

		if !skipNext {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// indent adds the specified prefix to each line of the input string
func indent(input, prefix string) string {
	if input == "" {
		return ""
	}
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" { // Don't indent empty lines
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
