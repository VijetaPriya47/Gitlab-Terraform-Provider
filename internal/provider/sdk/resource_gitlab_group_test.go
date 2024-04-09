//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabGroup_basic(t *testing.T) {
	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                 fmt.Sprintf("foo-name-%d", rInt),
						Path:                 fmt.Sprintf("foo-path-%d", rInt),
						Description:          "Terraform acceptance tests",
						ProjectCreationLevel: "developer",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to change the description
			{
				Config: testAccGitlabGroupUpdateConfig(rInt, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                    fmt.Sprintf("bar-name-%d", rInt),
						Path:                    fmt.Sprintf("bar-path-%d", rInt),
						Description:             "Terraform acceptance tests! Updated description",
						LFSEnabled:              gitlab.Ptr(false),
						RequestAccessEnabled:    gitlab.Ptr(true),
						RequireTwoFactorAuth:    gitlab.Ptr(true),
						TwoFactorGracePeriod:    gitlab.Ptr(56),
						AutoDevopsEnabled:       gitlab.Ptr(true),
						EmailsDisabled:          gitlab.Ptr(true),
						ShareWithGroupLock:      gitlab.Ptr(true),
						DefaultBranchProtection: gitlab.Ptr(1),
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to use zero-value `default_branch_protection`
			{
				Config: testAccGitlabGroupUpdateConfig(rInt, 0),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                    fmt.Sprintf("bar-name-%d", rInt),
						Path:                    fmt.Sprintf("bar-path-%d", rInt),
						Description:             "Terraform acceptance tests! Updated description",
						LFSEnabled:              gitlab.Ptr(false),
						RequestAccessEnabled:    gitlab.Ptr(true),
						RequireTwoFactorAuth:    gitlab.Ptr(true),
						TwoFactorGracePeriod:    gitlab.Ptr(56),
						AutoDevopsEnabled:       gitlab.Ptr(true),
						EmailsDisabled:          gitlab.Ptr(true),
						ShareWithGroupLock:      gitlab.Ptr(true),
						DefaultBranchProtection: gitlab.Ptr(0),
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to use new value 4 for `default_branch_protection`
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.Background(), testutil.TestGitlabClient, "16.1"),
				Config:   testAccGitlabGroupUpdateConfig(rInt, 4),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                    fmt.Sprintf("bar-name-%d", rInt),
						Path:                    fmt.Sprintf("bar-path-%d", rInt),
						Description:             "Terraform acceptance tests! Updated description",
						LFSEnabled:              gitlab.Ptr(false),
						RequestAccessEnabled:    gitlab.Ptr(true),
						RequireTwoFactorAuth:    gitlab.Ptr(true),
						TwoFactorGracePeriod:    gitlab.Ptr(56),
						AutoDevopsEnabled:       gitlab.Ptr(true),
						EmailsDisabled:          gitlab.Ptr(true),
						ShareWithGroupLock:      gitlab.Ptr(true),
						DefaultBranchProtection: gitlab.Ptr(4),
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to put the name and description back
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("foo-name-%d", rInt),
						Path:        fmt.Sprintf("foo-path-%d", rInt),
						Description: "Terraform acceptance tests",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroup_basicPushRulesEE(t *testing.T) {
	testutil.SkipIfCE(t)

	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                 fmt.Sprintf("foo-name-%d", rInt),
						Path:                 fmt.Sprintf("foo-path-%d", rInt),
						Description:          "Terraform acceptance tests",
						ProjectCreationLevel: "developer",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add all push rules to an existing group, setting 'commit_committer_check' & 'reject_unsigned_commits' to true
			// these attributes are not returned on GitLab versions < 16.4 so skip test if not running >= 16.4
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.TODO(), testutil.TestGitlabClient, "16.4"),
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				    branch_name_regex = "foo_branch"
				    commit_message_regex = "foo_commit"
				    commit_message_negative_regex = "foo_not_commit"
				    file_name_regex = "foo_file"
				    commit_committer_check = true
				    deny_delete_tag = true
				    member_check = true
				    prevent_secrets = true
				    reject_unsigned_commits = true
				    max_file_size = 123
				  }
				}
					`, rInt, rInt),

				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file",
					CommitCommitterCheck:       gitlab.Ptr(true),
					DenyDeleteTag:              gitlab.Ptr(true),
					MemberCheck:                gitlab.Ptr(true),
					PreventSecrets:             gitlab.Ptr(true),
					RejectUnsignedCommits:      gitlab.Ptr(true),
					MaxFileSize:                gitlab.Ptr(123),
				}),
			},
			// Test import with a all push rules defined (checks read function)
			{
				SkipFunc:          api.IsGitLabVersionLessThan(context.TODO(), testutil.TestGitlabClient, "16.4"),
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update some push rules but not others, setting 'commit_committer_check' & 'reject_unsigned_commits' to true
			// these attributes are not returned on GitLab versions < 16.4 so skip test if not running >= 16.4
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.TODO(), testutil.TestGitlabClient, "16.4"),
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				    branch_name_regex = "foo_branch"
				    commit_message_regex = "foo_commit"
				    commit_message_negative_regex = "foo_not_commit"
				    file_name_regex = "foo_file_2"
				    commit_committer_check = true
				    deny_delete_tag = true
				    member_check = false
				    prevent_secrets = true
				    reject_unsigned_commits = true
				    max_file_size = 1234
				  }
				}
					`, rInt, rInt),

				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file_2",
					CommitCommitterCheck:       gitlab.Ptr(true),
					DenyDeleteTag:              gitlab.Ptr(true),
					MemberCheck:                gitlab.Ptr(false),
					PreventSecrets:             gitlab.Ptr(true),
					RejectUnsignedCommits:      gitlab.Ptr(true),
					MaxFileSize:                gitlab.Ptr(1234),
				}),
			},
			// Add all push rules to an existing group, 'commit_committer_check' & 'reject_unsigned_commits' set to false
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				    branch_name_regex = "foo_branch"
				    commit_message_regex = "foo_commit"
				    commit_message_negative_regex = "foo_not_commit"
				    file_name_regex = "foo_file"
				    commit_committer_check = false
				    deny_delete_tag = true
				    member_check = true
				    prevent_secrets = true
				    reject_unsigned_commits = false
				    max_file_size = 123
				  }
				}
					`, rInt, rInt),

				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file",
					CommitCommitterCheck:       gitlab.Ptr(false),
					DenyDeleteTag:              gitlab.Ptr(true),
					MemberCheck:                gitlab.Ptr(true),
					PreventSecrets:             gitlab.Ptr(true),
					RejectUnsignedCommits:      gitlab.Ptr(false),
					MaxFileSize:                gitlab.Ptr(123),
				}),
			},
			// Test import with a all push rules defined (checks read function)
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update some push rules but not others, 'commit_committer_check' & 'reject_unsigned_commits' set to false
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				    branch_name_regex = "foo_branch"
				    commit_message_regex = "foo_commit"
				    commit_message_negative_regex = "foo_not_commit"
				    file_name_regex = "foo_file_2"
				    commit_committer_check = false
				    deny_delete_tag = true
				    member_check = false
				    prevent_secrets = true
				    reject_unsigned_commits = false
				    max_file_size = 1234
				  }
				}
					`, rInt, rInt),

				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex:           "foo_author",
					BranchNameRegex:            "foo_branch",
					CommitMessageRegex:         "foo_commit",
					CommitMessageNegativeRegex: "foo_not_commit",
					FileNameRegex:              "foo_file_2",
					CommitCommitterCheck:       gitlab.Ptr(false),
					DenyDeleteTag:              gitlab.Ptr(true),
					MemberCheck:                gitlab.Ptr(false),
					PreventSecrets:             gitlab.Ptr(true),
					RejectUnsignedCommits:      gitlab.Ptr(false),
					MaxFileSize:                gitlab.Ptr(1234),
				}),
			},
			// Update push rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				  }
				}
					`, rInt, rInt),
				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex: "foo_author",
				}),
			},
			// Remove the push_rules block entirely.
			// NOTE: The push rules will still exist upstream because the push_rules block is computed.
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					AuthorEmailRegex: "foo_author",
				}),
			},
			// Add different push rules after the block was removed previously
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    branch_name_regex = "(feature|hotfix)\\/*"
				  }
				}
					`, rInt, rInt),
				Check: testAccCheckGitlabGroupPushRules("gitlab_group.foo", &testAccGitlabGroupPushRuleExpectedAttributes{
					BranchNameRegex: `(feature|hotfix)\/*`,
				}),
			},
		},
	})
}

func TestAccGitlabGroup_basicPushRulesCE(t *testing.T) {
	testutil.SkipIfEE(t)

	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				Config: testAccGitlabGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                 fmt.Sprintf("foo-name-%d", rInt),
						Path:                 fmt.Sprintf("foo-path-%d", rInt),
						Description:          "Terraform acceptance tests",
						ProjectCreationLevel: "developer",
					}),
				),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Try to add push rules to an existing group in CE
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group" "foo" {
				  name = "foo-name-%d"
				  path = "foo-path-%d"
				  description = "Terraform acceptance tests"
				
				  # So that acceptance tests can be run in a gitlab organization
				  # with no billing
				  visibility_level = "public"
				
				  push_rules {
				    author_email_regex = "foo_author"
				  }
				}
					`, rInt, rInt),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("Group push rules are not supported in your version of GitLab")),
			},
		},
	})
}

func TestAccGitlabGroup_IPRestricted(t *testing.T) {
	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_group" "this" {
					name = "test-ip-restrictions-%d"
					path = "path-%d"

					ip_restriction_ranges = ["192.168.0.0/24"]
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.this", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                fmt.Sprintf("test-ip-restrictions-%d", rInt),
						Path:                fmt.Sprintf("path-%d", rInt),
						IPRestrictionRanges: "192.168.0.0/24",
					}),
				),
			},
			// Verify Import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to generate a comma in the ranges
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_group" "this" {
					name = "test-ip-restrictions-%d"
					path = "path-%d"

					ip_restriction_ranges = ["192.168.0.0/24", "10.1.0.0/24"]
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.this", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                fmt.Sprintf("test-ip-restrictions-%d", rInt),
						Path:                fmt.Sprintf("path-%d", rInt),
						IPRestrictionRanges: "192.168.0.0/24,10.1.0.0/24",
					}),
				),
			},
			// Verify Import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group back to unrestricted
			{
				SkipFunc: testutil.IsRunningInCE,
				Config: fmt.Sprintf(`
				resource "gitlab_group" "this" {
					name = "test-ip-restrictions-%d"
					path = "path-%d"

					ip_restriction_ranges = []
				}
				`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.this", &group),
					testAccCheckGitlabGroupAttributes(&group, &testAccGitlabGroupExpectedAttributes{
						Name:                fmt.Sprintf("test-ip-restrictions-%d", rInt),
						Path:                fmt.Sprintf("path-%d", rInt),
						IPRestrictionRanges: "",
					}),
				),
			},
			// Verify Import
			{
				SkipFunc:          testutil.IsRunningInCE,
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroup_nested(t *testing.T) {
	var group gitlab.Group
	var group2 gitlab.Group
	var nestedGroup gitlab.Group
	var lastGid int
	testGidNotChanged := func(s *terraform.State) error {
		if lastGid == 0 {
			lastGid = nestedGroup.ID
		}
		if lastGid != nestedGroup.ID {
			return fmt.Errorf("group id changed")
		}
		return nil
	}
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGitlabNestedGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupExists("gitlab_group.foo2", &group2),
					testAccCheckGitlabGroupExists("gitlab_group.nested_foo", &nestedGroup),
					testAccCheckGitlabGroupAttributes(&nestedGroup, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("nfoo-name-%d", rInt),
						Path:        fmt.Sprintf("nfoo-path-%d", rInt),
						Description: "Terraform acceptance tests",
						Parent:      &group,
					}),
					testGidNotChanged,
				),
			},
			{
				Config: testAccGitlabNestedGroupChangeParentConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupExists("gitlab_group.foo2", &group2),
					testAccCheckGitlabGroupExists("gitlab_group.nested_foo", &nestedGroup),
					testAccCheckGitlabGroupAttributes(&nestedGroup, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("nfoo-name-%d", rInt),
						Path:        fmt.Sprintf("nfoo-path-%d", rInt),
						Description: "Terraform acceptance tests - new parent",
						Parent:      &group2,
					}),
					testGidNotChanged,
				),
			},
			{
				Config: testAccGitlabNestedGroupRemoveParentConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupExists("gitlab_group.foo2", &group2),
					testAccCheckGitlabGroupExists("gitlab_group.nested_foo", &nestedGroup),
					testAccCheckGitlabGroupAttributes(&nestedGroup, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("nfoo-name-%d", rInt),
						Path:        fmt.Sprintf("nfoo-path-%d", rInt),
						Description: "Terraform acceptance tests - updated",
						Parent:      &group2,
					}),
					testGidNotChanged,
				),
			},
			{
				Config: testAccGitlabNestedGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					testAccCheckGitlabGroupExists("gitlab_group.foo2", &group2),
					testAccCheckGitlabGroupExists("gitlab_group.nested_foo", &nestedGroup),
					testAccCheckGitlabGroupAttributes(&nestedGroup, &testAccGitlabGroupExpectedAttributes{
						Name:        fmt.Sprintf("nfoo-name-%d", rInt),
						Path:        fmt.Sprintf("nfoo-path-%d", rInt),
						Description: "Terraform acceptance tests",
						Parent:      &group,
					}),
					testGidNotChanged,
				),
			},
		},
	})
}

func TestAccGitlabGroup_EE(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroupName := acctest.RandomWithPrefix("acctest-group")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group" "this" {
						name = "%[1]s"
						path = "%[1]s"

						membership_lock                    = true
						extra_shared_runners_minutes_limit = 21
						shared_runners_minutes_limit       = 42
					}
				`, testGroupName),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group" "this" {
						name = "%[1]s"
						path = "%[1]s"

						membership_lock                    = false
						extra_shared_runners_minutes_limit = 0
						shared_runners_minutes_limit       = 0
						wiki_access_level                  = "disabled"
					}
				`, testGroupName),
			},
			// Verify import
			{
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroup_PreventForkingOutsideGroup(t *testing.T) {
	var group gitlab.Group
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   testAccGitlabGroupPreventForkingOutsideGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					resource.TestCheckResourceAttr("gitlab_group.foo", "prevent_forking_outside_group", "true"),
				),
			},
			{
				SkipFunc: testutil.IsRunningInCE,
				Config:   testAccGitlabGroupPreventForkingOutsideGroupUpdateConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupExists("gitlab_group.foo", &group),
					resource.TestCheckResourceAttr("gitlab_group.foo", "prevent_forking_outside_group", "false"),
				),
			},
		},
	})
}

func TestAccGitlabGroup_SetDefaultFalseBooleansOnCreate(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group" "this" {
						name             = "foo-%d"
						path             = "path-%d"
						visibility_level = "public"

						require_two_factor_authentication = false
						auto_devops_enabled               = false
						emails_enabled                    = true
						mentions_disabled                 = false
						prevent_forking_outside_group     = false
					}`, rInt, rInt),
			},
			{
				ResourceName:      "gitlab_group.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabGroup_WithoutAvatarHash(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_group" "test" {
		name             =  "%[1]s"
		path             =  "%[1]s"
		visibility_level = "public"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithoutAvatarHash(t, "gitlab_group.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabGroupDestroy
	resource.Test(t, testCase)
}

func TestAccGitlabGroup_WithAvatar(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_group" "test" {
		name             =  "%[1]s"
		path             =  "%[1]s"
		visibility_level = "public"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithAvatar(t, "gitlab_group.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabGroupDestroy
	resource.Test(t, testCase)
}

func TestAccGitlabGroup_sharedRunnersSetting(t *testing.T) {
	rInt := acctest.RandInt()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabGroupDestroy,
		Steps: []resource.TestStep{
			// Create a group
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group" "foo" {
					  name = "foo-name-%d"
					  path = "foo-path-%d"
					  description = "Terraform acceptance tests"
					  shared_runners_setting = "disabled_and_unoverridable"
					
					  # So that acceptance tests can be run in a gitlab organization
					  # with no billing
					  visibility_level = "public"
					}`, rInt, rInt),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the group to change the shared_runners_setting
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group" "foo" {
					  name = "foo-name-%d"
					  path = "foo-path-%d"
					  description = "Terraform acceptance tests"
					  shared_runners_setting = "enabled"
			
					  # So that acceptance tests can be run in a gitlab organization
					  # with no billing
					  visibility_level = "public"
					}`, rInt, rInt),
			},
			// Verify Import
			{
				ResourceName:      "gitlab_group.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabGroupExists(n string, group *gitlab.Group) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		groupID := rs.Primary.ID
		if groupID == "" {
			return fmt.Errorf("No group ID is set")
		}

		gotGroup, _, err := testutil.TestGitlabClient.Groups.GetGroup(groupID, nil)
		if err != nil {
			return err
		}
		*group = *gotGroup
		return nil
	}
}

type testAccGitlabGroupExpectedAttributes struct {
	Name                    string
	Path                    string
	Description             string
	Parent                  *gitlab.Group
	LFSEnabled              *bool
	RequestAccessEnabled    *bool
	Visibility              gitlab.VisibilityValue
	ShareWithGroupLock      *bool
	AutoDevopsEnabled       *bool
	EmailsDisabled          *bool
	EmailsEnabled           *bool
	MentionsDisabled        *bool
	ProjectCreationLevel    gitlab.ProjectCreationLevelValue
	SubGroupCreationLevel   gitlab.SubGroupCreationLevelValue
	RequireTwoFactorAuth    *bool
	TwoFactorGracePeriod    *int
	DefaultBranchProtection *int
	IPRestrictionRanges     string
}

func testAccCheckGitlabGroupAttributes(group *gitlab.Group, want *testAccGitlabGroupExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if group.Name != want.Name {
			return fmt.Errorf("got repo %q; want %q", group.Name, want.Name)
		}

		if group.Path != want.Path {
			return fmt.Errorf("got path %q; want %q", group.Path, want.Path)
		}

		if group.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", group.Description, want.Description)
		}

		if want.LFSEnabled != nil && group.LFSEnabled != *want.LFSEnabled {
			return fmt.Errorf("got lfs_enabled %t; want %t", group.LFSEnabled, *want.LFSEnabled)
		}

		if want.Visibility != "" && group.Visibility != want.Visibility {
			return fmt.Errorf("got request_visibility_level: %q; want %q", group.Visibility, want.Visibility)
		}

		if want.AutoDevopsEnabled != nil && group.AutoDevopsEnabled != *want.AutoDevopsEnabled {
			return fmt.Errorf("got request_auto_devops_enabled: %t; want %t", group.AutoDevopsEnabled, *want.AutoDevopsEnabled)
		}

		// nolint:staticcheck // SA1019 ignore deprecated EmailsDisabled
		if want.EmailsDisabled != nil && group.EmailsDisabled != *want.EmailsDisabled {

			// nolint:staticcheck // SA1019 ignore deprecated EmailsDisabled
			return fmt.Errorf("got request_emails_disabled: %t; want %t", group.EmailsDisabled, *want.EmailsDisabled)
		}

		if want.EmailsEnabled != nil && group.EmailsEnabled != *want.EmailsEnabled {
			return fmt.Errorf("got request_emails_enabled: %t; want %t", group.EmailsEnabled, *want.EmailsEnabled)
		}

		if want.MentionsDisabled != nil && group.MentionsDisabled != *want.MentionsDisabled {
			return fmt.Errorf("got request_mentions_disabled: %t; want %t", group.MentionsDisabled, *want.MentionsDisabled)
		}

		if want.RequestAccessEnabled != nil && group.RequestAccessEnabled != *want.RequestAccessEnabled {
			return fmt.Errorf("got request_access_enabled %t; want %t", group.RequestAccessEnabled, *want.RequestAccessEnabled)
		}

		if want.ProjectCreationLevel != "" && group.ProjectCreationLevel != want.ProjectCreationLevel {
			return fmt.Errorf("got project_creation_level %s; want %s", group.ProjectCreationLevel, want.ProjectCreationLevel)
		}

		if want.SubGroupCreationLevel != "" && group.SubGroupCreationLevel != want.SubGroupCreationLevel {
			return fmt.Errorf("got subgroup_creation_level %s; want %s", group.SubGroupCreationLevel, want.SubGroupCreationLevel)
		}

		if want.RequireTwoFactorAuth != nil && group.RequireTwoFactorAuth != *want.RequireTwoFactorAuth {
			return fmt.Errorf("got require_two_factor_authentication %t; want %t", group.RequireTwoFactorAuth, *want.RequireTwoFactorAuth)
		}

		if want.TwoFactorGracePeriod != nil && group.TwoFactorGracePeriod != *want.TwoFactorGracePeriod {
			return fmt.Errorf("got two_factor_grace_period %d; want %d", group.TwoFactorGracePeriod, *want.TwoFactorGracePeriod)
		}

		if want.ShareWithGroupLock != nil && group.ShareWithGroupLock != *want.ShareWithGroupLock {
			return fmt.Errorf("got share_with_group_lock %t; want %t", group.ShareWithGroupLock, *want.ShareWithGroupLock)
		}

		if want.DefaultBranchProtection != nil && group.DefaultBranchProtection != *want.DefaultBranchProtection {
			return fmt.Errorf("got default_branch_protection %d; want %d", group.DefaultBranchProtection, *want.DefaultBranchProtection)
		}

		if group.IPRestrictionRanges != want.IPRestrictionRanges {
			return fmt.Errorf("got ip_restriction_ranges %s; want %s", group.IPRestrictionRanges, want.IPRestrictionRanges)
		}

		if want.Parent != nil {
			if group.ParentID != want.Parent.ID {
				return fmt.Errorf("got parent_id %d; want %d", group.ParentID, want.Parent.ID)
			}
		} else {
			if group.ParentID != 0 {
				return fmt.Errorf("got parent_id %d; want %d", group.ParentID, 0)
			}
		}

		return nil
	}
}

func testAccCheckGitlabGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group" {
			continue
		}

		group, _, err := testutil.TestGitlabClient.Groups.GetGroup(rs.Primary.ID, nil)
		if err == nil {
			if group != nil && fmt.Sprintf("%d", group.ID) == rs.Primary.ID {
				if group.MarkedForDeletionOn == nil {
					return fmt.Errorf("Group still exists")
				}
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

type testAccGitlabGroupPushRuleExpectedAttributes struct {
	CommitMessageRegex         string
	CommitMessageNegativeRegex string
	BranchNameRegex            string
	DenyDeleteTag              *bool
	MemberCheck                *bool
	PreventSecrets             *bool
	AuthorEmailRegex           string
	FileNameRegex              string
	MaxFileSize                *int
	CommitCommitterCheck       *bool
	RejectUnsignedCommits      *bool
}

func testAccCheckGitlabGroupPushRules(name string, wantPushRules *testAccGitlabGroupPushRuleExpectedAttributes) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not Found: %s", name)
		}

		gotPushRules, _, err := testutil.TestGitlabClient.Groups.GetGroupPushRules(rs.Primary.ID, nil)
		if err != nil {
			return err
		}

		var messages []string

		if wantPushRules.AuthorEmailRegex != "" && gotPushRules.AuthorEmailRegex != wantPushRules.AuthorEmailRegex {
			messages = append(messages, fmt.Sprintf("author_email_regex (got: %q, wanted: %q)",
				gotPushRules.AuthorEmailRegex, wantPushRules.AuthorEmailRegex))
		}

		if wantPushRules.BranchNameRegex != "" && gotPushRules.BranchNameRegex != wantPushRules.BranchNameRegex {
			messages = append(messages, fmt.Sprintf("branch_name_regex (got: %q, wanted: %q)",
				gotPushRules.BranchNameRegex, wantPushRules.BranchNameRegex))
		}

		if wantPushRules.CommitMessageRegex != "" && gotPushRules.CommitMessageRegex != wantPushRules.CommitMessageRegex {
			messages = append(messages, fmt.Sprintf("commit_message_regex (got: %q, wanted: %q)",
				gotPushRules.CommitMessageRegex, wantPushRules.CommitMessageRegex))
		}

		if wantPushRules.CommitMessageNegativeRegex != "" && gotPushRules.CommitMessageNegativeRegex != wantPushRules.CommitMessageNegativeRegex {
			messages = append(messages, fmt.Sprintf("commit_message_negative_regex (got: %q, wanted: %q)",
				gotPushRules.CommitMessageNegativeRegex, wantPushRules.CommitMessageNegativeRegex))
		}

		if wantPushRules.FileNameRegex != "" && gotPushRules.FileNameRegex != wantPushRules.FileNameRegex {
			messages = append(messages, fmt.Sprintf("file_name_regex (got: %q, wanted: %q)",
				gotPushRules.FileNameRegex, wantPushRules.FileNameRegex))
		}

		if wantPushRules.CommitCommitterCheck != nil && gotPushRules.CommitCommitterCheck != *wantPushRules.CommitCommitterCheck {
			messages = append(messages, fmt.Sprintf("commit_committer_check (got: %t, wanted: %t)",
				gotPushRules.CommitCommitterCheck, *wantPushRules.CommitCommitterCheck))
		}

		if wantPushRules.DenyDeleteTag != nil && gotPushRules.DenyDeleteTag != *wantPushRules.DenyDeleteTag {
			messages = append(messages, fmt.Sprintf("deny_delete_tag (got: %t, wanted: %t)",
				gotPushRules.DenyDeleteTag, *wantPushRules.DenyDeleteTag))
		}

		if wantPushRules.MemberCheck != nil && gotPushRules.MemberCheck != *wantPushRules.MemberCheck {
			messages = append(messages, fmt.Sprintf("member_check (got: %t, wanted: %t)",
				gotPushRules.MemberCheck, *wantPushRules.MemberCheck))
		}

		if wantPushRules.PreventSecrets != nil && gotPushRules.PreventSecrets != *wantPushRules.PreventSecrets {
			messages = append(messages, fmt.Sprintf("prevent_secrets (got: %t, wanted: %t)",
				gotPushRules.PreventSecrets, *wantPushRules.PreventSecrets))
		}

		if wantPushRules.RejectUnsignedCommits != nil && gotPushRules.RejectUnsignedCommits != *wantPushRules.RejectUnsignedCommits {
			messages = append(messages, fmt.Sprintf("reject_unsigned_commits (got: %t, wanted: %t)",
				gotPushRules.RejectUnsignedCommits, *wantPushRules.RejectUnsignedCommits))
		}

		if wantPushRules.MaxFileSize != nil && gotPushRules.MaxFileSize != *wantPushRules.MaxFileSize {
			messages = append(messages, fmt.Sprintf("max_file_size (got: %d, wanted: %d)",
				gotPushRules.MaxFileSize, *wantPushRules.MaxFileSize))
		}

		if len(messages) > 0 {
			return fmt.Errorf("unexpected push_rules:\n\t- %s", strings.Join(messages, "\n\t- "))
		}

		return nil
	}
}

func testAccGitlabGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt)
}

func testAccGitlabGroupUpdateConfig(rInt int, defaultBranchProtection int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "bar-name-%d"
  path = "bar-path-%d"
  description = "Terraform acceptance tests! Updated description"
  lfs_enabled = false
  request_access_enabled = true
  project_creation_level = "developer"
  subgroup_creation_level = "maintainer"
  require_two_factor_authentication = true
  two_factor_grace_period = 56
  auto_devops_enabled = true
  emails_enabled = false
  mentions_disabled = true
  share_with_group_lock = true
  default_branch_protection = %d

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt, defaultBranchProtection)
}

func testAccGitlabNestedGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "foo2" {
  name = "foo2-name-%d"
  path = "foo2-path-%d"
  description = "Terraform acceptance tests - parent2"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "nested_foo" {
  name = "nfoo-name-%d"
  path = "nfoo-path-%d"
  parent_id = "${gitlab_group.foo.id}"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccGitlabNestedGroupRemoveParentConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "foo2" {
  name = "foo2-name-%d"
  path = "foo2-path-%d"
  description = "Terraform acceptance tests - parent2"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "nested_foo" {
  name = "nfoo-name-%d"
  path = "nfoo-path-%d"
  description = "Terraform acceptance tests - updated"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccGitlabNestedGroupChangeParentConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "foo2" {
  name = "foo2-name-%d"
  path = "foo2-path-%d"
  description = "Terraform acceptance tests - parent2"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
resource "gitlab_group" "nested_foo" {
  name = "nfoo-name-%d"
  path = "nfoo-path-%d"
  description = "Terraform acceptance tests - new parent"
  parent_id = "${gitlab_group.foo2.id}"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"
}
  `, rInt, rInt, rInt, rInt, rInt, rInt)
}

func testAccGitlabGroupPreventForkingOutsideGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"

  prevent_forking_outside_group = true
}
  `, rInt, rInt)
}

func testAccGitlabGroupPreventForkingOutsideGroupUpdateConfig(rInt int) string {
	return fmt.Sprintf(`
resource "gitlab_group" "foo" {
  name = "foo-name-%d"
  path = "foo-path-%d"
  description = "Terraform acceptance tests"

  # So that acceptance tests can be run in a gitlab organization
  # with no billing
  visibility_level = "public"

  prevent_forking_outside_group = false
}
  `, rInt, rInt)
}
