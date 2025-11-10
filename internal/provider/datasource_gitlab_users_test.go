//go:build acceptance

package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabUsers_basic(t *testing.T) {
	rInt := acctest.RandInt()
	testutil.CreateUsersWithPrefix(t, 12, fmt.Sprintf("ds-%d-acctest-a", rInt))
	testUsersGroupB := testutil.CreateUsersWithPrefix(t, 12, fmt.Sprintf("ds-%d-acctest-b", rInt))
	testUsername := testUsersGroupB[0].Username
	project := testutil.CreateProject(t)
	accessTokenName := "acc-tests"
	testutil.CreateProjectAccessToken(t, project.ID, accessTokenName, []string{"read_repository", "read_registry"}, gitlab.DeveloperPermissions, nil)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_users" "test" {
					  search = "ds-%d-acctest-"

					  sort     = "desc"
					  order_by = "name"
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_users.test", "users.#", "24"),
					resource.TestCheckResourceAttrWith("data.gitlab_users.test", "users.0.username", func(value string) error {
						if !strings.HasPrefix(value, fmt.Sprintf("ds-%d-acctest-b-", rInt)) {
							return fmt.Errorf("expected first user to be of group a with prefix `ds-acctest-a-` got `%s` instead", value)
						}
						return nil
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_users" "test" {
					  search = "ds-%d-acctest-b-"
					}
				`, rInt),
				Check: resource.TestCheckResourceAttr("data.gitlab_users.test", "users.#", fmt.Sprintf("%d", len(testUsersGroupB))),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_users" "test" {
						username = "%s"
					}
				`, testUsername),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_users.test", "users.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_users.test", "users.0.username", testUsername),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_users" "test" {
						search = "project_%d_bot_"
						humans = false
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_users.test", "users.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(`
					data "gitlab_users" "test" {
						search = "project_%d_bot_"
						humans = true
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_users.test", "users.#", "0"),
				),
			},
		},
	})
}
