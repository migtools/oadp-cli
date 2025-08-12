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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// ClientConfig represents the structure of Velero's client configuration file
type ClientConfig struct {
	Namespace string `json:"namespace,omitempty"`
	Features  string `json:"features,omitempty"`
}

// readVeleroClientConfig reads the Velero client configuration from ~/.config/velero/config.json
func readVeleroClientConfig() (*ClientConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, return empty config (no error)
		if os.IsNotExist(err) {
			return &ClientConfig{}, nil
		}
		return nil, err
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// newVeleroFactory creates a Velero client factory that respects client configuration.
// This allows admin commands to follow the same namespace precedence as standard Velero:
// 1. Client config (oadp client config set namespace=...)
// 2. Kubeconfig context namespace
// 3. Velero default (usually "velero")
func newVeleroFactory() client.Factory {
	cfg := client.VeleroConfig{}

	// Read client configuration to respect namespace settings
	clientConfig, err := readVeleroClientConfig()
	if err == nil && clientConfig.Namespace != "" {
		// Use namespace from client config if set
		cfg[client.ConfigKeyNamespace] = clientConfig.Namespace
	}
	// If no client config namespace, let Velero use its default resolution:
	// kubeconfig context > velero default

	return client.NewFactory("oadp-velero-cli", cfg)
}

// NewNonAdminFactory creates a client factory for NonAdminBackup operations
// that uses the current kubeconfig context namespace instead of hardcoded openshift-adp
func NewNonAdminFactory() client.Factory {
	// Don't set a default namespace, let it use the kubeconfig context
	cfg := client.VeleroConfig{}
	return client.NewFactory("oadp-nonadmin-cli", cfg)
}

// updateCommandHelpText recursively updates help text in commands and subcommands
func updateCommandHelpText(cmd *cobra.Command, usagePrefix string) {
	// Update examples that contain "velero"
	if strings.Contains(cmd.Example, "velero") {
		cmd.Example = strings.ReplaceAll(cmd.Example, "velero", usagePrefix)
	}

	// Update long description if it contains "velero"
	if strings.Contains(cmd.Long, "velero") {
		cmd.Long = strings.ReplaceAll(cmd.Long, "velero", "oadp")
	}

	// Update short description if it contains "velero"
	if strings.Contains(cmd.Short, "velero") {
		cmd.Short = strings.ReplaceAll(cmd.Short, "velero", "oadp")
	}

	// Recursively update subcommands
	for _, subCmd := range cmd.Commands() {
		updateCommandHelpText(subCmd, usagePrefix)
	}
}
