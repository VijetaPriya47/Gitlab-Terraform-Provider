//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

// Note that this test does not currently test `force_https = true` as it requires
// the GitLab instance to have specific settings it is not configured to use yet.
// See this comment for more information: https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/merge_requests/2527#note_2552722667
func TestAcc_GitlabProjectPagesSettings_basic(t *testing.T) {
	project := testutil.CreateProject(t)
	_, _, err := testutil.TestGitlabClient.Pages.UpdatePages(project.ID, gitlab.UpdatePagesOptions{
		PagesUniqueDomainEnabled: gitlab.Ptr(false),
		PagesHTTPSOnly:           gitlab.Ptr(false),
	})
	if err != nil {
		t.Fatalf("Unable to change project pages settings: %s", err.Error())
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectPagesSettings_CheckDestroyResetsSettings(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pages_settings" "test" {
						project                  = "%d"
						is_unique_domain_enabled = false
						force_https              = false
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "id", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "keep_settings_on_destroy", "true"),
					resource.TestCheckResourceAttrSet("gitlab_project_pages_settings.test", "url"),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "is_unique_domain_enabled", "false"),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "force_https", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_pages_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pages_settings" "test" {
						project                  = "%d"
						keep_settings_on_destroy = false
						is_unique_domain_enabled = true
						force_https              = false
					}
				`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "id", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "keep_settings_on_destroy", "false"),
					resource.TestCheckResourceAttrSet("gitlab_project_pages_settings.test", "url"),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "is_unique_domain_enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_pages_settings.test", "force_https", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_pages_settings.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
		},
	})
}

func testAcc_GitlabProjectPagesSettings_CheckDestroyResetsSettings() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_project_pages_settings" {
				settings, _, err := testutil.TestGitlabClient.Pages.GetPages(rs.Primary.ID)
				if err != nil {
					return fmt.Errorf("Unable to check project pages settings: %s", err.Error())
				}
				if settings.ForceHTTPS || settings.IsUniqueDomainEnabled {
					return fmt.Errorf("Project pages settings have not been set to original: force_https=%t, is_unique_domain_enabled=%t", settings.ForceHTTPS, settings.IsUniqueDomainEnabled)
				}
			}
		}
		return nil
	}
}
