package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/client-go/util/retry"
)

const (
	buildTimeout = 30 * time.Second
	cliTimeout   = 30 * time.Second
)

// buildCLIBinaryFromProject builds the CLI binary from the project root
func buildCLIBinaryFromProject() string {
	// Create temporary directory for the binary
	tmpDir, err := os.MkdirTemp("", "oadp-cli-e2e-*")
	Expect(err).NotTo(HaveOccurred())

	binaryPath := filepath.Join(tmpDir, "oadp-cli-e2e")

	// Build the binary from project root
	projectRoot := getProjectRoot()

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	fmt.Printf("Building CLI binary at: %s\n", binaryPath)
	fmt.Printf("Project root: %s\n", projectRoot)

	err = cmd.Run()
	Expect(err).NotTo(HaveOccurred(), "Failed to build CLI binary: %v\nStderr: %s", err, stderr.String())

	// Verify the binary was created
	_, err = os.Stat(binaryPath)
	Expect(err).NotTo(HaveOccurred(), "Binary not found after build")

	return binaryPath
}

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	// Start from the current directory (tests/e2e)
	dir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	// Look for go.mod in current dir and parent directories
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Check if this is the main project go.mod
			if !strings.Contains(dir, "tests") || filepath.Base(dir) == "oadp-cli-local" {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If not found, try to go up from the e2e directory
	currentDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	// Assume we're in tests/e2e, so go up two levels
	projectRoot := filepath.Join(currentDir, "../..")
	projectRoot, err = filepath.Abs(projectRoot)
	Expect(err).NotTo(HaveOccurred())

	// Verify go.mod exists
	goModPath := filepath.Join(projectRoot, "go.mod")
	_, err = os.Stat(goModPath)
	Expect(err).NotTo(HaveOccurred(), "Could not find project root (go.mod not found)")

	return projectRoot
}

// runCLICommand executes a CLI command and returns the output
func runCLICommand(binaryPath string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Return both stdout and stderr for failed commands
		combined := append(stdout.Bytes(), stderr.Bytes()...)
		return combined, fmt.Errorf("command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return stdout.Bytes(), nil
}

// runCLICommandWithInput executes a CLI command with input and returns the output
func runCLICommandWithInput(binaryPath string, input string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	return stdout.Bytes(), nil
}

// expectCLISuccess runs a CLI command and expects it to succeed
func expectCLISuccess(binaryPath string, args ...string) []byte {
	output, err := runCLICommand(binaryPath, args...)
	Expect(err).NotTo(HaveOccurred(), "CLI command should succeed: %v", args)
	return output
}

// expectCLIFailure runs a CLI command and expects it to fail
func expectCLIFailure(binaryPath string, args ...string) error {
	_, err := runCLICommand(binaryPath, args...)
	Expect(err).To(HaveOccurred(), "CLI command should fail: %v", args)
	return err
}

// retryOnConflict retries a function on conflict errors
func retryOnConflict(fn func() error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, fn)
}

// logOutput logs test output with proper formatting
func logOutput(title string, output []byte) {
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Printf("%s\n", string(output))
	fmt.Printf("=== End %s ===\n\n", title)
}

// cleanupBinary removes the temporary CLI binary
func cleanupBinary(binaryPath string) {
	if binaryPath != "" {
		tmpDir := filepath.Dir(binaryPath)
		os.RemoveAll(tmpDir)
	}
}
