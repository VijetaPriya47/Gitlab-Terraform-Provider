//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabGroupVariable_basic(t *testing.T) {
	testGroup := testutil.CreateGroups(t, 1)[0]
	testVariable := testutil.CreateGroupVariable(t, testGroup.ID)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_group_variable" "this" {
						group             = %d
						key               = "%s"
						environment_scope = "%s"
					}
					`, testGroup.ID, testVariable.Key, testVariable.EnvironmentScope,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_group_variable.this", "key", testVariable.Key),
					resource.TestCheckResourceAttr("data.gitlab_group_variable.this", "value", testVariable.Value),
					resource.TestCheckResourceAttr("data.gitlab_group_variable.this", "environment_scope", testVariable.EnvironmentScope),
				),
			},
		},
	})
}
