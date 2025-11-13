//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupHooks_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	groups := testutil.CreateGroups(t, 2)
	testGroup := groups[0]
	testGroup2 := groups[1]
	testHooks := testutil.CreateGroupHooks(t, testGroup.ID, 25)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_hooks" "this" {
						group = "%s"
					}
				`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_hooks.this", "hooks.#", fmt.Sprintf("%d", len(testHooks))),
					resource.TestCheckResourceAttr("data.gitlab_group_hooks.this", "hooks.0.url", testHooks[0].URL),
					resource.TestCheckResourceAttr("data.gitlab_group_hooks.this", "hooks.1.url", testHooks[1].URL),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_hooks" "this" {
						group = "%s"
					}
				`, testGroup2.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_hooks.this", "hooks.#", "0"),
				),
			},
		},
	})
}
