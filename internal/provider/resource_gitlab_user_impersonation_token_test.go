//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabUserImpersonationToken_basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	expiresAt := time.Now().Add(time.Hour * 48).Format(api.Iso8601)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabUserImpersonationToken_destroy,
		Steps: []resource.TestStep{
			// Create a basic user impersonation token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_impersonation_token" "this" {
					user_id = %d
					name    = "this"
					scopes  = ["api"]

					expires_at = "%s"
				}
				`, user.ID, expiresAt),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "id"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "user_id", strconv.Itoa(user.ID)),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "token_id"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "name", "this"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "expires_at", expiresAt),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "scopes.#", "1"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "impersonation", "true"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "token"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_user_impersonation_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate the access token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_user_impersonation_token" "this" {
					user_id = %d
					name    = "this2"
					scopes  = ["api"]
					expires_at = %q
				}
				`, user.ID, expiresAt),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "id"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "user_id", strconv.Itoa(user.ID)),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "token_id"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "name", "this2"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "expires_at", expiresAt),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "scopes.#", "1"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "impersonation", "true"),
					resource.TestCheckResourceAttr("gitlab_user_impersonation_token.this", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "token"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_user_impersonation_token.this", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_user_impersonation_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabUserImpersonationToken_destroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_user_impersonation_token" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		userId := rs.Primary.Attributes["user_id"]

		userIdInt, err := strconv.Atoi(userId)
		if err != nil {
			return fmt.Errorf("Error converting user ID to string: %v", userId)
		}

		tokens, _, err := testutil.TestGitlabClient.Users.GetAllImpersonationTokens(userIdInt, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name && !token.Revoked {
				return fmt.Errorf("user impersonation token with name %q is not in a revoked state", name)
			}
		}
	}

	return nil
}
