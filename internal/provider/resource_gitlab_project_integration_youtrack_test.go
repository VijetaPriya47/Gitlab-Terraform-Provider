//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectIntegrationYouTrack_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationYouTrackDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_youtrack" "this" {
						project     = "%d"
						issues_url  = "https://youtrack.example.com/issue/:id"
						project_url = "https://youtrack.example.com/1"
					}
				`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "id", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "issues_url", "https://youtrack.example.com/issue/:id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "project_url", "https://youtrack.example.com/1"),
				),
			},
			{
				ResourceName:      "gitlab_project_integration_youtrack.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_youtrack" "this" {
						project     = "%d"
						issues_url  = "https://youtrack.example.com/issue/:id"
						project_url = "https://youtrack.example.com/2"
					}
				`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "id", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "issues_url", "https://youtrack.example.com/issue/:id"),
					resource.TestCheckResourceAttr("gitlab_project_integration_youtrack.this", "project_url", "https://youtrack.example.com/2"),
				),
			},
			{
				ResourceName:      "gitlab_project_integration_youtrack.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationYouTrack_basic_validation(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_youtrack" "this" {
						project     = "%d"
						issues_url  = "oh no! not an url"
						project_url = "https://youtrack.example.com/1"
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute issues_url value should be an URL`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_youtrack" "this" {
						project     = "%d"
						issues_url  = "https://youtrack.example.com/issue/:id"
						project_url = "oh no! not an url"
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute project_url value should be an URL`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_youtrack" "this" {
						project     = "%d"
						issues_url  = "https://youtrack.example.com/issue/not-an-id"
						project_url = "https://youtrack.example.com/1"
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute issues_url must contain ':id'.`),
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationYouTrackDestroy(projectID int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetYouTrackService(projectID)
		if err != nil {
			return fmt.Errorf("Error calling API to get the YouTrack integration: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("YouTrack integration still exists")
		}
		return nil
	}
}
