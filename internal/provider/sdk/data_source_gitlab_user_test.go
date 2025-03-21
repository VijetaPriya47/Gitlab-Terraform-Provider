//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabUser_basic(t *testing.T) {
	users := testutil.CreateUsers(t, 2)
	user1 := users[0]
	user2 := users[1]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Get user by email
			{
				Config: fmt.Sprintf(`				
				data "gitlab_user" "foo" {
				  email = "%s"
				}
				`, user1.Email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "username", user1.Username),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "email", user1.Email),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "name", user1.Name),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "is_admin", fmt.Sprintf("%t", user1.IsAdmin)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "can_create_group", fmt.Sprintf("%t", user1.CanCreateGroup)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "projects_limit", fmt.Sprintf("%d", user1.ProjectsLimit)),
				),
			},
			// Get user by ID
			{
				Config: fmt.Sprintf(`
				data "gitlab_user" "foo2" {
				  user_id = "%d"
				}
				`, user2.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "username", user2.Username),
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "email", user2.Email),
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "name", user2.Name),
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "is_admin", fmt.Sprintf("%t", user2.IsAdmin)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "can_create_group", fmt.Sprintf("%t", user2.CanCreateGroup)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo2", "projects_limit", fmt.Sprintf("%d", user2.ProjectsLimit)),
				),
			},
			// Get user by username
			{
				Config: fmt.Sprintf(`				
				data "gitlab_user" "foo" {
				  username = "%s"
				}
				`, user1.Username),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "username", user1.Username),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "email", user1.Email),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "name", user1.Name),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "is_admin", fmt.Sprintf("%t", user1.IsAdmin)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "can_create_group", fmt.Sprintf("%t", user1.CanCreateGroup)),
					resource.TestCheckResourceAttr("data.gitlab_user.foo", "projects_limit", fmt.Sprintf("%d", user1.ProjectsLimit)),
				),
			},
		},
	})
}
