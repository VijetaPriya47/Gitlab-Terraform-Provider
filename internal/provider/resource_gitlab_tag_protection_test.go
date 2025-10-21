//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabTagProtection_basic(t *testing.T) {
	var pt gitlab.ProtectedTag
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_tag_protection" "tag_protect" {
				  project         = %d
				  tag             = "TagProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.tag_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Configure the Tag Protection create access level
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "tag_protect" {
				  project         = %d
				  tag             = "TagProtect-%d"

				  create_access_level = "developer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.tag_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Tag Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "tag_protect" {
					project = "%d"
					tag     = "TagProtect-%d"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
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

func TestAccGitlabTagProtection_wildcard(t *testing.T) {
	var pt gitlab.ProtectedTag
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()
	wildcard := "-*"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d%s"
				}`, project.ID, rInt, wildcard),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d%s", rInt, wildcard),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Tag Protection
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d%s"
					create_access_level = "developer"
				}`, project.ID, rInt, wildcard),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d%s", rInt, wildcard),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Tag Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d%s"
				}`, project.ID, rInt, wildcard),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d%s", rInt, wildcard),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_tag_protection.TagProtect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabTagProtection_customAccessLevel_allowedToCreateUnavailableInCe(t *testing.T) {
	testutil.SkipIfEE(t)

	rInt := acctest.RandInt()

	project := testutil.CreateProject(t)

	myUser := testutil.CreateUsers(t, 1)
	testutil.AddProjectMembers(t, project.ID, myUser)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = %d
				  tag = "TagProtect-%d"
				  create_access_level = "developer"
				  allowed_to_create {
					user_id = %d
				  }
				}`,
					project.ID, rInt, myUser[0].ID),

				ExpectError: regexp.MustCompile("feature unavailable: `allowed_to_create`, Premium or Ultimate license"),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = %d
				  tag = "TagProtect-%d"
				  create_access_level = "developer"
				  allowed_to_create {
					access_level = "maintainer"
				  }
				}`,
					project.ID, rInt),

				ExpectError: regexp.MustCompile("feature unavailable: `allowed_to_create`, Premium or Ultimate license"),
			},
		},
	})
}

func TestAccGitlabTagProtection_customAccessLevel_userIdAndGroupIdAreMutuallyExclusive(t *testing.T) {
	rInt := acctest.RandInt()

	project := testutil.CreateProject(t)

	myUser := testutil.CreateUsers(t, 1)
	testutil.AddProjectMembers(t, project.ID, myUser)

	myGroup := testutil.CreateGroups(t, 1)
	testutil.ProjectShareGroup(t, project.ID, myGroup[0].ID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = %d
				  tag = "TagProtect-%d"
				  create_access_level = "developer"
				  allowed_to_create {
				    user_id = %d
				    group_id = %d
				  }
				}`,
					project.ID, rInt, myUser[0].ID, myGroup[0].ID),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func TestAccGitlabTagProtection_adminCreateAccessLevel(t *testing.T) {
	var pt gitlab.ProtectedTag
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with admin create access level
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_tag_protection" "tag_protect" {
				  project            = %d
				  tag                = "TagProtect-%d"
				  create_access_level = "admin"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: "admin",
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_tag_protection.tag_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to change from admin to maintainer
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "tag_protect" {
				  project            = %d
				  tag                = "TagProtect-%d"
				  create_access_level = "maintainer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionPersistsInStateCorrectly("gitlab_tag_protection.tag_protect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Verify import after update
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
		switch rs.Type {
		case "gitlab_project":
			project = rs.Primary.ID
		case "gitlab_tag_protection":
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
