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
	"testing"
)

// TestNewSchemeWithTypesCaching verifies that schemes are cached by configuration
func TestNewSchemeWithTypesCaching(t *testing.T) {
	opts := ClientOptions{
		IncludeNonAdminTypes: true,
		IncludeVeleroTypes:   true,
	}

	scheme1, err1 := NewSchemeWithTypes(opts)
	scheme2, err2 := NewSchemeWithTypes(opts)

	if err1 != nil || err2 != nil {
		t.Fatal("Unexpected error creating schemes")
	}

	if scheme1 != scheme2 {
		t.Error("Expected same scheme instance for same options")
	}
}

// TestNewSchemeWithTypesCachingDifferentOptions verifies different options yield different schemes
func TestNewSchemeWithTypesCachingDifferentOptions(t *testing.T) {
	opts1 := ClientOptions{
		IncludeNonAdminTypes: true,
		IncludeVeleroTypes:   true,
	}

	opts2 := ClientOptions{
		IncludeNonAdminTypes: true,
		IncludeVeleroTypes:   false,
	}

	scheme1, err1 := NewSchemeWithTypes(opts1)
	scheme2, err2 := NewSchemeWithTypes(opts2)

	if err1 != nil || err2 != nil {
		t.Fatal("Unexpected error creating schemes")
	}

	if scheme1 == scheme2 {
		t.Error("Expected different scheme instances for different options")
	}
}

// TestNewSchemeWithTypes verifies that the function creates schemes with correct types
func TestNewSchemeWithTypes(t *testing.T) {
	tests := []struct {
		name string
		opts ClientOptions
	}{
		{
			name: "NonAdmin types only",
			opts: ClientOptions{
				IncludeNonAdminTypes: true,
			},
		},
		{
			name: "Velero types only",
			opts: ClientOptions{
				IncludeVeleroTypes: true,
			},
		},
		{
			name: "Core types only",
			opts: ClientOptions{
				IncludeCoreTypes: true,
			},
		},
		{
			name: "All types",
			opts: ClientOptions{
				IncludeNonAdminTypes: true,
				IncludeVeleroTypes:   true,
				IncludeCoreTypes:     true,
			},
		},
		{
			name: "No types",
			opts: ClientOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, err := NewSchemeWithTypes(tt.opts)
			if err != nil {
				t.Errorf("NewSchemeWithTypes() unexpected error: %v", err)
			}
			if scheme == nil {
				t.Error("NewSchemeWithTypes() returned nil scheme")
			}
		})
	}
}
