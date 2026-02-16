//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAccGitlabProjectIntegrationGithub_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with token and repository_url
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
						static_context = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "static_context", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Step 3: Update to different repository_url and static_context
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/terraform-providers/terraform-provider-github"
						static_context = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "repository_url", "https://github.com/terraform-providers/terraform-provider-github"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "static_context", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
						static_context = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "static_context", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.test", "active", "true"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_project_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationGithub_basic_deprecated(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with deprecated resource name
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
						static_context = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "static_context", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_github.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_github.test", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_github.test", "created_at"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Step 3: Update to different repository_url and static_context
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/terraform-providers/terraform-provider-github"
						static_context = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "repository_url", "https://github.com/terraform-providers/terraform-provider-github"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "static_context", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_github.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
						static_context = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "static_context", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_github.test", "active", "true"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_integration_github.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationGithub_migrateFromSDKToFramework(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with SDK provider (version 18.8.1)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.8.1",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "test" {
						project        = %d
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.test", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_github.test",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationGithub_stateMove verifies that the moved block works
// when migrating from gitlab_integration_github to gitlab_project_integration_github.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationGithub_stateMove(t *testing.T) {
	testutil.SkipIfCE(t)
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Create a Github integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_github" "old" {
					project        = %d
					token          = "test"
					repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					static_context = true
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_github.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_github" "new" {
					project        = %d
					token          = "test"
					repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					static_context = true
				}

				moved {
					from = gitlab_integration_github.old
					to   = gitlab_project_integration_github.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_github.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_github.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationGithubDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_github" && rs.Type != "gitlab_project_integration_github" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetGithubService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("GitHub integration for project %s is still active", project)
			}
		}
	}
	return nil
}
