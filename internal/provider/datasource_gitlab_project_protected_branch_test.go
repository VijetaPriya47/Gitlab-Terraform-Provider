//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedBranch_search(t *testing.T) {
	// Create a project using default branch protection, which
	// will protect "main" by default.
	project := testutil.CreateProject(t)

	//lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_project_protected_branch" "test" {
				  project_id = %d
				  name       = "main"
				}
				`, project.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_branch.test",
						"name",
						"main",
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
