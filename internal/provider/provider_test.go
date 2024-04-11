//go:build acceptance || flakey
// +build acceptance flakey

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"gitlab": providerserver.NewProtocol6WithError(New("acctest")()),
	}

	// testAccProtoV6MuxProviderFactories are used to instantiate a provider during acceptance testing
	// when both the SDK and Framework provider are required.
	// Only use these factories if you require SDK and Framework data sources / resources in the test.
	testAccProtoV6MuxProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"gitlab": func() (tfprotov6.ProviderServer, error) {
			providerServer, err := NewMuxedProviderServer(context.Background(), "acctest")
			if err != nil {
				return nil, fmt.Errorf("failed to create mux provider server for testing: %v", err)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to create mux provider server for testing: %v", err)
			}
			return providerServer(), nil
		},
	}
)
