//go:build acceptance
// +build acceptance

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

func TestAccGitlabIntegrationRedmine_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabIntegrationRedmineDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_integration_redmine.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      = "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci-new"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_integration_redmine.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabIntegrationRedmine_validation(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "oh no! not an url"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),

				ExpectError: regexp.MustCompile(`Attribute new_issue_url value should be an URL`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "oh no! not an url"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),

				ExpectError: regexp.MustCompile(`Attribute project_url value should be an URL`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci"
						issues_url  	= "oh no! not an url"
					}
				`, testProject.ID),

				ExpectError: regexp.MustCompile(`Attribute issues_url value should be an URL`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci"
						issues_url  	= "https://redmine.example.com/issues/not-an-id"
					}
				`, testProject.ID),

				ExpectError: regexp.MustCompile(`Attribute issues_url must contain ':id'.`),
			},
		},
	})
}

func testAccCheckGitlabIntegrationRedmineDestroy(projectId int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetRedmineService(projectId)
		if err != nil {
			return fmt.Errorf("Error calling API to get the Redmine integration: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Redmine integration still exists")
		}
		return nil
	}
}
