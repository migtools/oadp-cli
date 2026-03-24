# Using the OADP CLI tools

## 1. Installing the OADP CLI

The OADP command-line interface (CLI) is a tool for managing backup and restore operations on an OpenShift Container Platform cluster. It is available as an `oc` plugin (`oc oadp`).

### Prerequisites

- You have access to an OpenShift Container Platform cluster with the OADP Operator installed.

### Installing the CLI

The OADP CLI is available from the **Command-line tools** page in the OpenShift web console when the OADP Operator is installed.

#### Procedure

1. Log in to the OpenShift web console as a user with access to the cluster.
2. Click the **?** icon in the toolbar and select **Command-line tools**.
3. Download the `oc-oadp` binary for your operating system and architecture.
4. Extract the archive and place the `oc-oadp` binary in a directory on your `PATH`.
5. Verify the installation by running the following command:

```terminal
$ oc oadp version
```

## 2. Setting up the OADP CLI

After installing the OADP CLI, you must run the setup command to configure it for your user permissions. The setup command automatically detects whether you have cluster-wide administrator permissions and configures the CLI accordingly.

The CLI operates in one of two modes:

- **Admin mode**: Provides access to cluster-wide Velero backup and restore commands.
- **Non-admin mode**: Provides access to namespace-scoped self-service backup and restore commands.

### Prerequisites

- You installed the OADP CLI.
- You are logged in to the OpenShift cluster with `oc login`.

### Procedure

1. Run the setup command to auto-detect your permissions and configure the CLI:

```terminal
$ oc oadp setup
```

The CLI checks whether you can create `backups.velero.io` resources across all namespaces. If you can, admin mode is enabled. Otherwise, non-admin mode is enabled.

Configuration is saved to `~/.config/velero/config.json`.

2. To reconfigure the CLI, for example after a change in permissions, run the setup command with the `--force` flag:

```terminal
$ oc oadp setup --force
```

### Verification

- Run `oc oadp --help` to confirm that the available commands match your configured mode.

> **Note:** OADP CLI commands support both noun-verb and verb-noun ordering (e.g., `oc oadp backup create` and `oc oadp create backup`). Both forms are equivalent.

## 3. Configuring the client

You can use the OADP CLI to view and modify client configuration settings. Configuration is stored in `~/.config/velero/config.json`.

### Prerequisites

- The OADP CLI is installed.

### Viewing the current configuration

You view the current client configuration by running the following command:

```terminal
$ oc oadp client config get
```

### Setting a configuration value

You set a configuration value by running the following command:

```terminal
$ oc oadp client config set <key>=<value>
```

Example:

```terminal
$ oc oadp client config set namespace=openshift-adp
```

## 4. Shell completion

You can use the OADP CLI to generate and install shell completion scripts for command auto-completion.

### Prerequisites

- The OADP CLI is installed.

### Installing shell completions

You install shell completions automatically by running the following command:

```terminal
$ oc oadp completion install [flags]
```

| Flag | Description |
|------|-------------|
| `--shell` | Shell type to install completions for. Supported values: `bash`, `zsh`, `fish`. If not specified, the current shell is auto-detected. |

The `install` subcommand writes completion scripts to the appropriate location for your shell and updates your shell profile if needed.

Example:

```terminal
$ oc oadp completion install --shell zsh
```

### Generating shell completion scripts

You generate a shell completion script without installing it by running the following command:

```terminal
$ oc oadp completion bash
$ oc oadp completion zsh
$ oc oadp completion fish
$ oc oadp completion powershell
```

You can redirect the output to a file or source it directly. For example:

```terminal
$ oc oadp completion bash > /etc/bash_completion.d/oc-oadp
```

---

## 5. Administrator perspective

The administrator perspective provides cluster-wide backup and restore operations using Velero resources. These commands are available when the OADP CLI is configured in admin mode.

### 5.1 Managing backups

You can use the OADP CLI to create, view, describe, download, and delete backups.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.

#### Creating a backup

You create a backup of cluster resources by running the following command:

```terminal
$ oc oadp backup create <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--include-namespaces` | Namespaces to include in the backup. Default: `*` (all namespaces). |
| `--exclude-namespaces` | Namespaces to exclude from the backup. |
| `--include-resources` | Resources to include in the backup. Accepts simple kind names (e.g., `deployments,services`) or `resource.group` format (e.g., `deployments.apps`) for disambiguation. Default: `*` (all resources). |
| `--exclude-resources` | Resources to exclude from the backup. Same format as `--include-resources`. |
| `--storage-location` | Name of the backup storage location to use. |
| `--volume-snapshot-locations` | Volume snapshot location(s) to use. |
| `--selector` / `-l` | Label selector to filter resources. |
| `--or-selector` | OR combination of label selectors. |
| `--snapshot-volumes` | Take PersistentVolume snapshots. Default: `true`. |
| `--snapshot-move-data` | Move snapshot data to the backup storage location. |
| `--default-volumes-to-fs-backup` | Use filesystem backup for all volumes. |
| `--include-cluster-resources` | Include cluster-scoped resources. |
| `--ttl` | Backup retention period. Default: `720h`. |
| `--csi-snapshot-timeout` | Timeout for CSI snapshot creation. |
| `--item-operation-timeout` | Timeout for asynchronous plugin operations. |
| `--request-timeout` | Timeout for the request to the Kubernetes API server. |

Example:

```terminal
$ oc oadp backup create my-backup \
    --include-namespaces my-namespace \
    --snapshot-volumes \
    --ttl 720h
```

#### Listing backups

You list all backups by running the following command:

```terminal
$ oc oadp backup get [<backup_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Describing a backup

You describe a backup to view its details by running the following command:

```terminal
$ oc oadp backup describe <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--details` | Display additional detail in the output. |

#### Viewing backup logs

You view the logs for a backup by running the following command:

```terminal
$ oc oadp backup logs <backup_name>
```

#### Downloading a backup

You download the contents of a backup by running the following command:

```terminal
$ oc oadp backup download <backup_name> [flags]
```

#### Deleting a backup

You delete a backup by running the following command:

```terminal
$ oc oadp backup delete <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Confirm deletion without prompting. |

### 5.2 Managing restores

You can use the OADP CLI to create, view, describe, and delete restores.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.
- A completed backup exists to restore from.

#### Creating a restore

You create a restore from an existing backup by running the following command:

```terminal
$ oc oadp restore create <restore_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--from-backup` | Name of the backup to restore from. |
| `--from-schedule` | Name of the schedule to restore from (uses the most recent backup). |
| `--include-namespaces` | Namespaces to include in the restore. Default: `*` (all namespaces). |
| `--exclude-namespaces` | Namespaces to exclude from the restore. |
| `--include-resources` | Resources to include in the restore. Accepts simple kind names (e.g., `deployments,services`) or `resource.group` format (e.g., `deployments.apps`) for disambiguation. Default: `*` (all resources). |
| `--exclude-resources` | Resources to exclude from the restore. Same format as `--include-resources`. |
| `--selector` / `-l` | Label selector to filter resources. |
| `--or-selector` | OR combination of label selectors. |
| `--include-cluster-resources` | Include cluster-scoped resources. |
| `--restore-volumes` | Restore PersistentVolume data from snapshots. |
| `--preserve-nodeports` | Preserve NodePort service port assignments. |
| `--item-operation-timeout` | Timeout for asynchronous plugin operations. |
| `--request-timeout` | Timeout for the request to the Kubernetes API server. |

Example:

```terminal
$ oc oadp restore create my-restore \
    --from-backup my-backup \
    --include-namespaces my-namespace
```

#### Listing restores

You list all restores by running the following command:

```terminal
$ oc oadp restore get [<restore_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Describing a restore

You describe a restore to view its details by running the following command:

```terminal
$ oc oadp restore describe <restore_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--details` | Display additional detail in the output. |

#### Viewing restore logs

You view the logs for a restore by running the following command:

```terminal
$ oc oadp restore logs <restore_name>
```

#### Deleting a restore

You delete a restore by running the following command:

```terminal
$ oc oadp restore delete <restore_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Confirm deletion without prompting. |

### 5.3 Managing schedules

You can use the OADP CLI to create, view, describe, and delete backup schedules. Schedules automate the creation of backups at specified intervals using a cron expression.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.

#### Creating a schedule

You create a backup schedule by running the following command:

```terminal
$ oc oadp schedule create <schedule_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--schedule` | Cron expression for the schedule (e.g., `0 1 * * *` for daily at 1 AM). |
| `--include-namespaces` | Namespaces to include in scheduled backups. Default: `*` (all namespaces). |
| `--exclude-namespaces` | Namespaces to exclude from scheduled backups. |
| `--include-resources` | Resources to include in scheduled backups. Accepts simple kind names (e.g., `deployments,services`) or `resource.group` format (e.g., `deployments.apps`) for disambiguation. Default: `*` (all resources). |
| `--exclude-resources` | Resources to exclude from scheduled backups. Same format as `--include-resources`. |
| `--storage-location` | Name of the backup storage location to use. |
| `--volume-snapshot-locations` | Volume snapshot location(s) to use. |
| `--selector` / `-l` | Label selector to filter resources. |
| `--snapshot-volumes` | Take PersistentVolume snapshots. Default: `true`. |
| `--snapshot-move-data` | Move snapshot data to the backup storage location. |
| `--default-volumes-to-fs-backup` | Use filesystem backup for all volumes. |
| `--include-cluster-resources` | Include cluster-scoped resources. |
| `--ttl` | Backup retention period. Default: `720h`. |
| `--request-timeout` | Timeout for the request to the Kubernetes API server. |

Example:

```terminal
$ oc oadp schedule create daily-backup \
    --schedule "0 1 * * *" \
    --include-namespaces my-namespace \
    --ttl 720h
```

#### Listing schedules

You list all schedules by running the following command:

```terminal
$ oc oadp schedule get [<schedule_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Describing a schedule

You describe a schedule to view its details by running the following command:

```terminal
$ oc oadp schedule describe <schedule_name> [flags]
```

#### Deleting a schedule

You delete a schedule by running the following command:

```terminal
$ oc oadp schedule delete <schedule_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Confirm deletion without prompting. |

### 5.4 Managing backup storage locations

You can use the OADP CLI to create, view, set, and delete backup storage locations (BSLs). Backup storage locations define where backup data is stored, such as an object storage bucket.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.

#### Creating a backup storage location

You create a backup storage location by running the following command:

```terminal
$ oc oadp backup-location create <bsl_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--provider` | Name of the cloud provider (e.g., `aws`, `gcp`, `azure`). |
| `--bucket` | Name of the object storage bucket. |
| `--prefix` | Path prefix within the bucket. |
| `--credential` | Secret and key for provider credentials in the format `SECRET_NAME=KEY`. |
| `--config` | Provider-specific configuration as key=value pairs. |
| `--backup-sync-period` | How often to sync backup contents from object storage. |
| `--request-timeout` | Timeout for the request to the Kubernetes API server. |

Example:

```terminal
$ oc oadp backup-location create my-bsl \
    --provider aws \
    --bucket my-velero-bucket \
    --prefix velero \
    --credential cloud-credentials=cloud
```

#### Listing backup storage locations

You list all backup storage locations by running the following command:

```terminal
$ oc oadp backup-location get [<bsl_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Setting a default backup storage location

You set the default backup storage location by running the following command:

```terminal
$ oc oadp backup-location set <bsl_name> [flags]
```

#### Deleting a backup storage location

You delete a backup storage location by running the following command:

```terminal
$ oc oadp backup-location delete <bsl_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Confirm deletion without prompting. |

### 5.5 Managing snapshot locations

You can use the OADP CLI to create, view, set, and delete volume snapshot locations (VSLs). Volume snapshot locations define where PersistentVolume snapshots are stored.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.

#### Creating a snapshot location

You create a volume snapshot location by running the following command:

```terminal
$ oc oadp snapshot-location create <vsl_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--provider` | Name of the cloud provider (e.g., `aws`, `gcp`, `azure`). |
| `--config` | Provider-specific configuration as key=value pairs. |
| `--request-timeout` | Timeout for the request to the Kubernetes API server. |

Example:

```terminal
$ oc oadp snapshot-location create my-vsl \
    --provider aws \
    --config region=us-east-1
```

#### Listing snapshot locations

You list all volume snapshot locations by running the following command:

```terminal
$ oc oadp snapshot-location get [<vsl_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Setting a default snapshot location

You set the default volume snapshot location by running the following command:

```terminal
$ oc oadp snapshot-location set <vsl_name> [flags]
```

#### Deleting a snapshot location

You delete a volume snapshot location by running the following command:

```terminal
$ oc oadp snapshot-location delete <vsl_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Confirm deletion without prompting. |

### 5.6 Managing NABSL approval requests

When the OADP Operator is configured with `nonAdmin.requireApprovalForBSL: true`, non-admin users who create a NonAdminBackupStorageLocation (NABSL) trigger an approval request in the OADP namespace. You can use the OADP CLI to view, describe, approve, and reject these requests.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.
- The DPA is configured with `nonAdmin.enable: true` and `nonAdmin.requireApprovalForBSL: true`.

#### Listing NABSL approval requests

You list all pending approval requests by running the following command:

```terminal
$ oc oadp nabsl-request get [<request_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

The output displays the request name, namespace, phase, requested NABSL name, requested namespace, and age.

Example:

```terminal
$ oc oadp nabsl-request get
```

#### Describing an NABSL approval request

You describe an approval request to view its full details, including the requested backup storage location spec, by running the following command:

```terminal
$ oc oadp nabsl-request describe <request_name>
```

You can specify the request by either the NABSL name or the full UUID.

Example:

```terminal
$ oc oadp nabsl-request describe my-bsl-request
```

#### Approving an NABSL approval request

You approve a pending request to allow the controller to create the corresponding BackupStorageLocation by running the following command:

```terminal
$ oc oadp nabsl-request approve <request_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--reason` | Reason for approval (optional). |

You can specify the request by either the NABSL name or the full UUID.

Example:

```terminal
$ oc oadp nabsl-request approve user-test-bsl --reason "Approved for production use"
```

#### Rejecting an NABSL approval request

You reject a pending request to deny the user's request for a backup storage location by running the following command:

```terminal
$ oc oadp nabsl-request reject <request_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--reason` | Reason for rejection (recommended). |

You can specify the request by either the NABSL name or the full UUID.

Example:

```terminal
$ oc oadp nabsl-request reject user-test-bsl --reason "Invalid configuration"
```

### 5.7 Collecting diagnostic data

You can use the OADP CLI to collect diagnostic information for OADP installations. The `must-gather` command runs the OADP must-gather tool to collect logs and cluster state information needed for troubleshooting and support cases.

#### Prerequisites

- The OADP CLI is installed and configured in admin mode.
- You are logged in to the OpenShift cluster as a user with `cluster-admin` privileges.
- The `oc` CLI is installed and available on your `PATH`.

#### Collecting diagnostic data

You collect OADP diagnostic information by running the following command:

```terminal
$ oc oadp must-gather [flags]
```

| Flag | Description |
|------|-------------|
| `--dest-dir` | Directory where must-gather output will be stored. Default: `./must-gather`. |
| `--request-timeout` | Timeout for the gather script (e.g., `30s`, `1m`). |
| `--skip-tls` | Skip TLS verification. |

The diagnostic bundle is saved to the specified directory.

Example:

```terminal
$ oc oadp must-gather --dest-dir=/tmp/oadp-diagnostics --request-timeout=1m
```

---

## 6. Non-administrator perspective

### 6.1 About OADP self-service

OADP self-service enables non-administrator users to perform backup and restore operations in their authorized namespaces without requiring cluster-wide administrator privileges. This feature provides secure self-service data protection capabilities while maintaining proper administrator controls over backup and restore operations.

#### Key capabilities

- Create and manage namespace-scoped backups and restores.
- View backup and restore status and logs.
- Create dedicated backup storage locations with user-owned buckets and credentials.

#### Limitations

- Cross-cluster operations and migrations are not supported for non-admin users.
- Non-admin volume snapshot locations (VSLs) are not supported. The VSL created by the cluster administrator in the DPA is used.
- Backups and restores are scoped to the namespace from which the command is run. You cannot specify a different namespace.
- Cluster-scoped resources cannot be included in backups or restores.
- ResourceModifiers and volume policies are not supported for non-admin backup and restore operations.
- Backup and restore logs via NonAdminDownloadRequest are not supported for default BSLs. NonAdminBackupStorageLocations must be created to access logs.

#### Prerequisites

Before using OADP self-service, the cluster administrator must have completed the following:

- Installed and configured the OADP Operator with `nonAdmin.enable: true` in the DPA spec.
- Created your user account, namespace, and namespace privileges (e.g., namespace admin).
- Granted editor roles for the following resources in your namespace:
  - `nonadminbackups.oadp.openshift.io`
  - `nonadminrestores.oadp.openshift.io`
  - `nonadminbackupstoragelocations.oadp.openshift.io`
  - `nonadmindownloadrequests.oadp.openshift.io`
- Optionally created a NonAdminBackupStorageLocation for your namespace.

### 6.2 Managing backups

You can use the OADP CLI to create, view, describe, and delete non-admin backups in your namespace.

#### Prerequisites

- The OADP CLI is installed and configured in non-admin mode.
- You are logged in to the OpenShift cluster and your current namespace context is set to the namespace you want to back up.
- You have editor roles for `nonadminbackups.oadp.openshift.io` in your namespace.
- A NonAdminBackupStorageLocation exists in your namespace, or a default has been configured with `oc oadp client config set default-nabsl=<name>`.

#### Creating a backup

You create a backup of resources in your current namespace by running the following command:

```terminal
$ oc oadp nonadmin backup create <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--storage-location` | Name of the NonAdminBackupStorageLocation to use. Required unless a default is configured. |
| `--include-resources` | Resources to include in the backup. Accepts simple kind names (e.g., `deployments,services`) or `resource.group` format (e.g., `deployments.apps`) for disambiguation. Default: `*` (all resources). |
| `--exclude-resources` | Resources to exclude from the backup. Same format as `--include-resources`. |
| `--selector` / `-l` | Only back up resources matching this label selector. |
| `--or-selector` | Back up resources matching at least one of the label selectors, separated by ` or `. |
| `--ttl` | How long before the backup can be garbage collected. Default: `720h`. |
| `--csi-snapshot-timeout` | Timeout for CSI snapshot creation. |
| `--item-operation-timeout` | Timeout for asynchronous plugin operations. |
| `--snapshot-volumes` | Take snapshots of PersistentVolumes as part of the backup. |
| `--snapshot-move-data` | Move snapshot data to the backup storage location. |
| `--default-volumes-to-fs-backup` | Use pod volume file system backup by default for volumes. |

Example:

```terminal
$ oc oadp nonadmin backup create my-backup \
    --storage-location my-nabsl \
    --include-resources deployments,services \
    --selector app=myapp \
    --snapshot-volumes \
    --ttl 720h
```

> **Tip:** To avoid specifying the storage location each time, run `oc oadp client config set default-nabsl=<NABSL_NAME>` to set a default.

#### Listing backups

You list all backups in your current namespace by running the following command:

```terminal
$ oc oadp nonadmin backup get [<backup_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Describing a backup

You describe a backup to view its details by running the following command:

```terminal
$ oc oadp nonadmin backup describe <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--details` | Display additional backup details including volume snapshots, resource lists, and item operations. |
| `--request-timeout` | Timeout for fetching backup details from the server. |

#### Viewing backup logs

You view the logs for a backup by running the following command:

```terminal
$ oc oadp nonadmin backup logs <backup_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--request-timeout` | Timeout for fetching logs from the server. |

> **Note:** Backup logs are only available when using a NonAdminBackupStorageLocation. Logs are not available for backups that use the default cluster BSL. Using the default cluster BSL is not recommended.

#### Deleting backups

You delete one or more backups by running the following command:

```terminal
$ oc oadp nonadmin backup delete [<backup_name>...] | --all [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Skip confirmation prompt and delete immediately. |
| `--all` | Delete all backups in the current namespace. |

The actual backup deletion is performed asynchronously by the OADP nonadmin controller.

Example:

```terminal
$ oc oadp nonadmin backup delete my-backup --confirm
```

### 6.3 Managing restores

You can use the OADP CLI to create, view, describe, and delete non-admin restores in your namespace.

#### Prerequisites

- The OADP CLI is installed and configured in non-admin mode.
- You are logged in to the OpenShift cluster and your current namespace context is set to the namespace you want to restore to.
- You have editor roles for `nonadminrestores.oadp.openshift.io` in your namespace.
- A completed non-admin backup exists to restore from.

#### Creating a restore

You create a restore from an existing non-admin backup by running the following command:

```terminal
$ oc oadp nonadmin restore create [<restore_name>] [flags]
```

The restore name is optional. If not provided, a name is automatically generated.

| Flag | Description |
|------|-------------|
| `--backup-name` | Name of the non-admin backup to restore from. Required. |
| `--include-resources` | Resources to include in the restore. Accepts simple kind names (e.g., `deployments,services`) or `resource.group` format (e.g., `deployments.apps`) for disambiguation. Default: `*` (all resources). |
| `--exclude-resources` | Resources to exclude from the restore. Same format as `--include-resources`. |
| `--selector` / `-l` | Only restore resources matching this label selector. |
| `--or-selector` | Restore resources matching at least one of the label selectors, separated by ` or `. |
| `--item-operation-timeout` | Timeout for asynchronous plugin operations. |

Example:

```terminal
$ oc oadp nonadmin restore create my-restore \
    --backup-name my-backup \
    --include-resources deployments,services \
    --selector app=myapp
```

#### Listing restores

You list all restores in your current namespace by running the following command:

```terminal
$ oc oadp nonadmin restore get [<restore_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |

#### Describing a restore

You describe a restore to view its details by running the following command:

```terminal
$ oc oadp nonadmin restore describe <restore_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--details` | Display additional restore details. |
| `--request-timeout` | Timeout for fetching restore details from the server. |

#### Viewing restore logs

You view the logs for a restore by running the following command:

```terminal
$ oc oadp nonadmin restore logs <restore_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--request-timeout` | Timeout for fetching logs from the server. |

> **Note:** Restore logs are only available when using a NonAdminBackupStorageLocation. Logs are not available for restores associated with backups that use the default cluster BSL. Using the default cluster BSL is not recommended.

#### Deleting restores

You delete one or more restores by running the following command:

```terminal
$ oc oadp nonadmin restore delete [<restore_name>...] | --all [flags]
```

| Flag | Description |
|------|-------------|
| `--confirm` | Skip confirmation prompt and delete immediately. |
| `--all` | Delete all restores in the current namespace. |

The actual restore deletion is performed asynchronously by the OADP nonadmin controller.

Example:

```terminal
$ oc oadp nonadmin restore delete my-restore --confirm
```

### 6.4 Managing backup storage locations (NonAdminBackupStorageLocations - NABSLs)

You can use the OADP CLI to create and view NonAdminBackupStorageLocations (NABSLs) in your namespace. NABSLs define where your backup data is stored using object storage that you own.

> **Note:** Updating or deleting NABSLs after creation is not supported for non-admin users.

#### Prerequisites

- The OADP CLI is installed and configured in non-admin mode.
- You are logged in to the OpenShift cluster and your current namespace context is set to the target namespace.
- You have editor roles for `nonadminbackupstoragelocations.oadp.openshift.io` in your namespace.
- You have a Kubernetes Secret in your namespace containing the credentials for your object storage provider.

#### Creating a backup storage location

You create a backup storage location by running the following command:

```terminal
$ oc oadp nonadmin bsl create <bsl_name> [flags]
```

| Flag | Description |
|------|-------------|
| `--provider` | Storage provider (required). Examples: `aws`, `azure`, `gcp`. |
| `--bucket` | Object storage bucket name (required). |
| `--credential` | Credential for this location as `SECRET_NAME=KEY` (required). The `SECRET_NAME` is the Kubernetes Secret name, and the `KEY` is the data key within the Secret. |
| `--prefix` | Prefix for backup objects in the bucket. |
| `--region` | Storage region (required for some providers like AWS). |
| `--config` | Additional provider-specific configuration as key=value pairs. |

If the cluster administrator has enabled `requireApprovalForBSL`, the NABSL will remain in a pending state until an administrator approves the request.

Example:

```terminal
$ oc oadp nonadmin bsl create my-storage \
    --provider aws \
    --bucket my-velero-bucket \
    --prefix velero-backups \
    --credential cloud-credentials=cloud \
    --region us-east-1
```

> **Tip:** After creating a BSL, run `oc oadp client config set default-nabsl=<BSL_NAME>` to set it as the default and avoid specifying the storage location on each backup.

#### Listing backup storage locations

You list all backup storage locations in your current namespace by running the following command:

```terminal
$ oc oadp nonadmin bsl get [<bsl_name>] [flags]
```

| Flag | Description |
|------|-------------|
| `-o` | Output format. Supported values: `json`, `yaml`. |
