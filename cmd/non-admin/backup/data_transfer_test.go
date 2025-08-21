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

package backup

import (
	"testing"
	"time"

	velerov2alpha1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDataUploadRelatedToBackup(t *testing.T) {
	tests := []struct {
		name       string
		upload     velerov2alpha1.DataUpload
		backupName string
		expected   bool
	}{
		{
			name: "upload with backup label",
			upload: velerov2alpha1.DataUpload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-upload",
					Labels: map[string]string{
						"velero.io/backup-name": "my-backup",
					},
				},
			},
			backupName: "my-backup",
			expected:   true,
		},
		{
			name: "upload with backup annotation",
			upload: velerov2alpha1.DataUpload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-upload",
					Annotations: map[string]string{
						"velero.io/backup-name": "my-backup",
					},
				},
			},
			backupName: "my-backup",
			expected:   true,
		},
		{
			name: "upload with backup name in resource name",
			upload: velerov2alpha1.DataUpload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-backup-upload-123",
				},
			},
			backupName: "my-backup",
			expected:   true,
		},
		{
			name: "upload not related to backup",
			upload: velerov2alpha1.DataUpload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "other-upload",
				},
			},
			backupName: "my-backup",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDataUploadRelatedToBackup(tt.upload, tt.backupName)
			if result != tt.expected {
				t.Errorf("isDataUploadRelatedToBackup() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsDataDownloadRelatedToBackup(t *testing.T) {
	tests := []struct {
		name       string
		download   velerov2alpha1.DataDownload
		backupName string
		expected   bool
	}{
		{
			name: "download with backup label",
			download: velerov2alpha1.DataDownload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-download",
					Labels: map[string]string{
						"velero.io/backup-name": "my-backup",
					},
				},
			},
			backupName: "my-backup",
			expected:   true,
		},
		{
			name: "download not related to backup",
			download: velerov2alpha1.DataDownload{
				ObjectMeta: metav1.ObjectMeta{
					Name: "other-download",
				},
			},
			backupName: "my-backup",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDataDownloadRelatedToBackup(tt.download, tt.backupName)
			if result != tt.expected {
				t.Errorf("isDataDownloadRelatedToBackup() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateTransferSpeed(t *testing.T) {
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	
	tests := []struct {
		name              string
		startTime         *metav1.Time
		endTime           *metav1.Time
		bytesTransferred  int64
		expectedMinSpeed  float64 // minimum expected speed (allowing for some variance in time)
		expectedMaxSpeed  float64 // maximum expected speed
	}{
		{
			name:              "completed transfer",
			startTime:         &metav1.Time{Time: oneMinuteAgo},
			endTime:           &metav1.Time{Time: now},
			bytesTransferred:  1024 * 60, // 60KB in 60 seconds = 1KB/s
			expectedMinSpeed:  900,        // ~1KB/s (allowing variance)
			expectedMaxSpeed:  1100,
		},
		{
			name:              "zero bytes transferred",
			startTime:         &metav1.Time{Time: oneMinuteAgo},
			endTime:           &metav1.Time{Time: now},
			bytesTransferred:  0,
			expectedMinSpeed:  0,
			expectedMaxSpeed:  0,
		},
		{
			name:              "no start time",
			startTime:         nil,
			endTime:           &metav1.Time{Time: now},
			bytesTransferred:  1024,
			expectedMinSpeed:  0,
			expectedMaxSpeed:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speed := calculateTransferSpeed(tt.startTime, tt.endTime, tt.bytesTransferred)
			if speed < tt.expectedMinSpeed || speed > tt.expectedMaxSpeed {
				t.Errorf("calculateTransferSpeed() = %v, want between %v and %v", speed, tt.expectedMinSpeed, tt.expectedMaxSpeed)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1536, // 1.5KB
			expected: "1.5 KiB",
		},
		{
			name:     "megabytes",
			bytes:    1048576, // 1MB
			expected: "1.0 MiB",
		},
		{
			name:     "gigabytes",
			bytes:    2147483648, // 2GB
			expected: "2.0 GiB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetDataTransferStatus(t *testing.T) {
	tests := []struct {
		name      string
		uploads   []velerov2alpha1.DataUpload
		downloads []velerov2alpha1.DataDownload
		expected  string
	}{
		{
			name:      "no transfers",
			uploads:   []velerov2alpha1.DataUpload{},
			downloads: []velerov2alpha1.DataDownload{},
			expected:  "-",
		},
		{
			name: "all completed",
			uploads: []velerov2alpha1.DataUpload{
				{Status: velerov2alpha1.DataUploadStatus{Phase: velerov2alpha1.DataUploadPhaseCompleted}},
			},
			downloads: []velerov2alpha1.DataDownload{
				{Status: velerov2alpha1.DataDownloadStatus{Phase: velerov2alpha1.DataDownloadPhaseCompleted}},
			},
			expected: "Completed",
		},
		{
			name: "some failed",
			uploads: []velerov2alpha1.DataUpload{
				{Status: velerov2alpha1.DataUploadStatus{Phase: velerov2alpha1.DataUploadPhaseCompleted}},
				{Status: velerov2alpha1.DataUploadStatus{Phase: velerov2alpha1.DataUploadPhaseFailed}},
			},
			downloads: []velerov2alpha1.DataDownload{},
			expected:  "Failed",
		},
		{
			name: "some in progress",
			uploads: []velerov2alpha1.DataUpload{
				{Status: velerov2alpha1.DataUploadStatus{Phase: velerov2alpha1.DataUploadPhaseInProgress}},
			},
			downloads: []velerov2alpha1.DataDownload{
				{Status: velerov2alpha1.DataDownloadStatus{Phase: velerov2alpha1.DataDownloadPhaseCompleted}},
			},
			expected: "InProgress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDataTransferStatus(tt.uploads, tt.downloads)
			if result != tt.expected {
				t.Errorf("getDataTransferStatus() = %v, want %v", result, tt.expected)
			}
		})
	}
}