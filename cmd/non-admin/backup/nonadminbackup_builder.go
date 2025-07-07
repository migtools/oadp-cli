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

package backup

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

/*

Example usage:

var nonAdminBackup = builder.ForNonAdminBackup("user-namespace", "backup-1").
	ObjectMeta(
		builder.WithLabels("foo", "bar"),
	).
	BackupSpec(nacv1alpha1.NonAdminBackupSpec{
		BackupSpec: &velerov1api.BackupSpec{
			IncludedNamespaces: []string{"app-namespace"},
		},
	}).
	Result()

*/

// NonAdminBackupBuilder builds NonAdminBackup objects.
type NonAdminBackupBuilder struct {
	object *nacv1alpha1.NonAdminBackup
}

// ForNonAdminBackup is the constructor for a NonAdminBackupBuilder.
func ForNonAdminBackup(ns, name string) *NonAdminBackupBuilder {
	return &NonAdminBackupBuilder{
		object: &nacv1alpha1.NonAdminBackup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: nacv1alpha1.GroupVersion.String(),
				Kind:       "NonAdminBackup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		},
	}
}

// Result returns the built NonAdminBackup.
func (b *NonAdminBackupBuilder) Result() *nacv1alpha1.NonAdminBackup {
	return b.object
}

// ObjectMeta applies functional options to the NonAdminBackup's ObjectMeta.
func (b *NonAdminBackupBuilder) ObjectMeta(opts ...ObjectMetaOpt) *NonAdminBackupBuilder {
	for _, opt := range opts {
		opt(b.object)
	}

	return b
}

// BackupSpec sets the NonAdminBackup's backup spec.
func (b *NonAdminBackupBuilder) BackupSpec(spec nacv1alpha1.NonAdminBackupSpec) *NonAdminBackupBuilder {
	b.object.Spec = spec
	return b
}

// Phase sets the NonAdminBackup's phase.
func (b *NonAdminBackupBuilder) Phase(phase nacv1alpha1.NonAdminPhase) *NonAdminBackupBuilder {
	b.object.Status.Phase = phase
	return b
}

// VeleroBackup sets the reference to the created Velero backup.
func (b *NonAdminBackupBuilder) VeleroBackup(backupName, backupNamespace string) *NonAdminBackupBuilder {
	if b.object.Status.VeleroBackup == nil {
		b.object.Status.VeleroBackup = &nacv1alpha1.VeleroBackup{}
	}
	b.object.Status.VeleroBackup.Name = backupName
	b.object.Status.VeleroBackup.Namespace = backupNamespace
	return b
}

// Conditions sets the NonAdminBackup's conditions.
func (b *NonAdminBackupBuilder) Conditions(conditions []metav1.Condition) *NonAdminBackupBuilder {
	b.object.Status.Conditions = conditions
	return b
}

// WithStatus sets the NonAdminBackup's status.
func (b *NonAdminBackupBuilder) WithStatus(status nacv1alpha1.NonAdminBackupStatus) *NonAdminBackupBuilder {
	b.object.Status = status
	return b
}

// ObjectMetaOpt is a functional option for setting ObjectMeta properties.
type ObjectMetaOpt func(obj metav1.Object)

// WithLabels returns a functional option that sets labels on an object.
func WithLabels(key, value string) ObjectMetaOpt {
	return func(obj metav1.Object) {
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[key] = value
		obj.SetLabels(labels)
	}
}

// WithLabelsMap returns a functional option that sets labels from a map on an object.
func WithLabelsMap(labels map[string]string) ObjectMetaOpt {
	return func(obj metav1.Object) {
		existingLabels := obj.GetLabels()
		if existingLabels == nil {
			existingLabels = make(map[string]string)
		}
		for k, v := range labels {
			existingLabels[k] = v
		}
		obj.SetLabels(existingLabels)
	}
}

// WithAnnotations returns a functional option that sets annotations on an object.
func WithAnnotations(key, value string) ObjectMetaOpt {
	return func(obj metav1.Object) {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[key] = value
		obj.SetAnnotations(annotations)
	}
}

// WithAnnotationsMap returns a functional option that sets annotations from a map on an object.
func WithAnnotationsMap(annotations map[string]string) ObjectMetaOpt {
	return func(obj metav1.Object) {
		existingAnnotations := obj.GetAnnotations()
		if existingAnnotations == nil {
			existingAnnotations = make(map[string]string)
		}
		for k, v := range annotations {
			existingAnnotations[k] = v
		}
		obj.SetAnnotations(existingAnnotations)
	}
}

// getCurrentNamespace gets the current namespace from the kubeconfig context
func getCurrentNamespace() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	namespace, _, err := kubeConfig.Namespace()
	if err != nil {
		return "", fmt.Errorf("failed to get current namespace from kubeconfig: %w", err)
	}

	// If no namespace is set in kubeconfig, default to the user's name from context
	if namespace == "" || namespace == "default" {
		rawConfig, err := kubeConfig.RawConfig()
		if err != nil {
			return "", fmt.Errorf("failed to get raw kubeconfig: %w", err)
		}

		currentContext := rawConfig.CurrentContext
		if _, exists := rawConfig.Contexts[currentContext]; exists {
			// Try to extract user namespace from context name (assuming format like "user/cluster/user")
			parts := strings.Split(currentContext, "/")
			if len(parts) >= 3 {
				userNamespace := parts[2] // Assuming the user namespace is the third part
				return userNamespace, nil
			}
		}

		return "default", nil
	}

	return namespace, nil
}
