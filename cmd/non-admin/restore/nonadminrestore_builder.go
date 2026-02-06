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

package restore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

/*

Example usage:

var nonAdminRestore = builder.ForNonAdminRestore("user-namespace", "restore-1").
	ObjectMeta(
		builder.WithLabels("foo", "bar"),
	).
	RestoreSpec(nacv1alpha1.NonAdminRestoreSpec{
		RestoreSpec: &velerov1api.RestoreSpec{
			BackupName: "backup-1",
		},
	}).
	Result()

*/

// NonAdminRestoreBuilder builds NonAdminRestore objects.
type NonAdminRestoreBuilder struct {
	object *nacv1alpha1.NonAdminRestore
}

// ForNonAdminRestore is the constructor for a NonAdminRestoreBuilder.
func ForNonAdminRestore(ns, name string) *NonAdminRestoreBuilder {
	objMeta := metav1.ObjectMeta{
		Namespace: ns,
	}

	// If name is empty, use GenerateName for auto-generation
	if name == "" {
		objMeta.GenerateName = "restore-"
	} else {
		objMeta.Name = name
	}

	return &NonAdminRestoreBuilder{
		object: &nacv1alpha1.NonAdminRestore{
			TypeMeta: metav1.TypeMeta{
				APIVersion: nacv1alpha1.GroupVersion.String(),
				Kind:       "NonAdminRestore",
			},
			ObjectMeta: objMeta,
		},
	}
}

// Result returns the built NonAdminRestore.
func (b *NonAdminRestoreBuilder) Result() *nacv1alpha1.NonAdminRestore {
	return b.object
}

// ObjectMeta applies functional options to the NonAdminRestore's ObjectMeta.
func (b *NonAdminRestoreBuilder) ObjectMeta(opts ...ObjectMetaOpt) *NonAdminRestoreBuilder {
	for _, opt := range opts {
		opt(b.object)
	}

	return b
}

// RestoreSpec sets the NonAdminRestore's restore spec.
func (b *NonAdminRestoreBuilder) RestoreSpec(spec nacv1alpha1.NonAdminRestoreSpec) *NonAdminRestoreBuilder {
	b.object.Spec = spec
	return b
}

// Phase sets the NonAdminRestore's phase.
func (b *NonAdminRestoreBuilder) Phase(phase nacv1alpha1.NonAdminPhase) *NonAdminRestoreBuilder {
	b.object.Status.Phase = phase
	return b
}

// VeleroRestore sets the reference to the created Velero restore.
func (b *NonAdminRestoreBuilder) VeleroRestore(restoreName, restoreNamespace string) *NonAdminRestoreBuilder {
	if b.object.Status.VeleroRestore == nil {
		b.object.Status.VeleroRestore = &nacv1alpha1.VeleroRestore{}
	}
	b.object.Status.VeleroRestore.Name = restoreName
	b.object.Status.VeleroRestore.Namespace = restoreNamespace
	return b
}

// Conditions sets the NonAdminRestore's conditions.
func (b *NonAdminRestoreBuilder) Conditions(conditions []metav1.Condition) *NonAdminRestoreBuilder {
	b.object.Status.Conditions = conditions
	return b
}

// WithStatus sets the NonAdminRestore's status.
func (b *NonAdminRestoreBuilder) WithStatus(status nacv1alpha1.NonAdminRestoreStatus) *NonAdminRestoreBuilder {
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
