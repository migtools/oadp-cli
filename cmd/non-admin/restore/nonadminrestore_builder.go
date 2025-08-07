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

var nonAdminRestore = ForNonAdminRestore("user-namespace", "restore-1").
	ObjectMeta(
		WithLabels("foo", "bar"),
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
	return &NonAdminRestoreBuilder{
		object: &nacv1alpha1.NonAdminRestore{
			TypeMeta: metav1.TypeMeta{
				APIVersion: nacv1alpha1.GroupVersion.String(),
				Kind:       "NonAdminRestore",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
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

// RestoreSpec sets the NonAdminRestore's spec.
func (b *NonAdminRestoreBuilder) RestoreSpec(spec nacv1alpha1.NonAdminRestoreSpec) *NonAdminRestoreBuilder {
	b.object.Spec = spec
	return b
}

// Phase sets the NonAdminRestore's status phase.
func (b *NonAdminRestoreBuilder) Phase(phase nacv1alpha1.NonAdminPhase) *NonAdminRestoreBuilder {
	b.object.Status.Phase = phase
	return b
}

// ObjectMetaOpt is a functional option for setting fields on a NonAdminRestore's ObjectMeta.
type ObjectMetaOpt func(*nacv1alpha1.NonAdminRestore)

// WithLabels sets the NonAdminRestore's labels.
func WithLabels(labels ...string) ObjectMetaOpt {
	return func(obj *nacv1alpha1.NonAdminRestore) {
		if len(labels)%2 != 0 {
			panic("labels must be specified in pairs")
		}

		if obj.Labels == nil {
			obj.Labels = make(map[string]string)
		}

		for i := 0; i < len(labels); i += 2 {
			obj.Labels[labels[i]] = labels[i+1]
		}
	}
}

// WithLabelsMap sets the NonAdminRestore's labels.
func WithLabelsMap(labels map[string]string) ObjectMetaOpt {
	return func(obj *nacv1alpha1.NonAdminRestore) {
		if obj.Labels == nil {
			obj.Labels = make(map[string]string)
		}

		for k, v := range labels {
			obj.Labels[k] = v
		}
	}
}

// WithAnnotations sets the NonAdminRestore's annotations.
func WithAnnotations(annotations ...string) ObjectMetaOpt {
	return func(obj *nacv1alpha1.NonAdminRestore) {
		if len(annotations)%2 != 0 {
			panic("annotations must be specified in pairs")
		}

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}

		for i := 0; i < len(annotations); i += 2 {
			obj.Annotations[annotations[i]] = annotations[i+1]
		}
	}
}

// WithAnnotationsMap sets the NonAdminRestore's annotations.
func WithAnnotationsMap(annotations map[string]string) ObjectMetaOpt {
	return func(obj *nacv1alpha1.NonAdminRestore) {
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}

		for k, v := range annotations {
			obj.Annotations[k] = v
		}
	}
}
