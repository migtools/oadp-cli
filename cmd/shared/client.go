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
	"fmt"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerov2alpha1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v2alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClientOptions holds configuration for creating Kubernetes clients
type ClientOptions struct {
	// IncludeNonAdminTypes adds OADP NonAdmin CRD types to the scheme
	IncludeNonAdminTypes bool
	// IncludeVeleroTypes adds Velero CRD types to the scheme
	IncludeVeleroTypes bool
	// IncludeVeleroV2Alpha1Types adds Velero v2alpha1 CRD types to the scheme (DataUpload/DataDownload)
	IncludeVeleroV2Alpha1Types bool
	// IncludeCoreTypes adds Kubernetes core types to the scheme
	IncludeCoreTypes bool
}

// NewClientWithScheme creates a controller-runtime client with the specified scheme types
func NewClientWithScheme(f client.Factory, opts ClientOptions) (kbclient.WithWatch, error) {
	kbClient, err := f.KubebuilderWatchClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime client: %w", err)
	}

	// Add schemes based on options
	if opts.IncludeNonAdminTypes {
		if err := nacv1alpha1.AddToScheme(kbClient.Scheme()); err != nil {
			return nil, fmt.Errorf("failed to add OADP non-admin types to scheme: %w", err)
		}
	}

	if opts.IncludeVeleroTypes {
		if err := velerov1.AddToScheme(kbClient.Scheme()); err != nil {
			return nil, fmt.Errorf("failed to add Velero types to scheme: %w", err)
		}
	}

	if opts.IncludeVeleroV2Alpha1Types {
		if err := velerov2alpha1.AddToScheme(kbClient.Scheme()); err != nil {
			return nil, fmt.Errorf("failed to add Velero v2alpha1 types to scheme: %w", err)
		}
	}

	if opts.IncludeCoreTypes {
		if err := corev1.AddToScheme(kbClient.Scheme()); err != nil {
			return nil, fmt.Errorf("failed to add Core types to scheme: %w", err)
		}
	}

	return kbClient, nil
}

// NewClientWithFullScheme creates a client with all commonly used scheme types
func NewClientWithFullScheme(f client.Factory) (kbclient.WithWatch, error) {
	return NewClientWithScheme(f, ClientOptions{
		IncludeNonAdminTypes:       true,
		IncludeVeleroTypes:         true,
		IncludeVeleroV2Alpha1Types: true,
		IncludeCoreTypes:           true,
	})
}

// NewSchemeWithTypes creates a new runtime scheme with the specified types
func NewSchemeWithTypes(opts ClientOptions) (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	if opts.IncludeNonAdminTypes {
		if err := nacv1alpha1.AddToScheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to add OADP non-admin types to scheme: %w", err)
		}
	}

	if opts.IncludeVeleroTypes {
		if err := velerov1.AddToScheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to add Velero types to scheme: %w", err)
		}
	}

	if opts.IncludeVeleroV2Alpha1Types {
		if err := velerov2alpha1.AddToScheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to add Velero v2alpha1 types to scheme: %w", err)
		}
	}

	if opts.IncludeCoreTypes {
		if err := corev1.AddToScheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to add Core types to scheme: %w", err)
		}
	}

	return scheme, nil
}

// GetCurrentNamespace gets the current namespace from the kubeconfig context
func GetCurrentNamespace() (string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	namespace, _, err := kubeConfig.Namespace()
	if err != nil {
		return "", fmt.Errorf("failed to get current namespace from kubeconfig: %w", err)
	}

	return namespace, nil
}
