//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	gitlab "gitlab.com/gitlab-org/api/client-go"
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

func TestAccGitlabComplianceFramework_EnsureErrorOnInvalidPermission(t *testing.T) {
	testutil.SkipIfCE(t)

	testGroup := testutil.CreateGroups(t, 1)[0]

	// Set up users, role mapping and personal access token.
	users := testutil.CreateUsers(t, 2)
	testutil.AddGroupMembersWithAccessLevel(t, testGroup.ID, []*gitlab.User{users[0]}, gitlab.OwnerPermissions)
	testutil.AddGroupMembersWithAccessLevel(t, testGroup.ID, []*gitlab.User{users[1]}, gitlab.DeveloperPermissions)
	ownerUserPATReadAPI := testutil.CreatePersonalAccessTokenWithScopes(t, users[0], []string{"read_api"})
	developerUserPAT := testutil.CreatePersonalAccessTokenWithScopes(t, users[1], []string{"api"})

	create_err_regex, err := regexp.Compile("Not permitted to create framework")
	if err != nil {
		t.Errorf("Unable to format expected permission error regex: %s", err)
	}
	permission_err_regex, err := regexp.Compile("you don't have permission to perform this action")
	if err != nil {
		t.Errorf("Unable to format expected permission error regex: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabComplianceFramework_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a compliance framework; should return not permitted to create framework error
			{
				// lintignore:AT004  // we need the provider configuration here to create the compliance framework as a different user
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = false
					}
						`, developerUserPAT.Token, testGroup.FullPath),
				ExpectError: create_err_regex,
			},
			// Create a compliance framework; should return no permission error
			{
				// lintignore:AT004  // we need the provider configuration here to create the compliance framework as a different user
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = false
					}
						`, ownerUserPATReadAPI.Token, testGroup.FullPath),
				ExpectError: permission_err_regex,
			},
			// Create a compliance framework, to be used to check update error
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = false
					}
						`, testGroup.FullPath),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_compliance_framework.foo", "name", "Compliance Framework"),
					resource.TestCheckResourceAttrSet("gitlab_compliance_framework.foo", "id"),
				),
			},
			{
				ResourceName:      "gitlab_compliance_framework.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update a compliance framework; should return no permission error
			{
				// lintignore:AT004  // we need the provider configuration here to create the compliance framework as a different user
				Config: fmt.Sprintf(`
					provider "gitlab" {
						token = "%s"
					}

					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "An updated Compliance Framework"
						color = "#42BEEF"
						default = false
					}
						`, developerUserPAT.Token, testGroup.FullPath),
				ExpectError: permission_err_regex,
			},
			{
				Config: fmt.Sprintf(`
					resource "gitlab_compliance_framework" "foo" {
						namespace_path = "%s"
						name = "Compliance Framework"
						description = "A test Compliance Framework"
						color = "#87BEEF"
						default = false
					}
						`, testGroup.FullPath),
				Destroy: true,
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

			query := gitlab.GraphQLQuery{
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

			var response ComplianceFrameworkResponse
			if _, err := testutil.TestGitlabClient.GraphQL.Do(query, &response); err != nil {
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
