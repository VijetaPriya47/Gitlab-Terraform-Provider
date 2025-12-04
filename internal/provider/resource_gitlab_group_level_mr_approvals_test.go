//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabGroupLevelMRApprovals_noResetOnDestroy(t *testing.T) {
	testutil.SkipIfCE(t)
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_level_mr_approvals" "test" {
						group                 = %d
						allow_author_approval = false
					}
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_level_mr_approvals.test", "id", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_level_mr_approvals.test", "allow_author_approval", "false"),
				),
			},
			{
				ResourceName:      "gitlab_group_level_mr_approvals.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_level_mr_approvals" "test" {
						group                 = %d
						allow_author_approval = true
						allow_committer_approval = true
						allow_overrides_to_approver_list_per_merge_request = true
						retain_approvals_on_push = true
						require_reauthentication_to_approve = true
					}
				`, group.ID),
			},
			{
				ResourceName:      "gitlab_group_level_mr_approvals.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_level_mr_approvals" "test" {
						group                 = %d
						allow_author_approval = true
						allow_committer_approval = true
						allow_overrides_to_approver_list_per_merge_request = true
						retain_approvals_on_push = true
						require_reauthentication_to_approve = true
					}
				`, group.ID),
				Destroy: true,
				Check: func(*terraform.State) error {
					// Get the group's merge request approval settings to verify they were reset
					settings, _, err := testutil.TestGitlabClient.MergeRequestApprovalSettings.GetGroupMergeRequestApprovalSettings(group.ID)
					if err != nil {
						return fmt.Errorf("Failed to get group MR approval settings: %v", err)
					}

					// Check that the settings were NOT reset to their original values
					// allow_author_approval should stay as true (the last updated value)
					if !settings.AllowAuthorApproval.Value {
						return fmt.Errorf("Expected allow_author_approval to be stay as true, got false")
					}

					return nil
				},
			},
		},
	})
}

func TestAcc_GitlabGroupMRApprovalSettings_resetOnDestroy(t *testing.T) {
	testutil.SkipIfCE(t)
	group := testutil.CreateGroups(t, 1)[0]
	opt := &gitlab.UpdateGroupMergeRequestApprovalSettingsOptions{
		AllowAuthorApproval: gitlab.Ptr(true),
	}
	_, _, err := testutil.TestGitlabClient.MergeRequestApprovalSettings.UpdateGroupMergeRequestApprovalSettings(group.ID, opt)
	if err != nil {
		t.Fatalf("Failed to update group MR approval settings: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_level_mr_approvals" "test" {
						group                    = %d
						allow_author_approval    = false
						allow_committer_approval = true
						keep_settings_on_destroy = false
					}
				`, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_level_mr_approvals.test", "id", fmt.Sprintf("%d", group.ID)),
					resource.TestCheckResourceAttr("gitlab_group_level_mr_approvals.test", "allow_author_approval", "false"),
				),
			},
			{
				ResourceName:      "gitlab_group_level_mr_approvals.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_level_mr_approvals" "test" {
						group                    = %d
						allow_author_approval    = false
						allow_committer_approval = true
						keep_settings_on_destroy = false
					}
				`, group.ID),
				Destroy: true,
				Check: func(*terraform.State) error {
					// Get the group's merge request approval settings to verify they were reset
					settings, _, err := testutil.TestGitlabClient.MergeRequestApprovalSettings.GetGroupMergeRequestApprovalSettings(group.ID)
					if err != nil {
						return fmt.Errorf("Failed to get group MR approval settings: %v", err)
					}

					// Check that the settings were reset to their original values
					// allow_author_approval should be back to true (the original value we set before the test)
					if !settings.AllowAuthorApproval.Value {
						return fmt.Errorf("Expected allow_author_approval to be reset to true, got false")
					}

					return nil
				},
			},
		},
	})
}
