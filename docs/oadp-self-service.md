# OADP Self Service Overview

## Overview

OADP Self Service enables non-administrator users to perform backup and restore operations in their authorized namespaces without requiring 
cluster-wide administrator privileges. This feature provides secure self-service data protection capabilities while maintaining proper administrator
controls, restrictions and enforcements over the user's backup and restore operations.

### Key Benefits

- Allows users to perform namespace-scoped backup and restore operations
- Provides users with secure access to backup logs and status information
- Enables users to create dedicated backup storage locations with user owned buckets and credentials
- Maintains cluster administrator control over non-administrator operations through 
restrictions and enforcements


## OADP Self Service Details

OADP self-service introduces a significant change to backup and restore operations in OpenShift. Previously, only cluster administrators could perform these operations. 
Now, regular OpenShift users can perform backup and restore operations within their authorized namespaces.  
This is achieved through custom resources that securely manage these operations while maintaining proper access controls and visibility. 
The self-service functionality is implemented in a way that ensures users can only operate within their assigned namespaces and permissions,
while cluster administrators maintain overall control through restrictions and enforcements.
 
### Glossary of terms

* **NAB**   - NonAdminBackup. A custom resource that users directly create to request a velero backup of the namespace from which the NAB object is created.  
* **NAR**   - NonAdminRestore. A custom resource that users directly create to request a velero restore of the namespace from which the NAR object is created.  
* **NAC**   - NonAdminController. A controller that validates the NAB and NAR objects and creates the velero backup and restore and related objects. The NAC is essentially a proxy between non admin users and velero.
* **NABSL** - NonAdminBackupStorageLocation. A custom resource that users directly create to request a velero backup storage location.  Users can use object storage that is specifically created for their project, deliniated from other users and projects.
* **NADR**  - NonAdminDownloadRequest. A custom resource that users directly create to request a velero backup download.  Users will be provided with a secured URL to download details regarding the backup or restore.

### Cluster Administrator Setup

Install and configure the OADP operator according to the documentation and your requirements.

To enable OADP Self-Service the DPA spec must the spec.nonAdmin.enable field to true.

```
  nonAdmin:
    enable: true
```

Once the OADP DPA is reconciled the cluster administrator should see the non-admin-controller running in the openshift-adp namespace.  The Openshift users without cluster admin rights will now be able to create NAB or NAR objects and related objects in their namespace to create a backup or restore.

## OpenShift User Instructions

Prior to OpenShift users taking advantage of OADP self-service feature the OpenShift cluster administrator must have completed the following prerequisite steps:

* The OADP DPA has been configured to support self-service
* The cluster administrator has created the users 
  * account 
  * namespace
  * namespace privileges, e.g. namespace admin.
  * optionally the cluster administrator can create a NABSL for the user.

### OpenShift self-service required permissions:

Ensure users have appropriate permissions in its namespace.  Users must have editor roles for the following objects in their namespace.
  * nonadminbackups.oadp.openshift.io                                
  * nonadminbackupstoragelocations.oadp.openshift.io                
  * nonadminrestores.oadp.openshift.io     
  * nonadmindownloadrequests.oadp.openshift.io                         

  For example
  ```yaml
    # config/rbac/nonadminbackup_editor_role.yaml
    - apiGroups:
        - oadp.openshift.io
      resources:
        - nonadminbackups
        - nonadminrestores
        - nonadminbackupstoragelocations
        - nonadmindownloadrequests
      verbs:
        - create
        - delete
        - get
        - list
        - patch
        - update
        - watch
    - apiGroups:
        - oadp.openshift.io
      resources:
        - nonadminbackups/status
      verbs:
        - get
    # config/rbac/nonadminrestore_editor_role.yaml
    - apiGroups:
      - oadp.openshift.io
      resources:
      - nonadminrestores
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - oadp.openshift.io
      resources:
      - nonadminrestores/status
      verbs:
      - get
  ```

  **note** Users will not be able to edit the NABSLs that led to the creation of the BSL. They will only have the ability to create new NABSLs. The NAC controller will disallow such edit and inform user within the condition of the edited NABSL object.

## Self-Service workflow

Non Cluster Administrators can utilize OADP self-service by creating NonAdminBackup (NAB) and NonAdminRestore (NAR) objects in the namespace to be backed up or restored.  A NonAdminBackup is an OpenShift custom resource that securely facilitates the creation, status and lifecycle of a Velero Backup custom resource.  

```mermaid
sequenceDiagram
    participant User
    participant NAB as NonAdminBackup
    participant NAC as NonAdminController
    participant VB as Velero Backup

    User->>NAB: Creates NonAdminBackup CR
    NAB->>NAC: Detected by controller
    NAC-->>NAB: Validates backup request
    NAC->>VB: Creates Velero Backup CR
    VB-->>NAB: Status updates
    NAB-->>User: View backup status
```

![nab-backup-workflow](https://hackmd.io/_uploads/BJz4bEbKyx.jpg)

For the most part one can think of a NonAdminBackup and a Velero Backup in very much the same way.  Both objects specify a velero backup and how the backup should be executed.  There are a few differences to keep in mind when creating a NonAdminBackup.

1. The NonAdminBackup creates the Velero Backup CR instance in a secure way that limits the users access.
2. A user cannot specify the namespace that will be backed up.  The namespace from which the NAB object is created is the defined namespace to be backed up.
3. In addition to the creation of the Velero Backup the NonAdminBackup object's main purpose is to track the status of the Velero Backup in a secure and clear way.

### NonAdminBackup NAB Example:

```
apiVersion: oadp.openshift.io/v1alpha1
kind: NonAdminBackup
metadata:
  name: mybackup-1
  namespace: nacuser1
spec:
  backupSpec:
    snapshotMoveData: true
```

Once created the NAB will look similar to the following:

```
apiVersion: oadp.openshift.io/v1alpha1
kind: NonAdminBackup
metadata:
  creationTimestamp: "2025-02-21T20:57:35Z"
  finalizers:
  - nonadminbackup.oadp.openshift.io/finalizer
  generation: 2
  name: mybackup-1
  namespace: nacuser1  <--- The namespace that NAC controller will set on the Velero Backup object to backup
  resourceVersion: "20714121"
  uid: 93effb39-9762-4d04-8e9e-194ebe6b9b31
spec:
  backupSpec:
    snapshotMoveData: true  
status:
  conditions:
  - lastTransitionTime: "2025-02-21T20:57:35Z"
    message: backup accepted  <--- The NAC controller reconciled the NAB object and created the Velero Backup object
    reason: BackupAccepted
    status: "True"
    type: Accepted
  - lastTransitionTime: "2025-02-21T20:57:35Z"
    message: Created Velero Backup object
    reason: BackupScheduled
    status: "True"
    type: Queued
  phase: Created  <--- The NAB object is in the Created phase
  queueInfo:
    estimatedQueuePosition: 0  <--- The NAB object status in the queue, once complete it is set to 0
  veleroBackup:
    nacuuid: nacuser1-mybackup-1-588ce989-387e-4352-b04a-1fd6b7712370  <--- The NAC controller created the Velero Backup object and set the nacuuid
    name: nacuser1-mybackup-1-588ce989-387e-4352-b04a-1fd6b7712370 <--- The associated Velero Backup name
    namespace: openshift-adp
    status:  <--- The status of the Velero backup object displayed by the NAB object
      backupItemOperationsAttempted: 3
      backupItemOperationsCompleted: 3
      completionTimestamp: "2025-02-21T20:59:28Z"
      expiration: "2025-03-23T20:57:35Z"
      formatVersion: 1.1.0
      hookStatus: {}
      phase: Completed  <--- The Velero backup object is in the Completed phase, successful
      progress:
        itemsBackedUp: 57
        totalItems: 57
      startTimestamp: "2025-02-21T20:57:35Z"
      version: 1
```
The complete nonAdminBackup resource definition can be found here: [NonAdminBackup CRD](https://github.com/openshift/oadp-operator/blob/master/bundle/manifests/oadp.openshift.io_nonadminbackups.yaml)

### NonAdminRestore NAR Example:

```
apiVersion: oadp.openshift.io/v1alpha1
kind: NonAdminRestore
metadata:
  name: example
  namespace: nacuser1
spec: 
  restoreSpec:
    backupName: mybackup-1 <-- references the NAB object, not the Velero backup object
```

Once created the NAR will look similar to the following:

```
apiVersion: oadp.openshift.io/v1alpha1
kind: NonAdminRestore
metadata:
  creationTimestamp: "2025-02-21T21:12:54Z"
  finalizers:
  - nonadminrestore.oadp.openshift.io/finalizer
  generation: 2
  name: example
  namespace: nacuser1
  resourceVersion: "20719136"
  uid: 0f1d8346-d8be-4621-8d67-0877f15e82fb
spec:
  restoreSpec:
    backupName: mybackup-1
    hooks: {}
    itemOperationTimeout: 0s
status:
  conditions:
  - lastTransitionTime: "2025-02-21T21:12:54Z"
    message: restore accepted  <--- The NAC controller reconciled the NAR object and created the Velero Restore object
    reason: RestoreAccepted
    status: "True"
    type: Accepted
  - lastTransitionTime: "2025-02-21T21:12:54Z"
    message: Created Velero Restore object  <--- The NAC controller created the Velero Restore object
    reason: RestoreScheduled
    status: "True"
    type: Queued
  phase: Created  <--- The NAR object is in the Created phase
  queueInfo:
    estimatedQueuePosition: 0  <--- The NAR object status in the queue, once complete it is set to 0
  veleroRestore:
    nacuuid: nacuser1-example-b844be97-7ee4-4702-91b8-ffc84697675a  <--- The NAC controller created the Velero Restore object and set the nacuuid
    name: nacuser1-example-b844be97-7ee4-4702-91b8-ffc84697675a  <--- The associated Velero Restore name
    namespace: openshift-adp
    status:
      completionTimestamp: "2025-02-21T21:12:57Z"
      hookStatus: {}
      phase: Completed  <--- The Velero restore object is in the Completed phase, successful
      progress:
        itemsRestored: 54
        totalItems: 54
      startTimestamp: "2025-02-21T21:12:55Z"
      warnings: 0
```
The complete nonAdminRestore resource definition can be found here: [NonAdminRestore CRD](https://github.com/openshift/oadp-operator/blob/master/bundle/manifests/oadp.openshift.io_nonadminrestores.yaml)

### NonAdminBackupStorageLocation NABSL:
Cluster administrators can gain efficiencies by delegating backup and restore operations to OpenShift users. It is recommended that cluster administrators carefully manage the NABSL to conform to any company policies, compliance requirements, etc.  

1. **Direct Creation**: Cluster administrators can create NABSLs directly for non-admin users.
2. **Approval Workflow**: Cluster administrators can enable an approval process where:
   - Users create a NABSL, which triggers the creation of a NonAdminBackupStorageLocationRequest object in the openshift-adp namespace.
   - Administrators review and either approve or reject these requests. Once approved, a Velero BSL is created in the openshift-adp namespace, and the user is notified of the approval on the NABSL status. If rejected, the status of the NABSL is updated to reflect the rejection.
   - Administrators can also revoke previously approved NABSL, which results in the removal of the Velero BSL, and the user is notified of the rejection. This is achieved by modifying the approve field back to pending or reject.
   - This is an opt-in feature and must be explicitly enabled by the cluster administrator.
3. **Automatic Approval**: Users create NABSL from the user namespace, these are automatically approved when nonAdmin.requireApprovalForBSL is set to false or not set.

For security purposes it is recommended that cluster administrators use either the direct creation or the approval workflow.  The automatic approval option is less secure as it does not require administrator review.  It should also be noted that updating the NABSL after the initial creation will NOT change the associated Velero BSL. Updating the NABSL is not supported for non-admin users.


To enable the approval workflow the DPA spec must be set as follows:
```
  nonAdmin:
    enable: true
    requireApprovalForBSL: true
```
Cluster administrators can view the NABSLApprovalRequest object in the openshift-adp namespace.

```
oc -n openshift-adp get NonAdminBackupStorageLocationRequests
```
Approval or rejection is accomplished by updating the NABSLApprovalRequest object.

```
spec:
  approvalDecision: reject  [accept, reject]
```

If approved both the NABSL and the BSL are created.  The NABSL is created in the users namespace, while the BSL is created in the OADP namespace, such as openshift-adp.

### User Creation of NABSL:

A non-admin user can create a NABSL in their namespace.

```
apiVersion: oadp.openshift.io/v1alpha1
kind: NonAdminBackupStorageLocation
metadata:
  name: nacuser1-nabsl
  namespace: nacuser1
spec: 
  backupStorageLocationSpec:
    config:
    checksumAlgorithm: ""
    profile: default
    region: us-west-2
    credential:
      key: cloud
      name: cloud-credentials
    objectStorage:
      bucket: bucket1uswest2
      prefix: velero
    provider: aws  
```

If the cluster administrator has enabled the requireApprovalForBSL flag then the NABSL will be in the Pending state until the cluster administrator approves the NABSL.



### NAB / NAR Status

#### Phase
The phase field is a simple one high-level summary of the lifecycle of the objects, that only moves forward. Once a phase changes, it can not return to the previous value.

| **Value** | **Description** |
|-----------|-----------------|
| New | *NonAdminBackup/NonAdminRestore* resource was accepted by the NAB/NAR Controller, but it has not yet been validated by the NAB/NAR Controller |
| BackingOff | *NonAdminBackup/NonAdminRestore* resource was invalidated by the NAB/NAR Controller, due to invalid Spec. NAB/NAR Controller will not reconcile the object further, until user updates it |
| Created | *NonAdminBackup/NonAdminRestore* resource was validated by the NAB/NAR Controller and Velero *Backup/restore* was created. The Phase will not have additional information about the *Backup/Restore* run |
| Deletion | *NonAdminBackup/NonAdminRestore* resource has been marked for deletion. The NAB/NAR Controller will delete the corresponding Velero *Backup/Restore* if it exists. Once this deletion completes, the *NonAdminBackup/NonAdminRestore* object itself will also be removed |





## Advanced Cluster Administrator Features

### Cluster Administrator Enforceable Spec Fields
There are several types of cluster scoped objects that non-admin users should not have access to backup or restore.  OADP self-service automatically excludes the following list of cluster scoped resources from being backed up or restored.

* SCCs
* ClusterRoles
* ClusterRoleBindings
* CRDs
* PriorityClasses
* virtualmachineclusterinstancetypes
* virtualmachineclusterpreferences

Cluster administrators may also enforce company or compliance policy by utilizing templated NABSL's, NAB's and NAR's that require fields values to be set and conform to the administrator defined policy.  Admin Enforceable fields are fields that the cluster administrator can enforce non cluster admin users to use. Restricted fields are automatically managed by OADP and cannot be modified by either administrators or users.

#### NABSL
The following NABSL fields are currently supported for template enforcement:

| **NABSL Field**            | **Admin Enforceable** | **Restricted** | **special case** |
|----------------------------|-----------------|----------------|-----------------|
| `backupSyncPeriod`         |                 |                | ⚠️ Must be set lower than the DPA.backupSyncPeriod and lower than the garbage collection period |
| `provider`                 |                 |                | ⚠️ special case |
| `objectStorage`            | ✅ Yes          |                |                 |
| `credential`               | ✅ Yes          |                |                 |
| `config`                   | ✅ Yes          |                |                 |
| `accessMode`               | ✅ Yes          |                |                 |
| `validationFrequency`      | ✅ Yes          |                |                 |
| `default`                  |                 |                | ⚠️ Must be false or empty |

For example if the cluster administrator wanted to mandate that all NABSL's used a particular aws s3 bucket.

```
spec:
  config:
    checksumAlgorithm: ""
    profile: default
    region: us-west-2
  credential:
    key: cloud
    name: cloud-credentials
  objectStorage:
    bucket: my-company-bucket <---
    prefix: velero
  provider: aws
```
The DPA spec must be set in the following way: 

```
nonAdmin:
  enable: true
  enforceBSLSpec:
    config:             <--- entire config must match expected NaBSL config
      checksumAlgorithm: ""
      profile: default
      region: us-west-2
    objectStorage:  <--- all of the objectStorage options must match expected NaBSL options
      bucket: my-company-bucket
      prefix: velero 
    provider: aws
```

#### Restricted NonAdminBackups

In the same sense as the NABSL, cluster administrators can also restrict the NonAdminBackup spec fields to ensure the backup request conforms to the administrator defined policy.  Most of the backup spec fields can be restricted by the cluster administrator, below is a table of reference for the current implementation.


| **Backup Spec Field**                                  | **Admin Enforceable** | **Restricted** | **special case** |
|--------------------------------------------|--------------|--------------------------|-----------------|
| `csiSnapshotTimeout`                       | ✅ Yes       |                          |                 |
| `itemOperationTimeout`                     | ✅ Yes       |                          |                 |
| `resourcePolicy`                           | ✅ Yes       |                          | ⚠️ Non-admin users can specify the config-map that admins created in OADP Operator NS(Admins enforcing this value be a good alternative here), they cannot specify their own configmap as its lifecycle handling is not currently managed by NAC controller |
| `includedNamespaces`                       | ❌ No        | ✅ Yes                   | ⚠️ Admins cannot enforce this because it does not make sense for a cluster wide non-admin backup setting, we have validations in place such that only the NS admins NS in included in the NAB spec.                 |
| `excludedNamespaces`                       | ✅ Yes       | ✅ Yes                   | ⚠️ This spec is restricted for non-admin users and hence not enforceable by admins                |
| `includedResources`                        | ✅ Yes       |                          |                 |
| `excludedResources`                        | ✅ Yes       |                          |                 |
| `orderedResources`                         | ✅ Yes       |                          |                 |
| `includeClusterResources`                  | ✅ Yes       |                          | ⚠️ Non-admin users can only set this spec to false if they want, all other values are restricted, similar rule for admin enforcement regarding this spec value.  |
| `excludedClusterScopedResources`           | ✅ Yes       |                          |                 |
| `includedClusterScopedResources`           | ✅ Yes       |                          | ⚠️ This spec is restricted and non-enforceable, only empty list is acceptable  |
| `excludedNamespaceScopedResources`         | ✅ Yes       |                          |                 |
| `includedNamespaceScopedResources`         | ✅ Yes       |                          |                 |
| `labelSelector`                            | ✅ Yes       |                          |                 |
| `orLabelSelectors`                         | ✅ Yes       |                          |                 |
| `snapshotVolumes`                          | ✅ Yes       |                          |                 |
| `storageLocation`                          |              |                          | ⚠️ Can be empty (implying default BSL usage) or needs to be an existing NABSL |
| `volumeSnapshotLocations`                  |              |                          | ⚠️ Not supported for non-admin users, default will be used if needed |
| `ttl`                                      | ✅ Yes       |                          |                 |
| `defaultVolumesToFsBackup`                 | ✅ Yes       |                          |                 |
| `snapshotMoveData`                         | ✅ Yes       |                          |                 |
| `datamover`                                | ✅ Yes       |                          |                 |
| `uploaderConfig.parallelFilesUpload`       | ✅ Yes       |                          |                 |
| `hooks`                                    |              |                          | ⚠️ special case |

An example enforcement set in the DPA spec to enforce the 
  * ttl to be set to "158h0m0s"
  * snapshotMoveData to be set to true

```
  nonAdmin:
    enable: true
    enforcedBackupSpec.ttl: "158h0m0s"
    enforcedBackupSpec.snapshotMoveData: true
```

#### Restricted NonAdminRestore NAR

NonAdminRestores spec fields can also be restricted by the cluster administrator.  The following NAR spec fields are currently supported for template enforcement:

| **Field**                     | **Admin Enforceable** | **Restricted**    | **special case** |
|-------------------------------|--------------|--------------------|-----------------|
| `backupName`                  | ❌ No        |                    |                 |
| `scheduleName`                | ❌ No        | ✅ Yes             | ⚠️  not supported for non-admin users, we don't have non-admin backup schedule API as of now.  |
| `itemOperationTimeout`        | ✅ Yes       |                    |                 |
| `uploaderConfig`              | ✅ Yes       |                    |                 |
| `includedNamespaces`          | ❌ No        | ✅ Yes             | ⚠️ restricted for non-admin users and hence non-enforceable by admins  |
| `excludedNamespaces`          | ❌ No        | ✅ Yes             | ⚠️  restricted for non-admin users and hence non-enforceable by admins |
| `includedResources`           | ✅ Yes       |                    |                 |
| `excludedResources`           | ✅ Yes       |                |                 |
| `restoreStatus`               | ✅ Yes       |                |                 |
| `includeClusterResources`     | ✅ Yes       |                |                 |
| `labelSelector`               | ✅ Yes       |                |                 |
| `orLabelSelectors`            | ✅ Yes       |                |                 |
| `namespaceMapping`            | ❌ No        | ✅ Yes         | ⚠️ restricted for non-admin users and hence non-enforceable by admins                |
| `restorePVs`                  | ✅ Yes       |                |                 |
| `preserveNodePorts`           | ✅ Yes       |                |                 |
| `existingResourcePolicy`      |              |                | ⚠️ special case |
| `hooks`                       |              |                | ⚠️ special case |
| `resourceModifers`            |              |                | ⚠️ Non-admin users can specify the config-map that admins created in OADP Operator NS(Admins enforcing this value be a good alternative here), they cannot specify their own configmap as its lifecycle handling is not currently managed by NAC controller | 



## OpenShift Console and OADP Self-Service

At the time of writing OADP self-service objects can only be created in the OpenShift Console via the API Explorer.

Navigate to: Administrator -> Home -> API Explorer -> Filter on `NonAdmin`. Choose the object you wish to create.

   * NonAdminBackup
   * NonAdminRestore
   * NonAdminBackupStorageLocation
   * NonAdminDownloadRequest

Click on instances and the create button.


## Unsupported features of OADP regarding self-service
 * Cross Cluster or Migrations are NOT supported by self-service.  This type of OADP operation is only supported for the cluster administrator.
 * non-admin VSL's are not supported.  The VSL created by the cluster-admin in DPA would be the only VSL non-admin users can employ.
 * ResourceModifiers and Volume policies are not supported for non-admin user backup and restore operations.
 * Backup and restore logs via NonAdminDownloadRequest is not supported for default BSL's.  If the cluster administrator would 
 like users to have access to logs, NonAdminBackupStorageLocation's must be created for the non-admin users.
 

## Security Considerations for Cluster Administrators

  * By enabling self-service, cluster administrators will expose the name of the namespace where OADP is running via the backup logs.  non-admin users are NOT granted any access to the OADP operator namespace.
  * The nonadmin controller will not allow users to set includeClusterResources in a backup or restore.  This is to prevent a scenario where a non-admin user would attempt to restore a cluster scoped resource to a namespace. OADP's backup policy is to automatically include cluster scoped resources like PV's that are associated with the namespace being backed up. Additionally cluster administrators can template an enforcement to `excludedClusterScopedResources` to prevent cluster scoped resources from being backed up.

---

# CLI Implementation: MVP Approach

## Overview

The OADP CLI implements a Minimal Viable Product (MVP) approach for the `nonadmin backup create` and `nonadmin restore create` commands. This approach uses **struct embedding** from Velero's CreateOptions to reduce code duplication while maintaining clear control over which features are exposed to non-admin users.

## Implementation Pattern: Struct Embedding

Both commands follow the same pattern established by Velero's CLI:

```go
type CreateOptions struct {
    *velero.CreateOptions  // Embed Velero's full feature set

    // NAB/NAR-specific fields
    Name             string
    Force            bool  // backup only
    client           kbclient.WithWatch
    currentNamespace string
}
```

### Benefits

- ✅ **Reduces Code Duplication** - No need to manually declare 30+ fields
- ✅ **Automatic Velero Compatibility** - Struct updates flow automatically
- ✅ **Clear Control Gate** - `BindFlags()` controls what's exposed to users
- ✅ **Easy Enhancement Path** - Add flags incrementally as features mature
- ✅ **Type Safety** - Reuses Velero's validated type definitions

## Backup Create: MVP Flags (13 total)

### Resource Filtering (2)
- `--include-resources` (default: `["*"]`)
- `--exclude-resources`

### Label Selection (2)
- `--selector` / `-l`
- `--or-selector`

### Cluster Resources (1)
- `--include-cluster-resources` (users can only set to false)

### Timing & Storage (4)
- `--ttl`
- `--storage-location`
- `--csi-snapshot-timeout`
- `--item-operation-timeout`

### Snapshot Control (3)
- `--snapshot-volumes`
- `--snapshot-move-data`
- `--default-volumes-to-fs-backup`

### Control Flags (1)
- `--force` (bypass storage-location requirement)

### Flags Removed from Original Implementation

**Restricted (API limitations):**
- `--include-namespaces` - Auto-managed by NAC
- `--exclude-namespaces` - Restricted for non-admin
- `--include-cluster-scoped-resources` - Only empty list acceptable
- `--volume-snapshot-locations` - Not supported

**Not in MVP (future enhancements):**
- `--labels` / `--annotations` - Metadata
- `--from-schedule` - Requires schedule API
- `--ordered-resources` - Advanced feature
- `--data-mover` - Advanced feature
- `--resource-policies-configmap` - Advanced feature
- `--parallel-files-upload` - Performance tuning
- `--exclude-cluster-scoped-resources` - Advanced filtering
- `--include-namespace-scoped-resources` - Advanced filtering
- `--exclude-namespace-scoped-resources` - Advanced filtering

## Restore Create: MVP Flags (7 total)

### Core (3)
- `--backup-name` (required)
- `--include-resources`
- `--exclude-resources`

### Label Selection (2)
- `--selector` / `-l`
- `--or-selector`

### Cluster Resources (1)
- `--include-cluster-resources`

### Timing (1)
- `--item-operation-timeout`

## Documentation

Comprehensive README files document the MVP approach:

- **cmd/non-admin/backup/README.md** - Complete backup flag reference
- **cmd/non-admin/restore/README.md** - Complete restore flag reference

Each README includes:
- MVP flags table with descriptions
- Restricted flags table with reasons
- Future enhancement flags table
- Examples
- Architecture notes

## Migration from Previous Version

Users previously using non-MVP flags will need to:

1. Remove unsupported flags from backup/restore commands
2. For metadata (`--labels`/`--annotations`): Use kubectl to add after creation
3. For `--from-schedule`: Create from admin schedule spec directly

## Alignment with NAB/NAR API

The MVP flags directly correspond to the API restrictions tables shown above:

| CLI Approach | API Field Status |
|--------------|------------------|
| Not exposed | Restricted |
| MVP flag | Admin enforceable or unrestricted |
| Future enhancement | Admin enforceable |

This ensures the CLI accurately reflects the underlying API capabilities and restrictions.





 

