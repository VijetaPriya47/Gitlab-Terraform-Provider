//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabGroupAccessToken_migrateFromSDKToFramework(t *testing.T) {
	// Set up group
	group := testutil.CreateGroups(t, 1)[0]

	// Create common config for testing
	config := fmt.Sprintf(`
	resource "gitlab_group_access_token" "foo" {
		group = %d
		name    = "foo"
		scopes  = ["api"]

		expires_at = "%s"
	}
	`, group.ID, time.Now().Add(time.Hour*48).Format(api.Iso8601))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupAccessTokenDestroy,
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
				Check:  resource.TestCheckResourceAttrSet("gitlab_group_access_token.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    resource.TestCheckResourceAttrSet("gitlab_group_access_token.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_group_access_token.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabGroupAccessToken_basic(t *testing.T) {
	var gat testAccGitlabGroupAccessTokenWrapper

	testGroup := testutil.CreateGroups(t, 1)[0]

	expiresAt := time.Now().AddDate(0, 1, 0)
	updatedExpiresAt := expiresAt.AddDate(0, 1, 0)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group and a Group Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my group token"
				  group = %d
				  expires_at = "%s"
				  access_level = "developer"
				  scopes = ["read_repository" , "api", "write_repository", "read_api", "ai_features", "k8s_proxy", "read_observability", "write_observability"]
				}
					`, testGroup.ID, expiresAt.Format(api.Iso8601)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupAccessTokenExists("gitlab_group_access_token.this", &gat),
					testAccCheckGitlabGroupAccessTokenAttributes(&gat, &testAccGitlabGroupAccessTokenExpectedAttributes{
						name:        "my group token",
						scopes:      map[string]bool{"read_repository": true, "api": true, "write_repository": true, "read_api": true, "ai_features": true, "k8s_proxy": true, "read_observability": true, "write_observability": true},
						expiresAt:   expiresAt.Format(api.Iso8601),
						accessLevel: gitlab.AccessLevelValue(gitlab.DeveloperPermissions),
					}),
				),
			},
			// Update the Group Access Token to change the parameters
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my new group token"
				  group = %d
				  expires_at = "%s"
				  access_level = "maintainer"
				  scopes = ["api"]
				}
					`, testGroup.ID, updatedExpiresAt.Format(api.Iso8601)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupAccessTokenExists("gitlab_group_access_token.this", &gat),
					testAccCheckGitlabGroupAccessTokenAttributes(&gat, &testAccGitlabGroupAccessTokenExpectedAttributes{
						name:        "my new group token",
						scopes:      map[string]bool{"read_repository": false, "api": true, "write_repository": false, "read_api": false},
						expiresAt:   updatedExpiresAt.Format(api.Iso8601),
						accessLevel: gitlab.AccessLevelValue(gitlab.MaintainerPermissions),
					}),
				),
			},
			// Update the Group Access Token Access Level to Owner
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my new group token"
				  group = %d
				  expires_at = "%s"
				  access_level = "owner"
				  scopes = ["api"]
				}
					`, testGroup.ID, updatedExpiresAt.Format(api.Iso8601)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupAccessTokenExists("gitlab_group_access_token.this", &gat),
					testAccCheckGitlabGroupAccessTokenAttributes(&gat, &testAccGitlabGroupAccessTokenExpectedAttributes{
						name:        "my new group token",
						scopes:      map[string]bool{"read_repository": false, "api": true, "write_repository": false, "read_api": false},
						expiresAt:   updatedExpiresAt.Format(api.Iso8601),
						accessLevel: gitlab.AccessLevelValue(gitlab.OwnerPermissions),
					}),
				),
			},
			// Add a CICD variable with Group Access Token value
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my new group token"
				  group = %[1]d
				  expires_at = "%[2]s"
				  access_level = "maintainer"
				  scopes = ["api"]
				}

				resource "gitlab_group_variable" "var" {
				  group   = %[1]d
				  key     = "my_grp_access_token"
				  value   = gitlab_group_access_token.this.token
				}

				`, testGroup.ID, updatedExpiresAt.Format(api.Iso8601)),
				// We aren't going to explicitly check the `gitlab_group_variable` because it's not part of our
				// test other than it existing and the fact that TF doesn't error means it was created properly.
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupAccessTokenExists("gitlab_group_access_token.this", &gat),
					testAccCheckGitlabGroupAccessTokenAttributes(&gat, &testAccGitlabGroupAccessTokenExpectedAttributes{
						name:        "my new group token",
						scopes:      map[string]bool{"read_repository": false, "api": true, "write_repository": false, "read_api": false},
						expiresAt:   updatedExpiresAt.Format(api.Iso8601),
						accessLevel: gitlab.AccessLevelValue(gitlab.MaintainerPermissions),
					}),
				),
			},
			//Restore Group Access Token initial parameters
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my group token"
				  group = %d
				  expires_at = "%s"
				  access_level = "developer"
				  scopes = ["read_repository" , "api", "write_repository", "read_api", "ai_features", "k8s_proxy", "read_observability", "write_observability"]
				}
					`, testGroup.ID, expiresAt.Format(api.Iso8601)),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupAccessTokenExists("gitlab_group_access_token.this", &gat),
					testAccCheckGitlabGroupAccessTokenAttributes(&gat, &testAccGitlabGroupAccessTokenExpectedAttributes{
						name:        "my group token",
						scopes:      map[string]bool{"read_repository": true, "api": true, "write_repository": true, "read_api": true, "ai_features": true, "k8s_proxy": true, "read_observability": true, "write_observability": true},
						expiresAt:   expiresAt.Format(api.Iso8601),
						accessLevel: gitlab.AccessLevelValue(gitlab.DeveloperPermissions),
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// the token is only known during creating. We explicitly mention this limitation in the docs.
					"token",
				},
			},
		},
	})
}

// This test checks that when a date is sufficiently in the future, the token
// will rotate automatically. This is done by mocking the time using
// the `GITLAB_TESTING_TIME` environment variable, which is parsed by
// `api.CurrentTime()`, which the resource uses instead of time.Now()
func TestAccGitlabGroupAccessToken_rotationUsingDate(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	futureDate := testutil.GetCurrentTimestampPlusDays(t, 10).Format(time.RFC3339)
	tokenToCheck := ""

	// Not parallel since "os.Setenv" leaks test state otherwise.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_group_access_token.this", "token", func(value string) error {
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
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 3 days, that rotates after 1 days
					rotation_configuration = {
						expiration_days = 3
						rotate_before_days = 1
					}
				}
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "rotation_configuration.expiration_days", "3"),
					resource.TestCheckResourceAttrWith("gitlab_group_access_token.this", "token", func(value string) error {
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
				ResourceName:      "gitlab_group_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

// This test checks an issue where using the `expires` to change the rotation would only work once.
// This is because the primary ID of the token was only stored on create, so after the first rotation,
// it attempts to re-use that primary key, which was already expired.
// It may look like the `basic` test covers this, but because `scopes` changes, that forces new and restarts
// the counter for when the error occurs.
func TestAccGitlabGroupAccessToken_rotationUsingExpiresAt(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]

	initialExpires := testutil.GetCurrentTimePlusDays(t, 10).String()
	updatedExpires := testutil.GetCurrentTimePlusDays(t, 20).String()
	secondUpdateExpires := testutil.GetCurrentTimePlusDays(t, 30).String()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a Group and a Group Access Token
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my group token"
				  group = %d
				  expires_at = "%s"
				  access_level = "developer"
				  scopes = ["api"]
				}
					`, testGroup.ID, initialExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", initialExpires),
				),
			},
			// Update the Group Access Token to change expires
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
				  name = "my new group token 2"
				  group = %d
				  expires_at = "%s"
				  access_level = "maintainer"
				  scopes = ["api"]
				}
					`, testGroup.ID, updatedExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", updatedExpires),
				),
			},
			// Update the Group Access Token once more to change expires a final time
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my new group token 3"
					group = %d
					expires_at = "%s"
					access_level = "maintainer"
					scopes = ["api"]
				}
					`, testGroup.ID, secondUpdateExpires),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", secondUpdateExpires),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_access_token.this",
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
func TestAccGitlabGroupAccessToken_rotationUsingExpiresAtTimeOffset(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]

	// lintignore:AT004  // we need the provider configuration for the time provider
	configString := `
		provider "time" {}

		resource "time_offset" "year" {
			offset_days = %d
		}

		resource "gitlab_group_access_token" "token" {
			group = %d
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
			// Create a Group Access Token
			{
				Config: fmt.Sprintf(configString, 360, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_access_token.token", "expires_at"),
				),
			},
			// Taint the timeoffset to have it re-create the token
			{
				Config: fmt.Sprintf(configString, 365, testGroup.ID),
				Taint:  []string{"time_offset.year"},
			},
			// Re-run the config for a Group Access Token
			{
				Config: fmt.Sprintf(configString, 365, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_access_token.token", "expires_at"),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_access_token.token",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
		},
	})
}

func TestAccGitlabGroupAccessToken_rotationConfiguration(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Create a basic access token.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 10 days, that rotates after 1 day
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}

				}
				`, group.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 10).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different expiration. The higher expiration should trigger a rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 20 days, that rotates immediately
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 30
					}

				}
				`, group.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 20).String()),
				),
			},
			// Verify upstream resource with an import.
			{
				ResourceName:      "gitlab_group_access_token.this",
				ImportState:       true,
				ImportStateVerify: true,
				// The token is only known during creating. We explicitly mention this limitation in the docs.
				ImportStateVerifyIgnore: []string{"token", "rotation_configuration"},
			},
			// Recreate the access token with a different rotation. The lower expiration should not trigger another rotation.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 20 days, that rotates 1 day before it expires
					rotation_configuration = {
						expiration_days = 20
						rotate_before_days = 2
					}

				}
				`, group.ID),
				// Check computed and default attributes.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "active", "true"),
					resource.TestCheckResourceAttr("gitlab_group_access_token.this", "expires_at", testutil.GetCurrentTimePlusDays(t, 20).String()),
				),
			},
		},
	})
}

func TestAccGitlabGroupAccessToken_attributeValidation(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupAccessTokenDestroy,
		Steps: []resource.TestStep{
			// Validate expires_at and rotation_configuration conflict
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d
					expires_at = %s

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = 10
						rotate_before_days = 1
					}

				}
				`, group.ID, testutil.GetCurrentTimePlusDays(t, 2).String()), // so it's always in the future.
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// At least one rotation_configuration or expires_at is required
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

				}
				`, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
			// Validate that expiration must be > 0
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = -1
						rotate_before_days = 1
					}

				}
				`, group.ID),
				ExpectError: regexp.MustCompile("rotation_configuration.expiration_days value must be at least 1"),
			},
			// Validate that rotate_before_days must be > 0
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_access_token" "this" {
					name = "my group token"
					group = %d

					access_level = "developer"
					scopes = ["api"]

					// Create a token good for 10 days, that rotates after 9 days
					rotation_configuration = {
						expiration_days = 1
						rotate_before_days = -1
					}

				}
				`, group.ID),
				ExpectError: regexp.MustCompile("rotation_configuration.rotate_before_days value must be at least 1"),
			},
		},
	})
}

func testAccCheckGitlabGroupAccessTokenExists(n string, gat *testAccGitlabGroupAccessTokenWrapper) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		group, tokenString, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error parsing ID: %s", rs.Primary.ID)
		}
		groupAccessTokenID, err := strconv.Atoi(tokenString)
		if err != nil {
			return fmt.Errorf("%s cannot be converted to int", tokenString)
		}

		groupId := rs.Primary.Attributes["group"]
		if groupId == "" {
			return fmt.Errorf("No group ID is set")
		}
		if groupId != group {
			return fmt.Errorf("Group [%s] in group identifier [%s] it's different from group stored into the state [%s]", group, rs.Primary.ID, groupId)
		}

		tokens, _, err := testutil.TestGitlabClient.GroupAccessTokens.ListGroupAccessTokens(groupId, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.ID == groupAccessTokenID {
				gat.groupAccessToken = token
				gat.group = groupId
				gat.token = rs.Primary.Attributes["token"]
				return nil
			}
		}
		return fmt.Errorf("Group Access Token does not exist")
	}
}

type testAccGitlabGroupAccessTokenExpectedAttributes struct {
	name        string
	scopes      map[string]bool
	expiresAt   string
	accessLevel gitlab.AccessLevelValue
}

type testAccGitlabGroupAccessTokenWrapper struct {
	groupAccessToken *gitlab.GroupAccessToken
	group            string
	token            string
}

func testAccCheckGitlabGroupAccessTokenAttributes(gatWrap *testAccGitlabGroupAccessTokenWrapper, want *testAccGitlabGroupAccessTokenExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		gat := gatWrap.groupAccessToken
		if gat.Name != want.name {
			return fmt.Errorf("got Name %q; want %q", gat.Name, want.name)
		}

		if gat.ExpiresAt.String() != want.expiresAt {
			return fmt.Errorf("got ExpiresAt %q; want %q", gat.ExpiresAt.String(), want.expiresAt)
		}

		if gat.AccessLevel != want.accessLevel {
			return fmt.Errorf("got AccessLevel %q; want %q", gat.AccessLevel, want.accessLevel)
		}

		for _, scope := range gat.Scopes {
			if !want.scopes[scope] {
				return fmt.Errorf("got a not wanted Scope %q, received %v", scope, gat.Scopes)
			}
			want.scopes[scope] = false
		}
		for k, v := range want.scopes {
			if v {
				return fmt.Errorf("not got a wanted Scope %q, received %v", k, gat.Scopes)
			}
		}

		git, err := gitlab.NewClient(gatWrap.token, gitlab.WithBaseURL(testutil.TestGitlabClient.BaseURL().String()))
		if err != nil {
			return fmt.Errorf("Cannot use the token to instantiate a new client %s", err)
		}
		_, _, err = git.Groups.GetGroup(gatWrap.group, nil)
		if err != nil {
			return fmt.Errorf("Cannot use the token to perform an API call %s", err)
		}

		return nil
	}
}

func testAccCheckGitlabGroupAccessTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_access_token" {
			continue
		}

		group := rs.Primary.Attributes["group"]
		name := rs.Primary.Attributes["name"]

		tokens, _, err := testutil.TestGitlabClient.GroupAccessTokens.ListGroupAccessTokens(group, nil)
		if err != nil {
			return err
		}

		for _, token := range tokens {
			if token.Name == name {
				return fmt.Errorf("group %q access token with name %q still exists", group, name)
			}
		}
	}

	return nil
}
