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
	"github.com/vmware-tanzu/velero/pkg/client"
)

// Default namespace for Velero resources
const veleroNamespace = "openshift-adp"

// newVeleroFactory creates a Velero client factory with the configured namespace.
func newVeleroFactory() client.Factory {
	cfg := client.VeleroConfig{
		client.ConfigKeyNamespace: veleroNamespace,
	}
	return client.NewFactory("oadp-velero-cli", cfg)
}
