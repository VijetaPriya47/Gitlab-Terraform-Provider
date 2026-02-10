//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupIntegrationMattermost_basic(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupIntegrationMattermostDestroy,
		Steps: []resource.TestStep{
			// Create with minimal settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_integration_mattermost" "mattermost" {
						group   = "%d"
						webhook = "https://mattermost.example.com/hooks/test"
					}
				`, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "webhook", "https://mattermost.example.com/hooks/test"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "push_events", "true"),
				),
			},
			// Import
			{
				ResourceName:      "gitlab_group_integration_mattermost.mattermost",
				ImportStateId:     fmt.Sprintf("%d", testGroup.ID),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook", // Webhook is sensitive and not returned by API
				},
			},
			// Update with more settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_integration_mattermost" "mattermost" {
						group                          = "%d"
						webhook                        = "https://mattermost.example.com/hooks/updated"
						username                       = "terraform-bot"
						channel                        = "#notifications"
						notify_only_broken_pipelines   = true
						branches_to_be_notified        = "all"
						push_events                    = false
						issues_events                  = true
						merge_requests_events          = true
						tag_push_events                = false
						note_events                    = true
						pipeline_events                = true
						wiki_page_events               = false
						push_channel                   = "#push"
						issue_channel                  = "#issues"
						merge_request_channel          = "#merge-requests"
						pipeline_channel               = "#pipelines"
						labels_to_be_notified          = "urgent,critical"
						labels_to_be_notified_behavior = "match_any"
						use_inherited_settings         = false
					}
				`, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "webhook", "https://mattermost.example.com/hooks/updated"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "username", "terraform-bot"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "channel", "#notifications"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "push_channel", "#push"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "labels_to_be_notified", "urgent,critical"),
					resource.TestCheckResourceAttr("gitlab_group_integration_mattermost.mattermost", "labels_to_be_notified_behavior", "match_any"),
				),
			},
			// Import after update
			{
				ResourceName:      "gitlab_group_integration_mattermost.mattermost",
				ImportStateId:     fmt.Sprintf("%d", testGroup.ID),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"webhook",                // Webhook is sensitive and not returned by API
					"use_inherited_settings", // Write-only, not returned by API
				},
			},
		},
	})
}

func testAccCheckGitlabGroupIntegrationMattermostDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_integration_mattermost" {
			continue
		}

		groupID := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Integrations.GetGroupMattermostIntegration(groupID)
		if err == nil && service.Active {
			return fmt.Errorf("Mattermost Integration in group %s still exists and is active", groupID)
		}
	}
	return nil
}
