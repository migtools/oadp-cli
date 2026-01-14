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
	"context"
	"fmt"
	"net"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewLogsCommand(f client.Factory, use string) *cobra.Command {
	var requestTimeout time.Duration

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Show logs for a non-admin backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get effective timeout (flag takes precedence over env var)
			effectiveTimeout := shared.GetHTTPTimeoutWithOverride(requestTimeout)

			// Create context with the effective timeout for the entire operation
			ctx, cancel := context.WithTimeout(context.Background(), effectiveTimeout)
			defer cancel()

			// Get the current namespace from kubectl context
			userNamespace, err := shared.GetCurrentNamespace()
			if err != nil {
				return fmt.Errorf("failed to determine current namespace: %w", err)
			}
			backupName := args[0]

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
			// Set timeout on REST config to prevent hanging when cluster is unreachable
			restConfig.Timeout = effectiveTimeout

			// Set a custom dial function with timeout to ensure TCP connection attempts
			// also respect the timeout (the default TCP dial timeout is ~30s)
			dialer := &net.Dialer{
				Timeout: effectiveTimeout,
			}
			restConfig.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, address)
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

			// Clean up the download request when done
			defer func() {
				deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelDelete()
				_ = kbClient.Delete(deleteCtx, req)
			}()

			fmt.Fprintf(cmd.OutOrStdout(), "Waiting for backup logs to be processed (timeout: %v)...\n", effectiveTimeout)

			// Wait for the download request to be processed
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			var signedURL string
		Loop:
			for {
				select {
				case <-ctx.Done():
					// Check if context was cancelled due to timeout or other reason
					if ctx.Err() == context.DeadlineExceeded {
						return shared.FormatDownloadRequestTimeoutError(kbClient, req, effectiveTimeout)
					}
					// Context cancelled for other reason (e.g., user interruption)
					return fmt.Errorf("operation cancelled: %w", ctx.Err())
				case <-ticker.C:
					fmt.Fprintf(cmd.OutOrStdout(), ".")
					var updated nacv1alpha1.NonAdminDownloadRequest
					if err := kbClient.Get(ctx, kbclient.ObjectKey{
						Namespace: req.Namespace,
						Name:      req.Name,
					}, &updated); err != nil {
						// If context expired during Get, handle it in next iteration
						if ctx.Err() != nil {
							continue
						}
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

			// Use the shared StreamDownloadContent function to download and stream logs
			// Note: We use the same effective timeout for the HTTP download
			if err := shared.StreamDownloadContentWithTimeout(signedURL, cmd.OutOrStdout(), effectiveTimeout); err != nil {
				return fmt.Errorf("failed to download and stream logs: %w", err)
			}

			return nil
		},
		Example: `  kubectl oadp nonadmin backup logs my-backup
  kubectl oadp nonadmin backup logs my-backup --request-timeout=30m`,
	}

	c.Flags().DurationVar(&requestTimeout, "request-timeout", 0, fmt.Sprintf("The length of time to wait before giving up on a single server request (e.g., 30s, 5m, 1h). Overrides %s env var. Default: %v", shared.TimeoutEnvVar, shared.DefaultHTTPTimeout))

	return c
}
