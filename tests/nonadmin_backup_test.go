package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// TestNonAdminBackupCreateWithMockClient tests the nonadmin backup creation with a mock client
func TestNonAdminBackupCreateWithMockClient(t *testing.T) {
	// Create a mock client
	mockClient := NewMockK8sClient()

	// Test cases for different backup configurations
	tests := []struct {
		name         string
		backupName   string
		namespace    string
		labels       map[string]string
		annotations  map[string]string
		resources    []string
		ttl          time.Duration
		expectedSpec func() *nacv1alpha1.NonAdminBackup
	}{
		{
			name:       "basic backup",
			backupName: "test-backup",
			namespace:  "test-namespace",
			resources:  []string{"*"},
			expectedSpec: func() *nacv1alpha1.NonAdminBackup {
				return CreateTestBackup("test-backup", "test-namespace", WithResources([]string{"*"}))
			},
		},
		{
			name:       "backup with labels and annotations",
			backupName: "labeled-backup",
			namespace:  "test-namespace",
			labels: map[string]string{
				"app": "test",
				"env": "dev",
			},
			annotations: map[string]string{
				"description": "test backup",
			},
			resources: []string{"deployments", "services"},
			expectedSpec: func() *nacv1alpha1.NonAdminBackup {
				return CreateTestBackup("labeled-backup", "test-namespace",
					WithLabels(map[string]string{"app": "test", "env": "dev"}),
					WithAnnotations(map[string]string{"description": "test backup"}),
					WithResources([]string{"deployments", "services"}),
				)
			},
		},
		{
			name:       "backup with TTL",
			backupName: "ttl-backup",
			namespace:  "test-namespace",
			resources:  []string{"*"},
			ttl:        24 * time.Hour,
			expectedSpec: func() *nacv1alpha1.NonAdminBackup {
				return CreateTestBackup("ttl-backup", "test-namespace",
					WithResources([]string{"*"}),
					WithTTL(24*time.Hour),
				)
			},
		},
		{
			name:       "backup with snapshot volumes disabled",
			backupName: "no-snapshot-backup",
			namespace:  "test-namespace",
			resources:  []string{"*"},
			expectedSpec: func() *nacv1alpha1.NonAdminBackup {
				return CreateTestBackup("no-snapshot-backup", "test-namespace",
					WithResources([]string{"*"}),
					WithSnapshotVolumes(false),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous objects
			mockClient.ClearObjects()

			// Create the expected backup
			expectedBackup := tt.expectedSpec()

			// Call the mock client to create the backup
			err := mockClient.Create(context.Background(), expectedBackup)
			require.NoError(t, err)

			// Verify the created object
			require.Len(t, mockClient.GetCreatedObjects(), 1)
			createdBackup := mockClient.GetCreatedObjects()[0].(*nacv1alpha1.NonAdminBackup)

			// Validate the backup spec
			err = ValidateBackupSpec(createdBackup)
			require.NoError(t, err)

			// Compare the created backup with expected
			err = CompareBackupSpecs(expectedBackup, createdBackup)
			require.NoError(t, err)

			// Additional specific checks
			assert.Equal(t, expectedBackup.Name, createdBackup.Name)
			assert.Equal(t, expectedBackup.Namespace, createdBackup.Namespace)
			assert.Equal(t, expectedBackup.Labels, createdBackup.Labels)
			assert.Equal(t, expectedBackup.Annotations, createdBackup.Annotations)

			// Compare the backup spec details
			if expectedBackup.Spec.BackupSpec != nil && createdBackup.Spec.BackupSpec != nil {
				assert.Equal(t, expectedBackup.Spec.BackupSpec.IncludedNamespaces, createdBackup.Spec.BackupSpec.IncludedNamespaces)
				assert.Equal(t, expectedBackup.Spec.BackupSpec.IncludedResources, createdBackup.Spec.BackupSpec.IncludedResources)
				assert.Equal(t, expectedBackup.Spec.BackupSpec.TTL, createdBackup.Spec.BackupSpec.TTL)
				assert.Equal(t, expectedBackup.Spec.BackupSpec.SnapshotVolumes, createdBackup.Spec.BackupSpec.SnapshotVolumes)
			}
		})
	}
}

// TestNonAdminBackupDeleteWithMockClient tests the nonadmin backup deletion with a mock client
func TestNonAdminBackupDeleteWithMockClient(t *testing.T) {
	// Create a mock client
	mockClient := NewMockK8sClient()

	// Test cases
	tests := []struct {
		name         string
		backupName   string
		namespace    string
		backupExists bool
		expectError  bool
	}{
		{
			name:         "delete existing backup",
			backupName:   "test-backup",
			namespace:    "test-namespace",
			backupExists: true,
			expectError:  false,
		},
		{
			name:         "delete non-existent backup",
			backupName:   "non-existent",
			namespace:    "test-namespace",
			backupExists: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any previous objects
			mockClient.ClearObjects()

			if tt.backupExists {
				// Create an existing backup
				existingBackup := CreateTestBackup(tt.backupName, tt.namespace, WithDeleteBackup(false))
				mockClient.AddObject(existingBackup)

				// Get the backup
				backup := &nacv1alpha1.NonAdminBackup{}
				err := mockClient.Get(context.Background(), types.NamespacedName{
					Name:      tt.backupName,
					Namespace: tt.namespace,
				}, backup)

				if !tt.expectError {
					require.NoError(t, err)

					// Verify the backup exists and delete flag is false
					assert.Equal(t, tt.backupName, backup.Name)
					assert.Equal(t, tt.namespace, backup.Namespace)
					assert.False(t, backup.Spec.DeleteBackup)

					// Set delete flag
					backup.Spec.DeleteBackup = true

					// Update the backup
					err = mockClient.Update(context.Background(), backup)
					require.NoError(t, err)

					// Verify the update was called
					require.Len(t, mockClient.GetUpdatedObjects(), 1)
					updatedBackup := mockClient.GetUpdatedObjects()[0].(*nacv1alpha1.NonAdminBackup)
					assert.True(t, updatedBackup.Spec.DeleteBackup)

					// Verify the backup still exists in storage
					retrievedBackup := &nacv1alpha1.NonAdminBackup{}
					err = mockClient.Get(context.Background(), types.NamespacedName{
						Name:      tt.backupName,
						Namespace: tt.namespace,
					}, retrievedBackup)
					require.NoError(t, err)
					assert.True(t, retrievedBackup.Spec.DeleteBackup)
				}
			} else {
				// Try to get non-existent backup
				backup := &nacv1alpha1.NonAdminBackup{}
				err := mockClient.Get(context.Background(), types.NamespacedName{
					Name:      tt.backupName,
					Namespace: tt.namespace,
				}, backup)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), "object not found")
			}
		})
	}
}

// TestBackupSpecValidation tests the validation of NonAdminBackup specifications
func TestBackupSpecValidation(t *testing.T) {
	tests := []struct {
		name        string
		backupSpec  *nacv1alpha1.NonAdminBackup
		expectValid bool
	}{
		{
			name:        "valid backup spec",
			backupSpec:  CreateTestBackup("valid-backup", "test-namespace"),
			expectValid: true,
		},
		{
			name: "backup without name",
			backupSpec: &nacv1alpha1.NonAdminBackup{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
				Spec: nacv1alpha1.NonAdminBackupSpec{
					BackupSpec: &velerov1api.BackupSpec{
						IncludedNamespaces: []string{"test-namespace"},
						IncludedResources:  []string{"*"},
					},
				},
			},
			expectValid: false,
		},
		{
			name: "backup without namespace",
			backupSpec: &nacv1alpha1.NonAdminBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-backup",
				},
				Spec: nacv1alpha1.NonAdminBackupSpec{
					BackupSpec: &velerov1api.BackupSpec{
						IncludedNamespaces: []string{"test-namespace"},
						IncludedResources:  []string{"*"},
					},
				},
			},
			expectValid: false,
		},
		{
			name: "backup without backup spec",
			backupSpec: &nacv1alpha1.NonAdminBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup",
					Namespace: "test-namespace",
				},
				Spec: nacv1alpha1.NonAdminBackupSpec{},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the backup spec
			err := ValidateBackupSpec(tt.backupSpec)

			if tt.expectValid {
				assert.NoError(t, err, "Expected backup spec to be valid")
			} else {
				assert.Error(t, err, "Expected backup spec to be invalid")
			}
		})
	}
}

// TestBackupSpecComparison tests the comparison of NonAdminBackup specifications
func TestBackupSpecComparison(t *testing.T) {
	tests := []struct {
		name           string
		expectedBackup *nacv1alpha1.NonAdminBackup
		actualBackup   *nacv1alpha1.NonAdminBackup
		expectMatch    bool
	}{
		{
			name: "identical backups",
			expectedBackup: CreateTestBackup("test-backup", "test-namespace",
				WithLabels(map[string]string{"app": "test"}),
				WithResources([]string{"deployments", "services"}),
			),
			actualBackup: CreateTestBackup("test-backup", "test-namespace",
				WithLabels(map[string]string{"app": "test"}),
				WithResources([]string{"deployments", "services"}),
			),
			expectMatch: true,
		},
		{
			name:           "different names",
			expectedBackup: CreateTestBackup("expected-backup", "test-namespace"),
			actualBackup:   CreateTestBackup("actual-backup", "test-namespace"),
			expectMatch:    false,
		},
		{
			name:           "different namespaces",
			expectedBackup: CreateTestBackup("test-backup", "expected-namespace"),
			actualBackup:   CreateTestBackup("test-backup", "actual-namespace"),
			expectMatch:    false,
		},
		{
			name: "different labels",
			expectedBackup: CreateTestBackup("test-backup", "test-namespace",
				WithLabels(map[string]string{"app": "expected"}),
			),
			actualBackup: CreateTestBackup("test-backup", "test-namespace",
				WithLabels(map[string]string{"app": "actual"}),
			),
			expectMatch: false,
		},
		{
			name: "different resources",
			expectedBackup: CreateTestBackup("test-backup", "test-namespace",
				WithResources([]string{"deployments"}),
			),
			actualBackup: CreateTestBackup("test-backup", "test-namespace",
				WithResources([]string{"services"}),
			),
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compare the backup specs
			err := CompareBackupSpecs(tt.expectedBackup, tt.actualBackup)

			if tt.expectMatch {
				assert.NoError(t, err, "Expected backup specs to match")
			} else {
				assert.Error(t, err, "Expected backup specs to not match")
			}
		})
	}
}

// TestBackupOptions tests the NonAdminBackup option functions
func TestBackupOptions(t *testing.T) {
	t.Run("with labels", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		labels := map[string]string{"app": "test", "env": "dev"}

		WithLabels(labels)(backup)

		assert.Equal(t, labels, backup.Labels)
	})

	t.Run("with annotations", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		annotations := map[string]string{"description": "test backup"}

		WithAnnotations(annotations)(backup)

		assert.Equal(t, annotations, backup.Annotations)
	})

	t.Run("with resources", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		resources := []string{"deployments", "services"}

		WithResources(resources)(backup)

		assert.Equal(t, resources, backup.Spec.BackupSpec.IncludedResources)
	})

	t.Run("with TTL", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		ttl := 24 * time.Hour

		WithTTL(ttl)(backup)

		assert.Equal(t, metav1.Duration{Duration: ttl}, backup.Spec.BackupSpec.TTL)
	})

	t.Run("with snapshot volumes", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		snapshotVolumes := false

		WithSnapshotVolumes(snapshotVolumes)(backup)

		assert.Equal(t, &snapshotVolumes, backup.Spec.BackupSpec.SnapshotVolumes)
	})

	t.Run("with delete backup", func(t *testing.T) {
		backup := CreateTestBackup("test-backup", "test-namespace")
		deleteBackup := true

		WithDeleteBackup(deleteBackup)(backup)

		assert.Equal(t, deleteBackup, backup.Spec.DeleteBackup)
	})
}
