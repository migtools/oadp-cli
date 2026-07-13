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
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/migtools/oadp-cli/cmd/completion"
	mustgather "github.com/migtools/oadp-cli/cmd/must-gather"
	"github.com/migtools/oadp-cli/cmd/nabsl-request"
	nonadmin "github.com/migtools/oadp-cli/cmd/non-admin"
	"github.com/migtools/oadp-cli/cmd/setup"
	"github.com/spf13/cobra"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	clientcmd "github.com/vmware-tanzu/velero/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backuplocation"
	cliclient "github.com/vmware-tanzu/velero/pkg/cmd/cli/client"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/create"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/debug"
	veldelete "github.com/vmware-tanzu/velero/pkg/cmd/cli/delete"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/describe"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/get"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/schedule"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/snapshotlocation"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/version"

	veleroflag "github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/features"
	"k8s.io/klog/v2"
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
		if ref.Kind == "DataProtectionApplication" && strings.Contains(ref.APIVersion, "oadp.openshift.io") {
			return fmt.Errorf(
				"backup storage location %q is managed by DataProtectionApplication %q.\n"+
					"Direct modifications via 'oc oadp backup-location set' will be overwritten by the DPA reconciler.\n"+
					"To change settings such as '--cacert', update the DataProtectionApplication spec instead",
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

// injectDPAManagedGuard wraps the "set" subcommand of the given backup-location command
// with a PreRunE that rejects modifications to DPA-managed BSLs before the update is attempted.
func injectDPAManagedGuard(bslCmd *cobra.Command, f clientcmd.Factory) {
	for _, sub := range bslCmd.Commands() {
		if strings.HasPrefix(sub.Use, "set ") {
			sub.PreRunE = wrapPreRunE(sub.PreRunE, func(c *cobra.Command, args []string) error {
				if len(args) == 0 {
					return nil
				}
				// Only guard flags the DPA reconciler overwrites; --default persists on DPA-managed BSLs.
				if !c.Flags().Changed("cacert") && !c.Flags().Changed("credential") {
					return nil
				}
				return checkBSLNotDPAManaged(c.Context(), f, args[0])
			})
			return
		}
	}
}

// globalRequestTimeout holds the request timeout value set by --request-timeout flag.
// This is used by the timeoutFactory wrapper to apply dial timeout to all clients.
var (
	globalRequestTimeout time.Duration
	globalTimeoutMu      sync.RWMutex
)

// setGlobalRequestTimeout sets the global request timeout value.
func setGlobalRequestTimeout(timeout time.Duration) {
	globalTimeoutMu.Lock()
	defer globalTimeoutMu.Unlock()
	globalRequestTimeout = timeout
}

// getGlobalRequestTimeout gets the global request timeout value.
func getGlobalRequestTimeout() time.Duration {
	globalTimeoutMu.RLock()
	defer globalTimeoutMu.RUnlock()
	return globalRequestTimeout
}

// timeoutFactory wraps a Velero client.Factory to apply dial timeout to REST configs.
type timeoutFactory struct {
	clientcmd.Factory
}

// applyTimeoutToConfig applies the global request timeout to a REST config.
func applyTimeoutToConfig(config *rest.Config) {
	timeout := getGlobalRequestTimeout()
	if timeout > 0 {
		config.Timeout = timeout

		// Set custom dial function with timeout for TCP connections
		dialer := &net.Dialer{
			Timeout: timeout,
		}
		config.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		}
	}
}

// ClientConfig returns a REST config with dial timeout applied.
func (f *timeoutFactory) ClientConfig() (*rest.Config, error) {
	config, err := f.Factory.ClientConfig()
	if err != nil {
		return nil, err
	}
	applyTimeoutToConfig(config)
	return config, nil
}

// KubeClient returns a Kubernetes client with dial timeout applied.
func (f *timeoutFactory) KubeClient() (kubernetes.Interface, error) {
	config, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

// DynamicClient returns a Kubernetes dynamic client with dial timeout applied.
func (f *timeoutFactory) DynamicClient() (dynamic.Interface, error) {
	config, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(config)
}

// DiscoveryClient returns a Kubernetes discovery client with dial timeout applied.
func (f *timeoutFactory) DiscoveryClient() (discovery.AggregatedDiscoveryInterface, error) {
	config, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(config)
}

// KubebuilderClient returns a controller-runtime client with dial timeout applied.
func (f *timeoutFactory) KubebuilderClient() (kbclient.Client, error) {
	config, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	if err := velerov1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return kbclient.New(config, kbclient.Options{Scheme: scheme})
}

// KubebuilderWatchClient returns a controller-runtime client with watch capability and dial timeout applied.
func (f *timeoutFactory) KubebuilderWatchClient() (kbclient.WithWatch, error) {
	config, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	scheme := runtime.NewScheme()
	if err := velerov1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return kbclient.NewWithWatch(config, kbclient.Options{Scheme: scheme})
}

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
// It also prepends "oc" or "kubectl" based on how the CLI was invoked.
func replaceVeleroCommandWithOADP(text string) string {
	// Use "oc" as the CLI prefix since OADP is primarily used on OpenShift
	cliPrefix := "oc"

	// Replace "velero <command>" patterns with "oc/kubectl oadp <command>"
	result := veleroCommandPattern.ReplaceAllStringFunc(text, func(match string) string {
		// Preserve leading whitespace or backtick
		if strings.HasPrefix(match, " ") || strings.HasPrefix(match, "\t") || strings.HasPrefix(match, "`") || strings.HasPrefix(match, "\n") {
			prefix := match[0:1]
			return prefix + cliPrefix + " " + strings.Replace(match[1:], "velero", "oadp", 1)
		}
		// Start of line - prepend cli prefix
		return cliPrefix + " " + strings.Replace(match, "velero", "oadp", 1)
	})
	return result
}

// replaceVeleroWithOADP recursively replaces all mentions of "velero" with "oadp" in the
// Example field of the given command and all its children. It also wraps the Run and RunE
// functions to replace "velero" with "oadp" in runtime output.
func replaceVeleroWithOADP(cmd *cobra.Command) *cobra.Command {
	// Replace in multiple command fields using context-aware replacement
	cmd.Example = replaceVeleroCommandWithOADP(cmd.Example)

	// Skip wrapping commands that stream output or make long-running requests
	// to allow real-time display without buffering. Check both the command itself
	// and its parent to handle both "backup describe" and "describe backup" patterns.
	isLogsCommand := cmd.Use == "logs" || strings.HasPrefix(cmd.Use, "logs ")
	isDescribeCommand := cmd.Use == "describe" || strings.HasPrefix(cmd.Use, "describe ")
	isDeleteCommand := cmd.Use == "delete" || strings.HasPrefix(cmd.Use, "delete ")

	// Also check if parent command is one we should skip
	if cmd.Parent() != nil {
		parentUse := cmd.Parent().Use
		isLogsCommand = isLogsCommand || parentUse == "logs" || strings.HasPrefix(parentUse, "logs ")
		isDescribeCommand = isDescribeCommand || parentUse == "describe" || strings.HasPrefix(parentUse, "describe ")
		isDeleteCommand = isDeleteCommand || parentUse == "delete" || strings.HasPrefix(parentUse, "delete ")
	}

	shouldSkipWrapper := isLogsCommand || isDescribeCommand || isDeleteCommand

	// Wrap the Run function to replace velero in output
	if cmd.Run != nil && !shouldSkipWrapper {
		originalRun := cmd.Run
		cmd.Run = func(c *cobra.Command, args []string) {
			// Capture stdout temporarily
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the original command
			originalRun(c, args)

			// Restore stdout
			_ = w.Close()
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

	// Wrap the RunE function to replace velero in output
	if cmd.RunE != nil && !shouldSkipWrapper {
		originalRunE := cmd.RunE
		cmd.RunE = func(c *cobra.Command, args []string) error {
			// Capture stdout temporarily
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the original command
			err := originalRunE(c, args)

			// Restore stdout
			_ = w.Close()
			os.Stdout = oldStdout

			// Read captured output and replace velero with oadp (context-aware)
			var buf strings.Builder
			_, copyErr := io.Copy(&buf, r)
			if copyErr != nil {
				fmt.Fprintf(os.Stderr, "WARNING: Error copying output: %v\n", copyErr)
			}
			output := replaceVeleroCommandWithOADP(buf.String())
			fmt.Print(output)

			return err
		}
	}

	// Recursively process all child commands
	for _, child := range cmd.Commands() {
		replaceVeleroWithOADP(child)
	}

	return cmd
}

// renameTimeoutFlag renames --timeout flag to --request-timeout for kubectl consistency.
// This applies to all commands recursively to ensure a consistent CLI experience.
func renameTimeoutFlag(cmd *cobra.Command) {
	// Check if this command has a --timeout flag
	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag != nil {
		// Get the current value and usage
		usage := timeoutFlag.Usage
		defValue := timeoutFlag.DefValue

		// Parse the default value as duration
		var defaultDuration time.Duration
		if defValue != "" && defValue != "0s" {
			if parsed, err := time.ParseDuration(defValue); err == nil {
				defaultDuration = parsed
			}
		}

		// Create a variable to hold the value
		var requestTimeout time.Duration

		// If there's a shorthand, we need to handle it
		shorthand := timeoutFlag.Shorthand

		// Hide the old flag instead of removing it (to avoid breaking existing scripts)
		timeoutFlag.Hidden = true

		// Add the new --request-timeout flag
		if shorthand != "" {
			cmd.Flags().DurationVarP(&requestTimeout, "request-timeout", shorthand, defaultDuration, usage)
		} else {
			cmd.Flags().DurationVar(&requestTimeout, "request-timeout", defaultDuration, usage)
		}

		// Link the flags so setting one affects the other and set global timeout
		cmd.PreRunE = wrapPreRunE(cmd.PreRunE, func(c *cobra.Command, args []string) error {
			// If request-timeout was set, copy its value to the timeout flag and set global
			if c.Flags().Changed("request-timeout") {
				rtFlag := c.Flags().Lookup("request-timeout")
				if rtFlag != nil {
					// Set the global timeout for the timeoutFactory wrapper
					if parsed, err := time.ParseDuration(rtFlag.Value.String()); err == nil {
						setGlobalRequestTimeout(parsed)
					}
					return c.Flags().Set("timeout", rtFlag.Value.String())
				}
			}
			return nil
		})
	}

	// Recursively process all child commands
	for _, child := range cmd.Commands() {
		renameTimeoutFlag(child)
	}
}

// wrapPreRunE wraps an existing PreRunE function with additional logic
func wrapPreRunE(existing func(*cobra.Command, []string) error, additional func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := additional(cmd, args); err != nil {
			return err
		}
		if existing != nil {
			return existing(cmd, args)
		}
		return nil
	}
}

// isNonadminEnabled checks if nonadmin mode is enabled in the VeleroConfig.
// Handles both boolean and string representations since
// `oc oadp client config set nonadmin=true` stores the value as a string.
func isNonadminEnabled(config clientcmd.VeleroConfig) bool {
	val, ok := config["nonadmin"]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

func configureGlobalCommandBehavior(
	config clientcmd.VeleroConfig,
	cmdFeatures veleroflag.StringArray,
	cmdColorized veleroflag.OptionalBool,
) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		features.Enable(config.Features()...)
		features.Enable(cmdFeatures...)

		switch {
		case cmdColorized.Value != nil:
			color.NoColor = !*cmdColorized.Value
		default:
			color.NoColor = !config.Colorized()
		}
	}
}

// NewVeleroRootCommand returns a root command with all Velero CLI subcommands attached.
func NewVeleroRootCommand(baseName string) *cobra.Command {

	config, err := clientcmd.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Error reading config file: %v\n", err)
	}
	if config == nil {
		config = clientcmd.VeleroConfig{}
	}

	// When nonadmin mode is enabled, remove the namespace override so the
	// factory uses the current kubeconfig context namespace instead of an
	// admin namespace like openshift-adp.
	if isNonadminEnabled(config) {
		delete(config, clientcmd.ConfigKeyNamespace)
	} else if config.Namespace() == "" {
		config[clientcmd.ConfigKeyNamespace] = "openshift-adp"
	}

	// Declare cmdFeatures and cmdColorzied here so we can access them in the PreRun hooks
	// without doing a chain of calls into the command's FlagSet
	var cmdFeatures veleroflag.StringArray
	var cmdColorzied veleroflag.OptionalBool

	globalPreRun := configureGlobalCommandBehavior(config, cmdFeatures, cmdColorzied)

	c := &cobra.Command{
		Use:   baseName,
		Short: "OADP CLI commands",
		Run: func(cmd *cobra.Command, args []string) {
			// Default action when no subcommand is provided
			fmt.Println("Welcome to the OADP CLI! Use --help to see available commands.")
		},
		PersistentPreRun: globalPreRun,
	}

	// Create Velero client factory for regular Velero commands
	// This factory is used to create clients for interacting with Velero resources.
	// We wrap it with timeoutFactory to apply dial timeout from --request-timeout flag.
	baseFactory := clientcmd.NewFactory(baseName, config)
	f := &timeoutFactory{Factory: baseFactory}

	// Bind factory flags to enable -n/--namespace flag for admin commands.
	// This allows admin Velero and NABSL-request commands to accept namespace via CLI flag.
	// Nonadmin commands continue using GetCurrentNamespace() for security isolation.
	f.BindFlags(c.PersistentFlags())

	bslCmd := backuplocation.NewCommand(f)
	injectDPAManagedGuard(bslCmd, f)

	c.AddCommand(
		backup.NewCommand(f),
		schedule.NewCommand(f),
		restore.NewCommand(f),
		version.NewCommand(f),
		get.NewCommand(f),
		describe.NewCommand(f),
		create.NewCommand(f),
		veldelete.NewCommand(f),
		cliclient.NewCommand(),
		completion.NewCommand(),
		bslCmd,
		snapshotlocation.NewCommand(f),
		debug.NewCommand(f),
	)

	// Admin NABSL request commands - use Velero factory (admin namespace)
	c.AddCommand(nabsl.NewNABSLRequestCommand(f))

	// Custom subcommands - use NonAdmin factory
	nonadminCmd := nonadmin.NewNonAdminCommand(f)
	nonadmin.ConfigureNamespaceBehavior(nonadminCmd, globalPreRun)
	c.AddCommand(nonadminCmd)

	// Must-gather command - diagnostic tool
	c.AddCommand(mustgather.NewMustGatherCommand(f))

	// Setup command - auto-detect and configure admin vs non-admin mode
	c.AddCommand(setup.NewSetupCommand(f))

	// Apply velero->oadp replacement to all commands recursively
	// Skip nonadmin commands since we have full control over their output
	for _, cmd := range c.Commands() {
		// Don't wrap nonadmin commands - we control them and they already use correct terminology
		// Don't wrap completion - it needs direct stdout access for shell completion generation
		if cmd.Use == "nonadmin" || cmd.Use == "nabsl-request" || cmd.Use == "must-gather" || cmd.Use == "setup" || strings.HasPrefix(cmd.Use, "completion") {
			continue
		}
		replaceVeleroWithOADP(cmd)
	}

	// Rename --timeout flags to --request-timeout for kubectl consistency
	for _, cmd := range c.Commands() {
		renameTimeoutFlag(cmd)
	}

	// When nonadmin mode is enabled, remove all admin commands so only
	// nonadmin, client (for toggling the config), and help are available.
	if isNonadminEnabled(config) {
		allowedCmds := map[string]bool{
			"nonadmin":   true,
			"client":     true,
			"completion": true,
			"setup":      true,
		}
		for _, cmd := range c.Commands() {
			if !allowedCmds[cmd.Use] {
				c.RemoveCommand(cmd)
			}
		}
	}

	// Set custom usage template to show "oc oadp" instead of just "oadp"
	usageTemplate := c.UsageTemplate()
	usageTemplate = strings.ReplaceAll(usageTemplate, "{{.CommandPath}}", "oc {{.CommandPath}}")
	usageTemplate = strings.ReplaceAll(usageTemplate, "{{.UseLine}}", "oc {{.UseLine}}")
	c.SetUsageTemplate(usageTemplate)

	klog.InitFlags(flag.CommandLine)
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return c
}
