//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabIntegrationGithub_backwardsCompatibleToService(t *testing.T) {
	testutil.SkipIfCE(t)

	var githubService gitlab.GithubService
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationGithubDestroy,
		Steps: []resource.TestStep{
			// Create a project and a github service
			{
				Config: fmt.Sprintf(`
					resource "gitlab_service_github" "github" {
						project        = "%d"
						token          = "test"
						repository_url = "https://github.com/gitlabhq/terraform-provider-gitlab"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabIntegrationGithubExists("gitlab_service_github.github", &githubService),
					resource.TestCheckResourceAttr("gitlab_service_github.github", "repository_url", "https://github.com/gitlabhq/terraform-provider-gitlab"),
					resource.TestCheckResourceAttr("gitlab_service_github.github", "static_context", "true"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_service_github.github",
				ImportStateIdFunc: getGithubProjectID("gitlab_service_github.github"),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"token",
				},
			},
		},
	})
}

func TestAccGitlabIntegrationGithub_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	var githubService gitlab.GithubService
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabIntegrationGithubDestroy,
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
					testAccCheckGitlabIntegrationGithubExists("gitlab_integration_github.github", &githubService),
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
					testAccCheckGitlabIntegrationGithubExists("gitlab_integration_github.github", &githubService),
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
					testAccCheckGitlabIntegrationGithubExists("gitlab_integration_github.github", &githubService),
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

func testAccCheckGitlabIntegrationGithubExists(n string, service *gitlab.GithubService) resource.TestCheckFunc {
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

func testAccCheckGitlabIntegrationGithubDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, _, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
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
