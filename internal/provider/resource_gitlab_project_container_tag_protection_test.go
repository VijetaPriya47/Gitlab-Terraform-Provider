//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectContainerTagProtectionRulesImmutable(t *testing.T) {
	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectContainerTagProtectionRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_container_tag_protection" "this" {
					project        = %d
					tag_name_regex = ".*"
					immutable      = true
				}`, project.ID),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "tag_name_regex", ".*"),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "immutable", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_container_tag_protection.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectContainerTagProtectionRulesFieldsConflicts(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectContainerTagProtectionRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_project_container_tag_protection" "this" {
						project        = 123
						tag_name_regex = ".*"
						immutable      = true
						minimum_access_level_for_push   = "MAINTAINER"
						minimum_access_level_for_delete = "OWNER"
					}
				`,

				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectContainerTagProtectionRulesProtected(t *testing.T) {
	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectContainerTagProtectionRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_container_tag_protection" "this" {
					project        		            = %d
					tag_name_regex 		            = "v.*"
					minimum_access_level_for_push   = "MAINTAINER"
					minimum_access_level_for_delete = "OWNER"
				}`, project.ID),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "tag_name_regex", "v.*"),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "minimum_access_level_for_push", "MAINTAINER"),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "minimum_access_level_for_delete", "OWNER"),
					resource.TestCheckResourceAttr("gitlab_project_container_tag_protection.this", "immutable", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_container_tag_protection.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGitlabProjectContainerTagProtectionRulesCheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_container_tag_protection" {
			continue
		}

		project := rs.Primary.Attributes["project"]
		protectionRuleID := rs.Primary.Attributes["protection_rule_id"]
		query := gitlab.GraphQLQuery{
			Query: fmt.Sprintf(`
				query {
					project(id: "gid://gitlab/Project/%s") {
						containerRegistryProtectionRules {
							nodes {
								id
								tagPathPattern
								minimumAccessLevelForPush
								minimumAccessLevelForDelete
							}
						}
					}
				}`,
				project,
			),
		}
		var response struct {
			Data struct {
				Project struct {
					ContainerRegistryProtectionRules struct {
						Nodes []ContainerTagProtectionRule `json:"nodes"`
					} `json:"containerRegistryProtectionRules"`
				} `json:"project"`
			} `json:"data"`
		}
		_, err := testutil.TestGitlabClient.GraphQL.Do(query, &response)
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return err
		}
		if response.Data.Project.ContainerRegistryProtectionRules.Nodes == nil {
			return nil
		}

		rules := response.Data.Project.ContainerRegistryProtectionRules.Nodes
		for _, rule := range rules {
			if rule.ID == protectionRuleID {
				return fmt.Errorf("Container tag protection rule with ID %s in project %s still exists", protectionRuleID, project)
			}
		}
		return nil
	}

	return nil
}
