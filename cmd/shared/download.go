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
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultHTTPTimeout is the default timeout for HTTP requests when downloading content from object storage.
// This prevents the CLI from hanging indefinitely if the connection stalls.
const DefaultHTTPTimeout = 10 * time.Minute

// TimeoutEnvVar is the environment variable name that can be used to override the default timeout.
// Example: OADP_CLI_REQUEST_TIMEOUT=30m kubectl oadp nonadmin backup logs my-backup
const TimeoutEnvVar = "OADP_CLI_REQUEST_TIMEOUT"

// getHTTPTimeout returns the HTTP timeout to use for download operations.
// It checks for an environment variable override first, then falls back to the default.
func getHTTPTimeout() time.Duration {
	return GetHTTPTimeoutWithOverride(0)
}

// GetHTTPTimeoutWithOverride returns the HTTP timeout to use for download operations.
// Priority order: override parameter (if > 0) > environment variable > default.
// This allows CLI flags to take precedence over environment variables.
func GetHTTPTimeoutWithOverride(override time.Duration) time.Duration {
	// If an explicit override is provided (e.g., from --timeout flag), use it
	if override > 0 {
		log.Printf("Using HTTP timeout from command-line flag: %v", override)
		return override
	}

	// Check for environment variable
	if envTimeout := os.Getenv(TimeoutEnvVar); envTimeout != "" {
		if parsed, err := time.ParseDuration(envTimeout); err == nil {
			log.Printf("Using custom HTTP timeout from %s: %v", TimeoutEnvVar, parsed)
			return parsed
		}
		log.Printf("Warning: Invalid duration in %s=%q, using default %v", TimeoutEnvVar, envTimeout, DefaultHTTPTimeout)
	}

	return DefaultHTTPTimeout
}

// httpClientWithTimeout returns an HTTP client with a configured timeout.
// Using a custom client instead of http.DefaultClient ensures downloads don't hang indefinitely.
func httpClientWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

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
	// HTTPTimeout is the timeout for downloading content from the signed URL.
	// If zero, uses the default timeout (env var or DefaultHTTPTimeout).
	HTTPTimeout time.Duration
}

// ProcessDownloadRequest creates a NonAdminDownloadRequest, waits for it to be processed,
// downloads the content from the signed URL, and returns it as a string.
// This function automatically cleans up the download request when done.
func ProcessDownloadRequest(ctx context.Context, kbClient kbclient.Client, opts DownloadRequestOptions) (string, error) {
	log.Printf("[ProcessDownloadRequest] Starting for BackupName=%q, DataType=%q, Namespace=%q",
		opts.BackupName, opts.DataType, opts.Namespace)

	// Set defaults
	if opts.Timeout == 0 {
		opts.Timeout = 120 * time.Second
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 2 * time.Second
	}
	log.Printf("[ProcessDownloadRequest] Using Timeout=%v, PollInterval=%v, HTTPTimeout=%v",
		opts.Timeout, opts.PollInterval, opts.HTTPTimeout)

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

	log.Printf("[ProcessDownloadRequest] Creating NonAdminDownloadRequest with GenerateName=%q", req.ObjectMeta.GenerateName)
	if err := kbClient.Create(ctx, req); err != nil {
		log.Printf("[ProcessDownloadRequest] ERROR: Failed to create NonAdminDownloadRequest: %v", err)
		return "", fmt.Errorf("failed to create NonAdminDownloadRequest for %s: %w", opts.DataType, err)
	}
	log.Printf("[ProcessDownloadRequest] Successfully created NonAdminDownloadRequest with Name=%q", req.Name)

	// Clean up the download request when done
	defer func() {
		log.Printf("[ProcessDownloadRequest] Cleaning up NonAdminDownloadRequest Name=%q", req.Name)
		deleteCtx, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelDelete()
		if err := kbClient.Delete(deleteCtx, req); err != nil {
			log.Printf("[ProcessDownloadRequest] WARNING: Failed to delete NonAdminDownloadRequest: %v", err)
		} else {
			log.Printf("[ProcessDownloadRequest] Successfully deleted NonAdminDownloadRequest")
		}
	}()

	// Wait for the download request to be processed
	log.Printf("[ProcessDownloadRequest] Waiting for download URL...")
	signedURL, err := waitForDownloadURL(ctx, kbClient, req, opts.Timeout, opts.PollInterval)
	if err != nil {
		log.Printf("[ProcessDownloadRequest] ERROR: Failed to get download URL: %v", err)
		return "", err
	}
	log.Printf("[ProcessDownloadRequest] Received signed URL (length=%d)", len(signedURL))

	// Download and return the content using the specified HTTP timeout
	httpTimeout := GetHTTPTimeoutWithOverride(opts.HTTPTimeout)
	log.Printf("[ProcessDownloadRequest] Downloading content with HTTP timeout=%v", httpTimeout)
	content, err := DownloadContentWithTimeout(signedURL, httpTimeout)
	if err != nil {
		log.Printf("[ProcessDownloadRequest] ERROR: Failed to download content: %v", err)
		return "", err
	}
	log.Printf("[ProcessDownloadRequest] Successfully downloaded content (length=%d bytes)", len(content))
	return content, nil
}

// waitForDownloadURL waits for a NonAdminDownloadRequest to be processed and returns the signed URL
func waitForDownloadURL(ctx context.Context, kbClient kbclient.Client, req *nacv1alpha1.NonAdminDownloadRequest, timeout, pollInterval time.Duration) (string, error) {
	log.Printf("[waitForDownloadURL] Starting wait for Name=%q, timeout=%v, pollInterval=%v", req.Name, timeout, pollInterval)
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	pollCount := 0
	for {
		select {
		case <-timeoutCtx.Done():
			log.Printf("[waitForDownloadURL] TIMEOUT after %d polls", pollCount)
			return "", fmt.Errorf("timed out waiting for NonAdminDownloadRequest to be processed")
		case <-ticker.C:
			pollCount++
			log.Printf("[waitForDownloadURL] Poll #%d: Getting status for Name=%q", pollCount, req.Name)
			var updated nacv1alpha1.NonAdminDownloadRequest
			if err := kbClient.Get(ctx, kbclient.ObjectKey{
				Namespace: req.Namespace,
				Name:      req.Name,
			}, &updated); err != nil {
				log.Printf("[waitForDownloadURL] ERROR on poll #%d: Failed to get NonAdminDownloadRequest: %v", pollCount, err)
				return "", fmt.Errorf("failed to get NonAdminDownloadRequest: %w", err)
			}

			log.Printf("[waitForDownloadURL] Poll #%d: Got %d conditions", pollCount, len(updated.Status.Conditions))

			// Check if the download request was processed successfully
			for i, condition := range updated.Status.Conditions {
				log.Printf("[waitForDownloadURL] Poll #%d: Condition[%d] Type=%q, Status=%q, Reason=%q, Message=%q",
					pollCount, i, condition.Type, condition.Status, condition.Reason, condition.Message)

				if condition.Type == "Processed" && condition.Status == "True" {
					if updated.Status.VeleroDownloadRequest.Status.DownloadURL != "" {
						log.Printf("[waitForDownloadURL] SUCCESS on poll #%d: Got download URL", pollCount)
						return updated.Status.VeleroDownloadRequest.Status.DownloadURL, nil
					} else {
						log.Printf("[waitForDownloadURL] Poll #%d: Processed=True but no DownloadURL yet", pollCount)
					}
				}
			}

			// Check for failure conditions
			for _, condition := range updated.Status.Conditions {
				if condition.Status == "True" && condition.Reason == "Error" {
					log.Printf("[waitForDownloadURL] FAILURE on poll #%d: Error condition found", pollCount)
					return "", fmt.Errorf("NonAdminDownloadRequest failed: %s - %s", condition.Type, condition.Message)
				}
			}
		}
	}
}

// DownloadContent fetches content from a signed URL and returns it as a string.
// It handles both gzipped and non-gzipped content automatically.
// Uses DefaultHTTPTimeout (or OADP_CLI_REQUEST_TIMEOUT env var) to prevent hanging indefinitely.
func DownloadContent(url string) (string, error) {
	return DownloadContentWithTimeout(url, getHTTPTimeout())
}

// DownloadContentWithTimeout fetches content from a signed URL with a specified timeout.
// It handles both gzipped and non-gzipped content automatically.
func DownloadContentWithTimeout(url string, timeout time.Duration) (string, error) {
	log.Printf("[DownloadContentWithTimeout] Starting download with timeout=%v", timeout)
	client := httpClientWithTimeout(timeout)

	log.Printf("[DownloadContentWithTimeout] Sending HTTP GET request...")
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[DownloadContentWithTimeout] ERROR: HTTP GET failed: %v", err)
		return "", fmt.Errorf("failed to download content from URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	log.Printf("[DownloadContentWithTimeout] HTTP response: Status=%s, Content-Encoding=%s",
		resp.Status, resp.Header.Get("Content-Encoding"))

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[DownloadContentWithTimeout] ERROR: Non-OK status code: %s, body length=%d", resp.Status, len(bodyBytes))
		return "", fmt.Errorf("failed to download content: status %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Use a buffered reader to peek at the content and detect gzip format
	bufReader := bufio.NewReader(resp.Body)
	var reader io.Reader = bufReader

	// Check if content is gzipped by:
	// 1. Content-Encoding header (HTTP-level compression)
	// 2. Magic bytes 0x1f 0x8b at start (file-level gzip)
	// Object storage signed URLs often serve .gz files without Content-Encoding header,
	// so we need to detect gzip by inspecting the actual file content.
	isGzipped := strings.Contains(resp.Header.Get("Content-Encoding"), "gzip")

	if !isGzipped {
		// Peek at first 2 bytes to check for gzip magic bytes (0x1f 0x8b)
		magicBytes, err := bufReader.Peek(2)
		if err == nil && len(magicBytes) == 2 && magicBytes[0] == 0x1f && magicBytes[1] == 0x8b {
			isGzipped = true
			log.Printf("[DownloadContentWithTimeout] Detected gzip format by magic bytes")
		}
	}

	if isGzipped {
		log.Printf("[DownloadContentWithTimeout] Content is gzipped, creating gzip reader...")
		gzr, err := gzip.NewReader(bufReader)
		if err != nil {
			log.Printf("[DownloadContentWithTimeout] ERROR: Failed to create gzip reader: %v", err)
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzr.Close()
		reader = gzr
	}

	// Read all content
	log.Printf("[DownloadContentWithTimeout] Reading content from response body...")
	content, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("[DownloadContentWithTimeout] ERROR: Failed to read content: %v", err)
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	log.Printf("[DownloadContentWithTimeout] Successfully read %d bytes", len(content))
	return string(content), nil
}

// StreamDownloadContent fetches content from a signed URL and streams it to the provided writer.
// This is useful for large files like logs that should be streamed rather than loaded into memory.
// Uses DefaultHTTPTimeout (or OADP_CLI_REQUEST_TIMEOUT env var) to prevent hanging indefinitely.
func StreamDownloadContent(url string, writer io.Writer) error {
	return StreamDownloadContentWithTimeout(url, writer, getHTTPTimeout())
}

// StreamDownloadContentWithTimeout fetches content from a signed URL with a specified timeout
// and streams it to the provided writer.
func StreamDownloadContentWithTimeout(url string, writer io.Writer, timeout time.Duration) error {
	client := httpClientWithTimeout(timeout)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download content from URL %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to download content: status %s, body: %s", resp.Status, string(bodyBytes))
	}

	// Use a buffered reader to peek at the content and detect gzip format
	bufReader := bufio.NewReader(resp.Body)
	var reader io.Reader = bufReader

	// Check if content is gzipped by:
	// 1. Content-Encoding header (HTTP-level compression)
	// 2. Magic bytes 0x1f 0x8b at start (file-level gzip)
	// Object storage signed URLs often serve .gz files without Content-Encoding header,
	// so we need to detect gzip by inspecting the actual file content.
	isGzipped := strings.Contains(resp.Header.Get("Content-Encoding"), "gzip")

	if !isGzipped {
		// Peek at first 2 bytes to check for gzip magic bytes (0x1f 0x8b)
		magicBytes, err := bufReader.Peek(2)
		if err == nil && len(magicBytes) == 2 && magicBytes[0] == 0x1f && magicBytes[1] == 0x8b {
			isGzipped = true
		}
	}

	if isGzipped {
		gzr, err := gzip.NewReader(bufReader)
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

// DefaultOperationTimeout is the default timeout for waiting for download requests to be processed.
const DefaultOperationTimeout = 5 * time.Minute

// defaultStatusCheckTimeout is the timeout for checking status when formatting timeout errors.
const defaultStatusCheckTimeout = 5 * time.Second

// FormatDownloadRequestTimeoutError creates a helpful error message when a download request times out.
// It attempts to fetch the current status of the request to provide diagnostic information.
func FormatDownloadRequestTimeoutError(kbClient kbclient.Client, req *nacv1alpha1.NonAdminDownloadRequest, timeout time.Duration) error {
	// If client is available, try to get the current status for better diagnostics
	if kbClient != nil {
		// Use a fresh context to check final status since the original context is expired
		statusCtx, cancel := context.WithTimeout(context.Background(), defaultStatusCheckTimeout)
		defer cancel()

		var updated nacv1alpha1.NonAdminDownloadRequest
		if err := kbClient.Get(statusCtx, kbclient.ObjectKey{
			Namespace: req.Namespace,
			Name:      req.Name,
		}, &updated); err == nil {
			// Format status conditions for helpful error message
			var statusInfo string
			if len(updated.Status.Conditions) > 0 {
				var conditions []string
				for _, c := range updated.Status.Conditions {
					conditions = append(conditions, fmt.Sprintf("%s=%s (reason: %s)", c.Type, c.Status, c.Reason))
				}
				statusInfo = fmt.Sprintf(" Current status: %s.", strings.Join(conditions, ", "))
			}
			return fmt.Errorf("timed out after %v waiting for NonAdminDownloadRequest %q to be processed. statusInfo: %s", timeout, req.Name, statusInfo)
		}
	}

	return fmt.Errorf("timed out after %v waiting for NonAdminDownloadRequest %q to be processed", timeout, req.Name)
}
