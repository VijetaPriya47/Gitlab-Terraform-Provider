//go:build acceptance
// +build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabComplianceFramework_basic(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a compliance framework
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
					}
						`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "default", "false"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, description, color of compliance framework
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework Updated"
						description = "A test Compliance Framework update"
						color = "#42BEEF"
					}
						`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "default", "false"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Set back to initial settings
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
					}
						`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "default", "false"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabComplianceFramework_basicWithDefaultFramework(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a compliance framework, setting default to true
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = true
					}
						`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "default", "true"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = true
					}
						`, testGroup.FullPath),
				Destroy: true,
			},
		},
	})
}

func TestAccGitlabComplianceFramework_basicWithPipelineConfiguration(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a compliance framework, setting the pipeline configuration path
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = false
						pipeline_configuration_full_path = "%s"
					}
						`, testGroup.FullPath, "path/pipeline.yml@group/project"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "default", "false"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabComplianceFramework_EnsureErrorOnInvalidColor(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	err_regex, err := regexp.Compile("Invalid Attribute Value Match")
	if err != nil {
		t.Errorf("Unable to format expected color error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			// Create a compliance framework, setting the pipeline configuration path
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "Blue"
						default = false
					}
						`, testGroup.FullPath),
				ExpectError: err_regex,
			},
		},
	})
}

func testAcc_GitlabComplianceFramework_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type == "gitlab_compliance_framework" {
			namespacePath, frameworkID, err := utils.ParseTwoPartID(rs.Primary.ID)
			if err != nil {
				return fmt.Errorf("Failed to parse compliance framework id %q: %w", rs.Primary.ID, err)
			}

			query := api.GraphQLQuery{
				Query: fmt.Sprintf(`
						query {
							namespace(fullPath: "%s") {
								fullPath,
								complianceFrameworks(id: "%s") {
									nodes {
										id
									}
								}
							}
						}`, namespacePath, frameworkID),
			}

			var response complianceFrameworkResponse
			if _, err := api.SendGraphQLRequest(context.Background(), testutil.TestGitlabClient, query, &response); err != nil {
				return err
			}

			// compliance framework still exists if nodes is not empty
			if len(response.Data.Namespace.ComplianceFrameworks.Nodes) > 0 {
				return fmt.Errorf("Compliance Framework: %s in namespace: %s still exists", frameworkID, namespacePath)
			}

			return nil
		}
	}
	return nil
}
