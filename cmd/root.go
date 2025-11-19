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

// replaceVeleroWithOADP recursively replaces all mentions of "velero" with "oadp" in the
// Example field of the given command and all its children. It also wraps the Run function
// to replace "velero" with "oadp" in runtime output.
func replaceVeleroWithOADP(cmd *cobra.Command) *cobra.Command {
	// Replace in multiple command fields
	cmd.Example = strings.ReplaceAll(cmd.Example, "velero", "oadp")

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

			// Read captured output and replace velero with oadp
			var buf strings.Builder
			io.Copy(&buf, r)
			output := strings.ReplaceAll(buf.String(), "velero", "oadp")
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
		replaceVeleroWithOADP(backup.NewCommand(f)),
		replaceVeleroWithOADP(schedule.NewCommand(f)),
		replaceVeleroWithOADP(restore.NewCommand(f)),
		replaceVeleroWithOADP(version.NewCommand(f)),
		replaceVeleroWithOADP(get.NewCommand(f)),
		replaceVeleroWithOADP(describe.NewCommand(f)),
		replaceVeleroWithOADP(create.NewCommand(f)),
		replaceVeleroWithOADP(delete.NewCommand(f)),
		replaceVeleroWithOADP(cliclient.NewCommand()),
		replaceVeleroWithOADP(completion.NewCommand()),
		replaceVeleroWithOADP(repo.NewCommand(f)),
		replaceVeleroWithOADP(bug.NewCommand()),
		replaceVeleroWithOADP(backuplocation.NewCommand(f)),
		replaceVeleroWithOADP(snapshotlocation.NewCommand(f)),
		replaceVeleroWithOADP(debug.NewCommand(f)),
		replaceVeleroWithOADP(repomantenance.NewCommand(f)),
		replaceVeleroWithOADP(datamover.NewCommand(f)),
	)

	// Admin NABSL request commands - use Velero factory (admin namespace)
	c.AddCommand(nabsl.NewNABSLRequestCommand(f))

	// Custom subcommands - use NonAdmin factory
	c.AddCommand(nonadmin.NewNonAdminCommand(f))

	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}
