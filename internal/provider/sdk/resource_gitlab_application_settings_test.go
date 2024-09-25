//go:build settings
// +build settings

package sdk

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

var originalSettings *gitlab.UpdateSettingsOptions

func init() {
	currentSettings, _, err := testutil.TestGitlabClient.Settings.GetSettings()
	if err != nil {
		tflog.Debug(context.Background(), "Failed to get application settings")
	}

	currentSettingsJSON, err := json.Marshal(currentSettings)
	if err != nil {
		tflog.Debug(context.Background(), "Failed to marshal application settings")
	}

	if err := json.Unmarshal(currentSettingsJSON, &originalSettings); err != nil {
		tflog.Debug(context.Background(), "Failed to unmarshal application settings")
	}
}

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
						max_terraform_state_size_bytes = 512
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "max_terraform_state_size_bytes", "512"),
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

// Unfortunately, in order for this test to work, there must be a running
// elasticsearch cluster, since GitLab pings the URL to ensure it's a cluster.
// As a result, a t.Skip() is used here, but the test can be validated
// by running a local elastcisearch docker image.
func TestAccGitlabApplicationSettings_elasticSearchSettings(t *testing.T) {
	t.Skip()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccGitlabApplicationSettingsDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						housekeeping_enabled = true
						housekeeping_optimize_repository_period = 10

						elasticsearch_indexing = true
						elasticsearch_search   = true
						elasticsearch_url = [
							"http://localhost:9200/
						]

						elasticsearch_namespace_ids = [
							1,
							2,
							3,
						]
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

func TestAccGitlabApplicationSettings_testMinimumPasswordLength(t *testing.T) {

	// lintignore:AT001
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Verify setting a minimum password length
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						minimum_password_length = 10
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "minimum_password_length", "10"),
			},
			// Verify updating the setting
			{
				Config: `
					resource "gitlab_application_settings" "this" {
						minimum_password_length = 12
					}
				`,
				Check: resource.TestCheckResourceAttr("gitlab_application_settings.this", "minimum_password_length", "12"),
			},
		},
	})
}

/*
README: Adding a test destroy function seems a easier-to-understand path to illustrate
application settings nature and its inability to be destroyed than simply using a nil
value in the acceptance test to satisfy the linter.
*/
func testAccGitlabApplicationSettingsDestroy(state *terraform.State) error {
	tflog.Debug(context.Background(), "[DEBUG] destroying application settings does not do anything yet.")
	return nil
}
