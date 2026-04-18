//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectProtectedBranches_searchCE(t *testing.T) {
	testutil.SkipIfEE(t)

	project := testutil.CreateProjectWithOptions(t, &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
		Description: gitlab.Ptr("Terraform acceptance tests"),
		// So that acceptance tests can be run in a gitlab organization with no billing.
		Visibility:           gitlab.Ptr(gitlab.PublicVisibility),
		InitializeWithReadme: gitlab.Ptr(false),
	})
	testutil.CreateProtectedBranchWithOptions(t, project, &gitlab.ProtectRepositoryBranchesOptions{
		PushAccessLevel:  gitlab.Ptr(gitlab.MaintainerPermissions),
		MergeAccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
	})

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_protected_branches" "test" {
						project_id = %d
					}
				`, project.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// number of protected branches is 'main' and the one created above
					resource.TestCheckResourceAttr("data.gitlab_project_protected_branches.test", "protected_branches.#", "2"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.push_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.merge_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.push_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.merge_access_levels.0.access_level"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectProtectedBranches_searchEE(t *testing.T) {
	testutil.SkipIfCE(t)

	project := testutil.CreateProjectWithOptions(t, &gitlab.CreateProjectOptions{
		Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
		Description: gitlab.Ptr("Terraform acceptance tests"),
		// So that acceptance tests can be run in a gitlab organization with no billing.
		Visibility:           gitlab.Ptr(gitlab.PublicVisibility),
		InitializeWithReadme: gitlab.Ptr(false),
	})
	testutil.CreateProtectedBranchWithOptions(t, project, &gitlab.ProtectRepositoryBranchesOptions{
		AllowedToPush: &[]*gitlab.BranchPermissionOptions{
			{
				AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
			},
		},
		AllowedToMerge: &[]*gitlab.BranchPermissionOptions{
			{
				AccessLevel: gitlab.Ptr(gitlab.DeveloperPermissions),
			},
		},
	})

	// lintignore:AT001 // Data sources don't need check destroy in their tests
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_protected_branches" "test" {
						project_id = %d
					}
				`, project.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					// number of protected branches is 'main' and the one created above
					resource.TestCheckResourceAttr("data.gitlab_project_protected_branches.test", "protected_branches.#", "2"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.push_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.0.merge_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.push_access_levels.0.access_level"),
					resource.TestCheckResourceAttrSet("data.gitlab_project_protected_branches.test", "protected_branches.1.merge_access_levels.0.access_level"),
				),
			},
		},
	})
}
