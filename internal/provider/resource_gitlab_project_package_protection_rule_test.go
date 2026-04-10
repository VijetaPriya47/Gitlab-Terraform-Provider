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

func TestAcc_GitlabProjectPackageProtectionRuleBasic(t *testing.T) {

	// Set up project environment.
	project := testutil.CreateProject(t)
	packageNamePattern := "@scope/package-*"
	packageNamePattern2 := "@scope/other-package-*"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectPackageProtectionRuleCheckDestroy,
		Steps: []resource.TestStep{
			// Create basic package protection rules.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_package_protection_rule" "this" {
					project                      = %d
					package_name_pattern         = "%s"
					package_type                 = "npm"

					minimum_access_level_for_push = "maintainer"
				}`, project.ID, packageNamePattern),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "package_name_pattern", packageNamePattern),
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "package_type", "npm"),
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "minimum_access_level_for_push", "maintainer"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_package_protection_rule.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update minimum access level and package name pattern
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_package_protection_rule" "this" {
					project                      = %d
					package_name_pattern         = "%s"
					package_type                 = "npm"

					minimum_access_level_for_push   = "owner"
				}`, project.ID, packageNamePattern2),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "package_name_pattern", packageNamePattern2),
					resource.TestCheckResourceAttr("gitlab_project_package_protection_rule.this", "minimum_access_level_for_push", "owner"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_package_protection_rule.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGitlabProjectPackageProtectionRuleCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_package_protection_rule" {
			continue
		}

		project := rs.Primary.Attributes["project"]
		packageProtectionRuleID := rs.Primary.Attributes["package_protection_rule_id"]

		rules, _, err := testutil.TestGitlabClient.ProtectedPackages.ListPackageProtectionRules(project, nil)

		if err != nil {
			return err
		}

		if len(rules) != 0 {
			return fmt.Errorf("Package protection rule with ID %s in project %s still exists", packageProtectionRuleID, project)
		}

		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
