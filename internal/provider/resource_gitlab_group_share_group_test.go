//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	// "os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupShareGroup_basic(t *testing.T) {
	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	resource.ParallelTest(t, resource.TestCase{
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
	// Group level custom roles don't work on self managed, so we can't test them without a SaaS project.
	// See https://gitlab.com/gitlab-org/gitlab/-/issues/439284 for more details
	t.Skip()

	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "17.9")

	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	// Create a custom role on that group - we don't need to clean this up, since it's bound to the group
	// which will be deleted when the test finishes.
	customRole, _, err := testutil.TestGitlabClient.MemberRolesService.CreateMemberRole(mainGroup.ID, &gitlab.CreateMemberRoleOptions{
		Name:              gitlab.Ptr("test-role"),
		BaseAccessLevel:   gitlab.Ptr(gitlab.MaintainerPermissions),
		ReadVulnerability: gitlab.Ptr(true),
	})
	if err != nil {
		t.Fatalf("Failed to create test custom role. Error: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
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
					resource.TestCheckResourceAttr("gitlab_group_share_group.test", "group_access", "guest"),
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
		},
	})
}

func TestAccGitlabGroupShareGroup_migrateFromSDKToFramework(t *testing.T) {
	groups := testutil.CreateGroups(t, 2)
	mainGroup := groups[0]
	sharedGroup := groups[1]

	resource.ParallelTest(t, resource.TestCase{
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
