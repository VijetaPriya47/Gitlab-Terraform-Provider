//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabRunnerControllerInstanceScope_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.10")

	controller := testutil.CreateRunnerController(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabRunnerControllerInstanceScopeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_runner_controller_instance_scope" "test" {
						runner_controller_id = %d
					}
				`, controller.ID),
			},
			{
				ResourceName:      "gitlab_runner_controller_instance_scope.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabRunnerControllerInstanceScopeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_runner_controller_instance_scope" {
			continue
		}

		controllerID, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return err
		}

		scopes, _, err := testutil.TestGitlabClient.RunnerControllerScopes.ListRunnerControllerScopes(controllerID)
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return err
		}

		if len(scopes.InstanceLevelScopings) > 0 {
			return fmt.Errorf("instance scope for runner controller %d still exists", controllerID)
		}
	}
	return nil
}
