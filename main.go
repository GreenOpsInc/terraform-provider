package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"greenops.io/terraform-provider-greenops/provider"
)

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
