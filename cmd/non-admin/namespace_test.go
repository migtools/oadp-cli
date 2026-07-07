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
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/migtools/oadp-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func TestNonAdminNamespaceFlagBehavior(t *testing.T) {
	binaryPath := testutil.BuildCLIBinary(t)

	t.Run("help hides namespace flag", func(t *testing.T) {
		output, _ := testutil.RunCommand(t, binaryPath, "nonadmin", "backup", "get", "--help")
		if strings.Contains(output, "-n, --namespace") || strings.Contains(output, "--namespace string") {
			t.Errorf("Expected help output not to contain the namespace flag\nFull output:\n%s", output)
		}
	})

	t.Run("calls global persistent setup", func(t *testing.T) {
		var globalSetupCalled bool
		globalSetup := func(cmd *cobra.Command, args []string) {
			globalSetupCalled = true
		}

		cmd := &cobra.Command{Use: "nonadmin"}
		ConfigureNamespaceBehavior(cmd, globalSetup)

		if err := cmd.PersistentPreRunE(cmd, []string{}); err != nil {
			t.Fatalf("PersistentPreRunE returned error: %v", err)
		}
		if !globalSetupCalled {
			t.Error("Expected global persistent setup to run for nonadmin command")
		}
	})

	t.Run("rejects namespace flag at runtime", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testutil.TestTimeout)
		defer cancel()

		output, err := exec.CommandContext(ctx, binaryPath, "nonadmin", "backup", "get", "-n", "other-namespace").CombinedOutput()
		if err == nil {
			t.Fatalf("Expected error when -n/--namespace is provided, got output:\n%s", string(output))
		}
		expected := "-n/--namespace is not supported for nonadmin commands; namespace is determined by your current context"
		if !strings.Contains(string(output), expected) {
			t.Errorf("Expected output to contain %q, got:\n%s", expected, string(output))
		}
	})
}
