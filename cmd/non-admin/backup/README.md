# NonAdminBackup Create Command

## Overview

The `nonadmin backup create` command creates backup requests for non-admin users within their authorized namespaces.

## Minimal MVP Flags

The following flags represent the minimal viable product for backup creation:

### Resource Filtering

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--include-resources` | StringArray | `["*"]` | Resources to include | ✅ MVP |
| `--exclude-resources` | StringArray | - | Resources to exclude | ✅ MVP |

### Label Selection

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--selector`, `-l` | LabelSelector | - | Label selector filter | ✅ MVP |
| `--or-selector` | OrLabelSelector | - | OR label selectors | ✅ MVP |

### Cluster Resources

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--include-cluster-resources` | OptionalBool | - | Include cluster resources (users can only set to false) | ✅ MVP |

### Timing & Storage

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--ttl` | Duration | - | Backup retention time | ✅ MVP |
| `--storage-location` | String | - | NABSL reference | ✅ MVP |
| `--csi-snapshot-timeout` | Duration | - | CSI snapshot timeout | ✅ MVP |
| `--item-operation-timeout` | Duration | - | Async operation timeout | ✅ MVP |

### Snapshot Control

| Flag | Type | Default | Description | Status |
|------|------|---------|-------------|--------|
| `--snapshot-volumes` | OptionalBool | - | Enable volume snapshots | ✅ MVP |
| `--snapshot-move-data` | OptionalBool | - | Move snapshot data | ✅ MVP |
| `--default-volumes-to-fs-backup` | OptionalBool | - | Use filesystem backup | ✅ MVP |

## Restricted Flags (Not Available)

The following flags are **restricted** for non-admin users per the NAB API restrictions:

| Flag | Reason | Doc Reference |
|------|--------|---------------|
| `--include-namespaces` | Restricted - automatically set to current namespace | NAB API docs |
| `--exclude-namespaces` | Restricted for non-admin users | NAB API docs |
| `--include-cluster-scoped-resources` | Restricted - only empty list acceptable | NAB API docs |
| `--volume-snapshot-locations` | Not supported - defaults used | NAB API docs |

## Flags Not in MVP (Future Enhancements)

The following flags are **allowed by the API** but not included in the minimal MVP:

### Metadata
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--labels` | ✅ Yes | Future |
| `--annotations` | ✅ Yes | Future |

### Advanced Features
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--from-schedule` | N/A | Future (requires schedule API) |
| `--ordered-resources` | ✅ Yes | Future |
| `--data-mover` | ✅ Yes | Future |
| `--resource-policies-configmap` | ✅ Yes | Future (admin-created only) |
| `--parallel-files-upload` | ✅ Yes | Future |

### Scoped Resources
| Flag | Admin Enforceable | Could Add Later |
|------|------------------|-----------------|
| `--exclude-cluster-scoped-resources` | ✅ Yes | Future |
| `--include-namespace-scoped-resources` | ✅ Yes | Future |
| `--exclude-namespace-scoped-resources` | ✅ Yes | Future |

## Examples

```bash
# Create a simple backup of all resources in the current namespace
oadp nonadmin backup create my-backup

# Create backup with specific resources
oadp nonadmin backup create my-backup \
  --include-resources deployments,services

# Create backup with label selector
oadp nonadmin backup create my-backup \
  --selector app=myapp

# Create backup with snapshots and TTL
oadp nonadmin backup create my-backup \
  --snapshot-volumes \
  --ttl 720h

# Create backup with specific storage location
oadp nonadmin backup create my-backup \
  --storage-location my-nabsl
```

## Architecture Notes

The backup create command uses **struct embedding** from Velero's backup CreateOptions, matching the pattern used in `nonadmin restore create`. This approach:
- Reduces code duplication
- Ensures compatibility with Velero updates
- Uses BindFlags() as the control gate to expose only MVP features to non-admin users
- Maintains forward compatibility for future enhancements

## Implementation Details

### Struct Embedding Pattern

```go
type CreateOptions struct {
    *velerobackup.CreateOptions  // Embed Velero's CreateOptions

    // NAB-specific fields
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
