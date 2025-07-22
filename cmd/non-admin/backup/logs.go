package backup

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
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
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

			// Get the current namespace from kubectl context
			userNamespace, err := getCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}
			backupName := args[0]

			scheme := runtime.NewScheme()
			if err := nacv1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("failed to add OADP non-admin types to scheme: %w", err)
			}
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

			// Verify the NonAdminBackup exists before creating download request
			var nab nacv1alpha1.NonAdminBackup
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: userNamespace,
				Name:      backupName,
			}, &nab); err != nil {
				return fmt.Errorf("failed to get NonAdminBackup %q: %w", backupName, err)
			}

			req := &nacv1alpha1.NonAdminDownloadRequest{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: backupName + "-logs-",
					Namespace:    userNamespace,
				},
				Spec: nacv1alpha1.NonAdminDownloadRequestSpec{
					Target: velerov1.DownloadTarget{
						Kind: "BackupLog",
						Name: backupName, // Use NonAdminBackup name, controller will resolve to Velero backup
					},
				},
			}

			if err := kbClient.Create(ctx, req); err != nil {
				return fmt.Errorf("failed to create NonAdminDownloadRequest: %w", err)
			}

			defer func() {
				deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelDelete()
				_ = kbClient.Delete(deleteCtx, req)
			}()

			var signedURL string
			timeout := time.After(120 * time.Second) // Increased timeout to 2 minutes
			tick := time.Tick(2 * time.Second)       // Check every 2 seconds instead of 1

			fmt.Fprintf(cmd.OutOrStdout(), "Waiting for backup logs to be processed...")
		Loop:
			for {
				select {
				case <-timeout:
					return fmt.Errorf("timed out waiting for NonAdminDownloadRequest to be processed")
				case <-tick:
					fmt.Fprintf(cmd.OutOrStdout(), ".")
					var updated nacv1alpha1.NonAdminDownloadRequest
					if err := kbClient.Get(ctx, kbclient.ObjectKey{
						Namespace: req.Namespace,
						Name:      req.Name,
					}, &updated); err != nil {
						return fmt.Errorf("failed to get NonAdminDownloadRequest: %w", err)
					}

					// Check if the download request was processed successfully
					for _, condition := range updated.Status.Conditions {
						if condition.Type == "Processed" && condition.Status == "True" {
							if updated.Status.VeleroDownloadRequest.Status.DownloadURL != "" {
								signedURL = updated.Status.VeleroDownloadRequest.Status.DownloadURL
								fmt.Fprintf(cmd.OutOrStdout(), "\nDownload URL received, fetching logs...\n")
								break Loop
							}
						}
					}

					// Check for failure conditions
					for _, condition := range updated.Status.Conditions {
						if condition.Status == "True" && condition.Reason == "Error" {
							return fmt.Errorf("NonAdminDownloadRequest failed: %s - %s", condition.Type, condition.Message)
						}
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
		Example: `  kubectl oadp nonadmin backup logs my-backup`,
	}
}
