//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAccGitlabProjectIntegrationMicrosoftTeams_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with all event fields set to false and branches_to_be_notified="all"
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
						push_events                  = false
						issues_events                = false
						confidential_issues_events   = false
						merge_requests_events        = false
						tag_push_events              = false
						note_events                  = false
						confidential_note_events     = false
						pipeline_events              = false
						wiki_page_events             = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "wiki_page_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.test", "created_at"),
				),
			},
			// Step 2: Import verification ignoring webhook
			{
				ResourceName:      "gitlab_project_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 3: Update to all true with branches_to_be_notified="default"
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://testurl.com/?token=5"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "default"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "webhook", "https://testurl.com/?token=5"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "pipeline_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "wiki_page_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
						push_events                  = false
						issues_events                = false
						confidential_issues_events   = false
						merge_requests_events        = false
						tag_push_events              = false
						note_events                  = false
						confidential_note_events     = false
						pipeline_events              = false
						wiki_page_events             = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "wiki_page_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_microsoft_teams.test", "active", "true"),
				),
			},
			// Step 6: Final import verification
			{
				ResourceName:      "gitlab_project_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationMicrosoftTeams_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with all event fields set to false and branches_to_be_notified="all"
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
						push_events                  = false
						issues_events                = false
						confidential_issues_events   = false
						merge_requests_events        = false
						tag_push_events              = false
						note_events                  = false
						confidential_note_events     = false
						pipeline_events              = false
						wiki_page_events             = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "wiki_page_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_microsoft_teams.test", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_microsoft_teams.test", "created_at"),
				),
			},
			// Step 2: Import verification ignoring webhook
			{
				ResourceName:      "gitlab_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 3: Update to all true with branches_to_be_notified="default"
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://testurl.com/?token=5"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "default"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "webhook", "https://testurl.com/?token=5"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "pipeline_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "wiki_page_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_microsoft_teams.test", "updated_at"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 5: Update back to original values
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
						push_events                  = false
						issues_events                = false
						confidential_issues_events   = false
						merge_requests_events        = false
						tag_push_events              = false
						note_events                  = false
						confidential_note_events     = false
						pipeline_events              = false
						wiki_page_events             = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "wiki_page_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_microsoft_teams.test", "active", "true"),
				),
			},
			// Step 6: Final import verification
			{
				ResourceName:      "gitlab_integration_microsoft_teams.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationMicrosoftTeams_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with SDK provider (version 18.9.0)
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.9.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.test", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "test" {
						project                      = %d
						webhook                      = "https://test.com/?token=4"
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.test", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_microsoft_teams.test",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

// TestAccGitlabProjectIntegrationMicrosoftTeams_stateMove verifies that the moved block works
// when migrating from gitlab_integration_microsoft_teams to gitlab_project_integration_microsoft_teams.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationMicrosoftTeams_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Create a Microsoft Teams integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_microsoft_teams" "old" {
					project                      = %d
					webhook                      = "https://test.com/?token=4"
					notify_only_broken_pipelines = false
					branches_to_be_notified      = "all"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_microsoft_teams.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_microsoft_teams" "new" {
					project                      = %d
					webhook                      = "https://test.com/?token=4"
					notify_only_broken_pipelines = false
					branches_to_be_notified      = "all"
				}

				moved {
					from = gitlab_integration_microsoft_teams.old
					to   = gitlab_project_integration_microsoft_teams.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_microsoft_teams.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_microsoft_teams.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook"},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_microsoft_teams" && rs.Type != "gitlab_project_integration_microsoft_teams" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetMicrosoftTeamsService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("Microsoft Teams integration for project %s is still active", project)
			}
		}
		if !api.Is404(err) {
			return err
		}
	}
	return nil
}
