//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectComplianceFrameworks_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testComplianceFrameworkAlpha := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkBeta := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkGamma := testutil.CreateComplianceFramework(t, testGroup)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectComplianceFrameworks_CheckDestroy,
		Steps: []resource.TestStep{
			// Associate one compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s"]
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.foo", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate a different compliance framework with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s"]
						project = "%d"
					}
						`, testComplianceFrameworkBeta.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.foo", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkBeta.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate two compliance frameworks with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s", "%s"]
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testComplianceFrameworkBeta.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.foo", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkBeta.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate two different compliance frameworks with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s", "%s"]
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testComplianceFrameworkGamma.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.foo", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkGamma.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s"]
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.foo", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.foo", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test removing compliance framework association on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "foo" {
						compliance_framework_ids = ["%s"]
						project = "%d"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.ID),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabProjectComplianceFrameworks_basicWithFullPath(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testComplianceFrameworkAlpha := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkBeta := testutil.CreateComplianceFramework(t, testGroup)
	testComplianceFrameworkGamma := testutil.CreateComplianceFramework(t, testGroup)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectComplianceFrameworks_CheckDestroy,
		Steps: []resource.TestStep{
			// Associate a compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate a different compliance framework with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkBeta.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkBeta.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate two compliance frameworks with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s", "%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testComplianceFrameworkBeta.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkBeta.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Associate different compliance frameworks with the project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s", "%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testComplianceFrameworkGamma.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkGamma.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test removing compliance framework association on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabProjectComplianceFrameworks_removedOutsideOfTerraform(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)
	testComplianceFrameworkAlpha := testutil.CreateComplianceFramework(t, testGroup)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabProjectComplianceFrameworks_CheckDestroy,
		Steps: []resource.TestStep{
			// Associate a compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// remove the compliance framework from the project outside of terraform
			{
				PreConfig: func() {
					testutil.DeleteProjectComplianceFrameworks(t, testProject)
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_project_compliance_frameworks.bar", "id"),
					resource.TestCheckTypeSetElemAttr("gitlab_project_compliance_frameworks.bar", "compliance_framework_ids.*", testComplianceFrameworkAlpha.ID),
				),
			},
			{
				ResourceName:      "gitlab_project_compliance_frameworks.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test removing compliance framework association on project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["%s"]
						project = "%s"
					}
						`, testComplianceFrameworkAlpha.ID, testProject.PathWithNamespace),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabProjectComplianceFrameworks_EnsureErrorOnInvalidComplianceFrameworkGID(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]
	testProject := testutil.CreateProjectWithNamespace(t, testGroup.ID)

	err_regex, err := regexp.Compile("Invalid Attribute Value Match")
	if err != nil {
		t.Errorf("Unable to format expected compliance framework gid error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			// Associate a compliance framework with a project
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_compliance_frameworks" "bar" {
						compliance_framework_ids = ["' OR 1=1-"]
						project = "%s"
					}
						`, testProject.PathWithNamespace),
				ExpectError: err_regex,
			},
		},
	})
}

func testAcc_GitlabProjectComplianceFrameworks_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_project_compliance_frameworks" {
			projectPathWithNamespace := rs.Primary.Attributes["project_path"]

			query := gitlab.GraphQLQuery{
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
			if _, err := testutil.TestGitlabClient.GraphQL.Do(context.Background(), query, &response); err != nil {
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
