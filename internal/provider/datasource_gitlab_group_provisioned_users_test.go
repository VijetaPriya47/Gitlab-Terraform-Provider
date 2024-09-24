//go:build acceptance
// +build acceptance

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataGitlabGroupProvisionedUsers_basic(t *testing.T) {
	t.Skip("Skip this test since we can't automatically add provisioned users. This is usable locally after using rails to add a user.")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				data "gitlab_group_provisioned_users" "foo" {
					id = 2
					created_after = "2024-09-20T06:15:29Z"
					active = true
					blocked = false
					created_before = "2024-09-21T06:15:29Z"
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "id", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "created_after", "2024-09-20T06:15:29Z"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "created_before", "2024-09-21T06:15:29Z"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "active", "true"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "blocked", "false"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.id", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.name", "Hoa nguyen 1"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.username", "nvh0412"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.state", "active"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.bio", "Bio: test terraform"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.last_sign_in_at", "2024-09-20T06:15:29Z"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.job_title", "Software Engineer"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.location", "Sydney"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.two_factor_enabled", "false"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.external", "false"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.private_profile", "false"),
				),
			},
		},
	})
}

func TestAccDataGitlabGroupProvisionedUsers_search(t *testing.T) {
	t.Skip("Skip this test since we can't automatically add provisioned users. This is usable locally after using rails to add a user.")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				data "gitlab_group_provisioned_users" "foo" {
					id = 2
					search = "Hoa nguyen 1"
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "id", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.id", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.name", "Hoa nguyen 1"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.username", "nvh0412"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.state", "active"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.bio", "Bio: test terraform"),
					resource.TestCheckResourceAttr("data.gitlab_group_provisioned_users.foo", "provisioned_users.0.last_sign_in_at", "2024-09-20T06:15:29Z"),
				),
			},
		},
	})
}
