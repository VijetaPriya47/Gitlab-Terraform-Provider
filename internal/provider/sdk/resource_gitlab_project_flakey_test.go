//go:build flakey
// +build flakey

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

// The container expiration and registry in general are a little flakey in that
// the API will occassionally state they're disabled when they're actually
// enabled.
func TestAccGitlabProject_ContainerExpirationPolicy(t *testing.T) {
	testProjectName := acctest.RandomWithPrefix("acctest")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabProjectDestroy,
		Steps: []resource.TestStep{
			// Create project with container expiration policy
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
					  name                = "%s"
					  visibility_level    = "public"

					  container_expiration_policy {
						enabled = true
						cadence = "1d"
						keep_n  = 5
					  }
					}
				`, testProjectName),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disabling container expiration policy
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project" "test" {
					  name                = "%s"
					  visibility_level    = "public"

					  container_expiration_policy {
						enabled = false
					  }
					}
				`, testProjectName),
			},
			// Verify import
			{
				ResourceName:      "gitlab_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGitlabProjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project" {
			continue
		}
		gotRepo, resp, err := testutil.TestGitlabClient.Projects.GetProject(rs.Primary.ID, nil)
		if err == nil {
			if gotRepo != nil && fmt.Sprintf("%d", gotRepo.ID) == rs.Primary.ID {
				if gotRepo.MarkedForDeletionAt == nil {
					return fmt.Errorf("Repository still exists")
				}
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}
