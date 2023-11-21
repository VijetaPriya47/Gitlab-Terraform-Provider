//go:build acceptance
// +build acceptance

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"
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

					deploy_access_levels {
						access_level = "developer"
					}
				}`, project.ID, environment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "required_approval_count", "0"),
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
			
								deploy_access_levels {
									access_level = "developer"
								}
							}`, project.ID, updateEnvironment.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_protected_environment.this", "deploy_access_levels.0.access_level_description"),
					resource.TestCheckResourceAttr("gitlab_project_protected_environment.this", "required_approval_count", "0"),
				),
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

					deploy_access_levels {
						user_id = %d
						group_id = %d
					}
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

					deploy_access_levels {
						access_level = "developer"
					}

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

					deploy_access_levels {
						access_level = "developer"
					}

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
					deploy_access_levels {
						access_level = "developer"
					}

					deploy_access_levels {
						group_id = %d
					}
				}`, project.ID, environment.Name, additionalGroup.ID),
			},
		},
	})
}

func TestAcc_GitlabProjectProtectedEnvironment_regressionIssue6104(t *testing.T) {
	t.Skipf("Unable to run this test, because for_each is not supported with the testing framework. See https://github.com/hashicorp/terraform-plugin-sdk/issues/536")
	testutil.SkipIfCE(t)

	// Set up project environment.
	project := testutil.CreateProject(t)
	environment := testutil.CreateProjectEnvironment(t, project.ID, &gitlab.CreateEnvironmentOptions{
		Name: gitlab.Ptr(acctest.RandomWithPrefix("test-protected-environment")),
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectProtectedEnvironment_CheckDestroy(project.ID, environment.Name),
		Steps: []resource.TestStep{
			// Create a basic protected environment.
			{
				Config: fmt.Sprintf(`
				locals {
				  project_id = %d
				
				  environments = {
					%s = {
					  access_level = "maintainer"
					}
				  }
				}
				
				resource "gitlab_project_protected_environment" "this" {
				  for_each = local.environments
				
				  project     = local.project_id
				  environment = each.key
				
				  deploy_access_levels {
					for_each = [each.value.access_level]
					# for_each = each.value.access_level != null ? [each.value.access_level] : []
				
					content {
					  access_level = deploy_access_levels.value
					}
				  }
				}
				`, project.ID, environment.Name),
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

func TestAcc_GitlabProjectProtectedEnvironment_EnsureDeployAccessLevelsAreUnordered(t *testing.T) {
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

	resource.ParallelTest(t, resource.TestCase{
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
