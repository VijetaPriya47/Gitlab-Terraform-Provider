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
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupServiceAccountAccessToken_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "foo" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					expires_at = "%s"
				}
				`, groupID, serviceAccount.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate the access token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "foo" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = [
						"api",
						"read_user",
						"read_api",
						"read_repository",
						"write_repository",
						"read_registry",
						"write_registry",
						"sudo",
						"admin_mode",
						"create_runner",
						"manage_runner",
						"ai_features",
						"k8s_proxy",
						"read_service_ping",
					]
					expires_at = %q
				}
				`, groupID, serviceAccount.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

// This test checks an issue where using the `expires` to change the rotation would only work once.
// This is because the primary ID of the token was only stored on create, so after the first rotation,
// it attempts to re-use that primary key, which was already expired.
// It may look like the `basic` test covers this, but because `scopes` changes, that forces new and restarts
// the counter for when the error occurs.
func TestAccGitlabGroupServiceAccountAccessToken_rotationUsingExpiresAt(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	initialExpires := testutil.GetCurrentTimePlusDays(t, 10).String()
	updatedExpires := testutil.GetCurrentTimePlusDays(t, 20).String()
	secondUpdateExpires := testutil.GetCurrentTimePlusDays(t, 30).String()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
				  name = "sa token"
				  group = %s
				  user_id = %d
				  expires_at = "%s"
				  scopes = ["api"]
				}
				`, groupID, serviceAccount.ID, initialExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", initialExpires),
				),
			},

			// Update Access Token to change expires
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
				  name = "sa new token"
				  group = %s
				  user_id = %d
				  expires_at = "%s"
				  scopes = ["api"]
				}
				`, groupID, serviceAccount.ID, updatedExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", updatedExpires),
				),
			},
			// Update Access Token once more to change expires a final time
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa new token"
					group = %s
					user_id = %d
					expires_at = "%s"
					scopes = ["api"]
				}
				`, groupID, serviceAccount.ID, secondUpdateExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", secondUpdateExpires),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabGroupServiceAccountAccessTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_service_account_access_token" {
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
				return fmt.Errorf("service account access token with name %q is not in a revoked state", name)
			}
		}
	}

	return nil
}
