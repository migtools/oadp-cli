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

package mustgather

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// MustGatherOptions holds the options for the must-gather command
type MustGatherOptions struct {
	DestDir        string
	RequestTimeout time.Duration
	SkipTLS        bool
	Image          string

	// Internal state
	effectiveImage string // Resolved image (after version detection or default)
}

// BindFlags binds the flags to the command
func (o *MustGatherOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.DestDir, "dest-dir", "", "Directory where must-gather output will be stored (defaults to current directory)")
	flags.DurationVar(&o.RequestTimeout, "request-timeout", 0, "Timeout for the gather script (e.g., '1m', '30s')")
	flags.BoolVar(&o.SkipTLS, "skip-tls", false, "Skip TLS verification")
	flags.StringVar(&o.Image, "image", "", "Must-gather image to use (defaults to OADP must-gather image)")
	_ = flags.MarkHidden("image") // Hidden flag for advanced users
}

// Complete completes the options
func (o *MustGatherOptions) Complete(args []string, f client.Factory) error {
	// Determine effective image to use
	// For v1: Use hardcoded default if --image not specified
	if o.Image == "" {
		o.effectiveImage = "registry.redhat.io/oadp/oadp-mustgather-rhel9:v1.5"
		// TODO: Future enhancement - detect version and map to image
		// o.effectiveImage = o.getDefaultImage()
	} else {
		o.effectiveImage = o.Image
	}

	return nil
}

// Validate validates the options
func (o *MustGatherOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	// Verify oc command exists
	if _, err := exec.LookPath("oc"); err != nil {
		return fmt.Errorf("'oc' command not found in PATH. Install OpenShift CLI from: https://mirror.openshift.com/pub/openshift-v4/clients/ocp/")
	}

	// Validate dest-dir if specified
	if o.DestDir != "" {
		if !filepath.IsAbs(o.DestDir) {
			// Convert to absolute path
			absPath, err := filepath.Abs(o.DestDir)
			if err != nil {
				return fmt.Errorf("invalid dest-dir path: %w", err)
			}
			o.DestDir = absPath
		}
	}

	return nil
}

// Run executes the must-gather command
func (o *MustGatherOptions) Run(c *cobra.Command, f client.Factory) error {
	// Build command arguments
	args := []string{"adm", "must-gather", "--image=" + o.effectiveImage}

	// Add dest-dir if specified, otherwise use ./must-gather
	if o.DestDir != "" {
		args = append(args, "--dest-dir="+o.DestDir)
	} else {
		args = append(args, "--dest-dir=./must-gather")
	}

	// Add gather script arguments if any flags are set
	if o.RequestTimeout > 0 || o.SkipTLS {
		args = append(args, "--")
		args = append(args, "/usr/bin/gather")

		if o.RequestTimeout > 0 {
			// Format duration for gather script (e.g., "1m", "30s")
			args = append(args, "--request-timeout", o.RequestTimeout.String())
		}

		if o.SkipTLS {
			args = append(args, "--skip-tls")
		}
	}

	// Execute with real-time output streaming
	cmd := exec.Command("oc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return o.formatError(err)
	}

	return nil
}

// formatError formats error messages for common failure scenarios
func (o *MustGatherOptions) formatError(err error) error {
	// Check if oc not installed (shouldn't happen as we validate, but defensive)
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("'oc' command not found. Install OpenShift CLI from: https://mirror.openshift.com/pub/openshift-v4/clients/ocp/")
	}

	// Check exit code for permission/auth issues
	if exitErr, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("must-gather failed with exit code %d. Check that you're logged in and have appropriate permissions", exitErr.ExitCode())
	}

	return fmt.Errorf("must-gather failed: %w", err)
}

// NewMustGatherCommand creates the must-gather command
func NewMustGatherCommand(f client.Factory) *cobra.Command {
	o := &MustGatherOptions{}

	c := &cobra.Command{
		Use:   "must-gather",
		Short: "Collect diagnostic information for OADP",
		Long: `Collect diagnostic information for OADP installations.

This command runs the OADP must-gather tool to collect logs and cluster state
information needed for troubleshooting and support cases. The diagnostic bundle
will be saved to the specified directory (or current directory by default).

Examples:
  # Collect diagnostics to current directory
  oc oadp must-gather

  # Collect diagnostics to a specific directory
  oc oadp must-gather --dest-dir=/tmp/oadp-diagnostics

  # Collect diagnostics with custom timeout
  oc oadp must-gather --request-timeout=30s

  # Collect diagnostics and skip TLS verification
  oc oadp must-gather --skip-tls

  # Combine multiple options
  oc oadp must-gather --dest-dir=/tmp/output --request-timeout=1m --skip-tls`,
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
