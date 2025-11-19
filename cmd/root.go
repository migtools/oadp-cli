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
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/migtools/oadp-cli/cmd/nabsl-request"
	nonadmin "github.com/migtools/oadp-cli/cmd/non-admin"
	"github.com/spf13/cobra"
	clientcmd "github.com/vmware-tanzu/velero/pkg/client"

	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backuplocation"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/bug"
	cliclient "github.com/vmware-tanzu/velero/pkg/cmd/cli/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/create"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/datamover"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/debug"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/delete"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/describe"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/get"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/repo"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/repomantenance"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/schedule"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/snapshotlocation"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/version"

	veleroflag "github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/features"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kustomize/cmd/config/completion"
)

// veleroCommandPattern matches "velero" when used as a CLI command.
// It matches "velero" followed by common command patterns, including two-word commands
// like "backup create", "restore get", etc.
var veleroCommandPattern = regexp.MustCompile(`(?m)(?:^|[\s\x60])velero\s+(?:` +
	// Two-word commands: "backup create", "restore get", etc.
	`(?:backup|restore|schedule)\s+(?:create|get|delete|describe|logs|download|patch)` +
	`|` +
	// Single-word commands
	`(?:version|install|uninstall|plugin|snapshot-location|backup-location|restic|repo|client|completion|bug|debug|datamover)` +
	`)`)

// replaceVeleroCommandWithOADP performs context-aware replacement of "velero" with "oadp".
// It only replaces "velero" when it's being used as a CLI command, not when referring to
// the Velero project, server, or components.
func replaceVeleroCommandWithOADP(text string) string {
	// Replace "velero <command>" patterns with "oadp <command>"
	result := veleroCommandPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Preserve leading whitespace or backtick
		if strings.HasPrefix(match, " ") || strings.HasPrefix(match, "\t") || strings.HasPrefix(match, "`") {
			prefix := match[0:1]
			return prefix + strings.Replace(match[1:], "velero", "oadp", 1)
		}
		// Start of line - just replace velero
		return strings.Replace(match, "velero", "oadp", 1)
	})
	return result
}

// replaceVeleroWithOADP recursively replaces all mentions of "velero" with "oadp" in the
// Example field of the given command and all its children. It also wraps the Run function
// to replace "velero" with "oadp" in runtime output.
func replaceVeleroWithOADP(cmd *cobra.Command) *cobra.Command {
	// Replace in multiple command fields using context-aware replacement
	cmd.Example = replaceVeleroCommandWithOADP(cmd.Example)

	// Wrap the Run function to replace velero in output
	if cmd.Run != nil {
		originalRun := cmd.Run
		cmd.Run = func(c *cobra.Command, args []string) {
			// Capture stdout temporarily
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the original command
			originalRun(c, args)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output and replace velero with oadp (context-aware)
			var buf strings.Builder
			_, err := io.Copy(&buf, r)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: Error copying output: %v\n", err)
			}
			output := replaceVeleroCommandWithOADP(buf.String())
			fmt.Print(output)
		}
	}

	// Recursively process all child commands
	for _, child := range cmd.Commands() {
		replaceVeleroWithOADP(child)
	}

	return cmd
}

// NewVeleroRootCommand returns a root command with all Velero CLI subcommands attached.
func NewVeleroRootCommand(baseName string) *cobra.Command {

	config, err := clientcmd.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error reading config file: %v\n", err)
	}

	// Declare cmdFeatures and cmdColorzied here so we can access them in the PreRun hooks
	// without doing a chain of calls into the command's FlagSet
	var cmdFeatures veleroflag.StringArray
	var cmdColorzied veleroflag.OptionalBool

	c := &cobra.Command{
		Use:   baseName,
		Short: "OADP CLI commands",
		Run: func(cmd *cobra.Command, args []string) {
			// Default action when no subcommand is provided
			fmt.Println("Welcome to the OADP CLI! Use --help to see available commands.")
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			features.Enable(config.Features()...)
			features.Enable(cmdFeatures...)

			switch {
			case cmdColorzied.Value != nil:
				color.NoColor = !*cmdColorzied.Value
			default:
				color.NoColor = !config.Colorized()
			}
		},
	}

	// Create Velero client factory for regular Velero commands
	// This factory is used to create clients for interacting with Velero resources.
	f := clientcmd.NewFactory(baseName, config)

	c.AddCommand(
		backup.NewCommand(f),
		schedule.NewCommand(f),
		restore.NewCommand(f),
		version.NewCommand(f),
		get.NewCommand(f),
		describe.NewCommand(f),
		create.NewCommand(f),
		delete.NewCommand(f),
		cliclient.NewCommand(),
		completion.NewCommand(),
		repo.NewCommand(f),
		bug.NewCommand(),
		backuplocation.NewCommand(f),
		snapshotlocation.NewCommand(f),
		debug.NewCommand(f),
		repomantenance.NewCommand(f),
		datamover.NewCommand(f),
	)

	// Admin NABSL request commands - use Velero factory (admin namespace)
	c.AddCommand(nabsl.NewNABSLRequestCommand(f))

	// Custom subcommands - use NonAdmin factory
	c.AddCommand(nonadmin.NewNonAdminCommand(f))

	// Apply velero->oadp replacement to all commands recursively
	for _, cmd := range c.Commands() {
		replaceVeleroWithOADP(cmd)
	}

	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}
