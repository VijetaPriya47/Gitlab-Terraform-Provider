//go:build flakey

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedBranch_search(t *testing.T) {
	// Create a project using default branch protection, which
	// will protect "main" by default.
	project := testutil.CreateProjectWithOptions(t, &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
		Description: gitlab.Ptr("Terraform acceptance tests"),
		// So that acceptance tests can be run in a gitlab organization with no billing.
		Visibility:           gitlab.Ptr(gitlab.PublicVisibility),
		InitializeWithReadme: gitlab.Ptr(false),
	})
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
					resource.TestCheckResourceAttr(
						"data.gitlab_project_protected_branch.test",
						"merge_access_levels.0.access_level",
						"maintainer",
					),
				),
			},
		},
	})
}
