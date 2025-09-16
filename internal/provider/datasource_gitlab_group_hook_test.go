//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupHook_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testHook := testutil.CreateGroupHooks(t, testGroup.ID, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_hook" "this" {
						group   = "%s"
						hook_id = %d
					}
				`, testGroup.FullPath, testHook.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_hook.this", "hook_id", fmt.Sprintf("%d", testHook.ID)),
					resource.TestCheckResourceAttr("data.gitlab_group_hook.this", "group_id", fmt.Sprintf("%d", testGroup.ID)),
					resource.TestCheckResourceAttr("data.gitlab_group_hook.this", "url", testHook.URL),
					resource.TestCheckResourceAttr("data.gitlab_group_hook.this", "emoji_events", "true"),
				),
			},
		},
	})
}
