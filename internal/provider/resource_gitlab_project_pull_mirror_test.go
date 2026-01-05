//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectPullMirror_basic(t *testing.T) {
	testProjectToMirror := testutil.CreateProject(t)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with minimal config - GitLab applies defaults
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project       = %d
						url           = "%s"
						auth_user     = "testuser"
						auth_password = "testtoken"
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttrSet("gitlab_project_pull_mirror.test", "mirror_id"),
					resource.TestCheckResourceAttrSet("gitlab_project_pull_mirror.test", "id"),
				),
			},
			// Verify import after create
			{
				ResourceName:      "gitlab_project_pull_mirror.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Configuration fields are not returned by the API, only status fields
				ImportStateVerifyIgnore: []string{
					"auth_password",
					"url",
					"auth_user",
					"enabled",
					"mirror_trigger_builds",
					"only_mirror_protected_branches",
					"mirror_overwrites_diverged_branches",
					"mirror_branch_regex",
				},
			},
			// Update - add explicit option
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project               = %d
						url                   = "%s"
						auth_user             = "testuser"
						auth_password         = "testtoken"
						mirror_trigger_builds = true
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "mirror_trigger_builds", "true"),
				),
			},
			// Verify import after update
			{
				ResourceName:            "gitlab_project_pull_mirror.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_password", "url", "auth_user", "enabled", "mirror_trigger_builds", "only_mirror_protected_branches", "mirror_overwrites_diverged_branches", "mirror_branch_regex"},
			},
		},
	})
}

func TestAccGitlabProjectPullMirror_withOptions(t *testing.T) {
	testProjectToMirror := testutil.CreateProject(t)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with all options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project                              = %d
						url                                  = "%s"
						enabled                              = true
						auth_user                            = "testuser"
						auth_password                        = "testtoken"
						mirror_trigger_builds                = true
						only_mirror_protected_branches       = true
						mirror_overwrites_diverged_branches  = false
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "mirror_trigger_builds", "true"),
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "only_mirror_protected_branches", "true"),
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "mirror_overwrites_diverged_branches", "false"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_pull_mirror.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_password", "url", "auth_user", "enabled", "mirror_trigger_builds", "only_mirror_protected_branches", "mirror_overwrites_diverged_branches", "mirror_branch_regex"},
			},
			// Update - change options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project                              = %d
						url                                  = "%s"
						enabled                              = true
						auth_user                            = "testuser"
						auth_password                        = "testtoken"
						mirror_trigger_builds                = false
						only_mirror_protected_branches       = false
						mirror_overwrites_diverged_branches  = true
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "mirror_trigger_builds", "false"),
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "only_mirror_protected_branches", "false"),
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "mirror_overwrites_diverged_branches", "true"),
				),
			},
			// Verify import after update
			{
				ResourceName:            "gitlab_project_pull_mirror.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_password", "url", "auth_user", "enabled", "mirror_trigger_builds", "only_mirror_protected_branches", "mirror_overwrites_diverged_branches", "mirror_branch_regex"},
			},
		},
	})
}

func TestAccGitlabProjectPullMirror_disable(t *testing.T) {
	testProjectToMirror := testutil.CreateProject(t)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project       = %d
						url           = "%s"
						enabled       = true
						auth_user     = "testuser"
						auth_password = "testtoken"
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "enabled", "true"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_pull_mirror.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_password", "url", "auth_user", "enabled", "mirror_trigger_builds", "only_mirror_protected_branches", "mirror_overwrites_diverged_branches", "mirror_branch_regex"},
			},
			// Disable
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_pull_mirror" "test" {
						project       = %d
						url           = "%s"
						enabled       = false
						auth_user     = "testuser"
						auth_password = "testtoken"
					}
				`, testProject.ID, testProjectToMirror.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_pull_mirror.test", "enabled", "false"),
				),
			},
			// Note: Import is not supported for disabled mirrors because the GitLab API
			// returns a 400 error and doesn't provide mirror details when disabled
		},
	})
}

func TestAccGitlabProjectPullMirror_Validation(t *testing.T) {

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// This should fail because we disallow in-url username/password (even though gitlab supports it)
			{
				Config: `
					resource "gitlab_project_pull_mirror" "test" {
						project       = 1234
						url           = "https://test:pass@example.gitlab.com"
					}
				`,
				ExpectError: regexp.MustCompile("URL contains username or password"),
			},
		},
	})
}
