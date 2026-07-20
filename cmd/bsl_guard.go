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
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	clientcmd "github.com/vmware-tanzu/velero/pkg/client"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// bslDPAManagedError returns an error if the BSL has an ownerReference pointing to a
// DataProtectionApplication, indicating the location is managed by the DPA reconciler.
// This is the testable core — it accepts a pre-built client so tests can pass a fake.
func bslDPAManagedError(ctx context.Context, kbClient kbclient.Client, namespace, bslName string) error {
	location := &velerov1.BackupStorageLocation{}
	if err := kbClient.Get(ctx, kbclient.ObjectKey{
		Namespace: namespace,
		Name:      bslName,
	}, location); err != nil {
		return err
	}

	for _, ref := range location.OwnerReferences {
		if ref.Kind == "DataProtectionApplication" && strings.HasPrefix(ref.APIVersion, "oadp.openshift.io/") {
			return fmt.Errorf(
				"backup storage location %q is managed by DataProtectionApplication %q.\n"+
					"Direct modifications via 'oc oadp backup-location set' will be overwritten by the DPA reconciler.\n"+
					"To change these settings, update the DataProtectionApplication spec",
				bslName, ref.Name,
			)
		}
	}
	return nil
}

// checkBSLNotDPAManaged is the factory-aware wrapper used by the CLI command's PreRunE.
func checkBSLNotDPAManaged(ctx context.Context, f clientcmd.Factory, bslName string) error {
	kbClient, err := f.KubebuilderClient()
	if err != nil {
		return err
	}
	return bslDPAManagedError(ctx, kbClient, f.Namespace(), bslName)
}

// onlyDefaultFlagChanged returns true if --default is the only flag that was changed.
// This is the allowlist check: --default is the one flag the DPA reconciler preserves,
// so it is safe to set on DPA-managed BSLs. Any other changed flag risks being overwritten.
func onlyDefaultFlagChanged(c *cobra.Command) bool {
	onlyDefault := true
	c.Flags().Visit(func(f *pflag.Flag) {
		if f.Name != "default" {
			onlyDefault = false
		}
	})
	return onlyDefault
}

// injectDPAManagedGuard wraps the "set" subcommand of the given backup-location command
// with a PreRunE that rejects modifications to DPA-managed BSLs before the update is attempted.
// Uses an allowlist: only --default is permitted on DPA-managed BSLs. Any other changed flag
// triggers the guard, so future upstream flags are automatically protected without code changes.
func injectDPAManagedGuard(bslCmd *cobra.Command, f clientcmd.Factory) {
	for _, sub := range bslCmd.Commands() {
		if strings.HasPrefix(sub.Use, "set ") {
			sub.PreRunE = wrapPreRunE(sub.PreRunE, func(c *cobra.Command, args []string) error {
				if len(args) == 0 {
					return nil
				}
				// Allow if no flags were changed or if --default is the only changed flag.
				if !c.Flags().HasFlags() || onlyDefaultFlagChanged(c) {
					return nil
				}
				return checkBSLNotDPAManaged(c.Context(), f, args[0])
			})
			return
		}
	}
}
