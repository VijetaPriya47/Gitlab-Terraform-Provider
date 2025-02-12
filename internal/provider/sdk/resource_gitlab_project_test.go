//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

type testAccGitlabProjectExpectedAttributes struct {
	DefaultBranch string
}

func TestAccGitlabProject_minimal(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"
						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_basic(t *testing.T) {
	var received, defaults, defaultsMainBranch gitlab.Project
	rInt := acctest.RandInt()
	topic := acctest.RandString(4)

	defaults = testProjectDefaults(rInt)

	defaultsMainBranch = testProjectDefaults(rInt)
	defaultsMainBranch.DefaultBranch = "main"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project with all the features on (note: "archived" is "false")
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					testAccCheckAggregateGitlabProject(&defaults, &received),
				),
			},
			// Update the project to turn the features off (note: "archived" is "true")
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  path = "foo.%d"
				  description = "Terraform acceptance tests!"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				  merge_method = "ff"
				  only_allow_merge_if_pipeline_succeeds = true
				  only_allow_merge_if_all_discussions_are_resolved = true
				  squash_option = "default_on"
				  allow_merge_on_skipped_pipeline = true
				  restrict_user_defined_variables = false
				  request_access_enabled = false
				  issues_enabled = false
				  merge_requests_enabled = false
				  pipelines_enabled = false
				  approvals_before_merge = 0
				  wiki_enabled = false
				  snippets_enabled = false
				  lfs_enabled = false
				  shared_runners_enabled = false
				  group_runners_enabled = false
				  archived = true
				  packages_enabled = false
				  pages_access_level = "disabled"
				  ci_forward_deployment_enabled = false
				  ci_separated_caches = false
				  keep_latest_artifact = false
				  merge_pipelines_enabled = false
				  merge_trains_enabled = false
				  resolve_outdated_diff_discussions = false
				  analytics_access_level = "disabled"
				  auto_cancel_pending_pipelines = "disabled"
				  auto_devops_deploy_strategy = "manual"
				  auto_devops_enabled = false
				  autoclose_referenced_issues = false
				  build_git_strategy = "fetch"
				  build_timeout = 10 * 60
				  builds_access_level = "disabled"
				  emails_enabled = true
				  forking_access_level = "disabled"
				  issues_access_level = "disabled"
				  merge_requests_access_level = "disabled"
				  public_jobs = false
				  repository_access_level = "disabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "disabled"
				  snippets_access_level = "disabled"
				  topics = []
				  wiki_access_level = "disabled"
				  squash_commit_template = "goodby squash"
				  merge_commit_template = "goodby merge"
				  ci_default_git_depth = 84
				  releases_access_level = "disabled"
				  environments_access_level = "disabled"
				  feature_flags_access_level = "disabled"
				  infrastructure_access_level = "disabled"
				  monitor_access_level = "disabled"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					testAccCheckAggregateGitlabProject(&gitlab.Project{
						Namespace:                        &gitlab.ProjectNamespace{ID: 0},
						Name:                             fmt.Sprintf("foo-%d", rInt),
						Path:                             fmt.Sprintf("foo.%d", rInt),
						Description:                      "Terraform acceptance tests!",
						TagList:                          []string{"foo", "bar"},
						JobsEnabled:                      false,
						ApprovalsBeforeMerge:             0,
						RequestAccessEnabled:             false,
						ContainerRegistryEnabled:         false,
						LFSEnabled:                       false,
						SharedRunnersEnabled:             false,
						GroupRunnersEnabled:              false,
						Visibility:                       gitlab.PublicVisibility,
						MergeMethod:                      gitlab.FastForwardMerge,
						PrintingMergeRequestLinkEnabled:  true,
						OnlyAllowMergeIfPipelineSucceeds: true,
						OnlyAllowMergeIfAllDiscussionsAreResolved: true,
						SquashOption:                   gitlab.SquashOptionDefaultOn,
						AllowMergeOnSkippedPipeline:    true,
						Archived:                       true,
						PackagesEnabled:                false,
						PagesAccessLevel:               gitlab.DisabledAccessControl,
						CIForwardDeploymentEnabled:     false,
						CISeperateCache:                false,
						KeepLatestArtifact:             false,
						ResolveOutdatedDiffDiscussions: false,
						AnalyticsAccessLevel:           gitlab.DisabledAccessControl,
						AutoCancelPendingPipelines:     "disabled",
						AutoDevopsDeployStrategy:       "manual",
						AutoDevopsEnabled:              false,
						AutocloseReferencedIssues:      false,
						BuildGitStrategy:               "fetch",
						BuildTimeout:                   10 * 60,
						BuildsAccessLevel:              gitlab.DisabledAccessControl,
						ContainerExpirationPolicy: &gitlab.ContainerExpirationPolicy{
							Enabled:   true,
							Cadence:   "4h",
							KeepN:     10,
							OlderThan: "10d",
						},
						ContainerRegistryAccessLevel:     gitlab.DisabledAccessControl,
						EmailsDisabled:                   false,
						ForkingAccessLevel:               gitlab.DisabledAccessControl,
						IssuesAccessLevel:                gitlab.DisabledAccessControl,
						MergeRequestsAccessLevel:         gitlab.DisabledAccessControl,
						OperationsAccessLevel:            gitlab.DisabledAccessControl,
						PublicBuilds:                     false,
						RepositoryAccessLevel:            gitlab.DisabledAccessControl,
						RepositoryStorage:                "default",
						SecurityAndComplianceAccessLevel: gitlab.DisabledAccessControl,
						SnippetsAccessLevel:              gitlab.DisabledAccessControl,
						Topics:                           []string{},
						WikiAccessLevel:                  gitlab.DisabledAccessControl,
						SquashCommitTemplate:             "goodby squash",
						MergeCommitTemplate:              "goodby merge",
					}, &received),
				),
			},
			// Update the project to turn the features on again (note: "archived" is "false")
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					testAccCheckAggregateGitlabProject(&defaults, &received),
				),
			},
			// Update the project creating the default branch
			{
				// Get the ID from the project data at the previous step
				SkipFunc: testAccGitlabProjectConfigDefaultBranchSkipFunc(&received, "main"),
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					default_branch = "main"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					testAccCheckAggregateGitlabProject(&defaultsMainBranch, &received),
				),
			},
			// Test import without push rules (checks read function)
			{
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add all push rules to an existing project
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
					branch_name_regex = "foo_branch"
					commit_message_regex = "foo_commit"
					commit_message_negative_regex = "foo_not_commit"
					file_name_regex = "foo_file"
					commit_committer_check = true
					commit_committer_name_check = true
					deny_delete_tag = true
					member_check = true
					prevent_secrets = true
					reject_unsigned_commits = true
					reject_non_dco_commits = true
					max_file_size = 123
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file",
					CommitCommitterCheck:       true,
					CommitCommitterNameCheck:   true,
					DenyDeleteTag:              true,
					MemberCheck:                true,
					PreventSecrets:             true,
					RejectUnsignedCommits:      true,
					RejectNonDCOCommits:        true,
					MaxFileSize:                123,
				}),
			},
			// Test import with a all push rules defined (checks read function)
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update some push rules but not others
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
					branch_name_regex = "foo_branch"
					commit_message_regex = "foo_commit"
					commit_message_negative_regex = "foo_not_commit"
					file_name_regex = "foo_file_2"
					commit_committer_check = true
					commit_committer_name_check = false
					deny_delete_tag = true
					member_check = false
					prevent_secrets = true
					reject_unsigned_commits = true
					reject_non_dco_commits = true
					max_file_size = 1234
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file_2",
					CommitCommitterCheck:       true,
					CommitCommitterNameCheck:   false,
					DenyDeleteTag:              true,
					MemberCheck:                false,
					PreventSecrets:             true,
					RejectUnsignedCommits:      true,
					RejectNonDCOCommits:        true,
					MaxFileSize:                1234,
				}),
			},
			// Try to add push rules to an existing project in CE
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("Project push rules are not supported in your version of GitLab")),
			},
			// Update push rules
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					AuthorEmailRegex: "foo_author",
				}),
			},
			// Remove the push_rules block entirely.
			// NOTE: The push rules will still exist upstream because the push_rules block is computed.
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					default_branch = "main"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					AuthorEmailRegex: "foo_author",
				}),
			},
			// Add different push rules after the block was removed previously
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					branch_name_regex = "(feature|hotfix)\\/*"
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					BranchNameRegex: `(feature|hotfix)\/*`,
				}),
			},
		},
	})
}

func TestAccGitlabProject_templates(t *testing.T) {
	// There is an issue where the _templates
	// test _frequently_ fails due to being unable
	// to push the file to the template project branch.

	// Skip it to ensure we have reasonable failure rates.
	t.Skip()

	var received gitlab.Project
	rInt := acctest.RandInt()

	templateFileName := "test.txt"
	templateProject := testAccGitLabProjectCreateTemplateProject(t, templateFileName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project using custom template name
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "template-name-custom" {
				  name = "template-name-custom-%d"
				  path = "template-name-custom.%d"
				  description = "Terraform acceptance tests"
				  template_name = "%s"
				  use_custom_template = true
				  skip_wait_for_default_branch_protection = "false"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt, templateProject.Name),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.template-name-custom", &received),
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.template-name-custom"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, templateFileName, &gitlab.GetFileOptions{Ref: gitlab.Ptr(received.DefaultBranch)}, nil)
						if err != nil {
							return fmt.Errorf("failed to get %s' file from template project: %w", templateFileName, err)
						}

						return nil
					},
				),
			},
			// Create a project using custom template project id
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "template-id" {
				  name = "template-id-%d"
				  path = "template-id.%d"
				  description = "Terraform acceptance tests"
				  template_project_id = %d
				  use_custom_template = true
				  skip_wait_for_default_branch_protection = "false"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt, templateProject.ID),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.template-id", &received),
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.template-id"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, templateFileName, &gitlab.GetFileOptions{Ref: gitlab.Ptr(received.DefaultBranch)}, nil)
						if err != nil {
							return fmt.Errorf("failed to get '%s' file from template project: %w", templateFileName, err)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_PushRules(t *testing.T) {
	rInt := acctest.RandInt()
	topic := acctest.RandString(4)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a new project with push rules
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
					max_file_size = 123
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					AuthorEmailRegex: "foo_author",
					MaxFileSize:      123,
				}),
			},
			// Verify import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to original project config
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
			},
			// Verify import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Try to create a new project with all push rules in CE
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					author_email_regex = "foo_author"
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("Project push rules are not supported in your version of GitLab")),
			},
		},
	})
}

func TestAccGitlabProject_initializeWithReadme(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name                   = "foo-%d"
				  path                   = "foo.%d"
				  description            = "Terraform acceptance tests"
				  initialize_with_readme = true
				
				  # Not required for the test, and makes it much more stable
				  skip_wait_for_default_branch_protection = true
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(state *terraform.State) error {
						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(project.ID, "README.md", &gitlab.GetFileOptions{Ref: gitlab.Ptr("main")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get 'README.md' file from project: %w", err)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_initializeWithoutReadme(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name                   = "foo-%d"
				  path                   = "foo.%d"
				  description            = "Terraform acceptance tests"
				  initialize_with_readme = false
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(s *terraform.State) error {
						branches, _, err := testutil.TestGitlabClient.Branches.ListBranches(project.ID, nil)
						if err != nil {
							return fmt.Errorf("failed to list branches: %w", err)
						}

						if len(branches) != 0 {
							return fmt.Errorf("expected no branch for new project when initialized without README; found %d", len(branches))
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_archiveOnDestroy(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectArchivedOnDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  path = "foo.%d"
				  description = "Terraform acceptance tests"
				  archive_on_destroy = true
				  archived = false
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
			},
		},
	})
}

func TestAccGitlabProject_setSinglePushRuleToDefault(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%[1]d"
				  path = "foo.%[1]d"
				  description = "Terraform acceptance tests"
				
				  push_rules {
					member_check = false
				  }
				
				  resolve_outdated_diff_discussions = true
				  analytics_access_level = "enabled"
				  auto_cancel_pending_pipelines = "enabled"
				  auto_devops_deploy_strategy = "continuous"
				  auto_devops_enabled = true
				  autoclose_referenced_issues = true
				  build_git_strategy = "fetch"
				  build_timeout = 42 * 60
				  builds_access_level = "enabled"
				  emails_enabled = false
				  forking_access_level = "enabled"
				  issues_access_level = "enabled"
				  merge_requests_access_level = "enabled"
				  public_jobs = false
				  repository_access_level = "enabled"
				  repository_storage = "default"
				  security_and_compliance_access_level = "enabled"
				  snippets_access_level = "enabled"
				  suggestion_commit_message = "hello suggestion"
				  wiki_access_level = "enabled"
				  squash_commit_template = "hello squash"
				  merge_commit_template = "hello merge"
				  releases_access_level = "enabled"
				  environments_access_level = "enabled"
				  feature_flags_access_level = "enabled"
				  infrastructure_access_level = "enabled"
				  monitor_access_level = "enabled"
				
				  # So that acceptance tests can be run in a gitlab organization with no billing.
				  visibility_level = "public"
				}
				`, rInt),
				Check: testAccCheckGitlabProjectPushRules("gitlab_project.foo", &gitlab.ProjectPushRules{
					MemberCheck: false,
				}),
			},
		},
	})
}

func TestAccGitlabProject_groupWithoutDefaultBranchProtection(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foogroup-%d"
				  path = "foogroup-%d"
				  default_branch_protection = 0
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  description = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo.id}"
				}
				`, rInt, rInt, rInt),
				Check: testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foogroup-%d"
				  path = "foogroup-%d"
				  default_branch_protection = 0
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  description = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo.id}"
				}
				`, rInt, rInt, rInt),
				Destroy: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foogroup2-%d"
				  path = "foogroup2-%d"
				  default_branch_protection = 0
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  description = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo.id}"
				  initialize_with_readme = true
				}
				`, rInt, rInt, rInt),
				Check: testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
			},
		},
	})
}

func TestAccGitlabProject_IssueMergeRequestTemplates(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  path = "foo.%d"
				  description = "Terraform acceptance tests"
				  issues_template = "foo"
				  merge_requests_template = "bar"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(s *terraform.State) error {
						if project.IssuesTemplate != "foo" {
							return fmt.Errorf("expected issues template to be 'foo'; got '%s'", project.IssuesTemplate)
						}

						if project.MergeRequestsTemplate != "bar" {
							return fmt.Errorf("expected merge requests template to be 'bar'; got '%s'", project.MergeRequestsTemplate)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_MergePipelines(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  path = "foo.%d"
				  description = "Terraform acceptance tests"
				  merge_pipelines_enabled = true
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(s *terraform.State) error {
						if project.MergePipelinesEnabled != true {
							return fmt.Errorf("expected merge pipelines to be enabled")
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_MergeTrains(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  path = "foo.%d"
				  description = "Terraform acceptance tests"
				  merge_pipelines_enabled = true
				  merge_trains_enabled = true
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(s *terraform.State) error {
						if project.MergeTrainsEnabled != true {
							return fmt.Errorf("expected merge trains to be enabled")
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_willErrorOnAPIFailure(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Step0 Create a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
				),
			},
			// Step1 Verify that passing bad values will fail.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"
						repository_storage = "non-existing"

						visibility_level = "public"
					}`, rInt),
				// This will fail because the repository_storage is not valid.
				ExpectError: regexp.MustCompile(`\[is invalid\]`),
			},
			// Step2 Reset
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
				),
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAccGitlabProject_import(t *testing.T) {
	rInt := acctest.RandInt()
	topic := acctest.RandString(4)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInEE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
				}
				`, rInt, rInt, topic),
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_project" "foo" {
					name = "foo-%d"
					path = "foo.%d"
					description = "Terraform acceptance tests"
					default_branch = "main"
					
					# NOTE: replaces by topics
					# tags = [
					# "tag1",
					# ]
					
					# So that acceptance tests can be run in a gitlab organization
					# with no billing
					visibility_level = "public"
					merge_method = "ff"
					only_allow_merge_if_pipeline_succeeds = true
					only_allow_merge_if_all_discussions_are_resolved = true
					squash_option = "default_off"
					pages_access_level = "public"
					allow_merge_on_skipped_pipeline = false
					restrict_user_defined_variables = false
					ci_config_path = ".gitlab-ci.yml@mynamespace/myproject"
					resolve_outdated_diff_discussions = true
					analytics_access_level = "enabled"
					auto_cancel_pending_pipelines = "enabled"
					auto_devops_deploy_strategy = "continuous"
					auto_devops_enabled = true
					autoclose_referenced_issues = true
					build_git_strategy = "fetch"
					build_timeout = 42 * 60
					builds_access_level = "enabled"
					emails_enabled = false
					forking_access_level = "enabled"
					issues_access_level = "enabled"
					merge_requests_access_level = "enabled"
					public_jobs = false
					repository_access_level = "enabled"
					repository_storage = "default"
					security_and_compliance_access_level = "enabled"
					snippets_access_level = "enabled"
					suggestion_commit_message = "hello suggestion"
					topics = ["%s"]
					wiki_access_level = "enabled"
					squash_commit_template = "hello squash"
					merge_commit_template = "hello merge"
					ci_default_git_depth = 42
					releases_access_level = "enabled"
					environments_access_level = "enabled"
					feature_flags_access_level = "enabled"
					infrastructure_access_level = "enabled"
					monitor_access_level = "enabled"
					
					# EE features
					approvals_before_merge = 2
					external_authorization_classification_label = "test"
					requirements_access_level = "enabled"
					model_experiments_access_level = "enabled"
					model_registry_access_level = "enabled"
					# are tested in separate test case
					# mirror_trigger_builds = true
					# mirror = true
				}
				`, rInt, rInt, topic),
			},
			{
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAccGitlabProject_nestedImport(t *testing.T) {
	rInt := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name             = "foogroup-%d"
				  path             = "foogroup-%d"
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name         = "foo-%d"
				  description  = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo.id}"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt, rInt),
			},
			{
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_transfer(t *testing.T) {
	var transferred, received gitlab.Project
	rInt := acctest.RandInt()

	transferred = gitlab.Project{
		Namespace:                        &gitlab.ProjectNamespace{Name: fmt.Sprintf("foo2group-%d", rInt)},
		Name:                             fmt.Sprintf("foo-%d", rInt),
		Path:                             fmt.Sprintf("foo-%d", rInt),
		Description:                      "Terraform acceptance tests",
		TagList:                          []string{},
		RequestAccessEnabled:             true,
		IssuesEnabled:                    true,
		MergeRequestsEnabled:             true,
		JobsEnabled:                      true,
		ApprovalsBeforeMerge:             0,
		WikiEnabled:                      true,
		SnippetsEnabled:                  true,
		ContainerRegistryEnabled:         true,
		LFSEnabled:                       true,
		SharedRunnersEnabled:             true,
		GroupRunnersEnabled:              true,
		Visibility:                       gitlab.PublicVisibility,
		MergeMethod:                      gitlab.NoFastForwardMerge,
		OnlyAllowMergeIfPipelineSucceeds: false,
		OnlyAllowMergeIfAllDiscussionsAreResolved: false,
		SquashOption:                    gitlab.SquashOptionDefaultOff,
		PackagesEnabled:                 true,
		PrintingMergeRequestLinkEnabled: true,
		PagesAccessLevel:                gitlab.PrivateAccessControl,
		CIForwardDeploymentEnabled:      true,
		CISeperateCache:                 true,
		KeepLatestArtifact:              true,
	}

	pathBeforeTransfer := fmt.Sprintf("foogroup-%d/foo-%d", rInt, rInt)
	pathAfterTransfer := fmt.Sprintf("foo2group-%d/foo-%d", rInt, rInt)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project in a group
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foogroup-%d"
				  path = "foogroup-%d"
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  description = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo.id}"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_project_variable" "foo" {
				  project = "${gitlab_project.foo.id}"
				
				  key = "FOO"
				  value = "${gitlab_project.foo.path_with_namespace}"
				}
				`, rInt, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					resource.TestCheckResourceAttrPtr("gitlab_project_variable.foo", "value", &pathBeforeTransfer),
				),
			},
			// Create a second group and set the transfer the project to this group
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foogroup-%d"
				  path = "foogroup-%d"
				  visibility_level = "public"
				}
				
				resource "gitlab_group" "foo2" {
				  name = "foo2group-%d"
				  path = "foo2group-%d"
				  visibility_level = "public"
				}
				
				resource "gitlab_project" "foo" {
				  name = "foo-%d"
				  description = "Terraform acceptance tests"
				  namespace_id = "${gitlab_group.foo2.id}"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				
				resource "gitlab_project_variable" "foo" {
				  project = "${gitlab_project.foo.id}"
				
				  key = "FOO"
				  value = "${gitlab_project.foo.path_with_namespace}"
				}
				`, rInt, rInt, rInt, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &received),
					testAccCheckAggregateGitlabProject(&transferred, &received),
					resource.TestCheckResourceAttrPtr("gitlab_project_variable.foo", "value", &pathAfterTransfer),
				),
			},
		},
	})
}

// lintignore: AT002 // not a Terraform import test
func TestAccGitlabProject_importURL(t *testing.T) {
	rInt := acctest.RandInt()

	// Create a base project for importing.
	baseProject, _, err := testutil.TestGitlabClient.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name:       gitlab.Ptr(fmt.Sprintf("base-%d", rInt)),
		Visibility: gitlab.Ptr(gitlab.PublicVisibility),
	})
	if err != nil {
		t.Fatalf("failed to create base project: %v", err)
	}

	defer testutil.TestGitlabClient.Projects.DeleteProject(baseProject.ID, nil) // nolint // TODO: Resolve this golangci-lint issue: Error return value of `TestGitlabClient.Projects.DeleteProject` is not checked (errcheck)

	// Add a file to the base project, for later verifying the import.
	_, _, err = testutil.TestGitlabClient.RepositoryFiles.CreateFile(baseProject.ID, "foo.txt", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("add file"),
		Content:       gitlab.Ptr(""),
	})
	if err != nil {
		t.Fatalf("failed to commit file to base project: %v", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "imported" {
				  name      = "imported-%d"
				  default_branch = "main"
				  import_url     = "%s"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, baseProject.HTTPURLToRepo),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.imported", "import_url", baseProject.HTTPURLToRepo),
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.imported"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, "foo.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr("main")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get file from imported project: %w", err)
						}

						return nil
					},
				),
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAccGitlabProject_importURLWithPassword(t *testing.T) {
	rInt := acctest.RandInt()

	// Create a base project for importing.
	baseProject, _, err := testutil.TestGitlabClient.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name:       gitlab.Ptr(fmt.Sprintf("base-%d", rInt)),
		Visibility: gitlab.Ptr(gitlab.PrivateVisibility),
	})
	if err != nil {
		t.Fatalf("failed to create base project: %v", err)
	}

	// Get an access token to use for cloning a private project
	// The expiration must be greater than 24 hours, or the token will instantly expire.
	expiration := gitlab.ISOTime(time.Now().Add(time.Hour * 48))
	token, _, err := testutil.TestGitlabClient.ProjectAccessTokens.CreateProjectAccessToken(baseProject.ID, &gitlab.CreateProjectAccessTokenOptions{
		Name:        gitlab.Ptr("clone"),
		Scopes:      &[]string{"api", "read_repository"},
		AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
		ExpiresAt:   &expiration,
	})
	if err != nil {
		t.Fatalf("failed to create project access token: %v", err)
	}

	defer testutil.TestGitlabClient.Projects.DeleteProject(baseProject.ID, nil) // nolint // TODO: Resolve this golangci-lint issue: Error return value of `TestGitlabClient.Projects.DeleteProject` is not checked (errcheck)

	// Add a file to the base project, for later verifying the import.
	_, _, err = testutil.TestGitlabClient.RepositoryFiles.CreateFile(baseProject.ID, "foo.txt", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("add file"),
		Content:       gitlab.Ptr(""),
	})
	if err != nil {
		t.Fatalf("failed to commit file to base project: %v", err)
	}

	// add our username and token so we can clone a private project.
	importUrl := strings.ReplaceAll(baseProject.HTTPURLToRepo, "://", fmt.Sprintf("://root:%s@", token.Token))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "this" {
					name        = "import-url-with-password-%d"
					import_url  = "%s"

          lifecycle {
            ignore_changes = [import_url]
          }
				}
				`, rInt, importUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project.this", "import_url"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["gitlab_project.this"]
						if !ok {
							return fmt.Errorf("gitlab_project.this not found")
						}

						projectId := rs.Primary.ID
						thisProject, _, _ := testutil.TestGitlabClient.Projects.GetProject(projectId, &gitlab.GetProjectOptions{})
						tflog.Trace(context.TODO(), fmt.Sprintf("%d", thisProject.ID))

						return nil
					},
				),
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAccGitlabProject_importURL_publicRepository(t *testing.T) {
	testImportedProjectName := acctest.RandomWithPrefix("acctest")
	testProject := testutil.CreateProject(t)

	config := fmt.Sprintf(`
    resource "gitlab_project" "test" {
      name = "%s"

      import_url = "%s"
    }
  `, testImportedProjectName, testProject.HTTPURLToRepo)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			// Expect empty plan on re-apply
			{
				Config:   config,
				PlanOnly: true,
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// lintignore: AT002 // specialized import test
func TestAccGitlabProject_importURL_privateRepository(t *testing.T) {
	testutil.SkipIfCE(t)

	testImportedProjectName := acctest.RandomWithPrefix("acctest")
	testProject := testutil.CreateProjectWithOptions(t, &gitlab.CreateProjectOptions{
		Name:                 gitlab.Ptr(testImportedProjectName),
		Visibility:           gitlab.Ptr(gitlab.PrivateVisibility),
		InitializeWithReadme: gitlab.Ptr(true),
	})

	createToken := func() string {
		expiration := gitlab.ISOTime(time.Now().Add(time.Hour * 48))
		token, _, err := testutil.TestGitlabClient.ProjectAccessTokens.CreateProjectAccessToken(testProject.ID, &gitlab.CreateProjectAccessTokenOptions{
			Name:        gitlab.Ptr(acctest.RandomWithPrefix("acctest")),
			Scopes:      &[]string{"read_api", "read_repository"},
			AccessLevel: gitlab.Ptr(gitlab.MaintainerPermissions),
			ExpiresAt:   &expiration,
		})
		if err != nil {
			t.Fatalf("failed to create project access token: %v", err)
		}
		return token.Token
	}

	tokenForCreate := createToken()
	tokenForUpdate := createToken()

	createConfig := fmt.Sprintf(`
    resource "gitlab_project" "test" {
      name = "imported-%s"

      import_url          = "%s"
      import_url_username = "__token__"
      import_url_password = "%s"

      mirror = true
    }
  `, testImportedProjectName, testProject.HTTPURLToRepo, tokenForCreate)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
			},
			// Expect empty plan on re-apply
			{
				Config:   createConfig,
				PlanOnly: true,
			},
			// Update the token and trigger a change
			{
				Config: fmt.Sprintf(`
          resource "gitlab_project" "test" {
            name = "imported-%s"

            import_url          = "%s"
            import_url_username = "__token__"
            import_url_password = "%s"

            mirror = true
          }
        `, testImportedProjectName, testProject.HTTPURLToRepo, tokenForUpdate),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"import_url_username", "import_url_password"},
			},
		},
	})
}

func TestAccGitlabProject_initializeWithReadmeAndCustomDefaultBranch(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name        = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"

  initialize_with_readme = true
  default_branch         = "foo"
}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.foo", "initialize_with_readme", "true"),
					resource.TestCheckResourceAttr("gitlab_project.foo", "default_branch", "foo"),
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					testAccCheckGitlabProjectDefaultBranch(&project, &testAccGitlabProjectExpectedAttributes{
						DefaultBranch: "foo",
					}),
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.foo"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, "README.md", &gitlab.GetFileOptions{Ref: gitlab.Ptr("foo")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get 'README.md' file from project: %w", err)
						}

						return nil
					},
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
		},
	})
}

func TestAccGitlabProject_restirctUserDefinedVariables(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "gitlab_project" "foo" {
  name        = "foo-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"

  restrict_user_defined_variables = true
}`, rInt),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_CreateProjectInUserNamespace(t *testing.T) {
	var project gitlab.Project
	rInt := acctest.RandInt()

	user := testutil.CreateUsers(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name              = "foo-%d"
						description       = "Terraform acceptance tests"
						visibility_level  = "public"

						namespace_id = %d
					}
				`, rInt, user.NamespaceID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.foo", &project),
					func(s *terraform.State) error {
						if project.Namespace.ID != user.NamespaceID {
							return fmt.Errorf("project was created in namespace %d but expected %d", project.Namespace.ID, user.NamespaceID)
						}
						return nil
					},
				),
			},
		},
	})
}

// tests to ensure that when skip_wait is set properly, it's injected into the state properly.
// This will also ensure that the `if` check evaluates properly, since it's presence in the
// state means the value is retrieved properly.
func TestAccGitlabProject_skipWaitSetProperly(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "private"
						default_branch   = "main"

						skip_wait_for_default_branch_protection          = true
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "skip_wait_for_default_branch_protection", "true"),
				),
			},
		},
	})
}

// tests to ensure that ci_restrict_pipeline_cancellation_role functions as expected
func TestAccGitlabProject_ciRestrictPipelineCancellationRole(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	// This value is only present in 16.8 and beyond, and only in EE
	testutil.SkipIfCE(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "private"
						default_branch   = "main"

						ci_restrict_pipeline_cancellation_role = "developer"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "ci_restrict_pipeline_cancellation_role", "developer"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "private"
						default_branch   = "main"

						ci_restrict_pipeline_cancellation_role = "maintainer"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "ci_restrict_pipeline_cancellation_role", "maintainer"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
		},
	})
}

// tests to ensure that ci_pipeline_variables_minimum_override_role functions as expected
func TestAccGitlabProject_ciPipelineVariablesMinimumOverrideRole(t *testing.T) {
	// This value is only present in 16.8 and beyond, and only in EE
	testutil.SkipIfCE(t)

	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "private"
						default_branch   = "main"

						ci_pipeline_variables_minimum_override_role = "no_one_allowed"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "ci_pipeline_variables_minimum_override_role", "no_one_allowed"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "private"
						default_branch   = "main"

						ci_pipeline_variables_minimum_override_role = "owner"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "ci_pipeline_variables_minimum_override_role", "owner"),
				),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
		},
	})
}

func TestAccGitlabProject_InstanceBranchProtectionDisabled(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				PreConfig: func() {
					settings, _, err := api.GetSettings(testutil.TestGitlabClient)
					if err != nil {
						t.Fatalf("failed to get settings: %v", err)
					}
					t.Cleanup(func() {
						if _, _, err := api.UpdateSettings(testutil.TestGitlabClient, &gitlab.UpdateSettingsOptions{DefaultBranchProtection: gitlab.Ptr(settings.DefaultBranchProtection)}); err != nil {
							t.Fatalf("failed to update instance-wide default branch protection setting to default: %v", err)
						}
					})

					if _, _, err := api.UpdateSettings(testutil.TestGitlabClient, &gitlab.UpdateSettingsOptions{DefaultBranchProtection: gitlab.Ptr(0)}); err != nil {
						t.Fatalf("failed to update instance-wide default branch protection setting: %v", err)
					}
				},
				Config: ` `, // requires a space for empty config
			},
			// Without explicit default branch
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						initialize_with_readme = true
					}
				`, rInt),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
			// Force a destroy for the project so that it can be recreated as the same resource
			{
				Config: ` `, // requires a space for empty config
			},
			// With explicit default branch set to instance-wide default
			// NOTE(@timofurrer): we create the project with a `-2` suffix,
			// because of the deletion delay, see https://gitlab.com/gitlab-org/gitlab/-/issues/383245
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d-2"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						default_branch         = "main"
						initialize_with_readme = true
					}
				`, rInt),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
			// Force a destroy for the project so that it can be recreated as the same resource
			{
				Config: ` `, // requires a space for empty config
			},
			// With custom default branch
			// NOTE(@timofurrer): we create the project with a `-custom-default-branch` suffix,
			// because of the deletion delay, see https://gitlab.com/gitlab-org/gitlab/-/issues/383245
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d-custom-default-branch"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						default_branch         = "foobar-non-default-branch"
						initialize_with_readme = true
					}
				`, rInt),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
			// Force a destroy for the project so that it can be recreated as the same resource
			{
				Config: ` `, // requires a space for empty config
			},
			// With `skip_wait_for_default_branch_protection` enabled
			// NOTE(@timofurrer): we create the project with a `-custom-default-branch-2` suffix,
			// because of the deletion delay, see https://gitlab.com/gitlab-org/gitlab/-/issues/383245
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d-custom-default-branch-2"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						initialize_with_readme = true

						skip_wait_for_default_branch_protection = true
					}
				`, rInt),
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme", "skip_wait_for_default_branch_protection"},
			},
			// Force a destroy for the project so that it can be recreated as the same resource
			{
				Config: ` `, // requires a space for empty config
			},
			// NOTE(@timofurrer): we create the project with a `-custom-default-branch-3` suffix,
			// because of the deletion delay, see https://gitlab.com/gitlab-org/gitlab/-/issues/383245
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d-custom-default-branch-3"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						initialize_with_readme = true

						skip_wait_for_default_branch_protection = false
					}
				`, rInt),
			},
			// Check if plan is empty after changing `skip_wait_for_default_branch_protection` attribute
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "foo" {
						name                   = "foo-%d-custom-default-branch-3"
						description            = "Terraform acceptance tests"
						visibility_level       = "public"
						initialize_with_readme = true

						skip_wait_for_default_branch_protection = true
					}
				`, rInt),
				PlanOnly: true,
			},
			// Verify Import
			{
				ResourceName:            "gitlab_project.foo",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme", "skip_wait_for_default_branch_protection"},
			},
		},
	})
}

// lintignore: AT002 // not a Terraform import test
func TestAccGitlabProject_ImportURLMirrored(t *testing.T) {
	var mirror gitlab.Project
	rInt := acctest.RandInt()

	// Create a base project for importing.
	baseProject, _, err := testutil.TestGitlabClient.Projects.CreateProject(&gitlab.CreateProjectOptions{
		Name:       gitlab.Ptr(fmt.Sprintf("base-%d", rInt)),
		Visibility: gitlab.Ptr(gitlab.PublicVisibility),
	})
	if err != nil {
		t.Fatalf("failed to create base project: %v", err)
	}

	defer testutil.TestGitlabClient.Projects.DeleteProject(baseProject.ID, nil) // nolint // TODO: Resolve this golangci-lint issue: Error return value of `TestGitlabClient.Projects.DeleteProject` is not checked (errcheck)

	// Add a file to the base project, for later verifying the import.
	_, _, err = testutil.TestGitlabClient.RepositoryFiles.CreateFile(baseProject.ID, "foo.txt", &gitlab.CreateFileOptions{
		Branch:        gitlab.Ptr("main"),
		CommitMessage: gitlab.Ptr("add file"),
		Content:       gitlab.Ptr(""),
	})
	if err != nil {
		t.Fatalf("failed to commit file to base project: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				// First, import, as mirrored
				Config: fmt.Sprintf(`
				resource "gitlab_project" "imported" {
				  name                                = "imported-%d"
				  default_branch                      = "main"
				  import_url                          = "%s"
				  mirror                              = true
				  mirror_trigger_builds               = true
				  mirror_overwrites_diverged_branches = true
				  only_mirror_protected_branches      = true
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, baseProject.HTTPURLToRepo),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.imported", &mirror),
					resource.TestCheckResourceAttr("gitlab_project.imported", "import_url", baseProject.HTTPURLToRepo),
					testAccCheckGitlabProjectMirroredAttributes(&mirror, &testAccGitlabProjectMirroredExpectedAttributes{
						Mirror:                           true,
						MirrorTriggerBuilds:              true,
						MirrorOverwritesDivergedBranches: true,
						OnlyMirrorProtectedBranches:      true,
					}),

					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.imported"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, "foo.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr("main")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get file from imported project: %w", err)
						}

						return nil
					},
				),
			},
			{
				// Second, disable all optional mirroring options
				Config: fmt.Sprintf(`
				resource "gitlab_project" "imported" {
				  name                                = "imported-%d"
				  default_branch                      = "main"
				  import_url                          = "%s"
				  mirror                              = true
				  mirror_trigger_builds               = false
				  mirror_overwrites_diverged_branches = false
				  only_mirror_protected_branches      = false
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, baseProject.HTTPURLToRepo),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.imported", &mirror),
					resource.TestCheckResourceAttr("gitlab_project.imported", "import_url", baseProject.HTTPURLToRepo),
					testAccCheckGitlabProjectMirroredAttributes(&mirror, &testAccGitlabProjectMirroredExpectedAttributes{
						Mirror:                           true,
						MirrorTriggerBuilds:              false,
						MirrorOverwritesDivergedBranches: false,
						OnlyMirrorProtectedBranches:      false,
					}),

					// Ensure the test file still is as expected
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.imported"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, "foo.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr("main")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get file from imported project: %w", err)
						}

						return nil
					},
				),
			},
			{
				// Third, disable mirroring, using the original ImportURL acceptance test
				Config: fmt.Sprintf(`
				resource "gitlab_project" "imported" {
				  name                                = "imported-%d"
				  default_branch                      = "main"
				  import_url                          = "%s"
				  mirror                              = false
				  mirror_trigger_builds               = false
				  mirror_overwrites_diverged_branches = false
				  only_mirror_protected_branches      = false
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, baseProject.HTTPURLToRepo),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.imported", &mirror),
					resource.TestCheckResourceAttr("gitlab_project.imported", "import_url", baseProject.HTTPURLToRepo),
					testAccCheckGitlabProjectMirroredAttributes(&mirror, &testAccGitlabProjectMirroredExpectedAttributes{
						Mirror:                           false,
						MirrorTriggerBuilds:              false,
						MirrorOverwritesDivergedBranches: false,
						OnlyMirrorProtectedBranches:      false,
					}),

					// Ensure the test file still is as expected
					func(state *terraform.State) error {
						projectID := state.RootModule().Resources["gitlab_project.imported"].Primary.ID

						_, _, err := testutil.TestGitlabClient.RepositoryFiles.GetFile(projectID, "foo.txt", &gitlab.GetFileOptions{Ref: gitlab.Ptr("main")}, nil)
						if err != nil {
							return fmt.Errorf("failed to get file from imported project: %w", err)
						}

						return nil
					},
				),
			},
		},
	})
}

func TestAccGitlabProject_templateMutualExclusiveNameAndID(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "template-mutual-exclusive" {
				  name = "template-mutual-exclusive-%d"
				  path = "template-mutual-exclusive.%d"
				  description = "Terraform acceptance tests"
				  template_name = "rails"
				  template_project_id = 999
				  use_custom_template = true
				  default_branch = "master"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				}
				`, rInt, rInt),
				SkipFunc:    testutil.IsRunningInCE,
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(`"template_project_id": conflicts with template_name`)),
			},
		},
	})
}

// Gitlab update project API call requires one from a subset of project fields to be set (See #1157)
// If only a non-blessed field is changed, this test checks that the provider ensures the code won't return an error.
func TestAccGitlabProject_UpdateAnalyticsAccessLevel(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create minimal test project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"
						visibility_level                = "public"
						analytics_access_level = "private"
					}`, rInt),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update `analytics_access_level`
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"
						visibility_level = "public"
						analytics_access_level = "disabled"
					}`, rInt),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_containerExpirationPolicy(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						container_expiration_policy {
							enabled = true
							cadence = "1d"
						}

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.cadence", "1d"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set more attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						container_expiration_policy {
							enabled = true
							cadence = "1month"
							name_regex_keep = "bar"
						}

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.cadence", "1month"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.name_regex_keep", "bar"),
					resource.TestCheckResourceAttrSet("gitlab_project.this", "container_expiration_policy.0.next_run_at"),
				),
			},
			// Clear attributes
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						container_expiration_policy {}

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.cadence", "1month"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.name_regex_keep", "bar"),
					resource.TestCheckResourceAttrSet("gitlab_project.this", "container_expiration_policy.0.next_run_at"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_containerExpirationPolicyRegex(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						container_expiration_policy {
							enabled = true
							cadence = "1d"
							keep_n            = 5
							name_regex_keep   = ""
							older_than        = "7d"
							name_regex_delete = "[0-9a-zA-Z]{40}"
						}

						visibility_level = "public"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.cadence", "1d"),

					// Check that both name_regex values are set properly, since setting one will set them both.
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.name_regex_delete", "[0-9a-zA-Z]{40}"),
					resource.TestCheckResourceAttr("gitlab_project.this", "container_expiration_policy.0.name_regex", "[0-9a-zA-Z]{40}"),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_doubleContainerExpirationPolicyRegexError(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name = "foo-%d"

						container_expiration_policy {
							enabled = true
							cadence = "1d"
							keep_n            = 5
							name_regex_keep   = ""
							older_than        = "7d"
							name_regex_delete = "[0-9a-zA-Z]{40}"
							name_regex        = "[0-9a-zA-Z]{40}"
						}

						visibility_level = "public"
					}`, rInt),
				ExpectError: regexp.MustCompile("Error: Conflicting configuration arguments"),
			},
		},
	})
}

func TestAccGitlabProject_SetDefaultFalseBooleansOnCreate(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "foo-%d"
						visibility_level = "public"

						initialize_with_readme              = false
						resolve_outdated_diff_discussions   = false
						auto_devops_enabled                 = false
						autoclose_referenced_issues         = false
						emails_enabled                      = true
						public_jobs                         = false
						merge_pipelines_enabled             = false
						merge_trains_enabled                = false
						ci_forward_deployment_enabled       = false
						group_runners_enabled               = false
					}`, rInt),
			},
			{
				ResourceName:            "gitlab_project.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"initialize_with_readme"},
			},
		},
	})
}

func TestAccGitlabProject_PublicJobs(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name        = "foo-%d"
						public_jobs = true
					}`, rInt),
			},
		},
	})
}

// Ensure fix for https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/1233
func TestAccGitlabProject_PublicBuilds(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name          = "foo-%d"
						public_builds = true

					}`, rInt),
			},
		},
	})
}

func TestAccGitlabProject_PublicJobsAndPublicBuilds(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`

					resource "gitlab_project" "this" {
						name          = "foo-%d"
						public_jobs   = true
						public_builds = true
					}`, rInt),
				ExpectError: regexp.MustCompile("Error: Conflicting configuration arguments"),
			},
		},
	})
}

func TestAccGitlabProject_ForkProject(t *testing.T) {
	// Create project to fork
	testProjectToFork := testutil.CreateProject(t)
	testProjectToFork2 := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a new `gitlab_project` resource by forking an existing project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
						name                   = "forked-%[1]s"
						path                   = "forked-%[3]s"
						description            = "Forked from %[1]s"
						visibility_level       = "public"

						# fork options
						forked_from_project_id = %[2]d
						mr_default_target_self = true

						# Set some attributes which are not part of the fork API
						build_timeout = 600
				   }
				`, testProjectToFork.Name, testProjectToFork.ID, testProjectToFork.Path),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove fork relationship
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
						name             = "forked-%[1]s"
						path             = "forked-%[3]s"
						description      = "No longer forked from %[1]s"
						visibility_level = "public"

						# Set some attributes which are not part of the fork API
						build_timeout = 660
				   }
				`, testProjectToFork.Name, testProjectToFork.ID, testProjectToFork.Path),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add fork relationship
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
						name                   = "forked-%[1]s"
						path                   = "forked-%[3]s"
						description            = "Forked from %[1]s"
						visibility_level       = "public"

						# fork options
						forked_from_project_id = %[2]d
						mr_default_target_self = false

						# Set some attributes which are not part of the fork API
						build_timeout = 690
				   }
				`, testProjectToFork.Name, testProjectToFork.ID, testProjectToFork.Path),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change fork relationship
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
						name                   = "forked-%[1]s"
						path                   = "forked-%[3]s"
						description            = "Forked from %[1]s"
						visibility_level       = "public"

						# fork options
						forked_from_project_id = %[2]d
						mr_default_target_self = true

						# Set some attributes which are not part of the fork API
						build_timeout = 720
				   }
				`, testProjectToFork.Name, testProjectToFork2.ID, testProjectToFork.Path),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_ForkProjectAndConfigurePullMirror(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create project to fork
	testProjectToFork := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a new `gitlab_project` resource by forking an existing project and configuring the pull mirror
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
						name                   = "forked-%[1]s"
						path                   = "forked-%[3]s"
						description            = "Forked from %[1]s"
						visibility_level       = "public"

						# fork options
						forked_from_project_id = %[2]d

						# Setup Pull mirror
						import_url                          = "%[4]s"
						mirror                              = true
						mirror_trigger_builds               = true
						mirror_overwrites_diverged_branches = true
						only_mirror_protected_branches      = true
				   }
				`, testProjectToFork.Name, testProjectToFork.ID, testProjectToFork.Path, testProjectToFork.HTTPURLToRepo),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_SetBuildsAccessLevel(t *testing.T) {
	var received gitlab.Project
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "this" {
						name             = "testname-%d"
						visibility_level = "public"
						default_branch   = "main"

						builds_access_level = "private"
					}`, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectExists("gitlab_project.this", &received),
					resource.TestCheckResourceAttr("gitlab_project.this", "builds_access_level", "private"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_WithoutAvatarHash(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_project" "test" {
		name             =  "%s"
		visibility_level = "public"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithoutAvatarHash(t, "gitlab_project.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabProjectDestroy
	resource.Test(t, testCase)
}

func TestAccGitlabProject_WithAvatar(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_project" "test" {
		name             =  "%s"
		visibility_level = "public"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithAvatar(t, "gitlab_project.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabProjectDestroy
	resource.Test(t, testCase)
}

// This test checks that a project with `use_custom_template` set to false doesn't encounter
// a 500 internal server error. This is related to a bug in the upstream GitLab API, but is
// something we can handle on the provider side by simply not passing the value in.
// See https://gitlab.com/gitlab-org/terraform-provider-gitlab/-/issues/6154#note_2143274743 for details.
func TestAccGitlabProject_FalseCustomTemplate(t *testing.T) {
	name := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "test" {
					name             =  "%s"
					visibility_level = "public"

					use_custom_template = false
				}
				`, name),
				// We only need to check one attribute; the bug this tests checks causes a 500 internal server error
				// if it hasn't been resolved.
				Check: resource.TestCheckResourceAttr("gitlab_project.test", "name", name),
			},
		},
	})
}

func TestAccGitlabProject_SecretsPushDetection(t *testing.T) {
	testutil.SkipIfCE(t)

	projectName := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project without secrets detection enabled
			{
				Config: fmt.Sprintf(`resource "gitlab_project" "test" {
					name =  "%s"

					pre_receive_secret_detection_enabled = false
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "pre_receive_secret_detection_enabled", "false"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Enable Secrets Detection
			{
				Config: fmt.Sprintf(`resource "gitlab_project" "test" {
								name =  "%s"

								pre_receive_secret_detection_enabled = true
							}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "pre_receive_secret_detection_enabled", "true"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_DeploymentApproverConfig_Create(t *testing.T) {
	// The `allow_pipeline_trigger_approve_deployment` attribute is only
	// available on Premium/Ultimate and from GitLab 15.10 onwards.
	testutil.SkipIfCE(t)

	projectName := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project with config
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "test" {
					name = "%s"

					allow_pipeline_trigger_approve_deployment = true
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "allow_pipeline_trigger_approve_deployment", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update resource to remove config (becomes unmanaged)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "test" {
					name = "%s"
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "allow_pipeline_trigger_approve_deployment", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_DeploymentApproverConfig_Update(t *testing.T) {
	// The `allow_pipeline_trigger_approve_deployment` attribute is only
	// available on Premium/Ultimate and from GitLab 15.10 onwards.
	testutil.SkipIfCE(t)

	projectName := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project without config (defaults to `false`)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "test" {
					name = "%s"
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "allow_pipeline_trigger_approve_deployment", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update resource with config set to `true`
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project" "test" {
					name = "%s"

					allow_pipeline_trigger_approve_deployment = true
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "allow_pipeline_trigger_approve_deployment", "true"),
				),
			},
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProject_RequireJiraIssue(t *testing.T) {
	testutil.SkipIfCE(t)

	projectName := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create a project with prevent_merge_without_jira_issue enabled
			{
				Config: fmt.Sprintf(`resource "gitlab_project" "test" {
					name =  "%s"

					prevent_merge_without_jira_issue = true
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "prevent_merge_without_jira_issue", "true"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disable prevent_merge_without_jira_issue
			{
				Config: fmt.Sprintf(`resource "gitlab_project" "test" {
					name =  "%s"

					prevent_merge_without_jira_issue = false
				}`, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project.test", "prevent_merge_without_jira_issue", "false"),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

type testAccGitlabProjectMirroredExpectedAttributes struct {
	Mirror                           bool
	MirrorTriggerBuilds              bool
	MirrorOverwritesDivergedBranches bool
	OnlyMirrorProtectedBranches      bool
}

func testAccCheckGitlabProjectMirroredAttributes(project *gitlab.Project, want *testAccGitlabProjectMirroredExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if project.Mirror != want.Mirror {
			return fmt.Errorf("got mirror %t; want %t", project.Mirror, want.Mirror)
		}

		if project.MirrorTriggerBuilds != want.MirrorTriggerBuilds {
			return fmt.Errorf("got mirror_trigger_builds %t; want %t", project.MirrorTriggerBuilds, want.MirrorTriggerBuilds)
		}

		if project.MirrorOverwritesDivergedBranches != want.MirrorOverwritesDivergedBranches {
			return fmt.Errorf("got mirror_overwrites_diverged_branches %t; want %t", project.MirrorOverwritesDivergedBranches, want.MirrorOverwritesDivergedBranches)
		}

		if project.OnlyMirrorProtectedBranches != want.OnlyMirrorProtectedBranches {
			return fmt.Errorf("got only_mirror_protected_branches %t; want %t", project.OnlyMirrorProtectedBranches, want.OnlyMirrorProtectedBranches)
		}
		return nil
	}
}

func testAccCheckGitlabProjectExists(n string, project *gitlab.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var err error
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}
		repoName := rs.Primary.ID
		if repoName == "" {
			return fmt.Errorf("No project ID is set")
		}
		if g, _, err := testutil.TestGitlabClient.Projects.GetProject(repoName, nil); err == nil {
			*project = *g
		}
		return err
	}
}

func testAccCheckGitlabProjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}
		gotRepo, resp, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccCheckGitlabProjectArchivedOnDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}

		gotRepo, _, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err != nil {
			return fmt.Errorf("unable to get project %s, to check if it has been archived on the destroy", rs.Primary.ID)
		}

		if !gotRepo.Archived {
			return fmt.Errorf("expected project to be archived, but it isn't")
		}
		return nil
	}

	return fmt.Errorf("no project resources found in state, but expected a `gitlab_project` resource marked as archvied")
}

func testAccCheckAggregateGitlabProject(expected, received *gitlab.Project) resource.TestCheckFunc {
	var checks []resource.TestCheckFunc

	testResource := allResources["gitlab_project"]()
	expectedData := testResource.TestResourceData()
	receivedData := testResource.TestResourceData()
	for a, v := range testResource.Schema {
		attribute := a
		attrValue := v
		checks = append(checks, func(_ *terraform.State) error {
			if attrValue.Computed {
				if attrDefault, err := attrValue.DefaultValue(); err == nil {
					if attrDefault == nil {
						return nil // Skipping because we have no way of pre-computing computed vars
					}
				} else {
					return err
				}
			}

			if err := resourceGitlabProjectSetToState(context.Background(), testutil.TestGitlabClient, expectedData, expected); err != nil {
				return err
			}

			if err := resourceGitlabProjectSetToState(context.Background(), testutil.TestGitlabClient, receivedData, received); err != nil {
				return err
			}

			// ignored for now
			if attribute == "container_expiration_policy" {
				return nil
			}

			return testAccCompareGitLabAttribute(attribute, expectedData, receivedData)
		})
	}
	return resource.ComposeAggregateTestCheckFunc(checks...)
}

// testAccCompareGitLabAttribute compares an attribute in two ResourceData's for
// equivalency.
func testAccCompareGitLabAttribute(attr string, expected, received *schema.ResourceData) error {
	e := expected.Get(attr)
	r := received.Get(attr)
	switch e.(type) { // nolint // TODO: Resolve this golangci-lint issue: S1034: assigning the result of this type assertion to a variable (switch e := e.(type)) could eliminate type assertions in switch cases (gosimple)
	case *schema.Set:
		if !e.(*schema.Set).Equal(r) { // nolint // TODO: Resolve this golangci-lint issue: S1034(related information): could eliminate this type assertion (gosimple)
			return fmt.Errorf(`attribute set %s expected "%+v" received "%+v"`, attr, e, r)
		}
	default:
		// Stringify to check because of type differences
		if fmt.Sprintf("%v", e) != fmt.Sprintf("%v", r) {
			return fmt.Errorf(`attribute %s expected "%+v" received "%+v"`, attr, e, r)
		}
	}
	return nil
}

func testAccCheckGitlabProjectDefaultBranch(project *gitlab.Project, want *testAccGitlabProjectExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if want != nil && project.DefaultBranch != want.DefaultBranch {
			return fmt.Errorf("got default branch %q; want %q", project.DefaultBranch, want.DefaultBranch)
		}

		branches, _, err := testutil.TestGitlabClient.Branches.ListBranches(project.ID, nil)
		if err != nil {
			return fmt.Errorf("failed to list branches: %w", err)
		}

		if len(branches) != 1 {
			return fmt.Errorf("expected 1 branch for new project; found %d", len(branches))
		}

		if !branches[0].Protected {
			return errors.New("expected default branch to be protected")
		}

		return nil
	}
}

func testAccCheckGitlabProjectPushRules(name string, wantPushRules *gitlab.ProjectPushRules) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		projectResource := state.RootModule().Resources[name].Primary

		gotPushRules, _, err := testutil.TestGitlabClient.Projects.GetProjectPushRules(projectResource.ID, nil)
		if err != nil {
			return err
		}

		var messages []string

		if gotPushRules.AuthorEmailRegex != wantPushRules.AuthorEmailRegex {
			messages = append(messages, fmt.Sprintf("author_email_regex (got: %q, wanted: %q)",
				gotPushRules.AuthorEmailRegex, wantPushRules.AuthorEmailRegex))
		}

		if gotPushRules.BranchNameRegex != wantPushRules.BranchNameRegex {
			messages = append(messages, fmt.Sprintf("branch_name_regex (got: %q, wanted: %q)",
				gotPushRules.BranchNameRegex, wantPushRules.BranchNameRegex))
		}

		if gotPushRules.CommitMessageRegex != wantPushRules.CommitMessageRegex {
			messages = append(messages, fmt.Sprintf("commit_message_regex (got: %q, wanted: %q)",
				gotPushRules.CommitMessageRegex, wantPushRules.CommitMessageRegex))
		}

		if gotPushRules.CommitMessageNegativeRegex != wantPushRules.CommitMessageNegativeRegex {
			messages = append(messages, fmt.Sprintf("commit_message_negative_regex (got: %q, wanted: %q)",
				gotPushRules.CommitMessageNegativeRegex, wantPushRules.CommitMessageNegativeRegex))
		}

		if gotPushRules.FileNameRegex != wantPushRules.FileNameRegex {
			messages = append(messages, fmt.Sprintf("file_name_regex (got: %q, wanted: %q)",
				gotPushRules.FileNameRegex, wantPushRules.FileNameRegex))
		}

		if gotPushRules.CommitCommitterCheck != wantPushRules.CommitCommitterCheck {
			messages = append(messages, fmt.Sprintf("commit_committer_check (got: %t, wanted: %t)",
				gotPushRules.CommitCommitterCheck, wantPushRules.CommitCommitterCheck))
		}

		if gotPushRules.CommitCommitterNameCheck != wantPushRules.CommitCommitterNameCheck {
			messages = append(messages, fmt.Sprintf("commit_committer_name_check (got: %t, wanted: %t)",
				gotPushRules.CommitCommitterNameCheck, wantPushRules.CommitCommitterNameCheck))
		}

		if gotPushRules.DenyDeleteTag != wantPushRules.DenyDeleteTag {
			messages = append(messages, fmt.Sprintf("deny_delete_tag (got: %t, wanted: %t)",
				gotPushRules.DenyDeleteTag, wantPushRules.DenyDeleteTag))
		}

		if gotPushRules.MemberCheck != wantPushRules.MemberCheck {
			messages = append(messages, fmt.Sprintf("member_check (got: %t, wanted: %t)",
				gotPushRules.MemberCheck, wantPushRules.MemberCheck))
		}

		if gotPushRules.PreventSecrets != wantPushRules.PreventSecrets {
			messages = append(messages, fmt.Sprintf("prevent_secrets (got: %t, wanted: %t)",
				gotPushRules.PreventSecrets, wantPushRules.PreventSecrets))
		}

		if gotPushRules.RejectUnsignedCommits != wantPushRules.RejectUnsignedCommits {
			messages = append(messages, fmt.Sprintf("reject_unsigned_commits (got: %t, wanted: %t)",
				gotPushRules.RejectUnsignedCommits, wantPushRules.RejectUnsignedCommits))
		}

		if gotPushRules.RejectNonDCOCommits != wantPushRules.RejectNonDCOCommits {
			messages = append(messages, fmt.Sprintf("reject_non_dco_commits (got: %t, wanted: %t)",
				gotPushRules.RejectNonDCOCommits, wantPushRules.RejectNonDCOCommits))
		}

		if gotPushRules.MaxFileSize != wantPushRules.MaxFileSize {
			messages = append(messages, fmt.Sprintf("max_file_size (got: %d, wanted: %d)",
				gotPushRules.MaxFileSize, wantPushRules.MaxFileSize))
		}

		if len(messages) > 0 {
			return fmt.Errorf("unexpected push_rules:\n\t- %s", strings.Join(messages, "\n\t- "))
		}

		return nil
	}
}

func testAccGitlabProjectConfigDefaultBranchSkipFunc(project *gitlab.Project, defaultBranch string) func() (bool, error) {
	return func() (bool, error) {
		// Commit data
		commitMessage := "Initial Commit"
		commitFile := "file.txt"
		commitFileAction := gitlab.FileCreate
		commitActions := []*gitlab.CommitActionOptions{
			{
				Action:   &commitFileAction,
				FilePath: &commitFile,
				Content:  &commitMessage,
			},
		}
		options := &gitlab.CreateCommitOptions{
			Branch:        &defaultBranch,
			CommitMessage: &commitMessage,
			Actions:       commitActions,
		}

		_, _, err := testutil.TestGitlabClient.Commits.CreateCommit(project.ID, options)

		return false, err
	}
}

func testProjectDefaults(rInt int) gitlab.Project {
	return gitlab.Project{
		Namespace:                &gitlab.ProjectNamespace{ID: 0},
		Name:                     fmt.Sprintf("foo-%d", rInt),
		Path:                     fmt.Sprintf("foo.%d", rInt),
		Description:              "Terraform acceptance tests",
		TagList:                  []string{"foo", "bar"},
		RequestAccessEnabled:     true,
		IssuesEnabled:            true,
		MergeRequestsEnabled:     true,
		JobsEnabled:              true,
		ApprovalsBeforeMerge:     0,
		WikiEnabled:              true,
		SnippetsEnabled:          true,
		ContainerRegistryEnabled: true,
		LFSEnabled:               true,
		SharedRunnersEnabled:     true,
		GroupRunnersEnabled:      true,
		Visibility:               gitlab.PublicVisibility,
		MergeMethod:              gitlab.FastForwardMerge,

		OnlyAllowMergeIfPipelineSucceeds:          true,
		OnlyAllowMergeIfAllDiscussionsAreResolved: true,

		SquashOption:                    gitlab.SquashOptionDefaultOff,
		AllowMergeOnSkippedPipeline:     false,
		Archived:                        false, // needless, but let's make this explicit
		PackagesEnabled:                 true,
		PrintingMergeRequestLinkEnabled: true,
		PagesAccessLevel:                gitlab.PublicAccessControl,
		IssuesTemplate:                  "",
		MergeRequestsTemplate:           "",
		CIConfigPath:                    ".gitlab-ci.yml@mynamespace/myproject",
		CIForwardDeploymentEnabled:      true,
		CISeperateCache:                 true,
		KeepLatestArtifact:              true,
		ResolveOutdatedDiffDiscussions:  true,
		AnalyticsAccessLevel:            gitlab.EnabledAccessControl,
		AutoCancelPendingPipelines:      "enabled",
		AutoDevopsDeployStrategy:        "continuous",
		AutoDevopsEnabled:               true,
		AutocloseReferencedIssues:       true,
		BuildGitStrategy:                "fetch",
		BuildTimeout:                    42 * 60,
		BuildsAccessLevel:               gitlab.EnabledAccessControl,
		ContainerExpirationPolicy: &gitlab.ContainerExpirationPolicy{
			Enabled:   true,
			Cadence:   "1month",
			KeepN:     10,
			OlderThan: "10d",
		},
		ContainerRegistryAccessLevel:     gitlab.EnabledAccessControl,
		EmailsDisabled:                   true,
		ForkingAccessLevel:               gitlab.EnabledAccessControl,
		IssuesAccessLevel:                gitlab.EnabledAccessControl,
		MergeRequestsAccessLevel:         gitlab.EnabledAccessControl,
		OperationsAccessLevel:            gitlab.EnabledAccessControl,
		PublicBuilds:                     false,
		RepositoryAccessLevel:            gitlab.EnabledAccessControl,
		RepositoryStorage:                "default",
		SecurityAndComplianceAccessLevel: gitlab.EnabledAccessControl,
		SnippetsAccessLevel:              gitlab.EnabledAccessControl,
		SuggestionCommitMessage:          "hello suggestion",
		Topics:                           []string{"foo", "bar"},
		WikiAccessLevel:                  gitlab.EnabledAccessControl,
		SquashCommitTemplate:             "hello squash",
		MergeCommitTemplate:              "hello merge",
	}
}

func testAccGitLabProjectCreateTemplateProject(t *testing.T, templateFileName string) *gitlab.Project {
	// Create template project in template group called `terraform` (created in `healthcheck-and-setup.sh`).
	templateGroup, _, err := testutil.TestGitlabClient.Groups.GetGroup("terraform", nil)
	if err != nil {
		t.Fatalf("Unable to find template group `terraform` - must be a bug when creating it in `scripts/healthcheck-and-setup.sh`: %+v", err)
	}
	templateProject := testutil.CreateProjectWithNamespace(t, templateGroup.ID)

	// Explicitly unprotect the main branch of the template project if it's protected so we can push a file
	// if 404 is returned, the branch wasn't protected
	_, err = testutil.TestGitlabClient.ProtectedBranches.UnprotectRepositoryBranches(templateProject.ID, templateProject.DefaultBranch, nil)
	if err != nil && !api.Is404(err) {
		t.Fatalf("Unable to unprotect the main branch of the template project: %+v", err)
	}

	// Add a file to the template repository
	testutil.CreateProjectFile(t, templateProject.ID, "meow", templateFileName, templateProject.DefaultBranch)
	return templateProject
}

func Test_constructImportURL(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name              string
		importURL         string
		username          string
		password          string
		expectedImportURL string
	}{
		{
			name:              "Import URL without credentials",
			importURL:         "https://example.com/repo.git",
			username:          "",
			password:          "",
			expectedImportURL: "https://example.com/repo.git",
		},
		{
			name:              "Import URL with credentials",
			importURL:         "https://example.com/repo.git",
			username:          "user",
			password:          "pass",
			expectedImportURL: "https://user:pass@example.com/repo.git",
		},
		{
			name:              "Import URL with credentials and without conflicts",
			importURL:         "https://user:pass@example.com/repo.git",
			username:          "user",
			password:          "pass",
			expectedImportURL: "https://user:pass@example.com/repo.git",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := constructImportUrl(tc.importURL, tc.username, tc.password)
			if err != nil {
				t.Fatal(err)
			}
			if actual != tc.expectedImportURL {
				t.Fatalf("Expected import URL %q, got %q", tc.expectedImportURL, actual)
			}
		})
	}
}

func TestErrors_constructImportURL(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name      string
		importURL string
		username  string
		password  string
	}{
		{
			name:      "Import URL with credentials and with conflicts",
			importURL: "https://user:pass@example.com/repo.git",
			username:  "another-user",
			password:  "pass",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := constructImportUrl(tc.importURL, tc.username, tc.password)
			if err == nil {
				t.Fatal(err)
			}
		})
	}
}
