//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabPipelineTrigger_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabPipelineTriggerDestroy,
		Steps: []resource.TestStep{
			// Create a pipeline trigger
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_trigger" "test" {
						project     = "%d"
						description = "External Pipeline Trigger"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "description", "External Pipeline Trigger"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "pipeline_trigger_id"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "token"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_pipeline_trigger.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the pipeline trigger
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_trigger" "test" {
						project     = "%d"
						description = "Trigger"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "description", "Trigger"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "pipeline_trigger_id"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "token"),
				),
			},
			// Verify import after update
			{
				ResourceName:            "gitlab_pipeline_trigger.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update back to original description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_trigger" "test" {
						project     = "%d"
						description = "External Pipeline Trigger"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_pipeline_trigger.test", "description", "External Pipeline Trigger"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "pipeline_trigger_id"),
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "token"),
				),
			},
			// Verify import after second update
			{
				ResourceName:            "gitlab_pipeline_trigger.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabPipelineTrigger_migrateFromSDKToFramework(t *testing.T) {
	testutil.SkipIfCE(t)

	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabPipelineTriggerDestroy,
		Steps: []resource.TestStep{
			// Create with SDK provider
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.10.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_trigger" "test" {
						project     = "%d"
						description = "External Pipeline Trigger"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "id"),
				),
			},
			// Migrate to Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_pipeline_trigger" "test" {
						project     = "%d"
						description = "External Pipeline Trigger"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_pipeline_trigger.test", "id"),
				),
			},
			// Verify import with Framework provider
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_pipeline_trigger.test",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabPipelineTriggerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_pipeline_trigger" {
			continue
		}

		project, pipelineTriggerId, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("failed to parse pipeline trigger ID: %w", err)
		}

		pipelineTriggerIdInt, err := strconv.Atoi(pipelineTriggerId)
		if err != nil {
			return fmt.Errorf("failed to convert pipeline trigger ID to int: %w", err)
		}

		_, _, err = testutil.TestGitlabClient.PipelineTriggers.GetPipelineTrigger(project, int64(pipelineTriggerIdInt))
		if err == nil {
			return fmt.Errorf("pipeline trigger %d for project %s still exists", pipelineTriggerIdInt, project)
		}

		if !api.Is404(err) {
			return fmt.Errorf("unexpected error checking for pipeline trigger deletion: %w", err)
		}

		return nil
	}

	return nil
}
