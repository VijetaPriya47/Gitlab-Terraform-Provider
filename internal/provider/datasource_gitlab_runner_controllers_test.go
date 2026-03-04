//go:build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabRunnerControllers_basic(t *testing.T) {
	testutil.SkipIfCE(t)
	testutil.RunIfAtLeast(t, "18.9")

	// Create page size + 1 controllers to verify pagination works.
	for range 21 {
		testutil.CreateRunnerController(t)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "gitlab_runner_controllers" "test" {}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrWith("data.gitlab_runner_controllers.test", "runner_controllers.#", func(value string) error {
						count, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						if count < 21 {
							return fmt.Errorf("expected at least 21 runner controllers, got %d", count)
						}
						return nil
					}),
				),
			},
		},
	})
}
