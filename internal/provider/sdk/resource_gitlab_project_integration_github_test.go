//go:build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectIntegrationGithub_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	var githubService gitlab.GithubService
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Create a project and a github service
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_project_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "static_context", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_project_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the github integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/terraform-providers/terraform-provider-github"
						static_context = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_project_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "repository_url", "https://github.com/terraform-providers/terraform-provider-github"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "static_context", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_project_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the github integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_project_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_project_integration_github.github", "static_context", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_project_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabProjectIntegrationGithub_basic_deprecated(t *testing.T) {
	testutil.SkipIfCE(t)

	var githubService gitlab.GithubService
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Create a project and a github service
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "static_context", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the github integration
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/terraform-providers/terraform-provider-github"
						static_context = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "repository_url", "https://github.com/terraform-providers/terraform-provider-github"),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "static_context", "false"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
			// Update the github integration to get back to previous settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_integration_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectIntegrationGithubExists("gitlab_integration_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_integration_github.github", "static_context", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_integration_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_integration_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func testAccCheckGitlabProjectIntegrationGithubExists(n string, service *gitlab.GithubService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("No project ID is set")
		}
		githubService, _, err := testutil.TestGitlabClient.Services.GetGithubService(project)
		if err != nil {
			return fmt.Errorf("Github integration does not exist in project %s: %v", project, err)
		}
		*service = *githubService

		return nil
	}
}

func testAccCheckGitlabProjectIntegrationGithubDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_integration_github" && rs.Type != "gitlab_project_integration_github" {
			continue
		}

		gotInt, _, err := testutil.TestGitlabClient.Services.GetGithubService(rs.Primary.ID, nil)
		if err == nil {
			if gotInt != nil && fmt.Sprintf("%d", gotInt.ID) == rs.Primary.ID {
				return fmt.Errorf("Integration still exists")
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

func getGithubProjectID(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("Not Found: %s", n)
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return "", fmt.Errorf("No project ID is set")
		}

		return project, nil
	}
}
