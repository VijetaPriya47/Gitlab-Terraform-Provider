//go:build acceptance || flakey || saas

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
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

func TestProvider_customHeaders(t *testing.T) {
	testutil.SkipIfSaaS(t)

	// Create a mock server for intercepting early auth commands. This also lets us check the
	// headers without trying to intercept http calls to the real GitLab API.
	earlyAuthCalls := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if our header value is applied
		if r.Header.Get("X-Custom-Header") != "test" {
			t.Errorf("Expected X-Custom-Header to be 'test', got '%s'", r.Header.Get("X-Custom-Header"))
		}

		// Early auth calls are made to /api/v4/user and datasource request is made to /api/graphql
		if r.URL.Path == "/api/v4/user" {
			earlyAuthCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1, "username": "test-user"}`)) // nolint - don't need to err check writing the response in the test
		}

		if r.URL.Path == "/api/graphql" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			// nolint - don't need to err check writing the response in the test
			w.Write([]byte(`{
			"data": {
				"currentUser": {
				"name": "KittyMeow",
				"id": "gid://gitlab/User/8095268",
				"namespace": {
					"id": "gid://gitlab/Namespaces::UserNamespace/123456"
				},
				"publicEmail": "",
				"username": "KittyMeow"
				}
			}
			}`))
		}
	}))
	defer mockServer.Close()

	// lintignore:AT001 // Providers don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				// lintignore:AT004 // Explicitly testing a provider configuration
				Config: fmt.Sprintf(`
					provider "gitlab" {
						base_url = "%s"
						token = "test-token" // doesn't matter, we're intercepting the call
						early_auth_check = true
						headers = {
							"X-Custom-Header" = "test"
						}
					}

					data "gitlab_current_user" "test" {}
					`, mockServer.URL),
				Check: func(*terraform.State) error {
					if earlyAuthCalls != 2 {
						return fmt.Errorf("expected 2 early_auth call, got %d", earlyAuthCalls)
					}
					return nil
				},
			},
		},
	})
}
