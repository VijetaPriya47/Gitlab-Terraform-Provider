//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabMemberRole_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	rint := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabMemberRole_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a member role with only required attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, description, and permissions of member role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d updated"
						description = "A test member role updated"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY", "REMOVE_PROJECT", "ADMIN_PROTECTED_BRANCH"]
					}
				`, rint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// updating permissions to not be in alphabetical order
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d updated"
						description = "A test member role updated"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY", "REMOVE_PROJECT", "ADMIN_CICD_VARIABLES"]
					}
				`, rint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
					resource.TestCheckResourceAttr("gitlab_member_role.foo", "enabled_permissions.#", "3"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabMemberRole_UpdateWithMemberAssigned(t *testing.T) {
	testutil.SkipIfCE(t)

	rint := acctest.RandInt()
	testGroup := testutil.CreateGroups(t, 1)[0]
	user := testutil.CreateUsers(t, 1)[0]
	var memberRoleIid int

	memberExistsErrRegex, err := regexp.Compile("Role is assigned to one or more group members")
	if err != nil {
		t.Errorf("Unable to format member exists error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a member role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),

				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
					resource.TestCheckResourceAttrWith("gitlab_member_role.foo", "iid", func(value string) error {
						// Capture the member role iid that we have in state
						memberRoleIid, _ = strconv.Atoi(value)
						return nil
					}),
				),
			},
			// Add a test user to a test group with the new custom member role
			// Verify the member role can be updated as long as replacement is not required
			{
				PreConfig: func() {
					if _, _, err := testutil.TestGitlabClient.GroupMembers.AddGroupMember(testGroup.ID, &gitlab.AddGroupMemberOptions{
						UserID:       gitlab.Ptr(user.ID),
						AccessLevel:  gitlab.Ptr(gitlab.ReporterPermissions),
						MemberRoleID: gitlab.Ptr(memberRoleIid),
					}); err != nil {
						t.Errorf("Error adding group member to test group: %s", err)
					}
				},
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY", "REMOVE_PROJECT"]
					}
				`, rint),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Verify the member role cannot be removed when the member role is assigned
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY", "REMOVE_PROJECT"]
					}
				`, rint),
				Destroy:     true,
				ExpectError: memberExistsErrRegex,
			},
			// Remove user assigned to member role and verify the role can now be removed
			{
				PreConfig: func() {
					if _, err := testutil.TestGitlabClient.GroupMembers.RemoveGroupMember(testGroup.ID, user.ID, nil, nil); err != nil {
						t.Errorf("Error removing group member from test group: %s", err)
					}

				},
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY", "REMOVE_PROJECT"]
					}
				`, rint),
				Destroy: true,
			},
		},
	})
}
func TestAccGitlabMemberRole_EnsureReplacement(t *testing.T) {
	testutil.SkipIfCE(t)

	rint := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabMemberRole_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a member role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update base_access_level forces replacement
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_member_role.foo", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabMemberRole_EnsureErrorOnInvalidAttribute(t *testing.T) {
	testutil.SkipIfCE(t)

	notPermittedRegex, err := regexp.Compile("Attribute Not Permitted")
	if err != nil {
		t.Errorf("Unable to format attribute not permitted error regex: %s", err)
	}

	invalidAttributeValueRegex, err := regexp.Compile("Invalid Attribute Value Match")
	if err != nil {
		t.Errorf("Unable to format invalid attribute value error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			// Group path not permitted for self-managed
			{
				Config: `
					resource "gitlab_member_role" "foo" {
						name = "Test role"
						base_access_level = "REPORTER"
						enabled_permissions = ["READ_VULNERABILITY"]
						group_path = "test-group"
					}
				`,
				ExpectError: notPermittedRegex,
			},
			// Invalid base_access_level value
			{
				Config: `
					resource "gitlab_member_role" "foo" {
						name = "Test role"
						base_access_level = "developer"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`,
				ExpectError: invalidAttributeValueRegex,
			},
			// Invalid enabled_permission value
			{
				Config: `
					resource "gitlab_member_role" "foo" {
						name = "Test role"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["INVALID_PERMISSION"]
					}
				`,
				ExpectError: invalidAttributeValueRegex,
			},
		},
	})
}

func TestAccGitlabMemberRole_EnsureErrorOnInvalidPermission(t *testing.T) {
	testutil.SkipIfCE(t)

	rint := acctest.RandInt()
	testGroup := testutil.CreateGroups(t, 1)[0]

	// Set up user, role mapping and personal access token.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, testGroup.ID, []*gitlab.User{user}, gitlab.DeveloperPermissions)
	developerUserPAT := testutil.CreatePersonalAccessTokenWithScopes(t, user, []string{"api"})

	permissionErrRegex, err := regexp.Compile("you don't have permission to perform this action")
	if err != nil {
		t.Errorf("Unable to format expected permission error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabMemberRole_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a member_role; should fail with permission error
			{
				// lintignore:AT004  // we need the provider configuration here to attempt to create the member_role as a different user
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, developerUserPAT.Token, rint),
				ExpectError: permissionErrRegex,
			},
			// Successfully create a member role, to be used to check update error
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "id"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "iid"),
					resource.TestCheckResourceAttrSet("gitlab_member_role.foo", "edit_path"),
				),
			},
			{
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a member role; should fail with permission error
			{
				// lintignore:AT004  // we need the provider configuration here to attempt to update the member_role as a different user
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_member_role" "foo" {
						name = "Test role %d updated"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, developerUserPAT.Token, rint),
				ExpectError: permissionErrRegex,
			},
			// Clean up member_role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_member_role" "foo" {
						name = "Test role %d"
						base_access_level = "DEVELOPER"
						enabled_permissions = ["READ_VULNERABILITY"]
					}
				`, rint),
				Destroy: true,
			},
		},
	})
}

// This tests ensures that the changes from types.List -> types.Set in the 17.6 release
// doesn't cause and problems. It can likely be removed after the release is completed.
// to save on CI time.
func TestAccGitlabMemberRole_convertListToSet(t *testing.T) {
	testutil.SkipIfCE(t)

	rint := acctest.RandInt()
	config := fmt.Sprintf(`
		resource "gitlab_member_role" "foo" {
			name = "Test role %d"
			base_access_level = "REPORTER"
			enabled_permissions = ["READ_VULNERABILITY"]
		}
	`, rint)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAcc_GitlabMemberRole_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a member role with only required attributes
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 17.5",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 17.5",
						Source:            "gitlabhq/gitlab",
					},
				},
				ResourceName:      "gitlab_member_role.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, description, and permissions of member role
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_member_role.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAcc_GitlabMemberRole_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_member_role" {

			query := api.GraphQLQuery{
				Query: fmt.Sprintf(`
					query {
						memberRole(id: "%s") {
							id
						}
					}`, rs.Primary.ID),
			}

			var response MemberRoleResponse
			if _, err := api.SendGraphQLRequest(context.Background(), testutil.TestGitlabClient, query, &response); err != nil {
				return err
			}

			// member role is destroyed if memberRole is null
			if response.Data.MemberRole.ID != "" {
				return fmt.Errorf("Member Role: %s still exists", rs.Primary.ID)
			}

			return nil
		}
	}
	return nil
}
