//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectTargetBranchRule_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectTargetBranchRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_target_branch_rule" "rule" {
						project               = "%[1]s"
						source_branch_pattern = "develop"
						target_branch_name    = "release"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_target_branch_rule.rule", "id"),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "source_branch_pattern", "develop"),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "target_branch_name", "release"),
				),
			},
			{
				ResourceName:      "gitlab_project_target_branch_rule.rule",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify update / re-creation
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_target_branch_rule" "rule" {
						project               = "%[1]s"
						source_branch_pattern = "develop"
						target_branch_name    = "master"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_target_branch_rule.rule", "id"),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "source_branch_pattern", "develop"),
					resource.TestCheckResourceAttr("gitlab_project_target_branch_rule.rule", "target_branch_name", "master"),
				),
			},
		},
	})
}

func TestAccGitlabProjectTargetBranchRule_DuplicateRule(t *testing.T) {
	testutil.SkipIfCE(t)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_target_branch_rule" "rule" {
						project               = "%[1]s"
						source_branch_pattern = "develop"
						target_branch_name    = "release"
					}

					resource "gitlab_project_target_branch_rule" "duplicate_source" {
						depends_on            = [gitlab_project_target_branch_rule.rule] # To avoid API race conditions (https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/merge_requests/2307#note_2372121119)
						project               = "%[1]s"
						source_branch_pattern = "develop"
						target_branch_name    = "master"
					}
				`, testProject.PathWithNamespace),
				ExpectError: regexp.MustCompile("Unable to create gitlab_project_target_branch_rule: Name has already been\ntaken"),
			},
		},
	})
}

func testAccGitlabProjectTargetBranchRuleDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_target_branch_rule" {
			continue
		}
		ctx := context.Background()

		project, _, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.Attributes["project"], nil, gitlab.WithContext(ctx))
		if err != nil {
			return err
		}

		query := gitlab.GraphQLQuery{
			Query: fmt.Sprintf(`
			query {
				project(fullPath:"%s") {
					targetBranchRules {
						nodes {
							id
							name
							targetBranch
						}
					}
				}
			}`, project.PathWithNamespace),
		}

		var response getProjectTargetBranchRuleReadResponse
		_, err = testutil.TestGitlabClient.GraphQL.Do(query, &response)
		if err != nil {
			return err
		}

		if len(response.Data.Project.TargetBranchRules.Nodes) > 0 {
			return fmt.Errorf("Target branch rule still exists, should have been destroyed by the tests")
		}
	}
	return nil
}
