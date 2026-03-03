//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabGroupIntegrationMicrosoftTeams_basic(t *testing.T) {

	group := testutil.CreateGroups(t, 1)[0]
	groupPath := group.FullPath

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupIntegrationMicrosoftTeams_CheckDestroy(),
		Steps: []resource.TestStep{
			// Create Microsoft Teams group integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_integration_microsoft_teams" "test" {
					group = "%s"
					webhook = "https://example.com/webhook"
					
					notify_only_broken_pipelines = true
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
				`, groupPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_integration_microsoft_teams.test", "id"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "group", groupPath),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "webhook", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "wiki_page_events", "false"),
				),
			},

			// Verify upstream attributes
			{
				ResourceName:            "gitlab_group_integration_microsoft_teams.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook", "notify_only_broken_pipelines", "branches_to_be_notified"},
			},

			// Update the Microsoft Teams group integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_integration_microsoft_teams" "test" {
					group = "%s"
					webhook = "https://example.com/webhook"
					
					notify_only_broken_pipelines = false
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
				}
				`, groupPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_integration_microsoft_teams.test", "id"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "group", groupPath),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "webhook", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "confidential_issues_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "merge_requests_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "confidential_note_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "pipeline_events", "true"),
					resource.TestCheckResourceAttr("gitlab_group_integration_microsoft_teams.test", "wiki_page_events", "true"),
				),
			},

			// Verify upstream attributes after the update
			{
				ResourceName:            "gitlab_group_integration_microsoft_teams.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"webhook", "notify_only_broken_pipelines", "branches_to_be_notified"},
			},
		},
	})

}

func testAcc_GitlabGroupIntegrationMicrosoftTeams_CheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "gitlab_group_integration_microsoft_teams" {
				continue
			}

			groupID := rs.Primary.ID
			integration, _, err := testutil.TestGitlabClient.Integrations.GetGroupMicrosoftTeamsNotifications(groupID)

			if err != nil {
				if api.Is404(err) {
					continue
				}
				return err
			}

			if integration.Active == true {
				return fmt.Errorf("Microsoft Teams group integration %s still exists", groupID)
			}
		}
		return nil
	}
}
