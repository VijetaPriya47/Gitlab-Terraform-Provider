//go:build acceptance

package provider

import (
	"bufio"
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

func TestAccGitlabGroupServiceAccountAccessToken_createWithPastExpiryDate_validationDisabled(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	pastDate := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				// This configuration should now SUCCEED because the new field defaults to false.
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "success" {
						name       = "this-token-should-succeed"
						group      = %s
						user_id    = %d
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, groupID, serviceAccount.ID, pastDate),
				// We now check that the resource was created successfully with the past date.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.success", "expires_at", pastDate),
				),
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_updateWithPastExpiryDate_validationDisabled(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	futureDate := api.CurrentTime().Add(48 * time.Hour).Format(api.Iso8601)
	pastDate := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create the resource successfully with a future date.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "update_success" {
						name       = "token-to-succeed-update"
						group      = %s
						user_id    = %d
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, groupID, serviceAccount.ID, futureDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.update_success", "expires_at", futureDate),
				),
			},
			// Step 2: Attempt to update the resource with a past date, which should now succeed.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "update_success" {
						name       = "token-to-succeed-update"
						group      = %s
						user_id    = %d
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, groupID, serviceAccount.ID, pastDate),
				// We check that the expires_at attribute was successfully updated to the past date.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.update_success", "expires_at", pastDate),
				),
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_failsWithPastExpiryDate_validationEnabled(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	pastDateForConfig := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	parsedDate, err := time.Parse(api.Iso8601, pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "fails" {
						name                          = "this-token-should-fail"
						group                         = %s
						user_id                       = %d
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, groupID, serviceAccount.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_failsToUpdateWithPastExpiryDate_validationEnabled(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)
	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	futureDate := api.CurrentTime().Add(48 * time.Hour).Format(api.Iso8601)
	pastDateForConfig := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	parsedDate, err := time.Parse(api.Iso8601, pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create the resource successfully with a future date.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "update_fail" {
						name                          = "token-to-fail-update"
						group                         = %s
						user_id                       = %d
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, groupID, serviceAccount.ID, futureDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.update_fail", "expires_at", futureDate),
				),
			},
			// Step 2: Attempt to update the resource with a past date and expect an error.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_service_account_access_token" "update_fail" {
						name                          = "token-to-fail-update"
						group                         = %s
						user_id                       = %d
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, groupID, serviceAccount.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}

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
				`, groupID, serviceAccount.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
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
				`, groupID, serviceAccount.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

// See issue https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/6537
// bug introduced in 18.1.0
func TestAccGitlabGroupServiceAccountAccessToken_regression6537(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create the token with config that caused an error in provider version 18.1.0
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
				variable "scopes" {
					default = ["api"]
				}

				resource "gitlab_group_service_account_access_token" "service_account_token" {
					group   = %s
					user_id = %d
					name    = "tests"
					scopes  = toset(concat(tolist(var.scopes), ["read_api"]))


					rotation_configuration = {
						rotate_before_days = 30
						expiration_days    = 365
					}
				}
				`, groupID, serviceAccount.ID),
			},
			// Verify upstream resource with an import.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_group_service_account_access_token.service_account_token",
				ImportState:              true,
				ImportStateVerify:        true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_noExpiration(t *testing.T) {
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
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api"]
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
			// Recreate the access token with an expiration. The expiration should trigger a rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa new token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 30 days
					rotation_configuration = {
						expiration_days = 30
						rotate_before_days = 20
					}
				}
				`, groupID, serviceAccount.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 30).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
			// Recreate the access token without rotation again. The lower expiration should not trigger another rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa new token"
					group = %s
					user_id = %d
					scopes = ["api"]
				}
				`, groupID, serviceAccount.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

func TestAccGitlabGroupServiceAccountAccessToken_rotationConfiguration(t *testing.T) {
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
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 10 days, that rotates after 1 day
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 10).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
			// Recreate the access token with a different expiration. The higher expiration should trigger a rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa new token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 20 days, that rotates immediately
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 30
					}
				}
				`, groupID, serviceAccount.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 20).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
			// Recreate the access token with a different rotation. The lower expiration should not trigger another rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa new token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 20 days, that rotates 1 day before it expires
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 2
					}
				}
				`, groupID, serviceAccount.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 20).String()),
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
						expiration_days = 10
						rotate_before_days = 1
					}

				}
				`, groupID, serviceAccount.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)), // so it's always in the future.
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// Validate that expiration must be > 0
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = -1
						rotate_before_days = 1
					}

				}
				`, groupID, serviceAccount.ID),
				ExpectError: regexp.MustCompile("Attribute rotation_configuration.expiration_days value must be at least 1"),
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
						expiration_days = 1
						rotate_before_days = -1
					}

				}
				`, groupID, serviceAccount.ID),
				ExpectError: regexp.MustCompile("Attribute rotation_configuration.rotate_before_days value must be at least 1"),
			},
			// Validate you can't use `self_rotate` in the scopes with no expiry
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					group = %s 
					user_id  = %d
					name     = "foo"
					scopes   = ["api", "self_rotate"]
				}
				`, groupID, serviceAccount.ID),
				ExpectError: regexp.MustCompile(`Invalid token scopes`),
			},
		},
	})
}

// This test checks that when a date is sufficiently in the future, the token
// will rotate automatically. This is done by mocking the time using
// the `GITLAB_TESTING_TIME` environment variable, which is parsed by
// `api.CurrentTime()`, which the resource uses instead of time.Now()
func TestAccGitlabGroupServiceAccountAccessToken_rotationUsingDate(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.this", "token", func(value string) error {
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
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.this", "token", func(value string) error {
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
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

// This test ensures that the `self_rotate` logic works properly in the `update`
// function. It does this by checking the logs for the `DEBUG` level log which which
// is emitted when `self_rotate` is run. Since the use of self_rotate is entirely
// transparent to the end user, this is the only way to integration test the functionality.
func TestAccGitlabGroupServiceAccountAccessToken_rotationUsingSelfRotate(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	// Ensure that the provider is configured to log to a specific location at "DEBUG" level
	// logs
	logPath := fmt.Sprintf("/tmp/tf-log%v.log", time.Now())
	os.Setenv("TF_LOG", "DEBUG")
	os.Setenv("TF_LOG_PATH", logPath)
	t.Cleanup(func() {
		os.Remove(logPath)
		os.Unsetenv("TF_LOG")
		os.Unsetenv("TF_LOG_PATH")
	})

	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api", "self_rotate"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.this", "token", func(value string) error {
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
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api", "self_rotate"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, groupID, serviceAccount.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.this", "rotation_configuration.expiration_days", "3"),
					// Check that self-rotate was used
					func(*terraform.State) error {
						logLineToCheck := "attempting to use the self-rotate method to update the token"

						// Read the log file to check for the self_rotate debug message
						logFile, err := os.Open(logPath)
						if err != nil {
							return fmt.Errorf("failed to read log file: %v", err)
						}
						defer logFile.Close()

						// read the file line-by-line to reduce memory usage
						scanner := bufio.NewScanner(logFile)
						for scanner.Scan() {
							if strings.Contains(scanner.Text(), logLineToCheck) {
								return nil
							}
						}

						return fmt.Errorf("Unable to find self rotate log entry %q", logLineToCheck)
					},
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.this", "token", func(value string) error {
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
				ResourceName:      "gitlab_group_service_account_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_service_account_access_token" "this" {
					name = "sa token"
					group = %s
					user_id = %d
					scopes = ["api"]

					expires_at = "%s"
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
					scopes = ["api"]

					expires_at = "%s"
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
					scopes = ["api"]

					expires_at = "%s"
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
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
				`, token.Token, groupID, serviceAccount.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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
				`, token.Token, groupID, serviceAccount.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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

// TestAccGitlabGroupServiceAccountAccessToken_rotateRevokedTokenGracefully tests the scenario where
// a token has been externally revoked but Terraform gracefully handles rotation
func TestAccGitlabGroupServiceAccountAccessToken_rotateRevokedTokenGracefully(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]
	tokenToCheck := ""

	// All steps use the same config. Only the external circumstances change.
	config := fmt.Sprintf(`
		resource "gitlab_group_service_account_access_token" "revoked" {
		  name = "token_to_be_revoked"
		  group = %s
		  user_id = %d
		  scopes = ["api"]

		  // Create a token good for 30 days, that rotates after 15 days
		  rotation_configuration = {
			  expiration_days = 30
			  rotate_before_days = 15
		  }
		}
		`, groupID, serviceAccount.ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Service Account Access Token
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.revoked", "rotation_configuration.expiration_days", "30"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.revoked", "token", func(value string) error {
						// Store token value to compare later
						tokenToCheck = value
						return nil
					}),
				),
			},
			// Simulate external revocation of the token, followed by `terraform refresh`.
			{
				PreConfig: func() {
					if err := revokeServiceAccountAccessToken(serviceAccount.ID, "token_to_be_revoked"); err != nil {
						t.Fatalf("Failed to revoke token: %v", err)
					}
				},
				// Use RefreshState to force reading the current state
				RefreshState: true,
				// We expect changes since the token is now revoked
				ExpectNonEmptyPlan: true,
			},
			// Apply the config. This should recreate the token since it was revoked externally.
			{
				Config: config,
				// Success case - no error expected
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.revoked", "name", "token_to_be_revoked"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.revoked", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.revoked", "revoked", "false"),
					resource.TestCheckResourceAttrWith("gitlab_group_service_account_access_token.revoked", "token", func(value string) error {
						// Verify new token is different from the revoked one
						if value == tokenToCheck {
							return fmt.Errorf("token was not rotated after being revoked")
						}
						return nil
					}),
				),
			},
			// Verify upstream resource with an import
			{
				ResourceName:            "gitlab_group_service_account_access_token.revoked",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

// TestAccGitlabGroupServiceAccountAccessToken_revokedTokenWithPastExpiry tests that a token
// with an absolute expiry date is not recreated once it expires.
func TestAccGitlabGroupServiceAccountAccessToken_revokedTokenWithPastExpiry(t *testing.T) {
	testutil.SkipIfCE(t)

	group := testutil.CreateGroups(t, 1)[0]
	groupID := strconv.Itoa(group.ID)

	serviceAccount := testutil.CreateGroupServiceAccounts(t, 1, groupID)[0]

	expiryDate := testutil.GetCurrentTimePlusDays(t, 2)
	expiryDateStr := time.Time(expiryDate).Format(api.Iso8601)
	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10)

	// Config with short expiration date
	config := fmt.Sprintf(`
		resource "gitlab_group_service_account_access_token" "expired" {
		  name = "token_to_expire"
		  group = %s
		  user_id = %d
		  scopes = ["api"]

		  // Will expire in 2 days
		  expires_at = "%s"
		}
	`, groupID, serviceAccount.ID, expiryDateStr)

	// Not running in parallel since we're manipulating environment variables
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupServiceAccountAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Service Account Access Token that will expire soon
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.expired", "expires_at", expiryDateStr),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.expired", "active", "true"),
				),
			},
			// Now move time forward past the expiration and revoke the token
			{
				PreConfig: func() {
					// Set time to future, so the token is expired
					t.Setenv("GITLAB_TESTING_TIME", futureDate.Format(time.RFC3339))

					if err := revokeServiceAccountAccessToken(serviceAccount.ID, "token_to_expire"); err != nil {
						t.Fatalf("Failed to revoke token: %v", err)
					}
				},
				// Refresh state to detect the token is expired and revoked
				RefreshState: true,
				// We do NOT expect changes since the token was expired anyway
				ExpectNonEmptyPlan: false,
			},
			// Apply the config again - this should not attempt to recreate the token
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.expired", "expires_at", expiryDateStr),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.expired", "active", "false"),
					resource.TestCheckResourceAttr("gitlab_group_service_account_access_token.expired", "revoked", "true"),
				),
			},
			// Verify with import
			{
				ResourceName:            "gitlab_group_service_account_access_token.expired",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

// Helper function to revoke a service account access token
func revokeServiceAccountAccessToken(userID int, tokenName string) error {
	tokenID, err := serviceAccountAccessTokenID(userID, tokenName)
	if err != nil {
		return err
	}

	_, err = testutil.TestGitlabClient.PersonalAccessTokens.RevokePersonalAccessToken(tokenID)
	return err
}

// Helper function to get the ID of a service account access token
func serviceAccountAccessTokenID(userID int, tokenName string) (int, error) {
	tokens, _, err := testutil.TestGitlabClient.PersonalAccessTokens.ListPersonalAccessTokens(&gitlab.ListPersonalAccessTokensOptions{
		UserID: gitlab.Ptr(userID),
	})
	if err != nil {
		return 0, err
	}

	for _, token := range tokens {
		if token.Name == tokenName {
			return token.ID, nil
		}
	}

	return 0, fmt.Errorf("service account %d token with name %s does not exist", userID, tokenName)
}
