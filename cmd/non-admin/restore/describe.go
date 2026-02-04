package restore

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
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	var (
		requestTimeout time.Duration
		details        bool
	)

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin restore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			restoreName := args[0]

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

			// Get the specific restore
			var nar nacv1alpha1.NonAdminRestore
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: userNamespace,
				Name:      restoreName,
			}, &nar); err != nil {
				// Check for context cancellation
				if ctx.Err() == context.DeadlineExceeded {
					return fmt.Errorf("timed out after %v getting NonAdminRestore %q", effectiveTimeout, restoreName)
				}
				if ctx.Err() == context.Canceled {
					return fmt.Errorf("operation cancelled: %w", ctx.Err())
				}
				return fmt.Errorf("NonAdminRestore %q not found in namespace %q: %w", restoreName, userNamespace, err)
			}

			// Print in Velero-style format
			printNonAdminRestoreDetails(cmd, &nar, kbClient, restoreName, userNamespace, effectiveTimeout)

			// Add detailed output if --details flag is set
			if details {
				if err := printDetailedRestoreInfo(cmd, kbClient, restoreName, userNamespace, effectiveTimeout); err != nil {
					return fmt.Errorf("failed to fetch detailed restore information: %w", err)
				}
			}

			return nil
		},
		Example: `  kubectl oadp nonadmin restore describe my-restore
  kubectl oadp nonadmin restore describe my-restore --details
  kubectl oadp nonadmin restore describe my-restore --details --request-timeout=30m`,
	}

	c.Flags().DurationVar(&requestTimeout, "request-timeout", 0, fmt.Sprintf("The length of time to wait before giving up on a single server request (e.g., 30s, 5m, 1h). Overrides %s env var. Default: %v", shared.TimeoutEnvVar, shared.DefaultHTTPTimeout))
	c.Flags().BoolVar(&details, "details", false, "Display additional restore details including resource lists and item operations")

	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// printNonAdminRestoreDetails prints restore details in Velero admin describe format
func printNonAdminRestoreDetails(cmd *cobra.Command, nar *nacv1alpha1.NonAdminRestore, kbClient kbclient.Client, restoreName string, userNamespace string, timeout time.Duration) {
	out := cmd.OutOrStdout()

	// Get Velero restore reference if available
	var vr *nacv1alpha1.VeleroRestore
	if nar.Status.VeleroRestore != nil {
		vr = nar.Status.VeleroRestore
	}

	// Name and Namespace
	fmt.Fprintf(out, "Name:         %s\n", nar.Name)
	fmt.Fprintf(out, "Namespace:    %s\n", nar.Namespace)

	// Labels
	fmt.Fprintf(out, "Labels:       ")
	if len(nar.Labels) == 0 {
		fmt.Fprintf(out, "<none>\n")
	} else {
		labelKeys := make([]string, 0, len(nar.Labels))
		for k := range nar.Labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)
		for i, k := range labelKeys {
			if i == 0 {
				fmt.Fprintf(out, "%s=%s\n", k, nar.Labels[k])
			} else {
				fmt.Fprintf(out, "              %s=%s\n", k, nar.Labels[k])
			}
		}
	}

	// Annotations
	fmt.Fprintf(out, "Annotations:  ")
	if len(nar.Annotations) == 0 {
		fmt.Fprintf(out, "<none>\n")
	} else {
		annotationKeys := make([]string, 0, len(nar.Annotations))
		for k := range nar.Annotations {
			annotationKeys = append(annotationKeys, k)
		}
		sort.Strings(annotationKeys)
		for i, k := range annotationKeys {
			if i == 0 {
				fmt.Fprintf(out, "%s=%s\n", k, nar.Annotations[k])
			} else {
				fmt.Fprintf(out, "              %s=%s\n", k, nar.Annotations[k])
			}
		}
	}

	fmt.Fprintf(out, "\n")

	// Phase (with color)
	phase := string(nar.Status.Phase)
	if vr != nil && vr.Status != nil && vr.Status.Phase != "" {
		phase = string(vr.Status.Phase)
	}
	fmt.Fprintf(out, "Phase:  %s\n", colorizePhase(phase))

	fmt.Fprintf(out, "\n")

	// Restore Spec details
	if nar.Spec.RestoreSpec != nil {
		spec := nar.Spec.RestoreSpec

		// Source Backup
		if spec.BackupName != "" {
			fmt.Fprintf(out, "Backup:  %s\n", spec.BackupName)
		} else {
			fmt.Fprintf(out, "Backup:  <none>\n")
		}

		fmt.Fprintf(out, "\n")

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

		// Namespace Mappings
		if len(spec.NamespaceMapping) == 0 {
			fmt.Fprintf(out, "Namespace mappings:  <none>\n")
		} else {
			fmt.Fprintf(out, "Namespace mappings:\n")
			// Sort the mappings for consistent output
			mappingKeys := make([]string, 0, len(spec.NamespaceMapping))
			for k := range spec.NamespaceMapping {
				mappingKeys = append(mappingKeys, k)
			}
			sort.Strings(mappingKeys)
			for _, from := range mappingKeys {
				fmt.Fprintf(out, "  %s: %s\n", from, spec.NamespaceMapping[from])
			}
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

		// Restore PVs setting
		if spec.RestorePVs != nil {
			if *spec.RestorePVs {
				fmt.Fprintf(out, "Restore PVs:  true\n")
			} else {
				fmt.Fprintf(out, "Restore PVs:  false\n")
			}
		} else {
			fmt.Fprintf(out, "Restore PVs:  auto\n")
		}

		fmt.Fprintf(out, "\n")

		// Existing Resource Policy
		if spec.ExistingResourcePolicy != "" {
			fmt.Fprintf(out, "Existing Resource Policy:  %s\n", spec.ExistingResourcePolicy)
		} else {
			fmt.Fprintf(out, "Existing Resource Policy:  <none>\n")
		}

		fmt.Fprintf(out, "\n")

		// Item Operation Timeout
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

	// Velero restore status information
	if vr != nil && vr.Status != nil {
		status := vr.Status

		// Started and Completed times
		if !status.StartTimestamp.IsZero() {
			fmt.Fprintf(out, "Started:    %s\n", status.StartTimestamp.Time.Format("2006-01-02 15:04:05 -0700 MST"))
		}
		if !status.CompletionTimestamp.IsZero() {
			fmt.Fprintf(out, "Completed:  %s\n", status.CompletionTimestamp.Time.Format("2006-01-02 15:04:05 -0700 MST"))
		}

		fmt.Fprintf(out, "\n")

		// Progress
		if status.Progress != nil {
			fmt.Fprintf(out, "Total items to be restored:  %d\n", status.Progress.TotalItems)
			fmt.Fprintf(out, "Items restored:              %d\n", status.Progress.ItemsRestored)
		}

		fmt.Fprintf(out, "\n")

		// Warnings and Errors
		if status.Warnings > 0 {
			fmt.Fprintf(out, "Warnings:  %d\n", status.Warnings)
		}
		if status.Errors > 0 {
			fmt.Fprintf(out, "Errors:    %d\n", status.Errors)
		}

		fmt.Fprintf(out, "\n")

		// Hooks
		fmt.Fprintf(out, "HooksAttempted:  %d\n", status.HookStatus.HooksAttempted)
		fmt.Fprintf(out, "HooksFailed:     %d\n", status.HookStatus.HooksFailed)
	} else {
		// Velero restore not available yet
		fmt.Fprintf(out, "Velero restore information not yet available.\n")
		fmt.Fprintf(out, "Request Phase: %s\n", nar.Status.Phase)
	}
}

// printDetailedRestoreInfo fetches and displays additional restore details when --details flag is used.
// It uses NonAdminDownloadRequest to fetch:
// - RestoreResourceList (list of restored resources)
// - RestoreResults (errors, warnings)
// - RestoreItemOperations (plugin operations)
func printDetailedRestoreInfo(cmd *cobra.Command, kbClient kbclient.Client, restoreName string, userNamespace string, timeout time.Duration) error {
	out := cmd.OutOrStdout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	hasOutput := false

	// 1. Fetch RestoreResourceList
	resourceList, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  restoreName,
		DataType:    "RestoreResourceList",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && resourceList != "" {
		if formattedList := formatRestoreResourceList(resourceList); formattedList != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
				hasOutput = true
			}
			fmt.Fprintf(out, "Resource List:\n")
			fmt.Fprintf(out, "%s\n", formattedList)
			fmt.Fprintf(out, "\n")
		}
	}

	// 2. Fetch RestoreResults
	results, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  restoreName,
		DataType:    "RestoreResults",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && results != "" {
		if formattedResults := formatRestoreResults(results); formattedResults != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
				hasOutput = true
			}
			fmt.Fprintf(out, "Restore Results:\n")
			fmt.Fprintf(out, "%s\n", formattedResults)
			fmt.Fprintf(out, "\n")
		}
	}

	// 3. Fetch RestoreItemOperations
	itemOps, err := shared.ProcessDownloadRequest(ctx, kbClient, shared.DownloadRequestOptions{
		BackupName:  restoreName,
		DataType:    "RestoreItemOperations",
		Namespace:   userNamespace,
		HTTPTimeout: timeout,
	})

	if err == nil && itemOps != "" {
		if formattedOps := formatRestoreItemOperations(itemOps); formattedOps != "" {
			if !hasOutput {
				fmt.Fprintf(out, "\n")
			}
			fmt.Fprintf(out, "Restore Item Operations:\n")
			fmt.Fprintf(out, "%s\n", formattedOps)
			fmt.Fprintf(out, "\n")
		}
	}

	return nil
}

// formatRestoreResourceList formats the resource list for display
func formatRestoreResourceList(resourceList string) string {
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

// formatRestoreResults formats restore results (errors/warnings) for display
func formatRestoreResults(results string) string {
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

// formatRestoreItemOperations formats restore item operations for display
func formatRestoreItemOperations(itemOps string) string {
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
