//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAcc_GitLab_ProviderMux(t *testing.T) {
	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					// The gitlab_metadata data source is based on the terraform-plugin-framework
					data "gitlab_metadata" "test" {}
					
					// The gitlab_current_user data source is based on the terraform-plugin-sdk
					data "gitlab_current_user" "test" {}
                `,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify framework based data source attribute
					resource.TestCheckResourceAttr("data.gitlab_metadata.test", "id", "1"),
					// Verify sdk based data source attribute
					resource.TestCheckResourceAttr("data.gitlab_current_user.test", "id", "1"),
				),
			},
		},
	})
}

func TestAcc_GitLab_ProviderRetries(t *testing.T) {

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusTooManyRequests)
	}))

	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				// we need the provider configuration here to configure the custom retries
				// lintignore:AT004
				Config: fmt.Sprintf(`
					provider "gitlab" {
						base_url = "%s"
						retries = 2
					}

					// The gitlab_current_user data source is based on the terraform-plugin-sdk
					data "gitlab_current_user" "test" {}
                `, server.URL),
				ExpectError: regexp.MustCompile("429"),
			},
			{
				// This needs to be in `PreConfig` instead of the `Check` in the previous test, because
				// when a step ends with `ExpectError` it doesn't run the checks.
				PreConfig: func() {
					// If requests isn't equal to 3, we need to error
					if requests != 3 {
						t.Errorf("Expected 3 requests, but got %d", requests)
					}
				},
				RefreshState: true,
				ExpectError:  regexp.MustCompile("429"),
			},
		},
	})
}
