//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectAccessToken_migrateFromSDKToFramework(t *testing.T) {
	// Set up project
	project := testutil.CreateProject(t)

	// Create common config for testing
	config := fmt.Sprintf(`
	resource "gitlab_project_access_token" "foo" {
		project = %d
		name    = "foo"
		scopes  = ["api"]

		expires_at = "%s"
	}
	`, project.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectAccessTokenDestroy,
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
				Check:  resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_project_access_token.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabProjectAccessToken_basic(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					expires_at = "%s"
				}
				`, project.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "access_level", "maintainer"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate the access token with updated attributes.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = [
						"api",
						"read_api",
						"read_registry",
						"write_registry",
						"read_repository",
						"write_repository",
						"ai_features",
						"k8s_proxy",
						"read_observability",
						"write_observability",
					]
					access_level = "developer"
					expires_at = %q
				}
				`, project.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601)),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "revoked", "false"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "token"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.foo", "user_id"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Recreate with `owner` access level.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api", "read_api", "read_repository", "write_repository"]
					access_level = "owner"
					expires_at = %q
				}
				`, project.ID, time.Now().Add(time.Hour*48).Format("2006-01-02")),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.foo",
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
func TestAccGitlabProjectAccessToken_rotationUsingExpiresAt(t *testing.T) {
	project := testutil.CreateProject(t)

	initialExpires := testutil.GetCurrentTimePlusDays(t, 10).String()
	updatedExpires := testutil.GetCurrentTimePlusDays(t, 20).String()
	secondUpdateExpires := testutil.GetCurrentTimePlusDays(t, 30).String()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Project Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "this" {
				  name = "my project token"
				  project = %d
				  expires_at = "%s"
				  access_level = "developer"
				  scopes = ["api"]
				}
					`, project.ID, initialExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.this", "expires_at", initialExpires),
				),
			},
			// Update the Project Access Token to change expires
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "this" {
				  name = "my new project token"
				  project = %d
				  expires_at = "%s"
				  access_level = "maintainer"
				  scopes = ["api"]
				}
					`, project.ID, updatedExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.this", "expires_at", updatedExpires),
				),
			},
			// Update the Project Access Token once more to change expires a final time
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "this" {
					name = "my new project token"
					project = %d
					expires_at = "%s"
					access_level = "maintainer"
					scopes = ["api"]
				}
					`, project.ID, secondUpdateExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.this", "expires_at", secondUpdateExpires),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

// This test checks an issue where using the `expires` to change the rotation would only work when
// `time_offset` isn't used. Prior to introducing rotation, `time_offset` was a common method
// for rotating the token automatically, so many users are likely using it when upgrading.
func TestAccGitlabProjectAccessToken_rotationUsingExpiresAtTimeOffset(t *testing.T) {
	project := testutil.CreateProject(t)

	// lintignore:AT004  // we need the provider configuration for the time provider
	configString := `
		provider "time" {}

		resource "time_offset" "year" {
			offset_days = %d
		}

		resource "gitlab_project_access_token" "token" {
			project      = %d
			name         = "token"
			expires_at   = split("T", time_offset.year.rfc3339)[0]
			access_level = "developer"
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
		CheckDestroy: testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Project Access Token
			{
				Config: fmt.Sprintf(configString, 360, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.token", "expires_at"),
				),
			},
			// Taint the timeoffset to have it re-create the token
			{
				Config:             fmt.Sprintf(configString, 365, project.ID),
				Taint:              []string{"time_offset.year"},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Plan only with an updated date
			{
				Config:             fmt.Sprintf(configString, 365, project.ID),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Re-run the config for a Project Access Token with an updated date
			{
				Config: fmt.Sprintf(configString, 365, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_access_token.token", "expires_at"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.token",
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
func TestAccGitlabProjectAccessToken_rotationUsingDate(t *testing.T) {
	project := testutil.CreateProject(t)

	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Project Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "this" {
				  name = "my project token"
				  project = %d
				  access_level = "developer"
				  scopes = ["api"]

				  // Create a token good for 3 days, that rotates after 1 days
				  rotation_configuration = {
					  expiration_days = 3
					  rotate_before_days = 1
				  }
				}
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_project_access_token.this", "token", func(value string) error {
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
				resource "gitlab_project_access_token" "this" {
				  name = "my new project token"
				  project = %d
				  access_level = "maintainer"
				  scopes = ["api"]

				  // Create a token good for 3 days, that rotates after 1 days
				  rotation_configuration = {
					  expiration_days = 3
					  rotate_before_days = 1
				  }
				}
					`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_project_access_token.this", "token", func(value string) error {
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
				ResourceName:      "gitlab_project_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

func TestAccGitlabProjectAccessToken_rotationConfiguration(t *testing.T) {
	project := testutil.CreateProject(t)

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
		CheckDestroy:             testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}
				}
				`, project.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "expires_at", getCurrentTimePlusDays(10).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_project_access_token.foo",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different expiration. The higher expiration should trigger a rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					// Create a token good for 20 days, that rotates immediately because the 15 days is
					// Greater than the 10 days we previously configured
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 30
					}
				}
				`, project.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "expires_at", getCurrentTimePlusDays(20).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:            "gitlab_project_access_token.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different rotation. The lower expiration should not trigger another rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					// Create a token good for 20 days, that rotates 1 day before expiration
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 1
					}
				}
				`, project.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_project_access_token.foo", "expires_at", getCurrentTimePlusDays(20).String()),
				),
			},
		},
	})
}

func TestAccGitlabProjectAccessToken_attributeValidation(t *testing.T) {
	project := testutil.CreateProject(t)

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
		CheckDestroy:             testAccCheckGitlabProjectAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Validate expires_at and rotation_configuration conflict
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					expires_at = %s

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}
				}
				`, project.ID, getCurrentTimePlusDays(2).String()), // To ensure it's always in the future
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// expires_at or rotation_configuration must be applied
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]
				}
				`, project.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// Validate expiration_days is at least 1
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					rotation_configuration = {
						expiration_days = -1
						rotate_before_days = 1
					}
				}
				`, project.ID),
				ExpectError: regexp.MustCompile("rotation_configuration.expiration_days value must be at least 1"),
			},
			// Validate rotate_before_days is at least 1
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_access_token" "foo" {
					project = %d
					name    = "foo"
					scopes  = ["api"]

					rotation_configuration = {
						expiration_days = 1
						rotate_before_days = -1
					}
				}
				`, project.ID),
				ExpectError: regexp.MustCompile("rotation_configuration.rotate_before_days value must be at least 1"),
			},
		},
	})
}

func testAccCheckGitlabProjectAccessTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_access_token" {
			continue
		}

		project := rs.Primary.Attributes["project"]
		name := rs.Primary.Attributes["name"]

		tokens, _, err := testutil.TestGitlabClient.ProjectAccessTokens.ListProjectAccessTokens(project, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name {
				return fmt.Errorf("project %q access token with name %q still exists", project, name)
			}
		}
	}

	return nil
}
