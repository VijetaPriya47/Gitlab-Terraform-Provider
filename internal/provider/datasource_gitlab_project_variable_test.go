//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectVariable_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testProjectVariable := testutil.CreateProjectVariable(t, testProject.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_variable" "this" {
						project           = %d
						key               = "%s"
						environment_scope = "%s"
					}
					`, testProject.ID, testProjectVariable.Key, testProjectVariable.EnvironmentScope,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "key", testProjectVariable.Key),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "value", testProjectVariable.Value),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "environment_scope", testProjectVariable.EnvironmentScope),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "protected", strconv.FormatBool(testProjectVariable.Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "masked", strconv.FormatBool(testProjectVariable.Masked)),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "variable_type", string(testProjectVariable.VariableType)),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabProjectVariable_noEnvironmentScope(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testProjectVariable := testutil.CreateProjectVariable(t, testProject.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_variable" "this" {
						project           = %d
						key               = "%s"
					}
					`, testProject.ID, testProjectVariable.Key,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "key", testProjectVariable.Key),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "value", testProjectVariable.Value),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "environment_scope", testProjectVariable.EnvironmentScope),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "protected", strconv.FormatBool(testProjectVariable.Protected)),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "masked", strconv.FormatBool(testProjectVariable.Masked)),
					resource.TestCheckResourceAttr("data.gitlab_project_variable.this", "variable_type", string(testProjectVariable.VariableType)),
				),
			},
		},
	})
}
