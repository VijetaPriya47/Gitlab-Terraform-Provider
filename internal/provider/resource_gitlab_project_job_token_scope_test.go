//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabProjectJobTokenScope_basic(t *testing.T) {
	testutil.RunIfAtLeast(t, "16.1")

	// Set up project environment.
	project := testutil.CreateProject(t)
	projectIDStr := strconv.Itoa(project.ID)
	targetProject1 := testutil.CreateProject(t)
	targetProject1IDStr := strconv.Itoa(targetProject1.ID)
	targetProject2 := testutil.CreateProject(t)
	targetProject2IDStr := strconv.Itoa(targetProject2.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScope_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject1.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildTwoPartID(&projectIDStr, &targetProject1IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_project_id", targetProject1IDStr),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scope" "this" {
					project     = %d
					target_project_id = %d
				}`, project.ID, targetProject2.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "id", utils.BuildTwoPartID(&projectIDStr, &targetProject2IDStr)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "project", projectIDStr),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scope.this", "target_project_id", targetProject2IDStr),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scope.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabProjectJobTokenScope_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_job_token_scope" {
			projectID := rs.Primary.Attributes["project"]
			targetProjectID, err := strconv.Atoi(rs.Primary.Attributes["target_project_id"])
			if err != nil {
				return fmt.Errorf("Failed to parse target project ID: %w", err)
			}

			projects, _, err := testutil.TestGitlabClient.JobTokenScope.GetProjectJobTokenInboundAllowList(projectID, nil, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch CI/CD Job Token Scope: %w", err)
			}

			for i := range projects {
				if projects[i].ID == targetProjectID {
					return fmt.Errorf("The target project ID (%d) still exists in the source project (%s)", targetProjectID, projectID)
				}
			}

			return nil
		}
	}
	return nil
}
