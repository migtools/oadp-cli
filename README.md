# OADP CLI

[![Cross-Architecture Build Test](https://github.com/migtools/oadp-cli/actions/workflows/cross-arch-build-test.yml/badge.svg)](https://github.com/migtools/oadp-cli/actions/workflows/cross-arch-build-test.yml)

A kubectl plugin for OpenShift API for Data Protection (OADP) that provides both administrative and non-administrative backup operations.

> **What it does**: Extends OADP functionality with a unified CLI that supports both cluster-wide Velero operations (admin) and namespace-scoped self-service operations (non-admin users).

## Key Capabilities

- **Admin Operations**: Full Velero backup, restore, and version commands (requires cluster admin permissions)
- **Non-Admin Operations**: Namespace-scoped backup operations using non-admin CRDs (works with regular user permissions)
- **Smart Namespace Handling**: Non-admin commands automatically operate in your current kubectl context namespace
- **Seamless Integration**: Works as a standard kubectl plugin

## Command Structure

```
kubectl oadp
├── backup          # Velero cluster-wide backups (admin)
├── restore         # Velero cluster-wide restores (admin) 
├── version         # Version information
└── nonadmin (na)   # Namespace-scoped operations (non-admin)
    └── backup
        ├── create
        ├── describe
        ├── logs
        └── delete
```

## Installation

### Using Krew (Available soon!)

```sh
# Install Krew if you haven't already
kubectl krew install krew

# Install the OADP plugin
kubectl krew install oadp

# Verify installation
kubectl oadp --help
```

### Manual Build and Install

```sh
# Build and install directly
make install

# Or build manually
make build
sudo mv kubectl-oadp /usr/local/bin/
```

## Usage Guide

### Non-Admin Backup Operations

Non-admin commands work within your current namespace and user permissions:

```sh
# Basic backup of current namespace
kubectl oadp nonadmin backup create my-backup
# Short form
kubectl oadp na backup create my-backup

# Include specific resource types only
kubectl oadp na backup create app-backup --include-resources deployments,services,configmaps

# Exclude sensitive data
kubectl oadp na backup create safe-backup --exclude-resources secrets

# Preview backup configuration without creating
kubectl oadp na backup create test-backup --snapshot-volumes=false -o yaml

# Create backup and wait for completion
kubectl oadp na backup create prod-backup --wait

# Check backup status
kubectl oadp na backup describe my-backup

# View backup logs
kubectl oadp na backup logs my-backup

# Delete a backup
kubectl oadp na backup delete my-backup
```

### Admin Operations

Admin commands require cluster-level permissions and operate across all namespaces:

```sh
# Cluster-wide backup operations
kubectl oadp backup create cluster-backup --include-namespaces namespace1,namespace2

# Restore operations
kubectl oadp restore create --from-backup cluster-backup

# Check OADP/Velero version
kubectl oadp version
```

## How Non-Admin Backups Work

1. **Namespace Detection**: Commands automatically use your current kubectl context namespace
2. **Permission Model**: Works with standard namespace-level RBAC permissions
3. **Resource Creation**: Creates `Non-admin` custom resources that are processed by the OADP operator
4. **Velero Integration**: OADP operator translates NonAdminBackup resources into standard Velero backup jobs

**Example workflow:**
```sh
# Switch to your project namespace
kubectl config set-context --current --namespace=my-project

# Create backup (automatically backs up 'my-project' namespace)
kubectl oadp na backup create project-backup --wait

# Monitor progress
kubectl oadp na backup logs project-backup
```

## Development

### Quick Development Commands

```sh
# Build and test locally
make build
./kubectl-oadp --help

# Run integration tests
make test

# Build release archives
make release

# Generate Krew manifest
make krew-manifest
```

### Project Structure

- **`cmd/`**: Command definitions and CLI logic
- **`cmd/non-admin/`**: Non-admin specific commands
- **`tests/`**: Integration tests and test utilities
- **`Makefile`**: Build automation and common tasks

## Testing

Comprehensive integration tests verify CLI functionality:

```bash
# Run all tests
make test

# For detailed test information
cat tests/README.md
```

## Technical Details

**Built with:**
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Velero client libraries](https://github.com/vmware-tanzu/velero) - Core backup functionality  
- [OADP NonAdmin APIs](https://github.com/migtools/oadp-non-admin) - NonAdminBackup CRD support

**Dependencies:**
- OADP Operator installed in cluster
- Appropriate RBAC permissions for your use case

## License

Apache License 2.0 - see [LICENSE](LICENSE) file.

Integrates with Apache 2.0 licensed projects: [Velero](https://github.com/vmware-tanzu/velero), [OADP](https://github.com/openshift/oadp-operator), [Kubernetes](https://github.com/kubernetes/kubernetes).