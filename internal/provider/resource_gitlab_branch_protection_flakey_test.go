//go:build flakey
// +build flakey

package provider

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabBranchProtection_allowSpecificUserAndNoRoleToPush(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)
	// Set up the groups to share the `testProject` with
	testUsers := testutil.CreateUsers(t, 2)

	// Add users as members to project
	testutil.AddProjectMembers(t, testProject.ID, testUsers)

	//list existing project members
	testutil.ListProjectMembers(t, testProject.ID)

	// add a sleep to determine if there is a race condition in group membership for protected
	// branches
	t.Log("Sleeping for 10s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(10 * time.Second)

	var pb gitlab.ProtectedBranch

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroyFlakey,
		Steps: []resource.TestStep{
			// Create a branch protection, with only user and no role allowed to push
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							user_id = %[2]d
						}
					}
				`, testProject.ID, testUsers[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                 "test-branch",
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToPush:   []string{testUsers[0].Username},
					}),
				),
			},
			{
				ResourceName:      "gitlab_branch_protection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a branch protection, with only user and no role allowed to push
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							user_id = %[2]d
						}

						allowed_to_push {
							user_id = %[3]d
						}
					}
				`, testProject.ID, testUsers[0].ID, testUsers[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                 "test-branch",
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToPush:   []string{testUsers[0].Username, testUsers[1].Username},
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_allowSpecificDeployKeyToPush(t *testing.T) {
	testutil.SkipIfCE(t)

	// deploy_key_id is only available for configuration since 17.5
	isFeatureSupported, err := api.IsGitLabVersionAtLeast(context.Background(), testutil.TestGitlabClient, "17.5")()
	if err != nil {
		t.Fatal("Failed to read GitLab version")
	}
	if !isFeatureSupported {
		t.Skipf("Feature not supported yet")
	}

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)

	// Create a deploy-key for `testProject`
	testDeployKey := testutil.CreateDeployKey(t, testProject.ID, &gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr("The Key"),
		Key:     gitlab.Ptr("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDlh9s4DjxXWFkTCvuPNQnGboQecs6Sauw4qH/0DqKo1b1EVPNL6Eww7C9h4PnVfc4EWRM6tcXEHtd7913BnM7GTAt717UCxCKO26LG6qi3PD4tR29/WZ5W/SbguRYXDU+qo5LJ2O6goZ97uA///ms/LApJqvdd905E6sPPKrzYmQ== test"),
		CanPush: gitlab.Ptr(true),
	})
	testDeployKey2 := testutil.CreateDeployKey(t, testProject.ID, &gitlab.AddDeployKeyOptions{
		Title:   gitlab.Ptr("Another Key"),
		Key:     gitlab.Ptr("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCScirY0/CpqAKXM/kkjrYHqvuardOecpzZCuyLH5lxQOxb4fzxM7tAaFod1Y9rNoc0hBaDNfB8t8SCJmKDthxF2OlTVUXIIBc/ltiZwLAQUnJV/Bz9v17JxGSDa6NQBGDlHNbhCixG9tCZt0rManaiHuq2WPcNIfWa3xiOCNefkw== test"),
		CanPush: gitlab.Ptr(true),
	})

	var pb gitlab.ProtectedBranch

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroyFlakey,
		Steps: []resource.TestStep{
			// Create a branch protection, with only a deploy-key allowed to push
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							deploy_key_id = %[2]d
						}
					}
				`, testProject.ID, testDeployKey.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                    "test-branch",
						PushAccessLevel:         api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:        api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:    api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						DeployKeysAllowedToPush: []string{testDeployKey.Title},
					}),
				),
			},
			{
				ResourceName:      "gitlab_branch_protection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a branch protection, with another deploy-key allowed to push
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							deploy_key_id = %[2]d
						}
					}
				`, testProject.ID, testDeployKey2.ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                    "test-branch",
						PushAccessLevel:         api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:        api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:    api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						DeployKeysAllowedToPush: []string{testDeployKey2.Title},
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithMultipleAccessLevels(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)
	// Set up the groups to share the `testProject` with
	testGroups := testutil.CreateGroups(t, 2)
	// Set up the users to add as members to the `testProject`
	testUsers := testutil.CreateUsers(t, 2)
	// Add users as members to project
	testutil.AddProjectMembers(t, testProject.ID, testUsers)
	// Add users to groups
	testutil.AddGroupMembers(t, testGroups[0].ID, []*gitlab.User{testUsers[0]})
	testutil.AddGroupMembers(t, testGroups[1].ID, []*gitlab.User{testUsers[1]})
	// Share project with groups
	testutil.ProjectShareGroup(t, testProject.ID, testGroups[0].ID)
	testutil.ProjectShareGroup(t, testProject.ID, testGroups[1].ID)

	var pb gitlab.ProtectedBranch

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroyFlakey,
		Steps: []resource.TestStep{
			// Create a project, groups, users and Branch Protection with advanced allowed_to blocks
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"

						allowed_to_push {
							user_id = %[3]d
						}
						
						allowed_to_push {
							group_id = %[4]d
						}

						allowed_to_push {
							group_id = %[5]d
						}


						allowed_to_merge { 
							user_id = %[2]d
						}
						
						allowed_to_merge { 
							group_id = %[4]d
						}
						
						allowed_to_merge { 
							user_id = %[3]d
						}
						
						allowed_to_merge { 
							group_id = %[5]d
						}


						allowed_to_unprotect {
							user_id = %[2]d
						}
						
						allowed_to_unprotect {
								group_id = %[4]d
						}

						allowed_to_unprotect {
							user_id = %[3]d
						}
						
						allowed_to_unprotect {
							group_id = %[5]d
						}
					}
				`, testProject.ID, testUsers[0].ID, testUsers[1].ID, testGroups[0].ID, testGroups[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                     "test-branch",
						PushAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:         api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToPush:       []string{testUsers[1].Username},
						UsersAllowedToMerge:      []string{testUsers[0].Username, testUsers[1].Username},
						UsersAllowedToUnprotect:  []string{testUsers[0].Username, testUsers[1].Username},
						GroupsAllowedToPush:      []string{testGroups[0].Name, testGroups[1].Name},
						GroupsAllowedToMerge:     []string{testGroups[0].Name, testGroups[1].Name},
						GroupsAllowedToUnprotect: []string{testGroups[0].Name, testGroups[1].Name},
					}),
				),
			},
			// Update to remove some allowed_to blocks and update access levels
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "developer"
						merge_access_level     = "developer"
						unprotect_access_level = "developer"

						allowed_to_push {
							user_id = %[3]d
						}
						
						allowed_to_push {
							group_id = %[4]d
						}


						allowed_to_merge {
							user_id = %[2]d
						}
						
						allowed_to_merge {
							group_id = %[4]d
						}


						allowed_to_unprotect {
							user_id = %[2]d
						}
						
						allowed_to_unprotect {
							group_id = %[5]d
						}
					}
				`, testProject.ID, testUsers[0].ID, testUsers[1].ID, testGroups[0].ID, testGroups[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                     "test-branch",
						PushAccessLevel:          api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel:         api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						UnprotectAccessLevel:     api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						UsersAllowedToPush:       []string{testUsers[1].Username},
						UsersAllowedToMerge:      []string{testUsers[0].Username},
						UsersAllowedToUnprotect:  []string{testUsers[0].Username},
						GroupsAllowedToPush:      []string{testGroups[0].Name},
						GroupsAllowedToMerge:     []string{testGroups[0].Name},
						GroupsAllowedToUnprotect: []string{testGroups[1].Name},
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_removeUsersAndGroupsFromAllowedTo(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)
	// Set up the groups to share the `testProject` with
	testGroups := testutil.CreateGroups(t, 2)
	// Set up the users to add as members to the `testProject`
	testUsers := testutil.CreateUsers(t, 2)
	// Add users as members to project
	testutil.AddProjectMembers(t, testProject.ID, testUsers)

	// Add users to groups
	testutil.AddGroupMembers(t, testGroups[0].ID, []*gitlab.User{testUsers[0]})
	testutil.AddGroupMembers(t, testGroups[1].ID, []*gitlab.User{testUsers[1]})

	// Share project with groups
	testutil.ProjectShareGroup(t, testProject.ID, testGroups[0].ID)
	testutil.ProjectShareGroup(t, testProject.ID, testGroups[1].ID)

	testutil.ListProjectMembers(t, testProject.ID)

	t.Log("Sleeping for 10s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(10 * time.Second)

	var pb gitlab.ProtectedBranch

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroyFlakey,
		Steps: []resource.TestStep{
			// Create a branch protection, with only user and no role allowed to push
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							user_id = %[2]d
						}
						
						allowed_to_push {
							user_id = %[3]d
						}

						allowed_to_push {
							group_id = %[4]d
						}
						
						allowed_to_push {
							group_id = %[5]d
						}


						allowed_to_merge {
							user_id = %[2]d
						}
					
						allowed_to_merge {
							user_id = %[3]d
						}
						
						allowed_to_merge {
							group_id = %[4]d
						}
						
						allowed_to_merge {
							group_id = %[5]d
						}


						allowed_to_unprotect {
							user_id = %[2]d
						}
						
						allowed_to_unprotect {
							user_id = %[3]d
						}
						
						allowed_to_unprotect {
							group_id = %[4]d
						}
						
						allowed_to_unprotect {
							group_id = %[5]d
						}
					}
				`, testProject.ID, testUsers[0].ID, testUsers[1].ID, testGroups[0].ID, testGroups[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                     "test-branch",
						PushAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:         api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToPush:       []string{testUsers[0].Username, testUsers[1].Username},
						UsersAllowedToMerge:      []string{testUsers[0].Username, testUsers[1].Username},
						UsersAllowedToUnprotect:  []string{testUsers[0].Username, testUsers[1].Username},
						GroupsAllowedToPush:      []string{testGroups[0].Name, testGroups[1].Name},
						GroupsAllowedToMerge:     []string{testGroups[0].Name, testGroups[1].Name},
						GroupsAllowedToUnprotect: []string{testGroups[0].Name, testGroups[1].Name},
					}),
				),
			},
			{
				ResourceName:      "gitlab_branch_protection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a branch protection, with removed user and group allowed to specific action
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "test" {
						project                = %d
						branch                 = "test-branch"
						push_access_level      = "maintainer"
						merge_access_level     = "maintainer"
						unprotect_access_level = "maintainer"


						allowed_to_push {
							user_id = %[2]d
						}

						allowed_to_push {
							group_id = %[3]d
						}


						allowed_to_merge {
							user_id = %[2]d
						}
						
						allowed_to_merge {
							group_id = %[3]d
						}


						allowed_to_unprotect {
							user_id = %[2]d
						}
						
						allowed_to_unprotect {
							group_id = %[3]d
						}
					}
				`, testProject.ID, testUsers[0].ID, testGroups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExistsFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributesFlakey("gitlab_branch_protection.test", &pb, &testAccGitlabBranchProtectionExpectedAttributesFlakey{
						Name:                     "test-branch",
						PushAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:         api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UsersAllowedToPush:       []string{testUsers[0].Username},
						UsersAllowedToMerge:      []string{testUsers[0].Username},
						UsersAllowedToUnprotect:  []string{testUsers[0].Username},
						GroupsAllowedToPush:      []string{testGroups[0].Name},
						GroupsAllowedToMerge:     []string{testGroups[0].Name},
						GroupsAllowedToUnprotect: []string{testGroups[0].Name},
					}),
				),
			},
		},
	})
}

func testAccCheckGitlabBranchProtectionPersistsInStateCorrectlyFlakey(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		var mergeAccessLevel gitlab.AccessLevelValue
		for _, v := range pb.MergeAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				mergeAccessLevel = v.AccessLevel
				break
			}
		}
		if rs.Primary.Attributes["merge_access_level"] != api.AccessLevelValueToName[mergeAccessLevel] {
			return fmt.Errorf("merge access level not persisted in state correctly")
		}

		var pushAccessLevel gitlab.AccessLevelValue
		for _, v := range pb.PushAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				pushAccessLevel = v.AccessLevel
				break
			}
		}
		if rs.Primary.Attributes["push_access_level"] != api.AccessLevelValueToName[pushAccessLevel] {
			return fmt.Errorf("push access level not persisted in state correctly")
		}

		if unprotectAccessLevel, err := firstValidAccessLevel(pb.UnprotectAccessLevels); err == nil {
			if rs.Primary.Attributes["unprotect_access_level"] != api.AccessLevelValueToName[*unprotectAccessLevel] {
				return fmt.Errorf("unprotect access level not persisted in state correctly")
			}
		}

		if rs.Primary.Attributes["allow_force_push"] != strconv.FormatBool(pb.AllowForcePush) {
			return fmt.Errorf("allow_force_push not persisted in state correctly")
		}

		if rs.Primary.Attributes["code_owner_approval_required"] != strconv.FormatBool(pb.CodeOwnerApprovalRequired) {
			return fmt.Errorf("code_owner_approval_required not persisted in state correctly")
		}

		return nil
	}
}

func testAccCheckGitlabBranchProtectionExistsFlakey(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}
		project, branch, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Error in Splitting Project and Branch Ids")
		}

		pbs, _, err := testutil.TestGitlabClient.ProtectedBranches.ListProtectedBranches(project, nil)
		if err != nil {
			return err
		}
		for _, gotpb := range pbs {
			if gotpb.Name == branch {
				*pb = *gotpb
				return nil
			}
		}
		return fmt.Errorf("Protected Branch does not exist")
	}
}

func testAccCheckGitlabBranchProtectionComputedAttributesFlakey(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return resource.TestCheckResourceAttr(n, "branch_protection_id", strconv.Itoa(pb.ID))(s)
	}
}

type testAccGitlabBranchProtectionExpectedAttributesFlakey struct {
	Name                      string
	PushAccessLevel           string
	MergeAccessLevel          string
	UnprotectAccessLevel      string
	AllowForcePush            bool
	UsersAllowedToPush        []string
	UsersAllowedToMerge       []string
	UsersAllowedToUnprotect   []string
	GroupsAllowedToPush       []string
	GroupsAllowedToMerge      []string
	GroupsAllowedToUnprotect  []string
	DeployKeysAllowedToPush   []string
	CodeOwnerApprovalRequired bool
}

func testAccCheckGitlabBranchProtectionAttributesFlakey(n string, pb *gitlab.ProtectedBranch, want *testAccGitlabBranchProtectionExpectedAttributesFlakey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		project, _, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error in splitting Project and Branch Ids")
		}

		if pb.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", pb.Name, want.Name)
		}

		var pushAccessLevel gitlab.AccessLevelValue
		for _, v := range pb.PushAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				pushAccessLevel = v.AccessLevel
				break
			}
		}
		if pushAccessLevel != api.AccessLevelNameToValue[want.PushAccessLevel] {
			return fmt.Errorf("got push access level %v; want %v", pushAccessLevel, api.AccessLevelNameToValue[want.PushAccessLevel])
		}

		var mergeAccessLevel gitlab.AccessLevelValue
		for _, v := range pb.MergeAccessLevels {
			if v.UserID == 0 && v.GroupID == 0 {
				mergeAccessLevel = v.AccessLevel
				break
			}
		}
		if mergeAccessLevel != api.AccessLevelNameToValue[want.MergeAccessLevel] {
			return fmt.Errorf("got merge access level %v; want %v", mergeAccessLevel, api.AccessLevelNameToValue[want.MergeAccessLevel])
		}

		// unprotect access level will be nil in CE as it is not returned on the response, but in EE it is returned
		if pb.UnprotectAccessLevels != nil {
			var unprotectAccessLevel gitlab.AccessLevelValue
			for _, v := range pb.UnprotectAccessLevels {
				if v.UserID == 0 && v.GroupID == 0 {
					unprotectAccessLevel = v.AccessLevel
					break
				}
			}
			if unprotectAccessLevel != api.AccessLevelNameToValue[want.UnprotectAccessLevel] {
				return fmt.Errorf("got unprotect access level %v; want %v", unprotectAccessLevel, api.AccessLevelNameToValue[want.UnprotectAccessLevel])
			}
		}

		if pb.AllowForcePush != want.AllowForcePush {
			return fmt.Errorf("got allow_force_push %v; want %v", pb.AllowForcePush, want.AllowForcePush)
		}

		remainingWantedUserIDsAllowedToPush := map[int]struct{}{}
		for _, v := range want.UsersAllowedToPush {
			users, _, err := testutil.TestGitlabClient.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: gitlab.Ptr(v),
			})
			if err != nil {
				return fmt.Errorf("error looking up user by path %v: %v", v, err)
			}
			if len(users) != 1 {
				return fmt.Errorf("error finding user by username %v; found %v", v, len(users))
			}
			remainingWantedUserIDsAllowedToPush[users[0].ID] = struct{}{}
		}
		remainingWantedGroupIDsAllowedToPush := map[int]struct{}{}
		for _, v := range want.GroupsAllowedToPush {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToPush[group.ID] = struct{}{}
		}
		remainingWantedDeployKeyIDsAllowedToPush := map[int]struct{}{}
		for _, v := range want.DeployKeysAllowedToPush {
			deployKeys, _, err := testutil.TestGitlabClient.DeployKeys.ListProjectDeployKeys(project, &gitlab.ListProjectDeployKeysOptions{})
			if err != nil {
				return fmt.Errorf("error looking up deploy key for project %v: %v", project, err)
			}
			filteredDeployKeys := slices.DeleteFunc(deployKeys, func(key *gitlab.ProjectDeployKey) bool { return key.Title != v })
			if len(filteredDeployKeys) != 1 {
				return fmt.Errorf("error finding deploy key by name %v; found %v", v, len(filteredDeployKeys))
			}
			remainingWantedDeployKeyIDsAllowedToPush[filteredDeployKeys[0].ID] = struct{}{}
		}
		for _, v := range pb.PushAccessLevels {
			if v.UserID != 0 {
				if _, ok := remainingWantedUserIDsAllowedToPush[v.UserID]; !ok {
					return fmt.Errorf("found unwanted user ID %v", v.UserID)
				}
				delete(remainingWantedUserIDsAllowedToPush, v.UserID)
			} else if v.GroupID != 0 {
				if _, ok := remainingWantedGroupIDsAllowedToPush[v.GroupID]; !ok {
					return fmt.Errorf("found unwanted group ID %v", v.GroupID)
				}
				delete(remainingWantedGroupIDsAllowedToPush, v.GroupID)
			} else if v.DeployKeyID != 0 {
				if _, ok := remainingWantedDeployKeyIDsAllowedToPush[v.DeployKeyID]; !ok {
					return fmt.Errorf("found unwanted deploy key ID %v", v.DeployKeyID)
				}
				delete(remainingWantedDeployKeyIDsAllowedToPush, v.DeployKeyID)
			}
		}
		if len(remainingWantedUserIDsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToPush)
		}
		if len(remainingWantedGroupIDsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToPush)
		}
		if len(remainingWantedDeployKeyIDsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted deploy key IDs %v", remainingWantedDeployKeyIDsAllowedToPush)
		}

		remainingWantedUserIDsAllowedToMerge := map[int]struct{}{}
		for _, v := range want.UsersAllowedToMerge {
			users, _, err := testutil.TestGitlabClient.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: gitlab.Ptr(v),
			})
			if err != nil {
				return fmt.Errorf("error looking up user by path %v: %v", v, err)
			}
			if len(users) != 1 {
				return fmt.Errorf("error finding user by username %v; found %v", v, len(users))
			}
			remainingWantedUserIDsAllowedToMerge[users[0].ID] = struct{}{}
		}
		remainingWantedGroupIDsAllowedToMerge := map[int]struct{}{}
		for _, v := range want.GroupsAllowedToMerge {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToMerge[group.ID] = struct{}{}
		}
		for _, v := range pb.MergeAccessLevels {
			if v.UserID != 0 {
				if _, ok := remainingWantedUserIDsAllowedToMerge[v.UserID]; !ok {
					return fmt.Errorf("found unwanted user ID %v", v.UserID)
				}
				delete(remainingWantedUserIDsAllowedToMerge, v.UserID)
			} else if v.GroupID != 0 {
				if _, ok := remainingWantedGroupIDsAllowedToMerge[v.GroupID]; !ok {
					return fmt.Errorf("found unwanted group ID %v", v.GroupID)
				}
				delete(remainingWantedGroupIDsAllowedToMerge, v.GroupID)
			}
		}
		if len(remainingWantedUserIDsAllowedToMerge) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToMerge)
		}
		if len(remainingWantedGroupIDsAllowedToMerge) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToMerge)
		}

		remainingWantedUserIDsAllowedToUnprotect := map[int]struct{}{}
		for _, v := range want.UsersAllowedToUnprotect {
			users, _, err := testutil.TestGitlabClient.Users.ListUsers(&gitlab.ListUsersOptions{
				Username: gitlab.Ptr(v),
			})
			if err != nil {
				return fmt.Errorf("error looking up user by path %v: %v", v, err)
			}
			if len(users) != 1 {
				return fmt.Errorf("error finding user by username %v; found %v", v, len(users))
			}
			remainingWantedUserIDsAllowedToUnprotect[users[0].ID] = struct{}{}
		}
		remainingWantedGroupIDsAllowedToUnprotect := map[int]struct{}{}
		for _, v := range want.GroupsAllowedToUnprotect {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToUnprotect[group.ID] = struct{}{}
		}
		for _, v := range pb.UnprotectAccessLevels {
			if v.UserID != 0 {
				if _, ok := remainingWantedUserIDsAllowedToUnprotect[v.UserID]; !ok {
					return fmt.Errorf("found unwanted user ID %v", v.UserID)
				}
				delete(remainingWantedUserIDsAllowedToUnprotect, v.UserID)
			} else if v.GroupID != 0 {
				if _, ok := remainingWantedGroupIDsAllowedToUnprotect[v.GroupID]; !ok {
					return fmt.Errorf("found unwanted group ID %v", v.GroupID)
				}
				delete(remainingWantedGroupIDsAllowedToUnprotect, v.GroupID)
			}
		}
		if len(remainingWantedUserIDsAllowedToUnprotect) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToUnprotect)
		}
		if len(remainingWantedGroupIDsAllowedToUnprotect) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToUnprotect)
		}

		if pb.CodeOwnerApprovalRequired != want.CodeOwnerApprovalRequired {
			return fmt.Errorf("got code_owner_approval_required %v; want %v", pb.CodeOwnerApprovalRequired, want.CodeOwnerApprovalRequired)
		}

		return nil
	}
}

func testAccCheckGitlabBranchProtectionDestroyFlakey(s *terraform.State) error {
	var project string
	var branch string
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project" {
			project = rs.Primary.ID
		} else if rs.Type == "gitlab_branch_protection" {
			branch = rs.Primary.ID
		}
	}

	pb, _, err := testutil.TestGitlabClient.ProtectedBranches.GetProtectedBranch(project, branch)
	if err == nil {
		if pb != nil {
			return fmt.Errorf("project branch protection %s still exists", branch)
		}
	}
	if !api.Is404(err) {
		return err
	}
	return nil
}
