# OADP CLI

[![Cross-Architecture Build Test](https://github.com/migtools/oadp-cli/actions/workflows/cross-arch-build-test.yml/badge.svg)](https://github.com/migtools/oadp-cli/actions/workflows/cross-arch-build-test.yml)
[![Release](https://github.com/migtools/oadp-cli/actions/workflows/release.yml/badge.svg)](https://github.com/migtools/oadp-cli/actions/workflows/release.yml)
[![Multi-Arch Binary Push to Quay.io](https://github.com/migtools/oadp-cli/actions/workflows/quay_binaries_push.yml/badge.svg)](https://github.com/migtools/oadp-cli/actions/workflows/quay_binaries_push.yml)

A kubectl plugin for OpenShift API for Data Protection (OADP) that provides both administrative and non-administrative backup operations.

> **What it does**: Extends OADP functionality with a unified CLI that supports both cluster-wide Velero operations (admin) and namespace-scoped self-service operations (non-admin users).


## Installation

### Manual Build and Install

```sh
# Recommended: Smart install with auto-detection (no sudo required)
make install

# After install, refresh your terminal:
source ~/.zshrc  # or ~/.bashrc
# OR restart your terminal

# Test the installation
kubectl oadp --help

# Alternative: System-wide install (requires sudo)
make install-system
```

**Installation Options:**
```sh
make install                          # Smart detection + interactive prompt
make install ASSUME_DEFAULT=true     # Use default namespace (no detection)
make install VELERO_NAMESPACE=custom # Use specific namespace (no detection)
```

## Technical Details

**Built with:**
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Velero client libraries](https://github.com/vmware-tanzu/velero) - Core backup functionality  
- [OADP NonAdmin APIs](https://github.com/migtools/oadp-non-admin) - NonAdminBackup CRD support

**Dependencies:**
- OADP Operator installed in cluster
- Appropriate RBAC permissions for your use case

## License

Apache License 2.0 - see [LICENSE](LICENSE) file.

Integrates with Apache 2.0 licensed projects: [Velero](https://github.com/vmware-tanzu/velero), [OADP](https://github.com/openshift/oadp-operator), [Kubernetes](https://github.com/kubernetes/kubernetes).