# Verb-Noun Command System

This document explains how to add new nouns (resource types) and verbs (actions) to the OADP CLI's verb-noun command system.

## Overview

The OADP CLI supports both command patterns:
- **Noun-Verb**: `oadp backup get` (traditional Velero style)
- **Verb-Noun**: `oadp get backup` (kubectl style)

This system is implemented in two places:
- **Main commands**: `cmd/verbs/` - for admin-level resources
- **Non-admin commands**: `cmd/non-admin/verbs/` - for non-admin resources

## Architecture

### Core Components

1. **Builder** (`builder.go`) - Generic command builder that handles delegation logic
2. **Registry** (`registry.go`) - Resource registration system
3. **Verbs** (`verbs.go`) - Individual verb command definitions

### How It Works

```
User runs: oadp get backup
    ↓
Verb command (get) receives: ["backup"]
    ↓
Looks up "backup" in resource registry
    ↓
Gets backup.NewCommand() and finds "get" subcommand
    ↓
Delegates to: backup get (with all flags preserved)
```

## Adding New Verbs

### Step 1: Add Verb Function

In `cmd/verbs/verbs.go` (or `cmd/non-admin/verbs/verbs.go`):

```go
// NewLogsCommand creates the "logs" verb command
func NewLogsCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
    builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
    RegisterBackupResources(builder, "logs")
    RegisterRestoreResources(builder, "logs")
    RegisterScheduleResources(builder, "logs")
    
    return builder.BuildVerbCommand(VerbConfig{
        Use:   "logs",
        Short: "Get logs for resources",
        Long:  "Get logs for resources. This is a verb-based command that delegates to the appropriate noun command.",
        Example: `  # Get logs for a backup
  kubectl oadp logs backup my-backup

  # Get logs for a restore
  kubectl oadp logs restore my-restore`,
    })
}
```

### Step 2: Register in Root Command

In `cmd/root.go` (or `cmd/non-admin/nonadmin.go`):

```go
// Add verb-based commands
rootCmd.AddCommand(verbs.NewLogsCommand(veleroFactory, nonAdminFactory))
```

### Step 3: Update Flag Collection (if needed)

If your new verb has unique flags, update the `addFlagsFromResources` function in `builder.go`:

```go
for _, verb := range []string{"get", "create", "delete", "describe", "logs", "your-new-verb"} {
    // ... existing code
}
```

## Adding New Nouns (Resources)

### Step 1: Create Resource Commands

First, ensure your resource has the standard noun-verb structure:

```
cmd/your-resource/
├── your-resource.go      # Main command (e.g., "backup")
├── get.go               # get subcommand
├── create.go            # create subcommand
├── delete.go            # delete subcommand
└── describe.go          # describe subcommand
```

### Step 2: Add Resource Registration

In `cmd/verbs/registry.go` (or `cmd/non-admin/verbs/registry.go`):

```go
// RegisterYourResourceResources registers your-resource for a specific verb
func RegisterYourResourceResources(builder *VerbBuilder, verb string) {
    builder.RegisterResource("your-resource", ResourceHandler{
        GetCommandFunc: func(factory client.Factory) *cobra.Command {
            return yourresource.NewCommand(factory)
        },
        GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
            return getSubCommand(resourceCmd, verb)
        },
    })
}
```

### Step 3: Register in All Verb Commands

Update each verb function in `verbs.go`:

```go
func NewGetCommand(veleroFactory, nonAdminFactory client.Factory) *cobra.Command {
    builder := NewVerbBuilder(veleroFactory, nonAdminFactory)
    RegisterBackupResources(builder, "get")
    RegisterRestoreResources(builder, "get")
    RegisterScheduleResources(builder, "get")
    RegisterYourResourceResources(builder, "get") // Add this line
    
    return builder.BuildVerbCommand(VerbConfig{
        // ... existing config
    })
}
```

Repeat for all verbs: `create`, `delete`, `describe`, `logs`, etc.

### Step 4: Update Examples

Update the `Example` field in each verb to include your new resource:

```go
Example: `  # Get all backups
  kubectl oadp get backup

  # Get all your-resources
  kubectl oadp get your-resource`,
```

## Conditional Resource Registration

Some resources may not support all verbs. Use conditional registration:

```go
// RegisterBSLResources - BSL only supports create
func RegisterBSLResources(builder *NonAdminVerbBuilder, verb string) {
    if verb == "create" {
        builder.RegisterResource("bsl", NonAdminResourceHandler{
            // ... registration logic
        })
    }
}
```

## Non-Admin Resources

For non-admin resources, follow the same pattern but use:

- **Directory**: `cmd/non-admin/verbs/`
- **Builder**: `NonAdminVerbBuilder`
- **Config**: `NonAdminVerbConfig`
- **Handler**: `NonAdminResourceHandler`

## Testing Your Changes

### 1. Build the CLI
```bash
go build -o kubectl-oadp .
```

### 2. Test Verb-Noun Commands
```bash
# Test new verb
./kubectl-oadp your-verb --help

# Test new noun
./kubectl-oadp get your-resource --help

# Test actual delegation
./kubectl-oadp get your-resource
```

### 3. Test Flag Preservation
```bash
# Ensure flags work correctly
./kubectl-oadp get your-resource -o json
```

## Common Patterns

### Resource with Limited Verbs

If your resource only supports certain verbs:

```go
func RegisterYourResourceResources(builder *VerbBuilder, verb string) {
    supportedVerbs := []string{"get", "describe"}
    for _, supportedVerb := range supportedVerbs {
        if verb == supportedVerb {
            builder.RegisterResource("your-resource", ResourceHandler{
                // ... registration
            })
            break
        }
    }
}
```

### Resource with Custom Flags

If your resource has unique flags, ensure they're properly copied:

```go
// In builder.go, addFlagsFromResources function
// The system automatically collects flags from all registered resources
// No additional changes needed unless you have special flag handling
```

## Troubleshooting

### "unknown resource type" Error
- Ensure the resource is registered in the verb's builder
- Check that the resource name matches exactly

### "command not found" Error
- Verify the resource has the expected subcommand (get, create, etc.)
- Check the `getSubCommand` function in registry.go

### Flags Not Working
- Ensure flags are added to the resource's subcommand
- Check that `addFlagsFromResources` includes your verb

### Build Errors
- Verify all imports are correct
- Check that resource command functions exist and return `*cobra.Command`

## Examples

### Complete Example: Adding "schedule" Resource

1. **Resource exists**: `cmd/schedule/` with `get.go`, `create.go`, etc.

2. **Add to registry** (`cmd/verbs/registry.go`):
```go
func RegisterScheduleResources(builder *VerbBuilder, verb string) {
    builder.RegisterResource("schedule", ResourceHandler{
        GetCommandFunc: func(factory client.Factory) *cobra.Command {
            return schedule.NewCommand(factory)
        },
        GetSubCommandFunc: func(resourceCmd *cobra.Command) *cobra.Command {
            return getSubCommand(resourceCmd, verb)
        },
    })
}
```

3. **Update all verbs** (`cmd/verbs/verbs.go`):
```go
// In each verb function, add:
RegisterScheduleResources(builder, "get")    // for NewGetCommand
RegisterScheduleResources(builder, "create") // for NewCreateCommand
// etc.
```

4. **Update examples** in each verb's `Example` field.

That's it! The system will automatically handle delegation, flag preservation, and error handling.
