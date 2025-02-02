//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroups_basic(t *testing.T) {
	prefixFoo := "acctest-group-foo"
	groupsFoo := testutil.CreateGroupsWithPrefix(t, 2, prefixFoo)

	prefixLotsOf := "acctest-group-lotsof"
	testutil.CreateGroupsWithPrefix(t, 42, prefixLotsOf)

	prefixParent := "acctest-group-parent"
	testGroup := testutil.CreateGroupsWithPrefix(t, 1, prefixParent)[0]
	testSubgroup := testutil.CreateSubGroupsWithPrefix(t, testGroup, 1, prefixParent)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: `
					data "gitlab_groups" "foos" {
						sort = "asc"
						search = "acctest-group-foo"
						order_by = "id"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.group_id", fmt.Sprint(groupsFoo[0].ID)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.full_path", groupsFoo[0].FullPath),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.name", groupsFoo[0].Name),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.full_name", groupsFoo[0].FullName),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.web_url", groupsFoo[0].WebURL),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.path", groupsFoo[0].Path),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.description", groupsFoo[0].Description),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.lfs_enabled", strconv.FormatBool(groupsFoo[0].LFSEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.request_access_enabled", strconv.FormatBool(groupsFoo[0].RequestAccessEnabled)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.visibility_level", fmt.Sprint(groupsFoo[0].Visibility)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.parent_id", fmt.Sprint(groupsFoo[0].ParentID)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.runners_token", groupsFoo[0].RunnersToken),

					// nolint:staticcheck // SA1019 ignore deprecated DefaultBranchProtection
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.default_branch_protection", fmt.Sprint(groupsFoo[0].DefaultBranchProtection)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.prevent_forking_outside_group", strconv.FormatBool(groupsFoo[0].PreventForkingOutsideGroup)),
					resource.TestCheckResourceAttr("data.gitlab_groups.foos", "groups.0.shared_runners_setting", fmt.Sprint(groupsFoo[0].SharedRunnersSetting)),
				),
			},
			{
				Config: `
					data "gitlab_groups" "lotsof" {
						search = "acctest-group-lotsof"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_groups.lotsof", "groups.#", "42"),
				),
			},
			{
				Config: `
					data "gitlab_groups" "toplevel" {
						top_level_only = true
						search = "acctest-group-parent"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_groups.toplevel", "groups.#", "1"),
				),
			},
			{
				Config: `
					data "gitlab_groups" "sublevel" {
						top_level_only = false
						search = "acctest-group-parent"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					// check if all subgroups are returned
					resource.TestCheckResourceAttr("data.gitlab_groups.sublevel", "groups.#", "2"),
					// for each subgroup, verify basic attributes
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_groups.sublevel", "groups.*", map[string]string{
						"group_id":  fmt.Sprint(testSubgroup.ID),
						"parent_id": fmt.Sprint(testGroup.ID),
					}),
				),
			},
		},
	})
}
