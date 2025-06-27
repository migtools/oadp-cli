# OADP CLI

A kubectl plugin for working with OpenShift API for Data Protection (OADP) resources, including NonAdminBackup operations.

> This project provides a `kubectl` plugin CLI that extends OADP functionality, allowing users to work with both regular Velero resources and NonAdminBackup resources through a unified interface.

## Features

- **Regular OADP operations**: Standard Velero backup, restore, and version commands
- **NonAdmin operations**: Create and manage NonAdminBackup resources for namespace-scoped backup operations
- **Automatic namespace detection**: NonAdminBackup automatically uses your current kubectl context namespace
- **Kubectl plugin integration**: Works seamlessly as a kubectl plugin

## Command Structure

```
oadp
├── backup (Velero backups)
├── restore (Velero restores) 
├── version
└── nonadmin
    └── backup
        └── create
```

## Build and Install

### Quick Installation

Use the provided script for quick build and installation:

```sh
chmod +x quick-create.sh
./quick-create.sh
```

### Manual Installation

1. **Build the CLI:**
   ```sh
   go build -o kubectl-oadp .
   ```

2. **Install as kubectl plugin:**
   ```sh
   sudo mv kubectl-oadp /usr/local/bin/
   ```

3. **Verify installation:**
   ```sh
   kubectl oadp --help
   ```

## Usage Examples

### NonAdminBackup Operations

```sh
# Create a non-admin backup of the current namespace
kubectl oadp nonadmin backup create my-backup

# Create backup with specific resource types
kubectl oadp nonadmin backup create my-backup --include-resources deployments,services

# Create backup excluding certain resources
kubectl oadp nonadmin backup create my-backup --exclude-resources secrets

# View backup YAML without creating it
kubectl oadp nonadmin backup create my-backup --snapshot-volumes=false -o yaml

# Wait for backup completion
kubectl oadp nonadmin backup create my-backup --wait
```

### Regular OADP Operations

```sh
# Work with regular Velero backups
kubectl oadp backup --help

# Work with restores
kubectl oadp restore --help

# Check version
kubectl oadp version
```

## Key Features of NonAdminBackup

- **Namespace-scoped**: Automatically backs up the namespace where the NonAdminBackup resource is created
- **Simplified workflow**: No need to specify `--include-namespaces` - it uses your current kubectl context
- **Permission-aware**: Works within the permissions of the current user/service account
- **Integration with OADP**: Leverages the underlying Velero infrastructure managed by OADP operator

## Testing

This project includes comprehensive CLI integration tests organized by functionality.

### Quick Test Commands

```bash
# Run all tests (standard Go pattern)
go test ./...

📖 **For detailed test documentation, see [tests/README.md](tests/README.md)**

## Development

This CLI is built using:
- [Cobra](https://github.com/spf13/cobra) for CLI framework
- [Velero client libraries](https://github.com/vmware-tanzu/velero) for core functionality  
- [OADP NonAdmin APIs](https://github.com/migtools/oadp-non-admin) for NonAdminBackup operations