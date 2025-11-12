//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

func TestAcc_GitlabRelease_basic(t *testing.T) {
	project := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAcc_GitlabRelease_CheckDestroy,
		Steps: []resource.TestStep{
			// Create a basic release (including ref since the tag won't exist)
			{
				Config: fmt.Sprintf(`
				resource "gitlab_release" "this" {
					project     = "%d"
					tag_name    = "v1.0.0"
					ref         = "main"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_release.this", "id", fmt.Sprintf("%d:v1.0.0", project.ID)),
					resource.TestCheckResourceAttr("gitlab_release.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_release.this", "tag_name", "v1.0.0"),
				),
			},
			// Update the release with optional attributes
			{
				Config: fmt.Sprintf(`
				resource "gitlab_release" "this" {
					project     = "%d"
					name        = "test-release"
					tag_name    = "v1.0.0"
					description = "Test release description"
					ref         = "main"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_release.this", "id", fmt.Sprintf("%d:v1.0.0", project.ID)),
					resource.TestCheckResourceAttr("gitlab_release.this", "project", fmt.Sprintf("%d", project.ID)),
					resource.TestCheckResourceAttr("gitlab_release.this", "name", "test-release"),
					resource.TestCheckResourceAttr("gitlab_release.this", "tag_name", "v1.0.0"),
					resource.TestCheckResourceAttr("gitlab_release.this", "description", "Test release description"),
					resource.TestCheckResourceAttr("gitlab_release.this", "ref", "main"),
					resource.TestCheckResourceAttrSet("gitlab_release.this", "created_at"),
					resource.TestCheckResourceAttrSet("gitlab_release.this", "released_at"),
					resource.TestCheckResourceAttrSet("gitlab_release.this", "author.id"),
					resource.TestCheckResourceAttrSet("gitlab_release.this", "commit.id"),
					resource.TestCheckResourceAttr("gitlab_release.this", "assets.count", "4"),
					resource.TestCheckResourceAttrSet("gitlab_release.this", "links.self"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_release.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ref"},
			},
			// Update the release
			{
				Config: fmt.Sprintf(`
				resource "gitlab_release" "this" {
					project     = "%d"
					name        = "updated-release"
					tag_name    = "v1.0.0"
					description = "Updated release description"
					ref         = "main"
				}`, project.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_release.this", "name", "updated-release"),
					resource.TestCheckResourceAttr("gitlab_release.this", "description", "Updated release description"),
				),
			},
			// Verify import
			{
				ResourceName:            "gitlab_release.this",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ref"},
			},
			// Destroy the release
			{
				Config: fmt.Sprintf(`
				resource "gitlab_release" "this" {
					project     = "%d"
					name        = "updated-release"
					tag_name    = "v1.0.0"
					description = "Updated release description"
					ref         = "main"
				}`, project.ID),
				Destroy: true,
			},
		},
	})
}

func testAcc_GitlabRelease_CheckDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_release" {
			continue
		}

		project, tagName, err := utils.ParseTwoPartID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.Releases.GetRelease(project, tagName)
		if err == nil {
			return fmt.Errorf("Release %s in project %s still exists", tagName, project)
		}
	}
	return nil
}
