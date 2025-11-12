//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabInstanceVariable_basic(t *testing.T) {
	variable := testutil.CreateInstanceVariable(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_instance_variable" "this" {
						key = "%s"
					}
				`, variable.Key),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "key", variable.Key),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "value", variable.Value),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "description", variable.Description),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "variable_type", string(variable.VariableType)),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "protected", fmt.Sprintf("%t", variable.Protected)),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "masked", fmt.Sprintf("%t", variable.Masked)),
					resource.TestCheckResourceAttr("data.gitlab_instance_variable.this", "raw", fmt.Sprintf("%t", variable.Raw)),
				),
			},
		},
	})
}
