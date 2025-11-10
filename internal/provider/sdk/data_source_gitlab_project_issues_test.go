//go:build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectIssues_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testIssues := testutil.CreateProjectIssues(t, testProject.ID, 25)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_issues" "this" {
						project = %d
					
						// only for determinism
						order_by = "relative_position"
						sort     = "asc"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.#", fmt.Sprintf("%d", len(testIssues))),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.0.iid", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.0.title", "Issue 0"),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.0.description", "Description 0"),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.1.iid", "2"),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.1.title", "Issue 1"),
					resource.TestCheckResourceAttr("data.gitlab_project_issues.this", "issues.1.description", "Description 1"),
				),
			},
		},
	})
}
