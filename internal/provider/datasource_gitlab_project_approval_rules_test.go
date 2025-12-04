//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectApprovalRules_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Need to get the current user (usually the admin) because they are automatically added as group members, and we
	// will need the user ID for our assertions later.
	currentUser := testutil.GetCurrentUser(t)

	// Full set up requires at least a project, an user, a group and a protected branch
	project := testutil.CreateProject(t)

	user := testutil.CreateUsers(t, 1)
	testutil.AddProjectMembers(t, project.ID, user)

	group := testutil.CreateGroups(t, 1)
	groupUser := testutil.CreateUsers(t, 1)
	testutil.AddGroupMembers(t, group[0].ID, groupUser)

	// add a sleep since project/group membership is async and takes a while for
	// it to take effect in order for the tests below to pass
	t.Log("Sleeping for 60s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(60 * time.Second)

	protectedBranch := testutil.CreateProtectedBranches(t, project, 1)

	completeRule, err := testutil.CreateProjectApprovalRule(t, project.ID, "Complete Rule", 2, []int64{user[0].ID}, []int64{group[0].ID}, []int64{protectedBranch[0].ID})
	if err != nil {
		t.Fatalf("Failed to create approval rule: %v", err)
	}
	simpleRule, err := testutil.CreateProjectApprovalRule(t, project.ID, "Simple Rule", 1, []int64{}, []int64{}, []int64{})
	if err != nil {
		t.Fatalf("Failed to create approval rule: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_approval_rules" "this" {
						project = %d
					}
					`,
					project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.#", "2"),
					// Complete Rule
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.id", strconv.FormatInt(completeRule.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.name", completeRule.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.rule_type", completeRule.RuleType),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.approvals_required", strconv.FormatInt(completeRule.ApprovalsRequired, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.applies_to_all_protected_branches", strconv.FormatBool(completeRule.AppliesToAllProtectedBranches)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.eligible_approver_ids.#", "3"),
					resource.TestCheckTypeSetElemAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.eligible_approver_ids.*", strconv.FormatInt(currentUser.ID, 10)),
					resource.TestCheckTypeSetElemAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.eligible_approver_ids.*", strconv.FormatInt(groupUser[0].ID, 10)),
					resource.TestCheckTypeSetElemAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.eligible_approver_ids.*", strconv.FormatInt(user[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.group_ids.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.group_ids.0", strconv.FormatInt(group[0].ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.protected_branch_ids.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.0.protected_branch_ids.0", strconv.FormatInt(protectedBranch[0].ID, 10)),
					// Simple Rule
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.id", strconv.FormatInt(simpleRule.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.name", simpleRule.Name),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.rule_type", simpleRule.RuleType),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.approvals_required", strconv.FormatInt(simpleRule.ApprovalsRequired, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.applies_to_all_protected_branches", strconv.FormatBool(simpleRule.AppliesToAllProtectedBranches)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.eligible_approver_ids.#", "0"),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.group_ids.#", "0"),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.1.protected_branch_ids.#", "0"),
				),
			},
		},
	})
}

func TestAccDataGitlabProjectApprovalRules_pagination(t *testing.T) {
	testutil.SkipIfCE(t)

	// Create more than 20 approval rules to test pagination
	project := testutil.CreateProject(t)

	for i := range 25 {
		_, err := testutil.CreateProjectApprovalRule(t, project.ID, fmt.Sprintf("Simple Rule %d", i), 1, []int64{}, []int64{}, []int64{})
		if err != nil {
			t.Fatalf("Failed to create approval rule: %v", err)
		}
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					`
					data "gitlab_project_approval_rules" "this" {
						project = %d
					}
					`,
					project.ID,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("data.gitlab_project_approval_rules.this", "approval_rules.#", "25"),
				),
			},
		},
	})
}
