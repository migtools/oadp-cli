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
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	velerov2alpha1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// getDataUploadsForBackup fetches DataUpload resources related to a backup
func getDataUploadsForBackup(ctx context.Context, kbClient kbclient.Client, backupName string) ([]velerov2alpha1.DataUpload, error) {
	var dataUploadList velerov2alpha1.DataUploadList
	
	// List all DataUploads - they are cluster-scoped resources
	if err := kbClient.List(ctx, &dataUploadList); err != nil {
		// If we can't list DataUploads (maybe missing RBAC), return empty list without error
		// This allows the describe command to continue working for users without DataUpload permissions
		return nil, nil
	}

	var relatedUploads []velerov2alpha1.DataUpload
	for _, upload := range dataUploadList.Items {
		// Check if this DataUpload is related to our backup
		// DataUploads are typically labeled or have references to the backup
		if isDataUploadRelatedToBackup(upload, backupName) {
			relatedUploads = append(relatedUploads, upload)
		}
	}

	return relatedUploads, nil
}

// getDataDownloadsForBackup fetches DataDownload resources related to a backup
func getDataDownloadsForBackup(ctx context.Context, kbClient kbclient.Client, backupName string) ([]velerov2alpha1.DataDownload, error) {
	var dataDownloadList velerov2alpha1.DataDownloadList
	
	// List all DataDownloads - they are cluster-scoped resources
	if err := kbClient.List(ctx, &dataDownloadList); err != nil {
		// If we can't list DataDownloads (maybe missing RBAC), return empty list without error
		return nil, nil
	}

	var relatedDownloads []velerov2alpha1.DataDownload
	for _, download := range dataDownloadList.Items {
		// Check if this DataDownload is related to our backup
		if isDataDownloadRelatedToBackup(download, backupName) {
			relatedDownloads = append(relatedDownloads, download)
		}
	}

	return relatedDownloads, nil
}

// isDataUploadRelatedToBackup checks if a DataUpload is related to the given backup
func isDataUploadRelatedToBackup(upload velerov2alpha1.DataUpload, backupName string) bool {
	// Check labels for backup name
	if upload.Labels != nil {
		if labelValue, exists := upload.Labels["velero.io/backup-name"]; exists && labelValue == backupName {
			return true
		}
	}
	
	// Check annotations for backup name
	if upload.Annotations != nil {
		if annotationValue, exists := upload.Annotations["velero.io/backup-name"]; exists && annotationValue == backupName {
			return true
		}
	}
	
	// Check if the DataUpload name contains the backup name (common pattern)
	if strings.Contains(upload.Name, backupName) {
		return true
	}
	
	return false
}

// isDataDownloadRelatedToBackup checks if a DataDownload is related to the given backup
func isDataDownloadRelatedToBackup(download velerov2alpha1.DataDownload, backupName string) bool {
	// Check labels for backup name
	if download.Labels != nil {
		if labelValue, exists := download.Labels["velero.io/backup-name"]; exists && labelValue == backupName {
			return true
		}
	}
	
	// Check annotations for backup name
	if download.Annotations != nil {
		if annotationValue, exists := download.Annotations["velero.io/backup-name"]; exists && annotationValue == backupName {
			return true
		}
	}
	
	// Check if the DataDownload name contains the backup name (common pattern)
	if strings.Contains(download.Name, backupName) {
		return true
	}
	
	return false
}

// formatDataTransferInfo formats DataUpload/DataDownload information for display
func formatDataTransferInfo(uploads []velerov2alpha1.DataUpload, downloads []velerov2alpha1.DataDownload) string {
	var output strings.Builder
	
	if len(uploads) > 0 {
		output.WriteString("Data Uploads:\n")
		for _, upload := range uploads {
			output.WriteString(formatDataUploadInfo(upload))
			output.WriteString("\n")
		}
	}
	
	if len(downloads) > 0 {
		output.WriteString("Data Downloads:\n")
		for _, download := range downloads {
			output.WriteString(formatDataDownloadInfo(download))
			output.WriteString("\n")
		}
	}
	
	return output.String()
}

// formatDataUploadInfo formats a single DataUpload for display
func formatDataUploadInfo(upload velerov2alpha1.DataUpload) string {
	var info strings.Builder
	
	info.WriteString(fmt.Sprintf("  Name: %s\n", upload.Name))
	info.WriteString(fmt.Sprintf("  Status: %s\n", upload.Status.Phase))
	
	if upload.Status.StartTimestamp != nil {
		info.WriteString(fmt.Sprintf("  Started: %s\n", upload.Status.StartTimestamp.Format(time.RFC3339)))
	}
	
	// Show progress information
	if upload.Status.Progress.TotalBytes > 0 {
		info.WriteString(fmt.Sprintf("  Progress: %s/%s", 
			formatBytes(upload.Status.Progress.BytesDone),
			formatBytes(upload.Status.Progress.TotalBytes)))
		
		percentage := float64(upload.Status.Progress.BytesDone) / float64(upload.Status.Progress.TotalBytes) * 100
		info.WriteString(fmt.Sprintf(" (%.1f%%)\n", percentage))
		
		// Calculate and show transfer speed if possible
		if speed := calculateTransferSpeed(upload.Status.StartTimestamp, upload.Status.CompletionTimestamp, upload.Status.Progress.BytesDone); speed > 0 {
			info.WriteString(fmt.Sprintf("  Transfer Speed: %s/s\n", formatBytes(int64(speed))))
		}
	}
	
	if upload.Spec.BackupStorageLocation != "" {
		info.WriteString(fmt.Sprintf("  Storage Location: %s\n", upload.Spec.BackupStorageLocation))
	}
	
	if upload.Status.Node != "" {
		info.WriteString(fmt.Sprintf("  Node: %s\n", upload.Status.Node))
	}
	
	return info.String()
}

// formatDataDownloadInfo formats a single DataDownload for display
func formatDataDownloadInfo(download velerov2alpha1.DataDownload) string {
	var info strings.Builder
	
	info.WriteString(fmt.Sprintf("  Name: %s\n", download.Name))
	info.WriteString(fmt.Sprintf("  Status: %s\n", download.Status.Phase))
	
	if download.Status.StartTimestamp != nil {
		info.WriteString(fmt.Sprintf("  Started: %s\n", download.Status.StartTimestamp.Format(time.RFC3339)))
	}
	
	// Show progress information
	if download.Status.Progress.TotalBytes > 0 {
		info.WriteString(fmt.Sprintf("  Progress: %s/%s", 
			formatBytes(download.Status.Progress.BytesDone),
			formatBytes(download.Status.Progress.TotalBytes)))
		
		percentage := float64(download.Status.Progress.BytesDone) / float64(download.Status.Progress.TotalBytes) * 100
		info.WriteString(fmt.Sprintf(" (%.1f%%)\n", percentage))
		
		// Calculate and show transfer speed if possible
		if speed := calculateTransferSpeed(download.Status.StartTimestamp, download.Status.CompletionTimestamp, download.Status.Progress.BytesDone); speed > 0 {
			info.WriteString(fmt.Sprintf("  Transfer Speed: %s/s\n", formatBytes(int64(speed))))
		}
	}
	
	if download.Spec.BackupStorageLocation != "" {
		info.WriteString(fmt.Sprintf("  Storage Location: %s\n", download.Spec.BackupStorageLocation))
	}
	
	if download.Status.Node != "" {
		info.WriteString(fmt.Sprintf("  Node: %s\n", download.Status.Node))
	}
	
	return info.String()
}

// calculateTransferSpeed calculates transfer speed in bytes per second
func calculateTransferSpeed(startTime, endTime *metav1.Time, bytesTransferred int64) float64 {
	if startTime == nil || bytesTransferred <= 0 {
		return 0
	}
	
	var duration time.Duration
	if endTime != nil {
		// Transfer is complete, use actual duration
		duration = endTime.Sub(startTime.Time)
	} else {
		// Transfer is ongoing, use time since start
		duration = time.Since(startTime.Time)
	}
	
	if duration.Seconds() <= 0 {
		return 0
	}
	
	return float64(bytesTransferred) / duration.Seconds()
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	
	const unit = 1024
	exp := int(math.Log(float64(bytes)) / math.Log(unit))
	pre := "KMGTPE"[exp-1]
	
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/math.Pow(unit, float64(exp)), pre)
}

// getDataTransferStatus summarizes the status of data transfers
func getDataTransferStatus(uploads []velerov2alpha1.DataUpload, downloads []velerov2alpha1.DataDownload) string {
	totalCompleted := 0
	totalInProgress := 0
	totalFailed := 0
	total := len(uploads) + len(downloads)

	for _, upload := range uploads {
		switch upload.Status.Phase {
		case velerov2alpha1.DataUploadPhaseCompleted:
			totalCompleted++
		case velerov2alpha1.DataUploadPhaseInProgress, velerov2alpha1.DataUploadPhaseAccepted, velerov2alpha1.DataUploadPhasePrepared:
			totalInProgress++
		case velerov2alpha1.DataUploadPhaseFailed, velerov2alpha1.DataUploadPhaseCanceled:
			totalFailed++
		}
	}

	for _, download := range downloads {
		switch download.Status.Phase {
		case velerov2alpha1.DataDownloadPhaseCompleted:
			totalCompleted++
		case velerov2alpha1.DataDownloadPhaseInProgress, velerov2alpha1.DataDownloadPhaseAccepted, velerov2alpha1.DataDownloadPhasePrepared:
			totalInProgress++
		case velerov2alpha1.DataDownloadPhaseFailed, velerov2alpha1.DataDownloadPhaseCanceled:
			totalFailed++
		}
	}

	if total == 0 {
		return "-"
	} else if totalFailed > 0 {
		return "Failed"
	} else if totalInProgress > 0 {
		return "InProgress"
	} else if totalCompleted == total {
		return "Completed"
	} else {
		return "Mixed"
	}
}