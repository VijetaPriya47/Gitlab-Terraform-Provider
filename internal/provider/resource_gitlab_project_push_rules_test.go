//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectPushRules_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectPushRules_CheckDestroy,
		Steps: []resource.TestStep{
			// Define push rules on the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_push_rules" "foo" {
						project                       = %d
						author_email_regex            = "@gitlab.com$"
						branch_name_regex             = "(feat|fix)\\/*"
						commit_committer_check        = true
						commit_committer_name_check   = true
						commit_message_negative_regex = "ssh\\:\\/\\/"
						commit_message_regex          = "(feat|fix):.*"
						deny_delete_tag               = false
						file_name_regex               = "(jar|exe)$"
						max_file_size                 = 42
						member_check                  = true
						prevent_secrets               = true
						reject_unsigned_commits       = false
					}
						`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "author_email_regex", "@gitlab.com$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "branch_name_regex", "(feat|fix)\\/*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_name_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_negative_regex", "ssh\\:\\/\\/"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_regex", "(feat|fix):.*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "deny_delete_tag", "false"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "file_name_regex", "(jar|exe)$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "max_file_size", "42"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "member_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "prevent_secrets", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "reject_unsigned_commits", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_push_rules.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update some push rules
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_push_rules" "foo" {
						project                       = %d
						author_email_regex            = "@gitlab.com$"
						branch_name_regex             = "(feat|fix|chore)\\/*"
						commit_message_regex          = "(feat|fix|chore):.*"
						deny_delete_tag               = true
						reject_unsigned_commits       = true
					}
						`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "author_email_regex", "@gitlab.com$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "branch_name_regex", "(feat|fix|chore)\\/*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_regex", "(feat|fix|chore):.*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "deny_delete_tag", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "reject_unsigned_commits", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project_push_rules.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_push_rules" "foo" {
						project                       = %d
						author_email_regex            = "@gitlab.com$"
						branch_name_regex             = "(feat|fix)\\/*"
						commit_committer_check        = true
						commit_committer_name_check   = true
						commit_message_negative_regex = "ssh\\:\\/\\/"
						commit_message_regex          = "(feat|fix):.*"
						deny_delete_tag               = false
						file_name_regex               = "(jar|exe)$"
						max_file_size                 = 42
						member_check                  = true
						prevent_secrets               = true
						reject_unsigned_commits       = false
					}
						`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "author_email_regex", "@gitlab.com$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "branch_name_regex", "(feat|fix)\\/*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_name_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_negative_regex", "ssh\\:\\/\\/"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_regex", "(feat|fix):.*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "deny_delete_tag", "false"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "file_name_regex", "(jar|exe)$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "max_file_size", "42"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "member_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "prevent_secrets", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "reject_unsigned_commits", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_push_rules.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectPushRules_ExistingPushRules(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithDefaultPushRules(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectPushRules_CheckDestroy,
		Steps: []resource.TestStep{
			// Update existing push rules on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_push_rules" "foo" {
						project                       = %d
						author_email_regex            = "@gitlab.com$"
						branch_name_regex             = "(feat|fix)\\/*"
						commit_committer_check        = true
						commit_committer_name_check   = true
						commit_message_negative_regex = "ssh\\:\\/\\/"
						commit_message_regex          = "(feat|fix):.*"
						deny_delete_tag               = false
						file_name_regex               = "(jar|exe)$"
						max_file_size                 = 42
						member_check                  = true
						prevent_secrets               = true
						reject_unsigned_commits       = false
					}
						`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "author_email_regex", "@gitlab.com$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "branch_name_regex", "(feat|fix)\\/*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_committer_name_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_negative_regex", "ssh\\:\\/\\/"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "commit_message_regex", "(feat|fix):.*"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "deny_delete_tag", "false"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "file_name_regex", "(jar|exe)$"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "max_file_size", "42"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "member_check", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "prevent_secrets", "true"),
					resource.TestCheckResourceAttr("gitlab_project_push_rules.foo", "reject_unsigned_commits", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_push_rules.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabProjectPushRules_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_push_rules" {
			id, _ := strconv.Atoi(rs.Primary.ID)
			pushRules, _, err := testutil.TestGitlabClient.Projects.GetProjectPushRules(rs.Primary.ID)
			if err == nil && pushRules != nil && pushRules.ProjectID == id {
				return fmt.Errorf("gitlab_project_push_rules resource '%s' still exists", rs.Primary.ID)
			}

			if err != nil && !api.Is404(err) {
				return err
			}

			return nil
		}
	}
	return nil
}
