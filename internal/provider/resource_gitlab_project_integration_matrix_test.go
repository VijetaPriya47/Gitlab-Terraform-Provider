//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectIntegrationMatrix_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationMatrixCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Matrix integration with only required fields
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_matrix" "this" {
						project  = "%s"
						hostname = "https://matrix.org"
						token    = "tokenvalue"
						room     = "!abcdefg:matrix.org"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_matrix.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "hostname", "https://matrix.org"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "token", "tokenvalue"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "room", "!abcdefg:matrix.org"),
				),
			},
			// Verify upstream attributes
			{
				ResourceName:            "gitlab_project_integration_matrix.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update Matrix integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_matrix" "this" {
						project  = "%s"
						hostname = "https://matrix.org"
						token    = "tokenvalue"
						room     = "!abcdefg:matrix.org"

						notify_only_broken_pipelines = true
						branches_to_be_notified      = "default"
						push_events                  = false
						issues_events				 = false
						confidential_issues_events	 = false
						merge_requests_events		 = false
						tag_push_events			     = false
						note_events                  = false
						confidential_note_events     = false
						pipeline_events			     = false
						wiki_page_events			 = false
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_matrix.this", "id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "hostname", "https://matrix.org"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "token", "tokenvalue"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "room", "!abcdefg:matrix.org"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "notify_only_broken_pipelines", "true"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "branches_to_be_notified", "default"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "confidential_issues_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "merge_requests_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "tag_push_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "confidential_note_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "pipeline_events", "false"),
					resource.TestCheckResourceAttr("gitlab_project_integration_matrix.this", "wiki_page_events", "false"),
				),
			},
			// Verify updated upstream attributes
			{
				ResourceName:            "gitlab_project_integration_matrix.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationMatrix_invalidValues(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationMatrixCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Invalid branches_to_be_notified value
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_matrix" "this" {
						project                 = "%s"
						hostname                = "https://matrix.org"
						token                   = "tokenvalue"
						room                    = "!abcdefg:matrix.org"
						branches_to_be_notified = "invalid"
					}
				`, testProject.PathWithNamespace),
				ExpectError: regexp.MustCompile(`Attribute branches_to_be_notified value must be one of`),
			},
		},
	})
}

func testAccGitlabProjectIntegrationMatrixCheckDestroy(projectID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetMatrixService(strconv.FormatInt(projectID, 10))
		if err != nil {
			return fmt.Errorf("Error calling API to get Matrix integration service: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Matrix integration service still exists")
		}
		return nil
	}
}
