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
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	kbclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/migtools/oadp-cli/cmd/shared"
	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/client"
	"github.com/vmware-tanzu/velero/pkg/cmd"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/flag"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCreateCommand(f client.Factory) *cobra.Command {
	o := NewCreateOptions()

	c := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a non-admin backup storage location",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(o.Complete(args, f))
			cmd.CheckError(o.Validate(c, args, f))
			cmd.CheckError(o.Run(c, f))
		},
		Example: `  # Create a non-admin backup storage location for AWS
  kubectl oadp nonadmin bsl create my-storage \
    --provider aws \
    --bucket my-velero-bucket \
    --credential cloud-credentials=cloud \
    --region us-east-1

  # Create with prefix for organizing backups
  kubectl oadp nonadmin bsl create my-storage \
    --provider aws \
    --bucket my-velero-bucket \
    --prefix velero-backups \
    --credential cloud-credentials=cloud \
    --region us-east-1

  # Create with custom credential key
  kubectl oadp nonadmin bsl create my-storage \
    --provider aws \
    --bucket my-velero-bucket \
    --credential my-secret=service-account-key \
    --region us-east-1

  # View the YAML without creating the resource
  kubectl oadp nonadmin bsl create my-storage \
    --provider aws \
    --bucket my-bucket \
    --credential cloud-credentials=cloud \
    --region us-east-1 \
    -o yaml`,
	}

	o.BindFlags(c.Flags())
	output.BindFlags(c.Flags())
	output.ClearOutputFlagDefault(c)

	return c
}

type CreateOptions struct {
	Name       string
	Namespace  string
	Provider   string
	Bucket     string
	Prefix     string
	Credential flag.Map
	Region     string
	Config     map[string]string
	client     kbclient.WithWatch
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Credential: flag.NewMap(),
		Config:     make(map[string]string),
	}
}

func (o *CreateOptions) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.Provider, "provider", "", "Storage provider (required). Examples: aws, azure, gcp")
	flags.StringVar(&o.Bucket, "bucket", "", "Object storage bucket name (required)")
	flags.StringVar(&o.Prefix, "prefix", "", "Prefix for backup objects in the bucket")
	flags.Var(&o.Credential, "credential", "The credential to be used by this location as a key-value pair, where the key is the Kubernetes Secret name, and the value is the data key name within the Secret. Required, one value only.")
	flags.StringVar(&o.Region, "region", "", "Storage region (required for some providers like AWS)")
	flags.StringToStringVar(&o.Config, "config", nil, "Additional provider-specific configuration (key=value pairs)")
}

func (o *CreateOptions) Complete(args []string, f client.Factory) error {
	o.Name = args[0]

	// Create client with full scheme including NonAdmin and Velero types
	client, err := shared.NewClientWithFullScheme(f)
	if err != nil {
		return err
	}

	o.client = client

	// Get the current namespace
	currentNS, err := shared.GetCurrentNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine current namespace: %w", err)
	}
	o.Namespace = currentNS

	return nil
}

func (o *CreateOptions) Validate(c *cobra.Command, args []string, f client.Factory) error {
	if o.Provider == "" {
		return fmt.Errorf("--provider is required")
	}
	if o.Bucket == "" {
		return fmt.Errorf("--bucket is required")
	}
	if len(o.Credential.Data()) == 0 {
		return errors.New("--credential is required")
	}
	if len(o.Credential.Data()) > 1 {
		return errors.New("--credential can only contain 1 key/value pair")
	}

	return nil
}

func (o *CreateOptions) Run(c *cobra.Command, f client.Factory) error {
	// Build config map
	config := make(map[string]string)
	if o.Region != "" {
		config["region"] = o.Region
	}
	// Add any additional config provided via --config flag
	for k, v := range o.Config {
		config[k] = v
	}

	// Create the NABSL
	nabsl := &nacv1alpha1.NonAdminBackupStorageLocation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: o.Namespace,
		},
		Spec: nacv1alpha1.NonAdminBackupStorageLocationSpec{
			BackupStorageLocationSpec: &velerov1.BackupStorageLocationSpec{
				Provider: o.Provider,
				Config:   config,
				StorageType: velerov1.StorageType{
					ObjectStorage: &velerov1.ObjectStorageLocation{
						Bucket: o.Bucket,
						Prefix: o.Prefix,
					},
				},
			},
		},
	}

	// Set credential from user-provided key-value pair
	for secretName, secretKey := range o.Credential.Data() {
		nabsl.Spec.BackupStorageLocationSpec.Credential = builder.ForSecretKeySelector(secretName, secretKey).Result()
		break
	}

	if printed, err := output.PrintWithFormat(c, nabsl); printed || err != nil {
		return err
	}

	err := o.client.Create(context.Background(), nabsl)
	if err != nil {
		return err
	}

	fmt.Printf("NonAdminBackupStorageLocation %q created successfully.\n", nabsl.Name)
	fmt.Printf("The controller will create a request for admin approval.\n")
	fmt.Printf("Use 'kubectl oadp nonadmin bsl request get' to view auto-created requests.\n")
	return nil
}
