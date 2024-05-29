//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabPersonalAccessToken_basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id = %d
					name    = "foo"
					scopes  = ["api"]

					expires_at = %q
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format(iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate the access token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id    = %d
					name       = "foo"
					scopes     = ["api", "read_user", "read_api", "read_repository", "write_repository", "sudo", "read_registry", "write_registry"]
					
					expires_at = %q
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format("2006-01-02")),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Recreate the access token with no expiry date
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id    = %d
					name       = "foo"
					scopes     = ["api", "read_api"]
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "expires_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Recreate the access token with no expiry date and check if it creates a new one
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id    = %d
					name       = "foo"
					scopes     = ["api", "read_api", "read_repository"]
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "expires_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Recreate the access token with expiry and check if it creates a new one
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id    = %d
					name       = "foo"
					scopes     = ["api", "read_api", "read_repository"]
					
					expires_at = %q
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format("2006-01-02")),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_admin_mode(t *testing.T) {
	testutil.RunIfAtLeast(t, "15.9")
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create access token with admin_mode scope.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id = %d
					name    = "foo"
					scopes  = ["admin_mode"]

					expires_at = %q
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format(iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "user_id", fmt.Sprintf("%d", user.ID)),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabPersonalAccessTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_personal_access_token" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		userId := rs.Primary.Attributes["user_id"]

		userIdInt, err := strconv.Atoi(userId)
		if err != nil {
			return fmt.Errorf("Error converting user ID to string: %v", userId)
		}

		tokens, _, err := testutil.TestGitlabClient.PersonalAccessTokens.ListPersonalAccessTokens(&gitlab.ListPersonalAccessTokensOptions{UserID: &userIdInt})
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name && !token.Revoked {
				return fmt.Errorf("personal access token with name %q is not in a revoked state", name)
			}
		}
	}

	return nil
}
