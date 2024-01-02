//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroup_basic(t *testing.T) {
	rString := fmt.Sprintf("%s", acctest.RandString(5)) // nolint // TODO: Resolve this golangci-lint issue: S1025: the argument is already a string, there's no need to use fmt.Sprintf (gosimple)

	groups := testutil.CreateGroups(t, 2)
	withShare := testutil.GroupShareGroup(t, groups[0].ID, &groups[1].ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Get group using its ID
			{
				Config: testAccDataGitlabGroupByID(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabGroup("gitlab_group.foo", "data.gitlab_group.foo"),
				),
			},
			// Get group using its full path
			{
				Config: testAccDataGitlabGroupByFullPath(rString),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabGroup("gitlab_group.sub_foo", "data.gitlab_group.sub_foo"),
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
					resource.TestCheckResourceAttr(
						"data.gitlab_group.this",
						"shared_with_groups.#",
						"1"),
					resource.TestCheckResourceAttr(
						"data.gitlab_group.this",
						"shared_with_groups.0.group_id",
						fmt.Sprintf("%d", withShare.SharedWithGroups[0].GroupID)),
					resource.TestCheckResourceAttr(
						"data.gitlab_group.this",
						"shared_with_groups.0.expires_at",
						withShare.SharedWithGroups[0].ExpiresAt.String()),
					resource.TestCheckResourceAttr(
						"data.gitlab_group.this",
						"shared_with_groups.0.group_full_path",
						withShare.SharedWithGroups[0].GroupFullPath),
					resource.TestCheckResourceAttr(
						"data.gitlab_group.this",
						"shared_with_groups.0.group_access_level",
						fmt.Sprintf("%d", withShare.SharedWithGroups[0].GroupAccessLevel)),
				),
			},
		},
	})
}
func testAccDataSourceGitlabGroup(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		group := s.RootModule().Resources[src]
		groupResource := group.Primary.Attributes

		search := s.RootModule().Resources[n]
		searchResource := search.Primary.Attributes

		testAttributes := []string{
			"id",
			"full_path",
			"name",
			"full_name",
			"web_url",
			"path",
			"description",
			"lfs_enabled",
			"request_access_enabled",
			"visibility_level",
			"parent_id",
			"default_branch_protection",
			"prevent_forking_outside_group",
			"shared_runners_setting",
		}
		for _, attribute := range testAttributes {
			if searchResource[attribute] != groupResource[attribute] {
				return fmt.Errorf("expected group's parameter `%s` to be: %s, but got: `%s`", attribute, groupResource[attribute], searchResource[attribute])
			}
		}

		return nil
	}
}

func testAccDataGitlabGroupByID(rString string) string {
	return fmt.Sprintf(`
%s

data "gitlab_group" "foo" {
  group_id = "${gitlab_group.foo.id}"
}
`, testAccDataGitlabGroupSetup(rString))
}

func testAccDataGitlabGroupByFullPath(rString string) string {
	return fmt.Sprintf(`
%s

data "gitlab_group" "sub_foo" {
  full_path = "${gitlab_group.foo.path}/${gitlab_group.sub_foo.path}"
}
`, testAccDataGitlabGroupSetup(rString))
}

func testAccDataGitlabGroupSetup(rString string) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%[1]s"
  path = "foo-path-%[1]s"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_group" "sub_foo" {
  name      = "sub-foo-name-%[1]s"
  path      = "sub-foo-path-%[1]s"
  parent_id = "${gitlab_group.foo.id}"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rString)
}
