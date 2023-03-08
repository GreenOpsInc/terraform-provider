package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"greenops.io/terraform-provider-greenops/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}