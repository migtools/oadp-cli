package backup

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewDescribeCommand(f client.Factory, use string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " NAME",
		Short: "Describe a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupName := args[0]
			veleroNamespace := "openshift-adp"

			dynClient, err := f.DynamicClient()
			if err != nil {
				return fmt.Errorf("failed to get dynamic client: %w", err)
			}

			backupList, err := dynClient.Resource(velerov1.SchemeGroupVersion.WithResource("backups")).Namespace(veleroNamespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list Velero backups: %w", err)
			}

			var found *velerov1.Backup
			for _, item := range backupList.Items {
				if item.GetName() == backupName ||
					item.GetAnnotations()["openshift.io/oadp-nab-origin-name"] == backupName {
					var b velerov1.Backup
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), &b); err == nil {
						found = &b
						break
					}
				}
			}

			if found == nil {
				return fmt.Errorf("no Velero backup found for non-admin backup %s", backupName)
			}

			// Print metadata
			fmt.Fprintf(cmd.OutOrStdout(), "Name:         %s\n", found.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Namespace:    %s\n", found.Namespace)

			// Print labels
			fmt.Fprintf(cmd.OutOrStdout(), "Labels:\n")
			labelKeys := make([]string, 0, len(found.Labels))
			for k := range found.Labels {
				labelKeys = append(labelKeys, k)
			}
			sort.Strings(labelKeys)
			for _, k := range labelKeys {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s=%s\n", k, found.Labels[k])
			}

			// Print annotations
			fmt.Fprintf(cmd.OutOrStdout(), "Annotations:\n")
			annotationKeys := make([]string, 0, len(found.Annotations))
			for k := range found.Annotations {
				annotationKeys = append(annotationKeys, k)
			}
			sort.Strings(annotationKeys)
			for _, k := range annotationKeys {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s=%s\n", k, found.Annotations[k])
			}

			// Print creation timestamp, etc.
			fmt.Fprintf(cmd.OutOrStdout(), "Creation Timestamp:  %s\n", found.CreationTimestamp.Format(time.RFC3339))
			fmt.Fprintf(cmd.OutOrStdout(), "Phase:               %s\n", found.Status.Phase)
			fmt.Fprintf(cmd.OutOrStdout(), "Start Timestamp:     %s\n", formatTime(found.Status.StartTimestamp))
			fmt.Fprintf(cmd.OutOrStdout(), "Completion Timestamp:%s\n", formatTime(found.Status.CompletionTimestamp))
			fmt.Fprintf(cmd.OutOrStdout(), "Expiration:          %s\n", formatTime(found.Status.Expiration))
			fmt.Fprintf(cmd.OutOrStdout(), "Format Version:      %s\n", found.Status.FormatVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "Version:             %d\n", found.Status.Version)

			// Print Spec (all fields, YAML for clarity)
			specYaml, err := yaml.Marshal(found.Spec)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Spec: <error marshaling spec: %v>\n", err)
			} else {
				// Remove the IncludedNamespaces line(s) from the YAML output
				lines := strings.Split(string(specYaml), "\n")
				var filtered []string
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if !strings.HasPrefix(trimmed, "includedNamespaces:") {
						filtered = append(filtered, line)
					}
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Spec:\n%s", indent(strings.Join(filtered, "\n"), "  "))
			}

			// Print Status (all fields, YAML for clarity)
			statusYaml, err := yaml.Marshal(found.Status)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Status: <error marshaling status: %v>\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Status:\n%s", indent(string(statusYaml), "  "))
			}

			// Print Events
			fmt.Fprintf(cmd.OutOrStdout(), "Events:\n")
			kubeClient, err := f.KubeClient()
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  <error fetching events: %v>\n", err)
			} else {
				eventsClient := kubeClient.CoreV1().Events(veleroNamespace)
				eventList, err := eventsClient.List(context.TODO(), metav1.ListOptions{
					FieldSelector: fmt.Sprintf("involvedObject.kind=Backup,involvedObject.name=%s", found.Name),
				})
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "  <error fetching events: %v>\n", err)
				} else if len(eventList.Items) == 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  <none>\n")
				} else {
					for _, e := range eventList.Items {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s\n", e.Reason, e.Message)
					}
				}
			}

			return nil
		},
		Example: `oc oadp nonadmin backup describe my-backup`,
	}
}

// Helper to indent YAML blocks for pretty output
func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) > 0 {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// Helper to format metav1.Time or nil
func formatTime(t *metav1.Time) string {
	if t == nil || t.IsZero() {
		return "<none>"
	}
	return t.Time.Format(time.RFC3339)
}
