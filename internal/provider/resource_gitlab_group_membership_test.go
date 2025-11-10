//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupMembership_basic(t *testing.T) {
	var groupMember gitlab.GroupMember
	group := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{
			// Assign member to the group as a developer
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group member to change the access level
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "guest"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "guest",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group member to change the access level back
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group member to add an expiry
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
					expires_at      = "2099-01-01"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
					expiresAt:   "2099-01-01",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupMembership_skipRemoveFromSubgroup(t *testing.T) {
	testUser := testutil.CreateUsers(t, 1)[0]
	testGroup := testutil.CreateGroups(t, 1)[0]
	testSubgroup := testutil.CreateSubGroups(t, testGroup, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

	// create a user
	user := testutil.CreateUsers(t, 1)[0]
	// create a group to give them a membership to
	group := testutil.CreateGroups(t, 1)[0]

	// Create an instance role (since group roles don't work on self-hosted anymore)
	rInt := acctest.RandInt()
	roleOne := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	roleTwo := testutil.CreateCustomInstanceRole(t, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-two-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})

	checkGroupMembershipViaAPI := func(s *terraform.State) error {
		group, _, err := testutil.TestGitlabClient.GroupMembers.GetGroupMember(group.ID, user.ID)
		if err != nil {
			return fmt.Errorf("Error getting group member via API: %v", err)
		}

		if group.MemberRole != nil {
			return fmt.Errorf("API CHECK FAILED: MemberRoleID should be nil: got %v", group.MemberRole.ID)
		}

		// Return nil to indicate the check passed.
		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_group_membership" "foo" {
						group_id        = "%d"
						user_id         = "%d"
						access_level    = "maintainer"
					}
					`, group.ID, user.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_membership.foo", "group_id", strconv.Itoa(group.ID)),
					// Assert that the member_role_id is no longer set in the state.
					resource.TestCheckNoResourceAttr("gitlab_group_membership.foo", "member_role_id"),
					checkGroupMembershipViaAPI,
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

		group := rs.Primary.Attributes["group_id"]
		if group == "" {
			return fmt.Errorf("no group is set")
		}

		userIdString := rs.Primary.Attributes["user_id"]
		userId, _ := strconv.Atoi(userIdString)
		if userIdString == "" {
			return fmt.Errorf("No user userId is set")
		}

		gotGroupMembership, _, err := testutil.TestGitlabClient.GroupMembers.GetGroupMember(group, userId)
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
		userId, err := strconv.Atoi(userIdString)
		if err != nil {
			return fmt.Errorf("Error when checking destroy. Unable to convert user_id to integer: %v", err)
		}
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

func TestAccGitlabGroupMembership_migrateFromSDKToFramework(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= %d
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}`, group.ID, user.ID),
				Check: resource.TestCheckResourceAttr("gitlab_group_membership.foo", "access_level", "developer"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}`, group.ID, user.ID),
				Check: resource.TestCheckResourceAttr("gitlab_group_membership.foo", "access_level", "developer"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_group_membership.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore: []string{
					"skip_subresources_on_destroy",
					"unassign_issuables_on_destroy",
				},
			},
		},
	})
}

func TestAccGitlabGroupMembership_No404WhenRemovedOutsideTF(t *testing.T) {
	var groupMember gitlab.GroupMember
	group := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupMembershipDestroy,
		Steps: []resource.TestStep{
			// Assign member to the group as a developer
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove the user from the group outside of terraform and verify we don't get a 404
			{
				PreConfig: func() {
					if _, err := testutil.TestGitlabClient.GroupMembers.RemoveGroupMember(group.ID, user.ID, nil, nil); err != nil {
						t.Errorf("Error removing group member from test group: %s", err)
					}
				},
				Config: fmt.Sprintf(`
				resource "gitlab_group_membership" "foo" {
  				    group_id 		= "%d"
  				    user_id 		= "%d"
  				    access_level 	= "developer"
				}
				`, group.ID, user.ID),
				Check: resource.ComposeTestCheckFunc(testAccCheckGitlabGroupMembershipExists("gitlab_group_membership.foo", &groupMember), testAccCheckGitlabGroupMembershipAttributes(&groupMember, &testAccGitlabGroupMembershipExpectedAttributes{
					accessLevel: "developer",
				})),
			},
			{
				ResourceName:      "gitlab_group_membership.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
