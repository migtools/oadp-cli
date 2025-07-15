# OADP CLI E2E Tests

This directory contains end-to-end tests for the OADP CLI using Ginkgo and Gomega.

## Overview

These tests validate the OADP CLI functionality in a real Kubernetes environment by:
1. Setting up a DataProtectionApplication (DPA) with AWS S3 backend
2. Testing CLI commands against the configured environment
3. Verifying integration with the OADP operator
4. Testing real backup/restore workflows

## Prerequisites

Before running the e2e tests, you **must** have:

1. **Kubernetes cluster** running and accessible via `kubectl`
2. **OADP operator** installed in the `openshift-adp` namespace
3. **KUBECONFIG** set up to access the cluster
4. **Go 1.24+** installed
5. **AWS credentials** configured with S3 bucket access

### Installing OADP Operator

```bash
# Create namespace
kubectl create namespace openshift-adp

# Install OADP operator
kubectl apply -f https://github.com/openshift/oadp-operator/releases/latest/download/oadp-operator.yaml

# Wait for operator to be ready
kubectl wait --for=condition=available deployment/openshift-adp-controller-manager -n openshift-adp --timeout=300s
```

## Required Environment Variables

The e2e tests require the following environment variables to be set:

```bash
export OADP_CRED_FILE=~/.aws/credentials
export OADP_BUCKET="your-s3-bucket-name"
export CI_CRED_FILE=~/.aws/credentials
export VSL_REGION="us-east-1"
```

| Variable | Description | Example |
|----------|-------------|---------|
| `OADP_CRED_FILE` | Path to AWS credentials file | `~/.aws/credentials` |
| `OADP_BUCKET` | S3 bucket name for backups | `jvaikath-velero` |
| `CI_CRED_FILE` | Path to CI credentials file | `~/.aws/credentials` |
| `VSL_REGION` | AWS region for backup storage | `us-east-1` |

## AWS Configuration

Your AWS credentials file should contain:

```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
region = us-east-1
```

Ensure your AWS credentials have permissions for:
- S3 bucket operations (list, get, put, delete)
- EBS snapshot operations (create, delete, describe)

## Running Tests

### Quick Start

```bash
# Set your AWS configuration
export OADP_CRED_FILE=~/.aws/credentials
export OADP_BUCKET="jvaikath-velero"
export CI_CRED_FILE=~/.aws/credentials
export VSL_REGION="us-east-1"

# From the project root
make test-e2e

# Or from the e2e directory
cd tests/e2e
go test -v -ginkgo.v --timeout=10m
```

### Focus on Specific Tests

```bash
# Run only basic CLI tests
go test -v -ginkgo.v -ginkgo.focus="CLI Help Commands"

# Run only DPA tests  
go test -v -ginkgo.v -ginkgo.focus="DPA Configuration"

# Skip long-running tests
go test -v -ginkgo.v -ginkgo.skip="version command multiple times"
```

## What the Tests Do

### BeforeSuite Setup:
1. ✅ Validates all required environment variables are set
2. ✅ Verifies AWS credentials file exists and is accessible
3. ✅ Checks that OADP operator is installed and running
4. ✅ Creates `cloud-credentials` secret from AWS credentials
5. ✅ Creates DPA with your S3 bucket and region configuration
6. ✅ Waits for DPA to be reconciled and ready

### Test Execution:
1. **CLI Help Commands** - Tests basic CLI functionality
2. **DPA Configuration Tests** - Validates DPA is properly configured
3. **Basic CLI Commands** - Tests backup commands and error handling
4. **Cluster Connectivity** - Verifies Kubernetes access
5. **DPA Status Verification** - Checks DPA, backup locations, and snapshot locations
6. **CLI Integration** - Tests end-to-end command workflows

### AfterSuite Cleanup:
1. ✅ Deletes the test DPA
2. ✅ Removes the cloud-credentials secret
3. ✅ Cleans up all test resources
4. ✅ Removes temporary CLI binary

## Test Structure

### Test Categories

1. **CLI Help Commands**
   - `--help` flag functionality
   - Version information display
   - Command structure validation

2. **DPA Configuration Tests**
   - DPA creation and validation
   - OADP operator status
   - AWS plugin configuration

3. **Basic CLI Commands**
   - Backup command execution
   - Error handling for invalid commands
   - Command consistency testing

4. **Cluster Connectivity**
   - Kubernetes API access
   - Namespace permissions
   - Resource discovery

5. **DPA Status Verification**
   - DPA reconciliation status
   - Backup location configuration
   - Snapshot location setup

6. **CLI Integration**
   - End-to-end command workflows
   - Error scenarios
   - Flag validation

## Troubleshooting

### Common Issues

1. **Missing Environment Variables**
   ```bash
   # Error: Environment variable OADP_BUCKET is required for DPA creation
   export OADP_BUCKET="your-s3-bucket-name"
   ```

2. **AWS Credentials Issues**
   ```bash
   # Verify credentials file exists
   ls -la ~/.aws/credentials
   
   # Test AWS connectivity
   aws s3 ls s3://your-bucket-name
   ```

3. **OADP Operator Not Ready**
   ```bash
   # Check operator installation
   kubectl get deployment -n openshift-adp
   
   # Check operator logs
   kubectl logs -n openshift-adp deployment/openshift-adp-controller-manager
   ```

4. **DPA Creation Timeout**
   ```bash
   # Check OADP operator logs
   kubectl logs -n openshift-adp deployment/openshift-adp-controller-manager
   
   # Check DPA status
   kubectl get dpa -n openshift-adp -o yaml
   ```

### Test Debugging

```bash
# Run with verbose output
go test -v -ginkgo.v -ginkgo.vv

# Run with progress reporting
go test -v -ginkgo.v -ginkgo.progress

# Run a single test
go test -v -ginkgo.v -ginkgo.focus="should create DPA"
```

## Adding New Tests

1. Create test in `basic_test.go`
2. Use appropriate `Context` and `It` blocks
3. Use `Eventually` for async operations
4. Clean up resources in `AfterEach` if needed

Example:
```go
Context("New Test Category", func() {
    It("should do something", func() {
        // Your test code here
        Eventually(func() error {
            // Test logic
            return nil
        }, testTimeout, pollInterval).Should(Succeed())
    })
})
```

## CI/CD Integration

The tests can be integrated into CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run E2E Tests
  env:
    OADP_CRED_FILE: ${{ secrets.AWS_CREDENTIALS_FILE }}
    OADP_BUCKET: ${{ secrets.S3_BUCKET }}
    CI_CRED_FILE: ${{ secrets.AWS_CREDENTIALS_FILE }}
    VSL_REGION: us-east-1
  run: |
    cd tests/e2e
    go test -v -ginkgo.v --timeout=10m
```

## Development Notes

- All tests require a real DPA to be created and configured
- Tests are designed to be run against a live OADP installation
- The DPA is created and destroyed for each test run
- AWS credentials and S3 bucket access are mandatory
- Tests validate both CLI functionality and OADP integration 