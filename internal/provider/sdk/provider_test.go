//go:build acceptance || flakey
// +build acceptance flakey

package sdk

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactoriesV6 = map[string]func() (tfprotov6.ProviderServer, error){
	"gitlab": func() (tfprotov6.ProviderServer, error) {
		provider, err := NewV6(context.Background(), "test")
		if err != nil {
			return nil, err
		}
		return provider, nil
	},
}

func TestProvider(t *testing.T) {
	t.Parallel()

	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	t.Parallel()
	var _ = New("dev")()
}
