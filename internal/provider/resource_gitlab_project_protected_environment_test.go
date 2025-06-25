//go:build acceptance
// +build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectProtectedEnvironment_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	updateEnvironment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment-update")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a basic protected environment.
			{
				Config: fmt.Sprintf(`
							resource "gitlab_project_protected_environment" "this" {
								project     = %d
								environment = %q
			
								deploy_access_levels_attribute = [{
									access_level = "developer"
								}]
							}`, project.ID, updateEnvironment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
				),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_basic_deprecated(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "developer"
					}
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
				),
			},
			// Update a basic protected environment.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_protected_environment" "this" {
						project     = %d
						environment = %q
	
						deploy_access_levels {
							access_level = "maintainer"
						}
					}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
				),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_deployAccessLevels_conflicting(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						user_id = %d
					}]

					deploy_access_levels {
						group_id = %d
					}
				}`, project.ID, environment.Name, user.ID, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_basicWithEncodedName(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("testing/testing")),
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level", "developer"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_deployAccessLevels_userIdAndGroupIdAreConflicting(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						user_id = %d
						group_id = %d
					}]
				}`, project.ID, environment.Name, user.ID, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_approvalRules_userIdAndGroupIdAreConflicting(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]

					approval_rules = [{
						user_id = %d
						group_id = %d
					}]
				}`, project.ID, environment.Name, user.ID, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_approvalRules_userIdAndRequiredApprovalsAreConflicting(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]

					approval_rules = [{
						user_id = %d
						required_approvals = 2
					}]
				}`, project.ID, environment.Name, user.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_regressionIssue1132(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	additionalGroup := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &additionalGroup.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, additionalGroup.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q
					deploy_access_levels_attribute = [{
						access_level = "developer"
					},
					{
						group_id = %d
					}]
				}`, project.ID, environment.Name, additionalGroup.ID),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_EnsureDeprecatedDeployAccessLevelsAreUnordered(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						group_id = %d
					}

					deploy_access_levels {
						access_level = "developer"
					}
				}`, project.ID, environment.Name, group.ID),
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "developer"
					}
					
					deploy_access_levels {
						group_id = %d
					}
				}`, project.ID, environment.Name, group.ID),
				PlanOnly: true,
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_deployAndApprovalRules(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	t.Log("Sleeping for 30s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(120 * time.Second)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create an approval rule.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "developer"
					}]

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.access_level", "maintainer"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.required_approvals", "2"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add more deploy access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "maintainer"
					},
					{
						user_id = %d
					},
					{
						group_id = %d
					}]

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, project.ID, environment.Name, user.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add more approval rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "maintainer"
					},
					{
						user_id = %d
					},
					{
						group_id = %d
					}]

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
				}`, project.ID, environment.Name, user.ID, group.ID, user.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove deploy access levels and rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "maintainer"
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.2.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_deprecatedDeployAndApprovalRules(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	t.Log("Sleeping for 30s to wait for membership to be accurate")
	//nolint // R018 this is part of testing code, not the provider itself.
	time.Sleep(120 * time.Second)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "developer"
					}
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
				),
			},
			// Create an approval rule.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "developer"
					}

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.access_level", "maintainer"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.required_approvals", "2"),
				),
			},
			// Add more deploy access levels
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "maintainer"
					}

					deploy_access_levels {
						user_id = %d
					}
					
					deploy_access_levels {
						group_id = %d
					}

					approval_rules = [{
						access_level = "maintainer"
						required_approvals = 2
					}]
				}`, project.ID, environment.Name, user.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
				),
			},
			// Add more approval rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "maintainer"
					}
					
					deploy_access_levels {
						user_id = %d
					}
					
					deploy_access_levels {
						group_id = %d
					}

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
				}`, project.ID, environment.Name, user.ID, group.ID, user.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
			// Remove deploy access levels and rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels {
						access_level = "maintainer"
					}
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels.2.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.1.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.2.access_level_description"),
				),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_GroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(group.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic group protected environment.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						group_id = %d
					}]
				}`, project.ID, environment, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						group_id = %d
					}]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
							group_inheritance_type = 1
						}
					] 
				}`, project.ID, environment, group.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.group_inheritance_type", "1"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove approval rules
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						group_id = %d
					}]
				}`, project.ID, environment, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckNoResourceAttr("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add deploy access level with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						group_id = %d
						group_inheritance_type = 1
					}]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
						}
					] 
				}`, project.ID, environment, group.ID, group.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.access_level_description"),
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "approval_rules.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "deploy_access_levels_attribute.0.group_inheritance_type", "1"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_protected_environment.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_approvalRules_InvalidGroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						access_level = "maintainer"
					}]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
							group_inheritance_type = 3
						}
					] 
				}`, project.ID, environment.Name, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_deployAccessLevels_InvalidGroupInheritanceType(t *testing.T) {
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	// Set up project user.
	user := testutil.CreateUsers(t, 1)[0]
	testutil.AddProjectMembers(t, project.ID, []*gitlab.User{user})

	// Set up group access.
	group := testutil.CreateGroups(t, 1)[0]
	if _, err := testutil.TestGitlabClient.Projects.ShareProjectWithGroup(project.ID, &gitlab.ShareWithGroupOptions{
		GroupID:     &group.ID,
		GroupAccess: gitlab.Ptr(gitlab.MaintainerPermissions),
	}); err != nil {
		t.Fatalf("unable to share project %d with group %d", project.ID, group.ID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Add approval rules with group inheritance type
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_protected_environment" "this" {
					project     = %d
					environment = %q

					deploy_access_levels_attribute = [{
						group_id = %d
						group_inheritance_type = 3
					}]

					approval_rules = [
						{
							group_id = %d
							required_approvals = 3
						}
					] 
				}`, project.ID, environment.Name, group.ID, group.ID),
				ExpectError: regexp.MustCompile("Error: Invalid Attribute Value Match"),
			},
		},
	})
}

func testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(projectID int, environmentName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, _, err := testutil.TestGitlabClient.ProtectedEnvironments.GetProtectedEnvironment(projectID, environmentName)
		if err == nil {
			return errors.New("environment is still protected")
		}
		if !api.Is404(err) {
			return fmt.Errorf("unable to get protected environment: %w", err)
		}
		return nil
	}
}
