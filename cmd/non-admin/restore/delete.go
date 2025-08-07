package restore

/*
Copyright The Velero Contributors.

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

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
)

// NewDeleteCommand creates a cobra command for deleting non-admin restores
func NewDeleteCommand(f client.Factory, use string) *cobra.Command {
	o := NewDeleteOptions()

	c := &cobra.Command{
		Use:   use + " NAME [NAME...]",
		Short: "Delete one or more non-admin restores",
		Long:  "Delete one or more non-admin restores permanently from the cluster",
		Args:  cobra.MinimumNArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate())
			cmd.CheckError(o.Run())
		},
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

// DeleteOptions holds the options for the delete command
type DeleteOptions struct {
	Names     []string
	Namespace string // Internal field - automatically determined from kubectl context
	Confirm   bool   // Skip confirmation prompt
	client    kbclient.Client
}

// NewDeleteOptions creates a new DeleteOptions instance
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{}
}

// BindFlags binds the command line flags to the options
func (o *DeleteOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.Confirm, "confirm", false, "Skip confirmation prompt and delete immediately")
}

// Complete completes the options by setting up the client and determining the namespace
func (o *DeleteOptions) Complete(args []string, f client.Factory) error {
	o.Names = args

	// Create client with NonAdmin scheme
	kbClient, err := shared.NewClientWithScheme(f, shared.ClientOptions{
		IncludeNonAdminTypes: true,
	})
	if err != nil {
		return err
	}

	o.client = kbClient

	// Always use the current namespace from kubectl context
	currentNS, err := shared.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}
	o.Namespace = currentNS

	return nil
}

// Validate validates the options
func (o *DeleteOptions) Validate() error {
	if len(o.Names) == 0 {
		return fmt.Errorf("at least one restore name is required")
	}
	if o.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	return nil
}

// Run executes the delete command
func (o *DeleteOptions) Run() error {
	// Show what will be deleted
	fmt.Printf("The following NonAdminRestore(s) will be permanently deleted from namespace '%s':\n", o.Namespace)
	for _, name := range o.Names {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Prompt for confirmation unless --confirm flag is used
	if !o.Confirm {
		confirmed, err := o.promptForConfirmation()
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Track results
	var successful []string
	var failed []string

	// Process each restore
	for _, name := range o.Names {
		err := o.deleteRestore(name)
		if err != nil {
			fmt.Printf("❌ Failed to delete %s: %v\n", name, err)
			failed = append(failed, name)
		} else {
			fmt.Printf("✓ %s deleted\n", name)
			successful = append(successful, name)
		}
	}

	// Print summary
	fmt.Println()
	if len(successful) > 0 {
		fmt.Printf("Successfully deleted %d restore(s):\n", len(successful))
		for _, name := range successful {
			fmt.Printf("  - %s\n", name)
		}
	}

	if len(failed) > 0 {
		fmt.Printf("Failed to delete %d restore(s):\n", len(failed))
		for _, name := range failed {
			fmt.Printf("  - %s\n", name)
		}
		return fmt.Errorf("some operations failed")
	}

	return nil
}

// promptForConfirmation prompts the user for confirmation
func (o *DeleteOptions) promptForConfirmation() (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	if len(o.Names) == 1 {
		fmt.Printf("Are you sure you want to delete restore '%s'? (y/N): ", o.Names[0])
	} else {
		fmt.Printf("Are you sure you want to delete these %d restores? (y/N): ", len(o.Names))
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// deleteRestore deletes a single restore
func (o *DeleteOptions) deleteRestore(name string) error {
	// Get the NonAdminRestore resource
	nar := &nacv1alpha1.NonAdminRestore{}
	err := o.client.Get(context.TODO(), kbclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}, nar)
	if err != nil {
		return o.translateError(name, err)
	}

	// Delete the resource directly
	err = o.client.Delete(context.TODO(), nar)
	if err != nil {
		return o.translateError(name, err)
	}

	return nil
}

// translateError converts verbose Kubernetes errors into user-friendly messages
func (o *DeleteOptions) translateError(name string, err error) error {
	if errors.IsNotFound(err) {
		return fmt.Errorf("restore '%s' not found", name)
	}

	if errors.IsForbidden(err) {
		return fmt.Errorf("permission denied")
	}

	if errors.IsUnauthorized(err) {
		return fmt.Errorf("authentication required")
	}

	if errors.IsConflict(err) {
		return fmt.Errorf("restore '%s' was modified, please try again", name)
	}

	if errors.IsTimeout(err) {
		return fmt.Errorf("request timed out")
	}

	if errors.IsServerTimeout(err) {
		return fmt.Errorf("server timeout")
	}

	if errors.IsServiceUnavailable(err) {
		return fmt.Errorf("service unavailable")
	}

	// Check for common connection issues
	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") {
		return fmt.Errorf("cannot connect to cluster")
	}

	if strings.Contains(errStr, "no such host") {
		return fmt.Errorf("cannot reach cluster")
	}

	// For any other error, provide a generic message
	return fmt.Errorf("operation failed")
}
