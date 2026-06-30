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

package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/migtools/oadp-cli/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// SetupOptions holds the options for the setup command
type SetupOptions struct {
	Force bool // Re-run setup even if already configured
}

// BindFlags binds the flags to the command
func (o *SetupOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.Force, "force", false, "Re-run setup even if already configured")
}

// Complete completes the options
func (o *SetupOptions) Complete(args []string, f client.Factory) error {
	return nil
}

// Validate validates the options
func (o *SetupOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	return nil
}

// Run executes the setup command
func (o *SetupOptions) Run(c *cobra.Command, f client.Factory) error {
	c.SilenceUsage = true

	// Check if already configured (unless --force flag set)
	if !o.Force {
		existingConfig, err := shared.ReadVeleroClientConfig()
		if err != nil {
			return fmt.Errorf("failed to read existing config: %w", err)
		}

		if existingConfig.Namespace != "" {
			fmt.Println("OADP CLI is already configured.")
			fmt.Println()
			o.printCurrentConfig(existingConfig)
			fmt.Println()
			fmt.Println("To reconfigure, run: oc oadp setup --force")
			return nil
		}
	}

	config, err := shared.ReadVeleroClientConfig()
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	if err := shared.WriteVeleroClientConfig(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	o.printSetupSuccess()

	return nil
}

// printCurrentConfig prints the current configuration
func (o *SetupOptions) printCurrentConfig(config *shared.ClientConfig) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	fmt.Printf("Namespace: %s\n", config.Namespace)
	fmt.Printf("Configuration file: %s\n", configPath)
}

// printSetupSuccess prints a success message after setup
func (o *SetupOptions) printSetupSuccess() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println()
	fmt.Println("You can now use OADP admin commands:")
	fmt.Println("  oc oadp backup create my-backup")
	fmt.Println("  oc oadp restore create my-restore")
}

// NewSetupCommand creates the setup command
func NewSetupCommand(f client.Factory) *cobra.Command {
	o := &SetupOptions{}

	c := &cobra.Command{
		Use:   "setup",
		Short: "Configure the OADP CLI",
		Long: `Configure the OADP CLI.

Saves the CLI configuration to ~/.config/velero/config.json.

On OADP 1.4, only cluster-admin operations are supported.
Use OADP 1.5 or later for non-admin backup and restore.

Examples:
  # Configure OADP CLI
  oc oadp setup

  # Reconfigure
  oc oadp setup --force`,
		Args: cobra.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(args, f); err != nil {
				return err
			}
			if err := o.Validate(c, args, f); err != nil {
				return err
			}
			return o.Run(c, f)
		},
	}

	o.BindFlags(c.Flags())

	return c
}
