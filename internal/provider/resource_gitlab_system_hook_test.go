//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabSystemHook_basic(t *testing.T) {
	var hook gitlab.Hook
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabSystemHookDestroy,
		Steps: []resource.TestStep{
			// Create a hook with all options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_system_hook" "this" {
						url                      = "https://example.com/hook-%d"
						token                    = "secret-token"
						push_events              = true
						tag_push_events          = true
						merge_requests_events    = true
						repository_update_events = true
						enable_ssl_verification  = true
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabSystemHookExists("gitlab_system_hook.this", &hook),
					resource.TestCheckResourceAttrSet("gitlab_system_hook.this", "created_at"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_system_hook.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
			// Update the hook to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
					resource "gitlab_system_hook" "this" {
						url                      = "https://example.com/hook-%d"
						token                    = "another-secret-token"
						push_events              = false
						tag_push_events          = false
						merge_requests_events    = false
						repository_update_events = false
						enable_ssl_verification  = false
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabSystemHookExists("gitlab_system_hook.this", &hook),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_system_hook.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func TestAccGitlabSystemHook_upgradeFromSDKToFramework(t *testing.T) {
	var hook gitlab.Hook
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabSystemHookDestroy,
		Steps: []resource.TestStep{
			// Create a hook with sdk version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 18.6.0",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_system_hook" "this" {
						url                      = "https://example.com/hook-%d"
						token                    = "secret-token"
						push_events              = true
						tag_push_events          = true
						merge_requests_events    = true
						repository_update_events = true
						enable_ssl_verification  = true
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabSystemHookExists("gitlab_system_hook.this", &hook),
					resource.TestCheckResourceAttrSet("gitlab_system_hook.this", "created_at"),
				),
			},
			// Switch to framework version
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_system_hook" "this" {
						url                      = "https://example.com/hook-%d"
						token                    = "secret-token"
						push_events              = true
						tag_push_events          = true
						merge_requests_events    = true
						repository_update_events = true
						enable_ssl_verification  = true
					}
				`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabSystemHookExists("gitlab_system_hook.this", &hook),
					resource.TestCheckResourceAttrSet("gitlab_system_hook.this", "created_at"),
				),
			},
			// Verify import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_system_hook.this",
				ImportState:              true,
				ImportStateVerify:        true,
				ImportStateVerifyIgnore:  []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabSystemHookExists(n string, hook *gitlab.Hook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		hookID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		gotHook, _, err := testutil.TestGitlabClient.SystemHooks.GetHook(hookID)
		if err != nil {
			return err
		}
		*hook = *gotHook
		return nil
	}
}

func testAccCheckGitlabSystemHookDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_system_hook" {
			continue
		}
		hookID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		gotHook, _, err := testutil.TestGitlabClient.SystemHooks.GetHook(hookID, nil)
		if err == nil {
			if gotHook != nil && gotHook.ID == hookID {
				return fmt.Errorf("System Hook %d still exists after deletion", hookID)
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
