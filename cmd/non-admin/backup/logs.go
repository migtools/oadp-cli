package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewLogsCommand(f client.Factory, use string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " NAME",
		Short: "Show logs for a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			veleroNamespace := "openshift-adp"
			backupName := args[0]

			scheme := runtime.NewScheme()
			if err := velerov1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("failed to add Velero types to scheme: %w", err)
			}
			restConfig, err := f.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get rest config: %w", err)
			}
			kbClient, err := kbclient.New(restConfig, kbclient.Options{Scheme: scheme})
			if err != nil {
				return fmt.Errorf("failed to create controller-runtime client: %w", err)
			}

			req := &velerov1.DownloadRequest{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: backupName + "-logs-",
					Namespace:    veleroNamespace,
				},
				Spec: velerov1.DownloadRequestSpec{
					Target: velerov1.DownloadTarget{
						Kind: "BackupLog",
						Name: backupName,
					},
				},
			}

			if err := kbClient.Create(ctx, req); err != nil {
				return fmt.Errorf("failed to create DownloadRequest: %w", err)
			}

			defer func() {
				deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelDelete()
				if err := kbClient.Delete(deleteCtx, req); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: Failed to delete DownloadRequest %s/%s: %v\n", req.Namespace, req.Name, err)
				}
			}()

			var signedURL string
			timeout := time.After(60 * time.Second)
			tick := time.Tick(1 * time.Second)
		Loop:
			for {
				select {
				case <-timeout:
					return fmt.Errorf("timed out waiting for DownloadRequest to be processed")
				case <-tick:
					var updated velerov1.DownloadRequest
					if err := kbClient.Get(ctx, kbclient.ObjectKey{
						Namespace: req.Namespace,
						Name:      req.Name,
					}, &updated); err != nil {
						return fmt.Errorf("failed to get DownloadRequest: %w", err)
					}

					switch updated.Status.Phase {
					case velerov1.DownloadRequestPhaseProcessed:
						if updated.Status.DownloadURL != "" {
							signedURL = updated.Status.DownloadURL
							break Loop
						}
					case "Failed":
						return fmt.Errorf("DownloadRequest failed: phase=%s", updated.Status.Phase)
					default:
					}
				}
			}

			resp, err := http.Get(signedURL)
			if err != nil {
				return fmt.Errorf("failed to download logs from URL %q: %w", signedURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed to download logs: status %s, body: %s", resp.Status, string(bodyBytes))
			}

			gzr, err := gzip.NewReader(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzr.Close()

			scanner := bufio.NewScanner(gzr)
			for scanner.Scan() {
				fmt.Fprintln(cmd.OutOrStdout(), scanner.Text())
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				return fmt.Errorf("failed to read logs: %w", err)
			}

			return nil
		},
		Example: `  kubectl oadp backup logs my-backup`,
	}
}
