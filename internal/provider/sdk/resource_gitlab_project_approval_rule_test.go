//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestAccGitLabProjectApprovalRule_Basic(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.

	testutil.SkipIfCE(t)

	// Need to get the current user (usually the admin) because they are automatically added as group members, and we
	// will need the user ID for our assertions later.
	currentUser := testutil.GetCurrentUser(t)

	project := testutil.CreateProject(t)
	projectUsers := testutil.CreateUsers(t, 2)
	branches := testutil.CreateProtectedBranches(t, project, 2)
	groups := testutil.CreateGroups(t, 2)
	group0Users := testutil.CreateUsers(t, 1)
	group1Users := testutil.CreateUsers(t, 1)

	testutil.AddProjectMembers(t, project.ID, projectUsers) // Users must belong to the project for rules to work.
	testutil.AddGroupMembers(t, groups[0].ID, group0Users)
	testutil.AddGroupMembers(t, groups[1].ID, group1Users)

	// Terraform test starts here.

	var projectApprovalRule gitlab.ProjectApprovalRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "foo" {
						project              = %d
						name                 = "foo"
						approvals_required   = %d
						user_ids             = [%d]
						group_ids            = [%d]
						protected_branch_ids = [%d]
					}
				`, project.ID, 3, projectUsers[0].ID, groups[0].ID, branches[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.foo", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_Basic(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_Basic{
						Name:                "foo",
						ApprovalsRequired:   3,
						EligibleApproverIDs: []int{currentUser.ID, projectUsers[0].ID, group0Users[0].ID},
						GroupIDs:            []int{groups[0].ID},
						ProtectedBranchIDs:  []int{branches[0].ID},
					}),
				),
			},
			// Update rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "foo" {
						project              = %d
						name                 = "foo"
						approvals_required   = %d
						user_ids             = [%d]
						group_ids            = [%d]
						protected_branch_ids = [%d]
					}
				`, project.ID, 2, projectUsers[1].ID, groups[1].ID, branches[1].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.foo", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_Basic(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_Basic{
						Name:                "foo",
						ApprovalsRequired:   2,
						EligibleApproverIDs: []int{currentUser.ID, projectUsers[1].ID, group1Users[0].ID},
						GroupIDs:            []int{groups[1].ID},
						ProtectedBranchIDs:  []int{branches[1].ID},
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_approval_rule.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disable_importing_default_any_approver_rule_on_create",
				},
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "bar" {
				  project              = %d
				  name                 = "bar"
				  approvals_required   = %d
				  user_ids             = [%d]
				  group_ids            = [%d]
				  applies_to_all_protected_branches = true
				}`, project.ID, 3, projectUsers[0].ID, groups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_Basic(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_Basic{
						Name:                          "bar",
						ApprovalsRequired:             3,
						EligibleApproverIDs:           []int{currentUser.ID, projectUsers[0].ID, group0Users[0].ID},
						GroupIDs:                      []int{groups[0].ID},
						AppliesToAllProtectedBranches: true,
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_approval_rule.bar",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disable_importing_default_any_approver_rule_on_create",
				},
			},
		},
	})
}

func TestAccGitLabProjectApprovalRule_AnyApprover(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.

	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)

	// Terraform test starts here.

	var projectApprovalRule gitlab.ProjectApprovalRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "bar" {
						project              = %d
						name                 = "bar"
						approvals_required   = %d
						rule_type            = "%s"
					}
				`, project.ID, 3, "any_approver"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover{
						Name:              "bar",
						ApprovalsRequired: 3,
						RuleType:          "any_approver",
					}),
				),
			},
			// Update rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "bar" {
						project              = %d
						name                 = "bar"
						approvals_required   = %d
						rule_type            = "%s"
					}
				`, project.ID, 2, "any_approver"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover{
						Name:              "bar",
						ApprovalsRequired: 2,
						RuleType:          "any_approver",
					}),
				),
			},
			// Re-create rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "bar" {
						project              = %d
						name                 = "bar"
						approvals_required   = %d
						rule_type            = "%s"
					}
				`, project.ID, 2, "regular"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover{
						Name:              "bar",
						ApprovalsRequired: 2,
						RuleType:          "regular",
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_approval_rule.bar",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disable_importing_default_any_approver_rule_on_create",
				},
			},
		},
	})
}

func TestAccGitLabProjectApprovalRule_ReportType(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "17.2")

	project := testutil.CreateProject(t)

	projectUsers := testutil.CreateUsers(t, 2)
	groups := testutil.CreateGroups(t, 1)
	group0Users := testutil.CreateUsers(t, 1)

	testutil.AddProjectMembers(t, project.ID, projectUsers) // Users must belong to the project for rules to work.
	testutil.AddGroupMembers(t, groups[0].ID, group0Users)

	// Terraform test starts here.
	var projectApprovalRule gitlab.ProjectApprovalRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "rt" {
					project              = %d
					name                 = "Coverage-Check"
					approvals_required   = 1
					rule_type            = "report_approver"
					report_type          = "code_coverage"
					group_ids            = [ %d ]
				}`, project.ID, groups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.rt", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_ReportType(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_ReportType{
						Name:              "Coverage-Check",
						ApprovalsRequired: 1,
						RuleType:          "report_approver",
						ReportType:        "code_coverage",
					}),
				),
			},
			// Update rule
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "rt" {
					project              = %d
					name                 = "Coverage-Check"
					approvals_required   = 2
					rule_type            = "report_approver"
					report_type          = "code_coverage"
					group_ids            = [ %d ]
				  }`, project.ID, groups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.rt", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_ReportType(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_ReportType{
						Name:              "Coverage-Check",
						ApprovalsRequired: 2,
						RuleType:          "report_approver",
						ReportType:        "code_coverage",
					}),
				),
			},
			// Re-create rule
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "rt" {
					project              = %d
					name                 = "Coverage-Check"
					approvals_required   = 2
					rule_type            = "report_approver"
					report_type          = "code_coverage"
					group_ids            = [ %d ]
				}`, project.ID, groups[0].ID),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.rt", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_ReportType(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_ReportType{
						Name:              "Coverage-Check",
						ApprovalsRequired: 2,
						RuleType:          "report_approver",
						ReportType:        "code_coverage",
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project_approval_rule.rt",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disable_importing_default_any_approver_rule_on_create",
				},
			},
		},
	})
}

func TestAccGitLabProjectApprovalRule_ReportTypeMisConfigured(t *testing.T) {
	// Set up project and branches to use in the test.
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "17.2")

	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create needs correct 'rule_type'
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "bar" {
				  project              = %d
				  name                 = "Coverage-Check"
				  approvals_required   = %d
				  rule_type            = "regular"
				  report_type          = "code_coverage"
				}`, project.ID, 3),
				ExpectError: regexp.MustCompile("rule_type incorrect"),
			},
			// If 'rule_type' == report_approver then name should be Coverage-Check
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "bar" {
					project              = %d
					name                 = "foo"
					approvals_required   = %d
					rule_type            = "report_approver"
					report_type          = "code_coverage"
				}`, project.ID, 3),
				ExpectError: regexp.MustCompile("name should be set to 'Coverage-Check'"),
			},
		},
	})
}

// This test will ensure the default behavior of auto-importing rules with a 0 value
// works appropriately.
func TestAccGitLabProjectApprovalRule_AnyApproverAutoImport(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.

	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)

	// pre-create the any_approver rule to ensure it exists
	_, _, err := testutil.TestGitlabClient.Projects.CreateProjectApprovalRule(project.ID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr("any_approver"),
		RuleType:          gitlab.Ptr("any_approver"),
		ApprovalsRequired: gitlab.Ptr(0),
	})
	if err != nil {
		t.Fatal("Failed to create approval rule prior to testing", err)
	}

	// Terraform test starts here.
	var projectApprovalRule gitlab.ProjectApprovalRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "bar" {
						project              = %d
						name                 = "bar"
						approvals_required   = %d
						rule_type            = "%s"
					}
				`, project.ID, 3, "any_approver"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover{
						Name:              "bar",
						ApprovalsRequired: 3,
						RuleType:          "any_approver",
					}),
				),
			},
			{
				ResourceName:      "gitlab_project_approval_rule.bar",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"disable_importing_default_any_approver_rule_on_create",
				},
			},
		},
	})
}

// This test ensures that we only auto-import rules that have a "0" approval required. So
// we create a rule with 1 approver required, and expect an error in the test.
func TestAccGitLabProjectApprovalRule_AnyApproverAutoImportWithOneApprover(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.

	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)

	// pre-create the any_approver rule to ensure it exists
	_, _, err := testutil.TestGitlabClient.Projects.CreateProjectApprovalRule(project.ID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr("any_approver"),
		RuleType:          gitlab.Ptr("any_approver"),
		ApprovalsRequired: gitlab.Ptr(1),
	})
	if err != nil {
		t.Fatal("Failed to create approval rule prior to testing", err)
	}

	// Terraform test starts here.
	var projectApprovalRule gitlab.ProjectApprovalRule

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_approval_rule" "bar" {
						project              = %d
						name                 = "bar"
						approvals_required   = %d
						rule_type            = "%s"
					}
				`, project.ID, 3, "any_approver"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabProjectApprovalRuleExists("gitlab_project_approval_rule.bar", &projectApprovalRule),
					testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(&projectApprovalRule, &testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover{
						Name:              "bar",
						ApprovalsRequired: 3,
						RuleType:          "any_approver",
					}),
				),
				ExpectError: regexp.MustCompile("any-approver for the project already exists"),
			},
		},
	})
}

// This test ensures that we get an error when auto-import is disabled, and a rule already pre-exists,
// even if the rule has a value of 0 that would otherwise be auto imported.
func TestAccGitLabProjectApprovalRule_AnyApproverDisableAutoImport(t *testing.T) {
	// Set up project, groups, users, and branches to use in the test.

	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)

	// pre-create the any_approver rule to ensure it exists so our apply fails when disabling import
	_, _, err := testutil.TestGitlabClient.Projects.CreateProjectApprovalRule(project.ID, &gitlab.CreateProjectLevelRuleOptions{
		Name:              gitlab.Ptr("any_approver"),
		RuleType:          gitlab.Ptr("any_approver"),
		ApprovalsRequired: gitlab.Ptr(0),
	})
	if err != nil {
		t.Fatal("Failed to create approval rule prior to testing", err)
	}
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "bar" {
				  project              = %d
				  name                 = "bar"
				  approvals_required   = %d
				  rule_type            = "%s"

				  disable_importing_default_any_approver_rule_on_create = true
				}`, project.ID, 3, "any_approver"),
				ExpectError: regexp.MustCompile("any-approver for the project already exists"),
			},
		},
	})
}

// Test checks to make sure an error occurs if both applies_to_all_protected_branches and
// protected_branch_ids are configured
func TestAccGitLabProjectApprovalRule_AppliesAllProtectedBranchesConflictBranchIds(t *testing.T) {
	// Set up project and branches to use in the test.

	testutil.SkipIfCE(t)

	project := testutil.CreateProject(t)
	branches := testutil.CreateProtectedBranches(t, project, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectApprovalRuleDestroy(project.ID),
		Steps: []resource.TestStep{
			// Create rule
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_approval_rule" "bar" {
				  project              = %d
				  name                 = "bar"
				  approvals_required   = %d
				  protected_branch_ids = [%d]
				  applies_to_all_protected_branches = true
				}`, project.ID, 3, branches[0].ID),
				ExpectError: regexp.MustCompile("Conflicting configuration arguments"),
			},
		},
	})
}

type testAccGitlabProjectApprovalRuleExpectedAttributes_Basic struct {
	Name                          string
	ApprovalsRequired             int
	EligibleApproverIDs           []int
	GroupIDs                      []int
	ProtectedBranchIDs            []int
	AppliesToAllProtectedBranches bool
}

type testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover struct {
	Name              string
	ApprovalsRequired int
	RuleType          string
}

type testAccGitlabProjectApprovalRuleExpectedAttributes_ReportType struct {
	Name              string
	ApprovalsRequired int
	RuleType          string
	ReportType        string
}

func testAccCheckGitlabProjectApprovalRuleAttributes_Basic(got *gitlab.ProjectApprovalRule, want *testAccGitlabProjectApprovalRuleExpectedAttributes_Basic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return InterceptGomegaFailure(func() {
			Expect(got.Name).To(Equal(want.Name), "name")
			Expect(got.ApprovalsRequired).To(Equal(want.ApprovalsRequired), "approvals_required")

			var approverIDs []int
			for _, approver := range got.EligibleApprovers {
				approverIDs = append(approverIDs, approver.ID)
			}
			Expect(approverIDs).To(ConsistOf(want.EligibleApproverIDs), "eligible_approvers")

			var groupIDs []int
			for _, group := range got.Groups {
				groupIDs = append(groupIDs, group.ID)
			}
			Expect(groupIDs).To(ConsistOf(want.GroupIDs), "groups")

			if want.ProtectedBranchIDs != nil {
				var protectedBranchIDs []int
				for _, branch := range got.ProtectedBranches {
					protectedBranchIDs = append(protectedBranchIDs, branch.ID)
				}
				Expect(protectedBranchIDs).To(ConsistOf(want.ProtectedBranchIDs), "protected_branches")
			}

			Expect(got.AppliesToAllProtectedBranches).To(Equal(want.AppliesToAllProtectedBranches), "applies_to_all_protected_branches")
		})
	}
}

func testAccCheckGitlabProjectApprovalRuleAttributes_AnyApprover(got *gitlab.ProjectApprovalRule, want *testAccGitlabProjectApprovalRuleExpectedAttributes_AnyApprover) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return InterceptGomegaFailure(func() {
			Expect(got.Name).To(Equal(want.Name), "name")
			Expect(got.ApprovalsRequired).To(Equal(want.ApprovalsRequired), "approvals_required")
			Expect(got.RuleType).To(Equal(want.RuleType), "rule_type")
		})
	}
}

func testAccCheckGitlabProjectApprovalRuleAttributes_ReportType(got *gitlab.ProjectApprovalRule, want *testAccGitlabProjectApprovalRuleExpectedAttributes_ReportType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return InterceptGomegaFailure(func() {
			Expect(got.Name).To(Equal(want.Name), "name")
			Expect(got.ApprovalsRequired).To(Equal(want.ApprovalsRequired), "approvals_required")
			Expect(got.RuleType).To(Equal(want.RuleType), "rule_type")
			Expect(got.ReportType).To(Equal(want.ReportType), "report_type")
		})
	}
}

func testAccCheckGitlabProjectApprovalRuleExists(n string, projectApprovalRule *gitlab.ProjectApprovalRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		projectID, ruleID, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		ruleIDInt, err := strconv.Atoi(ruleID)
		if err != nil {
			return err
		}

		rules, _, err := testutil.TestGitlabClient.Projects.GetProjectApprovalRules(projectID, &gitlab.GetProjectApprovalRulesListsOptions{})
		if err != nil {
			return err
		}

		for _, gotRule := range rules {
			if gotRule.ID == ruleIDInt {
				*projectApprovalRule = *gotRule
				return nil
			}
		}

		return fmt.Errorf("rule %d not found", ruleIDInt)
	}
}

func testAccCheckGitlabProjectApprovalRuleDestroy(pid interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return InterceptGomegaFailure(func() {
			rules, _, err := testutil.TestGitlabClient.Projects.GetProjectApprovalRules(pid, &gitlab.GetProjectApprovalRulesListsOptions{})
			Expect(err).To(BeNil())
			Expect(rules).To(BeEmpty())
		})
	}
}
