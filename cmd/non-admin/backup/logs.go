package nonadmin

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewLogsCommand(f client.Factory, use string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " NAME",
		Short: "Show logs for a non-admin backup (from the Velero controller)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			veleroNamespace := "openshift-adp"

			kubeClient, err := f.KubeClient()
			if err != nil {
				return fmt.Errorf("failed to get kube client: %w", err)
			}

			podList, err := kubeClient.CoreV1().Pods(veleroNamespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: "app.kubernetes.io/name=velero",
			})
			if err != nil {
				return fmt.Errorf("failed to list Velero controller pods: %w", err)
			}
			if len(podList.Items) == 0 {
				return fmt.Errorf("no Velero controller pod found in namespace %s", veleroNamespace)
			}

			// Print logs from the first Velero controller pod
			pod := podList.Items[0]
			fmt.Fprintf(cmd.OutOrStdout(), "Logs from Velero controller pod %s:\n", pod.Name)
			req := kubeClient.CoreV1().Pods(veleroNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
			logStream, err := req.Stream(context.TODO())
			if err != nil {
				return fmt.Errorf("failed to stream logs: %w", err)
			}
			defer logStream.Close()
			buf := new(strings.Builder)
			_, err = io.Copy(buf, logStream)
			if err != nil {
				return fmt.Errorf("failed to read logs: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), buf.String())

			return nil
		},
		Example: `  oc oadp backup logs my-backup`,
	}
}
