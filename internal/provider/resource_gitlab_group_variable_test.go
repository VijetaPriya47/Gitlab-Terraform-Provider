//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

// Validates that the migration from the SDK to the Framework works appropriately, and
// resources created using the SDK resource transition to the Framework properly.
func TestAccGitlabGroupVariable_migrateFromSDKToFramework(t *testing.T) {
	// Set up the group for testing variables
	group := testutil.CreateGroups(t, 1)[0]

	// Create common config for testing
	randomString := acctest.RandString(5)
	config := fmt.Sprintf(`resource "gitlab_group_variable" "foo" {
		group = %d
		key = "key_%s"
		value = "value-%s"
		variable_type = "file"
		masked = false
		description = "description-%s"
	}`, group.ID, randomString, randomString, randomString)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabGroupVariableDestroy,
		Steps: []resource.TestStep{
			// Create the pipeline in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 16.10",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: config,
				Check:  resource.TestCheckResourceAttrSet("gitlab_group_variable.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config:                   config,
				Check:                    resource.TestCheckResourceAttrSet("gitlab_group_variable.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_group_variable.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func TestAccGitlabGroupVariable_basic(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	var groupVariable gitlab.GroupVariable
	rString := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create a group and variable with default options
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						variable_type = "file"
						masked = false
						description = "description-%s"
						protected = false
					}
				`, group.ID, rString, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-%s", rString),
						EnvironmentScope: "*",
						Description:      fmt.Sprintf("description-%s", rString),
					}),
				),
			},
			// Update the group variable to toggle all the values to their inverse
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-inverse-%s"
						protected = true
						masked = false
						description = "description-inverse-%s"
					}
				`, group.ID, rString, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-inverse-%s", rString),
						Protected:        true,
						EnvironmentScope: "*",
						Description:      fmt.Sprintf("description-inverse-%s", rString),
					}),
				),
			},
			// // Update the group variable to toggle the options back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						variable_type = "file"
						masked = false
						description = "description-%s"
						protected = false
					}
				`, group.ID, rString, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-%s", rString),
						Protected:        false,
						EnvironmentScope: "*",
						Description:      fmt.Sprintf("description-%s", rString),
					}),
				),
			},
			// Update the group variable to enable "masked" for a value that does not meet masking requirements, and expect an error with no state change.
			// ref: https://docs.gitlab.com/ce/ci/variables/README/#masked-variable-requirements
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = <<EOF
value-%s"
i am multiline
EOF
						variable_type = "env_var"
						masked = true
					}
				`, group.ID, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-%s", rString),
						EnvironmentScope: "*",
					}),
				),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					"Invalid value for a masked variable. Check the masked variable requirements: https://docs.gitlab.com/ci/variables/#mask-a-cicd-variable",
				)),
			},
			// Update the group variable to to enable "masked" and meet masking requirements
			// ref: https://docs.gitlab.com/ce/ci/variables/README/#masked-variable-requirements
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						variable_type = "env_var"
						masked = true
					}
				`, group.ID, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-%s", rString),
						EnvironmentScope: "*",
						Masked:           true,
					}),
				),
			},
			// Update the group variable to toggle the options back
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						variable_type = "file"
						masked = false
						description = "description-%s"
						protected = false
					}
				`, group.ID, rString, rString, rString),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
					testAccCheckGitlabGroupVariableAttributes(&groupVariable, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("value-%s", rString),
						EnvironmentScope: "*",
						Protected:        false,
						Description:      fmt.Sprintf("description-%s", rString),
					}),
				),
			},
		},
	})
}

func TestAccGitlabGroupVariable_hidden(t *testing.T) {
	foodGroup := testutil.CreateGroups(t, 1)[0]
	rString := acctest.RandString(5)

	defaultValueA := fmt.Sprintf("value-%s-a", rString)
	defaultValueB := fmt.Sprintf("value-%s-b", rString)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Update to be masked and hidden.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "keys"
						value = "%s"
						variable_type = "env_var"
						masked = true
						hidden = true
					}
				`, foodGroup.ID, defaultValueA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("gitlab_group_variable.foo", "masked"),
					resource.TestCheckResourceAttrSet("gitlab_group_variable.foo", "hidden"),
					resource.TestCheckResourceAttr("gitlab_group_variable.foo", "value", defaultValueA),
				),
			},
			// Update value
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "keys"
						value = "%s"
						variable_type = "env_var"
						masked = true
						hidden = true
					}
				`, foodGroup.ID, defaultValueB),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_variable.foo", "value", defaultValueB),
				),
				// Check that an "Update" is being performed, not a Replace
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_group_variable.foo", plancheck.ResourceActionUpdate),
					},
				},
			},
			// Update type
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "keys"
						value = "%s"
						variable_type = "file"
						masked = true
						hidden = true
					}
				`, foodGroup.ID, defaultValueB),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_variable.foo", "value", defaultValueB),
				),
				// Check that an "Update" is being performed, not a Replace
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_group_variable.foo", plancheck.ResourceActionUpdate),
					},
				},
			},
			// Update hidden to false (should require replace)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "keys"
						value = "%s"
						variable_type = "file"
						masked = true
						hidden = false
					}
				`, foodGroup.ID, defaultValueA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_group_variable.foo", "value", defaultValueA),
				),
				// Check that a "Replace" is being performed, not a Update
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gitlab_group_variable.foo", plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func TestAccGitlabGroupVariable_validationErrors(t *testing.T) {
	foodGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Try to update hidden without masked.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key"
						value = "value"
						variable_type = "env_var"
						hidden = true
					}
				`, foodGroup.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					`Attribute "masked" must be specified when "hidden" is specified`,
				)),
			},
			// Try to update hidden without masked being set to "true" (which is checked in ValidatConfig)
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key"
						value = "value"
						variable_type = "env_var"
						masked = false
						hidden = true
					}
				`, foodGroup.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					`Invalid value for a masked variable`,
				)),
			},
		},
	})
}

func TestAccGitlabGroupVariable_sameVariableDifferentEnvironments(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		Steps: []resource.TestStep{
			// Create a group with 2 variables with different env scopes but the same name
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_group_variable" "env1" {
						group = %d
						key = "variableName"
						value = "val1"
						variable_type = "file"
						masked = false
						description = "description"
						environment_scope = "env1"
					}

					resource "gitlab_group_variable" "env2" {
						group = %d
						key = "variableName"
						value = "val2"
						variable_type = "file"
						masked = false
						description = "description"
						environment_scope = "env2"
					}
					`,
					group.ID, group.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					// Check environment scope
					resource.TestCheckResourceAttr("gitlab_group_variable.env1", "environment_scope", "env1"),
					resource.TestCheckResourceAttr("gitlab_group_variable.env2", "environment_scope", "env2"),
					// Check the `value`
					resource.TestCheckResourceAttr("gitlab_group_variable.env1", "value", "val1"),
					resource.TestCheckResourceAttr("gitlab_group_variable.env2", "value", "val2"),
				),
			},
			// Update both variables to have a different value
			{
				Config: fmt.Sprintf(
					`
					resource "gitlab_group_variable" "env1" {
						group = %d
						key = "variableName"
						value = "value1"
						variable_type = "file"
						masked = false
						description = "description"
						environment_scope = "env1"
					}

					resource "gitlab_group_variable" "env2" {
						group = %d
						key = "variableName"
						value = "value2"
						variable_type = "file"
						masked = false
						description = "description"
						environment_scope = "env2"
					}
					`,
					group.ID, group.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					// Check environment scope
					resource.TestCheckResourceAttr("gitlab_group_variable.env1", "environment_scope", "env1"),
					resource.TestCheckResourceAttr("gitlab_group_variable.env2", "environment_scope", "env2"),
					// Check the `value`
					resource.TestCheckResourceAttr("gitlab_group_variable.env1", "value", "value1"),
					resource.TestCheckResourceAttr("gitlab_group_variable.env2", "value", "value2"),
				),
			},
		},
	})
}

func TestAccGitlabGroupVariable_scope(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	var groupVariableA, groupVariableB gitlab.GroupVariable
	rString := acctest.RandString(5)

	defaultValueA := fmt.Sprintf("value-%s-a", rString)
	defaultValueB := fmt.Sprintf("value-%s-b", rString)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		Steps: []resource.TestStep{
			// Create a group and variables with same keys, different scopes
			{
				Config: fmt.Sprintf(`					
					resource "gitlab_group_variable" "a" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "*"
					}
					
					resource "gitlab_group_variable" "b" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "review/*"
					}
				`, group.ID, rString, defaultValueA, group.ID, rString, defaultValueB),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.a", &groupVariableA),
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.b", &groupVariableB),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableA, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueA,
						EnvironmentScope: "*",
					}),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableB, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueB,
						EnvironmentScope: "review/*",
					}),
				),
			},
			// Change a variable's scope
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "a" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "my-new-scope"
					}
					
					resource "gitlab_group_variable" "b" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "review/*"
					}
				`, group.ID, rString, defaultValueA, group.ID, rString, defaultValueB),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.a", &groupVariableA),
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.b", &groupVariableB),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableA, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueA,
						EnvironmentScope: "my-new-scope",
					}),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableB, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueB,
						EnvironmentScope: "review/*",
					}),
				),
			},
			// Change both variables scopes at the same time
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "a" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "my-new-new-scope"
					}
					
					resource "gitlab_group_variable" "b" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "review/hello-world"
					}
				`, group.ID, rString, defaultValueA, group.ID, rString, defaultValueB),
				SkipFunc: testutil.IsRunningInCE,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.a", &groupVariableA),
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.b", &groupVariableB),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableA, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueA,
						EnvironmentScope: "my-new-new-scope",
					}),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableB, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueB,
						EnvironmentScope: "review/hello-world",
					}),
				),
			},
			// Change value of one variable
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "a" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "my-new-new-scope"
					}
					
					resource "gitlab_group_variable" "b" {
						group             = "%d"
						key               = "key_%s"
						value             = "%s"
						environment_scope = "%s"
					}
				`, group.ID, rString, defaultValueA, group.ID, rString, fmt.Sprintf("new-value-for-b-%s", rString), "review/hello-world"),
				// SkipFunc: IsRunningInCE,
				// NOTE(TF): this test sporadically fails because of this: https://gitlab.com/gitlab-org/gitlab/-/issues/333296
				SkipFunc: func() (bool, error) { return true, nil },
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.a", &groupVariableA),
					testAccCheckGitlabGroupVariableExists("gitlab_group_variable.b", &groupVariableB),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableA, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            defaultValueA,
						EnvironmentScope: "my-new-new-scope",
					}),
					testAccCheckGitlabGroupVariableAttributes(&groupVariableB, &testAccGitlabGroupVariableExpectedAttributes{
						Key:              fmt.Sprintf("key_%s", rString),
						Value:            fmt.Sprintf("new-value-for-b-%s", rString),
						EnvironmentScope: "review/hello-world",
					}),
				),
			},
		},
	})
}

func TestAccGitlabGroupVariable_deletedOutsideTerraform(t *testing.T) {
	group := testutil.CreateGroups(t, 1)[0]
	var groupVariable gitlab.GroupVariable
	rString := acctest.RandString(5)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabGroupVariableDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a Group variable.
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						environment_scope = "*"
					}
				`, group.ID, rString, rString),
				Check: testAccCheckGitlabGroupVariableExists("gitlab_group_variable.foo", &groupVariable),
			},
			// Step 2: Delete the variable outside of Terraform and refresh the state.
			{
				PreConfig: func() {
					_, err := testutil.TestGitlabClient.GroupVariables.RemoveVariable(
						group.ID,
						fmt.Sprintf("key_%s", rString),
						&gitlab.RemoveGroupVariableOptions{Filter: &gitlab.VariableFilter{EnvironmentScope: "*"}},
					)
					if err != nil {
						t.Fatalf("failed to delete group variable outside of Terraform: %v", err)
					}
				},
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "foo" {
						group = "%d"
						key = "key_%s"
						value = "value-%s"
						environment_scope = "*"
					}
				`, group.ID, rString, rString),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(state *terraform.State) error {
						// Ensure the resource is removed from the state.
						if _, exists := state.RootModule().Resources["gitlab_group_variable.foo"]; exists {
							return fmt.Errorf("resource 'gitlab_group_variable.foo' still exists in Terraform state after being deleted outside of Terraform")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccCheckGitlabGroupVariableExists(n string, groupVariable *gitlab.GroupVariable) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not Found: %s", n)
		}

		repoName := rs.Primary.Attributes["group"]
		if repoName == "" {
			return fmt.Errorf("No group ID is set")
		}
		key := rs.Primary.Attributes["key"]
		if key == "" {
			return fmt.Errorf("No variable key is set")
		}
		gotVariable, _, err := testutil.TestGitlabClient.GroupVariables.GetVariable(repoName, key, nil, utils.WithEnvironmentScopeFilter(context.Background(), rs.Primary.Attributes["environment_scope"]))
		if err != nil {
			return err
		}
		*groupVariable = *gotVariable
		return nil
	}
}

type testAccGitlabGroupVariableExpectedAttributes struct {
	Key              string
	Value            string
	Protected        bool
	Masked           bool
	Hidden           bool
	EnvironmentScope string
	Description      string
}

func testAccCheckGitlabGroupVariableAttributes(variable *gitlab.GroupVariable, want *testAccGitlabGroupVariableExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if variable.Key != want.Key {
			return fmt.Errorf("got key %s; want %s", variable.Key, want.Key)
		}

		if variable.Value != want.Value {
			return fmt.Errorf("got value %s; value %s", variable.Value, want.Value)
		}

		if variable.Protected != want.Protected {
			return fmt.Errorf("got protected %t; want %t", variable.Protected, want.Protected)
		}

		if variable.Masked != want.Masked {
			return fmt.Errorf("got masked %t; want %t", variable.Masked, want.Masked)
		}

		if variable.Hidden != want.Hidden {
			return fmt.Errorf("got hidden %t; want %t", variable.Hidden, want.Hidden)
		}

		if variable.EnvironmentScope != want.EnvironmentScope {
			return fmt.Errorf("got environment_scope %s; want %s", variable.EnvironmentScope, want.EnvironmentScope)
		}
		if variable.Description != want.Description {
			return fmt.Errorf("got description %s; want %s", variable.Description, want.Description)
		}

		return nil
	}
}

func testAccCheckGitlabGroupVariableDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_group" {
			continue
		}

		_, _, err := testutil.TestGitlabClient.Groups.GetGroup(rs.Primary.ID, nil)
		if err == nil { // nolint // TODO: Resolve this golangci-lint issue: SA9003: empty branch (staticcheck)
			//if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
			//	if gotRepo.MarkedForDeletionAt == nil {
			//		return fmt.Errorf("Repository still exists")
			//	}
			//}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
