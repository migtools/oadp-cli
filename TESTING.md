# OADP CLI Testing Guide

This document describes the decentralized testing architecture for the OADP CLI.

## Architecture Overview

Tests are organized following Go best practices - they live next to the code they test:

```
├── cmd/
│   ├── root_test.go                    # Root command tests
│   ├── nabsl/
│   │   ├── nabsl.go
│   │   └── nabsl_test.go              # NABSL command tests
│   └── non-admin/
│       ├── nonadmin_test.go           # Non-admin command tests
│       └── bsl/
│           ├── bsl.go
│           └── bsl_test.go            # BSL command tests
├── internal/
│   └── testutil/
│       └── testutil.go                # Shared test utilities
└── integration_test.go                # Integration tests
```

## Test Types

### 1. Unit Tests
Located next to the source code they test:

- **`cmd/root_test.go`**: Tests root command functionality, help text, basic structure
- **`cmd/nabsl/nabsl_test.go`**: Tests NABSL commands (approve, reject, get, describe)
- **`cmd/non-admin/nonadmin_test.go`**: Tests non-admin command structure
- **`cmd/non-admin/bsl/bsl_test.go`**: Tests BSL creation, credential handling

### 2. Integration Tests
Located at the project root in `integration_test.go`:

- **Binary Build**: Tests that the CLI binary builds successfully
- **Makefile Integration**: Tests installation options and build system
- **Client Config**: Tests end-to-end client configuration workflow
- **Command Architecture**: Tests overall command structure and relationships

### 3. Shared Utilities
Located in `internal/testutil/`:

- **`BuildCLIBinary()`**: Builds test binary with proper cleanup
- **`RunCommand()`**: Executes CLI commands with timeout and logging
- **`TestHelpCommand()`**: Validates help text contains expected content
- **`SetupTempHome()`**: Creates isolated test environment for client config

## Running Tests

### All Tests
```bash
make test
```

### Unit Tests Only
```bash
make test-unit
```

### Integration Tests Only
```bash
make test-integration
```

### Specific Package
```bash
# Test specific command
go test ./cmd/nabsl -v

# Test with coverage
go test ./cmd/... -cover
```

## Test Coverage

### Unit Tests Verify:
- ✅ Command help text and structure
- ✅ Flag definitions and validation
- ✅ Subcommand availability
- ✅ Help flag consistency (`--help` and `-h`)
- ✅ Command architecture changes

### Integration Tests Verify:
- ✅ Binary builds successfully
- ✅ Makefile installation options work
- ✅ Client configuration end-to-end
- ✅ Cross-command functionality
- ✅ Overall system behavior

## Benefits of Decentralized Testing

### 1. **Locality**
- Tests live next to the code they test
- Easy to find and maintain
- Clear ownership and responsibility

### 2. **Focused Scope**
- Each test file has a narrow, clear scope
- Faster test execution for specific areas
- Better isolation of test failures

### 3. **Parallel Execution**
- Tests can run in parallel across packages
- Better CI/CD performance
- Independent test environments

### 4. **Maintainability**
- When code changes, related tests are immediately visible
- Easier to keep tests in sync with code
- Reduced cognitive overhead

## Adding New Tests

### For New Commands
1. Create `*_test.go` file in the same package as your command
2. Import `"github.com/migtools/oadp-cli/internal/testutil"`
3. Follow existing patterns for help text validation

### For Integration Scenarios
1. Add tests to `integration_test.go`
2. Use `testutil.BuildCLIBinary()` for binary-based tests
3. Focus on cross-package functionality

### Example Test Structure
```go
func TestNewCommand(t *testing.T) {
    binaryPath := testutil.BuildCLIBinary(t)
    
    tests := []struct {
        name           string
        args           []string
        expectContains []string
    }{
        {
            name: "command help",
            args: []string{"newcommand", "--help"},
            expectContains: []string{"expected text"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            testutil.TestHelpCommand(t, binaryPath, tt.args, tt.expectContains)
        })
    }
}
```

## Best Practices

1. **Use testutil helpers** for common operations
2. **Test help text** to verify command structure
3. **Use table-driven tests** for multiple scenarios
4. **Keep tests focused** on single responsibilities
5. **Mock external dependencies** when possible
6. **Use descriptive test names** that explain what's being tested

This testing architecture ensures comprehensive coverage while maintaining clarity and ease of maintenance.

## Installation Features

The `make install` command includes intelligent namespace detection that automatically discovers where OADP is deployed in your cluster:

### Automatic Detection Process

1. **OADP Controller Detection**: Looks for `openshift-adp-controller-manager` deployment
2. **DPA Resource Detection**: Searches for `DataProtectionApplication` custom resources  
3. **Velero Fallback**: Falls back to looking for `velero` deployment
4. **Interactive Prompt**: If no resources found, prompts for manual input

### Installation Modes

```bash
# Smart detection + interactive prompt (default)
make install

# Skip detection, use default namespace
make install ASSUME_DEFAULT=true

# Skip detection, use custom namespace  
make install VELERO_NAMESPACE=my-oadp-namespace
```

This intelligent detection eliminates the guesswork of finding the correct OADP namespace in your cluster.