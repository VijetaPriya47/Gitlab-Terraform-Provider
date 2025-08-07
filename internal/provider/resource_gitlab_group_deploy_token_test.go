//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupDeployToken_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	expireTime, _ := timetypes.NewRFC3339Value(time.Now().Add(time.Hour * 48).Format(time.RFC3339))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupDeployTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic deploy token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_deploy_token" "foo" {
					group  = %d
					name   = "foo"
					scopes = ["read_repository"]

					expires_at = %s
				}
				`, group.ID, expireTime),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_deploy_token.foo", "expired", "false"),
					resource.TestCheckResourceAttr("gitlab_group_deploy_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_deploy_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_group_deploy_token.foo", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_deploy_token.foo", "username"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_deploy_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creation. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate the deploy token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_deploy_token" "foo" {
					group    = %d
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
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_deploy_token.foo", "expired", "false"),
					resource.TestCheckResourceAttr("gitlab_group_deploy_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_deploy_token.foo", "token"),
					resource.TestCheckNoResourceAttr("gitlab_group_deploy_token.foo", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_deploy_token.foo", "username"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_deploy_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creation. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabGroupDeployToken_pagination(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	expireTime, _ := timetypes.NewRFC3339Value(time.Now().Add(time.Hour * 48).Format(time.RFC3339))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupDeployTokenDestroy,
		Steps: []resource.TestStep{
			// Create 25 basic deploy tokens.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_deploy_token" "foo" {
				    count  = %d
					group  = %d
					name   = "foo-${count.index}"
					scopes = ["read_repository"]

					expires_at = %s
				}
				`, 25, group.ID, expireTime),
			},
			// In case pagination wouldn't properly work, we would get that the plan isn't empty,
			// because some of the deploy tokens wouldn't be in the first page and therefore
			// considered non-existing, ...
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_deploy_token" "foo" {
				    count  = %d
					group  = %d
					name   = "foo-${count.index}"
					scopes = ["read_repository"]

					expires_at = %s
				}
				`, 25, group.ID, expireTime),
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckGitlabGroupDeployTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_deploy_token" {
			continue
		}

		group := rs.Primary.Attributes["group"]
		name := rs.Primary.Attributes["name"]

		tokens, _, err := testutil.TestGitlabClient.DeployTokens.ListGroupDeployTokens(group, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name && !token.Revoked {
				return fmt.Errorf("group %q deploy token with name %q still exists", group, name)
			}
		}
	}

	return nil
}
