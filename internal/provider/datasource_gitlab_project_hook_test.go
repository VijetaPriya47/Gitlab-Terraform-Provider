//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectHook_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testHook := testutil.CreateProjectHooks(t, testProject.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_hook" "this" {
						project = "%s"
						hook_id = %d
					}
				`, testProject.PathWithNamespace, testHook.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_hook.this", "hook_id", fmt.Sprintf("%d", testHook.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_hook.this", "project_id", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("data.gitlab_project_hook.this", "url", testHook.URL),
				),
			},
		},
	})
}
