//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectComplianceFramework_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testComplianceFrameworkAlpha := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkBeta := testutil.CreateComplianceFramework(t, testGroup)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Associate a compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "foo" {
						compliance_framework_id = "%s"
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate a different compliance framework with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "foo" {
						compliance_framework_id = "%s"
						project = "%d"
					}
						`, testComplianceFrameworkBeta.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "foo" {
						compliance_framework_id = "%s"
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test removing compliance framework association on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "foo" {
						compliance_framework_id = "%s"
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabProjectComplianceFramework_basicWithFullPath(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testComplianceFrameworkAlpha := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkBeta := testutil.CreateComplianceFramework(t, testGroup)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Associate a compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "bar" {
						compliance_framework_id = "%s"
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.bar", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate a different compliance framework with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "bar" {
						compliance_framework_id = "%s"
						project = "%s"
					}
						`, testComplianceFrameworkBeta.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.bar", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "bar" {
						compliance_framework_id = "%s"
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_framework.bar", "id"),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_framework.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test removing compliance framework association on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_framework" "bar" {
						compliance_framework_id = "%s"
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Destroy: true,
			},
		},
	})
}

func testAcc_GitlabProjectComplianceFramework_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_compliance_framework" {
			projectPathWithNamespace := rs.Primary.Attributes["project_path"]

			query := api.GraphQLQuery{
				Query: fmt.Sprintf(`
					query {
						project(fullPath: "%s") {
							id,
							complianceFrameworks {
								nodes {
									id
								}
							}
						}
					}`, projectPathWithNamespace),
			}

			var response projectResponse
			if _, err := api.SendGraphQLRequest(context.Background(), testutil.TestGitlabClient, query, &response); err != nil {
				return err
			}

			// compliance framework still associated if nodes is not empty
			if len(response.Data.Project.ComplianceFrameworks.Nodes) > 0 {
				return fmt.Errorf("Compliance Framework: %s is still associated to project: %s", response.Data.Project.ComplianceFrameworks.Nodes[0].ID, projectPathWithNamespace)
			}

			return nil
		}
	}
	return nil
}
