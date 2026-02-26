//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectBranches_search(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testBranches := testutil.CreateBranches(t, testProject, 25)
	expectedBranches := len(testBranches) + 1 // main branch already exists

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%d"
					}
				`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", fmt.Sprintf("%d", expectedBranches)),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.merged"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.protected"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.default"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.developers_can_push"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.developers_can_merge"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.can_push"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_branches.this", "branches.0.web_url"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.commit.#", "1"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectBranches_filters(t *testing.T) {
	testProject := testutil.CreateProject(t)
	branchPrefix := acctest.RandomWithPrefix("acctest-branch")
	searchToken := acctest.RandomWithPrefix("search")
	regexToken := acctest.RandomWithPrefix("regex")
	searchBranchName := fmt.Sprintf("%s-%s", branchPrefix, searchToken)
	regexBranchName := fmt.Sprintf("%s-%s", branchPrefix, regexToken)

	testutil.CreateBranch(t, testProject, searchBranchName)
	testutil.CreateBranch(t, testProject, regexBranchName)

	regexPattern := fmt.Sprintf("^%s$", regexBranchName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%[1]d"
						search  = "%[2]s"
					}
				`, testProject.ID, searchToken),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.name", searchBranchName),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_branches" "this" {
						project = "%[1]d"
						regex   = "%[2]s"
					}
				`, testProject.ID, regexPattern),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_branches.this", "branches.0.name", regexBranchName),
				),
			},
		},
	})
}
