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

package nonadmin

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ConfigureNamespaceBehavior hides the inherited --namespace flag from nonadmin
// help output and rejects runtime use of -n/--namespace. Nonadmin operations are
// scoped to the user's current context namespace for security.
func ConfigureNamespaceBehavior(
	cmd *cobra.Command,
	runGlobalSetup func(cmd *cobra.Command, args []string),
) {
	hideNamespaceFlagFromCommand(cmd)

	existingPersistentPreRunE := cmd.PersistentPreRunE
	existingPersistentPreRun := cmd.PersistentPreRun

	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		if c.Flags().Changed("namespace") {
			return fmt.Errorf("-n/--namespace is not supported for nonadmin commands; namespace is determined by your current context")
		}
		if runGlobalSetup != nil {
			runGlobalSetup(c, args)
		}
		if existingPersistentPreRunE != nil {
			return existingPersistentPreRunE(c, args)
		}
		if existingPersistentPreRun != nil {
			existingPersistentPreRun(c, args)
		}
		return nil
	}
}

func hideNamespaceFlagFromCommand(cmd *cobra.Command) {
	originalHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		withHiddenNamespaceFlag(c, func() {
			originalHelpFunc(c, args)
		})
	})

	originalUsageFunc := cmd.UsageFunc()
	cmd.SetUsageFunc(func(c *cobra.Command) error {
		var err error
		withHiddenNamespaceFlag(c, func() {
			err = originalUsageFunc(c)
		})
		return err
	})

	for _, subCmd := range cmd.Commands() {
		hideNamespaceFlagFromCommand(subCmd)
	}
}

func withHiddenNamespaceFlag(cmd *cobra.Command, fn func()) {
	if flag := cmd.InheritedFlags().Lookup("namespace"); flag != nil {
		originalHidden := flag.Hidden
		flag.Hidden = true
		defer func() { flag.Hidden = originalHidden }()
	}
	fn()
}
