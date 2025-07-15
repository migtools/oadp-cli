package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	nonadmin "github.com/migtools/oadp-cli/cmd/non-admin"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
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
	nonAdminFactory := nonadmin.NewNonAdminFactory()

	// Add subcommands to the root command
	rootCmd.AddCommand(version.NewCommand(veleroFactory))
	rootCmd.AddCommand(backup.NewCommand(veleroFactory))
	rootCmd.AddCommand(restore.NewCommand(veleroFactory))

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
