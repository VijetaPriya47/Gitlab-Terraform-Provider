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

// TestAcc_GitlabProjectIntegrationJenkins_stateMove verifies that the moved block works
// when migrating from gitlab_integration_jenkins to gitlab_project_integration_jenkins.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationJenkins_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationJenkinsDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Jenkins integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_jenkins" "old" {
					project      = "%d"
					jenkins_url  = "http://jenkins.example.com"
					project_name = "my_project_name"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_jenkins.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_jenkins" "new" {
					project      = "%d"
					jenkins_url  = "http://jenkins.example.com"
					project_name = "my_project_name"
				}

				moved {
					from = gitlab_integration_jenkins.old
					to   = gitlab_project_integration_jenkins.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_jenkins.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_jenkins.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationJenkinsDestroy(projectId int64) resource.TestCheckFunc {
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
