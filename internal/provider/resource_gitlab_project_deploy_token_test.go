//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectDeployToken_createWithPastExpiryDate_validationDisabled(t *testing.T) {
	project := testutil.CreateProject(t)
	pastDate, _ := timetypes.NewRFC3339Value(api.CurrentTime().Add(-24 * time.Hour).Format(time.RFC3339))
	pastDateTime, _ := pastDate.ValueRFC3339Time()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_deploy_token" "success" {
						project    = %d
						name       = "this-token-should-succeed"
						scopes     = ["read_repository"]
						expires_at = %s
					}
				`, project.ID, pastDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_deploy_token.success", "expires_at", pastDateTime.Format(time.RFC3339)),
				),
			},
		},
	})
}

func TestAccGitlabProjectDeployToken_failsWithPastExpiryDate_validationEnabled(t *testing.T) {
	project := testutil.CreateProject(t)
	pastDate, _ := timetypes.NewRFC3339Value(api.CurrentTime().Add(-24 * time.Hour).Format(time.RFC3339))
	pastDateTime, _ := pastDate.ValueRFC3339Time()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectDeployTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_deploy_token" "fails" {
						project                       = %d
						name                          = "this-token-should-fail"
						scopes                        = ["read_repository"]
						expires_at                    = %s
						validate_past_expiration_date = true
					}
				`, project.ID, pastDate),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateTime.Format(time.RFC3339))),
			},
		},
	})
}

func TestAccGitlabProjectDeployToken_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	expireTime, _ := timetypes.NewRFC3339Value(api.CurrentTime().Add(time.Hour * 48).Format(time.RFC3339))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectDeployTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic deploy token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_deploy_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["read_repository"]

					expires_at = %s
				}
				`, project.ID, expireTime),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_deploy_token.foo", "expired", "false"),
					resource.TestCheckResourceAttr("gitlab_project_deploy_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_project_deploy_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_project_deploy_token.foo", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_deploy_token.foo", "username"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_deploy_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creation. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
			// Recreate the deploy token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_deploy_token" "foo" {
					project  = %d
					name     = "foo"
					username = "bar"
					scopes   = [
						"read_repository",
						"read_registry",
						"write_registry",
						"read_package_registry",
						"write_package_registry",
						"read_virtual_registry",
						"write_virtual_registry",
					]
				}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_deploy_token.foo", "expired", "false"),
					resource.TestCheckResourceAttr("gitlab_project_deploy_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_project_deploy_token.foo", "token"),
					resource.TestCheckNoResourceAttr("gitlab_project_deploy_token.foo", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_deploy_token.foo", "username"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_deploy_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creation. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

func TestAccGitlabProjectDeployToken_pagination(t *testing.T) {
	project := testutil.CreateProject(t)
	expireTime, _ := timetypes.NewRFC3339Value(api.CurrentTime().Add(time.Hour * 48).Format(time.RFC3339))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectDeployTokenDestroy,
		Steps: []resource.TestStep{
			// Create 25 basic deploy tokens.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_deploy_token" "foo" {
				    count  = %d
					project = %d
					name    = "foo-${count.index}"
					scopes  = ["read_repository"]

					expires_at = %s
				}
				`, 25, project.ID, expireTime),
			},
			// In case pagination wouldn't properly work, we would get that the plan isn't empty,
			// because some of the deploy tokens wouldn't be in the first page and therefore
			// considered non-existing, ...
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_deploy_token" "foo" {
				    count  = %d
					project = %d
					name    = "foo-${count.index}"
					scopes  = ["read_repository"]

					expires_at = %s
				}
				`, 25, project.ID, expireTime),
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckGitlabProjectDeployTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_deploy_token" {
			continue
		}

		project := rs.Primary.Attributes["project"]
		name := rs.Primary.Attributes["name"]

		tokens, _, err := testutil.TestGitlabClient.DeployTokens.ListProjectDeployTokens(project, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name && !token.Revoked {
				return fmt.Errorf("project %q deploy token with name %q still exists", project, name)
			}
		}
	}

	return nil
}
