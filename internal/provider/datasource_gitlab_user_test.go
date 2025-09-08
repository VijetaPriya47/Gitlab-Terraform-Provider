//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabUser_basic(t *testing.T) {
	users := testutil.CreateUsers(t, 2)
	user1 := users[0]
	user2 := users[1]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			// Check error when email doesn't match
			{
				Config: `
					data "gitlab_user" "test" {
						email = "potato"
					}
				`,
				ExpectError: regexp.MustCompile("No matching users found"),
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
			// Check error when username doesn't match
			{
				Config: `
					data "gitlab_user" "test" {
						username = "potato"
					}
				`,
				ExpectError: regexp.MustCompile("No matching users found"),
			},
		},
	})
}

func TestAccDataSourceGitlabUser_emailExactMatch(t *testing.T) {
	// Create some users for the test. Ensure more than 1 so that
	// the fuzzy test would return a non-exact match.
	user := testutil.CreateUsers(t, 5)[1]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Validate the `email_exact_match` conflicts with `username`
			{
				Config: `
					data "gitlab_user" "test" {
						username          = "asdf"
						email_exact_match = true
					}
				`,
				ExpectError: regexp.MustCompile(`"username" cannot be specified when "email_exact_match"`),
			},
			// Validate the `email_exact_match` conflicts with `user_id`
			{
				Config: `
					data "gitlab_user" "test" {
						user_id           = "1234"
						email_exact_match = true
					}
				`,
				ExpectError: regexp.MustCompile(`"user_id" cannot be specified when "email_exact_match"`),
			},
			// Validate that when we search with a valid email, we get the correct user back.
			{
				Config: fmt.Sprintf(`
					data "gitlab_user" "test" {
					  email             = "%s"
					  email_exact_match = true
					}
				`, user.Email),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_user.test", "email", user.Email),
				),
			},
			// Validate that when we search with a fuzzy match, we get an error instead of
			// an invalid user (acctest-user is the prefix for all test created users with the helper)
			{
				Config: `
					data "gitlab_user" "test" {
					  email             = "acctest-user@example.com"
					  email_exact_match = true
					}
				`,
				ExpectError: regexp.MustCompile("No users matching email acctest-user@example.com"),
			},
		},
	})
}
