//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabRunnerController_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.9")

	controller := testutil.CreateRunnerController(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_runner_controller" "test" {
						id = %d
					}
				`, controller.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_runner_controller.test", "id", fmt.Sprintf("%d", controller.ID)),
					resource.TestCheckResourceAttrSet("data.gitlab_runner_controller.test", "description"),
					resource.TestCheckResourceAttrSet("data.gitlab_runner_controller.test", "state"),
					resource.TestCheckResourceAttrSet("data.gitlab_runner_controller.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_runner_controller.test", "updated_at"),
				),
			},
		},
	})
}
