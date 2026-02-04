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
	"net"
	"time"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewLogsCommand(f client.Factory, use string) *cobra.Command {
	var requestTimeout time.Duration

	c := &cobra.Command{
		Use:   use + " NAME",
		Short: "Show logs for a non-admin restore",
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

			// Verify the NonAdminRestore exists before creating download request
			var nar nacv1alpha1.NonAdminRestore
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: userNamespace,
				Name:      restoreName,
			}, &nar); err != nil {
				return fmt.Errorf("failed to get NonAdminRestore %q: %w", restoreName, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Waiting for restore logs to be processed (timeout: %v)...\n", effectiveTimeout)

			// Create download request and wait for signed URL
			req, signedURL, err := shared.CreateAndWaitForDownloadURL(ctx, kbClient, shared.DownloadRequestOptions{
				BackupName:   restoreName,
				DataType:     "RestoreLog",
				Namespace:    userNamespace,
				Timeout:      effectiveTimeout,
				PollInterval: 2 * time.Second,
				HTTPTimeout:  effectiveTimeout,
				OnProgress: func() {
					fmt.Fprintf(cmd.OutOrStdout(), ".")
				},
			})

			if err != nil {
				if req != nil {
					// Clean up on error
					if ctx.Err() == context.DeadlineExceeded {
						return shared.FormatDownloadRequestTimeoutError(kbClient, req, effectiveTimeout)
					}
					deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancelDelete()
					_ = kbClient.Delete(deleteCtx, req)
				}
				return err
			}

			// Clean up the download request when done
			defer func() {
				deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancelDelete()
				_ = kbClient.Delete(deleteCtx, req)
			}()

			fmt.Fprintf(cmd.OutOrStdout(), "\nDownload URL received, fetching logs...\n")

			// Use the shared StreamDownloadContent function to download and stream logs
			// Note: We use the same effective timeout for the HTTP download
			if err := shared.StreamDownloadContentWithTimeout(signedURL, cmd.OutOrStdout(), effectiveTimeout); err != nil {
				return fmt.Errorf("failed to download and stream logs: %w", err)
			}

			return nil
		},
		Example: `  kubectl oadp nonadmin restore logs my-restore
  kubectl oadp nonadmin restore logs my-restore --request-timeout=30m`,
	}

	c.Flags().DurationVar(&requestTimeout, "request-timeout", 0, fmt.Sprintf("The length of time to wait before giving up on a single server request (e.g., 30s, 5m, 1h). Overrides %s env var. Default: %v", shared.TimeoutEnvVar, shared.DefaultHTTPTimeout))

	return c
}
