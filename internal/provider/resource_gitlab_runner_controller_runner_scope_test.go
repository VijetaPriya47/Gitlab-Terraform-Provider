//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabRunnerControllerRunnerScope_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.10")

	controller := testutil.CreateRunnerController(t)

	runner := testutil.CreateRunnerWithOptions(t, &gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("instance_type"),
	})

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabRunnerControllerRunnerScopeDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_runner_controller_runner_scope" "test" {
						runner_controller_id = %d
						runner_id            = %d
					}
				`, controller.ID, runner.ID),
			},
			{
				ResourceName:      "gitlab_runner_controller_runner_scope.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabRunnerControllerRunnerScopeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_runner_controller_runner_scope" {
			continue
		}

		controllerIDStr, runnerIDStr, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		controllerID, err := strconv.ParseInt(controllerIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse runner controller ID %q: %w", controllerIDStr, err)
		}

		runnerID, err := strconv.ParseInt(runnerIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse runner ID %q: %w", runnerIDStr, err)
		}

		scopes, _, err := testutil.TestGitlabClient.RunnerControllerScopes.ListRunnerControllerScopes(controllerID)
		if err != nil {
			if api.Is404(err) {
				return nil
			}
			return err
		}

		for _, s := range scopes.RunnerLevelScopings {
			if s.RunnerID == runnerID {
				return fmt.Errorf("runner scope %d for runner controller %d still exists", runnerID, controllerID)
			}
		}
	}
	return nil
}
