package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupName := args[0]

			// Get the current namespace from kubectl context
			userNamespace, err := getCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}

			// Setup scheme and client for NonAdminBackup resources
			scheme := runtime.NewScheme()
			if err := nacv1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("failed to add OADP non-admin types to scheme: %w", err)
			}
			if err := velerov1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("failed to add Velero types to scheme: %w", err)
			}
			if err := corev1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("failed to add core v1 types to scheme: %w", err)
			}

			restConfig, err := f.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get rest config: %w", err)
			}

			kbClient, err := kbclient.New(restConfig, kbclient.Options{Scheme: scheme})
			if err != nil {
				return fmt.Errorf("failed to create controller-runtime client: %w", err)
			}

			// Shows NonAdminBackup resources
			var nabList nacv1alpha1.NonAdminBackupList
			if err := kbClient.List(context.TODO(), &nabList, kbclient.InNamespace(userNamespace)); err != nil {
				return fmt.Errorf("failed to list NonAdminBackup resources: %w", err)
			}

			// Finds the backup
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

			return NonAdminDescribeBackup(cmd, kbClient, foundNAB, userNamespace)
		},
		Example: `  # Describe a non-admin backup with detailed information
  kubectl oadp nonadmin backup describe my-backup`,
	}
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// NonAdminDescribeBackup mirrors Velero's output.DescribeBackup functionality
// but works within non-admin RBAC boundaries using NonAdminDownloadRequest
func NonAdminDescribeBackup(cmd *cobra.Command, kbClient kbclient.Client, nab *nacv1alpha1.NonAdminBackup, userNamespace string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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
		if results, err := downloadBackupData(ctx, kbClient, userNamespace, veleroBackupName, "BackupResults"); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Results:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(results, "  "))
		}

		// Get backup details using NonAdminDownloadRequest for BackupResourceList
		if resourceList, err := downloadBackupData(ctx, kbClient, userNamespace, veleroBackupName, "BackupResourceList"); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Resource List:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(resourceList, "  "))
		}

		// Get backup volume info using NonAdminDownloadRequest
		if volumeInfo, err := downloadBackupData(ctx, kbClient, userNamespace, veleroBackupName, "BackupVolumeInfos"); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "\nBackup Volume Info:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "%s", indent(volumeInfo, "  "))
		}

		// Get backup item operations using NonAdminDownloadRequest
		if itemOps, err := downloadBackupData(ctx, kbClient, userNamespace, veleroBackupName, "BackupItemOperations"); err == nil {
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
