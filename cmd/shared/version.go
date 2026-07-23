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
	"time"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/vmware-tanzu/velero/pkg/client"
)

const (
	oadpOperatorDisplayName = "OADP Operator"
	oadpMustGatherImageName = "oadp-mustgather-rhel9"
	oadpCSVLookupTimeout    = 30 * time.Second
)

// findOADPCSV finds the OADP operator ClusterServiceVersion across all namespaces
// and returns the first match by DisplayName. Using DisplayName ensures compatibility
// with both the Red Hat and community operator distributions.
func findOADPCSV(f client.Factory) (*operatorsv1alpha1.ClusterServiceVersion, error) {
	kbClient, err := NewClientWithScheme(f, ClientOptions{
		IncludeOLMTypes: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), oadpCSVLookupTimeout)
	defer cancel()

	var csvList operatorsv1alpha1.ClusterServiceVersionList
	if err := kbClient.List(ctx, &csvList); err != nil {
		return nil, fmt.Errorf("failed to list CSVs: %w", err)
	}

	for i := range csvList.Items {
		if csvList.Items[i].Spec.DisplayName == oadpOperatorDisplayName {
			return &csvList.Items[i], nil
		}
	}

	return nil, fmt.Errorf("OADP operator CSV not found in any namespace")
}

// GetMustGatherImage returns the must-gather image pinned in the OADP operator CSV's
// relatedImages. Returns an error if the CSV or the must-gather image entry is not found.
func GetMustGatherImage(f client.Factory) (string, error) {
	csv, err := findOADPCSV(f)
	if err != nil {
		return "", fmt.Errorf("could not detect OADP version: %w", err)
	}

	for _, img := range csv.Spec.RelatedImages {
		if img.Name == oadpMustGatherImageName {
			return img.Image, nil
		}
	}

	return "", fmt.Errorf("must-gather image not found in OADP operator CSV relatedImages")
}
