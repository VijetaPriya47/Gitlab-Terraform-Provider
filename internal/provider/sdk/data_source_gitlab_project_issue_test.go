//go:build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectIssue_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	issue := testutil.CreateProjectIssues(t, testProject.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_issue" "this" {
						project = %d
						iid     = %d
					}
				`, testProject.ID, issue.IID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_issue.this", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_issue.this", "title", issue.Title),
					resource.TestCheckResourceAttr("data.gitlab_project_issue.this", "issue_id", fmt.Sprintf("%d", issue.ID)),
				),
			},
		},
	})
}
