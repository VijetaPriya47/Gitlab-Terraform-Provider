//go:build acceptance

package provider

import (
	"fmt"

	// "os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupShareGroup_basic(t *testing.T) {
	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupShareGroupDestroy,
		Steps: []resource.TestStep{
			// Share a new group with another group
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "guest"
						expires_at     = "2099-01-01"
					}
				`, mainGroup.ID, sharedGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "id", fmt.Sprintf("%d:%d", mainGroup.ID, sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_id", fmt.Sprintf("%d", mainGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "share_group_id", fmt.Sprintf("%d", sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_access", "guest"),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "expires_at", "2099-01-01"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_share_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the share group
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "reporter"
					}
				`, mainGroup.ID, sharedGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "id", fmt.Sprintf("%d:%d", mainGroup.ID, sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_id", fmt.Sprintf("%d", mainGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "share_group_id", fmt.Sprintf("%d", sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_access", "reporter"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_share_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroupShareGroup_customRoles(t *testing.T) {
	testutil.SkipIfCE(t)

	rInt := acctest.RandInt()
	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	customRole, _, err := testutil.TestGitlabClient.MemberRolesService.CreateInstanceMemberRole(&gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr(fmt.Sprintf("test-role-%d", rInt)),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})
	if err != nil {
		t.Fatalf("Failed to create test custom role. Error: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupShareGroupDestroy,
		Steps: []resource.TestStep{
			// Share a new group with another group
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "maintainer"
						expires_at     = "2099-01-01"
						member_role_id = %d
					}
				`, mainGroup.ID, sharedGroup.ID, customRole.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "id", fmt.Sprintf("%d:%d", mainGroup.ID, sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_id", fmt.Sprintf("%d", mainGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "share_group_id", fmt.Sprintf("%d", sharedGroup.ID)),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_access", "maintainer"),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "expires_at", "2099-01-01"),
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "member_role_id", fmt.Sprintf("%d", customRole.ID)),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_share_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove the custom role from the group share and revert to a base role
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "maintainer"
						expires_at     = "2099-01-01"
					}
				`, mainGroup.ID, sharedGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					// read the group share from API and ensure that `member_role_id` is set to null or 0
					func(s *terraform.State) error {
						// Get the resource
						rs, ok := s.RootModule().Resources["gitlab_group_share_group.test"]
						if !ok {
							return fmt.Errorf("gitlab_group_share_group.test not found")
						}

						groupId, sharedGroupId, err := groupIdsFromId(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("[ERROR] cannot get Group ID and ShareGroupId from input: %v", rs.Primary.ID)
						}

						// Get the group from GitLab API
						group, _, err := testutil.TestGitlabClient.Groups.GetGroup(groupId, nil)
						if err != nil {
							return err
						}

						// Find the shared group in the SharedWithGroups list
						for _, sharedGroup := range group.SharedWithGroups {
							if sharedGroupId == sharedGroup.GroupID {
								// Verify that member_role_id is 0 (null in API response)
								if sharedGroup.MemberRoleID != 0 {
									return fmt.Errorf("Expected member_role_id to be 0, but got %d", sharedGroup.MemberRoleID)
								}
								return nil
							}
						}

						return fmt.Errorf("Could not find shared group %d in group %s", sharedGroupId, groupId)
					},
				),
			},
		},
	})
}

func TestAccGitlabGroupShareGroup_migrateFromSDKToFramework(t *testing.T) {
	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	resource.Test(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupShareGroupDestroy,
		Steps: []resource.TestStep{
			// Create the badge in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "guest"
						expires_at     = "2099-01-01"
					}
				`, mainGroup.ID, sharedGroup.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_share_group.test", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_group_share_group" "test" {
						group_id       = %d
						share_group_id = %d
						group_access   = "guest"
						expires_at     = "2099-01-01"
					}
				`, mainGroup.ID, sharedGroup.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_group_share_group.test", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_group_share_group.test",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAccCheckGitlabGroupShareGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_share_group" {
			continue
		}

		groupId, sharedGroupId, err := groupIdsFromId(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("[ERROR] cannot get Group ID and ShareGroupId from input: %v", rs.Primary.ID)
		}

		// Get Main Group
		group, _, err := testutil.TestGitlabClient.Groups.GetGroup(groupId, nil)
		if err != nil {
			return err
		}

		// Make sure that SharedWithGroups attribute on the main group does not contain the shared group id at all
		for _, sharedGroup := range group.SharedWithGroups {
			if sharedGroupId == sharedGroup.GroupID {
				return fmt.Errorf("GitLab Group Share %d still exists", sharedGroupId)
			}
		}
		return nil
	}

	return nil
}
