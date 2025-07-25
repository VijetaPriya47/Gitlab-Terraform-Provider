//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGlobalLevelNotifications_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabGlobalLevelNotificationsDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "gitlab_global_level_notifications" "foo" {
					level      = "watch"
				}
			`,
				Check: resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "level", "watch"),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_global_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add one custom notification
			{
				Config: `
				resource "gitlab_global_level_notifications" "foo" {
					level      = "custom"
					new_merge_request = true
				}
			`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "new_merge_request", "true"),
				),
			},
			{
				ResourceName:      "gitlab_global_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add all custom notification
			{
				Config: `
							resource "gitlab_global_level_notifications" "foo" {
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
						`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "new_note", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "new_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "reopen_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "close_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "reassign_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "issue_due", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "new_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "push_to_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "reopen_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "close_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "reassign_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "merge_merge_request", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "failed_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "fixed_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "success_pipeline", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "moved_project", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.foo", "merge_when_pipeline_succeeds", "true"),
				),
			},
			{
				ResourceName:      "gitlab_global_level_notifications.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGlobalLevelNotifications_withVariables(t *testing.T) {
	// This test validates the fix for the unknown value issue that occurs when using
	// variables with the global level notifications resource. Previously, this would
	// fail with "level must be set to custom" error during planning phase.
	//
	// Note: Using resource.Test (not ParallelTest) because global notification settings
	// are shared across the GitLab instance and parallel tests would interfere with each other.
	resource.Test(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabGlobalLevelNotificationsDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					variable "notification_level" {
					type    = string
					default = "custom"
					}

					variable "close_issue_enabled" {
					type    = bool
					default = true
					}

					variable "new_issue_enabled" {
					type    = bool
					default = false
					}

					resource "gitlab_global_level_notifications" "test" {
					level       = var.notification_level
					close_issue = var.close_issue_enabled
					new_issue   = var.new_issue_enabled
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "close_issue", "true"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "new_issue", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_global_level_notifications.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test updating variables
			{
				Config: `
					variable "notification_level" {
					type    = string
					default = "custom"
					}

					variable "close_issue_enabled" {
					type    = bool
					default = false
					}

					variable "new_issue_enabled" {
					type    = bool
					default = true
					}

					resource "gitlab_global_level_notifications" "test" {
					level       = var.notification_level
					close_issue = var.close_issue_enabled
					new_issue   = var.new_issue_enabled
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "level", "custom"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "close_issue", "false"),
					resource.TestCheckResourceAttr("gitlab_global_level_notifications.test", "new_issue", "true"),
				),
			},
			// Test validation error: level is "participating" but individual notification options are set
			// This should fail with validation error during planning phase
			{
				Config: `
					variable "notification_level" {
					type    = string
					default = "participating"
					}

					variable "close_issue_enabled" {
					type    = bool
					default = true
					}

					resource "gitlab_global_level_notifications" "test" {
					level       = var.notification_level
					close_issue = var.close_issue_enabled
					}
				`,
				ExpectError: regexp.MustCompile(`"level" must be set to "custom" to set individual notification levels`),
			},
		},
	})
}

func testAccCheckGitlabGlobalLevelNotificationsDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_global_level_notifications" {
			continue
		}

		settings, _, err := testutil.TestGitlabClient.NotificationSettings.GetGlobalSettings()
		if err != nil {
			return err
		}

		if settings.Level.String() != "disabled" {
			return fmt.Errorf("notifications are disabled")
		}
	}
	return nil
}
