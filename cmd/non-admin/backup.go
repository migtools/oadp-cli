package nonadmin

/*
Copyright 2017 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewNonAdminFactory creates a client factory for NonAdminBackup operations
// that uses the current kubeconfig context namespace instead of hardcoded openshift-adp
func NewNonAdminFactory() client.Factory {
	// Don't set a default namespace, let it use the kubeconfig context
	cfg := client.VeleroConfig{}
	return client.NewFactory("oadp-nonadmin-cli", cfg)
}

func NewCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "na-backup",
		Short: "Work with non-admin backups",
		Long:  "Work with non-admin backups",
	}

	c.AddCommand(
		NewCreateCommand(f, "create"),
		// NewGetCommand(f, "get"),
		// NewLogsCommand(f),
		// NewDescribeCommand(f, "describe"),
		// NewDownloadCommand(f),
		// NewDeleteCommand(f, "delete"),
	)

	return c
}
