//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabRunnerControllerScopes_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.10")

	controller := testutil.CreateRunnerController(t)

	_, _, err := testutil.TestGitlabClient.RunnerControllerScopes.AddRunnerControllerInstanceScope(controller.ID)
	if err != nil {
		t.Fatalf("failed to add instance scope: %v", err)
	}
	t.Cleanup(func() {
		testutil.TestGitlabClient.RunnerControllerScopes.RemoveRunnerControllerInstanceScope(controller.ID) //nolint:errcheck
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_runner_controller_scopes" "test" {
						runner_controller_id = %d
					}
				`, controller.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "runner_controller_id", fmt.Sprintf("%d", controller.ID)),
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "instance_level_scopings.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "runner_level_scopings.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceGitlabRunnerControllerScopes_withRunnerScope(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.10")

	controller := testutil.CreateRunnerController(t)

	runner := testutil.CreateRunnerWithOptions(t, &gitlab.CreateUserRunnerOptions{
		RunnerType: gitlab.Ptr("instance_type"),
	})

	_, _, err := testutil.TestGitlabClient.RunnerControllerScopes.AddRunnerControllerRunnerScope(controller.ID, runner.ID)
	if err != nil {
		t.Fatalf("failed to add runner scope: %v", err)
	}
	t.Cleanup(func() {
		testutil.TestGitlabClient.RunnerControllerScopes.RemoveRunnerControllerRunnerScope(controller.ID, runner.ID) //nolint:errcheck
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_runner_controller_scopes" "test" {
						runner_controller_id = %d
					}
				`, controller.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "instance_level_scopings.#", "0"),
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "runner_level_scopings.#", "1"),
					resource.TestCheckResourceAttr("data.gitlab_runner_controller_scopes.test", "runner_level_scopings.0.runner_id", fmt.Sprintf("%d", runner.ID)),
				),
			},
		},
	})
}
