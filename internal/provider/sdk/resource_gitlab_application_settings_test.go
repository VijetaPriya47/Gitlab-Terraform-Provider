//go:build acceptance
// +build acceptance

package sdk

import (
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGitlabApplicationSettings_basic(t *testing.T) {
	// lintignore:AT001
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Verify empty application settings
			{
				Config: `
					resource "gitlab_application_settings" "this" {}
				`,
			},
			// Verify changing some application settings
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						after_sign_up_text = "Welcome to GitLab!"
					}
				`,
			},
		},
	})
}

func TestAccGitlabApplicationSettings_testCanCreateGroup(t *testing.T) {
	// lintignore:AT001
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						can_create_group = true
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "can_create_group", "true"),
			},
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						can_create_group = false
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "can_create_group", "false"),
			},
		},
	})
}

func TestAccGitlabApplicationSettings_testNullGitProtocol(t *testing.T) {
	// lintignore:AT001
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Verify we can set the git access to non-nil (which will limit it to just SSH)
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						enabled_git_access_protocol = "ssh"
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "enabled_git_access_protocol", "ssh"),
			},
			// Verify we can set to "nil" and this works properly.
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						enabled_git_access_protocol = "nil"
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "enabled_git_access_protocol", ""),
			},
			// Verify we can re-set the git access to non-nil
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						enabled_git_access_protocol = "ssh"
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "enabled_git_access_protocol", "ssh"),
			},
			// Verify can insensitivity of diffSuppress and the logic
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						enabled_git_access_protocol = "NIL"
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "enabled_git_access_protocol", ""),
			},
		},
	})
}

func TestAccGitlabApplicationSettings_testConflicts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccGitlabApplicationSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						housekeeping_enabled = true
						housekeeping_full_repack_period = 10
						housekeeping_optimize_repository_period = 10
					}		
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_full_repack_period", "10"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_optimize_repository_period", "10"),
				),
				// conflicts with housekeeping_full_repack_period
				ExpectError: regexp.MustCompile("housekeeping_optimize_repository_period"),
			},
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						housekeeping_enabled = true
						housekeeping_gc_period = 10
						housekeeping_optimize_repository_period = 10
					}		
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_gc_period", "10"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_optimize_repository_period", "10"),
				),
				// conflicts with housekeeping_gc_period
				ExpectError: regexp.MustCompile("housekeeping_optimize_repository_period"),
			},
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						housekeeping_enabled = true
						housekeeping_incremental_repack_period = 10
						housekeeping_optimize_repository_period = 10
					}		
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_incremental_repack_period", "10"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_optimize_repository_period", "10"),
				),
				// conflicts with housekeeping_incremental_repack_period
				ExpectError: regexp.MustCompile("housekeeping_optimize_repository_period"),
			},
		},
	})
}

func TestAccGitlabApplicationSettings_testState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccGitlabApplicationSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						housekeeping_enabled = true
						housekeeping_optimize_repository_period = 10
					}		
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_application_settings.this", "housekeeping_optimize_repository_period", "10"),
				),
			},
		},
	})
}

/*
README: Adding a test destroy function seems a easier-to-understand path to ilustrate
application settings nature and its inhability to be destroyed than simply using a nil
value in the acceptance test to satisfy the linter.
*/
func testAccGitlabApplicationSettingsDestroy(state *terraform.State) error {
	log.Printf("[DEBUG] destroying application settings does not do anything yet.")
	return nil
}
