# OADP CLI — `oadp-1.4` branch

Developer guide for the **OADP 1.4 release line** of `migtools/oadp-cli`.

## Purpose

Branch **`oadp-1.4`** builds the kubectl plugin (`kubectl oadp`) and the **download server** (`Containerfile.download`) against the same dependency stack as **OADP Operator 1.4** (Velero 1.14, `oadp-non-admin` oadp-1.4, k8s v0.29.x, etc.).

It is branched from **`oadp-1.5`** with **dependency and version-string changes only**. No CLI commands were removed for this backport.

| OpenShift (typical) | OADP operator | oadp-cli branch |
|---------------------|---------------|-----------------|
| 4.17 – 4.18         | 1.4           | **`oadp-1.4`**  |
| 4.19 – 4.21         | 1.5           | **`oadp-1.5`**  |
| 4.22+               | 1.6           | **`oadp-1.6`**  |

Reference: [OADP PARTNERS.md](https://github.com/openshift/oadp-operator/blob/oadp-dev/PARTNERS.md)

## `oadp-1.4` vs `oadp-1.5`

### Code changes

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | Downgrade module pins to OADP 1.4 line |
| `Makefile` | `VERSION ?= oadp-1.4` |
| `Containerfile.download` | `OADP_VERSION=oadp-1.4` |

### Dependency comparison

| Component | `oadp-1.4` (this branch) | `oadp-1.5` |
|-----------|---------------------------|------------|
| **Velero module** | `v1.14.1-rc.1` | `v1.16.0` |
| **openshift/velero replace** | `…20260526…ea5de9549ff4` (oadp-1.4 tip) | `…20260526…87a03c3d2c32` (oadp-1.5 tip) |
| **controller-runtime** | `v0.17.2` | `v0.19.3` |
| **k8s.io/client-go** (direct) | `v0.29.0` | `v0.33.11` |
| **k8s.io/api / apimachinery** | `v0.29.2` | `v0.33.11` |
| **oadp-non-admin** | `30487177ef60` (oadp-1.4) | `aad3132759e1` (oadp-1.5) |
| **openshift/oadp-operator replace** | `…20260618…c10a22a15a36` (oadp-1.4 tip) | not pinned in go.mod (via `oadp-non-admin`) |
| **kopia replace** | `github.com/project-velero/kopia` | removed |
| **Go** | `1.25.8` | `1.25.8` |
| **Version string** | `oadp-1.4` | `oadp-1.5` |

### CLI features: same as `oadp-1.5`

The plugin on **`oadp-1.4`** exposes the **same commands** as **`oadp-1.5`**, including:

- Admin Velero commands (`backup`, `restore`, `schedule`, …)
- `nonadmin` (NAB / NAR)
- `nabsl-request`
- `must-gather`, `setup`, `client`, `completion`

`oadp-non-admin` `api/v1alpha1` types (including `NonAdminDownloadRequest`) are present on both the oadp-1.4 and oadp-1.5 branches; no CLI code paths were cut for this downport.

### OADP product differences (platform, not oadp-cli-only)

Use **OADP 1.4 operator** on the cluster with this CLI. OADP **1.5+** adds platform changes this branch does **not** target, for example:

- **Velero 1.16+** (vs 1.14 on 1.4)
- **k8s v0.31+** on the operator line (vs v0.29.x on 1.4)
- OCP **4.19+** requires OADP **1.5+** (1.4 is for OCP 4.17–4.18)
- Features and CRDs added after the 1.4 operator line

See [OADP operator wiki](https://github.com/openshift/oadp-operator/wiki/Latest-OADP-product-release-updates) and release notes for product-level detail.

## Build and test

```bash
git checkout oadp-1.4
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
podman build -f Containerfile.download -t oadp-cli-oadp-1.4:local .
```

## Branch workflow

| Branch | Use |
|--------|-----|
| **`oadp-1.4`** | OADP 1.4 releases, OCP 4.17–4.18 |
| **`oadp-1.5`** | OADP 1.5 releases, OCP 4.19–4.21 |
| **`oadp-1.6`** | OADP 1.6 releases, OCP 4.22+ |
| **`oadp-dev`** | Development / next release |

Bugfixes for 1.4: branch from **`oadp-1.4`**, cherry-pick forward to **`oadp-1.5`** / **`oadp-1.6`** / **`oadp-dev`** as maintainers direct.

## Related repos

| Repo | Branch | Role |
|------|--------|------|
| [openshift/oadp-operator](https://github.com/openshift/oadp-operator) | `oadp-1.4` | Operator, Velero, DPA |
| [migtools/oadp-non-admin](https://github.com/migtools/oadp-non-admin) | `oadp-1.4` | NAB CRDs / controllers |
| **migtools/oadp-cli** | **`oadp-1.4`** | This plugin + download server |

## Downstream delivery (OADP 1.4)

After this branch merges, **`oadp/oadp-cli-binaries-rhel9`** still needs the same downstream wiring as OADP 1.5:

1. **`openshift-eng/ocp-build-data`** @ `oadp-1.4` — add `oadp-cli-binaries-rhel9` build config
2. **`openshift/oadp-operator`** @ `oadp-1.4` — add image to bundle `image-references`

Konflux stage/prod advisories (`oadp-advisory-stage-1-4` / `oadp-advisory-prod-1-4`) may need a rebuild after those land.
