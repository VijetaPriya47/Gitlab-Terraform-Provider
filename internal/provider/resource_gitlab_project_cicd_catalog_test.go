//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectCicdCatalog_basic(t *testing.T) {
	// Create a project with a README (required for CI/CD Catalog)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectCicdCatalog_CheckDestroy,
		Steps: []resource.TestStep{
			// Create and enable a CI/CD Catalog resource
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%d"
						enabled                  = true
						keep_settings_on_destroy = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "id", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
			// Update to disable catalog
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%d"
						enabled                  = false
						keep_settings_on_destroy = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "false"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
				),
			},
			// Verify import after disabling
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
			// Update to enable catalog again to make sure it correctly test destroy reset the catalog status
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%d"
						enabled                  = true
						keep_settings_on_destroy = false
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
				),
			},
			// Verify import after enabling
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectCicdCatalog_withPath(t *testing.T) {
	// Create a project with a README (required for CI/CD Catalog)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectCicdCatalog_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a CI/CD Catalog resource using project path
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%s"
						enabled                  = true
						keep_settings_on_destroy = false
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
					resource.TestCheckResourceAttrSet("gitlab_project_cicd_catalog.test", "id"),
				),
			},
			// Verify import - project path is preserved in ID and state
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
			// Update to disable catalog
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%s"
						enabled                  = false
						keep_settings_on_destroy = false
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "false"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "id", testProject.PathWithNamespace),
				),
			},
			// Verify import after disabling
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
			// Update to enable catalog again to make sure it correctly test destroy reset the catalog status
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project                  = "%s"
						enabled                  = true
						keep_settings_on_destroy = false
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "project", testProject.PathWithNamespace),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "false"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "id", testProject.PathWithNamespace),
				),
			},
			// Verify import after enabling
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
		},
	})
}

func TestAccGitlabProjectCicdCatalog_noResetOnDestroy(t *testing.T) {
	// Create a project with a README (required for CI/CD Catalog)
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project = "%d"
						enabled = true
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "id", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "enabled", "true"),
					resource.TestCheckResourceAttr("gitlab_project_cicd_catalog.test", "keep_settings_on_destroy", "true"),
				),
			},
			{
				ResourceName:            "gitlab_project_cicd_catalog.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"keep_settings_on_destroy"},
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_cicd_catalog" "test" {
						project = "%d"
						enabled = true
					}
				`, testProject.ID),
				Destroy: true,
				Check: func(*terraform.State) error {
					// Verify catalog status was NOT reset (keep_settings_on_destroy defaults to true)
					project, _, err := testutil.TestGitlabClient.Projects.GetProject(fmt.Sprintf("%d", testProject.ID), nil)
					if err != nil {
						return fmt.Errorf("Failed to get project: %v", err)
					}

					query := gitlab.GraphQLQuery{
						Query: `
							query($fullPath: ID!) {
								project(fullPath: $fullPath) {
									isCatalogResource
								}
							}`,
						Variables: map[string]any{
							"fullPath": project.PathWithNamespace,
						},
					}

					var response struct {
						Data struct {
							Project struct {
								IsCatalogResource bool `json:"isCatalogResource"`
							} `json:"project"`
						} `json:"data"`
					}

					if _, err := testutil.TestGitlabClient.GraphQL.Do(query, &response); err != nil {
						return fmt.Errorf("Failed to query catalog status: %v", err)
					}

					// Check that the settings were NOT reset to their original values
					// enabled should stay as true (the last updated value)
					if !response.Data.Project.IsCatalogResource {
						return fmt.Errorf("Expected catalog to stay as true, got false")
					}

					return nil
				},
			},
		},
	})
}

func testAcc_GitlabProjectCicdCatalog_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_cicd_catalog" {
			projectID := rs.Primary.ID

			// Get project to check if it still exists
			project, _, err := testutil.TestGitlabClient.Projects.GetProject(projectID, nil)
			if err != nil {
				return fmt.Errorf("Failed to get project: %v", err)
			}

			// Query the project to check if it's still a catalog resource
			query := gitlab.GraphQLQuery{
				Query: `
          query($fullPath: ID!) {
            project(fullPath: $fullPath) {
              id
              isCatalogResource
            }
          }`,
				Variables: map[string]any{
					"fullPath": project.PathWithNamespace,
				},
			}

			var response struct {
				Data struct {
					Project struct {
						ID                string `json:"id"`
						IsCatalogResource bool   `json:"isCatalogResource"`
					} `json:"project"`
				} `json:"data"`
			}

			if _, err := testutil.TestGitlabClient.GraphQL.Do(query, &response); err != nil {
				return fmt.Errorf("Failed to query catalog status: %v", err)
			}

			// Check that the catalog status was reset to its original value
			// enabled should be false (the original value)
			if response.Data.Project.IsCatalogResource {
				return fmt.Errorf("CI/CD Catalog resource for project %s should have been restored to disabled state, but is still enabled", projectID)
			}
		}
	}
	return nil
}
