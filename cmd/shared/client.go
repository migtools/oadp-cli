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
	"context"
	"fmt"
	"net"
	"time"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
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
	// IncludeCoreTypes adds Kubernetes core types to the scheme
	IncludeCoreTypes bool
	// Timeout sets a timeout on the REST client configuration.
	// This prevents the client from hanging indefinitely when the cluster is unreachable.
	// If zero, no timeout is set.
	Timeout time.Duration
}

// NewClientWithScheme creates a controller-runtime client with the specified scheme types
func NewClientWithScheme(f client.Factory, opts ClientOptions) (kbclient.WithWatch, error) {
	// If a timeout is specified, we need to create the client manually with the timeout
	// applied to the REST config. Otherwise, use the factory's default method.
	if opts.Timeout > 0 {
		// Get REST config from factory
		restConfig, err := f.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get rest config: %w", err)
		}

		// Set timeout on REST config to prevent hanging when cluster is unreachable
		restConfig.Timeout = opts.Timeout

		// Set a custom dial function with timeout to ensure TCP connection attempts
		// also respect the timeout (the default TCP dial timeout is ~30s)
		dialer := &net.Dialer{
			Timeout: opts.Timeout,
		}
		restConfig.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		}

		// Create scheme with required types
		scheme, err := NewSchemeWithTypes(opts)
		if err != nil {
			return nil, err
		}

		// Create client with the timeout-configured REST config
		kbClient, err := kbclient.NewWithWatch(restConfig, kbclient.Options{Scheme: scheme})
		if err != nil {
			return nil, fmt.Errorf("failed to create controller-runtime client: %w", err)
		}

		return kbClient, nil
	}

	// No timeout specified, use factory's default method
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
		IncludeNonAdminTypes: true,
		IncludeVeleroTypes:   true,
		IncludeCoreTypes:     true,
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
