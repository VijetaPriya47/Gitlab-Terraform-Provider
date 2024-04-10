//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

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
		CheckDestroy: testAccCheckGitlabPipelineScheduleDestroy,
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
						"read_repository",
						"write_repository",
						"read_registry",
						"write_registry",
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
