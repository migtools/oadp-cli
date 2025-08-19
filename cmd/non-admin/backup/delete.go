package backup

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

// NewDeleteCommand creates a cobra command for deleting non-admin backups
func NewDeleteCommand(f client.Factory, use string) *cobra.Command {
	o := NewDeleteOptions()

	c := &cobra.Command{
		Use:   use + " [NAME...] | --all",
		Short: "Delete one or more non-admin backups",
		Long:  "Delete one or more non-admin backups by setting the deletebackup field to true",
		Args:  cobra.ArbitraryArgs,
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(args))
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
	All       bool   // Delete all backups in the namespace
	client    kbclient.Client
}

// NewDeleteOptions creates a new DeleteOptions instance
func NewDeleteOptions() *DeleteOptions {
	return &DeleteOptions{}
}

// BindFlags binds the command line flags to the options
func (o *DeleteOptions) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.Confirm, "confirm", false, "Skip confirmation prompt and delete immediately")
	flags.BoolVar(&o.All, "all", false, "Delete all backups in the current namespace")
}

// Complete completes the options by setting up the client and determining the namespace
func (o *DeleteOptions) Complete(args []string, f client.Factory) error {
	// If --all flag is not used, use the provided args
	if !o.All {
		o.Names = args
	}

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

	// If --all flag is used, get all backup names in the namespace
	if o.All {
		var nabList nacv1alpha1.NonAdminBackupList
		err := o.client.List(context.TODO(), &nabList, kbclient.InNamespace(o.Namespace))
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		// Extract backup names
		var allNames []string
		for _, nab := range nabList.Items {
			allNames = append(allNames, nab.Name)
		}
		o.Names = allNames
	}

	return nil
}

// Validate validates the options
func (o *DeleteOptions) Validate(args []string) error {
	if o.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	
	// Check for conflicting options: both args and --all flag
	if o.All && len(args) > 0 {
		return fmt.Errorf("cannot specify both backup names and --all flag")
	}
	
	// Check if neither args nor --all flag provided
	if !o.All && len(args) == 0 {
		return fmt.Errorf("at least one backup name is required, or use --all to delete all backups")
	}
	
	// Special case: if --all is used but no backups found (after Complete)
	if o.All && len(o.Names) == 0 {
		return fmt.Errorf("no backups found in namespace '%s'", o.Namespace)
	}
	
	return nil
}

// Run executes the delete command
func (o *DeleteOptions) Run() error {
	// Show what will be deleted
	fmt.Printf("The following NonAdminBackup(s) will be marked for deletion in namespace '%s':\n", o.Namespace)
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

	// Process each backup
	for _, name := range o.Names {
		err := o.deleteBackup(name)
		if err != nil {
			fmt.Printf("❌ Failed to mark %s for deletion: %v\n", name, err)
			failed = append(failed, name)
		} else {
			fmt.Printf("✓ %s marked for deletion\n", name)
			successful = append(successful, name)
		}
	}

	// Print summary
	fmt.Println()
	if len(successful) > 0 {
		fmt.Printf("Successfully marked %d backup(s) for deletion:\n", len(successful))
		for _, name := range successful {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()
		fmt.Println("ℹ️  Note: The actual backup deletion will be performed asynchronously by the OADP controller.")
		fmt.Println("   This may take some time to complete. You can monitor progress with:")
		fmt.Printf("   kubectl get nonadminbackup -n %s\n", o.Namespace)
	}

	if len(failed) > 0 {
		fmt.Printf("Failed to mark %d backup(s) for deletion:\n", len(failed))
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
		fmt.Printf("Are you sure you want to delete backup '%s'? (y/N): ", o.Names[0])
	} else {
		fmt.Printf("Are you sure you want to delete these %d backups? (y/N): ", len(o.Names))
	}

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// deleteBackup deletes a single backup
func (o *DeleteOptions) deleteBackup(name string) error {
	// Get the NonAdminBackup resource
	nab := &nacv1alpha1.NonAdminBackup{}
	err := o.client.Get(context.TODO(), kbclient.ObjectKey{
		Name:      name,
		Namespace: o.Namespace,
	}, nab)
	if err != nil {
		return o.translateError(name, err)
	}

	// Set the deletebackup field to true
	nab.Spec.DeleteBackup = true

	// Update the resource
	err = o.client.Update(context.TODO(), nab)
	if err != nil {
		return o.translateError(name, err)
	}

	return nil
}

// translateError converts verbose Kubernetes errors into user-friendly messages
func (o *DeleteOptions) translateError(name string, err error) error {
	if errors.IsNotFound(err) {
		return fmt.Errorf("backup '%s' not found", name)
	}

	if errors.IsForbidden(err) {
		return fmt.Errorf("permission denied")
	}

	if errors.IsUnauthorized(err) {
		return fmt.Errorf("authentication required")
	}

	if errors.IsConflict(err) {
		return fmt.Errorf("backup '%s' was modified, please try again", name)
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
