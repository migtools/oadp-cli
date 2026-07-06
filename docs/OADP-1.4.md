# OADP CLI — `oadp-1.4` branch

Developer guide for the **OADP 1.4 release line** of `migtools/oadp-cli`.

## Purpose

Branch **`oadp-1.4`** builds the kubectl plugin (`kubectl oadp`) and the **download server** (`Containerfile.download`) against the same dependency stack as **OADP Operator 1.4** (Velero 1.14, k8s v0.29.x).

| OpenShift (typical) | OADP operator | oadp-cli branch |
|---------------------|---------------|-----------------|
| 4.17 – 4.18         | 1.4           | **`oadp-1.4`**  |
| 4.19 – 4.21         | 1.5           | **`oadp-1.5`**  |
| 4.22+               | 1.6           | **`oadp-1.6`**  |

Reference: [OADP PARTNERS.md](https://github.com/openshift/oadp-operator/blob/oadp-dev/PARTNERS.md)

## Non-admin commands not included

**OADP 1.4 does not ship the non-admin controller.**
The `RELATED_IMAGE_NON_ADMIN_CONTROLLER` image is absent from the 1.4 operator and the NAC controller requires OADP operator 1.5 or later.

As a result, this CLI branch does not include nonadmin commands.
Use **OADP 1.5** or later for namespace-scoped self-service backup and restore.

## CLI features on `oadp-1.4`

- Admin Velero commands: `backup`, `restore`, `schedule`, `backup-location`, `snapshot-location`, `get`, `describe`, `delete`, `create`
- `must-gather`, `setup`, `client`, `completion`

## `oadp-1.4` vs `oadp-1.5`

### Code changes

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | OADP 1.4 dependency pins; `oadp-non-admin` removed |
| `cmd/non-admin/` | Removed (not available in 1.4) |
| `cmd/nabsl-request/` | Removed (not available in 1.4) |
| `cmd/shared/download.go` | Removed (was for non-admin download requests) |
| `Makefile` | `VERSION ?= oadp-1.4` |
| `Containerfile.download` | `OADP_VERSION=oadp-1.4` |

### Dependency comparison

| Component | `oadp-1.4` | `oadp-1.5` |
|-----------|---------------------------|------------|
| **Velero module** | `v1.14.1-rc.1` | `v1.16.0` |
| **openshift/velero replace** | `…20260526…ea5de9549ff4` (oadp-1.4 tip) | `…20260526…87a03c3d2c32` (oadp-1.5 tip) |
| **controller-runtime** | `v0.17.2` | `v0.19.3` |
| **k8s.io/client-go** (direct) | `v0.29.0` | `v0.33.11` |
| **k8s.io/api / apimachinery** | `v0.29.2` | `v0.33.11` |
| **oadp-non-admin** | removed | `aad3132759e1` (oadp-1.5) |
| **kopia replace** | `github.com/project-velero/kopia` | Inherited from oadp-non-admin @ oadp-1.5 |
| **Go** | `1.25.8` | `1.25.8` |
| **Version string** | `oadp-1.4` | `oadp-1.5` |

## Build and test

```bash
git checkout oadp-1.4
make build
make install ASSUME_DEFAULT=true
kubectl oadp version
```

Run the full dev test suite:

```bash
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
| **migtools/oadp-cli** | **`oadp-1.4`** | This plugin + download server |
