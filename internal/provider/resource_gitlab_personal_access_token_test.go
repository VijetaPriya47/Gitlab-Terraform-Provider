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

func TestAccGitlabPersonalAccessToken_createWithPastExpiryDate_validationDisabled(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	pastDate := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "success" {
						user_id    = %d
						name       = "this-token-should-succeed"
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, user.ID, pastDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.success", "expires_at", pastDate),
				),
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_updateWithPastExpiryDate_validationDisabled(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	futureDate := api.CurrentTime().Add(48 * time.Hour).Format(api.Iso8601)
	pastDate := api.CurrentTime().Add(-24 * time.Hour)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "update_success" {
						user_id    = %d
						name       = "token-to-succeed-update"
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, user.ID, futureDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.update_success", "expires_at", futureDate),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "update_success" {
						user_id    = %d
						name       = "token-to-succeed-update"
						scopes     = ["api"]
						expires_at = "%s"
					}
				`, user.ID, pastDate.Format(api.Iso8601)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.update_success", "expires_at", pastDate.Format(api.Iso8601)),
				),
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_failsWithPastExpiryDate_validationEnabled(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	pastDateForConfig := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	parsedDate, err := time.Parse(api.Iso8601, pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "fails" {
						user_id                       = %d
						name                          = "this-token-should-fail"
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, user.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_failsToUpdateWithPastExpiryDate_validationEnabled(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	futureDate := api.CurrentTime().Add(48 * time.Hour).Format(api.Iso8601)

	pastDateForConfig := api.CurrentTime().Add(-24 * time.Hour).Format(api.Iso8601)

	parsedDate, err := time.Parse(api.Iso8601, pastDateForConfig)
	if err != nil {
		t.Fatalf("Failed to parse date for test setup: %v", err)
	}

	pastDateForError := parsedDate.Format(time.RFC3339)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "update_fail" {
						user_id                       = %d
						name                          = "token-to-fail-update"
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, user.ID, futureDate),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.update_fail", "expires_at", futureDate),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_personal_access_token" "update_fail" {
						user_id                       = %d
						name                          = "token-to-fail-update"
						scopes                        = ["api"]
						expires_at                    = "%s"
						validate_past_expiration_date = true
					}
				`, user.ID, pastDateForConfig),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`(?s)Expiry date %s must be in the future\. Current time is\s*.*`, pastDateForError)),
			},
		},
	})
}

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
	`, user.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601))

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
					"validate_past_expiration_date",
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
					description = "hunt by meowing loudly"

					expires_at = "%s"
				}
				`, user.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date", "validate_past_expiration_date"},
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
				`, user.ID, api.CurrentTime().Add(time.Hour*48).Format(api.Iso8601)),
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
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date", "validate_past_expiration_date"},
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

// This test ensures that the `self_rotate` logic works properly in the `update`
// function. It does this by checking the logs for the `DEBUG` level log which which
// is emitted when `self_rotate` is run. Since the use of self_rotate is entirely
// transparent to the end user, this is the only way to integration test the functionality.
func TestAccGitlabPersonalAccessToken_rotationUsingSelfRotate(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

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
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_personal_access_token" "this" {
					name = "my personal token"
					user_id = %d
					scopes = ["api", "self_rotate"]

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
					name = "my personal token"
					user_id = %d
					scopes = ["api", "self_rotate"]

				  // Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
					`, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.this", "rotation_configuration.expiration_days", "3"),
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

func TestAccGitlabPersonalAccessToken_rotationConfiguration(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	// Function for easily calculating the expiry days from the current time.
	getCurrentTimePlusDays := func(days int) gitlab.ISOTime {
		now := api.CurrentTime()
		expiryDate := now.AddDate(0, 0, days)
		expiryIsoTime, err := gitlab.ParseISOTime(expiryDate.Format(api.Iso8601))
		if err != nil {
			t.Fatal("Somehow failed to generate a good date", err)
		}
		return expiryIsoTime
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
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
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
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

// This test can't be run normally in CI/CD since we don't configure out instance
// to disable the token expiration requirement. However, it validates that we can create
// a new token without an expiration.
func TestAccGitlabPersonalAccessToken_tokenWithoutExpiration(t *testing.T) {
	t.Skip()

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
				}
				`, user.ID),
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

// TestAccGitlabPersonalAccessToken_rotateRevokedTokenGracefully tests the scenario where
// a token has been externally revoked but Terraform gracefully handles rotation
func TestAccGitlabPersonalAccessToken_rotateRevokedTokenGracefully(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	tokenToCheck := ""

	// All steps use the same config. Only the external circumstances change.
	config := fmt.Sprintf(`
		resource "gitlab_personal_access_token" "revoked" {
		  name = "token_to_be_revoked"
		  user_id = %d
		  scopes = ["api"]

		  // Create a token good for 30 days, that rotates after 15 days
		  rotation_configuration = {
			  expiration_days = 30
			  rotate_before_days = 15
		  }
		}
		`, user.ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.revoked", "rotation_configuration.expiration_days", "30"),
					resource.TestCheckResourceAttrWith("gitlab_personal_access_token.revoked", "token", func(value string) error {
						// Store token value to compare later
						tokenToCheck = value
						return nil
					}),
				),
			},
			// Simulate external revocation of the token, followed by `terraform refresh`.
			{
				PreConfig: func() {
					if err := revokePersonalAccessToken(user.ID, "token_to_be_revoked"); err != nil {
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
					resource.TestCheckResourceAttr("gitlab_personal_access_token.revoked", "name", "token_to_be_revoked"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.revoked", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.revoked", "revoked", "false"),
					resource.TestCheckResourceAttrWith("gitlab_personal_access_token.revoked", "token", func(value string) error {
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
				ResourceName:            "gitlab_personal_access_token.revoked",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration", "validate_past_expiration_date"},
			},
		},
	})
}

// TestAccGitlabPersonalAccessToken_revokedTokenWithPastExpiry tests that a token
// with an absolute expiry date is not recreated once it expires.
func TestAccGitlabPersonalAccessToken_revokedTokenWithPastExpiry(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]

	expiryDate := testutil.GetCurrentTimePlusDays(t, 2)
	expiryDateStr := time.Time(expiryDate).Format(api.Iso8601)
	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10)

	// Config with short expiration date
	config := fmt.Sprintf(`
		resource "gitlab_personal_access_token" "expired" {
		  name = "token_to_expire"
		  user_id = %d
		  scopes = ["api"]

		  // Will expire in 2 days
		  expires_at = "%s"
		}
	`, user.ID, expiryDateStr)

	// Not running in parallel since we're manipulating environment variables
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Personal Access Token that will expire soon
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.expired", "expires_at", expiryDateStr),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.expired", "active", "true"),
				),
			},
			// Now move time forward past the expiration and revoke the token
			{
				PreConfig: func() {
					// Set time to future, so the token is expired
					t.Setenv("GITLAB_TESTING_TIME", futureDate.Format(time.RFC3339))

					if err := revokePersonalAccessToken(user.ID, "token_to_expire"); err != nil {
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
					resource.TestCheckResourceAttr("gitlab_personal_access_token.expired", "expires_at", expiryDateStr),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.expired", "active", "false"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.expired", "revoked", "true"),
				),
			},
			// Verify with import
			{
				ResourceName:            "gitlab_personal_access_token.expired",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

// TestAccGitlabPersonalAccessToken_withTimeRotating verifies when
// expires_at is set via time_rotating. During plan this value is "unknown", and
// when the token is revoked the provider must handle that unknown value without
// segfaulting while computing a replacement expiry date.
func TestAccGitlabPersonalAccessToken_withTimeRotating(t *testing.T) {
	user := testutil.CreateUsers(t, 1)[0]
	tokenName := "time-rotating-test"

	config := fmt.Sprintf(`
		resource "time_rotating" "example" {
			rotation_days = 30
		}

		resource "gitlab_personal_access_token" "test" {
			user_id    = %d
			name       = %q
			expires_at = split("T", time_rotating.example.rotation_rfc3339)[0]
			scopes     = ["api"]
		}
	`, user.ID, tokenName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"time": {
				Source: "hashicorp/time",
			},
		},
		CheckDestroy: testAccCheckGitlabPersonalAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Step 1: create token whose expires_at is unknown during plan.
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.test", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.test", "token"),
				),
			},
			// Step 2: revoke the token externally, then plan & apply. Without the fix this used to
			// segfault when modifyPlanRevoked called DetermineExpiryDate with an unknown expires_at value.
			{
				PreConfig: func() {
					if err := revokePersonalAccessToken(user.ID, tokenName); err != nil {
						t.Fatalf("failed to revoke token: %v", err)
					}
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_personal_access_token.test", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_personal_access_token.test", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.test", "expires_at"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.test", "token"),
					resource.TestCheckResourceAttrSet("gitlab_personal_access_token.test", "created_at"),
				),
			},
			// Step 3: verify import works with the expected ignores for sensitive fields.
			{
				ResourceName:            "gitlab_personal_access_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "validate_past_expiration_date"},
			},
		},
	})
}

func revokePersonalAccessToken(userID int, tokenName string) error {
	tokenID, err := personalAccessTokenID(userID, tokenName)
	if err != nil {
		return err
	}

	_, err = testutil.TestGitlabClient.PersonalAccessTokens.RevokePersonalAccessToken(tokenID)
	return err
}

func personalAccessTokenID(userID int, tokenName string) (int, error) {
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

	return 0, fmt.Errorf("user %d token with name %s does not exist", userID, tokenName)
}
