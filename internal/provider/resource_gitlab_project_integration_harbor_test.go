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

func TestAccGitlabProjectIntegrationHarbor_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationHarborDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username"
						password     = "my_password"
						project_name = "my_project_name"
					}
				`, testProject.ID),
			},
			{
				ResourceName:            "gitlab_project_integration_harbor.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username_new"
						password     = "my_password_new"
						project_name = "my_project_name_new"	
					}
				`, testProject.ID),
			},
			{
				ResourceName:            "gitlab_project_integration_harbor.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationHarbor_basic_deprecated(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationHarborDestroy(testProject.ID),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username"
						password     = "my_password"
						project_name = "my_project_name"
					}
				`, testProject.ID),
			},
			{
				ResourceName:            "gitlab_integration_harbor.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username_new"
						password     = "my_password_new"
						project_name = "my_project_name_new"	
					}
				`, testProject.ID),
			},
			{
				ResourceName:            "gitlab_integration_harbor.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationHarbor_validation(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "invalid-url"
						username     = "my_username"
						password     = "my_password"
						project_name = "my_project_name"
					}
				`, testProject.ID),

				ExpectError: regexp.MustCompile(`Attribute url value should be an URL with http or https schema, got:`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username"
						password     = "my_password"
						project_name = ""
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute project_name string length must be at least 1, got:`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = ""
						password     = "my_password"
						project_name = "my_project_name"
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute username string length must be at least 1, got:`),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_harbor" "this" {
						project      = "%d"
						url          = "http://harbor.example.com"
						username     = "my_username"
						password     = ""
						project_name = "my_project_name"
					}
				`, testProject.ID),
				ExpectError: regexp.MustCompile(`Attribute password string length must be at least 1, got:`),
			},
		},
	})
}

// TestAcc_GitlabProjectIntegrationHarbor_stateMove verifies that the moved block works
// when migrating from gitlab_integration_harbor to gitlab_project_integration_harbor.
// This test requires Terraform 1.8+ because cross-resource-type state moves
// were introduced in that version.
func TestAccGitlabProjectIntegrationHarbor_stateMove(t *testing.T) {
	testProject := testutil.CreateProject(t)

	// Run this test explicitly with the 1.8 version of TF; this helper will run the
	// test independently (not in parallel), and reset the TF version when the
	// test finishes.
	framework.RunTestWithVersion(t, "1.8.0", resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.RequireAbove(tfversion.Version1_8_0), // fail if the TF version isn't set properly.
		},
		CheckDestroy: testAccCheckGitlabProjectIntegrationHarborDestroy(testProject.ID),
		Steps: []resource.TestStep{
			// Create a Harbor integration using the old resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_integration_harbor" "old" {
					project      = "%d"
					url          = "http://harbor.example.com"
					username     = "my_username"
					password     = "my_password"
					project_name = "my_project_name"
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_integration_harbor.old", "id"),
				),
			},
			// Move the state to the new resource
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_integration_harbor" "new" {
					project      = "%d"
					url          = "http://harbor.example.com"
					username     = "my_username"
					password     = "my_password"
					project_name = "my_project_name"
				}

				moved {
					from = gitlab_integration_harbor.old
					to   = gitlab_project_integration_harbor.new
				}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_integration_harbor.new", "id"),
				),
			},
			// Verify the resource still works after the move
			{
				ResourceName:            "gitlab_project_integration_harbor.new",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationHarborDestroy(projectId int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		service, _, err := testutil.TestGitlabClient.Services.GetHarborService(projectId)
		if err != nil {
			return fmt.Errorf("Error calling API to get the Harbor integration: %w", err)
		}
		if service != nil && service.Active != false {
			return errors.New("Harbor integration still exists")
		}
		return nil
	}
}
