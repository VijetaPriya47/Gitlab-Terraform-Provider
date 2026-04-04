//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
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
						url   = "http://example.com"
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
			// Update group hook to set name and description
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group       = "%s"
						url         = "http://example.com"
						name        = "example"
						description = "Example description"
					}
				`, testGroup.FullPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "name", "example"),
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "description", "Example description"),
				),
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
						group       = "%s"
						url         = "http://example.com"
						name        = "example"
						description = "Example description"

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
						project_events             = true
						member_events              = true
						pipeline_events            = true
						wiki_page_events           = true
						deployment_events          = true
						releases_events            = true
						subgroup_events            = true
						feature_flag_events        = true
						vulnerability_events       = true
						emoji_events               = true
						branch_filter_strategy     = "wildcard"
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
						url   = "http://example.com"
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
			// Update group hook to use a custom template
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_hook" "this" {
					group       = "%s"
					url         = "http://example.com"
					name        = "example"
					description = "Example description"

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
					project_events             = true
					member_events              = true
					pipeline_events            = true
					wiki_page_events           = true
					deployment_events          = true
					releases_events            = true
					subgroup_events            = true
					feature_flag_events        = true
					vulnerability_events       = true
					emoji_events               = true
					branch_filter_strategy     = "wildcard"
					custom_webhook_template    = "{\"event\":\"{{object_kind}}\"}"
				}
				`, testGroup.FullPath),
			},
		},
	})
}

func TestAccGitlabGroupHook_customHeaders(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupHookDestroy,
		Steps: []resource.TestStep{
			// Create a Group Hook with custom headers
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group = "%s"
						url   = "http://example.com"
						token = "supersecret"

						custom_headers = [
							{
								key = "test"
								value = "testValue"
							},
							{
								key = "test2"
								value = "testValue2"
							}
						]
						
					}
				`, testGroup.FullPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.#", "2"),
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.0.key", "test"),
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.0.value", "testValue"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_hook.this",
				ImportState:       true,
				ImportStateVerify: true,
				// Values don't come back on "read", so they can't be imported
				ImportStateVerifyIgnore: []string{"token", "custom_headers.0.value", "custom_headers.1.value"},
			},
			// Create a Group Hook with custom headers
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_hook" "this" {
						group = "%s"
						url   = "http://example.com"
						token = "supersecret"

						custom_headers = [
							{
								key = "test"
								value = "newValue"
							},
							{
								key = "test2"
								value = "newValue2"
							}
						]
						
					}
				`, testGroup.FullPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.#", "2"),
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.0.key", "test"),
					resource.TestCheckResourceAttr("gitlab_group_hook.this", "custom_headers.0.value", "newValue"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group_hook.this",
				ImportState:       true,
				ImportStateVerify: true,
				// Values don't come back on "read", so they can't be imported
				ImportStateVerifyIgnore: []string{"token", "custom_headers.0.value", "custom_headers.1.value"},
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

func TestAccGitlabGroupHook_validations(t *testing.T) {
	testutil.SkipIfCE(t)
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectHookDestroy,
		Steps: []resource.TestStep{
			// Validate that URLs may not contain whitepaces
			{
				Config: fmt.Sprintf(`resource "gitlab_group_hook" "foo" {
							group = "%d"
							url = "https://example.com/hook-1234    " // Whitepaces at the end (invalid)
						}`, group.ID),
				ExpectError: regexp.MustCompile("The URL may not contain whitespace"),
			},
			// Validate the branch filter strategy validator
			{
				Config: fmt.Sprintf(`resource "gitlab_group_hook" "foo" {
							group = "%d"
							url = "https://example.com/hook-1234"
							branch_filter_strategy = "all" // invalid value
						}`, group.ID),
				ExpectError: regexp.MustCompile("Attribute branch_filter_strategy value must be one of"),
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
