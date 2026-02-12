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

package output

import (
	"bytes"
	"fmt"
	"io"

	nacv1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerooutput "github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// NonAdminScheme returns a runtime.Scheme with NonAdmin types registered
func NonAdminScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	// Add NonAdmin types
	if err := nacv1alpha1.AddToScheme(scheme); err != nil {
		panic(fmt.Sprintf("failed to add NonAdmin types to scheme: %v", err))
	}

	// Add Velero types for compatibility
	if err := velerov1api.AddToScheme(scheme); err != nil {
		panic(fmt.Sprintf("failed to add Velero types to scheme: %v", err))
	}

	return scheme
}

// BindFlags wraps Velero's BindFlags to add output flags
func BindFlags(flags *pflag.FlagSet) {
	velerooutput.BindFlags(flags)
}

// ClearOutputFlagDefault wraps Velero's ClearOutputFlagDefault
func ClearOutputFlagDefault(cmd *cobra.Command) {
	velerooutput.ClearOutputFlagDefault(cmd)
}

// PrintWithFormat prints the provided object in the format specified by
// the command's flags. This is a custom implementation for nonadmin commands
// that supports NonAdmin CRD types (NonAdminBackup, NonAdminRestore, etc.)
func PrintWithFormat(c *cobra.Command, obj runtime.Object) (bool, error) {
	format := velerooutput.GetOutputFlagValue(c)
	if format == "" {
		return false, nil
	}

	switch format {
	case "json", "yaml":
		return printEncoded(obj, format)
	case "table":
		// Table format is not supported by this function
		// The caller should handle table printing
		return false, nil
	}

	return false, errors.Errorf("unsupported output format %q; valid values are 'table', 'json', and 'yaml'", format)
}

func printEncoded(obj runtime.Object, format string) (bool, error) {
	// assume we're printing obj
	toPrint := obj

	if meta.IsListType(obj) {
		list, _ := meta.ExtractList(obj)
		if len(list) == 1 {
			// if obj was a list and there was only 1 item, just print that 1 instead of a list
			toPrint = list[0]
		}
	}

	encoded, err := encode(toPrint, format)
	if err != nil {
		return false, err
	}

	fmt.Println(string(encoded))

	return true, nil
}

func encode(obj runtime.Object, format string) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := encodeTo(obj, format, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeTo(obj runtime.Object, format string, w io.Writer) error {
	encoder, err := encoderFor(format, obj)
	if err != nil {
		return err
	}

	return errors.WithStack(encoder.Encode(obj, w))
}

func encoderFor(format string, obj runtime.Object) (runtime.Encoder, error) {
	var encoder runtime.Encoder

	// Use NonAdminScheme instead of Velero's scheme
	codecFactory := serializer.NewCodecFactory(NonAdminScheme())

	desiredMediaType := fmt.Sprintf("application/%s", format)
	serializerInfo, found := runtime.SerializerInfoForMediaType(codecFactory.SupportedMediaTypes(), desiredMediaType)
	if !found {
		return nil, errors.Errorf("unable to locate an encoder for %q", desiredMediaType)
	}
	if serializerInfo.PrettySerializer != nil {
		encoder = serializerInfo.PrettySerializer
	} else {
		encoder = serializerInfo.Serializer
	}

	if !obj.GetObjectKind().GroupVersionKind().Empty() {
		return encoder, nil
	}

	// Use the appropriate GroupVersion for encoding
	// For NonAdmin types, use nacv1alpha1.GroupVersion
	encoder = codecFactory.EncoderForVersion(encoder, nacv1alpha1.GroupVersion)
	return encoder, nil
}
