package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Ensure we can find the project root
	if err := findProjectRoot(); err != nil {
		panic("Could not find project root: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Clean up any global test artifacts here if needed

	os.Exit(code)
}

// findProjectRoot ensures we can locate the project root from the tests directory
func findProjectRoot() error {
	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Look for go.mod in current dir and parent directories
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Check if this is the main project go.mod (not the tests go.mod)
			if filepath.Base(dir) != "tests" {
				return nil // Found main project go.mod
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return os.ErrNotExist
}
