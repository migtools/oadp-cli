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
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
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
		// Show the full Velero-style detailed output using filtering approach
		return NonAdminDescribeRestoreDetailed(cmd, kbClient, nar, userNamespace, ctx)
	} else {
		// Show a concise summary
		return NonAdminDescribeRestoreSummary(cmd, kbClient, nar, userNamespace, ctx)
	}
}

// NonAdminDescribeRestoreSummary provides a concise restore summary
func NonAdminDescribeRestoreSummary(cmd *cobra.Command, kbClient kbclient.Client, nar *nacv1alpha1.NonAdminRestore, _ string, ctx context.Context) error {
	_ = ctx // Context not currently used but kept for future use

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

	// NOTE: backupName is not admin enforceable (restricted) and therefore not displayed

	return nil
}

// NonAdminDescribeRestoreDetailed leverages Velero's native restore data with filtering for non-admin users
func NonAdminDescribeRestoreDetailed(cmd *cobra.Command, kbClient kbclient.Client, nar *nacv1alpha1.NonAdminRestore, _ string, ctx context.Context) error {
	// Step 1: Get the underlying Velero Restore
	veleroRestore, err := getVeleroRestoreFromNAR(nar, kbClient, ctx)
	if err != nil {
		return fmt.Errorf("failed to get Velero restore: %w", err)
	}

	// Step 2: Create our own detailed output by filtering Velero-style information
	filteredOutput := createFilteredVeleroRestoreOutput(veleroRestore, nar)

	// Step 3: Present the refined output
	fmt.Fprint(cmd.OutOrStdout(), filteredOutput)
	return nil
}

// getVeleroRestoreFromNAR retrieves the underlying Velero Restore from NonAdminRestore
func getVeleroRestoreFromNAR(nar *nacv1alpha1.NonAdminRestore, kbClient kbclient.Client, ctx context.Context) (*velerov1.Restore, error) {
	if nar.Status.VeleroRestore == nil || nar.Status.VeleroRestore.Name == "" {
		return nil, fmt.Errorf("no Velero restore associated with NonAdminRestore %s", nar.Name)
	}

	veleroRestoreName := nar.Status.VeleroRestore.Name
	veleroNamespace := nar.Status.VeleroRestore.Namespace
	if veleroNamespace == "" {
		veleroNamespace = "openshift-adp" // Default OADP namespace
	}

	var veleroRestore velerov1.Restore
	err := kbClient.Get(ctx, kbclient.ObjectKey{
		Namespace: veleroNamespace,
		Name:      veleroRestoreName,
	}, &veleroRestore)

	if err != nil {
		return nil, fmt.Errorf("failed to get Velero restore %s/%s: %w", veleroNamespace, veleroRestoreName, err)
	}

	return &veleroRestore, nil
}

// createFilteredVeleroRestoreOutput creates a Velero-style output with non-admin field restrictions applied
func createFilteredVeleroRestoreOutput(_ *velerov1.Restore, nar *nacv1alpha1.NonAdminRestore) string {
	var output strings.Builder

	// Header in Velero style - Admin Enforceable: Yes
	output.WriteString(fmt.Sprintf("Name:         %s\n", nar.Name))
	output.WriteString(fmt.Sprintf("Namespace:    %s\n", nar.Namespace))

	// Labels (Velero-style format) - Admin Enforceable: Yes
	if len(nar.Labels) > 0 {
		output.WriteString("Labels:       ")
		labelPairs := make([]string, 0, len(nar.Labels))
		for k, v := range nar.Labels {
			labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(labelPairs)
		output.WriteString(fmt.Sprintf("%s\n", strings.Join(labelPairs, ",")))
	} else {
		output.WriteString("Labels:       <none>\n")
	}

	// Annotations (Velero-style format) - Admin Enforceable: Yes
	if len(nar.Annotations) > 0 {
		output.WriteString("Annotations:  ")
		annotationPairs := make([]string, 0, len(nar.Annotations))
		for k, v := range nar.Annotations {
			annotationPairs = append(annotationPairs, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(annotationPairs)
		output.WriteString(fmt.Sprintf("%s\n", strings.Join(annotationPairs, ",")))
	} else {
		output.WriteString("Annotations:  <none>\n")
	}

	output.WriteString("\n")

	// Phase/Status information - Admin Enforceable: Yes
	output.WriteString(fmt.Sprintf("Phase:  %s\n", nar.Status.Phase))

	// Add error/warning information if available - Admin Enforceable: Yes
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		vStatus := nar.Status.VeleroRestore.Status
		if vStatus.Errors > 0 {
			output.WriteString(fmt.Sprintf("Errors:    %d\n", vStatus.Errors))
		}
		if vStatus.Warnings > 0 {
			output.WriteString(fmt.Sprintf("Warnings:  %d\n", vStatus.Warnings))
		}
	}

	output.WriteString("\n")

	// === ADMIN ENFORCEABLE FIELDS (only show if configured) ===

	// ItemOperationTimeout - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.ItemOperationTimeout.Duration > 0 {
		output.WriteString(fmt.Sprintf("Item Operation Timeout:    %s\n", nar.Spec.RestoreSpec.ItemOperationTimeout.Duration.String()))
	}

	// UploaderConfig - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.UploaderConfig != nil {
		output.WriteString("Uploader Config:           configured\n")
	}

	// IncludedResources - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.IncludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Included Resources:        %s\n", strings.Join(nar.Spec.RestoreSpec.IncludedResources, ", ")))
	} else {
		output.WriteString("Included Resources:        * (all)\n")
	}

	// ExcludedResources - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.ExcludedResources) > 0 {
		output.WriteString(fmt.Sprintf("Excluded Resources:        %s\n", strings.Join(nar.Spec.RestoreSpec.ExcludedResources, ", ")))
	}

	// RestoreStatus - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.RestoreStatus != nil {
		output.WriteString("Restore Status:            configured\n")
	}

	// IncludeClusterResources - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.IncludeClusterResources != nil {
		output.WriteString(fmt.Sprintf("Include Cluster Resources: %v\n", *nar.Spec.RestoreSpec.IncludeClusterResources))
	} else {
		output.WriteString("Include Cluster Resources: auto\n")
	}

	// LabelSelector - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.LabelSelector != nil {
		output.WriteString(fmt.Sprintf("Label Selector:            %v\n", nar.Spec.RestoreSpec.LabelSelector))
	}

	// OrLabelSelectors - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.OrLabelSelectors) > 0 {
		output.WriteString(fmt.Sprintf("Or Label Selectors:        %v\n", nar.Spec.RestoreSpec.OrLabelSelectors))
	}

	// RestorePVs - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.RestorePVs != nil {
		output.WriteString(fmt.Sprintf("Restore PVs:               %v\n", *nar.Spec.RestoreSpec.RestorePVs))
	} else {
		output.WriteString("Restore PVs:               auto\n")
	}

	// PreserveNodePorts - Admin Enforceable: Yes
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.PreserveNodePorts != nil {
		output.WriteString(fmt.Sprintf("Preserve Node Ports:       %v\n", *nar.Spec.RestoreSpec.PreserveNodePorts))
	}

	// ExistingResourcePolicy - Admin Enforceable: (blank/unspecified)
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.ExistingResourcePolicy != "" {
		output.WriteString(fmt.Sprintf("Existing Resource Policy:  %s\n", nar.Spec.RestoreSpec.ExistingResourcePolicy))
	}

	// Hooks - Admin Enforceable: (blank/unspecified) - special case
	if nar.Spec.RestoreSpec != nil && len(nar.Spec.RestoreSpec.Hooks.Resources) > 0 {
		output.WriteString(fmt.Sprintf("Hooks:                     %d hook(s) configured\n", len(nar.Spec.RestoreSpec.Hooks.Resources)))
	}

	// ResourceModifiers - Admin Enforceable: (blank/unspecified) - special case (admins can enforce config-map in OADP Operator NS)
	if nar.Spec.RestoreSpec != nil && nar.Spec.RestoreSpec.ResourceModifier != nil {
		output.WriteString("Resource Modifiers:        configured\n")
	}

	output.WriteString("\n")
	output.WriteString("=== RESTRICTED FIELDS (Not shown for non-admin users) ===\n")
	output.WriteString("Backup Name:               [RESTRICTED]\n")
	output.WriteString("Schedule Name:             [RESTRICTED]\n")
	output.WriteString("Included Namespaces:       [RESTRICTED]\n")
	output.WriteString("Excluded Namespaces:       [RESTRICTED]\n")
	output.WriteString("Namespace Mapping:         [RESTRICTED]\n")

	output.WriteString("\n")

	// Restore timing and progress - Status information (always shown when available)
	if nar.Status.VeleroRestore != nil && nar.Status.VeleroRestore.Status != nil {
		vStatus := nar.Status.VeleroRestore.Status

		output.WriteString("=== STATUS INFORMATION ===\n")

		if vStatus.StartTimestamp != nil {
			output.WriteString(fmt.Sprintf("Started:                   %s\n", vStatus.StartTimestamp.Format("2006-01-02 15:04:05 -0700 MST")))
		}
		if vStatus.CompletionTimestamp != nil {
			output.WriteString(fmt.Sprintf("Completed:                 %s\n", vStatus.CompletionTimestamp.Format("2006-01-02 15:04:05 -0700 MST")))
		}

		output.WriteString("\n")

		// Progress information - Admin Enforceable: Yes
		if vStatus.Progress != nil {
			output.WriteString(fmt.Sprintf("Total items to be restored: %d\n", vStatus.Progress.TotalItems))
			output.WriteString(fmt.Sprintf("Items restored:            %d\n", vStatus.Progress.ItemsRestored))
		}

		output.WriteString("\n")

		// Simplified resource list info - Admin Enforceable: Yes for counts
		output.WriteString("Resource List:\n")
		if vStatus.Progress != nil {
			output.WriteString(fmt.Sprintf("  Total items restored:     %d\n", vStatus.Progress.ItemsRestored))
		} else {
			output.WriteString("  <detailed resource breakdown not available>\n")
		}

		output.WriteString("\n")

		// Volume information - Admin Enforceable: Yes for volume counts
		output.WriteString("Restore Volumes:\n")
		output.WriteString("  Persistent Volumes:       <none restored>\n")

		output.WriteString("\n")

		// Hooks information - Admin Enforceable: Yes for hook status
		output.WriteString(fmt.Sprintf("Hooks Attempted:           %d\n", vStatus.HookStatus.HooksAttempted))
		output.WriteString(fmt.Sprintf("Hooks Failed:              %d\n", vStatus.HookStatus.HooksFailed))
	}

	return output.String()
}
