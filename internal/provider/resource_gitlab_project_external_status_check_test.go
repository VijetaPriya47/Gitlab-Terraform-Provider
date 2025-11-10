//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectExternalStatusCheck_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)
	protectedBranches := testutil.CreateProtectedBranches(t, project, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectExternalStatusCheckDestroy,
		Steps: []resource.TestStep{
			// Create a project and external status check with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_external_status_check" "foo" {
						project_id   = %d
						name         = "foo"
						external_url = "https://api.gitlab.com"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExternalStatusCheckExists("gitlab_project_external_status_check.foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "name", "foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "external_url", "https://api.gitlab.com"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "hmac", "false"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "protected_branch_ids.#", "0"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project_external_status_check.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"shared_secret"},
			},
			// Update the external status check
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_external_status_check" "foo" {
						project_id    = %d
						name          = "bar"
						external_url  = "https://example.gitlab.com"
						shared_secret = "secret"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExternalStatusCheckExists("gitlab_project_external_status_check.foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "name", "bar"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "external_url", "https://example.gitlab.com"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "hmac", "true"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "protected_branch_ids.#", "0"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project_external_status_check.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"shared_secret"},
			},
			// Update the external status check, adding protected branches
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_external_status_check" "foo" {
						project_id           = %d
						name                 = "bar"
						external_url         = "https://example.gitlab.com"
						shared_secret        = "secret"
						protected_branch_ids = [
						    %d,
						    %d
						]
					}
				`, project.ID, protectedBranches[0].ID, protectedBranches[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExternalStatusCheckExists("gitlab_project_external_status_check.foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "name", "bar"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "external_url", "https://example.gitlab.com"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "hmac", "true"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "protected_branch_ids.#", "2"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project_external_status_check.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"shared_secret"},
			},
			// Update the external status check to get back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_external_status_check" "foo" {
						project_id   = %d
						name         = "foo"
						external_url = "https://api.gitlab.com"
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExternalStatusCheckExists("gitlab_project_external_status_check.foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "name", "foo"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "external_url", "https://api.gitlab.com"),
					resource.TestCheckResourceAttr("gitlab_project_external_status_check.foo", "hmac", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project_external_status_check.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"shared_secret"},
			},
		},
	})
}

func testAccCheckGitlabProjectExternalStatusCheckExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		projectID, externalCheckID, err := parseProjectExternalStatusCheckID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = findProjectExternalStatusCheck(testutil.TestGitlabClient, projectID, externalCheckID)
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckGitlabProjectExternalStatusCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_external_status_check" {
			continue
		}

		projectID, externalCheckID, err := parseProjectExternalStatusCheckID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Unable to parse resource ID: %s, %s", rs.Primary.ID, err.Error())
		}

		_, err = findProjectExternalStatusCheck(testutil.TestGitlabClient, projectID, externalCheckID)
		if err == nil {
			return fmt.Errorf("Project external status check %d in project %d still exists", externalCheckID, projectID)
		} else if api.Is404(err) {
			// external status check was not found, so it was successfully removed
			return nil
		} else {
			return err
		}
	}
	return nil
}
