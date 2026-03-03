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

func TestAccGitlabProjectIntegrationPipelinesEmail_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationPipelinesEmailDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with single recipient
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "recipients.#", "1"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_pipelines_email.test", "id"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_project_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update to multiple recipients and different settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_pipelines_email" "test" {
						project                      = %d
						recipients                   = ["test@example.com", "test2@example.com"]
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "recipients.#", "2"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_integration_pipelines_email.test", "recipients.*", "test2@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "branches_to_be_notified", "all"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_project_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 5: Update back to original
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "recipients.#", "1"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_pipelines_email.test", "branches_to_be_notified", "default"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_project_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationPipelinesEmail_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationPipelinesEmailDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with single recipient using deprecated resource name
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "recipients.#", "1"),
					resource.TestCheckTypeSetElemAttr("gitlab_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttrSet("gitlab_integration_pipelines_email.test", "id"),
				),
			},
			// Step 2: Import verification
			{
				ResourceName:      "gitlab_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update to multiple recipients and different settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_pipelines_email" "test" {
						project                      = %d
						recipients                   = ["test@example.com", "test2@example.com"]
						notify_only_broken_pipelines = false
						branches_to_be_notified      = "all"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "recipients.#", "2"),
					resource.TestCheckTypeSetElemAttr("gitlab_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckTypeSetElemAttr("gitlab_integration_pipelines_email.test", "recipients.*", "test2@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "notify_only_broken_pipelines", "false"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "branches_to_be_notified", "all"),
				),
			},
			// Step 4: Import verification
			{
				ResourceName:      "gitlab_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 5: Update back to original
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "recipients.#", "1"),
					resource.TestCheckTypeSetElemAttr("gitlab_integration_pipelines_email.test", "recipients.*", "test@example.com"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_integration_pipelines_email.test", "branches_to_be_notified", "default"),
				),
			},
			// Step 6: Import verification
			{
				ResourceName:      "gitlab_integration_pipelines_email.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationPipelinesEmail_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectIntegrationPipelinesEmailDestroy,
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
					resource "gitlab_project_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_pipelines_email.test", "id"),
				),
			},
			// Step 2: Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_pipelines_email" "test" {
						project    = %d
						recipients = ["test@example.com"]
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_pipelines_email.test", "id"),
				),
			},
			// Step 3: Import verification with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_integration_pipelines_email.test",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// TestAccGitlabProjectIntegrationPipelinesEmail_stateMove verifies that the moved block works
// when migrating from gitlab_integration_pipelines_email to gitlab_project_integration_pipelines_email.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationPipelinesEmail_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationPipelinesEmailDestroy,
		Steps: []resource.TestStep{
			// Create a Pipelines Email integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_pipelines_email" "old" {
					project    = %d
					recipients = ["test@example.com"]
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_pipelines_email.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_pipelines_email" "new" {
					project    = %d
					recipients = ["test@example.com"]
				}

				moved {
					from = gitlab_integration_pipelines_email.old
					to   = gitlab_project_integration_pipelines_email.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_pipelines_email.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_pipelines_email.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationPipelinesEmailDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_pipelines_email" && rs.Type != "gitlab_project_integration_pipelines_email" {
			continue
		}

		project := rs.Primary.ID

		service, _, err := testutil.TestGitlabClient.Services.GetPipelinesEmailService(project)
		if err == nil {
			if service != nil && service.Active {
				return fmt.Errorf("Pipelines Email integration for project %s is still active", project)
			}
		}
	}
	return nil
}
