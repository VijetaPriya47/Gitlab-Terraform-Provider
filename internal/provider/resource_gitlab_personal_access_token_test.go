//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

func TestAccGitlabPersonalAccessToken_rotationUsingExpiresAtTimeOffset(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	// lintignore:AT004  // we need the provider configuration for the time provider
	configString := `
		provider "time" {}

		resource "time_offset" "year" {
			offset_days = %d
		}

		resource "gitlab_personal_access_token" "this" {
			name = "my new personal token"
			user_id = %d
			expires_at   = split("T", time_offset.year.rfc3339)[0]
			scopes       = ["read_api"]
		}
		`

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"time": {
				Source: "hashicorp/time",
			},
		},
		CheckDestroy: testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token
			{
				Config: fmt.Sprintf(configString, 360, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.this", "expires_at"),
				),
			},
			// Taint the timeoffset to have it re-create the token
			{
				Config:             fmt.Sprintf(configString, 365, user.ID),
				Taint:              []string{"time_offset.year"},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Plan only with an updated date
			{
				Config:             fmt.Sprintf(configString, 365, user.ID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Re-run the config for a Personal Access Token with an updated date
			{
				Config: fmt.Sprintf(configString, 365, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.this", "expires_at"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

// This test checks that when a date is sufficiently in the future, the token
// will rotate automatically. This is done by mocking the time using
// the `GITLAB_TESTING_TIME` environment variable, which is parsed by
// `api.CurrentTime()`, which the resource uses instead of time.Now()
func TestAccGitlabPersonalAccessToken_rotationUsingDate(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					name = "my personal token"
					user_id = %d
					scopes = ["api"]

				  // Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
					`, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_personal_access_token.this", "token", func(value string) error {
						// Set the token that we have in state
						tokenToCheck = value
						return nil
					}),
				),
			},
			// Mock the date to ensure that the token properly rotates in the future
			{
				PreConfig: func() {
					os.Setenv("GITLAB_TESTING_TIME", futureDate)
					t.Cleanup(func() {
						os.Unsetenv("GITLAB_TESTING_TIME")
					})
				},
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					name = "my new personal token"
					user_id = %d
					scopes = ["api"]

				  // Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
					`, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_personal_access_token.this", "token", func(value string) error {
						// The token shouldn't match what we have from the previous apply. It should have been rotated
						if value == tokenToCheck {
							return fmt.Errorf("token did not rotate")
						}

						return nil
					}),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_rotationConfiguration(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	// Function for easily calculating the expiry days from the current time.
	getCurrentTimePlusDays := func(days int) gitlab.ISOTime {
		now := time.Now()
		expiryDate := now.AddDate(0, 0, days)
		expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))
		if err != nil {
			t.Fatal("Somehow failed to generate a good date", err)
		}
		return expiryIsoTime
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					user_id = %d
					name    = "my new personal token"
					scopes  = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", getCurrentTimePlusDays(10).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_personal_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different expiration. The higher expiration should trigger a rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					user_id = %d
					name    = "my new personal token"
					scopes  = ["api"]

					// Create a token good for 20 days, that rotates immediately because the 15 days is
					// Greater than the 10 days we previously configured
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 30
					}
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", getCurrentTimePlusDays(20).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_personal_access_token.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different rotation. The lower expiration should not trigger another rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					user_id = %d
					name    = "my new personal token"
					scopes  = ["api"]

					// Create a token good for 20 days, that rotates 1 day before expiration
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 1
					}
				}
				`, user.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "expires_at", getCurrentTimePlusDays(20).String()),
				),
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
