//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupBillableMembership_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	t.Parallel()

	group := testutil.CreateGroups(t, 1)
	groupID := fmt.Sprint(group[0].ID)
	subgroups := testutil.CreateSubGroups(t, group[0], 5)
	user := testutil.CreateUsers(t, 1)
	userID := fmt.Sprint(user[0].ID)
	testutil.AddGroupMembers(t, subgroups[0].ID, user)
	testutil.AddGroupMembers(t, subgroups[2].ID, user)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_billable_member_memberships" "group_memberships" {
						group_id = "%s"
						user_id = "%s"
					}
				`, groupID, userID),
				Check: resource.ComposeTestCheckFunc(
					// check if all subgroups are returned
					resource.TestCheckResourceAttr("data.gitlab_group_billable_member_memberships.group_memberships", "memberships.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_billable_member_memberships.group_memberships", "group_id", groupID),
					resource.TestCheckResourceAttr("data.gitlab_group_billable_member_memberships.group_memberships", "id", userID),

					// for each subgroup, verify basic attributes
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_billable_member_memberships.group_memberships", "memberships.*", map[string]string{
						"source_id":        fmt.Sprint(subgroups[0].ID),
						"source_full_name": subgroups[0].FullName,
						"access_level":     "developer",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.gitlab_group_billable_member_memberships.group_memberships", "memberships.*", map[string]string{
						"source_id":        fmt.Sprint(subgroups[2].ID),
						"source_full_name": subgroups[2].FullName,
						"access_level":     "developer",
					}),
				),
			},
		},
	})
}
