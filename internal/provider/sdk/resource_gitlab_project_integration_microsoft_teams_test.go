//go:build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectIntegrationMicrosoftTeams_basic(t *testing.T) {
	var teamsService gitlab.MicrosoftTeamsService
	project := testutil.CreateProject(t)
	teamsResourceName := "gitlab_project_integration_microsoft_teams.teams"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Create a project and a teams integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "false"),
				),
			},
			// Update the teams integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://testurl.com/?token=5"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "true"),
				),
			},
			// Update the teams integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "false"),
				),
			},
			{
				ResourceName:      teamsResourceName,
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
	var teamsService gitlab.MicrosoftTeamsService
	project := testutil.CreateProject(t)
	teamsResourceName := "gitlab_integration_microsoft_teams.teams"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy,
		Steps: []resource.TestStep{
			// Create a project and a teams integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "false"),
				),
			},
			// Update the teams integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://testurl.com/?token=5"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "true"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "true"),
				),
			},
			// Update the teams integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_microsoft_teams" "teams" {
						project                      = "%d"
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
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(teamsResourceName, &teamsService),
					resource.TestCheckResourceAttr(teamsResourceName, "webhook", "https://test.com/?token=4"),
					resource.TestCheckResourceAttr(teamsResourceName, "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr(teamsResourceName, "push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "merge_requests_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "tag_push_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "confidential_note_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "pipeline_events", "false"),
					resource.TestCheckResourceAttr(teamsResourceName, "wiki_page_events", "false"),
				),
			},
			{
				ResourceName:      teamsResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationMicrosoftTeamsExists(n string, service *gitlab.MicrosoftTeamsService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}
		teamsService, _, err := testutil.TestGitlabClient.Services.GetMicrosoftTeamsService(project)
		if err != nil {
			return fmt.Errorf("Microsoft Teams integration does not exist in project %s: %v", project, err)
		}
		*service = *teamsService

		return nil
	}
}

func testAccCheckGitlabProjectIntegrationMicrosoftTeamsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_integration_microsoft_teams" && rs.Type != "gitlab_integration_microsoft_teams" {
			continue
		}
		gotInt, resp, err := testutil.TestGitlabClient.Services.GetMicrosoftTeamsService(rs.Primary.ID, nil)
		if err == nil {
			if gotInt != nil && fmt.Sprintf("%d", gotInt.ID) == rs.Primary.ID {
				return fmt.Errorf("Integration still exists")
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}
