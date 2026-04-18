//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabBranchProtection_basicCE(t *testing.T) {
	testutil.SkipIfEE(t)

	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Branch Protection with default options
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Configure the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					push_access_level  = "developer"
					merge_access_level = "developer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection; unmanage the access levels
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection with allow force push enabled
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allow_force_push = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.DeveloperPermissions],
						AllowForcePush:   true,
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					push_access_level  = "maintainer"
					merge_access_level = "maintainer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_basicEE(t *testing.T) {
	testutil.SkipIfCE(t)

	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Branch Protection with default options
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Configure some of the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "developer"
					}]

					allowed_to_merge = [{
						access_level = "developer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Configure all of the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "developer"
					}]

					allowed_to_merge = [{
						access_level = "developer"
					}]

					allowed_to_unprotect = [{
						access_level = "developer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "maintainer"
					}]

					allowed_to_merge = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection; unmanage the allowed to attributes
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection with allow force push enabled
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allow_force_push = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AllowForcePush:                 true,
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Update the Branch Protection code owner approval setting
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					code_owner_approval_required = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						CodeOwnerApprovalRequired:      true,
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithCodeOwnerApproval(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Start with code owner approval required not set for CE
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
					`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Create a project and Branch Protection with code owner approval enabled for EE
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`			
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					code_owner_approval_required = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                      fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:           api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel:          api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						CodeOwnerApprovalRequired: true,
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Attempting to update code owner approval setting on CE should fail safely and with an informative error message
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					code_owner_approval_required = true
				}
					`, project.ID, rInt),
				ExpectError: regexp.MustCompile("feature unavailable `code_owner_approval_required`"),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithAllowForcePush(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Start with allow force push not set in CE
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						AllowForcePush:   false,
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AllowForcePush:                 false,
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create a project and Branch Protection with allow force push enabled in CE
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allow_force_push = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						AllowForcePush:   true,
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Create a project and Branch Protection with allow force push enabled in EE
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allow_force_push = true
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:           fmt.Sprintf("BranchProtect-%d", rInt),
						AllowForcePush: true,
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value; in this case the default of Maintainer
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_createWithUnprotectAccessLevel(t *testing.T) {
	testutil.SkipIfCE(t)

	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Configure the Branch Protection access levels
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "developer"
					}]

					allowed_to_merge = [{
						access_level = "developer"
					}]

					allowed_to_unprotect = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.DeveloperPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Branch Protection access levels
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				  
					allowed_to_push = [{
						access_level = "maintainer"
					}]

					allowed_to_merge = [{
						access_level = "maintainer"
					}]

					allowed_to_unprotect = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after update to maintainer
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the Branch Protection access levels using "admin"
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				  
					allowed_to_push = [{
						access_level = "maintainer"
					}]

					allowed_to_merge = [{
						access_level = "maintainer"
					}]

					allowed_to_unprotect = [{
						access_level = "admin"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.AdminPermissions},
					}),
				),
			},
			// Verify import after update to admin
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_createForProjectDefaultBranch(t *testing.T) {
	var protectedBranch gitlab.ProtectedBranch

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and protect its default branch with custom settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_branch_protection" "default_branch" {
						project = "%d"
						branch  = "%s"

						// non-default setting
						allow_force_push = true
					}
				`, project.ID, project.DefaultBranch),
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
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.default_branch",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_FailIfEnterpriseFeaturesUsedForCommunityLicense(t *testing.T) {
	testutil.SkipIfEE(t)

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)
	// Set up the groups to share the `testProject` with
	testUsers := testutil.CreateUsers(t, 1)
	// Add users as members to project
	testutil.AddProjectMembers(t, testProject.ID, testUsers)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectLevelMRApprovalsDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					push_access_level  = "maintainer"
					merge_access_level = "maintainer"

					allowed_to_push = [{
						user_id = %[3]d
					}]
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `allowed_to_push`"),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					push_access_level  = "maintainer"
					merge_access_level = "maintainer"

					allowed_to_merge = [{
						user_id = %[3]d
					}]
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `allowed_to_merge`"),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					push_access_level  = "maintainer"
					merge_access_level = "maintainer"

					allowed_to_unprotect = [{
						user_id = %[3]d
					}]
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `allowed_to_unprotect`"),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					push_access_level  = "maintainer"
					merge_access_level = "maintainer"

					code_owner_approval_required = true
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `code_owner_approval_required`"),
			},
		},
	})
}

func TestAccGitlabBranchProtection_FailIfCommunityFeaturesUsedForEnterpriseLicense(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up the project for the protected branch
	testProject := testutil.CreateProject(t)
	// Set up the groups to share the `testProject` with
	testUsers := testutil.CreateUsers(t, 1)
	// Add users as members to project
	testutil.AddProjectMembers(t, testProject.ID, testUsers)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectLevelMRApprovalsDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					push_access_level  = "maintainer"

					allowed_to_merge = [{
						user_id = %[3]d
					}]
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `push_access_level`"),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "default" {
					project            = "%d"
					branch             = "%[2]s"
					merge_access_level = "maintainer"

					allowed_to_push = [{
						user_id = %[3]d
					}]
				}
				`, testProject.ID, testProject.DefaultBranch, testUsers[0].ID),
				ExpectError: regexp.MustCompile("feature unavailable `merge_access_level`"),
			},
		},
	})
}

func TestAccGitlabBranchProtection_adminPushAccessLevel(t *testing.T) {
	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Branch Protection with admin push access level
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "admin"  
					}]
					
					allowed_to_merge = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.AdminPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project            = %d
					branch             = "BranchProtect-%d"
					push_access_level  = "admin"
					merge_access_level = "maintainer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  "admin",
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.AdminPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to change from admin to maintainer
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "maintainer"  
					}]
					
					allowed_to_merge = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Update to change from admin to maintainer
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project            = %d
					branch             = "BranchProtect-%d"
					push_access_level  = "maintainer"
					merge_access_level = "maintainer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after update to maintainer
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update back to admin
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				
					allowed_to_push = [{
						access_level = "admin"  
					}]
					
					allowed_to_merge = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.AdminPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Update back to admin
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project            = %d
					branch             = "BranchProtect-%d"
					push_access_level  = "admin"
					merge_access_level = "maintainer"
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:             fmt.Sprintf("BranchProtect-%d", rInt),
						PushAccessLevel:  "admin",
						MergeAccessLevel: api.AccessLevelValueToName[gitlab.MaintainerPermissions],
						// push_access_level is actually returned in the protected branch PushAccessLevels slice, so this ensures
						// it contains the expected PushAccessLevel value
						AccessLevelsAllowedToPush:  []gitlab.AccessLevelValue{gitlab.AdminPermissions},
						AccessLevelsAllowedToMerge: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after update back to admin
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabBranchProtection_SetAllowedToPushToNoOne(t *testing.T) {
	testutil.SkipIfCE(t)

	var pb gitlab.ProtectedBranch
	rInt := acctest.RandInt()
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabBranchProtectionDestroy,
		Steps: []resource.TestStep{
			// Create a project and Branch Protection with maintainer push access level
			{
				Config: fmt.Sprintf(`				
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"

					allowed_to_push = [{
						access_level = "maintainer"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionComputedAttributes("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after creation
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to change from no one to no one
			{
				Config: fmt.Sprintf(`
				resource "gitlab_branch_protection" "branch_protect" {
					project = %d
					branch  = "BranchProtect-%d"
				  
					allowed_to_push = [{
						access_level = "no one"
					}]
				}
				`, project.ID, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabBranchProtectionExists("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionPersistsInStateCorrectly("gitlab_branch_protection.branch_protect", &pb),
					testAccCheckGitlabBranchProtectionAttributes("gitlab_branch_protection.branch_protect", &pb, &testAccGitlabBranchProtectionExpectedAttributes{
						Name:                           fmt.Sprintf("BranchProtect-%d", rInt),
						AccessLevelsAllowedToPush:      []gitlab.AccessLevelValue{gitlab.NoPermissions},
						AccessLevelsAllowedToMerge:     []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
						AccessLevelsAllowedToUnprotect: []gitlab.AccessLevelValue{gitlab.MaintainerPermissions},
					}),
				),
			},
			// Verify import after update to no one
			{
				ResourceName:      "gitlab_branch_protection.branch_protect",
				ImportState:       true,
				ImportStateVerify: true,
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

		if stateMergeAccessLevel, ok := rs.Primary.Attributes["merge_access_level"]; ok {
			if mergeAccessLevel, err := firstValidAccessLevel(pb.MergeAccessLevels); err == nil {
				if stateMergeAccessLevel != api.AccessLevelValueToName[*mergeAccessLevel] {
					return fmt.Errorf("merge access level not persisted in state correctly")
				}
			}
		}

		if statePushAccessLevel, ok := rs.Primary.Attributes["push_access_level"]; ok {
			if pushAccessLevel, err := firstValidAccessLevel(pb.PushAccessLevels); err == nil {
				if statePushAccessLevel != api.AccessLevelValueToName[*pushAccessLevel] {
					return fmt.Errorf("push access level not persisted in state correctly")
				}
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

func testAccCheckGitlabBranchProtectionComputedAttributes(n string, pb *gitlab.ProtectedBranch) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return resource.TestCheckResourceAttr(n, "branch_protection_id", strconv.FormatInt(pb.ID, 10))(s)
	}
}

type testAccGitlabBranchProtectionExpectedAttributes struct {
	Name                           string
	PushAccessLevel                string
	MergeAccessLevel               string
	AllowForcePush                 bool
	UsersAllowedToPush             []string
	UsersAllowedToMerge            []string
	UsersAllowedToUnprotect        []string
	GroupsAllowedToPush            []string
	GroupsAllowedToMerge           []string
	GroupsAllowedToUnprotect       []string
	DeployKeysAllowedToPush        []string
	AccessLevelsAllowedToPush      []gitlab.AccessLevelValue
	AccessLevelsAllowedToMerge     []gitlab.AccessLevelValue
	AccessLevelsAllowedToUnprotect []gitlab.AccessLevelValue
	CodeOwnerApprovalRequired      bool
}

func testAccCheckGitlabBranchProtectionAttributes(n string, pb *gitlab.ProtectedBranch, want *testAccGitlabBranchProtectionExpectedAttributes) resource.TestCheckFunc {
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

		if pushAccessLevel, err := firstValidAccessLevel(pb.PushAccessLevels); err == nil {
			if want.PushAccessLevel != "" && *pushAccessLevel != api.AccessLevelNameToValue[want.PushAccessLevel] {
				return fmt.Errorf("got push access level %v; want %v", *pushAccessLevel, api.AccessLevelNameToValue[want.PushAccessLevel])
			}
		}

		if mergeAccessLevel, err := firstValidAccessLevel(pb.MergeAccessLevels); err == nil {
			if want.MergeAccessLevel != "" && *mergeAccessLevel != api.AccessLevelNameToValue[want.MergeAccessLevel] {
				return fmt.Errorf("got merge access level %v; want %v", *mergeAccessLevel, api.AccessLevelNameToValue[want.MergeAccessLevel])
			}
		}

		if pb.AllowForcePush != want.AllowForcePush {
			return fmt.Errorf("got allow_force_push %v; want %v", pb.AllowForcePush, want.AllowForcePush)
		}

		remainingWantedUserIDsAllowedToPush := map[int64]struct{}{}
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
		remainingWantedGroupIDsAllowedToPush := map[int64]struct{}{}
		for _, v := range want.GroupsAllowedToPush {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToPush[group.ID] = struct{}{}
		}
		remainingWantedDeployKeyIDsAllowedToPush := map[int64]struct{}{}
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
		remainingWantedAccessLevelsAllowedToPush := map[gitlab.AccessLevelValue]struct{}{}
		for _, v := range want.AccessLevelsAllowedToPush {
			remainingWantedAccessLevelsAllowedToPush[v] = struct{}{}
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
			} else if _, ok := remainingWantedAccessLevelsAllowedToPush[v.AccessLevel]; !ok {
				return fmt.Errorf("found unwanted push access level %v", v.AccessLevel)
			} else {
				delete(remainingWantedAccessLevelsAllowedToPush, v.AccessLevel)
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
		if len(remainingWantedAccessLevelsAllowedToPush) > 0 {
			return fmt.Errorf("failed to find wanted push access levels %v", remainingWantedAccessLevelsAllowedToPush)
		}

		remainingWantedUserIDsAllowedToMerge := map[int64]struct{}{}
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
		remainingWantedGroupIDsAllowedToMerge := map[int64]struct{}{}
		for _, v := range want.GroupsAllowedToMerge {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToMerge[group.ID] = struct{}{}
		}
		remainingWantedAccessLevelsAllowedToMerge := map[gitlab.AccessLevelValue]struct{}{}
		for _, v := range want.AccessLevelsAllowedToMerge {
			remainingWantedAccessLevelsAllowedToMerge[v] = struct{}{}
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
			} else if _, ok := remainingWantedAccessLevelsAllowedToMerge[v.AccessLevel]; !ok {
				return fmt.Errorf("found unwanted merge access level %v", v.AccessLevel)
			} else {
				delete(remainingWantedAccessLevelsAllowedToMerge, v.AccessLevel)
			}
		}
		if len(remainingWantedUserIDsAllowedToMerge) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToMerge)
		}
		if len(remainingWantedGroupIDsAllowedToMerge) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToMerge)
		}
		if len(remainingWantedAccessLevelsAllowedToMerge) > 0 {
			return fmt.Errorf("failed to find wanted merge access levels %v", remainingWantedAccessLevelsAllowedToMerge)
		}

		remainingWantedUserIDsAllowedToUnprotect := map[int64]struct{}{}
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
		remainingWantedGroupIDsAllowedToUnprotect := map[int64]struct{}{}
		for _, v := range want.GroupsAllowedToUnprotect {
			group, _, err := testutil.TestGitlabClient.Groups.GetGroup(v, nil)
			if err != nil {
				return fmt.Errorf("error looking up group by path %v: %v", v, err)
			}
			remainingWantedGroupIDsAllowedToUnprotect[group.ID] = struct{}{}
		}
		remainingWantedAccessLevelsAllowedToUnprotect := map[gitlab.AccessLevelValue]struct{}{}
		for _, v := range want.AccessLevelsAllowedToUnprotect {
			remainingWantedAccessLevelsAllowedToUnprotect[v] = struct{}{}
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
			} else if _, ok := remainingWantedAccessLevelsAllowedToUnprotect[v.AccessLevel]; !ok {
				return fmt.Errorf("found unwanted unprotect access level %v", v.AccessLevel)
			} else {
				delete(remainingWantedAccessLevelsAllowedToUnprotect, v.AccessLevel)
			}
		}
		if len(remainingWantedUserIDsAllowedToUnprotect) > 0 {
			return fmt.Errorf("failed to find wanted user IDs %v", remainingWantedUserIDsAllowedToUnprotect)
		}
		if len(remainingWantedGroupIDsAllowedToUnprotect) > 0 {
			return fmt.Errorf("failed to find wanted group IDs %v", remainingWantedGroupIDsAllowedToUnprotect)
		}
		if len(remainingWantedAccessLevelsAllowedToUnprotect) > 0 {
			return fmt.Errorf("failed to find wanted unprotect access levels %v", remainingWantedAccessLevelsAllowedToUnprotect)
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
		switch rs.Type {
		case "gitlab_project":
			project = rs.Primary.ID
		case "gitlab_branch_protection":
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
