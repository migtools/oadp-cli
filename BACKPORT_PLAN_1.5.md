# OADP CLI Backport to 1.5 - Complete Plan

## Goal
Create a production-ready oadp-1.5 branch with the download server and CLI compiled against OADP 1.5 compatible dependencies.

## Dependency Versions for OADP 1.5

Based on research of the oadp-operator oadp-1.5 branch:

### Core Dependencies (Target for 1.5)
```
Go: 1.25.0 (toolchain: go1.25.8)
Velero: v1.16.0 (operator uses openshift/velero fork)
K8s clients: v0.31.3 (api, apimachinery, client-go)
controller-runtime: v0.19.3
```

### Current OADP 1.6 Dependencies (oadp-cli)
```
Go: 1.25.0
Velero: v1.18.1
K8s clients: v0.33.11
controller-runtime: v0.21.0
oadp-non-admin: v0.0.0-20260526195205-54d1934bbb11
```

### What Needs to Change
- ✅ Go version: Same (1.25.0)
- ⬇️ Velero: v1.18.1 → v1.16.0
- ⬇️ K8s clients: v0.33.11 → v0.31.3
- ⬇️ controller-runtime: v0.21.0 → v0.19.3
- ⬇️ oadp-non-admin: Find appropriate 1.5-compatible commit

## Step-by-Step Backport Process

### Phase 1: Preparation

#### 1.1 Save Current Work
```bash
cd /Users/niyancey/git/claudestuff/oadp-cli
git stash push -m "WIP: pr-194 namespace changes"
```

#### 1.2 Fetch Latest Changes
```bash
git fetch origin
git fetch origin --tags
```

#### 1.3 Determine Base Commit
**Question to answer:** What should oadp-1.5 branch from?

**Options:**
- **Option A:** Start from oadp-1.6 and downgrade deps
- **Option B:** Find a historical commit before 1.6 features
- **Option C:** Ask team for specific base commit

**Recommended:** Option A (start from oadp-1.6)

### Phase 2: Create oadp-1.5 Branch

#### 2.1 Create Branch
```bash
# If starting from oadp-1.6
git checkout -b oadp-1.5 origin/oadp-1.6

# Or from specific commit if team provides one
# git checkout -b oadp-1.5 <commit-hash>
```

#### 2.2 Verify Download Server Exists
```bash
ls -la Containerfile.download
ls -la cmd/downloads/
```

### Phase 3: Downgrade Dependencies

#### 3.1 Edit go.mod
```bash
# Edit go.mod manually or use these commands:

# Downgrade Velero
go get github.com/vmware-tanzu/velero@v1.16.0

# Downgrade K8s clients
go get k8s.io/api@v0.31.3
go get k8s.io/apimachinery@v0.31.3
go get k8s.io/client-go@v0.31.3

# Downgrade controller-runtime
go get sigs.k8s.io/controller-runtime@v0.19.3

# Find compatible oadp-non-admin version
# Need to check oadp-non-admin repo for 1.5 compatible commit
```

#### 3.2 Update oadp-non-admin Dependency
```bash
# Check oadp-non-admin repo for tags/commits around OADP 1.5 timeframe
# Likely need a commit from late 2024 or early 2025

# Placeholder - replace with actual commit:
go get github.com/migtools/oadp-non-admin@<commit-hash-for-1.5>
```

#### 3.3 Tidy Dependencies
```bash
go mod tidy
```

### Phase 4: Update Version Strings

#### 4.1 Update Containerfile.download
```bash
# Find the line with OADP_VERSION
# Change from: ARG OADP_VERSION=oadp-1.6
# Change to:   ARG OADP_VERSION=oadp-1.5
```

#### 4.2 Check for Other Version References
```bash
# Search for version strings
git grep -i "1\.6" | grep -v ".git" | grep -v "go.mod"
git grep -i "oadp-1\.6"
```

### Phase 5: Verify and Test

#### 5.1 Build Test
```bash
# Test CLI build
make build

# Test download server container
podman build -f Containerfile.download -t oadp-cli-download:1.5 .
```

#### 5.2 Run Unit Tests
```bash
make test
```

#### 5.3 Verify Download Server
```bash
# Run the download server locally
podman run -p 8080:8080 oadp-cli-download:1.5

# Visit http://localhost:8080
# Verify binaries are served with 1.5 version
```

### Phase 6: Commit and Push

#### 6.1 Commit go.mod Changes
```bash
git add go.mod go.sum
git commit -m "Downgrade dependencies to OADP 1.5 compatible versions

- Velero: v1.18.1 → v1.16.0
- K8s clients: v0.33.11 → v0.31.3
- controller-runtime: v0.21.0 → v0.19.3
- oadp-non-admin: updated to 1.5-compatible commit
"
```

#### 6.2 Commit Version String Changes
```bash
git add Containerfile.download
git commit -m "Set version to oadp-1.5 for release branch"
```

#### 6.3 Push to Remote
```bash
# Push to origin (if you have write access)
git push origin oadp-1.5

# Or push to your fork first
git push fork oadp-1.5
```

### Phase 7: CI/CD Verification

#### 7.1 Trigger Prow CI
The CI config at:
`https://github.com/openshift/release/blob/main/ci-operator/config/openshift/oadp-operator/openshift-oadp-operator-oadp-1.5.yaml`

Should automatically:
- Build the CLI
- Run `make test`
- Build container images
- Promote to registry

#### 7.2 Monitor CI
```bash
# Check Prow dashboard for your branch
# https://prow.ci.openshift.org/
```

## Potential Issues and Solutions

### Issue 1: API Incompatibilities
**Symptom:** Compile errors about missing methods/types
**Solution:** Check if code uses Velero 1.18 APIs not in 1.16, adapt code

### Issue 2: oadp-non-admin Version Unknown
**Symptom:** Can't find right commit hash
**Solution:** 
1. Check oadp-non-admin repo release history
2. Look for commits tagged with 1.5 or around Oct 2024
3. Ask the oadp-non-admin team

### Issue 3: Test Failures
**Symptom:** Unit tests fail after dependency downgrade
**Solution:** Update test mocks/fixtures to match 1.5 API versions

### Issue 4: Download Server Build Fails
**Symptom:** Container build errors
**Solution:** Check if any Go packages are incompatible, adjust Containerfile

## Checklist

- [ ] Save current work (git stash)
- [ ] Create oadp-1.5 branch
- [ ] Downgrade Velero to v1.16.0
- [ ] Downgrade K8s clients to v0.31.3
- [ ] Downgrade controller-runtime to v0.19.3
- [ ] Find and update oadp-non-admin version
- [ ] Run `go mod tidy`
- [ ] Update OADP_VERSION in Containerfile.download
- [ ] Search for other version references
- [ ] Build CLI (`make build`)
- [ ] Build download server container
- [ ] Run tests (`make test`)
- [ ] Test download server locally
- [ ] Commit go.mod changes
- [ ] Commit version changes
- [ ] Push to remote
- [ ] Monitor CI/CD builds
- [ ] Verify artifacts are published

## Next Steps After Push

1. Create PR if pushing to fork
2. Coordinate with OADP team for release
3. Update documentation referencing 1.5 CLI
4. Test with actual OADP 1.5 cluster (optional but recommended)

## Questions to Ask Your Team

1. **Base commit:** "Should oadp-1.5 branch from oadp-1.6 HEAD or a specific commit?"
2. **oadp-non-admin version:** "What commit/tag of oadp-non-admin is compatible with OADP 1.5?"
3. **Testing:** "Do I need to test against a live OADP 1.5 cluster or is CI enough?"
4. **Approval:** "Who needs to review/approve the oadp-1.5 branch creation?"

## Estimated Time

- **Setup and branch creation:** 15 minutes
- **Dependency downgrades:** 30 minutes
- **Build and test locally:** 30 minutes
- **Fix any issues:** 1-2 hours (variable)
- **Commit and push:** 15 minutes
- **CI/CD monitoring:** 30 minutes

**Total: 3-4 hours** (assuming no major blockers)
