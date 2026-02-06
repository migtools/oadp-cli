# NonAdminRestore Create Command

## Overview

The `nonadmin restore create` command creates restore requests for non-admin users within their authorized namespaces.

## Minimal MVP Flags

The following flags represent the minimal viable product for restore creation (7 total):

### Core Flags

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--backup-name` | String | - | Source backup (required) | ✅ MVP |
| `--include-resources` | StringArray | `["*"]` | Resources to include | ✅ MVP |
| `--exclude-resources` | StringArray | - | Resources to exclude | ✅ MVP |

### Label Selection

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--selector`, `-l` | LabelSelector | - | Label selector | ✅ MVP |
| `--or-selector` | OrLabelSelector | - | OR label selectors | ✅ MVP |

### Cluster Resources

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--include-cluster-resources` | OptionalBool | - | Include cluster resources | ✅ MVP |

### Timing

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--item-operation-timeout` | Duration | - | Operation timeout | ✅ MVP |

## Restricted Flags (Not Available)

The following flags are **restricted** for non-admin users per the NAR API restrictions:

| Flag | Reason | Doc Reference |
|------|--------|---------------|
| `--from-schedule` | Not supported for non-admin | NAR API docs |
| `--include-namespaces` | Restricted - automatically set | NAR API docs |
| `--exclude-namespaces` | Restricted for non-admin users | NAR API docs |
| `--namespace-mappings` | Restricted for non-admin users | NAR API docs |

## Flags Not in MVP (Future Enhancements)

The following flags are **allowed by the API** but not included in the minimal MVP:

### Metadata
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--labels` | ✅ Yes | Future |
| `--annotations` | ✅ Yes | Future |

### Restore Behavior
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--restore-volumes` | ✅ Yes | Future |
| `--preserve-nodeports` | ✅ Yes | Future |
| `--existing-resource-policy` | ✅ Yes | Future |

### Advanced Features
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--resource-modifier-configmap` | ✅ Yes | Future |
| `--status-include-resources` | ✅ Yes | Future |
| `--status-exclude-resources` | ✅ Yes | Future |
| `--write-sparse-files` | ✅ Yes | Future |
| `--parallel-files-download` | ✅ Yes | Future |

### UX Flags
| Flag | Purpose | Could Add Later |
|------|---------|-----------------|
| `--wait` | Wait for restore completion | Future |

## Examples

```bash
# Create a simple restore from a backup
oadp nonadmin restore create my-restore --backup-name my-backup

# Create restore with specific resources
oadp nonadmin restore create my-restore \
  --backup-name my-backup \
  --include-resources deployments,services

# Create restore excluding certain resources
oadp nonadmin restore create my-restore \
  --backup-name my-backup \
  --exclude-resources secrets

# Create restore with label selector
oadp nonadmin restore create my-restore \
  --backup-name my-backup \
  --selector app=myapp

# View the YAML without creating it
oadp nonadmin restore create my-restore \
  --backup-name my-backup \
  -o yaml
```

## Architecture Notes

The restore create command uses **struct embedding** from Velero's restore CreateOptions. This approach:
- Reduces code duplication
- Ensures compatibility with Velero updates
- Uses BindFlags() as the control gate to expose only MVP features to non-admin users
- Maintains forward compatibility for future enhancements

## Implementation Details

### Struct Embedding Pattern

```go
type CreateOptions struct {
    *velerorestore.CreateOptions  // Embed Velero's CreateOptions

    // NAR-specific fields
    Name             string
    client           kbclient.WithWatch
    currentNamespace string
}
```

### MVP Flag Control

The `BindFlags()` method acts as a control gate, exposing only the MVP flags while the embedded struct contains all Velero options. This allows:
- Easy addition of new flags in the future (just bind them in BindFlags)
- Automatic compatibility with Velero struct updates
- Clear separation between what's exposed vs what's available
