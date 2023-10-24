//go:build acceptance || flakey
// +build acceptance flakey

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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

func TestAcc_GitLabProvider_UpgradeLatestMajor(t *testing.T) {
	testProjectName := acctest.RandomWithPrefix("acctest-upgrade-test")

	// commonConfig is used as a dummy configuration using the provider
	// which is expected not to break between major gitlab provider versions.
	// However, this may still happen in the future - in that case, it's
	// okay to change this test case accordingly.
	commonConfig := fmt.Sprintf(`
		resource "gitlab_project" "test" {
			name                   = "%s"
			initialize_with_readme = true
			visibility_level       = "public"
        }
	`, testProjectName)

	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			// Create resources with the latest major version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 3.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: commonConfig,
			},
			// Migrate to the current provider version
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   commonConfig,
			},
		},
	})
}
