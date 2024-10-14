//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroupHook_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

func TestAccGitlabGroupHook_migrateFromSDKToFramework(t *testing.T) {
	testutil.SkipIfCE(t)
	group := testutil.CreateGroups(t, 1)[0]

	// Create common config for testing
	config := fmt.Sprintf(`resource "gitlab_group_hook" "foo" {
		group = "%d"
		url = "https://example.com/hook-1234"
	  }`, group.ID)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 17.4",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
				Check:  resource.TestCheckResourceAttr("gitlab_group_hook.foo", "url", "https://example.com/hook-1234"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    resource.TestCheckResourceAttr("gitlab_group_hook.foo", "url", "https://example.com/hook-1234"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_group_hook.foo",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabGroupHookDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group_hook" {
			continue
		}

		group, hookID, err := (&gitlabGroupHookResourceModel{}).ResourceGitlabGroupHookParseID(rs.Primary.ID)
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
