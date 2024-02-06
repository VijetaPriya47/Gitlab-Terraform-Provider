//go:build acceptance
// +build acceptance

package sdk

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectVariable_StateUpgradeV0(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name            string
		givenV0State    map[string]interface{}
		expectedV1State map[string]interface{}
	}{
		{
			name: "Project With ID",
			givenV0State: map[string]interface{}{
				"project":           "99",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "99:myvar",
			},
			expectedV1State: map[string]interface{}{
				"project":           "99",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "99:myvar:*",
			},
		},
		{
			name: "Project With Namespace",
			givenV0State: map[string]interface{}{
				"project":           "foo/bar",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "foo/bar:myvar",
			},
			expectedV1State: map[string]interface{}{
				"project":           "foo/bar",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "foo/bar:myvar:*",
			},
		},
		{
			name: "Project With 3 Part ID",
			givenV0State: map[string]interface{}{
				"project":           "99",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "99:myvar:*",
			},
			expectedV1State: map[string]interface{}{
				"project":           "99",
				"key":               "myvar",
				"environment_scope": "*",
				"id":                "99:myvar:*",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualV1State, err := resourceGitlabProjectVariableStateUpgradeV0(context.Background(), tc.givenV0State, nil)
			if err != nil {
				t.Fatalf("Error migrating state: %s", err)
			}

			if !reflect.DeepEqual(tc.expectedV1State, actualV1State) {
				t.Fatalf("\n\nexpected:\n\n%#v\n\ngot:\n\n%#v\n\n", tc.expectedV1State, actualV1State)
			}
		})

	}
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

			got, _, err := testutil.TestGitlabClient.ProjectVariables.GetVariable(attributes["project"], attributes["key"], nil, gitlab.WithContext(context.Background()), withEnvironmentScopeFilter(context.Background(), attributes["environment_scope"]))
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

func TestAccGitlabProjectVariable_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccGitlabProjectVariableCheckAllVariablesDestroyed(testProject),
		Steps: []resource.TestStep{
			// Create a project variable from a project name.
			{
				Config: fmt.Sprintf(`
resource "gitlab_project_variable" "foo" {
  project = "%s"
  key = "my_key"
  value = "my_value"
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
  project = %d
  key = "my_key"
  value = "my_value"
  raw = true
}
`, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			// Check that the variable is recreated if deleted out-of-band.
			{
				PreConfig: func() {
					if _, err := testutil.TestGitlabClient.ProjectVariables.RemoveVariable(testProject.ID, "my_key", nil); err != nil {
						t.Fatalf("failed to remove variable: %v", err)
					}
				},
				Config: fmt.Sprintf(`
resource "gitlab_project_variable" "foo" {
  project = %d
  key = "my_key"
  value = "my_value"
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
}
`, testProject.ID, testProject.ID),
				Check: testAccCheckGitlabProjectVariableExists("gitlab_project_variable.foo"),
			},
			{
				ResourceName:      "gitlab_project_variable.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Try to update with an illegal masked variable.
			// ref: https://docs.gitlab.com/ce/ci/variables/README.html#masked-variable-requirements
			{
				Config: fmt.Sprintf(`
resource "gitlab_project_variable" "foo" {
  project = %d
  key = "my_key"
  value = <<EOF
i am multiline
EOF
  masked = true
}
`, testProject.ID),
				ExpectError: regexp.MustCompile(regexp.QuoteMeta(
					"Invalid value for a masked variable. Check the masked variable requirements: https://docs.gitlab.com/ee/ci/variables/#masked-variable-requirements",
				)),
			},
		},
	})
}

func TestAccGitlabProjectVariable_scoped(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
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
			// Updating a variable with a non-unique key only works reliably on GitLab v13.4+.
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.Background(), testutil.TestGitlabClient, "13.4"),
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
			// On versions of GitLab < 13.4 this can sometimes result in an inconsistent state instead of an error.
			{
				SkipFunc: api.IsGitLabVersionLessThan(context.Background(), testutil.TestGitlabClient, "13.4"),
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
