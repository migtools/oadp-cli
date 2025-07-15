package main

import (
	"github.com/migtools/oadp-cli/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Import authentication plugins for cloud providers
)

func main() {
	cmd.Execute()
}
