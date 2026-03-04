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
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAccGitlabRunnerControllerToken_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.9")

	controller := testutil.CreateRunnerController(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabRunnerControllerTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "gitlab_runner_controller_token" "test" {
						runner_controller_id = %d
						description          = "test token"
					}
				`, controller.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_runner_controller_token.test", "token"),
			},
			{
				ResourceName:            "gitlab_runner_controller_token.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

func testAccCheckGitlabRunnerControllerTokenDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_runner_controller_token" {
			continue
		}

		controllerIDStr, tokenIDStr, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		controllerID, err := strconv.ParseInt(controllerIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse runner controller ID %q: %w", controllerIDStr, err)
		}

		tokenID, err := strconv.ParseInt(tokenIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse token ID %q: %w", tokenIDStr, err)
		}

		_, _, err = testutil.TestGitlabClient.RunnerControllerTokens.GetRunnerControllerToken(controllerID, tokenID)
		if err == nil {
			return fmt.Errorf("runner controller token %d still exists", tokenID)
		}
		if !api.Is404(err) {
			return err
		}
	}
	return nil
}
