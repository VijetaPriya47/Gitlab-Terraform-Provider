//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupHook_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupHookDestroy,
		Steps: []resource.TestStep{
			// Create a Group Hook with required attributes only
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group = "%s"
						url = "http://example.com"
					}
				`, testGroup.FullPath),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_group_hook.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update Group Hook to set all attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group = "%s"
						url = "http://example.com"

						token                      = "supersecret"
						enable_ssl_verification    = false
						push_events                = true
						push_events_branch_filter  = "devel"
						issues_events              = false
						confidential_issues_events = false
						merge_requests_events      = true
						tag_push_events            = true
						note_events                = true
						confidential_note_events   = true
						job_events                 = true
						pipeline_events            = true
						wiki_page_events           = true
						deployment_events          = true
						releases_events            = true
						subgroup_events            = true
					}
				`, testGroup.FullPath),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_group_hook.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update Group Hook to defaults again
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group = "%s"
						url = "http://example.com"
					}
				`, testGroup.FullPath),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_group_hook.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update group hook to use a custom template, only after version 16.10
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.Background(), testutil.TestGitlabClient, "16.10"),
				Config: fmt.Sprintf(`
				resource "gitlab_group_hook" "this" {
					group = "%s"
					url = "http://example.com"

					token                      = "supersecret"
					enable_ssl_verification    = false
					push_events                = true
					push_events_branch_filter  = "devel"
					issues_events              = false
					confidential_issues_events = false
					merge_requests_events      = true
					tag_push_events            = true
					note_events                = true
					confidential_note_events   = true
					job_events                 = true
					pipeline_events            = true
					wiki_page_events           = true
					deployment_events          = true
					releases_events            = true
					subgroup_events            = true
					custom_webhook_template    = "{\"event\":\"{{object_kind}}\"}"
				}
				`, testGroup.FullPath),
			},
		},
	})
}

func testAccCheckGitlabGroupHookDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_hook" {
			continue
		}

		group, hookID, err := resourceGitlabGroupHookParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.Groups.GetGroupHook(group, hookID)
		if err == nil {
			return fmt.Errorf("Group Hook %d in group %s still exists", hookID, group)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
