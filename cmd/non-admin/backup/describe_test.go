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
)

// TestFormatPodVolumeBackupsCompact tests the formatPodVolumeBackupsCompact function
func TestFormatPodVolumeBackupsCompact(t *testing.T) {
	tests := []struct {
		name           string
		volumeInfo     string
		expectedOutput string
	}{
		{
			name:           "empty volume info",
			volumeInfo:     "",
			expectedOutput: "",
		},
		{
			name:           "whitespace only volume info",
			volumeInfo:     "   \n\t  ",
			expectedOutput: "",
		},
		{
			name:           "malformed JSON",
			volumeInfo:     "not json at all",
			expectedOutput: "",
		},
		{
			name:       "single pod with single volume succeeded",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"test-pod","podNamespace":"default","volumeName":"vol1","size":1024}}]`,
			expectedOutput: `    Completed:
      default/test-pod: vol1 (size: 1024)
`,
		},
		{
			name:       "single pod with multiple volumes",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"test-pod","podNamespace":"default","volumeName":"vol1","size":1024}},{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"test-pod","podNamespace":"default","volumeName":"vol2","size":2048}}]`,
			expectedOutput: `    Completed:
      default/test-pod: vol1 (size: 1024), vol2 (size: 2048)
`,
		},
		{
			name:       "multiple pods with volumes (sorted)",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod1","podNamespace":"ns1","volumeName":"vol1","size":1024}},{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod2","podNamespace":"ns2","volumeName":"vol2","size":2048}}]`,
			expectedOutput: `    Completed:
      ns1/pod1: vol1 (size: 1024)
      ns2/pod2: vol2 (size: 2048)
`,
		},
		{
			name:       "failed PVB should be excluded",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod1","podNamespace":"ns1","volumeName":"vol1","size":1024}},{"backupMethod":"PodVolumeBackup","result":"Failed","pvbInfo":{"podName":"pod2","podNamespace":"ns2","volumeName":"vol2","size":2048}}]`,
			expectedOutput: `    Completed:
      ns1/pod1: vol1 (size: 1024)
`,
		},
		{
			name:       "non-PodVolumeBackup entries should be ignored",
			volumeInfo: `[{"backupMethod":"VeleroSnapshot","result":"succeeded","pvbInfo":{"podName":"pod1","podNamespace":"ns1","volumeName":"vol1","size":1024}},{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod2","podNamespace":"ns2","volumeName":"vol2","size":2048}}]`,
			expectedOutput: `    Completed:
      ns2/pod2: vol2 (size: 2048)
`,
		},
		{
			name:           "empty array",
			volumeInfo:     `[]`,
			expectedOutput: "",
		},
		{
			name:           "missing pvbInfo should be skipped",
			volumeInfo:     `[{"backupMethod":"PodVolumeBackup","result":"succeeded"}]`,
			expectedOutput: "",
		},
		{
			name:       "missing size field defaults to 0",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod1","podNamespace":"ns1","volumeName":"vol1"}}]`,
			expectedOutput: `    Completed:
      ns1/pod1: vol1 (size: 0)
`,
		},
		{
			name:       "multiple pods unsorted input should be sorted in output",
			volumeInfo: `[{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod2","podNamespace":"ns2","volumeName":"vol2","size":2048}},{"backupMethod":"PodVolumeBackup","result":"succeeded","pvbInfo":{"podName":"pod1","podNamespace":"ns1","volumeName":"vol1","size":1024}}]`,
			expectedOutput: `    Completed:
      ns1/pod1: vol1 (size: 1024)
      ns2/pod2: vol2 (size: 2048)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPodVolumeBackupsCompact(tt.volumeInfo)

			if result != tt.expectedOutput {
				t.Errorf("output mismatch\nexpected:\n%q\ngot:\n%q", tt.expectedOutput, result)
			}
		})
	}
}
