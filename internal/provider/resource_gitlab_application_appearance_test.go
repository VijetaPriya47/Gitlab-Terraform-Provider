//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabApplicationAppearance_basic(t *testing.T) {
	title := acctest.RandString(10)
	_, _, err := testutil.TestGitlabClient.Appearance.ChangeAppearance(&gitlab.ChangeAppearanceOptions{
		Title: gitlab.Ptr("original"),
	})
	if err != nil {
		t.Fatalf("Unable to change application appearance: %s", err.Error())
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabApplicationAppearance_CheckDestroyResetsAppearance(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_application_appearance" "test" {
						title                           = "%s"
						description                     = "A test application"
						email_header_and_footer_enabled = false
					}
				`, title),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "title", title),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "description", "A test application"),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "email_header_and_footer_enabled", "false"),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "keep_settings_on_destroy", "true"),
				),
			},
			{
				ResourceName:      "gitlab_application_appearance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_application_appearance" "test" {
						title                           = "%s"
						description                     = "A test application again"
						header_message                  = "A header message"
						email_header_and_footer_enabled = true
					}
				`, title),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "title", title),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "description", "A test application again"),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "header_message", "A header message"),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "email_header_and_footer_enabled", "true"),
				),
			},
			{
				ResourceName:      "gitlab_application_appearance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore this as it's defaulted by the provider.
				ImportStateVerifyIgnore: []string{
					"keep_settings_on_destroy",
				},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_application_appearance" "test" {
						title                           = "%s"
						keep_settings_on_destroy        = false
						description                     = "A test application again"
						header_message                  = ""
						email_header_and_footer_enabled = false
					}
				`, title),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "title", title),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "description", "A test application again"),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "header_message", ""),
					resource.TestCheckResourceAttr("gitlab_application_appearance.test", "email_header_and_footer_enabled", "false"),
				),
			},
			{
				ResourceName:      "gitlab_application_appearance.test",
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

func TestAcc_GitlabApplicationAppearance_attributeValidation(t *testing.T) {
	title := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_application_appearance" "test" {
						title                    = "%s"
						description              = "A test application"
						message_background_color = "yellow"
					}
				`, title),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_application_appearance" "test" {
						title              = "%s"
						description        = "A test application"
						message_font_color = "yellow"
					}
				`, title),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func testAcc_GitlabApplicationAppearance_CheckDestroyResetsAppearance() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "gitlab_application_appearance" {
				a, _, err := testutil.TestGitlabClient.Appearance.GetAppearance()
				if err != nil {
					return fmt.Errorf("Unable to check application appearance: %s", err.Error())
				}
				if a.Title != "original" {
					return fmt.Errorf("Application appearance has not been set to original: %s", a.Title)
				}
			}
		}
		return nil
	}
}
