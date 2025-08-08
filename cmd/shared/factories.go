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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vmware-tanzu/velero/pkg/client"
)

// ClientConfig represents the structure of the Velero client configuration file
type ClientConfig struct {
	Namespace string `json:"namespace"`
}

// CreateVeleroFactory creates a client factory for Velero operations (admin-scoped)
// that uses the client configuration to determine the namespace.
// Priority order:
// 1. Velero client config (~/.config/velero/config.json)
// 2. Kubeconfig context namespace
// 3. Velero default (usually "velero")
func CreateVeleroFactory() client.Factory {
	cfg := client.VeleroConfig{}

	// Try to read client config to get configured namespace
	if clientConfig, err := ReadVeleroClientConfig(); err == nil {
		if clientConfig.Namespace != "" {
			cfg[client.ConfigKeyNamespace] = clientConfig.Namespace
		}
	}

	return client.NewFactory("oadp-velero-cli", cfg)
}

// CreateNonAdminFactory creates a client factory for NonAdminBackup operations
// that uses the current kubeconfig context namespace instead of hardcoded openshift-adp
func CreateNonAdminFactory() client.Factory {
	// Don't set a default namespace, let it use the kubeconfig context
	cfg := client.VeleroConfig{}
	return client.NewFactory("oadp-velero-cli", cfg)
}

// ReadVeleroClientConfig reads the Velero client configuration from ~/.config/velero/config.json
func ReadVeleroClientConfig() (*ClientConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &ClientConfig{}, nil // Return empty config if file doesn't exist
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse client config: %w", err)
	}

	return &config, nil
}