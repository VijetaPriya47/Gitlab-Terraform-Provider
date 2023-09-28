//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabTagProtection_basic(t *testing.T) {
	var pt gitlab.ProtectedTag
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d"
					create_access_level = "developer"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
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
			// Update the Tag Protection
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d"
					create_access_level = "maintainer"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
						CreateAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Tag Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d"
					create_access_level = "developer"
				}`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTagProtectionExists("gitlab_tag_protection.TagProtect", &pt),
					testAccCheckGitlabTagProtectionAttributes(&pt, &testAccGitlabTagProtectionExpectedAttributes{
						Name:              fmt.Sprintf("TagProtect-%d", rInt),
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
		},
	})
}

func TestAccGitlabTagProtection_wildcard(t *testing.T) {
	var pt gitlab.ProtectedTag
	project := testutil.CreateProject(t)
	rInt := acctest.RandInt()
	wildcard := "-*"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
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
			// Update the Tag Protection
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
					project = "%d"
					tag = "TagProtect-%d%s"
					create_access_level = "maintainer"
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
			// Update the Tag Protection to get back to initial settings
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
		},
	})
}

func TestAccGitlabTagProtection_customAccessLevel(t *testing.T) {
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

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabTagProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Tag Protection with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = "%d"
				  tag = "TagProtect-%d"
				  create_access_level = "developer"

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
						CreateAccessLevel:     api.AccessLevelValueToName[gitlab.DeveloperPermissions],
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
			// Update the Tag Protection and set "create_access_level" to "maintainer"
			{
				Config: fmt.Sprintf(`
				resource "gitlab_tag_protection" "TagProtect" {
				  project = "%d"
				  tag = "TagProtect-%d"

				  # Update to maintainer permission
				  create_access_level = "maintainer"

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
						CreateAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
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

				  # Update to maintainer permission
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
		ProtoV6ProviderFactories: providerFactoriesV6,
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

				ExpectError: regexp.MustCompile("feature unavailable: `allowed_to_create`, Premium or Ultimate license required."),
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

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
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
				ExpectError: regexp.MustCompile("both user_id and group_id cannot be present in the same allowed_to_create"),
			},
		},
	})
}

func testAccCheckGitlabTagProtectionExists(n string, pt *gitlab.ProtectedTag) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}
		project, tag, err := projectAndTagFromID(rs.Primary.ID)
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
	Name                  string
	CreateAccessLevel     string
	UsersAllowedToCreate  []string
	GroupsAllowedToCreate []string
}

func testAccCheckGitlabTagProtectionAttributes(pt *gitlab.ProtectedTag, want *testAccGitlabTagProtectionExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if pt.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", pt.Name, want.Name)
		}

		if pt.CreateAccessLevels[0].AccessLevel != api.AccessLevelNameToValue[want.CreateAccessLevel] {
			return fmt.Errorf("got Create access levels %q; want %q", pt.CreateAccessLevels[0].AccessLevel, api.AccessLevelNameToValue[want.CreateAccessLevel])
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
