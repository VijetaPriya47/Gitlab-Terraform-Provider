//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabInstanceVariables_basic(t *testing.T) {
	testVariables := make([]*gitlab.InstanceVariable, 0)
	for range 22 {
		testVariables = append(testVariables, testutil.CreateInstanceVariable(t))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "gitlab_instance_variables" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.#", fmt.Sprintf("%d", len(testVariables))),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.key", testVariables[0].Key),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.value", testVariables[0].Value),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.description", testVariables[0].Description),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.21.key", testVariables[21].Key),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.21.value", testVariables[21].Value),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.21.description", testVariables[21].Description),
				),
			},
		},
	})
}
