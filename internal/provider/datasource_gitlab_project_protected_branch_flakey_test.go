//go:build flakey
// +build flakey

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedBranch_search(t *testing.T) {
	// Create a project using default branch protection, which
	// will protect "main" by default.
	project := testutil.CreateProject(t)
	branch := testutil.CreateProtectedBranches(t, project, 1)[0]

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_protected_branch" "test" {
				  project_id = %d
				  name       = "%s"
				}
				`, project.ID, branch.Name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_branch.test",
						"name",
						branch.Name,
					),
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_branch.test",
						"push_access_levels.0.access_level",
						"maintainer",
					),
				),
			},
		},
	})
}
