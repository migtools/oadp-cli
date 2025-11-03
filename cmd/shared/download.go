/*
Copyright 2025 The OADP CLI Contributors.

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

package shared

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DownloadRequestOptions holds configuration for creating and processing NonAdminDownloadRequests
type DownloadRequestOptions struct {
	// BackupName is the name of the backup to download data for
	BackupName string
	// DataType is the type of data to download (e.g., "BackupLog", "BackupResults", etc.)
	DataType velerov1.DownloadTargetKind
	// Namespace is the namespace where the download request will be created
	Namespace string
	// Timeout is the maximum time to wait for the download request to be processed
	Timeout time.Duration
	// PollInterval is how often to check the status of the download request
	PollInterval time.Duration
}

// ProcessDownloadRequest creates a NonAdminDownloadRequest, waits for it to be processed,
// downloads the content from the signed URL, and returns it as a string.
// This function automatically cleans up the download request when done.
func ProcessDownloadRequest(ctx context.Context, kbClient kbclient.Client, opts DownloadRequestOptions) (string, error) {
	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 120 * time.Second
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 2 * time.Second
	}

	// Create NonAdminDownloadRequest
	req := &nacv1alpha1.NonAdminDownloadRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: opts.BackupName + "-" + strings.ToLower(string(opts.DataType)) + "-",
			Namespace:    opts.Namespace,
		},
		Spec: nacv1alpha1.NonAdminDownloadRequestSpec{
			Target: velerov1.DownloadTarget{
				Kind: opts.DataType,
				Name: opts.BackupName,
			},
		},
	}

	if err := kbClient.Create(ctx, req); err != nil {
		return "", fmt.Errorf("failed to create NonAdminDownloadRequest for %s: %w", opts.DataType, err)
	}

	// Clean up the download request when done
	defer func() {
		deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDelete()
		_ = kbClient.Delete(deleteCtx, req)
	}()

	// Wait for the download request to be processed
	signedURL, err := waitForDownloadURL(ctx, kbClient, req, opts.Timeout, opts.PollInterval)
	if err != nil {
		return "", err
	}

	// Download and return the content
	return DownloadContent(signedURL)
}

// waitForDownloadURL waits for a NonAdminDownloadRequest to be processed and returns the signed URL
func waitForDownloadURL(ctx context.Context, kbClient kbclient.Client, req *nacv1alpha1.NonAdminDownloadRequest, timeout, pollInterval time.Duration) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return "", fmt.Errorf("timed out waiting for NonAdminDownloadRequest to be processed")
		case <-ticker.C:
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
						return updated.Status.VeleroDownloadRequest.Status.DownloadURL, nil
					}
				}
			}

			// Check for failure conditions
			for _, condition := range updated.Status.Conditions {
				if condition.Status == "True" && condition.Reason == "Error" {
					return "", fmt.Errorf("NonAdminDownloadRequest failed: %s - %s", condition.Type, condition.Message)
				}
			}
		}
	}
}

// DownloadContent fetches content from a signed URL and returns it as a string.
// It handles both gzipped and non-gzipped content automatically.
func DownloadContent(url string) (string, error) {
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

// StreamDownloadContent fetches content from a signed URL and streams it to the provided writer.
// This is useful for large files like logs that should be streamed rather than loaded into memory.
func StreamDownloadContent(url string, writer io.Writer) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download content from URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download content: status %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Try to decompress if it's gzipped
	var reader io.Reader = resp.Body
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		reader = gzr
	}

	// Stream content to writer
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("failed to stream content: %w", err)
	}

	return nil
}
