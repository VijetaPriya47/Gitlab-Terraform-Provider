//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupVariables_basic(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]
	testVariables := make([]*gitlab.GroupVariable, 0)
	for range 25 {
		testVariables = append(testVariables, testutil.CreateGroupVariable(t, testGroup.ID))
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_variables" "this" {
						group = %d
					}
				`, testGroup.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_variables.this", "variables.#", fmt.Sprintf("%d", len(testVariables))),
					resource.TestCheckResourceAttr("data.gitlab_group_variables.this", "variables.0.key", testVariables[0].Key),
					resource.TestCheckResourceAttr("data.gitlab_group_variables.this", "variables.0.value", testVariables[0].Value),
					resource.TestCheckResourceAttr("data.gitlab_group_variables.this", "variables.24.key", testVariables[24].Key),
					resource.TestCheckResourceAttr("data.gitlab_group_variables.this", "variables.24.value", testVariables[24].Value),
				),
			},
		},
	})
}
