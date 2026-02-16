//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAcc_GitlabProjectIntegrationEmailsOnPush_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy,
		Steps: []resource.TestStep{
			// Create an Emails on Push integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project    = %d
					recipients = "mynumberonerecipient@example.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "mynumberonerecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "slug"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Emails on Push integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project    = %d
					recipients = "mynumbertworecipient@example.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "mynumbertworecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "updated_at"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationEmailsOnPush_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy,
		Steps: []resource.TestStep{
			// Create an Emails on Push integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_emails_on_push" "this" {
					project    = %d
					recipients = "mynumberonerecipient@example.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.this", "recipients", "mynumberonerecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.this", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "slug"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Emails on Push integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_emails_on_push" "this" {
					project    = %d
					recipients = "mynumbertworecipient@example.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.this", "recipients", "mynumbertworecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.this", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.this", "updated_at"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationEmailsOnPush_allAttributes(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy,
		Steps: []resource.TestStep{
			// Create an Emails on Push integration with all attributes
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project                   = %d
					recipients                = "test1@example.com test2@example.com"
					disable_diffs             = true
					send_from_committer_email = true
					push_events               = true
					tag_push_events           = true
					branches_to_be_notified   = "protected"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "test1@example.com test2@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "disable_diffs", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "send_from_committer_email", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "tag_push_events", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "branches_to_be_notified", "protected"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "slug"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "created_at"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to different values
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project                   = %d
					recipients                = "updated@example.com"
					disable_diffs             = false
					send_from_committer_email = false
					push_events               = false
					tag_push_events           = false
					branches_to_be_notified   = "default"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "updated@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "disable_diffs", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "send_from_committer_email", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "updated_at"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_integration_emails_on_push.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationEmailsOnPush_UpgradeFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy,
		Steps: []resource.TestStep{
			// Create resource with SDK version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 18.7.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project                   = %d
					recipients                = "sdk@example.com"
					disable_diffs             = true
					send_from_committer_email = false
					push_events               = false
					tag_push_events           = false
					branches_to_be_notified   = "all"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "sdk@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "disable_diffs", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "send_from_committer_email", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "active", "true"),
				),
			},
			// Verify resource works with Framework version without changes
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "this" {
					project                   = %d
					recipients                = "sdk@example.com"
					disable_diffs             = true
					send_from_committer_email = false
					push_events               = false
					tag_push_events           = false
					branches_to_be_notified   = "all"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "recipients", "sdk@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "disable_diffs", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "send_from_committer_email", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "branches_to_be_notified", "all"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.this", "active", "true"),
				),
			},
			// Verify import with Framework version
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_emails_on_push.this",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationEmailsOnPush_stateMove verifies that the moved block works
// when migrating from gitlab_integration_emails_on_push to gitlab_project_integration_emails_on_push.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationEmailsOnPush_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy,
		Steps: []resource.TestStep{
			// Create a Emails On Push integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_emails_on_push" "old" {
					project    = %d
					recipients = "mynumberonerecipient@example.com"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.old", "id"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.old", "recipients", "mynumberonerecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_emails_on_push.old", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.old", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.old", "title"),
					resource.TestCheckResourceAttrSet("gitlab_integration_emails_on_push.old", "slug"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_emails_on_push" "new" {
					project    = %d
					recipients = "mynumberonerecipient@example.com"
				}

				moved {
					from = gitlab_integration_emails_on_push.old
					to   = gitlab_project_integration_emails_on_push.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.new", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.new", "recipients", "mynumberonerecipient@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_emails_on_push.new", "active", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.new", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.new", "title"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_emails_on_push.new", "slug"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_emails_on_push.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGitlabProjectIntegrationEmailsOnPushCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_integration_emails_on_push" || rs.Type == "gitlab_project_integration_emails_on_push" {
			project := rs.Primary.ID

			service, _, err := testutil.TestGitlabClient.Services.GetEmailsOnPushService(project)
			if err == nil {
				if service != nil && service.Active != false {
					return errors.New("Emails on Push integration still exists")
				}
			}
		}
	}
	return nil
}
