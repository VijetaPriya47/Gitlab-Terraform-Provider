//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabBranchProtection_basic(t *testing.T) {

	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Branch Protection with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Configure the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project                = gitlab_project.foo.id
				  branch                 = "BranchProtect-%[1]d"
				  push_access_level      = "developer"
				  merge_access_level     = "developer"
				  unprotect_access_level = "developer"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
					}),
				),
			},
			// Update the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project                = gitlab_project.foo.id
				  branch                 = "BranchProtect-%[1]d"
				  push_access_level      = "maintainer"
				  merge_access_level     = "maintainer"
				  unprotect_access_level = "maintainer"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Branch Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Branch Protection with allow force push enabled
			{
				Config: testAccGitlabBranchProtectionUpdateConfigAllowForcePushTrue(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						AllowForcePush:       true,
					}),
				),
			},
			// Update the Branch Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Branch Protection code owner approval setting
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   testAccGitlabBranchProtectionUpdateConfigCodeOwnerTrue(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                      fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:           api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						CodeOwnerApprovalRequired: true,
					}),
				),
			},
			// Update the Branch Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithCodeOwnerApproval(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Start with code owner approval required disabled
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Create a project and Branch Protection with code owner approval enabled
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   testAccGitlabBranchProtectionUpdateConfigCodeOwnerTrue(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                      fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:           api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						CodeOwnerApprovalRequired: true,
					}),
				),
			},
			// Attempting to update code owner approval setting on CE should fail safely and with an informative error message
			{
				SkipFunc:    testutil.IsRunningInEE,
				Config:      testAccGitlabBranchProtectionUpdateConfigCodeOwnerTrue(rInt),
				ExpectError: regexp.MustCompile("feature unavailable: `code_owner_approval_required`"),
			},
			// Update the Branch Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithAllowForcePush(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Start with allow force push disabled
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Create a project and Branch Protection with allow force push enabled
			{
				Config: testAccGitlabBranchProtectionUpdateConfigAllowForcePushTrue(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						AllowForcePush:       true,
					}),
				),
			},
			// Update the Branch Protection to get back to initial settings
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project            = gitlab_project.foo.id
				  branch             = "BranchProtect-%[1]d"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithUnprotectAccessLevel(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Configure the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project                = gitlab_project.foo.id
				  branch                 = "BranchProtect-%[1]d"
				  push_access_level      = "developer"
				  merge_access_level     = "developer"
				  unprotect_access_level = "maintainer"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project                = gitlab_project.foo.id
				  branch                 = "BranchProtect-%[1]d"
				  push_access_level      = "maintainer"
				  merge_access_level     = "maintainer"
				  unprotect_access_level = "maintainer"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
					}),
				),
			},
			// Update the Branch Protection access levels using "admin"
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_branch_protection" "branch_protect" {
				  project                = gitlab_project.foo.id
				  branch                 = "BranchProtect-%[1]d"
				  push_access_level      = "maintainer"
				  merge_access_level     = "maintainer"
				  unprotect_access_level = "admin"
				}
					`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                 fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:      api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:     api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						UnprotectAccessLevel: api.AccessLevelValueToName[gitlab.AdminPermissions],
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
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
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
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
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
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.test", &pb),
					testAccCheckGitlabBranchProtectionAttributes(&pb, &testAccGitlabBranchProtectionExpectedAttributes{
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

func TestAccGitlabBranchProtection_createForProjectDefaultBranch(t *testing.T) {
	testProjectName := acctest.RandomWithPrefix("tf-acc-test")
	var protectedBranch gitlab.ProtectedBranch

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and protect its default branch with custom settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "%s"
						initialize_with_readme = true
					}

					resource "gitlab_branch_protection" "default_branch" {
						project = gitlab_project.this.id
						branch = gitlab_project.this.default_branch

						// non-default setting
						allow_force_push = true
					}
				`, testProjectName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.default_branch", &protectedBranch),
					func(_ *terraform.State) error {
						if protectedBranch.AllowForcePush != true {
							return fmt.Errorf("allow_force_push is not set to true")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckGitlabBranchProtectionPersistsInStateCorrectly(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
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

func testAccCheckGitlabBranchProtectionExists(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}
		project, branch, err := projectAndBranchFromID(rs.Primary.ID)
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

func testAccCheckGitlabBranchProtectionComputedAttributes(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return resource.TestCheckResourceAttr(n, "branch_protection_id", strconv.Itoa(pb.ID))(s)
	}
}

type testAccGitlabBranchProtectionExpectedAttributes struct {
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
	CodeOwnerApprovalRequired bool
}

func testAccCheckGitlabBranchProtectionAttributes(pb *gitlab.ProtectedBranch, want *testAccGitlabBranchProtectionExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
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
			}
		}
		if len(remainingWantedUserIDsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToPush)
		}
		if len(remainingWantedGroupIDsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToPush)
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

func testAccCheckGitlabBranchProtectionDestroy(s *terraform.State) error {
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

func testAccGitlabBranchProtectionUpdateConfigAllowForcePushTrue(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%[1]d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_branch_protection" "branch_protect" {
  project                      = gitlab_project.foo.id
  branch                       = "BranchProtect-%[1]d"
  allow_force_push             = true
}
	`, rInt)
}

func testAccGitlabBranchProtectionUpdateConfigCodeOwnerTrue(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name = "foo-%[1]d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}

resource "gitlab_branch_protection" "branch_protect" {
  project                      = gitlab_project.foo.id
  branch                       = "BranchProtect-%[1]d"
  code_owner_approval_required = true
}
	`, rInt)
}
