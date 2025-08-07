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
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
)

func NewLogsCommand(f client.Factory, use string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " NAME",
		Short: "Get logs for a non-admin restore",
		Long:  "Display logs for a specified non-admin restore operation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			// Get the current namespace from kubectl context
			userNamespace, err := shared.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}
			restoreName := args[0]

			// Create scheme with required types
			scheme, err := shared.NewSchemeWithTypes(shared.ClientOptions{
				IncludeNonAdminTypes: true,
				IncludeVeleroTypes:   true,
			})
			if err != nil {
				return err
			}

			restConfig, err := f.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get rest config: %w", err)
			}
			kbClient, err := kbclient.New(restConfig, kbclient.Options{Scheme: scheme})
			if err != nil {
				return fmt.Errorf("failed to create controller-runtime client: %w", err)
			}

			// Verify the NonAdminRestore exists before creating download request
			var nar nacv1alpha1.NonAdminRestore
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: userNamespace,
				Name:      restoreName,
			}, &nar); err != nil {
				return fmt.Errorf("failed to get NonAdminRestore %q: %w", restoreName, err)
			}

			req := &nacv1alpha1.NonAdminDownloadRequest{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: restoreName + "-logs-",
					Namespace:    userNamespace,
				},
				Spec: nacv1alpha1.NonAdminDownloadRequestSpec{
					Target: velerov1.DownloadTarget{
						Kind: "RestoreLog",
						Name: restoreName, // Use NonAdminRestore name, controller will resolve to Velero restore
					},
				},
			}

			if err := kbClient.Create(ctx, req); err != nil {
				return fmt.Errorf("failed to create NonAdminDownloadRequest: %w", err)
			}

			defer func() {
				deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancelDelete()
				_ = kbClient.Delete(deleteCtx, req)
			}()

			var signedURL string
			timeout := time.After(120 * time.Second) // Increased timeout to 2 minutes
			tick := time.Tick(2 * time.Second)       // Check every 2 seconds instead of 1

			fmt.Fprintf(cmd.OutOrStdout(), "Waiting for restore logs to be processed...")
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
				return fmt.Errorf("failed to download logs: %w", err)
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
		Example: `  # Get logs for a non-admin restore
  kubectl oadp nonadmin restore logs my-restore

  # Get logs for a restore in the current namespace
  kubectl oadp nonadmin restore logs production-restore`,
	}
}
