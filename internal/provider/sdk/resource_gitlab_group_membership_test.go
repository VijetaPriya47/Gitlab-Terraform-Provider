//go:build acceptance
// +build acceptance

package sdk

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupMembership_basic(t *testing.T) {
	var groupMember gitlab.GroupMember
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{

			// Assign member to the group as a developer
			{
				Config: testAccGitlabGroupMembershipConfig(rInt),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},

			//Update the group member to change the access level (use testAccGitlabGroupMembershipUpdateConfig for Config)
			{
				Config: testAccGitlabGroupMembershipUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "guest",
					expiresAt:   "2099-01-01",
				})),
			},

			// Update the group member to change the access level back
			{
				Config: testAccGitlabGroupMembershipConfig(rInt),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},
		},
	})
}

func TestAccGitlabGroupMembership_skipRemoveFromSubgroup(t *testing.T) {
	testUser := testutil.CreateUsers(t, 1)[0]
	testGroup := testutil.CreateGroups(t, 1)[0]
	testSubgroup := testutil.CreateSubGroups(t, testGroup, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{
			// Add user to main and subgroup individually
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_membership" "main_group" {
						group_id                     = "%d"
						user_id                      = %d
						access_level                 = "developer"
						skip_subresources_on_destroy = true
					}

					resource "gitlab_group_membership" "sub_group" {
						group_id     = "%d"
						user_id      = %d
						access_level = "maintainer"
					}
				`, testGroup.ID, testUser.ID, testSubgroup.ID, testUser.ID),
			},
			// Remove user from main group without removing from subgroup
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_membership" "sub_group" {
						group_id     = "%d"
						user_id      = %d
						access_level = "maintainer"
					}
				`, testSubgroup.ID, testUser.ID),
				Check: testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.sub_group", &gitlab.GroupMember{}),
			},
		},
	})
}

func TestAccGitlabGroupMembership_useCustomRole(t *testing.T) {
	// custom roles only available to EE ultimate
	testutil.SkipIfCE(t)

	// Group level custom roles don't work on self managed, so we can't test them without a SaaS project.
	// See https://gitlab.com/gitlab-org/gitlab/-/issues/439284 for more details
	t.Skip()

	// create a user
	user := testutil.CreateUsers(t, 1)[0]
	// create a group to give them a membership to
	group := testutil.CreateGroups(t, 1)[0]

	// Create a custom role on that group - we don't need to clean this up, since it's bound to the group
	// which will be deleted when the test finishes.
	roleOne, _, errOne := testutil.TestGitlabClient.MemberRolesService.CreateMemberRole(group.ID, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr("test-role"),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	// Create a second custom role on that group (for testing update)
	roleTwo, _, errTwo := testutil.TestGitlabClient.MemberRolesService.CreateMemberRole(group.ID, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr("test-role-update"),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	// If either of our role creations fail, short-circuit the test
	err := errors.Join(errOne, errTwo)
	if err != nil {
		t.Fatalf("Failed to create one of the two testing roles. Error: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{
			// Assign member to the group as a custom maintainer-based role
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_group_membership" "foo" {
						group_id        = "%d"
						user_id         = "%d"
						access_level 	= "maintainer"
						member_role_id  = %d
					}
					`, group.ID, user.ID, roleOne.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "group_id", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "member_role_id", strconv.Itoa(roleOne.ID)),
				),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"skip_subresources_on_destroy",
					"unassign_issuables_on_destroy",
				},
			},
			// Assign member to the group as a separate custom maintainer-based role
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_group_membership" "foo" {
						group_id        = "%d"
						user_id         = "%d"
						access_level    = "maintainer"
						member_role_id  = %d
					}
					`, group.ID, user.ID, roleTwo.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "group_id", strconv.Itoa(group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "member_role_id", strconv.Itoa(roleTwo.ID)),
				),
			},
		},
	})
}

func testAccCheckGitlabGroupMembershipExists(n string, membership *gitlab.GroupMember) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		groupId := rs.Primary.Attributes["group_id"]
		if groupId == "" {
			return fmt.Errorf("no group ID is set")
		}

		userIdString := rs.Primary.Attributes["user_id"]
		userId, _ := strconv.Atoi(userIdString)
		if userIdString == "" {
			return fmt.Errorf("No user userId is set")
		}

		gotGroupMembership, _, err := testutil.TestGitlabClient.GroupMembers.GetGroupMember(groupId, userId)
		if err != nil {
			return err
		}

		*membership = *gotGroupMembership
		return nil
	}
}

type testAccGitlabGroupMembershipExpectedAttributes struct {
	accessLevel string
	expiresAt   string
}

func testAccCheckGitlabGroupMembershipAttributes(membership *gitlab.GroupMember, want *testAccGitlabGroupMembershipExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		accessLevelId, ok := api.AccessLevelValueToName[membership.AccessLevel]
		if !ok {
			return fmt.Errorf("Invalid access level '%s'", accessLevelId)
		}
		if accessLevelId != want.accessLevel {
			return fmt.Errorf("got access level %s; want %s", accessLevelId, want.accessLevel)
		}
		return nil
	}
}

func testAccCheckGitlabGroupMembershipDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_membership" {
			continue
		}

		groupId := rs.Primary.Attributes["group_id"]
		userIdString := rs.Primary.Attributes["user_id"]

		// GetGroupMember needs int type for userIdString
		userId, err := strconv.Atoi(userIdString) // nolint // TODO: Resolve this golangci-lint issue: ineffectual assignment to err (ineffassign)
		groupMember, _, err := testutil.TestGitlabClient.GroupMembers.GetGroupMember(groupId, userId)
		if err != nil {
			if groupMember != nil && fmt.Sprintf("%d", groupMember.AccessLevel) == rs.Primary.Attributes["access_level"] {
				return fmt.Errorf("Group still has member.")
			}
			return nil
		}

		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

func testAccGitlabGroupMembershipConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo%d"
  path = "foo%d"
}

resource "gitlab_user" "test" {
  name 		= "foo%d"
  username  = "listest%d"
  password  = "SvNwfHhbvPmHZr-%d"
  email 	= "listest%d@ssss.com"
}

resource "gitlab_group_membership" "foo" {
  group_id 		= "${gitlab_group.foo.id}"
  user_id 		= "${gitlab_user.test.id}"
  access_level 	= "developer"
}`, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccGitlabGroupMembershipUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo%d"
  path = "foo%d"
}

resource "gitlab_user" "test" {
  name 		= "foo%d"
  username 	= "listest%d"
  password 	= "SvNwfHhbvPmHZr-%d"
  email 	= "listest%d@ssss.com"
}

resource "gitlab_group_membership" "foo" {
  group_id 		= "${gitlab_group.foo.id}"
  user_id 		= "${gitlab_user.test.id}"
  expires_at    = "2099-01-01"
  access_level 	= "guest"
}`, rInt, rInt, rInt, rInt, rInt, rInt)
}
