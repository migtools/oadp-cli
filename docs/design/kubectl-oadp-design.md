# Kubectl-oadp plugin design

## Abstract
The purpose of the kubectl-oadp plugin is to allow customers to create, delete, and manage backups and restores in OADP without needing to alias velero. On the `oadp-1.4` branch, only cluster-admin operations are supported; the OADP 1.4 operator does not ship the non-admin controller.

## Background
The current OpenShift CLI is suboptimal: `oc backup delete $foo` deletes the Kubernetes object instead of the backup, whereas `velero backup delete $foo` deletes the backup and its files in storage. Customers previously had to alias velero to get correct behaviour. The purpose of kubectl-oadp is to make the CLI experience cleaner and easier to use, and to provide direct access to backup and restore logs.

## Goals
- Customers can create, delete, describe, and get the logs of backups and restores
- Cluster admins can manage backup storage locations and schedules
- Plugin runs on OpenShift clusters with OADP installed

## Non-Goals (oadp-1.4)
- Non-admin / self-service backup commands (`oc oadp nonadmin …`) — the OADP 1.4 operator does not ship the non-admin controller (`oadp-non-admin-rhel9` is absent from 1.4 `image-references`). Non-admin support is available on OADP 1.5+.
- NABSL approval workflow (`oc oadp nabsl-request …`) — requires NAC / NABSL CRDs (OADP 1.5+).

## Use-Case
A cluster admin needs to back up a namespace on an OpenShift 4.17–4.18 cluster running OADP 1.4. They use `oc oadp backup create` instead of reaching for the raw velero binary, getting familiar `oc`-style output and error messages.

## High-Level Design
The plugin wraps the Velero CLI libraries directly (Cobra subcommands) and re-names `velero` output to `oadp` at runtime. Admin commands — `backup`, `restore`, `schedule`, `backup-location`, `snapshot-location`, `describe`, `get`, `delete`, `must-gather`, `setup` — are all provided this way with minimal custom code.

## Detailed Design
Admin commands are imported from the Velero package directly:

```go
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backup"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/restore"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/schedule"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/backuplocation"
	"github.com/vmware-tanzu/velero/pkg/cmd/cli/version"
)
```

CLI Examples (oadp-1.4)
```sh
oc oadp setup
oc oadp backup create <name> --include-namespaces <ns>
oc oadp backup describe <name>
oc oadp backup logs <name>
oc oadp backup delete <name>
oc oadp restore create --from-backup <name>
oc oadp schedule create <name> --schedule "0 1 * * *"
oc oadp backup-location get
oc oadp must-gather
```

## Alternatives Considered
An alternative considered was creating a CLI from scratch rather than using a plugin. Using an existing `oc` plugin avoids distribution complexity and integrates naturally with the OpenShift toolchain.

Aliasing velero was another option. This does not work well for users who want a consistent `oc oadp` experience or who do not have direct access to the velero binary.

## Security Considerations
Security is controlled entirely by OpenShift RBAC. Cluster-admin operations require the appropriate Velero RBAC (ability to create `backups.velero.io` cluster-wide). The plugin surfaces `Unauthorized` errors when permissions are insufficient. There is no privilege escalation in the plugin itself.

## Compatibility
This branch (`oadp-1.4`) targets OADP 1.4 / Velero 1.14 / OCP 4.17–4.18. The `go.mod` pins the `openshift/velero` and `openshift/oadp-operator` forks to the `oadp-1.4` release line.

## Future Work
Non-admin backup and restore (`oc oadp nonadmin …`) and NABSL management are implemented on the `oadp-1.5` branch and later. When OADP 1.4 clusters upgrade to 1.5, users can switch to the `oadp-1.5` CLI branch to access those commands.
