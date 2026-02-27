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

func TestAccGitlabProjectIntegrationMattermost_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMattermostDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a project and a mattermost integration with minimal settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "mattermost" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 3: Update mattermost integration with more settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://test.com"
						username                     = "test"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "all"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
						push_channel                 = "test"
						issue_channel                = "test"
						confidential_issue_channel   = "test"
						merge_request_channel        = "test"
						note_channel                 = "test"
						confidential_note_channel    = "test"
						tag_push_channel             = "test"
						pipeline_channel             = "test"
						wiki_page_channel            = "test"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 5: Update the mattermost integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://testwebhook.com"
						username                     = "test username"
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
						push_channel                 = "test push_channel"
						issue_channel                = "test issue_channel"
						confidential_issue_channel   = "test confidential_issue_channel"
						merge_request_channel        = "test merge_request_channel"
						note_channel                 = "test note_channel"
						confidential_note_channel    = "test note_channel"
						tag_push_channel             = "test tag_push_channel"
						pipeline_channel             = "test pipeline_channel"
						wiki_page_channel            = "test wiki_page_channel"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "webhook", "https://testwebhook.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_channel", "test push_channel"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "notify_only_broken_pipelines", "false"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_project_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 7: Update the mattermost integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://test.com"
						username                     = "test"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "all"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
						push_channel                 = "test"
						issue_channel                = "test"
						confidential_issue_channel   = "test"
						merge_request_channel        = "test"
						note_channel                 = "test"
						confidential_note_channel    = "test"
						tag_push_channel             = "test"
						pipeline_channel             = "test"
						wiki_page_channel            = "test"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			// Step 8: Import verification
			{
				ResourceName:      "gitlab_project_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 9: Update the mattermost integration to get back to minimal settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "mattermost" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			// Step 10: Verify Import
			{
				ResourceName:      "gitlab_project_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationMattermost_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMattermostDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a project and a mattermost integration with minimal settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_mattermost" "mattermost" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 3: Update mattermost integration with more settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://test.com"
						username                     = "test"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "all"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
						push_channel                 = "test"
						issue_channel                = "test"
						confidential_issue_channel   = "test"
						merge_request_channel        = "test"
						note_channel                 = "test"
						confidential_note_channel    = "test"
						tag_push_channel             = "test"
						pipeline_channel             = "test"
						wiki_page_channel            = "test"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 5: Update the mattermost integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://testwebhook.com"
						username                     = "test username"
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
						push_channel                 = "test push_channel"
						issue_channel                = "test issue_channel"
						confidential_issue_channel   = "test confidential_issue_channel"
						merge_request_channel        = "test merge_request_channel"
						note_channel                 = "test note_channel"
						confidential_note_channel    = "test note_channel"
						tag_push_channel             = "test tag_push_channel"
						pipeline_channel             = "test pipeline_channel"
						wiki_page_channel            = "test wiki_page_channel"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://testwebhook.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test push_channel"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "false"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 7: Update the mattermost integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_mattermost" "mattermost" {
						project                      = "%d"
						webhook                      = "https://test.com"
						username                     = "test"
						notify_only_broken_pipelines = true
						branches_to_be_notified      = "all"
						push_events                  = true
						issues_events                = true
						confidential_issues_events   = true
						merge_requests_events        = true
						tag_push_events              = true
						note_events                  = true
						confidential_note_events     = true
						pipeline_events              = true
						wiki_page_events             = true
						push_channel                 = "test"
						issue_channel                = "test"
						confidential_issue_channel   = "test"
						merge_request_channel        = "test"
						note_channel                 = "test"
						confidential_note_channel    = "test"
						tag_push_channel             = "test"
						pipeline_channel             = "test"
						wiki_page_channel            = "test"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			// Step 8: Import verification
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Step 9: Update the mattermost integration to get back to minimal settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_mattermost" "mattermost" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			// Step 10: Verify Import
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationMattermost_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationMattermostDestroy,
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
					resource "gitlab_project_integration_mattermost" "test" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_mattermost.test", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_mattermost" "test" {
						project = "%d"
						webhook = "https://test.com"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_mattermost.test", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_mattermost.test",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

// TestAccGitlabProjectIntegrationMattermost_stateMove verifies that the moved block works
// when migrating from gitlab_integration_mattermost to gitlab_project_integration_mattermost.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationMattermost_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationMattermostDestroy,
		Steps: []resource.TestStep{
			// Create a Mattermost integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_mattermost" "old" {
					project = "%d"
					webhook = "https://test.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_mattermost.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_mattermost" "new" {
					project = "%d"
					webhook = "https://test.com"
				}

				moved {
					from = gitlab_integration_mattermost.old
					to   = gitlab_project_integration_mattermost.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_mattermost.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_mattermost.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook"},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationMattermostDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_mattermost" && rs.Type != "gitlab_project_integration_mattermost" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetMattermostService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("Mattermost integration for project %s is still active", project)
			}
		}
		if !api.Is404(err) {
			return err
		}
	}
	return nil
}
