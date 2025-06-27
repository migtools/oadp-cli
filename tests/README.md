# OADP CLI Tests

This directory contains organized integration tests for the OADP CLI.

## Test Structure

### Core Test Files

- **`help_test.go`** - 🏆 **Help command tests** (baseline functionality)
  - Tests all `--help` and `-h` commands across all paths
  - Verifies expected help text appears
  - Core functionality that must always work

- **`build_test.go`** - Binary building and execution tests
  - Tests that binary can be built successfully
  - Smoke tests for basic command execution
  - Version and basic functionality tests

### Supporting Files

- **`common.go`** - Shared test utilities and helper functions
- **`main_test.go`** - Test setup and teardown
- **`go.mod`** - Module configuration

## Running Tests

### Run All Tests
```bash
# Standard Go command - runs all tests in project
go test ./...

# Run just the CLI tests
go test -v -timeout 60s ./tests

# From tests directory
cd tests && go test -v -timeout 60s
```

### Run Specific Test Categories
```bash
# From project root
go test -v ./tests -run TestCLIHelp      # Help command tests
go test -v ./tests -run TestCLIBinary    # Build and smoke tests

# From tests directory
go test -v -run TestCLIHelp      # Help command tests
go test -v -run TestCLIBinary    # Build and smoke tests
```

### Quick Test Run
```bash
# Standard Go pattern - finds all tests
go test ./...

# Just CLI tests
go test ./tests

# With output
go test -v ./tests
```

## Test Categories

### 🏆 Help Tests (`help_test.go`)
These are the **core functionality tests** - verify all help commands work:
- `TestCLIHelpCommands` - All help commands work across all paths
- `TestCLIHelpFlags` - Both `--help` and `-h` work consistently

### 🔧 Build Tests (`build_test.go`)
Tests that verify the binary itself works:
- `TestCLIBinaryBuild` - Binary builds and executes
- `TestCLIBinaryVersion` - Version command works
- `TestCLIBinarySmoke` - Basic commands don't crash

## Test Configuration

- **Build timeout**: 30 seconds
- **Test timeout**: 10 seconds per test
- **Binary name**: `oadp-test` (temporary)
- **Tests local code**: Whatever is currently on disk (including uncommitted changes)

## Adding New Tests

### Adding to Existing Categories
Add new test cases to the appropriate file:

```go
// In help_test.go for new help commands
{
    name: "new command help",
    args: []string{"new", "command", "--help"},
    expectContains: []string{
        "Description of new command",
    },
},
```

### Creating New Test Categories
1. Create a new `*_test.go` file
2. Import the `tests` package
3. Use helper functions from `common.go`
4. Follow existing patterns

Example:
```go
package tests

import "testing"

func TestCLINewFeature(t *testing.T) {
    binaryPath := buildCLIBinary(t)
    defer cleanup(t, binaryPath)
    
    // Your test logic here
}
```

## Troubleshooting

### Build Failures
- Check that all dependencies in `../go.mod` are available
- Verify you're running from the correct directory
- Check that parent directory has valid Go module

### Test Failures
- Check expected strings match actual CLI output
- Use `-v` flag to see detailed test output
- Look at the full command output in logs

### Common Commands
```bash
# Run with verbose output
go test -v ./tests

# Run with race detection
go test -race ./tests

# Run specific test function
go test -v ./tests -run TestCLIHelpCommands

# Run tests with coverage
go test -cover ./tests
``` 
