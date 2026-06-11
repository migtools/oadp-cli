# OADP CLI — `oadp-1.5` branch

Developer guide for the **OADP 1.5 release line** of `migtools/oadp-cli`.

## Purpose

Branch **`oadp-1.5`** builds the kubectl plugin (`kubectl oadp`) and the **download server** (`Containerfile.download`) against the same dependency stack as **OADP Operator 1.5** (Velero 1.16, `oadp-non-admin` oadp-1.5, etc.).

It is branched from **`oadp-1.6`** with **dependency and version-string changes only**. No CLI commands were removed for this backport.

| OpenShift (typical) | OADP operator | oadp-cli branch |
|---------------------|---------------|-----------------|
| 4.19 – 4.21         | 1.5           | **`oadp-1.5`**  |
| 4.22+               | 1.6           | **`oadp-1.6`**  |

Reference: [OADP PARTNERS.md](https://github.com/openshift/oadp-operator/blob/oadp-dev/PARTNERS.md)

## `oadp-1.5` vs `oadp-1.6`

### Code changes

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Downgrade module pins to OADP 1.5 line |
| `Makefile` | `VERSION ?= oadp-1.5` |
| `Containerfile.download` | `OADP_VERSION=oadp-1.5` |

### Dependency comparison

| Component | `oadp-1.5` (this branch) | `oadp-1.6` |
|-----------|---------------------------|------------|
| **Velero module** | `v1.16.0` | `v1.18.1` |
| **openshift/velero replace** | `…20260526…87a03c3d2c32` | `…20260601…af1b4409d3db` |
| **controller-runtime** | `v0.19.3` | `v0.21.0` |
| **oadp-non-admin** | `aad3132759e1` (oadp-1.5) | `54d1934bbb11` (oadp-1.6) |
| **openshift/oadp-operator replace** | not pinned in go.mod (via `oadp-non-admin`) | not pinned in go.mod |
| **kopia replace** | removed | `github.com/migtools/kopia` |
| **Go** | `1.25.8` | `1.25.0` |
| **k8s.io/client-go** (direct) | `v0.33.11` | `v0.33.11` |
| **Version string** | `oadp-1.5` | `oadp-1.6` |


### CLI features: same as `oadp-1.6`

The plugin on **`oadp-1.5`** exposes the **same commands** as **`oadp-1.6`**, including:

- Admin Velero commands (`backup`, `restore`, `schedule`, …)
- `nonadmin` (NAB / NAR)
- `nabsl-request`
- `must-gather`, `setup`, `client`, `completion`

### OADP product differences (platform, not oadp-cli-only)

Use **OADP 1.5 operator** on the cluster with this CLI. OADP **1.6** adds platform changes this branch does **not** target, for example:

- **Velero 1.18** (vs 1.16 on 1.5)
- **Restic uploader removed** for new backups on the 1.6 line (Kopia-only for new FS backups)
- Planned **automatic operator upgrades** (file-based catalogs) on 1.6
- New operator CRDs on 1.6 (e.g. VM file restore) — not required for this CLI backport

See [OADP operator wiki](https://github.com/openshift/oadp-operator/wiki/Latest-OADP-product-release-updates) and release notes for product-level detail.

## Build and test

```bash
git checkout oadp-1.5
make build
make install ASSUME_DEFAULT=true
kubectl oadp version    
```

Run the full dev test suite:

```bash
# If nonadmin=true in config.json, admin-command tests fail.
kubectl oadp client config set nonadmin=false

make test    
make lint    # optional
```

Local image build:

```bash
podman build -f Containerfile.download -t oadp-cli-oadp-1.5:local .
```

## Branch workflow

| Branch | Use |
|--------|-----|
| **`oadp-1.5`** | OADP 1.5 releases, OCP 4.19–4.21 |
| **`oadp-1.6`** | OADP 1.6 releases, OCP 4.22+ |
| **`oadp-dev`** | Development / next release |

Bugfixes for 1.5: branch from **`oadp-1.5`**, cherry-pick to **`oadp-1.6`** / **`oadp-dev`** as maintainers direct.

## Related repos

| Repo | Branch | Role |
|------|--------|------|
| [openshift/oadp-operator](https://github.com/openshift/oadp-operator) | `oadp-1.5` | Operator, Velero, DPA |
| [migtools/oadp-non-admin](https://github.com/migtools/oadp-non-admin) | `oadp-1.5` | NAB CRDs / controllers |
| **migtools/oadp-cli** | **`oadp-1.5`** | This plugin + download server |
