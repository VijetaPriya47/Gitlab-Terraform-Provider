//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectLevelNotifications_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabProjectLevelNotificationsDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_level_notifications" "foo" {
					project    = "%d"
					level      = "global"
				}
			`, testProject.ID),
				Check: resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "level", "global"),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add one custom notification
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_level_notifications" "foo" {
					project    = "%d"
					level      = "custom"

					new_merge_request = true
				}
			`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "new_merge_request", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add all custom notification
			{
				Config: fmt.Sprintf(`
							resource "gitlab_project_level_notifications" "foo" {
								project    = "%d"
								level      = "custom"
			
								new_note = true
								new_issue = true
								reopen_issue = true
								close_issue = true
								reassign_issue = true
								issue_due = true
								new_merge_request = true
								push_to_merge_request = true
								reopen_merge_request = true
								close_merge_request = true
								reassign_merge_request = true
								merge_merge_request = true
								failed_pipeline = true
								fixed_pipeline = true
								success_pipeline = true
								moved_project = true
								merge_when_pipeline_succeeds = true
							}
						`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "new_note", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "new_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "reopen_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "close_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "reassign_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "issue_due", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "new_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "push_to_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "reopen_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "close_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "reassign_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "merge_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "failed_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "fixed_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "success_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "moved_project", "true"),
					resource.TestCheckResourceAttr("gitlab_project_level_notifications.foo", "merge_when_pipeline_succeeds", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectLevelNotificationsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_level_notifications" {
			continue
		}

		settings, _, err := testutil.TestGitlabClient.NotificationSettings.GetSettingsForProject(rs.Primary.ID)
		if err != nil {
			return err
		}

		if settings.Level.String() != "global" {
			return fmt.Errorf("project level notification set to a non-global value")
		}
	}
	return nil
}
