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

package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kbfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBSLDPAManagedError(t *testing.T) {
	const ns = "openshift-adp"

	scheme := runtime.NewScheme()
	if err := velerov1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add velero scheme: %v", err)
	}

	tests := []struct {
		name        string
		bsl         *velerov1.BackupStorageLocation
		bslName     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "standalone BSL without ownerReference is allowed",
			bslName: "standalone",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "standalone",
					Namespace: ns,
				},
			},
			wantErr: false,
		},
		{
			name:    "BSL owned by DPA is rejected",
			bslName: "default",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: ns,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "oadp.openshift.io/v1alpha1",
							Kind:       "DataProtectionApplication",
							Name:       "velero",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "managed by DataProtectionApplication",
		},
		{
			name:    "error message names the DPA",
			bslName: "default",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default",
					Namespace: ns,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "oadp.openshift.io/v1alpha1",
							Kind:       "DataProtectionApplication",
							Name:       "my-dpa",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "my-dpa",
		},
		{
			name:    "DataProtectionApplication kind with wrong API group is allowed",
			bslName: "foreign-dpa",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foreign-dpa",
					Namespace: ns,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "other.io/v1",
							Kind:       "DataProtectionApplication",
							Name:       "some-other-dpa",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "BSL with unrelated ownerReference is allowed",
			bslName: "other-owned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-owned",
					Namespace: ns,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "ConfigMap",
							Name:       "some-config",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "non-existent BSL returns not-found error",
			bslName: "does-not-exist",
			bsl:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objs []runtime.Object
			if tt.bsl != nil {
				objs = append(objs, tt.bsl)
			}
			fakeClient := kbfake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				Build()

			err := bslDPAManagedError(context.Background(), fakeClient, ns, tt.bslName)
			if tt.wantErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.errContains != "" && err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestOnlyDefaultFlagChanged(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		want  bool
	}{
		{
			name:  "no flags changed",
			flags: map[string]string{},
			want:  true,
		},
		{
			name:  "only --default changed",
			flags: map[string]string{"default": "true"},
			want:  true,
		},
		{
			name:  "only --cacert changed",
			flags: map[string]string{"cacert": "/path/to/cert"},
			want:  false,
		},
		{
			name:  "only --credential changed",
			flags: map[string]string{"credential": "secret/key"},
			want:  false,
		},
		{
			name:  "--default and --cacert both changed",
			flags: map[string]string{"default": "true", "cacert": "/path/to/cert"},
			want:  false,
		},
		{
			name:  "unknown future flag changed",
			flags: map[string]string{"some-new-flag": "value"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("default", false, "")
			cmd.Flags().String("cacert", "", "")
			cmd.Flags().String("credential", "", "")
			cmd.Flags().String("some-new-flag", "", "")

			for name, val := range tt.flags {
				if err := cmd.Flags().Set(name, val); err != nil {
					t.Fatalf("failed to set flag %q: %v", name, err)
				}
			}

			got := onlyDefaultFlagChanged(cmd)
			if got != tt.want {
				t.Errorf("onlyDefaultFlagChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}
