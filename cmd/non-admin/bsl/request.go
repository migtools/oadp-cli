/*
Copyright 2025 The OADP CLI Contributors.

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

package bsl

import (
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/velero/pkg/client"
)

// NewRequestCommand creates the "request" subcommand under bsl
func NewRequestCommand(f client.Factory) *cobra.Command {
	c := &cobra.Command{
		Use:   "request",
		Short: "Manage backup storage location approval requests",
		Long:  "View and manage approval requests for backup storage locations. Requests are automatically created when users create backup storage locations and require admin approval.",
	}

	c.AddCommand(
		NewRequestGetCommand(f),
		NewRequestDescribeCommand(f),
		NewRequestApproveCommand(f),
		NewRequestDenyCommand(f),
	)

	return c
}
