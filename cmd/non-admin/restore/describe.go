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
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
)

// DescribeOptions holds options for the describe command
type DescribeOptions struct {
	Details bool
}

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	options := &DescribeOptions{}

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin restore",
		Long:  "Display detailed information about a specified non-admin restore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			restoreName := args[0]

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

			// Find the NonAdminRestore
			var narList nacv1alpha1.NonAdminRestoreList
			if err := kbClient.List(context.Background(), &narList, &kbclient.ListOptions{
				Namespace: userNamespace,
			}); err != nil {
				return fmt.Errorf("failed to list NonAdminRestore: %w", err)
			}

			var foundNAR *nacv1alpha1.NonAdminRestore
			for i := range narList.Items {
				if narList.Items[i].Name == restoreName {
					foundNAR = &narList.Items[i]
					break
				}
			}

			if foundNAR == nil {
				return fmt.Errorf("NonAdminRestore %q not found in namespace %q", restoreName, userNamespace)
			}

			return NonAdminDescribeRestore(cmd, kbClient, foundNAR, userNamespace, options)
		},
		Example: `  # Describe a non-admin restore (concise summary)
  kubectl oadp nonadmin restore describe my-restore
  
  # Describe with complete detailed output (same as Velero)
  kubectl oadp nonadmin restore describe my-restore --details`,
	}

	// Add the --details flag
	c.Flags().BoolVar(&options.Details, "details", false, "Show complete detailed output (same as Velero restore describe --details)")

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// NonAdminDescribeRestore provides a Velero-style detailed output format
// but works within non-admin RBAC boundaries using NonAdminDownloadRequest
func NonAdminDescribeRestore(cmd *cobra.Command, kbClient kbclient.Client, nar *nacv1alpha1.NonAdminRestore, userNamespace string, options *DescribeOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if options.Details {
		// Show the full Velero-style detailed output
		return NonAdminDescribeRestoreDetailed(cmd, kbClient, nar, userNamespace, ctx)
	} else {
		// Show a concise summary
		return NonAdminDescribeRestoreSummary(cmd, kbClient, nar, userNamespace, ctx)
	}
}

// NonAdminDescribeRestoreSummary provides a concise restore summary
func NonAdminDescribeRestoreSummary(cmd *cobra.Command, kbClient kbclient.Client, nar *nacv1alpha1.NonAdminRestore, userNamespace string, ctx context.Context) error {
	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", nar.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", nar.Namespace)
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:        %s\n", nar.Status.Phase)

	// Basic timing if available
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		vStatus := nar.Status.VeleroRestore.Status
		if vStatus.StartTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Started:      %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.CompletionTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Completed:    %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.Progress != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Items restored: %d\n", vStatus.Progress.ItemsRestored)
		}
		if vStatus.Errors > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Errors:       %d\n", vStatus.Errors)
		}
		if vStatus.Warnings > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Warnings:     %d\n", vStatus.Warnings)
		}
	}

	// Show backup source
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.BackupName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Backup:       %s\n", nar.Spec.RestoreSpec.BackupName)
	}

	return nil
}

// NonAdminDescribeRestoreDetailed provides a Velero-style detailed output format
func NonAdminDescribeRestoreDetailed(cmd *cobra.Command, kbClient kbclient.Client, nar *nacv1alpha1.NonAdminRestore, userNamespace string, ctx context.Context) error {
	// Header in Velero style
	fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", nar.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", nar.Namespace)

	// Labels (Velero-style format)
	if len(nar.Labels) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Labels:       ")
		labelPairs := make([]string, 0, len(nar.Labels))
		for k, v := range nar.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(labelPairs)
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(labelPairs, ","))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Labels:       <none>\n")
	}

	// Annotations (Velero-style format)
	if len(nar.Annotations) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Annotations:  ")
		annotationPairs := make([]string, 0, len(nar.Annotations))
		for k, v := range nar.Annotations {
			annotationPairs = append(annotationPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(annotationPairs)
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", strings.Join(annotationPairs, ","))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Annotations:  <none>\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Phase/Status information
	fmt.Fprintf(cmd.OutOrStdout(), "Phase:  %s\n", nar.Status.Phase)

	// Add error/warning information if available
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		vStatus := nar.Status.VeleroRestore.Status
		if vStatus.Errors > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Errors:    %d\n", vStatus.Errors)
		}
		if vStatus.Warnings > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Warnings:  %d\n", vStatus.Warnings)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Note: backupName is not admin enforceable and therefore not displayed in non-admin restore describe output

	// Note: Namespace mappings and namespace exclusions are restricted for non-admin users
	// and therefore not displayed in non-admin restore describe output

	// Resources section (Velero-style)
	fmt.Fprintf(cmd.OutOrStdout(), "Resources:\n")
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.IncludedResources) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Included:        %s\n", strings.Join(nar.Spec.RestoreSpec.IncludedResources, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Included:        *\n")
	}
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.ExcludedResources) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:        %s\n", strings.Join(nar.Spec.RestoreSpec.ExcludedResources, ", "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Excluded:        <none>\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Cluster-scoped:  auto\n")

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Label selector (Velero-style)
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.LabelSelector != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Label selector:  %v\n", nar.Spec.RestoreSpec.LabelSelector)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Label selector:  <none>\n")
	}

	// Or label selector (Velero-style)
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.OrLabelSelectors != nil && len(nar.Spec.RestoreSpec.OrLabelSelectors) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Or label selector:  %v\n", nar.Spec.RestoreSpec.OrLabelSelectors)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Or label selector:  <none>\n")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	// Restore settings
	if nar.Spec.RestoreSpec != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Restore PVs:               %v\n", getBoolPointerValue(nar.Spec.RestoreSpec.RestorePVs))
		fmt.Fprintf(cmd.OutOrStdout(), "Preserve Node Ports:       %v\n", getBoolPointerValue(nar.Spec.RestoreSpec.PreserveNodePorts))
		fmt.Fprintf(cmd.OutOrStdout(), "Include Cluster Resources: %v\n", getBoolPointerValue(nar.Spec.RestoreSpec.IncludeClusterResources))

		if nar.Spec.RestoreSpec.ExistingResourcePolicy != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Existing Resource Policy:  %s\n", nar.Spec.RestoreSpec.ExistingResourcePolicy)
		}

		if nar.Spec.RestoreSpec.ItemOperationTimeout.Duration > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Item Operation Timeout:    %s\n", nar.Spec.RestoreSpec.ItemOperationTimeout.Duration.String())
		}

		if nar.Spec.RestoreSpec.UploaderConfig != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Uploader Config:           configured\n")
		}

		if nar.Spec.RestoreSpec.ResourceModifier != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Resource Modifiers:        configured\n")
		}

		if nar.Spec.RestoreSpec.RestoreStatus != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Restore Status:            configured\n")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Hooks
		if len(nar.Spec.RestoreSpec.Hooks.Resources) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Hooks:  %d hook(s) configured\n", len(nar.Spec.RestoreSpec.Hooks.Resources))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Hooks:  <none>\n")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")
	}

	// Restore timing and progress
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		vStatus := nar.Status.VeleroRestore.Status

		if vStatus.StartTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Started:    %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if vStatus.CompletionTimestamp != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Completed:  %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST"))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Progress information
		if vStatus.Progress != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Total items to be restored:  %d\n", vStatus.Progress.TotalItems)
			fmt.Fprintf(cmd.OutOrStdout(), "Items restored:              %d\n", vStatus.Progress.ItemsRestored)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Resource List section (like regular Velero restore describe)
		if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Name != "" {
			narName := nar.Name

			fmt.Fprintf(cmd.OutOrStdout(), "Resource List:\n")

			// Try to get resource information using RestoreResourceList download target
			resourceListFound := false

			// Try RestoreResourceList which contains the structured resource data
			if resourceData, err := downloadRestoreData(ctx, kbClient, userNamespace, narName, "RestoreResourceList"); err == nil && resourceData != "" {
				// Decompress the data since it's gzipped
				if decompressed, err := decompressData(resourceData); err == nil {
					if formattedList := formatRestoreResourceList(decompressed); formattedList != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "%s", formattedList)
						resourceListFound = true
					}
				}
			}

			// If RestoreResourceList didn't work, show basic info
			if !resourceListFound {
				fmt.Fprintf(cmd.OutOrStdout(), "  <detailed resource breakdown not available via RestoreResourceList>\n")
				if vStatus != nil && vStatus.Progress != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  Total items restored: %d\n", vStatus.Progress.ItemsRestored)
				}
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Volume information
		fmt.Fprintf(cmd.OutOrStdout(), "Restore Volumes:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Persistent Volumes: <none restored>\n")

		fmt.Fprintf(cmd.OutOrStdout(), "\n")

		// Hooks information (at the very end)
		fmt.Fprintf(cmd.OutOrStdout(), "HooksAttempted:  %d\n", vStatus.HookStatus.HooksAttempted)
		fmt.Fprintf(cmd.OutOrStdout(), "HooksFailed:     %d\n", vStatus.HookStatus.HooksFailed)
	}

	return nil
}

// Helper functions for the detailed output
func getBoolPointerValue(b *bool) string {
	if b == nil {
		return "auto"
	}
	if *b {
		return "true"
	}
	return "false"
}

// downloadRestoreData uses NonAdminDownloadRequest to fetch detailed restore information
// This replaces direct access to Velero restores with RBAC-compliant requests
func downloadRestoreData(ctx context.Context, kbClient kbclient.Client, userNamespace, restoreName, dataType string) (string, error) {
	// Create NonAdminDownloadRequest for the specified data type
	req := &nacv1alpha1.NonAdminDownloadRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: restoreName + "-" + strings.ToLower(dataType) + "-",
			Namespace:    userNamespace,
		},
		Spec: nacv1alpha1.NonAdminDownloadRequestSpec{
			Target: velerov1.DownloadTarget{
				Kind: velerov1.DownloadTargetKind(dataType),
				Name: restoreName,
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

// formatRestoreResourceList formats the raw RestoreResourceList data into a readable string
func formatRestoreResourceList(data string) string {
	if data == "" {
		return "  <none>\n"
	}

	// Parse the JSON data (the RestoreResourceList is JSON format)
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
