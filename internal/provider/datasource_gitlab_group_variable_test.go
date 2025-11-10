//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupVariable_basic(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]

	resource.ParallelTest(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"gitlab": {
				VersionConstraint: "~> 16.10",
				Source:            "gitlabhq/gitlab",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_group_variable" "this" {
						group             = %d
						key               = "any_key"
					        value             = "any-value"
						environment_scope = "*"
					}

					data "gitlab_group_variable" "this" {
						group             = gitlab_group_variable.this.group
						key               = gitlab_group_variable.this.key
						environment_scope = gitlab_group_variable.this.environment_scope
					}
					`, testGroup.ID,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceGitlabGroupVariable("gitlab_group_variable.this", "data.gitlab_group_variable.this"),
				),
			},
		},
	})
}

func testAccDataSourceGitlabGroupVariable(src, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource := s.RootModule().Resources[src]
		resourceAttributes := resource.Primary.Attributes

		datasource := s.RootModule().Resources[n]
		datasourceAttributes := datasource.Primary.Attributes

		testAttributes := []string{"id", "group", "key", "value", "variable_type", "protected", "masked", "environment_scope", "raw", "description"}

		for _, attribute := range testAttributes {
			if datasourceAttributes[attribute] != resourceAttributes[attribute] {
				return fmt.Errorf("Expected variable's attribute `%s` to be: %s, but got: `%s`", attribute, resourceAttributes[attribute], datasourceAttributes[attribute])
			}
		}

		return nil
	}
}
