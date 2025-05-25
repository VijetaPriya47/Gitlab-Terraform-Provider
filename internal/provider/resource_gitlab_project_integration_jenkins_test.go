//go:build acceptance
// +build acceptance

package provider

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectIntegrationJenkins_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJenkinsDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jenkins" "this" {
						project      = "%d"
						jenkins_url  = "http://jenkins.example.com"
						project_name = "my_project_name"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_project_integration_jenkins.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_jenkins" "this" {
						project      = "%d"
						jenkins_url  = "http://jenkins.example.com"
						project_name = "my_project_name_new"	
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_project_integration_jenkins.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationJenkins_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationJenkinsDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jenkins" "this" {
						project      = "%d"
						jenkins_url  = "http://jenkins.example.com"
						project_name = "my_project_name"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_integration_jenkins.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_jenkins" "this" {
						project      = "%d"
						jenkins_url  = "http://jenkins.example.com"
						project_name = "my_project_name_new"	
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_integration_jenkins.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationJenkinsDestroy(projectId int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetJenkinsCIService(projectId)
		if err != nil {
			return fmt.Errorf("Error calling API to get the Jenkins integration: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Jenkins integration still exists")
		}
		return nil
	}
}
