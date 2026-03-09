//go:build acceptance

package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitlabProjectJobTokenScopes_basic(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)

	linkProject := testutil.CreateProject(t)
	linkTwoProject := testutil.CreateProject(t)

	linkGroups := testutil.CreateGroups(t, 2)

	// Create a project to add outside TF to ensure it's removed
	updateProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope array with 2 projects.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					enabled = true
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project_id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope to include a resource that should then be removed on apply.
			{
				PreConfig: func() {
					// Add the job token scope before we re-run TF
					options := &gitlab.JobTokenInboundAllowOptions{
						TargetProjectID: gitlab.Ptr(updateProject.ID),
					}

					_, _, err := testutil.TestGitlabClient.JobTokenScope.AddProjectToJobScopeAllowList(project.ID, options)
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				// After apply, the same 2 projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove projects by setting the list to empty
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = []
				}`, project.ID),
				// After apply, no projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "0"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Re-add one project to test destroy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = [%d]
				}`, project.ID, linkProject.ID),
				// After apply, the same 2 projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "1"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add groups only
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = []
					target_group_ids = [%d, %d]
				}`, project.ID, linkGroups[0].ID, linkGroups[1].ID),
				// After apply, 2 groups should be present
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "0"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "2"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove groups
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = [%d]
					target_group_ids = []
				}`, project.ID, linkProject.ID),
				// After apply, 2 groups should be present
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "1"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
		},
	})
}

func TestAcc_GitlabProjectJobTokenScopes_destroyRevertsToApplicationSettings(t *testing.T) {

	// Update the instance enforcement to be set to "false" since we're testing disabling
	// the scopes enforcement at the project level
	testutil.UpdateEnforceCIInboundJobTokenScopeEnabledSetting(t, false)

	// Set up project environment.
	project := testutil.CreateProject(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope array allowing all projects.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					enabled = false
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project_id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "enabled", "false"),
				),
			},
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					enabled = false
				}`, project.ID),
				Destroy: true,
			},
		},
	})
}

func TestAcc_GitlabProjectJobTokenScopes_testEnabledValidation(t *testing.T) {

	// Update the instance enforcement to be set to "true" since we're testing the
	// validation that only matters when instance enforcement is set to `true`
	testutil.UpdateEnforceCIInboundJobTokenScopeEnabledSetting(t, true)

	// Set up project environment.
	project := testutil.CreateProject(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope array allowing all projects.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					enabled = false
				}`, project.ID),
				ExpectError: regexp.MustCompile("`enabled` may not be set to false when the setting for `enforce_ci_inbound_job_token_scope_enabled` is set to true."),
			},
		},
	})
}

func TestAcc_GitlabProjectJobTokenScopes_basic_deprecated(t *testing.T) {

	// Set up project environment.
	project := testutil.CreateProject(t)

	linkProject := testutil.CreateProject(t)
	linkTwoProject := testutil.CreateProject(t)

	linkGroups := testutil.CreateGroups(t, 2)

	// Create a project to add outside TF to ensure it's removed
	updateProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy_deprecated,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope array with 2 projects.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project_id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the CI/CD job token scope to include a resource that should then be removed on apply.
			{
				PreConfig: func() {
					// Add the job token scope before we re-run TF
					options := &gitlab.JobTokenInboundAllowOptions{
						TargetProjectID: gitlab.Ptr(updateProject.ID),
					}

					_, _, err := testutil.TestGitlabClient.JobTokenScope.AddProjectToJobScopeAllowList(project.ID, options)
					if err != nil {
						t.Fatal(err)
					}
				},
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				// After apply, the same 2 projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove projects by setting the list to empty
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = []
				}`, project.ID),
				// After apply, no projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "0"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Re-add one project to test destroy
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = [%d]
				}`, project.ID, linkProject.ID),
				// After apply, the same 2 projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "1"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Add groups only
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = []
					target_group_ids = [%d, %d]
				}`, project.ID, linkGroups[0].ID, linkGroups[1].ID),
				// After apply, 2 groups should be present
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "0"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "2"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Remove groups
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = [%d]
					target_group_ids = []
				}`, project.ID, linkProject.ID),
				// After apply, 2 groups should be present
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "1"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
		},
	})
}

// Test checks to make sure an error occurs if both project and
// project_id are configured
func TestAcc_GitlabProjectJobTokenScopes_conflict(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)
	linkProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					project_id = %d
					target_project_ids = [%d]
				}`, project.ID, project.ID, linkProject.ID),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

// Test checks to make sure an error occurs if neither project or
// project_id are configured
func TestAcc_GitlabProjectJobTokenScopes_atLeastOneOf(t *testing.T) {
	// Set up project environment.
	linkProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					target_project_ids = [%d]
				}`, linkProject.ID),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}

func TestAcc_GitlabProjectJobTokenScopes_migrateProjectIdToProject(t *testing.T) {
	// Set up project environment.
	project := testutil.CreateProject(t)

	linkProject := testutil.CreateProject(t)
	linkTwoProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectJobTokenScopes_CheckDestroy_deprecated,
		Steps: []resource.TestStep{
			// Create a basic CI/CD job token scope array with 2 projects.
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project_id = %d
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project_id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the resource to use project instead of project_id
			{
				Config: fmt.Sprintf(`
				resource "gitlab_project_job_token_scopes" "this" {
					project = %d
					target_project_ids = [
						%d,
						%d
					]
				}`, project.ID, linkProject.ID, linkTwoProject.ID),
				// After apply, the same 2 projects should be present.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "project_id", strconv.FormatInt(project.ID, 10)),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_project_ids.#", "2"),
					resource.TestCheckResourceAttr("gitlab_project_job_token_scopes.this", "target_group_ids.#", "0"),
				),
			},
			// Verify upstream attributes with an import.
			{
				ResourceName:      "gitlab_project_job_token_scopes.this",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAcc_GitlabProjectJobTokenScopes_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_job_token_scopes" {
			projectID := rs.Primary.Attributes["project"]

			settings, _, err := testutil.TestGitlabClient.JobTokenScope.GetProjectJobTokenAccessSettings(projectID)
			if err != nil {
				return fmt.Errorf("Failed to fetch CI/CD Job Token settings: %w", err)
			}
			if !settings.InboundEnabled {
				return fmt.Errorf("Failed to destroy token scopes. The InboundEnabled flag should be true after destroy.")
			}

			projects, _, err := testutil.TestGitlabClient.JobTokenScope.GetProjectJobTokenInboundAllowList(projectID, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch CI/CD Job Token project allow list: %w", err)
			}

			if len(projects) > 1 {
				return fmt.Errorf("Failed to destroy token scopes. Except the one for the project itself, all tokens should be removed when finished.")
			}

			groups, _, err := testutil.TestGitlabClient.JobTokenScope.GetJobTokenAllowlistGroups(projectID, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch groups CI/CD Job Token group allow list: %w", err)
			}
			if len(groups) > 0 {
				return fmt.Errorf("Error destroying token scopes for groups: unexpected group found in token scopes (%v)", groups)
			}
			return nil
		}
	}
	return nil
}

func testAcc_GitlabProjectJobTokenScopes_CheckDestroy_deprecated(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_job_token_scopes" {
			projectID := rs.Primary.Attributes["project_id"]
			projects, _, err := testutil.TestGitlabClient.JobTokenScope.GetProjectJobTokenInboundAllowList(projectID, nil, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch CI/CD Job Token Scope: %w", err)
			}

			if len(projects) > 1 {
				return fmt.Errorf("Failed to destroy token scopes. Except the one for the project itself, all tokens should be removed when finished.")
			}

			groups, _, err := testutil.TestGitlabClient.JobTokenScope.GetJobTokenAllowlistGroups(projectID, nil, nil)
			if err != nil {
				return fmt.Errorf("Failed to fetch groups CI/CD Job Token Scope: %w", err)
			}
			if len(groups) > 0 {
				return fmt.Errorf("Error destroying token scopes for groups: unexpected group found in token scopes (%v)", groups)
			}
			return nil
		}
	}
	return nil
}
