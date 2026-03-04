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

func TestAccGitlabRunnerController_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.9")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabRunnerControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_runner_controller" "test" {
						description = "test controller"
						state       = "disabled"
					}
				`,
			},
			{
				ResourceName:      "gitlab_runner_controller.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
					resource "gitlab_runner_controller" "test" {
						description = "updated controller"
						state       = "enabled"
					}
				`,
			},
			{
				ResourceName:      "gitlab_runner_controller.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: `
					resource "gitlab_runner_controller" "test" {
						description = "updated controller"
						state       = "dry_run"
					}
				`,
			},
			{
				ResourceName:      "gitlab_runner_controller.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabRunnerController_defaults(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.9")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabRunnerControllerDestroy,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "gitlab_runner_controller" "test" {}
				`,
			},
			{
				ResourceName:      "gitlab_runner_controller.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabRunnerControllerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_runner_controller" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse runner controller ID: %s", err)
		}

		_, _, err = testutil.TestGitlabClient.RunnerControllers.GetRunnerController(id)
		if err == nil {
			return fmt.Errorf("runner controller %d still exists", id)
		}
		if !api.Is404(err) {
			return err
		}
	}
	return nil
}
