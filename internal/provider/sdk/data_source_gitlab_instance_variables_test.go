//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabInstanceVariables_basic(t *testing.T) {
	testVariables := make([]*gitlab.InstanceVariable, 0)
	for i := 0; i < 25; i++ {
		testVariables = append(testVariables, testutil.CreateInstanceVariable(t))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: `data "gitlab_instance_variables" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.#", fmt.Sprintf("%d", len(testVariables))),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.key", testVariables[0].Key),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.value", testVariables[0].Value),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.0.description", testVariables[0].Description),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.24.key", testVariables[24].Key),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.24.value", testVariables[24].Value),
					resource.TestCheckResourceAttr("data.gitlab_instance_variables.this", "variables.24.description", testVariables[24].Description),
				),
			},
		},
	})
}
