//go:build acceptance
// +build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabGroupProtectedEnvironment_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "testing"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						access_level = "developer"
					}]
				}`, group.ID, environment),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create an approval rule.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						access_level = "developer"
					}]

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, group.ID, environment),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_group_protected_environment.this", "approval_rules.0.access_level", "maintainer"),
					resource.TestCheckResourceAttr("gitlab_group_protected_environment.this", "approval_rules.0.required_approvals", "2"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add more deploy access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							access_level = "maintainer"
						},
						{
							user_id = %d
						},
						{
							group_id = %d
						}
					]

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, group.ID, environment, user.ID, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add more approval rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							access_level = "maintainer"
						},
						{
							user_id = %d
						},
						{
							group_id = %d
						}
					]

					approval_rules = [
						{
							access_level = "maintainer"
							required_approvals = 2
						},
						{
							user_id = %d
						},
						{
							group_id = %d
							required_approvals = 3
						}
					] 
				}`, group.ID, environment, user.ID, subGroup.ID, user.ID, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove deploy access levels and rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						access_level = "maintainer"
					}]
				}`, group.ID, environment),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_GroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "testing"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						group_id = %d
					}]
				}`, group.ID, environment, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							group_id = %d
						}
					]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
							group_inheritance_type = 1
						}
					] 
				}`, group.ID, environment, subGroup.ID, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_group_protected_environment.this", "approval_rules.0.group_inheritance_type", "1"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove approval rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						group_id = %d
					}]
				}`, group.ID, environment, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add deploy access level with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							group_id = %d
							group_inheritance_type = 1
						}
					]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
						}
					] 
				}`, group.ID, environment, subGroup.ID, subGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_group_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_group_protected_environment.this", "deploy_access_levels.0.group_inheritance_type", "1"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_group_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_approvalRules_InvalidGroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "testing"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							access_level = "maintainer"
						},
						{
							user_id = %d
						},
						{
							group_id = %d
						}
					]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
							group_inheritance_type = 3
						}
					] 
				}`, group.ID, environment, user.ID, subGroup.ID, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_deployAccessLevels_InvalidGroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "testing"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							group_id = %d
							group_inheritance_type = 3
						}
					]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
						}
					] 
				}`, group.ID, environment, subGroup.ID, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_InvalidEnvironment(t *testing.T) {
	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	environment := "release-1.0"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						group_id = %d
					}]
				}`, group.ID, environment, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_InvalidEnvironmentCase(t *testing.T) {
	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	environment := "Other"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						group_id = %d
					}]
				}`, group.ID, environment, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_deployAccessLevels_userIdAndGroupIdAreConflicting(t *testing.T) {
	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "staging"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						user_id = %d
						group_id = %d
					}]
				}`, group.ID, environment, user.ID, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_approvalRules_userIdAndGroupIdAreConflicting(t *testing.T) {
	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "other"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						access_level = "developer"
					}]

					approval_rules = [{
						user_id = %d
						group_id = %d
					}]
				}`, group.ID, environment, user.ID, subGroup.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_approvalRules_userIdAndRequiredApprovalsAreConflicting(t *testing.T) {
	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]

	// Set up group user with Maintainer access.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddGroupMembersWithAccessLevel(t, group.ID, []*gitlab.User{user}, gitlab.MaintainerPermissions)

	environment := "development"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [{
						access_level = "developer"
					}]

					approval_rules = [{
						user_id = %d
						required_approvals = 2
					}]
				}`, group.ID, environment, user.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabGroupProtectedEnvironment_EnsureDeployAccessLevelsAreUnordered(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up group and subgroup.
	group := testutil.CreateGroups(t, 1)[0]
	subGroup := testutil.CreateSubGroups(t, group, 1)[0]

	environment := "production"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(group.ID, environment),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels = [
						{
							group_id = %d
						},
						{
							access_level = "developer"
						}
					]
				}`, group.ID, environment, subGroup.ID),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_group_protected_environment" "this" {
					group       = %d
					environment = %q

					deploy_access_levels =[
						{
							access_level = "developer"
						},
						{
							group_id = %d
						}
					]
				}`, group.ID, environment, subGroup.ID),
				PlanOnly: true,
			},
		},
	})
}

func testAcc_GitlabGroupProtectedEnvironment_CheckDestroy(groupID int, environmentName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, err := testutil.TestGitlabClient.GroupProtectedEnvironments.GetGroupProtectedEnvironment(groupID, environmentName)
		if err == nil {
			return errors.New("environment is still protected")
		}
		if !api.Is404(err) {
			return fmt.Errorf("unable to get protected environment: %w", err)
		}
		return nil
	}
}
