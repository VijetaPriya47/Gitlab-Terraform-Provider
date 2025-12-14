//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectVariable_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	config := fmt.Sprintf(`
	resource "gitlab_project_variable" "foo" {
		project = "%s"
		key = "my_key"
		value = "my_value"
		environment_scope = "*"
	}
	`, testProject.PathWithNamespace)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectMirrorDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "= 17.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             "gitlab_project_variable.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAccGitlabProjectVariable_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectVariableCheckAllVariablesDestroyed(testProject),
		Steps: []resource.TestStep{
			// Create a project variable from a project name.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = "%s"
						key = "my_key"
						value = "my_value"
						environment_scope = "*"
					}
					`, testProject.PathWithNamespace),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			{
				ResourceName:      "gitlab_project_variable.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Same, using the project id.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = "%d"
						key = "my_key"
						value = "my_value"
						environment_scope = "*"
						raw = true
					}
					`, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Update the variable_type.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = %d
						key = "my_key"
						value = "my_value"
						variable_type = "file"
						environment_scope = "*"
					}
					`, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Update all other attributes.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = %d
						key = "my_key"
						value = "my_value_2"
						protected = true
						masked = true
						description = %d
						environment_scope = "*"
					}
					`, testProject.ID, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			{
				ResourceName:      "gitlab_project_variable.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update "hidden" which will cerate new
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = %d
						key = "my_key"
						value = "my_value_2"
						protected = true
						masked = true
						hidden = true
						description = %d
						environment_scope = "*"
					}
					`, testProject.ID, testProject.ID),
				// Check that an "Replace" is being performed, not a Update
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_project_variable.foo", plancheck.ResourceActionReplace),
					},
				},
			},
			{
				ResourceName:      "gitlab_project_variable.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"value", // will always be "nil" in the API call.
				},
			},
			// Try to update with an illegal masked variable.
			// ref: https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = %d
						key = "my_key"

						value = <<EOF
						i am multiline
						EOF

						masked = true
						environment_scope = "*"
					}
					`, testProject.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("Invalid value for a masked variable.")),
			},
		},
	})
}

func TestAccGitlabProjectVariable_scoped(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectVariableCheckAllVariablesDestroyed(testProject),
		Steps: []resource.TestStep{
			// Create a project variable from a project id
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
					project = %d
					key = "my_key"
					value = "my_value"
					}
					`, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Update the scope.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
					project = %d
					key = "my_key"
					value = "my_value"
					environment_scope = "foo"
				}
					`, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Add a second variable with the same key and different scope.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
					project = %[1]d
					key = "my_key"
					value = "my_value"
					environment_scope = "foo"
					}

					resource "gitlab_project_variable" "bar" {
					project = %[1]d
					key = "my_key"
					value = "my_value_2"
					environment_scope = "bar"
					}
					`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
					testAccCheckGitlabProjectVariableExists("gitlab_project_variable.bar"),
				),
			},
			{
				ResourceName:      "gitlab_project_variable.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "gitlab_project_variable.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update an attribute on one of the variables.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
					project = %[1]d
					key = "my_key"
					value = "my_value"
					environment_scope = "foo"
					}

					resource "gitlab_project_variable" "bar" {
					project = %[1]d
					key = "my_key"
					value = "my_value_2_updated"
					environment_scope = "bar"
					}
					`, testProject.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
					testAccCheckGitlabProjectVariableExists("gitlab_project_variable.bar"),
				),
			},
			// Try to have two variables with the same keys and scopes.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
					project = %[1]d
					key = "my_key"
					value = "my_value"
					environment_scope = "foo"
					}

					resource "gitlab_project_variable" "bar" {
					project = %[1]d
					key = "my_key"
					value = "my_value_2"
					environment_scope = "foo"
					}
					`, testProject.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta("(my_key) has already been taken")),
			},
		},
	})
}

func TestAccGitlabProjectVariable_deletedOutsideTerraform(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccGitlabProjectVariableCheckAllVariablesDestroyed(testProject),
		Steps: []resource.TestStep{
			// Step 1: Create a project variable.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = "%s"
						key = "my_key"
						value = "my_value"
						environment_scope = "*"
					}
				`, testProject.PathWithNamespace),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Step 2: Delete the variable outside of Terraform and refresh the state.
			{
				PreConfig: func() {
					_, err := testutil.TestGitlabClient.ProjectVariables.RemoveVariable(
						testProject.ID,
						"my_key",
						&gitlab.RemoveProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: "*"}},
					)
					if err != nil {
						t.Fatalf("failed to delete project variable outside of Terraform: %v", err)
					}
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_variable" "foo" {
						project = "%s"
						key = "my_key"
						value = "my_value"
						environment_scope = "*"
					}
				`, testProject.PathWithNamespace),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(state *terraform.State) error {
						// Ensure the resource is removed from the state.
						if _, exists := state.RootModule().Resources["gitlab_project_variable.foo"]; exists {
							return fmt.Errorf("resource 'gitlab_project_variable.foo' still exists in Terraform state after being deleted outside of Terraform")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckGitlabProjectVariableExists(name string) resource.TestCheckFunc {
	var (
		key              string
		value            string
		variableType     string
		protected        string
		masked           string
		environmentScope string
		raw              string
		description      string
	)

	return resource.ComposeTestCheckFunc(
		// Load the real resource values using the GitLab API.
		func(state *terraform.State) error {
			attributes := state.RootModule().Resources[name].Primary.Attributes

			got, _, err := testutil.TestGitlabClient.ProjectVariables.GetVariable(attributes["project"], attributes["key"], &gitlab.GetProjectVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: attributes["environment_scope"]}}, gitlab.WithContext(context.Background()))
			if err != nil {
				return err
			}

			key = got.Key
			value = got.Value
			variableType = string(got.VariableType)
			protected = strconv.FormatBool(got.Protected)
			masked = strconv.FormatBool(got.Masked)
			environmentScope = got.EnvironmentScope
			raw = strconv.FormatBool(got.Raw)
			description = string(got.Description)

			return nil
		},

		// Check that the real values match what was configured in the resource.
		resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrPtr(name, "key", &key),
			resource.TestCheckResourceAttrPtr(name, "value", &value),
			resource.TestCheckResourceAttrPtr(name, "variable_type", &variableType),
			resource.TestCheckResourceAttrPtr(name, "masked", &masked),
			resource.TestCheckResourceAttrPtr(name, "protected", &protected),
			resource.TestCheckResourceAttrPtr(name, "environment_scope", &environmentScope),
			resource.TestCheckResourceAttrPtr(name, "raw", &raw),
			resource.TestCheckResourceAttrPtr(name, "description", &description),
		),
	)
}

func testAccGitlabProjectVariableCheckAllVariablesDestroyed(project *gitlab.Project) func(state *terraform.State) error {
	return func(state *terraform.State) error {
		vars, _, err := testutil.TestGitlabClient.ProjectVariables.ListVariables(project.ID, nil)
		if err != nil {
			return err
		}

		if len(vars) > 0 {
			return fmt.Errorf("expected no project variables but found %d variables %v", len(vars), vars)
		}

		return nil
	}
}
