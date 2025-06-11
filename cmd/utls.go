package cmd

import (
	"github.com/vmware-tanzu/velero/pkg/client"
)

// Default namespace for Velero resources
const veleroNamespace = "openshift-adp"

// newVeleroFactory creates a Velero client factory with the configured namespace.
func newVeleroFactory() client.Factory {
	cfg := client.VeleroConfig{
		client.ConfigKeyNamespace: veleroNamespace,
	}
	return client.NewFactory("oadp-velero-cli", cfg)
}
