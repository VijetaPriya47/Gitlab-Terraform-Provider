//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabApplication_basic(t *testing.T) {
	name := acctest.RandString(10)
	url := "https://my_website.com"
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabApplication_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create a basic application.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_application" "this" {
					name     = %q
					redirect_url = %q
					scopes = ["openid"]
					confidential = true
				}`, name, url),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application.this", "redirect_url", url),
					resource.TestCheckResourceAttr("gitlab_application.this", "scopes.0", "openid"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_application.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "scopes"},
			},
		},
	})
}

func TestAcc_GitlabApplication_EnsureErrorOnInvalidScope(t *testing.T) {
	name := acctest.RandString(10)
	url := "https://my_website.com"
	err_regex, err := regexp.Compile("Invalid Attribute Value Match")
	if err != nil {
		t.Errorf("Unable to format expected application error regex: %s", err)
	}
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_application" "this" {
					name     = %q
					redirect_url = %q
					scopes = ["openid", "invalid"]
				}`, name, url),
				ExpectError: err_regex,
			},
		},
	})
}

func TestAcc_GitlabApplication_EnsureRecreate(t *testing.T) {
	name := acctest.RandString(10)
	name2 := acctest.RandString(10)
	url := "https://my_website.com"
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabApplication_CheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_application" "this" {
					name     = %q
					redirect_url = %q
					scopes = ["openid"]
				}`, name, url),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:            "gitlab_application.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "scopes"},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_application" "this" {
					name     = %q
					redirect_url = %q
					scopes = ["openid"]
				}`, name2, url),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application.this", "name", name2),
				),
			},
		},
	})
}

func testAcc_GitlabApplication_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_application" {
				application, err := findGitlabApplication(testutil.TestGitlabClient, rs.Primary.ID)
				if err == nil {
					return fmt.Errorf("Found GitLab application that should have been deleted: %s", gitlab.Stringify(application))
				}
			}
		}
		return nil
	}
}
