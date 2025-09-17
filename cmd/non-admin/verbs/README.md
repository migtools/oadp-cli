# Non-Admin Verb-Noun Command System

This directory contains the verb-noun command system for non-admin resources in the OADP CLI.

## Overview

The non-admin verb system works identically to the main verb system but is specifically designed for non-admin resources like:
- **backup** - Non-admin backups
- **bsl** - Backup storage locations

## Usage

### Supported Commands

```bash
# Get commands
kubectl oadp nonadmin get backup
kubectl oadp nonadmin get bsl        # Error: BSL doesn't support get

# Create commands  
kubectl oadp nonadmin create backup my-backup
kubectl oadp nonadmin create bsl my-bsl

# Delete commands
kubectl oadp nonadmin delete backup my-backup
kubectl oadp nonadmin delete bsl     # Error: BSL doesn't support delete

# Describe commands
kubectl oadp nonadmin describe backup my-backup
kubectl oadp nonadmin describe bsl   # Error: BSL doesn't support describe

# Logs commands
kubectl oadp nonadmin logs backup my-backup
kubectl oadp nonadmin logs bsl       # Error: BSL doesn't support logs
```

## Architecture

### Files

- **`builder.go`** - `NonAdminVerbBuilder` for building verb commands
- **`registry.go`** - Resource registration for backup and bsl
- **`verbs.go`** - Verb command definitions (get, create, delete, describe, logs)

### Key Differences from Main Verbs

1. **Builder Type**: `NonAdminVerbBuilder` instead of `VerbBuilder`
2. **Config Type**: `NonAdminVerbConfig` instead of `VerbConfig`
3. **Handler Type**: `NonAdminResourceHandler` instead of `ResourceHandler`
4. **Single Factory**: Only uses one factory (non-admin factory)

## Adding New Non-Admin Resources

### Step 1: Create Resource Commands

Ensure your resource follows the noun-verb pattern:

```
cmd/non-admin/your-resource/
├── your-resource.go      # Main command
├── get.go               # get subcommand (if supported)
├── create.go            # create subcommand (if supported)
└── describe.go          # describe subcommand (if supported)
```

### Step 2: Add Resource Registration

In `cmd/non-admin/verbs/registry.go`:

```go
// RegisterYourResourceResources registers your-resource for a specific verb
func RegisterYourResourceResources(builder *NonAdminVerbBuilder, verb string) {
    // Only register for supported verbs
    supportedVerbs := []string{"get", "create", "describe"}
    for _, supportedVerb := range supportedVerbs {
        if verb == supportedVerb {
            builder.RegisterResource("your-resource", NonAdminResourceHandler{
                GetCommandFunc: func(factory client.Factory) *cobra.Command {
                    return yourresource.NewCommand(factory)
                },
                GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
                    return getSubCommand(resourceCmd, verb)
                },
            })
            break
        }
    }
}
```

### Step 3: Register in Verb Commands

Update each verb function in `cmd/non-admin/verbs/verbs.go`:

```go
func NewGetCommand(factory client.Factory) *cobra.Command {
    builder := NewNonAdminVerbBuilder(factory)
    RegisterBackupResources(builder, "get")
    RegisterBSLResources(builder, "get")
    RegisterYourResourceResources(builder, "get") // Add this line
    
    return builder.BuildVerbCommand(NonAdminVerbConfig{
        // ... existing config
    })
}
```

### Step 4: Update Examples

Add your resource to the examples in each verb:

```go
Example: `  # Get all non-admin backups
  kubectl oadp nonadmin get backup

  # Get all your-resources
  kubectl oadp nonadmin get your-resource`,
```

## Conditional Registration Example

The BSL resource only supports `create`, so it uses conditional registration:

```go
func RegisterBSLResources(builder *NonAdminVerbBuilder, verb string) {
    if verb == "create" {
        builder.RegisterResource("bsl", NonAdminResourceHandler{
            // ... registration logic
        })
    }
}
```

## Testing

### Build and Test
```bash
go build -o kubectl-oadp .

# Test new resource
./kubectl-oadp nonadmin get your-resource
./kubectl-oadp nonadmin create your-resource test-name
```

### Verify Error Handling
```bash
# Should show "unknown resource type" for unsupported verbs
./kubectl-oadp nonadmin get bsl
./kubectl-oadp nonadmin describe bsl
```

## Current Resources

### Backup
- **Supported Verbs**: get, create, delete, describe, logs
- **Command**: `backup.NewBackupCommand(factory)`

### BSL (Backup Storage Location)
- **Supported Verbs**: create only
- **Command**: `bsl.NewBSLCommand(factory)`

## Integration

The non-admin verb commands are integrated into the main non-admin command in `cmd/non-admin/nonadmin.go`:

```go
// Add verb-based commands for compatibility with Velero CLI pattern
c.AddCommand(verbs.NewGetCommand(f))
c.AddCommand(verbs.NewCreateCommand(f))
c.AddCommand(verbs.NewDeleteCommand(f))
c.AddCommand(verbs.NewDescribeCommand(f))
c.AddCommand(verbs.NewLogsCommand(f))
```

This allows users to use either pattern:
- `oadp nonadmin backup get` (noun-verb)
- `oadp nonadmin get backup` (verb-noun)
