//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabSubGroups_basic(t *testing.T) {
	t.Parallel()

	group := testutil.CreateGroups(t, 1)
	groupID := fmt.Sprint(group[0].ID)
	subgroups := testutil.CreateSubGroups(t, group[0], 5)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_subgroups" "subs_foo" {
						group_id = "%s"
					}
				`, groupID),
				Check: resource.ComposeTestCheckFunc(
					// check if all subgroups are returned
					resource.TestCheckResourceAttr("data.gitlab_group_subgroups.subs_foo", "subgroups.#", "5"),

					// for each subgroup, verify basic attributes
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[0].ID),
						"name":      subgroups[0].Name,
						"path":      subgroups[0].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[1].ID),
						"name":      subgroups[1].Name,
						"path":      subgroups[1].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[2].ID),
						"name":      subgroups[2].Name,
						"path":      subgroups[2].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[3].ID),
						"name":      subgroups[3].Name,
						"path":      subgroups[3].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[4].ID),
						"name":      subgroups[4].Name,
						"path":      subgroups[4].Path,
						"parent_id": groupID,
					}),
				),
			},
			{
				// set skip_group param
				Config: fmt.Sprintf(`
					data "gitlab_group_subgroups" "subs_foo" {
						group_id = "%s"
						skip_groups=[%d, %d]
					}
				`, groupID, subgroups[0].ID, subgroups[3].ID),
				Check: resource.ComposeTestCheckFunc(
					// after apply, there should be 2 groups less
					resource.TestCheckResourceAttr("data.gitlab_group_subgroups.subs_foo", "subgroups.#", "3"),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[1].ID),
						"name":      subgroups[1].Name,
						"path":      subgroups[1].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[2].ID),
						"name":      subgroups[2].Name,
						"path":      subgroups[2].Path,
						"parent_id": groupID,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[4].ID),
						"name":      subgroups[4].Name,
						"path":      subgroups[4].Path,
						"parent_id": groupID,
					}),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabSubGroups_subgroupPagination(t *testing.T) {
	t.Parallel()
	group := testutil.CreateGroups(t, 1)
	groupID := fmt.Sprint(group[0].ID)

	// Needs to be greater than 20 to test pagination
	subgroups := testutil.CreateSubGroups(t, group[0], 25)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_subgroups" "subs_foo" {
						group_id = "%s"
					}
				`, groupID),
				Check: resource.ComposeTestCheckFunc(
					// check if all subgroups are returned
					resource.TestCheckResourceAttr("data.gitlab_group_subgroups.subs_foo", "subgroups.#", "25"),

					// Test the first subgroup is still set properly when paginating
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_subgroups.subs_foo", "subgroups.*", map[string]string{
						"group_id":  fmt.Sprint(subgroups[0].ID),
						"name":      subgroups[0].Name,
						"path":      subgroups[0].Path,
						"parent_id": groupID,
					}),
				),
			},
		},
	})
}
