//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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

// This test checks that when the resource is created with rotation_configuration
// the expires_at date is defaulted to seven days in the future
// Unfortunately, we cannot currently test rotating the token since we can't influence
// the new expiry date. It will always be seven days in the future based off
// the real time rather than any mocked time we give the provider as it
// is set by the API.
func TestAccGitlabGroupServiceAccountAccessToken_rotationConfiguration(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]
	expiryDate := testutil.GetCurrentTimestampPlusDays(t, 7).Format(time.DateOnly)
	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group Service Account Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group    = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					rotation_configuration = {
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.rotate_before_days", "1"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", expiryDate),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.this", "token"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Rotate the access token using the mocked date option to ensure the ID rotates properly.
			{
				PreConfig: func() {
					os.Setenv("GITLAB_TESTING_TIME", futureDate)
					t.Cleanup(func() {
						os.Unsetenv("GITLAB_TESTING_TIME")
					})
				},
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group    = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					rotation_configuration = {
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				// Because we can't set the `expiryDate` on service accounts, the instance will always set them to 7 days in the future.
				// This means that the mocked time doesn't prevent a second rotation after this one applies. However, if the rotation
				// didn't work, an error would still cause the test to fail.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.rotate_before_days", "1"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", expiryDate),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.this", "token"),
				),
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_attributeValidation(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Validate expires_at and rotation_configuration conflict
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					expires_at = "%s"

					rotation_configuration = {
						rotate_before_days = 1
					}

				}
				`, groupID, serviceAccount.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)), // so it's always in the future.
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// At least one rotation_configuration or expires_at is required
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]
				}
				`, groupID, serviceAccount.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// Validate that rotate_before_days must be > 0
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					rotation_configuration = {
						rotate_before_days = -1
					}

				}
				`, groupID, serviceAccount.ID),
				ExpectError: regexp.MustCompile("rotation_configuration.rotate_before_days value must be at least 1"),
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

// This test ensures that a user who is an Owner level will be able to use and rotate the token using the state data
// even when they can't normally read the service account token's information and the token is expired.
func TestAccGitlabGroupServiceAccountAccessToken_nonAdminTokenExpired(t *testing.T) {
	testutil.SkipIfCE(t)

	ownerUser := testutil.CreateUsers(t, 1)[0]
	token := testutil.CreatePersonalAccessToken(t, ownerUser)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	// Add the user to the group with owner permissions
	testutil.AddGroupMembersWithAccessLevel(t, groupID, []*gitlab.User{ownerUser}, gitlab.OwnerPermissions)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	initialExpires := testutil.GetCurrentTimePlusDays(t, 5).String()
	newExpires := testutil.GetCurrentTimePlusDays(t, 11).String()
	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Explicitly don't run this as a parallel test, since membership additions happen async, and a busier instance
	// means it's more likely to fail because the background process hasn't run yet.
	// And since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				// lintignore:AT004  // we need the provider configuration here to attempt to create the service account as a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_group_service_account_access_token" "foo" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					expires_at = "%s"
				}
				`, token.Token, groupID, serviceAccount.ID, initialExpires),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "user_id"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.foo", "token", func(value string) error {
						// Set the token that we have in state
						tokenToCheck = value
						return nil
					}),
				),
			},
			// Recreate the access token when it has expired.
			{
				PreConfig: func() {
					// Use the testClient to revoke the token (to ensure it's expired)
					_, err := testutil.TestGitlabClient.PersonalAccessTokens.RevokePersonalAccessTokenSelf(gitlab.WithToken(gitlab.PrivateToken, tokenToCheck))
					if err != nil {
						t.Fatalf("failed to revoke token: %v", err)
					}
					// Set the `apit.GetCurrentTime()` to return in the future so the provider knows the
					// token is expired when it checks.
					os.Setenv("GITLAB_TESTING_TIME", futureDate)
					t.Cleanup(func() {
						os.Unsetenv("GITLAB_TESTING_TIME")
					})
				},
				// lintignore:AT004  // we need the provider configuration here to attempt to create the service account as a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_group_service_account_access_token" "foo" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					expires_at = "%s"
				}
				`, token.Token, groupID, serviceAccount.ID, newExpires),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.foo", "token", func(value string) error {
						// The token shouldn't match what we have from the previous apply. It should have been rotated
						if value == tokenToCheck {
							return fmt.Errorf("token did not rotate")
						}

						return nil
					}),
				),
			},
		},
	})
}

// This test ensures that a user who is an Owner level will be able to use and rotate the token using the state data
// even when they can't normally read the service account token's information.
func TestAccGitlabGroupServiceAccountAccessToken_nonAdminToken(t *testing.T) {
	testutil.SkipIfCE(t)

	ownerUser := testutil.CreateUsers(t, 1)[0]
	token := testutil.CreatePersonalAccessToken(t, ownerUser)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	// Add the user to the group with owner permissions
	testutil.AddGroupMembersWithAccessLevel(t, groupID, []*gitlab.User{ownerUser}, gitlab.OwnerPermissions)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	// Explicitly don't run this as a parallel test, since membership additions happen async, and a busier instance
	// means it's more likely to fail because the background process hasn't run yet.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				// lintignore:AT004  // we need the provider configuration here to attempt to create the service account as a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

				resource "gitlab_group_service_account_access_token" "foo" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					expires_at = "%s"
				}
				`, token.Token, groupID, serviceAccount.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "user_id"),
				),
			},
			// Recreate the access token with updated attributes.
			{
				// lintignore:AT004  // we need the provider configuration here to attempt to create the service account as a different user
				Config: fmt.Sprintf(`
				provider "gitlab" {
					token = "%s"
				}

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
				`, token.Token, groupID, serviceAccount.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_group_service_account_access_token.foo", "user_id"),
				),
			},
		},
	})
}

func testAccCheckGitlabGroupServiceAccountAccessTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_service_account_access_token" {
			continue
		}

		id := rs.Primary.Attributes["id"]
		splitedID := strings.SplitN(id, ":", 3)
		if len(splitedID) != 3 {
			return fmt.Errorf("Invalid number of parts in ID %q", id)
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
			// index 2 is the access token ID
			if strconv.Itoa(token.ID) == splitedID[2] && !token.Revoked {
				return fmt.Errorf("service account access token with name %q is not in a revoked state", name)
			}
		}
	}

	return nil
}
