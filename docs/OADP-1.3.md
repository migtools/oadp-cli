# OADP CLI — `oadp-1.3` branch

Developer guide for the **OADP 1.3 release line** of `migtools/oadp-cli`.

## Purpose

Branch **`oadp-1.3`** builds the kubectl plugin (`kubectl oadp`) and the **download server** (`Containerfile.download`) against the same dependency stack as **OADP Operator 1.3** (Velero 1.12, k8s v0.25.x).

| OpenShift (typical) | OADP operator | oadp-cli branch |
|---------------------|---------------|-----------------|
| 4.15 – 4.16         | 1.3           | **`oadp-1.3`**  |
| 4.17 – 4.18         | 1.4           | **`oadp-1.4`**  |
| 4.19 – 4.21         | 1.5           | **`oadp-1.5`**  |
| 4.22+               | 1.6           | **`oadp-1.6`**  |

Reference: [OADP PARTNERS.md](https://github.com/openshift/oadp-operator/blob/oadp-dev/PARTNERS.md)

## Non-admin commands not included

**OADP 1.3 does not ship the non-admin controller.**
The `RELATED_IMAGE_NON_ADMIN_CONTROLLER` image is absent from the 1.3 operator and the NAC controller requires OADP operator 1.5 or later.

As a result, this CLI branch does not include nonadmin commands.
Use **OADP 1.5** or later for namespace-scoped self-service backup and restore.

## CLI features on `oadp-1.3`

- Admin Velero commands: `backup`, `restore`, `schedule`, `backup-location`, `snapshot-location`, `get`, `describe`, `delete`, `create`
- `must-gather`, `setup`, `client`, `completion`

## `oadp-1.3` vs `oadp-1.4`

### Code changes

| File | Change |
|------|--------|
| `go.mod` / `go.sum` | OADP 1.3 dependency pins; `legacy-aws` plugin support removed |
| `cmd/root.go` | Removed `DiscoveryClient()` method from `timeoutFactory` (not in Velero 1.12 `Factory` interface) |
| `Makefile` | `VERSION ?= oadp-1.3` |
| `Containerfile.download` | `OADP_VERSION=oadp-1.3` |

### Dependency comparison

| Component | `oadp-1.3` | `oadp-1.4` |
|-----------|------------|------------|
| **Go (module minimum)** | `1.25.8` | `1.25.8` |
| **Velero module** | `v1.12.4` | `v1.14.1-rc.1` |
| **openshift/velero replace** | `…20260707…e3ed032b0b82` (oadp-1.3 tip) | `…20260526…ea5de9549ff4` (oadp-1.4 tip) |
| **controller-runtime** | `v0.12.2` | `v0.17.2` |
| **k8s.io/client-go** | `v0.25.6` | `v0.29.0` |
| **k8s.io/api / apimachinery** | `v0.25.6` | `v0.29.2` |
| **kopia replace** | `github.com/migtools/kopia` @ oadp-1.3 | `github.com/project-velero/kopia` |
| **external-snapshotter** | `v4.2.0` | `v6.x` |
| **kustomize/cmd/config** | `v0.10.9` (downgraded) | `v0.21.0` |
| **kube-openapi replace** | `v0.0.0-20220803…` (pinned for gnostic compat) | not needed |
| **legacy-aws plugin** | not present | present |
| **Version string** | `oadp-1.3` | `oadp-1.4` |

### Why `DiscoveryClient()` was removed

The `timeoutFactory` struct in `cmd/root.go` on `oadp-1.4` implements the `AggregatedDiscoveryInterface` method added to `k8s.io/client-go/discovery` in Kubernetes 1.28+.
Velero 1.12 (`oadp-1.3`) targets Kubernetes 1.25 and its `client.Factory` interface does not include this method, so the implementation was removed to allow compilation.

### Why `kube-openapi` is pinned

`k8s.io/client-go v0.25.6` depends on `github.com/google/gnostic` (the original package).
Newer versions of `k8s.io/kube-openapi` import `github.com/google/gnostic-models` (the split package), causing a type mismatch at compile time.
The `replace` directive in `go.mod` pins `kube-openapi` to a version that still uses the original `gnostic` package, resolving the build error.

## Build and test

```bash
git checkout oadp-1.3
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
podman build -f Containerfile.download -t oadp-cli-oadp-1.3:local .
```

## Branch workflow

| Branch | Use |
|--------|-----|
| **`oadp-1.3`** | OADP 1.3 releases, OCP 4.15–4.16 |
| **`oadp-1.4`** | OADP 1.4 releases, OCP 4.17–4.18 |
| **`oadp-1.5`** | OADP 1.5 releases, OCP 4.19–4.21 |
| **`oadp-dev`** | Development / next release |

Bugfixes for 1.3: branch from **`oadp-1.3`**, cherry-pick forward to **`oadp-1.4`** / **`oadp-1.5`** / **`oadp-dev`** as maintainers direct.

## Related repos

| Repo | Branch | Role |
|------|--------|------|
| [openshift/oadp-operator](https://github.com/openshift/oadp-operator) | `oadp-1.3` | Operator, Velero, DPA |
| **migtools/oadp-cli** | **`oadp-1.3`** | This plugin + download server |
