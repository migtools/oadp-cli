package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/version"
)

// NewVeleroRootCommand returns a root command with all Velero CLI subcommands attached.
func NewVeleroRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "velero",
		Short: "Velero CLI commands",
		Run: func(cmd *cobra.Command, args []string) {
			// Default action when no subcommand is provided
			fmt.Println("Welcome to the custom Velero CLI! Use --help to see available commands.")
		},
	}

	// Create Velero client factory
	// This factory is used to create clients for interacting with Velero resources.
	factory := newVeleroFactory()

	// Add subcommands to the root command
	rootCmd.AddCommand(version.NewCommand(factory))

	return rootCmd
}

func Execute() {
	if err := NewVeleroRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
