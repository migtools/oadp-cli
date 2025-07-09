package tests

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// MockK8sClient is a mock implementation of the Kubernetes client for testing
type MockK8sClient struct {
	client.Client
	createdObjects []client.Object
	updatedObjects []client.Object
	deletedObjects []client.Object
	objects        map[types.NamespacedName]client.Object
}

// NewMockK8sClient creates a new mock Kubernetes client
func NewMockK8sClient() *MockK8sClient {
	return &MockK8sClient{
		objects: make(map[types.NamespacedName]client.Object),
	}
}

// Create mocks the Create method
func (m *MockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	m.createdObjects = append(m.createdObjects, obj)

	// Store the object for later retrieval
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	m.objects[key] = obj.DeepCopyObject().(client.Object)

	return nil
}

// Update mocks the Update method
func (m *MockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	m.updatedObjects = append(m.updatedObjects, obj)

	// Update the stored object
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	m.objects[key] = obj.DeepCopyObject().(client.Object)

	return nil
}

// Delete mocks the Delete method
func (m *MockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	m.deletedObjects = append(m.deletedObjects, obj)

	// Remove the object from storage
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	delete(m.objects, key)

	return nil
}

// Get mocks the Get method
func (m *MockK8sClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if storedObj, exists := m.objects[key]; exists {
		// Copy the stored object to the target object
		obj.SetName(storedObj.GetName())
		obj.SetNamespace(storedObj.GetNamespace())
		obj.SetLabels(storedObj.GetLabels())
		obj.SetAnnotations(storedObj.GetAnnotations())

		// Handle specific types
		if backup, ok := storedObj.(*nacv1alpha1.NonAdminBackup); ok {
			if targetBackup, ok := obj.(*nacv1alpha1.NonAdminBackup); ok {
				targetBackup.Spec = backup.Spec
				targetBackup.Status = backup.Status
			}
		}

		return nil
	}

	return fmt.Errorf("object not found: %s/%s", key.Namespace, key.Name)
}

// List mocks the List method
func (m *MockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	// For now, return empty list
	return nil
}

// Scheme returns the runtime scheme
func (m *MockK8sClient) Scheme() *runtime.Scheme {
	s := runtime.NewScheme()
	nacv1alpha1.AddToScheme(s)
	velerov1api.AddToScheme(s)
	return s
}

// GetCreatedObjects returns all objects that were created
func (m *MockK8sClient) GetCreatedObjects() []client.Object {
	return m.createdObjects
}

// GetUpdatedObjects returns all objects that were updated
func (m *MockK8sClient) GetUpdatedObjects() []client.Object {
	return m.updatedObjects
}

// GetDeletedObjects returns all objects that were deleted
func (m *MockK8sClient) GetDeletedObjects() []client.Object {
	return m.deletedObjects
}

// ClearObjects clears all stored objects
func (m *MockK8sClient) ClearObjects() {
	m.objects = make(map[types.NamespacedName]client.Object)
	m.createdObjects = nil
	m.updatedObjects = nil
	m.deletedObjects = nil
}

// AddObject adds an object to the mock client's storage
func (m *MockK8sClient) AddObject(obj client.Object) {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	m.objects[key] = obj.DeepCopyObject().(client.Object)
}

// CreateTestBackup creates a test NonAdminBackup object for nonadmin CLI testing
func CreateTestBackup(name, namespace string, options ...BackupOption) *nacv1alpha1.NonAdminBackup {
	backup := &nacv1alpha1.NonAdminBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nacv1alpha1.NonAdminBackupSpec{
			BackupSpec: &velerov1api.BackupSpec{
				IncludedNamespaces: []string{namespace},
				IncludedResources:  []string{"*"},
			},
		},
	}

	// Apply options
	for _, option := range options {
		option(backup)
	}

	return backup
}

// BackupOption is a functional option for configuring backup objects
type BackupOption func(*nacv1alpha1.NonAdminBackup)

// WithLabels adds labels to the backup
func WithLabels(labels map[string]string) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		backup.Labels = labels
	}
}

// WithAnnotations adds annotations to the backup
func WithAnnotations(annotations map[string]string) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		backup.Annotations = annotations
	}
}

// WithResources sets the included resources
func WithResources(resources []string) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		if backup.Spec.BackupSpec != nil {
			backup.Spec.BackupSpec.IncludedResources = resources
		}
	}
}

// WithTTL sets the TTL for the backup
func WithTTL(ttl time.Duration) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		if backup.Spec.BackupSpec != nil {
			backup.Spec.BackupSpec.TTL = metav1.Duration{Duration: ttl}
		}
	}
}

// WithSnapshotVolumes sets the snapshot volumes flag
func WithSnapshotVolumes(snapshotVolumes bool) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		if backup.Spec.BackupSpec != nil {
			backup.Spec.BackupSpec.SnapshotVolumes = &snapshotVolumes
		}
	}
}

// WithDeleteBackup sets the delete backup flag
func WithDeleteBackup(deleteBackup bool) BackupOption {
	return func(backup *nacv1alpha1.NonAdminBackup) {
		backup.Spec.DeleteBackup = deleteBackup
	}
}

// ValidateBackupSpec validates a NonAdminBackup specification for nonadmin CLI testing
func ValidateBackupSpec(backup *nacv1alpha1.NonAdminBackup) error {
	if backup.Name == "" {
		return fmt.Errorf("backup name is required")
	}
	if backup.Namespace == "" {
		return fmt.Errorf("backup namespace is required")
	}
	if backup.Spec.BackupSpec == nil {
		return fmt.Errorf("backup spec is required")
	}
	return nil
}

// CompareBackupSpecs compares two NonAdminBackup specifications for nonadmin CLI testing
func CompareBackupSpecs(expected, actual *nacv1alpha1.NonAdminBackup) error {
	if expected.Name != actual.Name {
		return fmt.Errorf("name mismatch: expected %s, got %s", expected.Name, actual.Name)
	}
	if expected.Namespace != actual.Namespace {
		return fmt.Errorf("namespace mismatch: expected %s, got %s", expected.Namespace, actual.Namespace)
	}

	// Compare labels
	if len(expected.Labels) != len(actual.Labels) {
		return fmt.Errorf("labels count mismatch: expected %d, got %d", len(expected.Labels), len(actual.Labels))
	}
	for k, v := range expected.Labels {
		if actual.Labels[k] != v {
			return fmt.Errorf("label mismatch for key %s: expected %s, got %s", k, v, actual.Labels[k])
		}
	}

	// Compare annotations
	if len(expected.Annotations) != len(actual.Annotations) {
		return fmt.Errorf("annotations count mismatch: expected %d, got %d", len(expected.Annotations), len(actual.Annotations))
	}
	for k, v := range expected.Annotations {
		if actual.Annotations[k] != v {
			return fmt.Errorf("annotation mismatch for key %s: expected %s, got %s", k, v, actual.Annotations[k])
		}
	}

	// Compare backup specs
	if expected.Spec.BackupSpec != nil && actual.Spec.BackupSpec != nil {
		if len(expected.Spec.BackupSpec.IncludedNamespaces) != len(actual.Spec.BackupSpec.IncludedNamespaces) {
			return fmt.Errorf("included namespaces count mismatch: expected %d, got %d",
				len(expected.Spec.BackupSpec.IncludedNamespaces), len(actual.Spec.BackupSpec.IncludedNamespaces))
		}
		if len(expected.Spec.BackupSpec.IncludedResources) != len(actual.Spec.BackupSpec.IncludedResources) {
			return fmt.Errorf("included resources count mismatch: expected %d, got %d",
				len(expected.Spec.BackupSpec.IncludedResources), len(actual.Spec.BackupSpec.IncludedResources))
		}
		// Compare the actual resource arrays
		for i, expectedResource := range expected.Spec.BackupSpec.IncludedResources {
			if i >= len(actual.Spec.BackupSpec.IncludedResources) || actual.Spec.BackupSpec.IncludedResources[i] != expectedResource {
				return fmt.Errorf("included resources mismatch at index %d: expected %s, got %s",
					i, expectedResource, actual.Spec.BackupSpec.IncludedResources[i])
			}
		}
	}

	return nil
}
