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

func TestAcc_GitlabProjectContainerRepositoryProtectionRulesBasic(t *testing.T) {

	// Set up project environment.
	project := testutil.CreateProject(t)
	repositoryPathPattern := fmt.Sprintf("%s/example*", project.PathWithNamespace)
	repositoryPathPattern2 := fmt.Sprintf("%s/example2*", project.PathWithNamespace)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectContainerRepositoryProtectionRulesCheckDestroy,
		Steps: []resource.TestStep{
			// Create basic container repository protection rules.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_container_repository_protection" "this" {
					project                      = %d
					repository_path_pattern      = "%s"

					minimum_access_level_for_push     = "maintainer"
				}`, project.ID, repositoryPathPattern),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "repository_path_pattern", repositoryPathPattern),
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "minimum_access_level_for_push", "maintainer"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_container_repository_protection.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update minimum access level
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_container_repository_protection" "this" {
					project                      = %d
					repository_path_pattern      = "%s"

					minimum_access_level_for_push     = "owner"
					minimum_access_level_for_delete   = "admin"
				}`, project.ID, repositoryPathPattern2),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "repository_path_pattern", repositoryPathPattern2),
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "minimum_access_level_for_delete", "admin"),
					resource.TestCheckResourceAttr("gitlab_project_container_repository_protection.this", "minimum_access_level_for_push", "owner"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_container_repository_protection.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGitlabProjectContainerRepositoryProtectionRulesCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_container_repository_protection" {
			continue
		}

		project := rs.Primary.Attributes["project"]
		protectionRuleID := rs.Primary.Attributes["protection_rule_id"]

		rules, _, err := testutil.TestGitlabClient.ContainerRegistryProtectionRules.ListContainerRegistryProtectionRules(project)

		if err != nil {
			return err
		}

		if len(rules) != 0 {
			return fmt.Errorf("Container protection rule with ID %s in project %s still exists", protectionRuleID, project)
		}

		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
