//go:build acceptance

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabProjectBadge_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	rInt := acctest.RandInt()
	rInt2 := acctest.RandInt()
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckGitlabProjectBadgeDestroy,
		Steps: []resource.TestStep{
			// Create a basic badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_badge" "foo" {
						project   = "%d"
						link_url  = "https://example.com/badge-%d"
						image_url = "https://example.com/badge-%d.svg"
					}
				`, testProject.ID, rInt, rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "link_url", fmt.Sprintf("https://example.com/badge-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "image_url", fmt.Sprintf("https://example.com/badge-%d.svg", rInt)),
				),
			},
			{
				ResourceName:      "gitlab_project_badge.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_badge" "foo" {
						project   = "%d"
						link_url  = "https://example.com/new-badge-%d"
						image_url = "https://example.com/new-badge-%d.svg"
						name      = "badge-updated"
					}
				`, testProject.ID, rInt, rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "link_url", fmt.Sprintf("https://example.com/new-badge-%d", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "image_url", fmt.Sprintf("https://example.com/new-badge-%d.svg", rInt)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo", "name", "badge-updated"),
				),
			},
			{
				ResourceName:      "gitlab_project_badge.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Create a fully setup badge
			{
				Config: fmt.Sprintf(`
					resource "gitlab_project_badge" "foo2" {
						project   = "%d"
						link_url  = "https://example.com/badge-%d"
						image_url = "https://example.com/badge-%d.svg"
						name      = "badge2"
					}
				`, testProject.ID, rInt2, rInt2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gitlab_project_badge.foo2", "project", fmt.Sprintf("%d", testProject.ID)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo2", "link_url", fmt.Sprintf("https://example.com/badge-%d", rInt2)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo2", "image_url", fmt.Sprintf("https://example.com/badge-%d.svg", rInt2)),
					resource.TestCheckResourceAttr("gitlab_project_badge.foo2", "name", "badge2"),
					resource.TestCheckResourceAttrSet("gitlab_project_badge.foo2", "rendered_link_url"),
					resource.TestCheckResourceAttrSet("gitlab_project_badge.foo2", "rendered_image_url"),
				),
			},
			{
				ResourceName:      "gitlab_project_badge.foo2",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabProjectBadge_migrateFromSDKToFramework(t *testing.T) {
	testProject := testutil.CreateProject(t)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: testAccCheckGitlabProjectBadgeDestroy,
		Steps: []resource.TestStep{
			// Create the badge in the old provider version
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"gitlab": {
						VersionConstraint: "~> 18.8",
						Source:            "gitlabhq/gitlab",
					},
				},
				Config: fmt.Sprintf(`
					resource "gitlab_project_badge" "foo" {
						project = "%d"
						link_url = "https://example.com/badge"
						image_url = "https://example.com/badge.svg"
					}
				`, testProject.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_project_badge.foo", "id"),
			},
			// Create the config in the new provider version to ensure migration works
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				Config: fmt.Sprintf(`
					resource "gitlab_project_badge" "foo" {
						project = "%d"
						link_url = "https://example.com/badge"
						image_url = "https://example.com/badge.svg"
					}
				`, testProject.ID),
				Check: resource.TestCheckResourceAttrSet("gitlab_project_badge.foo", "id"),
			},
			// Verify upstream attributes with an import
			{
				ProtoV6ProviderFactories: testAccProtoV6MuxProviderFactories,
				ResourceName:             "gitlab_project_badge.foo",
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

func testAccCheckGitlabProjectBadgeDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_project_badge" {
			continue
		}

		project, badgeID, err := ResourceGitlabProjectBadgeParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, _, err = testutil.TestGitlabClient.ProjectBadges.GetProjectBadge(project, badgeID)
		if err == nil {
			return fmt.Errorf("Project Badge %d in project %s still exists", badgeID, project)
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}
