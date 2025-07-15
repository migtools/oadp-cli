# OADP CLI Functionality Analysis for E2E Testing

## **Step 1: CLI Functionality Analysis**

### **Available CLI Commands**

The OADP CLI provides both standard Velero commands and custom non-admin commands:

#### **Root Command Structure**
```
oadp
├── version          # Velero version command
├── backup           # Standard Velero backup operations
├── restore          # Standard Velero restore operations  
└── nonadmin         # Custom non-admin operations
    └── backup
        ├── create   # Create non-admin backup
        ├── delete   # Delete non-admin backup
        ├── describe # Describe non-admin backup
        └── logs     # Show logs for non-admin backup
```

### **E2E Testing Scope - Phase 1**

**🎯 FOCUSED SCOPE**: For initial e2e implementation, we'll focus on **admin commands** first:
- **`oadp backup create`** - Standard Velero backup creation
- **`oadp backup get`** - List backups (for verification)

**Why these commands?**
- ✅ Standard Velero commands - well-established and stable
- ✅ Work in `openshift-adp` namespace - no complex namespace setup needed
- ✅ Create real Kubernetes resources (`Backup` CRDs)
- ✅ Simpler RBAC - uses existing Velero factory
- ✅ Once working, provides foundation for non-admin commands

**Future expansion**:
- **Phase 2**: Add `oadp restore` commands
- **Phase 3**: Add `oadp nonadmin backup` commands (requires namespace creation and RBAC setup)

#### **Command Details**

1. **Root Command (`oadp`)**
   - **Purpose**: Entry point for all OADP CLI operations
   - **Default behavior**: Shows welcome message and help
   - **Factory**: Uses both Velero factory and NonAdmin factory

2. **Version Command (`oadp version`)**
   - **Purpose**: Display Velero version information
   - **Uses**: Standard Velero version command
   - **Factory**: Velero factory (openshift-adp namespace)

3. **Backup Command (`oadp backup`)**
   - **Purpose**: Standard Velero backup operations
   - **Uses**: Standard Velero backup command
   - **Factory**: Velero factory (openshift-adp namespace)

4. **Restore Command (`oadp restore`)**
   - **Purpose**: Standard Velero restore operations
   - **Uses**: Standard Velero restore command
   - **Factory**: Velero factory (openshift-adp namespace)

5. **NonAdmin Backup Create (`oadp nonadmin backup create`)**
   - **Purpose**: Create NonAdminBackup resources for non-admin users
   - **Key functionality**: Creates `NonAdminBackup` CRD in user's namespace
   - **Factory**: NonAdmin factory (uses current kubeconfig context namespace)

6. **NonAdmin Backup Delete (`oadp nonadmin backup delete`)**
   - **Purpose**: Delete NonAdminBackup resources by setting deletebackup field
   - **Key functionality**: Sets `spec.deleteBackup = true` on NonAdminBackup resource
   - **Factory**: NonAdmin factory (uses current kubeconfig context namespace)

---

### **Kubernetes Resources Created/Modified**

#### **1. Backup Create Command (`oadp backup create`)**
- **Resource Type**: `Backup` (Standard Velero CRD)
- **API Version**: `velero.io/v1`
- **Kind**: `Backup`
- **Namespace**: `openshift-adp` (from Velero factory)
- **Fields Set**:
  - `metadata.name`: Backup name (from CLI args)
  - `metadata.namespace`: `openshift-adp`
  - `metadata.labels`: Optional labels from `--labels` flag
  - `metadata.annotations`: Optional annotations from `--annotations` flag
  - `spec`: Velero BackupSpec containing:
    - `includedNamespaces`: From `--include-namespaces` flag
    - `includedResources`: From `--include-resources` flag
    - `excludedResources`: From `--exclude-resources` flag
    - `labelSelector`: From `--selector` flag
    - `snapshotVolumes`: From `--snapshot-volumes` flag
    - `ttl`: From `--ttl` flag
    - `storageLocation`: From `--storage-location` flag
    - And many other standard Velero backup options

#### **2. Backup Get Command (`oadp backup get`)**
- **Resource Type**: `Backup` (existing resources)
- **Operation**: LIST/GET
- **Effect**: Retrieves and displays backup resources from `openshift-adp` namespace

---

### **Current Test Structure Analysis**

#### **Existing Tests (`tests/` directory)**

1. **`common.go`** - Test utilities:
   - `buildCLIBinary()`: Builds CLI binary for testing
   - `testHelpCommand()`: Tests help command output
   - `cleanup()`: Removes test artifacts
   - Constants: `binaryName`, `buildTimeout`, `testTimeout`

2. **`help_test.go`** - Help command tests:
   - Tests all help commands work (`--help` and `-h`)
   - Verifies help output contains expected strings
   - Covers: root, version, backup, restore, nonadmin commands

3. **`build_test.go`** - Build and smoke tests:
   - Tests binary can be built successfully
   - Tests binary is executable
   - Basic smoke tests for all commands

4. **`main_test.go`** - Test suite setup:
   - Basic test suite configuration

---

### **Test Scenarios for E2E Testing**

#### **1. Admin Backup Create Tests (`oadp backup create`)**

**Basic Functionality**:
- ✅ Create backup with default settings
- ✅ Create backup with custom name
- ✅ Create backup with namespace inclusion (`--include-namespaces`)
- ✅ Create backup with resource filters (`--include-resources`, `--exclude-resources`)
- ✅ Create backup with label selector (`--selector`)
- ✅ Create backup with labels and annotations
- ✅ Create backup with volume snapshots disabled (`--snapshot-volumes=false`)
- ✅ Create backup with TTL setting (`--ttl`)
- ✅ Create backup with storage location (`--storage-location`)
- ✅ Create backup with wait flag (`--wait`)

**Advanced Functionality**:
- ✅ Create backup from schedule (`--from-schedule`)
- ✅ Create backup with ordered resources (`--ordered-resources`)
- ✅ Create backup with different resource scopes (cluster/namespace)
- ✅ Create backup with output formats (`-o yaml`, `-o json`)

**Error Scenarios**:
- ❌ Create backup without name (should fail)
- ❌ Create backup with invalid resource filters
- ❌ Create backup with conflicting selectors
- ❌ Create backup without cluster access
- ❌ Create backup without OADP operator
- ❌ Create backup without BackupStorageLocation

#### **2. Admin Backup Get Tests (`oadp backup get`)**

**Basic Functionality**:
- ✅ List all backups
- ✅ Get specific backup by name
- ✅ Get backup with output formats (`-o yaml`, `-o json`, `-o wide`)
- ✅ Get backup with label selector

**Error Scenarios**:
- ❌ Get non-existent backup
- ❌ Get without cluster access
- ❌ Get without permissions

#### **3. Integration Tests**

**Full Workflow**:
- ✅ Create backup → Verify resource created → Get backup → Verify details match
- ✅ Create backup with `--wait` → Verify completion status
- ✅ Create backup → Check BackupStorageLocation used
- ✅ Create multiple backups → Verify all listed correctly

**Resource Verification**:
- ✅ Verify Backup resource structure
- ✅ Verify backup spec matches CLI options
- ✅ Verify backup is in `openshift-adp` namespace
- ✅ Verify backup status progresses correctly

#### **4. Standard Velero Command Tests**

**Basic Functionality**:
- ✅ Version command works
- ✅ Backup command works (admin functionality)
- ✅ Restore command works (admin functionality)

---

### **Prerequisites for E2E Testing**

#### **Infrastructure Requirements**:
1. **Kubernetes Cluster**: Accessible via kubectl
2. **OADP Operator**: Installed and running
3. **BackupStorageLocation**: Configured and ready
4. **Test Namespace**: For creating test resources
5. **RBAC**: Proper permissions for NonAdminBackup operations

#### **Test Dependencies**:
1. **CLI Binary**: Built and accessible
2. **Kubernetes Client**: For resource verification
3. **Test Data**: Sample applications and resources
4. **Environment Variables**: For configuration

---

### **Expected Resource States**

#### **After Successful Backup Creation**:
```yaml
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: test-backup
  namespace: openshift-adp
spec:
  includedNamespaces:
  - my-app-namespace
  includedResources:
  - "*"
  storageLocation: default
  ttl: 720h0m0s
  # ... other spec fields
status:
  phase: Completed  # Eventually
  startTimestamp: "2023-01-01T10:00:00Z"
  completionTimestamp: "2023-01-01T10:05:00Z"
  expiration: "2023-01-31T10:00:00Z"
```

#### **After Successful Backup Get**:
```
NAME          STATUS      ERRORS   WARNINGS   CREATED              EXPIRES   STORAGE LOCATION   SELECTOR
test-backup   Completed   0        0          2023-01-01 10:00:00   29d       default            <none>
```

---

### **Test Environment Setup**

#### **Required Environment Variables**:
```bash
# Cluster access
KUBECONFIG=/path/to/kubeconfig

# Test configuration
TEST_NAMESPACE=oadp-e2e-test
CLI_BINARY_PATH=/path/to/oadp-cli
TEST_TIMEOUT=300s
```

#### **Test Data Requirements**:
1. **Sample Application**: Deployments, Services, ConfigMaps for backup testing
2. **Test Resources**: Various Kubernetes resources in test namespace
3. **Mock Data**: For testing different backup scenarios

---

### **Next Steps for Step 2**

Based on this analysis, Step 2 should focus on:

1. **Add Ginkgo dependencies** to `go.mod`
2. **Create basic test suite structure** with proper BeforeSuite/AfterSuite
3. **Implement prerequisite validation** (cluster, OADP operator, BackupStorageLocation)
4. **Create test utilities** for Kubernetes resource verification
5. **Set up test namespace management** (create test namespaces to backup)
6. **Implement utilities for Velero Backup resource verification**

**Key differences from non-admin approach**:
- ✅ **Simpler RBAC**: Uses existing `openshift-adp` namespace
- ✅ **Standard resources**: Works with standard Velero `Backup` CRDs
- ✅ **No complex namespace setup**: Uses Velero factory with fixed namespace
- ✅ **Established patterns**: Can follow existing Velero testing patterns

This analysis provides the foundation for implementing comprehensive e2e tests that validate both the CLI functionality and the resulting Kubernetes resource state. 