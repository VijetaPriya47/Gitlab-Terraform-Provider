//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroup_basic(t *testing.T) {
	groups := testutil.CreateGroups(t, 3)
	subgroup := testutil.CreateSubGroups(t, groups[2], 1)[0]
	withShare := testutil.GroupShareGroup(t, groups[0].ID, &groups[1].ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Get group using its ID
			{
				Config: fmt.Sprintf(`
				data "gitlab_group" "foo" {
				  group_id = "%d"
				}
				`, groups[2].ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "id", fmt.Sprintf("%d", groups[2].ID)),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "full_path", groups[2].FullPath),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "name", groups[2].Name),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "full_name", groups[2].FullName),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "web_url", groups[2].WebURL),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "path", groups[2].Path),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "default_branch", groups[2].DefaultBranch),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "description", groups[2].Description),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "lfs_enabled", fmt.Sprintf("%t", groups[2].LFSEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "request_access_enabled", fmt.Sprintf("%t", groups[2].RequestAccessEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "parent_id", fmt.Sprintf("%d", groups[2].ParentID)),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "prevent_forking_outside_group", fmt.Sprintf("%t", groups[2].PreventForkingOutsideGroup)),
					resource.TestCheckResourceAttr("data.gitlab_group.foo", "shared_with_groups.#", "0"),
				),
			},
			// Get group using its full path
			{
				Config: fmt.Sprintf(`
				data "gitlab_group" "sub_foo" {
				  full_path = "%s"
				}
				  `, subgroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "id", fmt.Sprintf("%d", subgroup.ID)),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "full_path", subgroup.FullPath),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "name", subgroup.Name),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "full_name", subgroup.FullName),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "web_url", subgroup.WebURL),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "path", subgroup.Path),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "default_branch", subgroup.DefaultBranch),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "description", subgroup.Description),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "lfs_enabled", fmt.Sprintf("%t", subgroup.LFSEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "request_access_enabled", fmt.Sprintf("%t", subgroup.RequestAccessEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "parent_id", fmt.Sprintf("%d", subgroup.ParentID)),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "prevent_forking_outside_group", fmt.Sprintf("%t", subgroup.PreventForkingOutsideGroup)),
					resource.TestCheckResourceAttr("data.gitlab_group.sub_foo", "shared_with_groups.#", "0"),
				),
			},
			// Group shared with another group
			{
				Config: fmt.Sprintf(`
					data "gitlab_group" "this" {
						group_id = %d
					}
					`, groups[0].ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group.this", "shared_with_groups.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_group.this", "shared_with_groups.0.group_id", fmt.Sprintf("%d", withShare.SharedWithGroups[0].GroupID)),
					resource.TestCheckResourceAttr("data.gitlab_group.this", "shared_with_groups.0.expires_at", withShare.SharedWithGroups[0].ExpiresAt.String()),
					resource.TestCheckResourceAttr("data.gitlab_group.this", "shared_with_groups.0.group_full_path", withShare.SharedWithGroups[0].GroupFullPath),
					resource.TestCheckResourceAttr("data.gitlab_group.this", "shared_with_groups.0.group_access_level", fmt.Sprintf("%d", withShare.SharedWithGroups[0].GroupAccessLevel)),
				),
			},
		},
	})
}
