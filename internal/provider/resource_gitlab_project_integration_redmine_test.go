//go:build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil/framework"
)

func TestAccGitlabProjectIntegrationRedmine_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationRedmineDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_redmine" "this" {
						project      	= "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_project_integration_redmine.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_redmine" "this" {
						project      = "%d"
						new_issue_url  	= "https://redmine.example.com/projects/gitlab-ci/issues/new"
						project_url 	= "https://redmine.example.com/projects/gitlab-ci-new"
						issues_url  	= "https://redmine.example.com/issues/:id"
					}
				`, testProject.ID),
			},
			{
				ResourceName:      "gitlab_project_integration_redmine.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectIntegrationRedmine_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationRedmineDestroy(testProject.ID),
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

func TestAccGitlabProjectIntegrationRedmine_validation(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_redmine" "this" {
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
					resource "gitlab_project_integration_redmine" "this" {
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
					resource "gitlab_project_integration_redmine" "this" {
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
					resource "gitlab_project_integration_redmine" "this" {
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

// TestAcc_GitlabProjectIntegrationRedmine_stateMove verifies that the moved block works
// when migrating from gitlab_integration_redmine to gitlab_project_integration_redmine.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationRedmine_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationRedmineDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Redmine integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_redmine" "old" {
					project       = "%d"
					new_issue_url = "https://redmine.example.com/projects/gitlab-ci/issues/new"
					project_url   = "https://redmine.example.com/projects/gitlab-ci"
					issues_url    = "https://redmine.example.com/issues/:id"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_redmine.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_redmine" "new" {
					project       = "%d"
					new_issue_url = "https://redmine.example.com/projects/gitlab-ci/issues/new"
					project_url   = "https://redmine.example.com/projects/gitlab-ci"
					issues_url    = "https://redmine.example.com/issues/:id"
				}

				moved {
					from = gitlab_integration_redmine.old
					to   = gitlab_project_integration_redmine.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_redmine.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_redmine.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationRedmineDestroy(projectId int64) resource.TestCheckFunc {
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
