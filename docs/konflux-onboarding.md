# Onboarding a New OADP Component to Konflux

This document describes the end-to-end process for adding a new image component to the OADP release stream, built via Konflux.

## Overview

OADP ships as a set of container images built by Red Hat's Konflux build system. The build configuration lives in [openshift-eng/ocp-build-data](https://github.com/openshift-eng/ocp-build-data) on product-specific branches (e.g. `oadp-1.5`, `oadp-1.6`). Source code is pulled from private mirrors in the `openshift-priv` GitHub org.

```
Source repo (migtools/oadp-cli)
        |
        | auto-mirrored
        v
Private mirror (openshift-priv/migtools-oadp-cli)
        |
        | Konflux pulls source + builds image
        v
Build config (ocp-build-data oadp-1.6 branch)
        |
        | image pushed to
        v
Delivery repo (registry.redhat.io/oadp/oadp-cli-rhel9)
        |
        | released via RPA
        v
Stage / Prod registries
```

## Repositories Involved

| Repository | Purpose | Where |
|---|---|---|
| **Source repo** (e.g. `migtools/oadp-cli`) | Component source code + `konflux.Dockerfile` | GitHub |
| **openshift/release** | Whitelist config for `openshift-priv` mirror creation | [GitHub](https://github.com/openshift/release) |
| **openshift-priv/migtools-\<repo\>** | Private mirror of source repo (ART's midstream) | GitHub (restricted) |
| **openshift-eng/ocp-build-data** | Build configuration (image YAMLs, group config, streams) | [GitHub](https://github.com/openshift-eng/ocp-build-data) |
| **releng/pyxis-repo-configs** | Delivery/Comet repo definitions | [GitLab (internal)](https://gitlab.cee.redhat.com/releng/pyxis-repo-configs) |
| **Stage/Prod RPAs** | Release pipeline access for the image | Internal |

## Step-by-Step Process

### Step 1: Create the openshift-priv mirror

**Repo:** `openshift/release`
**File:** `core-services/openshift-priv/_whitelist.yaml`

Add your repo under the appropriate org section. For `migtools` repos:

```yaml
  migtools:
    - oadp-cli          # <-- add your repo here
    - oadp-non-admin
    # ...
```

**Reference PR:** [openshift/release#68955](https://github.com/openshift/release/pull/68955) (ART-14080: whitelist MTC & OADP repos)

After the PR merges, the mirror at `openshift-priv/migtools-<repo-name>` is auto-created within ~3-4 hours. ART can then perform a test build.

**Naming convention:** repos under `migtools/` are mirrored as `openshift-priv/migtools-<repo-name>`.

### Step 2: Prepare the source repo

Your source repo needs a `konflux.Dockerfile` that follows the standard OADP build pattern:

```dockerfile
FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25 AS builder

COPY . /workspace
WORKDIR /workspace

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GOOS=linux go build -mod=readonly -a -tags strictfipsruntime \
    -o /workspace/bin/<your-binary> ./<your-cmd-path>/

FROM registry.redhat.io/ubi9/ubi:latest

RUN dnf -y install openssl && dnf -y reinstall tzdata && dnf clean all

COPY --from=builder /workspace/bin/<your-binary> /usr/local/bin/<your-binary>
COPY LICENSE /licenses/

LABEL description="<description>"
LABEL io.k8s.description="<description>"
LABEL io.k8s.display-name="<display name>"
LABEL io.openshift.tags="oadp,migration,backup"
LABEL summary="<summary>"
```

Key requirements:
- **Builder image:** `brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.25`
- **FIPS compliance:** `CGO_ENABLED=1`, `GOEXPERIMENT=strictfipsruntime`, `-tags strictfipsruntime`
- **Build flags:** `-mod=readonly` (hermetic builds don't allow network fetches)
- **Runtime base:** `registry.redhat.io/ubi9/ubi:latest`
- **Use `dnf`** not `microdnf` -- ART auto-replaces `dnf` with `microdnf` at build time via ocp-build-data modifications
- **License:** Copy `LICENSE` to `/licenses/`

### Step 3: Create the delivery repo

**Repo:** `gitlab.cee.redhat.com/releng/pyxis-repo-configs`
**Path:** `products/oadp/oadp.yaml`

For OADP, this is the first delivery repo created post-Konflux migration, so the `products/oadp/` directory needs to be created. Reference `products/ocp/` or `products/mta/` for the file format.

Add an entry for your new image (e.g. `oadp/oadp-cli-rhel9`).

### Step 4: Add to stage and prod RPAs

Add the new image as a component in the stage and prod RPA configurations. This allows the image to be released through the pipeline to stage and production registries.

### Step 5: Add build config to ocp-build-data

**Repo:** `openshift-eng/ocp-build-data`
**Branch:** `oadp-1.6` (or whichever release stream)

#### 5a. Create image config

Create `images/<your-component>.yml`:

```yaml
cachito:
  packages:
    gomod:
      - path: .
content:
  source:
    dockerfile: konflux.Dockerfile
    git:
      branch:
        target: oadp-1.6
      url: git@github.com:openshift-priv/migtools-<your-repo>.git
      web: https://github.com/migtools/<your-repo>
    modifications:
      - action: replace
        match: "dnf -y"
        replacement: "microdnf -y"
      - action: replace
        match: "dnf clean all"
        replacement: "microdnf clean all"
distgit:
  component: <your-component>-container
  branch: rhaos-{MAJOR}.{MINOR}-rhel-9
delivery:
  delivery_repo_names:
    - oadp/<your-delivery-repo>
for_payload: false
enabled_repos:
  - rhel-9-appstream-rpms
  - rhel-9-baseos-rpms
from:
  builder:
    - stream: rhel-9-golang
  member: base-rhel9
name: oadp/<your-delivery-repo>
owners:
  - oadp-maintainers@redhat.com
dependents:
  - oadp-operator
konflux:
  cachi2:
    lockfile:
      rpms:
        - tzdata
jira:
  project: OADP
  component: <your-component>-container
```

#### 5b. Add public_upstreams mapping

In `group.yml`, add the mapping between the private mirror and public repo:

```yaml
public_upstreams:
  # ... existing entries ...
  - private: "https://github.com/openshift-priv/migtools-<your-repo>"
    public:  "https://github.com/migtools/<your-repo>"
```

### Step 6: Wire into the operator bundle

Add the new image as a `relatedImage` in the OADP operator's CSV/bundle so it ships when the operator is installed.

## Build System Details

### How Konflux builds work for OADP

- **No `.tekton/` or `.konflux/` directories needed** in source repos -- pipelines are managed externally by ART tooling
- **No cachi2 lockfiles needed in-repo** -- cachi2 dependency resolution is handled externally based on ocp-build-data config
- **Hermetic builds:** network is blocked during build. Dependencies are prefetched by Konflux/cachi2 before the build starts
- **Multi-arch:** images are built for x86_64, aarch64, ppc64le, s390x

### ocp-build-data branch structure

OADP uses product-level branches, not per-component branches:

```
ocp-build-data/
  oadp-1.5/          <-- all OADP 1.5 components
  oadp-1.6/          <-- all OADP 1.6 components
```

Each branch contains:
- `group.yml` -- shared config (Go version, arches, RHEL repos, Konflux settings, OCP targets)
- `streams.yml` -- base image references (golang builder, UBI, ose-cli)
- `images/` -- per-component build configs
- `releases.yml` -- release assembly config (may be empty)

### Key config in group.yml

```yaml
konflux:
  cachi2:
    enabled: true
    gomod_version_patch: true
    lockfile:
      force: true
  sast:
    enabled: true
  network_mode: hermetic
```

## References

- [ART Konflux onboarding docs](https://art-docs.engineering.redhat.com/konflux/onboard-external-operators/) (internal, requires cert)
- [openshift/release#68955](https://github.com/openshift/release/pull/68955) -- reference PR for whitelist additions
- [openshift-eng/ocp-build-data](https://github.com/openshift-eng/ocp-build-data) -- build configuration repo
