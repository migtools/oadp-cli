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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/migtools/oadp-cli/cmd/nabsl"
	nonadmin "github.com/migtools/oadp-cli/cmd/non-admin"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/version"
)

// isRunningAsPlugin detects if the executable is running as a kubectl plugin
func isRunningAsPlugin() bool {
	return strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-")
}

// getUsagePrefix returns the appropriate command prefix for help messages
func getUsagePrefix() string {
	if isRunningAsPlugin() {
		return "kubectl oadp"
	}
	return "oadp"
}

// NewVeleroRootCommand returns a root command with all Velero CLI subcommands attached.
func NewVeleroRootCommand() *cobra.Command {
	usagePrefix := getUsagePrefix()

	rootCmd := &cobra.Command{
		Use:   "oadp",
		Short: "OADP CLI commands",
		Run: func(cmd *cobra.Command, args []string) {
			// Default action when no subcommand is provided
			if isRunningAsPlugin() {
				fmt.Printf("Welcome to the OADP CLI! Use '%s --help' to see available commands.\n", usagePrefix)
			} else {
				fmt.Println("Welcome to the OADP CLI! Use --help to see available commands.")
			}
		},
	}

	// Create Velero client factory for regular Velero commands
	// This factory is used to create clients for interacting with Velero resources.
	veleroFactory := newVeleroFactory()

	// Create NonAdmin client factory for NonAdminBackup commands
	// This factory uses the current kubeconfig context namespace instead of hardcoded openshift-adp
	nonAdminFactory := NewNonAdminFactory()

	// Create the commands and modify their help text before adding them
	backupCmd := backup.NewCommand(veleroFactory)
	restoreCmd := restore.NewCommand(veleroFactory)
	clientCmd := client.NewCommand()

	// Modify help text to replace "velero" with "oadp"
	updateCommandHelpText(backupCmd, usagePrefix)
	updateCommandHelpText(restoreCmd, usagePrefix)

	// Add subcommands to the root command
	rootCmd.AddCommand(version.NewCommand(veleroFactory))
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(clientCmd)

	// Admin NABSL commands - use Velero factory (admin namespace)
	rootCmd.AddCommand(nabsl.NewNABSLCommand(veleroFactory))

	// Custom subcommands - use NonAdmin factory
	rootCmd.AddCommand(nonadmin.NewNonAdminCommand(nonAdminFactory))

	return rootCmd
}

func Execute() {
	if err := NewVeleroRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
