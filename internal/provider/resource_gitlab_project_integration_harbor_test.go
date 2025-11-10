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

func testAccCheckGitlabProjectIntegrationHarborDestroy(projectId int) resource.TestCheckFunc {
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
