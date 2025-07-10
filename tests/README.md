# OADP NonAdmin CLI Test Suite

This directory contains comprehensive tests for the OADP CLI **nonadmin** commands, specifically focusing on testing the nonadmin backup commands with mocked Kubernetes clients.

## Overview

The test suite is designed to verify that the **nonadmin CLI commands** create and manipulate **NonAdminBackup** Kubernetes resources with the correct specifications according to the CLI arguments. The tests use mocked Kubernetes clients to avoid requiring a real cluster connection while still validating the complete behavior of the nonadmin CLI commands.

## Test Structure

### Core Test Files

1. **`mock_client.go`** - Mock Kubernetes client implementation and test utilities for NonAdminBackup resources
2. **`nonadmin_backup_test.go`** - Unit tests for nonadmin backup creation and deletion with mock clients
3. **`cli_integration_test.go`** - Integration tests for nonadmin CLI commands (requires cluster connection)
4. **`common.go`** - Common test utilities and helpers
5. **`build_test.go`** - CLI binary build and smoke tests
6. **`help_test.go`** - Help command validation tests for nonadmin commands

### Mock Client Implementation

The `MockK8sClient` provides a complete mock implementation of the Kubernetes client interface:

- **Create/Update/Delete operations** - Tracks all operations and stores objects in memory
- **Get operations** - Retrieves objects from the mock storage
- **Object tracking** - Maintains lists of created, updated, and deleted objects
- **Scheme support** - Includes proper scheme registration for custom resources

### Test Utilities

The test suite includes several utility functions for creating and validating **NonAdminBackup** objects:

- `CreateTestBackup()` - Creates test NonAdminBackup objects with functional options
- `ValidateBackupSpec()` - Validates NonAdminBackup specifications
- `CompareBackupSpecs()` - Compares NonAdminBackup specifications for equality
- Functional options for configuring NonAdminBackup resources (labels, annotations, resources, TTL, etc.)

## Test Categories

### 1. Mock Client Tests (Recommended)

These tests use the mock Kubernetes client and don't require a real cluster:

```bash
go test -v -run "TestNonAdminBackupCreateWithMockClient|TestNonAdminBackupDeleteWithMockClient|TestBackupSpecValidation|TestBackupOptions|TestBackupSpecGeneration|TestBackupDeleteOperation|TestBackupSpecComparison"
```

**Test Coverage:**
- ✅ NonAdminBackup creation with various configurations
- ✅ NonAdminBackup deletion operations (setting delete flag)
- ✅ NonAdminBackup specification validation
- ✅ NonAdminBackup option functions
- ✅ NonAdminBackup spec generation and comparison
- ✅ Error handling for non-existent NonAdminBackup resources

### 2. CLI Integration Tests (Requires Cluster)

These tests run the actual nonadmin CLI binary and require a Kubernetes cluster:

```bash
go test -v -run "TestCLIBackupCreate|TestCLIBackupDelete"
```

**Note:** These tests currently fail because they require a real cluster connection. They demonstrate the intended behavior but need a proper test environment to run successfully.

### 3. Build and Smoke Tests

These tests verify the CLI binary builds correctly and basic functionality:

```bash
go test -v -run "TestCLIBinary|TestCLIHelp"
```

## Test Scenarios Covered

### NonAdminBackup Creation Tests

1. **Basic NonAdminBackup creation** - Tests minimal NonAdminBackup creation with default settings
2. **Custom labels and annotations** - Verifies labels and annotations are properly applied to NonAdminBackup resources
3. **Specific resource inclusion** - Tests NonAdminBackup with specific resource types
4. **TTL configuration** - Validates TTL settings for NonAdminBackup
5. **Snapshot volume settings** - Tests snapshot volume configuration for NonAdminBackup
6. **Error handling** - Tests validation errors for missing required fields

### NonAdminBackup Deletion Tests

1. **Existing NonAdminBackup deletion** - Tests successful marking of existing NonAdminBackup resources for deletion
2. **Non-existent NonAdminBackup handling** - Verifies proper error handling for missing NonAdminBackup resources
3. **Multiple NonAdminBackup deletion** - Tests marking multiple NonAdminBackup resources for deletion
4. **Delete flag verification** - Ensures the delete flag is properly set on NonAdminBackup resources

### NonAdminBackup Specification Validation Tests

1. **Valid NonAdminBackup specs** - Ensures valid NonAdminBackup configurations pass validation
2. **Invalid NonAdminBackup specs** - Tests validation failures for:
   - Missing NonAdminBackup name
   - Missing namespace
   - Missing NonAdminBackup specification

### NonAdminBackup Comparison Tests

1. **Identical NonAdminBackup resources** - Verifies identical NonAdminBackup specs are considered equal
2. **Different NonAdminBackup configurations** - Tests that different NonAdminBackup specs are properly detected:
   - Different names
   - Different namespaces
   - Different labels
   - Different resources

## Running Tests

### Prerequisites

1. Go 1.24 or later
2. Required dependencies (automatically managed by go.mod):
   - `github.com/stretchr/testify` for assertions
   - `k8s.io/client-go` for Kubernetes client interfaces
   - `sigs.k8s.io/controller-runtime` for fake client

### Running All Tests

```bash
cd tests
go test -v ./...
```

### Running Specific Test Categories

```bash
# Mock client tests only (recommended for development)
go test -v -run "TestNonAdminBackupCreateWithMockClient|TestNonAdminBackupDeleteWithMockClient|TestBackupSpecValidation|TestBackupOptions|TestBackupSpecGeneration|TestBackupDeleteOperation|TestBackupSpecComparison"

# Build and smoke tests
go test -v -run "TestCLIBinary|TestCLIHelp"

# All tests except integration tests
go test -v -run "TestCLIBackupCreate|TestCLIBackupDelete" -skip
```

### Test Output

The tests provide detailed output including:
- Command execution details
- Expected vs actual results
- Error messages and stack traces
- Resource specification comparisons

## Test Data and Examples

### Example NonAdminBackup Creation

```go
// Create a basic NonAdminBackup
backup := CreateTestBackup("test-backup", "test-namespace", WithResources([]string{"*"}))

// Create a NonAdminBackup with custom configuration
backup := CreateTestBackup("labeled-backup", "test-namespace",
    WithLabels(map[string]string{"app": "test", "env": "dev"}),
    WithAnnotations(map[string]string{"description": "test backup"}),
    WithResources([]string{"deployments", "services"}),
    WithTTL(24*time.Hour),
    WithSnapshotVolumes(false),
)
```

### Example Test Structure

```go
func TestNonAdminBackupCreation(t *testing.T) {
    mockClient := NewMockK8sClient()
    
    expectedBackup := CreateTestBackup("test-backup", "test-namespace")
    
    err := mockClient.Create(context.Background(), expectedBackup)
    require.NoError(t, err)
    
    // Verify the created NonAdminBackup object
    createdBackup := mockClient.GetCreatedObjects()[0].(*nacv1alpha1.NonAdminBackup)
    err = CompareBackupSpecs(expectedBackup, createdBackup)
    require.NoError(t, err)
}
```

## Contributing

When adding new tests:

1. **Use the mock client** for unit tests that don't require cluster access
2. **Follow the existing patterns** for test structure and naming
3. **Add comprehensive validation** for all NonAdminBackup specifications
4. **Include error cases** to ensure robust error handling
5. **Update this README** with new test scenarios
6. **Focus on nonadmin commands** - these tests are specifically for nonadmin CLI functionality

## Future Enhancements

1. **Enhanced mock client** - Add support for more complex Kubernetes operations
2. **Performance tests** - Test CLI performance with large numbers of resources
3. **Concurrency tests** - Test concurrent backup operations
4. **Integration test environment** - Set up a proper test cluster for integration tests
5. **API version compatibility** - Test with different Kubernetes API versions

## Troubleshooting

### Common Issues

1. **Import errors** - Ensure all dependencies are properly installed with `go mod tidy`
2. **Scheme registration errors** - Verify custom resource schemes are properly registered
3. **Mock client issues** - Check that the mock client properly implements all required interfaces
4. **Test timeouts** - Increase timeout values for slow operations

### Debug Mode

Run tests with verbose output and debug information:

```bash
go test -v -debug ./...
```

## Conclusion

This test suite provides comprehensive coverage of the OADP CLI **nonadmin backup functionality**, ensuring that **nonadmin CLI commands** create and manipulate **NonAdminBackup** Kubernetes resources with the correct specifications. The mock client approach allows for thorough testing without requiring a real cluster, making the tests fast, reliable, and suitable for CI/CD pipelines.

**Scope:** These tests are specifically focused on the `oadp nonadmin backup` commands and NonAdminBackup resources, not the regular Velero backup commands. 
