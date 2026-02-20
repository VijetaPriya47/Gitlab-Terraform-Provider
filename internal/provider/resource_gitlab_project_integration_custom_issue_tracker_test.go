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

func TestAcc_GitlabProjectIntegrationCustomIssueTracker_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationCustomIssueTrackerCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Custom Issue Tracker integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = "%s"
					project_url = "https://customtracker.com"
					issues_url  = "https://customtracker.com/:id"
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "active"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "created_at"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_integration_custom_issue_tracker.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Custom Issue Tracker integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "https://anotherracker.com"
					issues_url  = "https://anotherracker.com/:id"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "active"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.this", "updated_at"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_integration_custom_issue_tracker.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationCustomIssueTracker_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationCustomIssueTrackerCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Custom Issue Tracker integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_custom_issue_tracker" "this" {
					project     = "%s"
					project_url = "https://customtracker.com"
					issues_url  = "https://customtracker.com/:id"
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "active"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "created_at"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_integration_custom_issue_tracker.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Custom Issue Tracker integration
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "https://anotherracker.com"
					issues_url  = "https://anotherracker.com/:id"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "project"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "active"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.this", "updated_at"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_integration_custom_issue_tracker.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectIntegrationCustomIssueTracker_failures(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectIntegrationCustomIssueTrackerCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Fail if project missing
			{
				Config: `
             		resource "gitlab_project_integration_custom_issue_tracker" "this" {
						project_url = "https://customtracker.org"
						issues_url  = "https://customtracker.org/:id"
					}`,
				ExpectError: regexp.MustCompile(`The argument "project" is required, but no definition was found`),
			},
			// Fail if project_url missing
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project    = %d
					issues_url = "https://customtracker.org/:id"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`The argument "project_url" is required, but no definition was found`),
			},
			// Fail if project_url is invalid
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "customtracker.org"
					issues_url  = "https://customtracker.org/:id"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute project_url value should be an URL with http or https schema`),
			},
			// Fail if issues_url missing
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "https://customtracker.org"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`The argument "issues_url" is required, but no definition was found`),
			},
			// Fail if issues_url is invalid
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "https://customtracker.org"
					issues_url  = "customtracker.org/:id"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute issues_url value should be an URL with http or https schema`),
			},
			// Fail if issues_url doesn't contain :id
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "this" {
					project     = %d
					project_url = "https://customtracker.org"
					issues_url  = "https://customtracker.org/no-id"
				}`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute issues_url value should contain :id placeholder`),
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationCustomIssueTracker_stateMove verifies that the moved block works
// when migrating from gitlab_integration_custom_issue_tracker to gitlab_project_integration_custom_issue_tracker.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAcc_GitlabProjectIntegrationCustomIssueTracker_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccGitlabProjectIntegrationCustomIssueTrackerCheckDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Custom Issue Tracker integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_custom_issue_tracker" "old" {
					project     = "%s"
					project_url = "https://customtracker.com"
					issues_url  = "https://customtracker.com/:id"
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "id"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "project"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "active"),
					resource.TestCheckResourceAttrSet("gitlab_integration_custom_issue_tracker.old", "created_at"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_custom_issue_tracker" "new" {
					project     = "%s"
					project_url = "https://customtracker.com"
					issues_url  = "https://customtracker.com/:id"
				}

				moved {
					from = gitlab_integration_custom_issue_tracker.old
					to   = gitlab_project_integration_custom_issue_tracker.new
				}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "id"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "project"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "project_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "issues_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "active"),
					resource.TestCheckResourceAttrSet("gitlab_project_integration_custom_issue_tracker.new", "created_at"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:      "gitlab_project_integration_custom_issue_tracker.new",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGitlabProjectIntegrationCustomIssueTrackerCheckDestroy(projectId int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetCustomIssueTrackerService(projectId)
		if err != nil {
			return fmt.Errorf("Error calling API to get the Custom Issue Tracker: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Custom issue tracker still exists")
		}
		return nil
	}
}
