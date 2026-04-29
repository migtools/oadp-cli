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
	"strings"
)

// ClientConfig represents the structure of the Velero client configuration file
type ClientConfig struct {
	Namespace    string      `json:"namespace"`
	NonAdmin     interface{} `json:"nonadmin,omitempty"`
	DefaultNABSL string      `json:"default-nabsl,omitempty"`
}

// IsNonAdmin returns true if the nonadmin configuration is enabled.
// Handles both boolean and string representations since
// `oc oadp client config set nonadmin=true` stores the value as a string.
func (c *ClientConfig) IsNonAdmin() bool {
	if c == nil {
		return false
	}
	switch v := c.NonAdmin.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

// GetDefaultNABSL returns the default NonAdminBackupStorageLocation if set.
// Returns empty string if not configured.
func (c *ClientConfig) GetDefaultNABSL() string {
	if c == nil {
		return ""
	}
	return c.DefaultNABSL
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

// getVeleroConfigPath returns the path to the Velero client config file
func getVeleroConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "velero", "config.json"), nil
}

// readConfigMap reads the existing config file as a map, or returns an empty map if it doesn't exist
func readConfigMap(configPath string) (map[string]interface{}, error) {
	configMap := make(map[string]interface{})

	existingData, err := os.ReadFile(configPath)
	if err == nil {
		// File exists, unmarshal it
		if err := json.Unmarshal(existingData, &configMap); err != nil {
			return nil, fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		// Error other than file not existing
		return nil, fmt.Errorf("failed to read existing config: %w", err)
	}
	// If file doesn't exist, configMap remains empty

	return configMap, nil
}

// mergeClientConfig merges the ClientConfig into the config map, updating only managed keys
func mergeClientConfig(configMap map[string]interface{}, config *ClientConfig) {
	configMap["namespace"] = config.Namespace

	if config.NonAdmin != nil {
		configMap["nonadmin"] = config.NonAdmin
	} else {
		delete(configMap, "nonadmin")
	}

	if config.DefaultNABSL != "" {
		configMap["default-nabsl"] = config.DefaultNABSL
	} else {
		delete(configMap, "default-nabsl")
	}
}

// writeConfigMap writes the config map to the specified path
func writeConfigMap(configPath string, configMap map[string]interface{}) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(configMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// WriteVeleroClientConfig writes the client configuration to ~/.config/velero/config.json
// It merges only the keys managed by this CLI (namespace, nonadmin, default-nabsl)
// with the existing config file, preserving any other Velero configuration keys.
func WriteVeleroClientConfig(config *ClientConfig) error {
	configPath, err := getVeleroConfigPath()
	if err != nil {
		return err
	}

	configMap, err := readConfigMap(configPath)
	if err != nil {
		return err
	}

	mergeClientConfig(configMap, config)

	return writeConfigMap(configPath, configMap)
}
