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
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

// TestDefaultHTTPTimeout verifies the default timeout constant
func TestDefaultHTTPTimeout(t *testing.T) {
	expected := 10 * time.Minute
	if DefaultHTTPTimeout != expected {
		t.Errorf("DefaultHTTPTimeout = %v, want %v", DefaultHTTPTimeout, expected)
	}
}

// TestTimeoutEnvVar verifies the environment variable name constant
func TestTimeoutEnvVar(t *testing.T) {
	expected := "OADP_CLI_REQUEST_TIMEOUT"
	if TimeoutEnvVar != expected {
		t.Errorf("TimeoutEnvVar = %q, want %q", TimeoutEnvVar, expected)
	}
}

// TestGetHTTPTimeout tests the getHTTPTimeout function
func TestGetHTTPTimeout(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{
			name:     "no env var set returns default",
			envValue: "",
			want:     DefaultHTTPTimeout,
		},
		{
			name:     "valid duration in minutes",
			envValue: "30m",
			want:     30 * time.Minute,
		},
		{
			name:     "valid duration in seconds",
			envValue: "120s",
			want:     120 * time.Second,
		},
		{
			name:     "valid duration in hours",
			envValue: "1h",
			want:     1 * time.Hour,
		},
		{
			name:     "valid complex duration",
			envValue: "1h30m",
			want:     90 * time.Minute,
		},
		{
			name:     "invalid duration falls back to default",
			envValue: "invalid",
			want:     DefaultHTTPTimeout,
		},
		{
			name:     "empty string returns default",
			envValue: "",
			want:     DefaultHTTPTimeout,
		},
		{
			name:     "numeric only (no unit) falls back to default",
			envValue: "30",
			want:     DefaultHTTPTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env var
			originalValue := os.Getenv(TimeoutEnvVar)
			defer os.Setenv(TimeoutEnvVar, originalValue)

			if tt.envValue != "" {
				os.Setenv(TimeoutEnvVar, tt.envValue)
			} else {
				os.Unsetenv(TimeoutEnvVar)
			}

			got := getHTTPTimeout()
			if got != tt.want {
				t.Errorf("getHTTPTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHttpClientWithTimeout verifies that the HTTP client is created with the correct timeout
func TestHttpClientWithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "1 minute timeout",
			timeout: 1 * time.Minute,
		},
		{
			name:    "30 second timeout",
			timeout: 30 * time.Second,
		},
		{
			name:    "default timeout",
			timeout: DefaultHTTPTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := httpClientWithTimeout(tt.timeout)
			if client == nil {
				t.Fatal("httpClientWithTimeout returned nil")
			}
			if client.Timeout != tt.timeout {
				t.Errorf("client.Timeout = %v, want %v", client.Timeout, tt.timeout)
			}
		})
	}
}

// TestDownloadContentWithTimeout tests downloading content with explicit timeout
func TestDownloadContentWithTimeout(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		contentType    string
		gzipped        bool
		timeout        time.Duration
		wantContent    string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful plain text download",
			serverResponse: "Hello, World!",
			serverStatus:   http.StatusOK,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantContent:    "Hello, World!",
			wantErr:        false,
		},
		{
			name:           "successful gzipped download",
			serverResponse: "Gzipped content here",
			serverStatus:   http.StatusOK,
			contentType:    "application/gzip",
			gzipped:        true,
			timeout:        5 * time.Second,
			wantContent:    "Gzipped content here",
			wantErr:        false,
		},
		{
			name:           "server returns 404",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantErr:        true,
			errContains:    "404",
		},
		{
			name:           "server returns 500",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantErr:        true,
			errContains:    "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				if tt.gzipped {
					w.Header().Set("Content-Encoding", "gzip")
					var buf bytes.Buffer
					gz := gzip.NewWriter(&buf)
					_, _ = gz.Write([]byte(tt.serverResponse))
					gz.Close()
					w.WriteHeader(tt.serverStatus)
					_, _ = w.Write(buf.Bytes())
				} else {
					w.WriteHeader(tt.serverStatus)
					_, _ = w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			content, err := DownloadContentWithTimeout(server.URL, tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DownloadContentWithTimeout() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DownloadContentWithTimeout() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("DownloadContentWithTimeout() unexpected error: %v", err)
				return
			}

			if content != tt.wantContent {
				t.Errorf("DownloadContentWithTimeout() = %q, want %q", content, tt.wantContent)
			}
		})
	}
}

// TestDownloadContent tests that DownloadContent uses the default timeout mechanism
func TestDownloadContent(t *testing.T) {
	// Save and restore original env var
	originalValue := os.Getenv(TimeoutEnvVar)
	defer os.Setenv(TimeoutEnvVar, originalValue)
	os.Unsetenv(TimeoutEnvVar)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	content, err := DownloadContent(server.URL)
	if err != nil {
		t.Errorf("DownloadContent() unexpected error: %v", err)
		return
	}

	if content != "test content" {
		t.Errorf("DownloadContent() = %q, want %q", content, "test content")
	}
}

// TestStreamDownloadContentWithTimeout tests streaming content with explicit timeout
func TestStreamDownloadContentWithTimeout(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		contentType    string
		gzipped        bool
		timeout        time.Duration
		wantContent    string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful plain text stream",
			serverResponse: "Streaming content",
			serverStatus:   http.StatusOK,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantContent:    "Streaming content",
			wantErr:        false,
		},
		{
			name:           "successful gzipped stream",
			serverResponse: "Gzipped streaming content",
			serverStatus:   http.StatusOK,
			contentType:    "application/gzip",
			gzipped:        true,
			timeout:        5 * time.Second,
			wantContent:    "Gzipped streaming content",
			wantErr:        false,
		},
		{
			name:           "server returns 403",
			serverResponse: "Forbidden",
			serverStatus:   http.StatusForbidden,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantErr:        true,
			errContains:    "403",
		},
		{
			name:           "large content stream",
			serverResponse: strings.Repeat("Large content block. ", 1000),
			serverStatus:   http.StatusOK,
			contentType:    "text/plain",
			gzipped:        false,
			timeout:        5 * time.Second,
			wantContent:    strings.Repeat("Large content block. ", 1000),
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				if tt.gzipped {
					w.Header().Set("Content-Encoding", "gzip")
					var buf bytes.Buffer
					gz := gzip.NewWriter(&buf)
					_, _ = gz.Write([]byte(tt.serverResponse))
					gz.Close()
					w.WriteHeader(tt.serverStatus)
					_, _ = w.Write(buf.Bytes())
				} else {
					w.WriteHeader(tt.serverStatus)
					_, _ = w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			var buf bytes.Buffer
			err := StreamDownloadContentWithTimeout(server.URL, &buf, tt.timeout)

			if tt.wantErr {
				if err == nil {
					t.Errorf("StreamDownloadContentWithTimeout() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("StreamDownloadContentWithTimeout() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("StreamDownloadContentWithTimeout() unexpected error: %v", err)
				return
			}

			if buf.String() != tt.wantContent {
				t.Errorf("StreamDownloadContentWithTimeout() = %q, want %q", buf.String(), tt.wantContent)
			}
		})
	}
}

// TestStreamDownloadContent tests that StreamDownloadContent uses the default timeout mechanism
func TestStreamDownloadContent(t *testing.T) {
	// Save and restore original env var
	originalValue := os.Getenv(TimeoutEnvVar)
	defer os.Setenv(TimeoutEnvVar, originalValue)
	os.Unsetenv(TimeoutEnvVar)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("streamed test content"))
	}))
	defer server.Close()

	var buf bytes.Buffer
	err := StreamDownloadContent(server.URL, &buf)
	if err != nil {
		t.Errorf("StreamDownloadContent() unexpected error: %v", err)
		return
	}

	if buf.String() != "streamed test content" {
		t.Errorf("StreamDownloadContent() = %q, want %q", buf.String(), "streamed test content")
	}
}

// TestDownloadContentWithTimeout_InvalidURL tests handling of invalid URLs
func TestDownloadContentWithTimeout_InvalidURL(t *testing.T) {
	_, err := DownloadContentWithTimeout("http://invalid-url-that-does-not-exist.local:12345", 1*time.Second)
	if err == nil {
		t.Error("DownloadContentWithTimeout() expected error for invalid URL, got nil")
	}
}

// TestStreamDownloadContentWithTimeout_InvalidURL tests handling of invalid URLs in streaming
func TestStreamDownloadContentWithTimeout_InvalidURL(t *testing.T) {
	var buf bytes.Buffer
	err := StreamDownloadContentWithTimeout("http://invalid-url-that-does-not-exist.local:12345", &buf, 1*time.Second)
	if err == nil {
		t.Error("StreamDownloadContentWithTimeout() expected error for invalid URL, got nil")
	}
}

// TestGetHTTPTimeoutWithEnvVar tests that the env var override works correctly
func TestGetHTTPTimeoutWithEnvVar(t *testing.T) {
	// Save and restore original env var
	originalValue := os.Getenv(TimeoutEnvVar)
	defer os.Setenv(TimeoutEnvVar, originalValue)

	// Set custom timeout
	os.Setenv(TimeoutEnvVar, "5m")

	timeout := getHTTPTimeout()
	expected := 5 * time.Minute

	if timeout != expected {
		t.Errorf("getHTTPTimeout() with env var = %v, want %v", timeout, expected)
	}
}

// TestGetHTTPTimeoutWithOverride tests the priority order: override > env var > default
func TestGetHTTPTimeoutWithOverride(t *testing.T) {
	tests := []struct {
		name     string
		override time.Duration
		envValue string
		want     time.Duration
	}{
		{
			name:     "override takes precedence over env var",
			override: 15 * time.Minute,
			envValue: "30m",
			want:     15 * time.Minute,
		},
		{
			name:     "override takes precedence over default when no env var",
			override: 20 * time.Minute,
			envValue: "",
			want:     20 * time.Minute,
		},
		{
			name:     "zero override falls back to env var",
			override: 0,
			envValue: "25m",
			want:     25 * time.Minute,
		},
		{
			name:     "zero override and no env var falls back to default",
			override: 0,
			envValue: "",
			want:     DefaultHTTPTimeout,
		},
		{
			name:     "zero override with invalid env var falls back to default",
			override: 0,
			envValue: "invalid",
			want:     DefaultHTTPTimeout,
		},
		{
			name:     "small override value is respected",
			override: 30 * time.Second,
			envValue: "10m",
			want:     30 * time.Second,
		},
		{
			name:     "large override value is respected",
			override: 2 * time.Hour,
			envValue: "5m",
			want:     2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env var
			originalValue := os.Getenv(TimeoutEnvVar)
			defer os.Setenv(TimeoutEnvVar, originalValue)

			if tt.envValue != "" {
				os.Setenv(TimeoutEnvVar, tt.envValue)
			} else {
				os.Unsetenv(TimeoutEnvVar)
			}

			got := GetHTTPTimeoutWithOverride(tt.override)
			if got != tt.want {
				t.Errorf("GetHTTPTimeoutWithOverride(%v) = %v, want %v", tt.override, got, tt.want)
			}
		})
	}
}

// TestDefaultOperationTimeout verifies the default operation timeout constant
func TestDefaultOperationTimeout(t *testing.T) {
	expected := 5 * time.Minute
	if DefaultOperationTimeout != expected {
		t.Errorf("DefaultOperationTimeout = %v, want %v", DefaultOperationTimeout, expected)
	}
}

// TestFormatDownloadRequestTimeoutError_NilClient tests error formatting when client is nil or request fails
func TestFormatDownloadRequestTimeoutError_BasicMessage(t *testing.T) {
	// Test that the function returns a properly formatted error message
	// even when we can't fetch the status (simulated by passing nil client)
	timeout := 5 * time.Minute

	// Create a mock request
	req := &nacv1alpha1.NonAdminDownloadRequest{}
	req.Name = "test-backup-logs-abc123"
	req.Namespace = "test-namespace"

	// With a nil client, the Get will fail, so we'll get the basic error message
	err := FormatDownloadRequestTimeoutError(nil, req, timeout)

	// Should contain timeout duration and request name
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "5m0s") {
		t.Errorf("error should contain timeout duration '5m0s', got: %s", errStr)
	}
	if !strings.Contains(errStr, "test-backup-logs-abc123") {
		t.Errorf("error should contain request name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "timed out") {
		t.Errorf("error should contain 'timed out', got: %s", errStr)
	}
}

// TestGetHTTPTimeoutWithOverride_ZeroReturnsDefault verifies that zero override with no env var returns default
func TestGetHTTPTimeoutWithOverride_ZeroReturnsDefault(t *testing.T) {
	// Save and restore original env var
	originalValue := os.Getenv(TimeoutEnvVar)
	defer os.Setenv(TimeoutEnvVar, originalValue)
	os.Unsetenv(TimeoutEnvVar)

	got := GetHTTPTimeoutWithOverride(0)
	if got != DefaultHTTPTimeout {
		t.Errorf("GetHTTPTimeoutWithOverride(0) without env var = %v, want %v", got, DefaultHTTPTimeout)
	}
}
