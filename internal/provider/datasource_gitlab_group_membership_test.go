//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupMembership_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the group and one member
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
				  group_id     = "%d"
				  user_id      = "%d"
				  access_level = "developer"
				}`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "access_level", "developer"),
				),
			},
			{
				Config: fmt.Sprintf(`
				data "gitlab_group_membership" "foo" {
				  group_id = "%d"
				}`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					// Members is 2 because the user owning the token is always added to the group
					resource.TestCheckResourceAttr("data.gitlab_group_membership.foo", "members.#", "2"),
					resource.TestCheckResourceAttr("data.gitlab_group_membership.foo", "members.1.username", user.Username),
				),
			},

			// Get group using its ID, but return maintainers only
			{
				Config: fmt.Sprintf(`
				data "gitlab_group_membership" "foomaintainers" {
				  group_id     = "%d"
				  access_level = "maintainer"
				}`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_membership.foomaintainers", "members.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabGroupMembership_inherited(t *testing.T) {
	// create the parent group
	parentGroup := testutil.CreateGroups(t, 1)[0]
	// create the nested group
	nestedGroup := testutil.CreateSubGroups(t, parentGroup, 1)[0]
	// create user
	user := testutil.CreateUsers(t, 1)
	// add user to the parent_group (will be added as Developer)
	testutil.AddGroupMembers(t, parentGroup.ID, user)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_membership" "this" {
					  group_id     = "%d"
					  access_level = "developer"
					  inherited    = true
					}`, nestedGroup.ID),
				Check: resource.TestCheckResourceAttr("data.gitlab_group_membership.this", "members.0.username", user[0].Username),
			},
		},
	})
}

func TestAccDataSourceGitlabGroupMembership_pagination(t *testing.T) {
	userCount := 21

	group := testutil.CreateGroups(t, 1)[0]
	users := testutil.CreateUsers(t, userCount)
	testutil.AddGroupMembers(t, group.ID, users)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_group_membership" "this" {
				  group_id     = "%d"
				  access_level = "developer"
				}`, group.ID),
				Check: resource.TestCheckResourceAttr("data.gitlab_group_membership.this", "members.#", fmt.Sprintf("%d", userCount)),
			},
		},
	})
}

func TestAccDataSourceGitlabGroupMembership_samlIdentity(t *testing.T) {
	// This test verifies that the group_saml_identity field is properly handled
	// when it's null (which is the common case without SAML configuration).
	// Testing with actual SAML data would require a SAML provider setup.
	group := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembers(t, group.ID, []*gitlab.User{user})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_group_membership" "saml_test" {
				  group_id = "%d"
				}`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					// Verify members are returned
					resource.TestCheckResourceAttr("data.gitlab_group_membership.saml_test", "members.#", "2"),
					// Verify group_saml_identity doesn't cause errors when null
					// The field should not be set/populated for regular users without SAML
					resource.TestCheckResourceAttr("data.gitlab_group_membership.saml_test", "members.1.username", user.Username),
				),
			},
		},
	})
}
