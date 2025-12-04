//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabGroupIDs_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`				
				data "gitlab_group_ids" "foo" {
				  group = "%s"
				}
				`, group.FullPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_ids.foo", "group_id", strconv.FormatInt(group.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_group_ids.foo", "group_full_path", group.FullPath),
					resource.TestCheckResourceAttr("data.gitlab_group_ids.foo", "group_graphql_id", fmt.Sprintf("gid://gitlab/Group/%d", group.ID)),
				),
			},
		},
	})
}
