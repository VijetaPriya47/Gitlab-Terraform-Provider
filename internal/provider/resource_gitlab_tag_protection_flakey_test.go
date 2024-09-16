//go:build flakey
// +build flakey

package provider

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

// This test is flakey because the users aren't granted permissions on the project
// right away, it seems to be more of an async operation.  Even with the long sleep,
// the test will still fail because the users getting added as allowed to create
// aren't yet members of the project.
func TestAccGitlabTagProtection_customAccessLevel(t *testing.T) {

	// This test is VERY flakey, so we're going to skip it until we can figure out how
	// to cache bust group membership. We're not sure how to do that via API right now.
	t.Skip()

	// Below is the rest of the test.
	testutil.SkipIfCE(t)

	var pt gitlab.ProtectedTag
	rInt := acctest.RandInt()

	// Project to set protections for
	project := testutil.CreateProject(t)

	// Set of user/group for create
	myUser := testutil.CreateUsers(t, 1)
	myGroup := testutil.CreateGroups(t, 1)

	// Set of user/group for update
	myUpdatedUser := testutil.CreateUsers(t, 1)
	myUpdatedGroup := testutil.CreateGroups(t, 1)

	// Add new users and groups to the project
	// Yes, this could be slightly easier if I passed "2" above, but this makes the
	// tests more readable below.
	testutil.AddProjectMembers(t, project.ID, myUser)
	testutil.AddProjectMembers(t, project.ID, myUpdatedUser)
	testutil.ProjectShareGroup(t, project.ID, myGroup[0].ID)
	testutil.ProjectShareGroup(t, project.ID, myUpdatedGroup[0].ID)

	// add a sleep since project membership is async and takes awhile for
	// it to take effect in order for the tests below to pass
	t.Log("Sleeping for 60s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(60 * time.Second)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = "%d"
				  tag = "TagProtect-%d"

				  allowed_to_create {
					user_id = %d
				  }
				  allowed_to_create {
					group_id = %d
				  }
				}
				`, project.ID, rInt, myUser[0].ID, myGroup[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:                  fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToCreate:  []string{myUser[0].Username},
						GroupsAllowedToCreate: []string{myGroup[0].Name},
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Tag Protection and set "create_access_level" to "developer"
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = "%d"
				  tag = "TagProtect-%d"

				  # Update to developer permission
				  create_access_level = "developer"

				  allowed_to_create {
					user_id = %d
				  }
				  allowed_to_create {
					group_id = %d
				  }
				}
				`, project.ID, rInt, myUpdatedUser[0].ID, myUpdatedGroup[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:                  fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel:     api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						UsersAllowedToCreate:  []string{myUpdatedUser[0].Username},
						GroupsAllowedToCreate: []string{myUpdatedGroup[0].Name},
					}),
				),
			},
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Tag Protection and set "create_access_level to "no one"
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = "%d"
				  tag = "TagProtect-%d"

				  # Update to no one permission
				  create_access_level = "no one"

				  allowed_to_create {
					user_id = %d
				  }
				  allowed_to_create {
					group_id = %d
				  }
				}
				`, project.ID, rInt, myUpdatedUser[0].ID, myUpdatedGroup[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:                  fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel:     api.AccessLevelValueToName[gitlab.NoPermissions],
						UsersAllowedToCreate:  []string{myUpdatedUser[0].Username},
						GroupsAllowedToCreate: []string{myUpdatedGroup[0].Name},
					}),
				),
			},
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Configure the Tag Protection allowed to create using access_level
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "tag_protect" {
				  project         = %d
				  tag             = "TagProtect-%d"

				  allowed_to_create {
					access_level = "developer"
				  }
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:                        fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel:           api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						AccessLevelsAllowedToCreate: []string{api.AccessLevelValueToName[gitlab.DeveloperPermissions]},
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.tag_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabTagProtectionPersistsInStateCorrectly(n string, pt *gitlab.ProtectedTag) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		var createAccessLevel gitlab.AccessLevelValue
		for _, v := range pt.CreateAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				createAccessLevel = v.AccessLevel
				break
			}
		}
		if rs.Primary.Attributes["create_access_level"] != api.AccessLevelValueToName[createAccessLevel] {
			return fmt.Errorf("create access level not persisted in state correctly")
		}

		if createAccessLevel, err := firstValidAccessLevelForTag(pt.CreateAccessLevels); err == nil {
			if rs.Primary.Attributes["create_access_level"] != api.AccessLevelValueToName[createAccessLevel.AccessLevel] {
				return fmt.Errorf("create access level not persisted in state correctly")
			}
		}

		return nil
	}
}

func testAccCheckGitlabTagProtectionExists(n string, pt *gitlab.ProtectedTag) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}
		project, tag, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error in Splitting Project and Tag Ids")
		}

		pts, _, err := testutil.TestGitlabClient.ProtectedTags.ListProtectedTags(project, nil)
		if err != nil {
			return err
		}
		for _, gotpt := range pts {
			if gotpt.Name == tag {
				*pt = *gotpt
				return nil
			}
		}
		return fmt.Errorf("Protected Tag does not exist")
	}
}

type testAccGitlabTagProtectionExpectedAttributes struct {
	Name                        string
	CreateAccessLevel           string
	UsersAllowedToCreate        []string
	GroupsAllowedToCreate       []string
	AccessLevelsAllowedToCreate []string
}

func testAccCheckGitlabTagProtectionAttributes(pt *gitlab.ProtectedTag, want *testAccGitlabTagProtectionExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if pt.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", pt.Name, want.Name)
		}

		var createAccessLevel *gitlab.TagAccessDescription
		for _, v := range pt.CreateAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				createAccessLevel = v
				break
			}
		}
		if createAccessLevel.AccessLevel != api.AccessLevelNameToValue[want.CreateAccessLevel] {
			return fmt.Errorf("got create access level %v; want %v", createAccessLevel, api.AccessLevelNameToValue[want.CreateAccessLevel])
		} else {
			// remove the corresponding create access level so it's not added to 'allowed to create' in the state
			pt.CreateAccessLevels = slices.DeleteFunc(pt.CreateAccessLevels, func(desc *gitlab.TagAccessDescription) bool {
				return desc == createAccessLevel
			})

			pt.CreateAccessLevels = slices.Clip(pt.CreateAccessLevels)
		}

		remainingWantedAccessLevelsAllowedToCreate := map[int]struct{}{}
		for _, v := range want.AccessLevelsAllowedToCreate {
			remainingWantedAccessLevelsAllowedToCreate[int(api.AccessLevelNameToValue[v])] = struct{}{}
		}

		remainingWantedUserIDsAllowedToCreate := map[int]struct{}{}
		for _, v := range want.UsersAllowedToCreate {
			users, _, err := testutil.TestGitlabClient.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: gitlab.Ptr(v),
			})
			if err != nil {
				return fmt.Errorf("error looking up user by path %v: %v", v, err)
			}
			if len(users) != 1 {
				return fmt.Errorf("error finding user by username %v; found %v", v, len(users))
			}
			remainingWantedUserIDsAllowedToCreate[users[0].ID] = struct{}{}
		}
		remainingWantedGroupIDsAllowedToCreate := map[int]struct{}{}
		for _, v := range want.GroupsAllowedToCreate {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToCreate[group.ID] = struct{}{}
		}
		for _, v := range pt.CreateAccessLevels {
			if v.UserID != 0 {
				if _, ok := remainingWantedUserIDsAllowedToCreate[v.UserID]; !ok {
					return fmt.Errorf("found unwanted user ID %v", v.UserID)
				}
				delete(remainingWantedUserIDsAllowedToCreate, v.UserID)
			} else if v.GroupID != 0 {
				if _, ok := remainingWantedGroupIDsAllowedToCreate[v.GroupID]; !ok {
					return fmt.Errorf("found unwanted group ID %v", v.GroupID)
				}
				delete(remainingWantedGroupIDsAllowedToCreate, v.GroupID)
			} else if api.AccessLevelValueToName[v.AccessLevel] != "" {
				if _, ok := remainingWantedAccessLevelsAllowedToCreate[int(v.AccessLevel)]; !ok {
					return fmt.Errorf("found unwanted access level %v", v.AccessLevel)
				}
				delete(remainingWantedAccessLevelsAllowedToCreate, int(v.AccessLevel))
			}
		}
		if len(remainingWantedUserIDsAllowedToCreate) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToCreate)
		}
		if len(remainingWantedGroupIDsAllowedToCreate) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToCreate)
		}
		if len(remainingWantedAccessLevelsAllowedToCreate) > 0 {
			return fmt.Errorf("failed to find wanted access levels %v", remainingWantedAccessLevelsAllowedToCreate)
		}

		return nil
	}
}

func testAccCheckGitlabTagProtectionDestroy(s *terraform.State) error {
	var project string
	var tag string
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project" {
			project = rs.Primary.ID
		} else if rs.Type == "gitlab_tag_protection" {
			tag = rs.Primary.ID
		}
	}

	pt, _, err := testutil.TestGitlabClient.ProtectedTags.GetProtectedTag(project, tag)
	if err == nil {
		if pt != nil {
			return fmt.Errorf("project tag protection %s still exists", tag)
		}
	}
	if !api.Is404(err) {
		return err
	}
	return nil
}
