//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabIntegrationMattermost_basic(t *testing.T) {
	var mattermostService gitlab.MattermostService
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabServiceMattermostDestroy,
		Steps: []resource.TestStep{
			// Create a project and a mattermost integration with minimal settings
			{
				Config: fmt.Sprintf(`
                                    resource "gitlab_integration_mattermost" "mattermost" {
                                        project = "%d"
                                        webhook = "https://test.com"
                                        }`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabMattermostIntegrationExists("gitlab_integration_mattermost.mattermost", &mattermostService),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			{

				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportStateIdFunc: getMattermostProjectID("gitlab_integration_mattermost.mattermost"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Update mattermost integration with more settings
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
									}`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabMattermostIntegrationExists("gitlab_integration_mattermost.mattermost", &mattermostService),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportStateIdFunc: getMattermostProjectID("gitlab_integration_mattermost.mattermost"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Update the mattermost integration
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
									}`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabMattermostIntegrationExists("gitlab_integration_mattermost.mattermost", &mattermostService),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://testwebhook.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test push_channel"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "false"),
				),
			},
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportStateIdFunc: getMattermostProjectID("gitlab_integration_mattermost.mattermost"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Update the mattermost integration to get back to previous settings
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
									}`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabMattermostIntegrationExists("gitlab_integration_mattermost.mattermost", &mattermostService),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "push_channel", "test"),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
				),
			},
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportStateIdFunc: getMattermostProjectID("gitlab_integration_mattermost.mattermost"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
			// Update the mattermost integration to get back to minimal settings
			{
				Config: fmt.Sprintf(`
									resource "gitlab_integration_mattermost" "mattermost" {
									  project                      = "%d"
									  webhook                      = "https://test.com"
									}
									`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabMattermostIntegrationExists("gitlab_integration_mattermost.mattermost", &mattermostService),
					resource.TestCheckResourceAttr("gitlab_integration_mattermost.mattermost", "webhook", "https://test.com"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_mattermost.mattermost",
				ImportStateIdFunc: getMattermostProjectID("gitlab_integration_mattermost.mattermost"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",
				},
			},
		},
	})
}

func testAccCheckGitlabMattermostIntegrationExists(n string, service *gitlab.MattermostService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}
		mattermostService, _, err := testutil.TestGitlabClient.Services.GetMattermostService(project)
		if err != nil {
			return fmt.Errorf("mattermost integration does not exist in project %s: %v", project, err)
		}
		*service = *mattermostService

		return nil
	}
}

func testAccCheckGitlabServiceMattermostDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_mattermost" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetMattermostService(project)
		if err == nil {
			// If the service is still active, throw an error. It wasn't properly deleted.
			if service.Active {
				return fmt.Errorf("Mattermost Integration in project %s still exists", project)
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

func getMattermostProjectID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return "", fmt.Errorf("No project ID is set")
		}

		return project, nil
	}
}
