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

package setup

import (
	"fmt"
	"os/exec"
	"strings"
)

// DetectionResult holds the result of detecting user mode
type DetectionResult struct {
	IsAdmin bool
	Error   error
}

// detectUserMode detects whether the user has cluster-admin permissions by checking
// if they can create Velero Backup resources across all namespaces.
// On OADP 1.4, only admin mode is supported; non-admin commands are not available.
func detectUserMode() DetectionResult {
	// Check if user can create Velero Backups across all namespaces (cluster-admin check)
	cmd := exec.Command("oc", "auth", "can-i", "create", "backups.velero.io", "--all-namespaces")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if this is because oc command failed vs permission check
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 typically means "no" for can-i
			if exitErr.ExitCode() == 1 {
				return DetectionResult{IsAdmin: false}
			}
		}
		// Check if output indicates not logged in
		outputStr := string(output)
		if strings.Contains(outputStr, "Unauthorized") || strings.Contains(outputStr, "not logged in") {
			return DetectionResult{Error: fmt.Errorf("not logged in to cluster")}
		}
		// Other errors (oc not found, cluster unreachable, etc.)
		return DetectionResult{Error: fmt.Errorf("failed to check permissions: %w", err)}
	}

	// Parse the output
	result := strings.TrimSpace(string(output))

	// "yes" means user can create backups cluster-wide (admin mode)
	if result == "yes" {
		return DetectionResult{IsAdmin: true}
	}

	// "no" means user does not have cluster-admin access
	return DetectionResult{IsAdmin: false}
}
