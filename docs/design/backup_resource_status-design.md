# Backup resource status (rollup) design

_High-level design for oadp-cli commands that answer: after a backup finishes (especially `PartiallyFailed`), which logical units failed, and why?_

**Tracks:** [#269](https://github.com/migtools/oadp-cli/issues/269) ([OADP-8697](https://redhat.atlassian.net/browse/OADP-8697)), [#270](https://github.com/migtools/oadp-cli/issues/270), [#271](https://github.com/migtools/oadp-cli/issues/271)

**Related:**
- Upstream Velero: [velero-io/velero#10503](https://github.com/velero-io/velero/issues/10503) (sparse per-item backup outcomes)
- oadp-operator designs: [backup outcomes](https://github.com/openshift/oadp-operator/blob/design/backup-outcomes-observability/docs/design/backup_outcomes-design.md), [resource rollup](https://github.com/openshift/oadp-operator/blob/design/backup-outcomes-observability/docs/design/backup_resource_rollup-design.md) ([PR #2442](https://github.com/openshift/oadp-operator/pull/2442))

## Problem

Velero reports backup outcomes at the **raw Kubernetes object** level. Operators usually care about a **logical unit** (e.g. a VirtualMachine, StatefulSet, or labeled app).

When a namespace backup ends `PartiallyFailed`, the `Backup` CR only exposes a count. Detail is scattered across object-storage metadata. At scale (hundreds or thousands of VMs), that is not operable.

**Rollup** means: take per-item outcomes and **group them under a parent** (via `ownerReferences` or a label), then combine into one status per group (`failed` if any child failed).

## What we have to work with

oadp-cli already reaches the data we need. No new Velero server APIs are required for a first version.

### Already in this repo

| Capability | Where | Relevance |
|------------|--------|-----------|
| Admin backup commands | Velero CLI packages via `cmd/root.go` | `backup describe` / logs patterns to follow |
| Non-admin backup describe | `cmd/non-admin/backup/describe.go` | Already fetches BSL metadata via DownloadRequest |
| Shared download helper | `cmd/shared/download.go` (`ProcessDownloadRequest`) | Reusable path for results / volumeinfo / resource-list / itemoperations |

Non-admin describe already pulls (when `--details`):

- `BackupResourceList`
- `BackupResults`
- `BackupVolumeInfos`
- `BackupItemOperations`

Admin path can use the same Velero client/DownloadRequest patterns the Velero CLI already uses.

### Velero artifacts (per backup)

| Source | Role for rollup |
|--------|-----------------|
| `Backup.status` | Phase, error **counts** only — not enough alone |
| `resource-list.json.gz` | Inventory (which objects were in the backup) |
| `results.gz` | Sync errors/warnings as **strings** (parse carefully; `name:` can mean different kinds) |
| `volumeinfo.json.gz` | Structured PVC volume results |
| `itemoperations.json.gz` | Async ops; may disagree with `results` ([velero#9377](https://github.com/vmware-tanzu/velero/issues/9377)) |
| Backup tarball manifests | `ownerReferences` when the live namespace is gone |

### Live cluster (optional)

When the source namespace still exists, read `ownerReferences` from the API. Prefer that over tarball parsing when available.

## High-level approach

```
Backup metadata (existing)
        │
        ▼
  Parse / merge into per-item outcomes
  (resource-list + results + volumeinfo + itemoperations)
        │
        ▼
  Group by strategy (owner:Kind | owner | label:key)
        │
        ▼
  CLI output (failures-first table, -o json/yaml)
```

1. **Library** (`pkg/backuprollup` or similar) — merge metadata into per-item outcomes; walk ownership / labels to roll up. Unit-testable with recorded fixtures.
2. **CLI** — thin commands on top of that library.

### Proposed commands

```
oc oadp backup resource-status <backup> --group-by=owner:VirtualMachine
oc oadp backup vm-status <backup>   # convenience alias for VM grouping
```

Default UX: summary line + **failures only** (output size tracks failure count).

### Grouping strategies (v1)

| Strategy | Meaning |
|----------|---------|
| `owner:<Kind>` | Walk `controller: true` ownerRefs until Kind matches |
| `owner` | Walk to topmost owner |
| `label:<key>` | Flat group by label value (e.g. Helm instance) |

VM support is a **preset** (`owner:VirtualMachine`), not bespoke KubeVirt vol parsing.

## Phasing

| Phase | Deliverable |
|-------|-------------|
| **1** | Library: merge existing metadata + owner/label rollup; fixtures from real `PartiallyFailed` backups |
| **2** | CLI: `resource-status` + `vm-status` |
| **Later** | Consume upstream structured outcomes ([#10503](https://github.com/velero-io/velero/issues/10503)) when available — thin the string parser |

## Non-goals

- Changing Velero backup orchestration or phase semantics
- Upstream Velero API in v1 (track [#10503](https://github.com/velero-io/velero/issues/10503); prototype with what we have)
- Persisted in-cluster CRD for rollup status
- Live progress monitoring during backup (separate from post-backup status)
- Automatic retry / remediation of failed VMs

## Open questions

- Admin vs non-admin: one command tree with shared library, or non-admin first given DownloadRequest is already wired?
- Conflict rules when `results` and `itemoperations` disagree (prefer itemoperations for async paths?)
- Exact `skipped` semantics (Velero does not always mark skips explicitly)

## Success criteria ([OADP-8697](https://redhat.atlassian.net/browse/OADP-8697))

- [ ] Per-VM (or other `--group-by`) status: succeeded / failed / skipped + reason
- [ ] Programmatic output (`-o json` / `-o yaml`)
- [ ] Failures-first default view usable at scale
- [ ] Works when source namespace is gone (tarball ownerRefs fallback)
