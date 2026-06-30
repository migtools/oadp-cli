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
	"strings"

	"github.com/migtools/oadp-cli/cmd/shared"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// SetupOptions holds the options for the setup command
type SetupOptions struct {
	Force bool // Re-run detection even if already configured

	// Internal state
	detectionResult DetectionResult
}

// BindFlags binds the flags to the command
func (o *SetupOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.Force, "force", false, "Re-run detection even if already configured")
}

// Complete completes the options
func (o *SetupOptions) Complete(args []string, f client.Factory) error {
	// No setup needed - detection uses oc CLI directly
	return nil
}

// Validate validates the options
func (o *SetupOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	// No validation needed for setup command
	return nil
}

// Run executes the setup command
func (o *SetupOptions) Run(c *cobra.Command, f client.Factory) error {
	fmt.Println("Detecting user permissions...")
	fmt.Println()

	// Silence usage help on errors during Run (we provide clear error messages)
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

	// Run detection
	o.detectionResult = detectUserMode()

	// Handle detection errors
	if o.detectionResult.Error != nil {
		// Provide specific guidance based on error type
		errMsg := o.detectionResult.Error.Error()
		if strings.Contains(errMsg, "not logged in") || strings.Contains(errMsg, "Unauthorized") {
			fmt.Println("Error: Not logged in to cluster")
			fmt.Println()
			fmt.Println("Please log in to your cluster:")
			fmt.Println("  oc login <cluster-url>")
			return fmt.Errorf("not logged in to cluster")
		} else {
			fmt.Printf("Error: %v\n", o.detectionResult.Error)
			fmt.Println()
			fmt.Println("This could mean:")
			fmt.Println("  - Your cluster is not accessible")
			fmt.Println("  - Your kubeconfig is invalid")
			fmt.Println("  - Network connectivity issues")
			return o.detectionResult.Error
		}
	}

	config, err := shared.ReadVeleroClientConfig()
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Write config file (preserves namespace; no non-admin mode on OADP 1.4)
	if err := shared.WriteVeleroClientConfig(config); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Print success message
	o.printSetupSuccess()

	return nil
}

// printCurrentConfig prints the current configuration
func (o *SetupOptions) printCurrentConfig(config *shared.ClientConfig) {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	fmt.Println("Current mode: admin")
	fmt.Printf("Configuration file: %s\n", configPath)
}

// printSetupSuccess prints a success message after setup
func (o *SetupOptions) printSetupSuccess() {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "velero", "config.json")

	if o.detectionResult.IsAdmin {
		fmt.Println("✓ Admin access confirmed")
	} else {
		fmt.Println("⚠ Cluster-admin permissions not detected")
		fmt.Println("  OADP 1.4 CLI requires cluster-admin access; non-admin mode is not supported on this release.")
	}
	fmt.Println()
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
		Short: "Verify cluster-admin access and configure the OADP CLI",
		Long: `Verify cluster-admin access and configure the OADP CLI.

This command checks whether you have cluster-wide admin permissions required
for the OADP 1.4 CLI. Non-admin (self-service) mode is not supported on
OADP 1.4; use OADP 1.5 or later for non-admin backup and restore.

The detection works by checking RBAC permissions: oc auth can-i create backups.velero.io --all-namespaces

Configuration is saved to: ~/.config/velero/config.json

Examples:
  # Auto-detect and configure OADP CLI
  oc oadp setup

  # Re-run detection (reconfigure)
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
