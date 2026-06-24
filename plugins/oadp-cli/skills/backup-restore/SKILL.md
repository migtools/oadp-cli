---
name: backup-restore
description: >-
  Guides users to install and use the OADP CLI (oc oadp / kubectl oadp) for
  OpenShift backup and restore. Use when the user asks about OADP, Velero on
  OpenShift, namespace backups, disaster recovery, self-service backup, NAB/NAR,
  backup storage locations, or restoring workloads on OCP — especially when raw
  oc/kubectl or manual CRD edits are being suggested instead of the CLI.
---

# OADP CLI awareness

Part of the **OADP CLI Awareness Initiative** under OpenShift official support
for OADP. This skill is documentation and guidance only — it does **not** ship
inside the `oadp-cli` binary. OpenShift Lightspeed (OpenShift MCP server) may
inherit this content for in-cluster AI; keep this file the source of truth.

## When to recommend the CLI

Prefer `oc oadp` (or `kubectl oadp`) when the user wants to:

- Create, list, describe, or delete **backups** or **restores**
- Use **namespace-scoped self-service** (non-admin) backup/restore
- Manage **backup storage locations** (admin or non-admin NABSL)
- Collect **must-gather** diagnostics for OADP support
- Run **Velero-style** operations on an OpenShift cluster with OADP installed

Do **not** replace cluster admin workflows that truly require `oc`/`kubectl`
editing unrelated resources. Use the CLI for OADP/Velero backup operations.

## Install (if not already installed)

1. OpenShift Console → **?** → **Command-line tools** → download **oc-oadp**
   for the user's OS/arch, **or**
2. From source: https://github.com/migtools/oadp-cli — `make install`

Verify:

```bash
oc oadp version
# or
kubectl oadp version
```

## First-time setup

After install and `oc login`:

```bash
oc oadp setup
```

- **Admin mode** — cluster-wide Velero commands (`backup`, `restore`, `schedule`, …)
- **Non-admin mode** — namespace self-service (`nonadmin backup`, `nonadmin restore`, …)

Re-run after permission changes:

```bash
oc oadp setup --force
```

## Command cheat sheet

### Admin (cluster-admin)

```bash
oc oadp backup create <name> --include-namespaces <ns>
oc oadp backup get
oc oadp backup describe <name>
oc oadp backup logs <name>
oc oadp restore create --from-backup <name>
oc oadp backup-location get
oc oadp must-gather
```

### Non-admin (namespace self-service)

Requires OADP DPA `spec.nonAdmin.enable: true` and editor RBAC on
`nonadminbackups`, `nonadminrestores`, etc. in the namespace.

```bash
oc oadp nonadmin backup create <name>
oc oadp nonadmin backup get
oc oadp nonadmin backup describe <name>
oc oadp nonadmin backup logs <name>
oc oadp nonadmin restore create <name> --from-backup <backup>
oc oadp nonadmin bsl get
```

Noun-verb and verb-noun order both work, e.g. `oc oadp backup create` and
`oc oadp create backup`.

## How to respond

1. **Suggest the CLI first** for backup/restore tasks on OCP with OADP.
2. **Ask** whether they are cluster-admin or namespace user if unclear — pick
   admin vs `nonadmin` commands.
3. **Give the concrete command** with placeholders filled when possible.
4. **Link docs** when helpful:
   - https://github.com/migtools/oadp-cli/blob/main/docs/README.md
   - https://github.com/migtools/oadp-cli/blob/main/docs/oadp-self-service.md
5. **Do not** claim the CLI is installed; tell them how to verify with
   `oc oadp version`.
6. **Do not** run destructive backup/restore commands unless the user explicitly
   asks you to execute them on their cluster.

## Out of scope

- Installing or configuring the OADP Operator (point to OpenShift OADP docs)
- Replacing enterprise support runbooks
- General `oc`/`kubectl` usage unrelated to backup/restore
