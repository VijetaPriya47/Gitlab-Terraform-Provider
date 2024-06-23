//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabPersonalAccessToken_migrateFromSDKToFramework(t *testing.T) {
	// Set up user
	user := testutil.CreateUsers(t, 1)[0]

	// Create common config for testing
	config := fmt.Sprintf(`
	resource "gitlab_personal_access_token" "foo" {
		user_id = %d
		name    = "foo"
		scopes  = ["api"]

		expires_at = "%s"
	}
	`, user.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 16.10",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_personal_access_token.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_basic(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "foo" {
					user_id = %d
					name    = "foo"
					scopes  = ["api"]

					expires_at = "%s"
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "user_id"),
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
					user_id = %d
					name    = "foo"
					scopes  = [
						"api",
						"read_user",
						"read_api",
						"read_repository",
						"write_repository",
						"read_registry",
						"write_registry",
						"sudo",
						"admin_mode",
						"ai_features",
						"k8s_proxy",
						"read_service_ping",
					]
					expires_at = %q
				}
				`, user.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.foo", "user_id"),
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

// This test checks an issue where using the `expires` to change the rotation would only work once.
// This is because the primary ID of the token was only stored on create, so after the first rotation,
// it attempts to re-use that primary key, which was already expired.
// It may look like the `basic` test covers this, but because `scopes` changes, that forces new and restarts
// the counter for when the error occurs.
func TestAccGitlabPersonalAccessToken_rotationUsingExpiresAt(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	initialExpires := testutil.GetCurrentTimePlusDays(t, 10).String()
	updatedExpires := testutil.GetCurrentTimePlusDays(t, 20).String()
	secondUpdateExpires := testutil.GetCurrentTimePlusDays(t, 30).String()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
				  name = "my personal token"
				  user_id = %d
				  expires_at = "%s"
				  scopes = ["api"]
				}
					`, user.ID, initialExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", initialExpires),
				),
			},
			// Update the Personal Access Token to change expires
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
				  name = "my new personal token"
				  user_id = %d
				  expires_at = "%s"
				  scopes = ["api"]
				}
					`, user.ID, updatedExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", updatedExpires),
				),
			},
			// Update the Personal Access Token once more to change expires a final time
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					name = "my new personal token"
					user_id = %d
					expires_at = "%s"
					scopes = ["api"]
				}
					`, user.ID, secondUpdateExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", secondUpdateExpires),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.this",
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
